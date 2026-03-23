// SWE100821: PicoLM provider — local-first LLM inference via picolm binary.
// Spawns the picolm C binary as a subprocess for each Chat call.
// Supports --json grammar mode for structured tool calling, --cache for
// KV cache persistence (skips prompt re-processing), and ARM NEON SIMD.
// 45MB RAM, 80KB binary, zero network, zero Python.

package providers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync/atomic"

	"github.com/Dawgomatic/Xagent/pkg/config"
	"github.com/Dawgomatic/Xagent/pkg/logger"
)

// PicoLMProvider implements LLMProvider by calling the local picolm binary.
// Designed for Jetson Xavier, Raspberry Pi, and other ARM/embedded platforms.
type PicoLMProvider struct {
	cfg     *config.PicoLMConfig
	callSeq atomic.Uint64 // monotonic counter for tool-call IDs
}

func NewPicoLMProvider(cfg *config.PicoLMConfig) *PicoLMProvider {
	// SWE100821: Default KV cache path so --cache is always enabled.
	// Skips prompt re-processing on repeated conversations (~200ms+ on ARM).
	// From PicoLM model.c L507-602 KV cache persistence mechanism.
	if cfg.CachePath == "" {
		home, err := os.UserHomeDir()
		if err == nil {
			cacheDir := filepath.Join(home, ".xagent", "cache")
			os.MkdirAll(cacheDir, 0755)
			cfg.CachePath = filepath.Join(cacheDir, "picolm.kvcache")
		}
	}
	return &PicoLMProvider{cfg: cfg}
}

func (p *PicoLMProvider) GetDefaultModel() string {
	if p.cfg.ModelPath != "" {
		return p.cfg.ModelPath
	}
	return "/opt/picolm/models/tinyllama-1.1b-chat-v1.0.Q4_K_M.gguf"
}

// formatChatML formats messages into ChatML template that TinyLlama expects.
func (p *PicoLMProvider) formatChatML(messages []Message, tools []ToolDefinition) string {
	var sb strings.Builder

	for _, msg := range messages {
		switch msg.Role {
		case "system":
			sb.WriteString("<|system|>\n")
			sb.WriteString(msg.Content)
			sb.WriteString("</s>\n")
		case "user":
			sb.WriteString("<|user|>\n")
			sb.WriteString(msg.Content)
			sb.WriteString("</s>\n")
		case "assistant":
			sb.WriteString("<|assistant|>\n")
			sb.WriteString(msg.Content)
			sb.WriteString("</s>\n")
		case "tool":
			sb.WriteString("<|user|>\n")
			sb.WriteString(fmt.Sprintf("[Tool Result] %s", msg.Content))
			sb.WriteString("</s>\n")
		}
	}

	// If tools are available, inject a compact tool instruction
	if len(tools) > 0 {
		sb.WriteString("<|user|>\n")
		sb.WriteString("If you need to call a tool, respond with ONLY valid JSON: ")
		sb.WriteString(`{"tool_calls":[{"name":"<tool>","arguments":{...}}]}`)
		sb.WriteString("\nAvailable tools: ")
		names := make([]string, 0, len(tools))
		for _, t := range tools {
			names = append(names, t.Function.Name)
		}
		sb.WriteString(strings.Join(names, ", "))
		sb.WriteString("</s>\n")
	}

	sb.WriteString("<|assistant|>\n")
	return sb.String()
}

