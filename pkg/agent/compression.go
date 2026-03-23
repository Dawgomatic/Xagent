// SWE100821: Context compression via distillation — uses a two-model approach
// to preserve more information in the same context window.
// A small/fast model summarizes older context; the main model gets compressed
// history + recent messages at full fidelity.

package agent

import (
	"context"
	"fmt"
	"strings"

	"github.com/Dawgomatic/Xagent/pkg/logger"
	"github.com/Dawgomatic/Xagent/pkg/providers"
)

// ContextCompressor distills older conversation history into compressed summaries.
type ContextCompressor struct {
	mainProvider      providers.LLMProvider
	distillProvider   providers.LLMProvider // can be same or a smaller/faster model
	mainModel         string
	distillModel      string
	recentWindowSize  int // number of recent messages to keep at full fidelity
	maxCompressedLen  int // max chars for compressed history
}

// NewContextCompressor creates a compressor. If distillProvider is nil, uses mainProvider.
func NewContextCompressor(mainProvider, distillProvider providers.LLMProvider, mainModel, distillModel string) *ContextCompressor {
	if distillProvider == nil {
		distillProvider = mainProvider
	}
	if distillModel == "" {
		distillModel = mainModel
	}
	return &ContextCompressor{
		mainProvider:     mainProvider,
		distillProvider:  distillProvider,
		mainModel:        mainModel,
		distillModel:     distillModel,
		recentWindowSize: 6,
		maxCompressedLen: 2000,
	}
}

// SetRecentWindowSize sets how many recent messages to keep uncompressed.
func (cc *ContextCompressor) SetRecentWindowSize(n int) {
	cc.recentWindowSize = n
}

// CompressHistory takes full message history and returns compressed older history +
// unmodified recent messages. The compressed portion uses the distill model for speed.
func (cc *ContextCompressor) CompressHistory(ctx context.Context, messages []providers.Message) (compressed string, recent []providers.Message, err error) {
	if len(messages) <= cc.recentWindowSize {
		return "", messages, nil
	}

	older := messages[:len(messages)-cc.recentWindowSize]
	recent = messages[len(messages)-cc.recentWindowSize:]

	// SWE100821: Include tool results in compression (condensed) so the summary
	// retains what actions were taken and their outcomes.
	var toCompress []providers.Message
	for _, m := range older {
		switch m.Role {
		case "user", "assistant":
			toCompress = append(toCompress, m)
		case "tool":
			condensed := m.Content
			if len(condensed) > 200 {
				condensed = condensed[:200] + "..."
			}
			toCompress = append(toCompress, providers.Message{
				Role:    "user",
				Content: fmt.Sprintf("[Tool Result] %s", condensed),
			})
		}
	}

	if len(toCompress) == 0 {
		return "", recent, nil
	}

	// Build compression prompt
	var sb strings.Builder
	for _, m := range toCompress {
		sb.WriteString(fmt.Sprintf("%s: %s\n", m.Role, truncateMsg(m.Content, 500)))
	}

	prompt := fmt.Sprintf(`Compress the following conversation into a dense summary.
Preserve: key facts, decisions, code snippets mentioned, file paths, tool results.
Remove: pleasantries, repetition, verbose explanations.
Keep it under %d characters.

CONVERSATION:
%s`, cc.maxCompressedLen, sb.String())

	resp, err := cc.distillProvider.Chat(ctx, []providers.Message{
		{Role: "user", Content: prompt},
	}, nil, cc.distillModel, map[string]interface{}{
		"max_tokens":  512,
		"temperature": 0.2,
	})

	if err != nil {
		logger.WarnCF("compression", "Context compression failed, using truncation fallback",
			map[string]interface{}{"error": err.Error()})
		// Fallback: just truncate
		return truncateMsg(sb.String(), cc.maxCompressedLen), recent, nil
	}

	compressed = resp.Content
	if len(compressed) > cc.maxCompressedLen {
		compressed = compressed[:cc.maxCompressedLen]
	}

	logger.InfoCF("compression", "Context compressed",
		map[string]interface{}{
			"original_messages": len(toCompress),
			"compressed_chars":  len(compressed),
			"recent_kept":       len(recent),
		})

	return compressed, recent, nil
}

// SWE100821: Rune-safe truncation — byte slicing can split multibyte chars (CJK, emoji).
func truncateMsg(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen]) + "..."
}
