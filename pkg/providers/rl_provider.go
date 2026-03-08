// Xagent - Ultra-lightweight personal AI agent
// OpenClaw-RL integration: RL Provider wrapping HTTPProvider with session headers
// License: MIT
//
// Copyright (c) 2026 Xagent contributors

package providers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/Dawgomatic/Xagent/pkg/logger"
)

// RLTurnType classifies conversation turns for the RL server.
type RLTurnType string

const (
	RLTurnMain RLTurnType = "main" // User-facing turns (trainable)
	RLTurnSide RLTurnType = "side" // Internal/tool/heartbeat turns (not trainable)
)

// RLSessionContext carries RL metadata for a single Chat call.
type RLSessionContext struct {
	SessionID   string
	TurnType    RLTurnType
	SessionDone bool
}

// RLProvider wraps HTTPProvider and adds OpenClaw-RL session tracking headers.
// The RL server (Python/Slime) uses these to collect training data from live conversations.
type RLProvider struct {
	apiKey     string
	serverURL  string // e.g. "http://gpu-box:30000/v1"
	model      string
	httpClient *http.Client

	// Current session context, set by the agent loop before each Chat call
	mu         sync.RWMutex
	sessionCtx *RLSessionContext
}

// NewRLProvider creates a provider that routes through the OpenClaw-RL proxy server.
func NewRLProvider(serverURL, apiKey, model string) *RLProvider {
	return &RLProvider{
		apiKey:    apiKey,
		serverURL: strings.TrimRight(serverURL, "/"),
		model:     model,
		httpClient: &http.Client{
			Timeout: 180 * time.Second, // RL server may be slower due to logprob collection
		},
	}
}

// SetSessionContext sets the RL metadata for the next Chat call.
// This should be called by the agent loop before each Chat invocation.
func (p *RLProvider) SetSessionContext(ctx *RLSessionContext) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.sessionCtx = ctx
}

// GetSessionContext returns the current session context (thread-safe).
func (p *RLProvider) GetSessionContext() *RLSessionContext {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if p.sessionCtx == nil {
		return &RLSessionContext{SessionID: "unknown", TurnType: RLTurnSide}
	}
	return p.sessionCtx
}

func (p *RLProvider) Chat(ctx context.Context, messages []Message, tools []ToolDefinition, model string, options map[string]interface{}) (*LLMResponse, error) {
	if p.serverURL == "" {
		return nil, fmt.Errorf("RL server URL not configured")
	}

	// Use configured model if caller didn't specify one
	if model == "" {
		model = p.model
	}

	requestBody := map[string]interface{}{
		"model":    model,
		"messages": messages,
	}

	if len(tools) > 0 {
		requestBody["tools"] = tools
		requestBody["tool_choice"] = "auto"
	}

	if maxTokens, ok := options["max_tokens"].(int); ok {
		requestBody["max_tokens"] = maxTokens
	}

	if temperature, ok := options["temperature"].(float64); ok {
		requestBody["temperature"] = temperature
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Build request with RL session headers
	req, err := http.NewRequestWithContext(ctx, "POST", p.serverURL+"/chat/completions", bytes.NewReader(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if p.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+p.apiKey)
	}

	// Set RL-specific headers that the OpenClaw-RL proxy server expects
	sessionCtx := p.GetSessionContext()
	req.Header.Set("X-Session-Id", sessionCtx.SessionID)
	req.Header.Set("X-Turn-Type", string(sessionCtx.TurnType))
	if sessionCtx.SessionDone {
		req.Header.Set("X-Session-Done", "true")
	}

	logger.DebugCF("rl", "RL request",
		map[string]interface{}{
			"server":       p.serverURL,
			"session_id":   sessionCtx.SessionID,
			"turn_type":    string(sessionCtx.TurnType),
			"session_done": sessionCtx.SessionDone,
			"model":        model,
			"messages":     len(messages),
		})

	// Retry loop with exponential backoff (reuse constants from http_provider.go)
	var lastErr error
	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			delay := retryDelay(attempt - 1)
			logger.WarnCF("rl", "Retrying RL request", map[string]interface{}{
				"attempt": attempt,
				"delay":   delay.String(),
				"error":   lastErr.Error(),
			})
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(delay):
			}

			// Rebuild request for retry (body was consumed)
			req, err = http.NewRequestWithContext(ctx, "POST", p.serverURL+"/chat/completions", bytes.NewReader(jsonData))
			if err != nil {
				return nil, fmt.Errorf("failed to create retry request: %w", err)
			}
			req.Header.Set("Content-Type", "application/json")
			if p.apiKey != "" {
				req.Header.Set("Authorization", "Bearer "+p.apiKey)
			}
			req.Header.Set("X-Session-Id", sessionCtx.SessionID)
			req.Header.Set("X-Turn-Type", string(sessionCtx.TurnType))
			if sessionCtx.SessionDone {
				req.Header.Set("X-Session-Done", "true")
			}
		}

		resp, err := p.httpClient.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("failed to send request to RL server: %w", err)
			continue
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			lastErr = fmt.Errorf("failed to read RL server response: %w", err)
			continue
		}

		if resp.StatusCode == http.StatusOK {
			return parseRLResponse(body)
		}

		// 503 means submission paused for weight update — always retry
		if resp.StatusCode == 503 {
			lastErr = fmt.Errorf("RL server paused for weight update (503)")
			continue
		}

		lastErr = fmt.Errorf("RL server request failed:\n  Status: %d\n  Body:   %s", resp.StatusCode, string(body))

		if !isRetryableStatus(resp.StatusCode) {
			return nil, lastErr
		}
	}

	return nil, fmt.Errorf("RL request failed after %d retries: %w", maxRetries, lastErr)
}