// Chat runs inference via the picolm binary.
// SWE100821: Supports --json for tool calling, --cache for KV persistence.
func (p *PicoLMProvider) Chat(ctx context.Context, messages []Message, tools []ToolDefinition, model string, options map[string]interface{}) (*LLMResponse, error) {
	binary := p.cfg.Binary
	if binary == "" {
		binary = "picolm"
	}
	if _, err := exec.LookPath(binary); err != nil {
		return nil, fmt.Errorf("picolm binary not found at %q: %w (install: cd reference/picolm/picolm && make native && sudo make install)", binary, err)
	}

	modelPath := model
	if modelPath == "" || modelPath == "picolm-local" {
		modelPath = p.GetDefaultModel()
	}
	if _, err := os.Stat(modelPath); err != nil {
		return nil, fmt.Errorf("picolm model not found at %q: %w", modelPath, err)
	}

	prompt := p.formatChatML(messages, tools)
	useJSON := len(tools) > 0

	maxTokens := p.cfg.MaxTokens
	if maxTokens <= 0 {
		maxTokens = 256
	}
	if v, ok := options["max_tokens"].(int); ok && v > 0 {
		maxTokens = v
	}

	temp := p.cfg.Temperature
	if temp <= 0 {
		temp = 0.7
	}
	if v, ok := options["temperature"].(float64); ok && v > 0 {
		temp = v
	}

	threads := p.cfg.Threads
	if threads <= 0 {
		threads = 4
	}

	args := []string{
		modelPath,
		"-n", fmt.Sprintf("%d", maxTokens),
		"-t", fmt.Sprintf("%.2f", temp),
		"-j", fmt.Sprintf("%d", threads),
	}
	if p.cfg.ContextSize > 0 {
		args = append(args, "-c", fmt.Sprintf("%d", p.cfg.ContextSize))
	}
	if useJSON {
		args = append(args, "--json")
	}
	if p.cfg.CachePath != "" {
		args = append(args, "--cache", p.cfg.CachePath)
	}

	logger.InfoCF("picolm", "Local inference", map[string]interface{}{
		"model":      modelPath,
		"tokens":     maxTokens,
		"threads":    threads,
		"json_mode":  useJSON,
		"cache":      p.cfg.CachePath != "",
		"prompt_len": len(prompt),
	})

	cmd := exec.CommandContext(ctx, binary, args...)
	cmd.Stdin = strings.NewReader(prompt)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		logger.ErrorCF("picolm", "Inference failed", map[string]interface{}{
			"error":  err.Error(),
			"stderr": stderr.String(),
		})
		return nil, fmt.Errorf("picolm error: %w — %s", err, stderr.String())
	}

	content := strings.TrimSpace(stdout.String())

	// SWE100821: Parse tool calls from --json output
	if useJSON && looksLikeToolJSON(content) {
		toolCalls, textContent := p.parseToolCalls(content)
		if len(toolCalls) > 0 {
			return &LLMResponse{
				Content:      textContent,
				ToolCalls:    toolCalls,
				FinishReason: "tool_calls",
				Usage:        &UsageInfo{},
			}, nil
		}
	}

	return &LLMResponse{
		Content:      content,
		FinishReason: "stop",
		Usage:        &UsageInfo{},
	}, nil
}

// looksLikeToolJSON is a quick check before attempting JSON parse.
func looksLikeToolJSON(s string) bool {
	return strings.Contains(s, "tool_calls") && strings.Contains(s, "{")
}

// parseToolCalls extracts tool calls from picolm --json output.
// SWE100821: Uses json.Decoder to find the first valid JSON object containing tool_calls,
// instead of fragile first-brace/last-brace slicing which breaks on nested objects or
// trailing text from the model.
func (p *PicoLMProvider) parseToolCalls(content string) ([]ToolCall, string) {
	start := strings.Index(content, "{")
	if start < 0 {
		return nil, content
	}

	var parsed struct {
		ToolCalls []struct {
			Name      string                 `json:"name"`
			Arguments map[string]interface{} `json:"arguments"`
		} `json:"tool_calls"`
	}

	// Try progressively from the first { using Decoder (handles nested braces correctly)
	dec := json.NewDecoder(strings.NewReader(content[start:]))
	if err := dec.Decode(&parsed); err != nil {
		return nil, content
	}

	var calls []ToolCall
	for _, tc := range parsed.ToolCalls {
		seq := p.callSeq.Add(1)
		argsJSON, _ := json.Marshal(tc.Arguments)
		calls = append(calls, ToolCall{
			ID:        fmt.Sprintf("picolm_%d", seq),
			Type:      "function",
			Name:      tc.Name,
			Arguments: tc.Arguments,
			Function: &FunctionCall{
				Name:      tc.Name,
				Arguments: string(argsJSON),
			},
		})
	}

	// Any text before the JSON is regular content
	textContent := strings.TrimSpace(content[:start])
	return calls, textContent
}
