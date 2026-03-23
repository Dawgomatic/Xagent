package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/Dawgomatic/Xagent/pkg/auth"
	"github.com/Dawgomatic/Xagent/pkg/logger"
)

type ClaudeProvider struct {
	client      *anthropic.Client
	tokenSource func() (string, error)
}

func NewClaudeProvider(token string) *ClaudeProvider {
	client := anthropic.NewClient(
		option.WithAuthToken(token),
		option.WithBaseURL("https://api.anthropic.com"),
	)
	return &ClaudeProvider{client: &client}
}

func NewClaudeProviderWithTokenSource(token string, tokenSource func() (string, error)) *ClaudeProvider {
	p := NewClaudeProvider(token)
	p.tokenSource = tokenSource
	return p
}

// SWE100821: Retry with exponential backoff on transient errors (429, 5xx, network).
func (p *ClaudeProvider) Chat(ctx context.Context, messages []Message, tools []ToolDefinition, model string, options map[string]interface{}) (*LLMResponse, error) {
	var opts []option.RequestOption
	if p.tokenSource != nil {
		tok, err := p.tokenSource()
		if err != nil {
			return nil, fmt.Errorf("refreshing token: %w", err)
		}
		opts = append(opts, option.WithAuthToken(tok))
	}

	params, err := buildClaudeParams(messages, tools, model, options)
	if err != nil {
		return nil, err
	}

	const maxRetries = 3
	var lastErr error
	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			delay := time.Duration(1<<uint(attempt-1)) * time.Second
			logger.WarnCF("claude", "Retrying Claude API call", map[string]interface{}{
				"attempt": attempt, "delay": delay.String(), "error": lastErr.Error(),
			})
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(delay):
			}
		}

		resp, err := p.client.Messages.New(ctx, params, opts...)
		if err != nil {
			lastErr = err
			errStr := err.Error()
			if strings.Contains(errStr, "429") || strings.Contains(errStr, "500") ||
				strings.Contains(errStr, "502") || strings.Contains(errStr, "503") ||
				strings.Contains(errStr, "overloaded") {
				continue
			}
			return nil, fmt.Errorf("claude API call: %w", err)
		}
		return parseClaudeResponse(resp), nil
	}
	return nil, fmt.Errorf("claude API call failed after %d retries: %w", maxRetries, lastErr)
}

func (p *ClaudeProvider) GetDefaultModel() string {
	return "claude-sonnet-4-5-20250929"
}

func buildClaudeParams(messages []Message, tools []ToolDefinition, model string, options map[string]interface{}) (anthropic.MessageNewParams, error) {
	var system []anthropic.TextBlockParam
	var anthropicMessages []anthropic.MessageParam

	for _, msg := range messages {
		switch msg.Role {
		case "system":
			// SWE100821: Prompt caching — mark system prompt for Anthropic prefix caching (from Nanobot).
			// Saves ~200ms latency + 90% cost on cached tokens.
			block := anthropic.TextBlockParam{Text: msg.Content}
			block.CacheControl = anthropic.NewCacheControlEphemeralParam()
			system = append(system, block)
		case "user":
			if msg.ToolCallID != "" {
				anthropicMessages = append(anthropicMessages,
					anthropic.NewUserMessage(anthropic.NewToolResultBlock(msg.ToolCallID, msg.Content, false)),
				)
			} else {
				anthropicMessages = append(anthropicMessages,
					anthropic.NewUserMessage(anthropic.NewTextBlock(msg.Content)),
				)
			}
		case "assistant":
			if len(msg.ToolCalls) > 0 {
				var blocks []anthropic.ContentBlockParamUnion
				if msg.Content != "" {
					blocks = append(blocks, anthropic.NewTextBlock(msg.Content))
				}
				for _, tc := range msg.ToolCalls {
					blocks = append(blocks, anthropic.NewToolUseBlock(tc.ID, tc.Arguments, tc.Name))
				}
				anthropicMessages = append(anthropicMessages, anthropic.NewAssistantMessage(blocks...))
			} else {
				anthropicMessages = append(anthropicMessages,
					anthropic.NewAssistantMessage(anthropic.NewTextBlock(msg.Content)),
				)
			}
		case "tool":
			anthropicMessages = append(anthropicMessages,
				anthropic.NewUserMessage(anthropic.NewToolResultBlock(msg.ToolCallID, msg.Content, false)),
			)
		}
	}

	maxTokens := int64(4096)
	if mt, ok := options["max_tokens"].(int); ok {
		maxTokens = int64(mt)
	}

	params := anthropic.MessageNewParams{
		Model:     anthropic.Model(model),
		Messages:  anthropicMessages,
		MaxTokens: maxTokens,
	}

	if len(system) > 0 {
		params.System = system
	}

	if temp, ok := options["temperature"].(float64); ok {
		params.Temperature = anthropic.Float(temp)
	}

	if len(tools) > 0 {
		params.Tools = translateToolsForClaude(tools)
	}

	return params, nil
}