func (p *RLProvider) GetDefaultModel() string {
	return p.model
}

// parseRLResponse parses the OpenAI-compatible response from the RL server.
// The RL server may include extra fields (session_id) which we ignore.
func parseRLResponse(body []byte) (*LLMResponse, error) {
	var apiResponse struct {
		Choices []struct {
			Message struct {
				Content   string `json:"content"`
				ToolCalls []struct {
					ID       string `json:"id"`
					Type     string `json:"type"`
					Function *struct {
						Name      string `json:"name"`
						Arguments string `json:"arguments"`
					} `json:"function"`
				} `json:"tool_calls"`
			} `json:"message"`
			FinishReason string `json:"finish_reason"`
		} `json:"choices"`
		Usage *UsageInfo `json:"usage"`
	}

	if err := json.Unmarshal(body, &apiResponse); err != nil {
		return nil, fmt.Errorf("failed to unmarshal RL response: %w", err)
	}

	if len(apiResponse.Choices) == 0 {
		return &LLMResponse{
			Content:      "",
			FinishReason: "stop",
		}, nil
	}

	choice := apiResponse.Choices[0]

	toolCalls := make([]ToolCall, 0, len(choice.Message.ToolCalls))
	for _, tc := range choice.Message.ToolCalls {
		arguments := make(map[string]interface{})
		name := ""

		if tc.Type == "function" && tc.Function != nil {
			name = tc.Function.Name
			if tc.Function.Arguments != "" {
				if err := json.Unmarshal([]byte(tc.Function.Arguments), &arguments); err != nil {
					arguments["raw"] = tc.Function.Arguments
				}
			}
		} else if tc.Function != nil {
			name = tc.Function.Name
			if tc.Function.Arguments != "" {
				if err := json.Unmarshal([]byte(tc.Function.Arguments), &arguments); err != nil {
					arguments["raw"] = tc.Function.Arguments
				}
			}
		}

		toolCalls = append(toolCalls, ToolCall{
			ID:        tc.ID,
			Name:      name,
			Arguments: arguments,
		})
	}

	return &LLMResponse{
		Content:      choice.Message.Content,
		ToolCalls:    toolCalls,
		FinishReason: choice.FinishReason,
		Usage:        apiResponse.Usage,
	}, nil
}
