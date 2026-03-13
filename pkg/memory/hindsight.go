package memory

import (
	"context"
	"fmt"
	"strings"

	"github.com/Dawgomatic/Xagent/pkg/providers"
	"github.com/Dawgomatic/Xagent/pkg/vault"
)

// HindsightMemory implements a biomimetic agent memory system
// with Retain, Recall, and Reflect operations, organized into World,
// Experiences, and Mental Models banks.
type HindsightMemory struct {
	vaultWriter *vault.VaultWriter
	provider    providers.LLMProvider
}

// NewHindsightMemory creates a new HindsightMemory instance.
func NewHindsightMemory(vw *vault.VaultWriter, p providers.LLMProvider) *HindsightMemory {
	return &HindsightMemory{
		vaultWriter: vw,
		provider:    p,
	}
}

// Retain pushes memories into either the World facts bank or Experiences bank.
func (h *HindsightMemory) Retain(ctx context.Context, memory string, source string) error {
	if h.vaultWriter == nil {
		return nil
	}

	// Simple heuristic mapping: first-person/subjective tone goes to Experiences.
	// Otherwise it goes to World facts.
	// Note: in a production setting, this would actively prompt the LLM to classify.
	lower := strings.ToLower(memory)
	if strings.Contains(lower, " i ") || strings.HasPrefix(lower, "i ") ||
		strings.Contains(lower, " my ") || strings.HasPrefix(lower, "my ") ||
		strings.Contains(lower, " me ") {
		return h.vaultWriter.WriteExperience(memory, source)
	}

	return h.vaultWriter.WriteWorldFact(memory, source)
}

// Recall retrieves memories across all banks using a query.
// This is the stub architecture where a robust cross-encoder + hybrid search runs.
func (h *HindsightMemory) Recall(ctx context.Context, query string) ([]string, error) {
	// Return empty slices for now - stub for full pipeline
	return []string{}, nil
}

// Reflect is a background synthesis operation that creates Mental Models
// from raw experiences to establish deeper learning over time.
func (h *HindsightMemory) Reflect(ctx context.Context, topic string) error {
	if h.vaultWriter == nil {
		return nil
	}

	// Stub: In reality, we'd pull experiences matching the topic and use `h.provider.Chat` to summarize.
	synthesis := fmt.Sprintf("Autonomous synthesis of knowledge regarding '%s'.", topic)

	return h.vaultWriter.WriteMentalModel(topic, synthesis)
}