func translateToolsForClaude(tools []ToolDefinition) []anthropic.ToolUnionParam {
	result := make([]anthropic.ToolUnionParam, 0, len(tools))
	for i, t := range tools {
		tool := anthropic.ToolParam{
			Name: t.Function.Name,
			InputSchema: anthropic.ToolInputSchemaParam{
				Properties: t.Function.Parameters["properties"],
			},
		}
		if desc := t.Function.Description; desc != "" {
			tool.Description = anthropic.String(desc)
		}
		if req, ok := t.Function.Parameters["required"].([]interface{}); ok {
			required := make([]string, 0, len(req))
			for _, r := range req {
				if s, ok := r.(string); ok {
					required = append(required, s)
				}
			}
			tool.InputSchema.Required = required
		}
		// SWE100821: Mark the last tool with cache_control so Anthropic caches
		// system prompt + all tool definitions as a single prefix.
		if i == len(tools)-1 {
			tool.CacheControl = anthropic.NewCacheControlEphemeralParam()
		}
		result = append(result, anthropic.ToolUnionParam{OfTool: &tool})
	}
	return result
}

func parseClaudeResponse(resp *anthropic.Message) *LLMResponse {
	// SWE100821: strings.Builder — was using content += (O(n²) allocation)
	var sb strings.Builder
	var toolCalls []ToolCall

	for _, block := range resp.Content {
		switch block.Type {
		case "text":
			tb := block.AsText()
			sb.WriteString(tb.Text)
		case "tool_use":
			tu := block.AsToolUse()
			var args map[string]interface{}
			if err := json.Unmarshal(tu.Input, &args); err != nil {
				args = map[string]interface{}{"raw": string(tu.Input)}
			}
			toolCalls = append(toolCalls, ToolCall{
				ID:        tu.ID,
				Name:      tu.Name,
				Arguments: args,
			})
		}
	}

	finishReason := "stop"
	switch resp.StopReason {
	case anthropic.StopReasonToolUse:
		finishReason = "tool_calls"
	case anthropic.StopReasonMaxTokens:
		finishReason = "length"
	case anthropic.StopReasonEndTurn:
		finishReason = "stop"
	}

	return &LLMResponse{
		Content:      sb.String(),
		ToolCalls:    toolCalls,
		FinishReason: finishReason,
		Usage: &UsageInfo{
			PromptTokens:     int(resp.Usage.InputTokens),
			CompletionTokens: int(resp.Usage.OutputTokens),
			TotalTokens:      int(resp.Usage.InputTokens + resp.Usage.OutputTokens),
		},
	}
}

func createClaudeTokenSource() func() (string, error) {
	return func() (string, error) {
		cred, err := auth.GetCredential("anthropic")
		if err != nil {
			return "", fmt.Errorf("loading auth credentials: %w", err)
		}
		if cred == nil {
			return "", fmt.Errorf("no credentials for anthropic. Run: xagent auth login --provider anthropic")
		}
		return cred.AccessToken, nil
	}
}
