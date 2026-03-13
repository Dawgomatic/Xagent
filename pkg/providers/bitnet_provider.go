package providers

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/Dawgomatic/Xagent/pkg/config"
	"github.com/Dawgomatic/Xagent/pkg/logger"
)

// BitNetProvider implements LLMProvider by calling the local BitNet runtime
// optimized for 1.58-bit quantized models, enabling ultra-light inference
// on edge devices such as Jetson Xavier.
type BitNetProvider struct {
	cfg *config.BitNetConfig
}

// NewBitNetProvider creates a new provider instance.
func NewBitNetProvider(cfg *config.BitNetConfig) *BitNetProvider {
	return &BitNetProvider{
		cfg: cfg,
	}
}

// GetDefaultModel returns the configured default model.
func (b *BitNetProvider) GetDefaultModel() string {
	if b.cfg.Model != "" {
		return b.cfg.Model
	}
	return "bitnet_b1_58-3B"
}

// formatPrompt formats standard messages into a flat string for BitNet inference.
func (b *BitNetProvider) formatPrompt(messages []Message) string {
	var sb strings.Builder
	for _, msg := range messages {
		sb.WriteString(fmt.Sprintf("<|%s|>\n%s\n", msg.Role, msg.Content))
	}
	sb.WriteString("<|assistant|>\n")
	return sb.String()
}

// Chat runs inferences via the setup environment for BitNet.
func (b *BitNetProvider) Chat(ctx context.Context, messages []Message, tools []ToolDefinition, model string, options map[string]interface{}) (*LLMResponse, error) {
	if !b.cfg.Enabled {
		return nil, fmt.Errorf("bitnet provider is not enabled")
	}

	prompt := b.formatPrompt(messages)
	if model == "" {
		model = b.GetDefaultModel()
	}

	logger.InfoCF("bitnet", "Executing 1.58b inference", map[string]interface{}{
		"model":   model,
		"runtime": b.cfg.Runtime,
		"threads": b.cfg.Threads,
	})

	var cmd *exec.Cmd

	if b.cfg.Runtime == "python" || b.cfg.Runtime == "bitnet_inference.py" {
		// Run via bitnet_inference.py
		args := []string{
			"pkg/providers/bitnet_inference.py",
			"-m", model,
			"-n", "512", // Prediction tokens
			"-p", prompt,
			"-t", fmt.Sprintf("%d", b.cfg.Threads),
			"-c", fmt.Sprintf("%d", b.cfg.ContextSize),
		}
		cmd = exec.CommandContext(ctx, "python3", args...)
	} else {
		// Default to llama.cpp approach
		args := []string{
			"-m", model,
			"-n", "512",
			"-p", prompt,
			"-t", fmt.Sprintf("%d", b.cfg.Threads),
			"-c", fmt.Sprintf("%d", b.cfg.ContextSize),
		}
		cmd = exec.CommandContext(ctx, b.cfg.Runtime, args...)
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		logger.ErrorCF("bitnet", "Inference failed", map[string]interface{}{
			"error":  err.Error(),
			"stderr": stderr.String(),
		})
		return nil, fmt.Errorf("bitnet execution error: %w. stderr: %s", err, stderr.String())
	}

	out := stdout.String()
	// run_inference.py usually outputs everything. We might need to split off the prompt if it echoes it.
	// For now, we return it stripped.
	content := strings.TrimSpace(out)

	usage := &UsageInfo{
		PromptTokens:     0, // BitNet currently might not report this exactly formatted
		CompletionTokens: 0,
		TotalTokens:      0,
	}

	return &LLMResponse{
		Content:      content,
		FinishReason: "stop",
		Usage:        usage,
	}, nil
}
