// SWE100821: Biomimetic agent memory — Retain, Recall, Reflect.
// Retain classifies into World/Experience banks. Recall searches vault + semantic.
// Reflect synthesizes experiences into Mental Models via LLM.

package memory

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Dawgomatic/Xagent/pkg/logger"
	"github.com/Dawgomatic/Xagent/pkg/providers"
	"github.com/Dawgomatic/Xagent/pkg/vault"
)

// HindsightMemory implements a biomimetic agent memory system
// with Retain, Recall, and Reflect operations, organized into World,
// Experiences, and Mental Models banks.
type HindsightMemory struct {
	vaultWriter    *vault.VaultWriter
	provider       providers.LLMProvider
	semanticMemory *SemanticMemory // SWE100821: vector search for recall
	vaultRoot      string          // SWE100821: vault path for file-based search
	model          string
}

// NewHindsightMemory creates a new HindsightMemory instance.
func NewHindsightMemory(vw *vault.VaultWriter, p providers.LLMProvider) *HindsightMemory {
	return &HindsightMemory{
		vaultWriter: vw,
		provider:    p,
	}
}

// SetSemanticMemory attaches a semantic memory backend for vector-based recall.
func (h *HindsightMemory) SetSemanticMemory(sm *SemanticMemory) {
	h.semanticMemory = sm
}

// SetVaultRoot sets the vault root path for file-based keyword recall.
func (h *HindsightMemory) SetVaultRoot(path string) {
	h.vaultRoot = path
}

// SetModel sets the LLM model name for reflection prompts.
func (h *HindsightMemory) SetModel(model string) {
	h.model = model
}

// Retain pushes memories into either the World facts bank or Experiences bank.
func (h *HindsightMemory) Retain(ctx context.Context, memory string, source string) error {
	if h.vaultWriter == nil {
		return nil
	}

	lower := strings.ToLower(memory)
	if strings.Contains(lower, " i ") || strings.HasPrefix(lower, "i ") ||
		strings.Contains(lower, " my ") || strings.HasPrefix(lower, "my ") ||
		strings.Contains(lower, " me ") {
		return h.vaultWriter.WriteExperience(memory, source)
	}

	return h.vaultWriter.WriteWorldFact(memory, source)
}

// Recall retrieves memories across vault files and semantic memory using a query.
// SWE100821: Searches vault keyword files + Qdrant vector store, merges and deduplicates.
func (h *HindsightMemory) Recall(ctx context.Context, query string) ([]string, error) {
	var results []string
	seen := make(map[string]bool)

	// SWE100821: Search semantic memory (vector similarity)
	if h.semanticMemory != nil && h.semanticMemory.IsAvailable() {
		points, err := h.semanticMemory.Search(ctx, query, 5)
		if err == nil {
			for _, p := range points {
				if p.Score >= 0.5 && !seen[p.Text] {
					results = append(results, p.Text)
					seen[p.Text] = true
				}
			}
		}
	}

	// SWE100821: Search vault files by keyword matching
	if h.vaultRoot != "" {
		vaultResults := h.searchVaultFiles(query, 5)
		for _, r := range vaultResults {
			if !seen[r] {
				results = append(results, r)
				seen[r] = true
			}
		}
	}

	return results, nil
}

// Reflect synthesizes experiences about a topic into a Mental Model via LLM.
// SWE100821: Uses Recall to gather context, then LLM to synthesize.
func (h *HindsightMemory) Reflect(ctx context.Context, topic string) error {
	if h.vaultWriter == nil || h.provider == nil {
		return nil
	}

	memories, err := h.Recall(ctx, topic)
	if err != nil || len(memories) == 0 {
		synthesis := fmt.Sprintf("Initial mental model for '%s' — insufficient data for deep synthesis.", topic)
		return h.vaultWriter.WriteMentalModel(topic, synthesis)
	}

	prompt := fmt.Sprintf(`Synthesize these memories and experiences into a concise mental model about "%s".
Focus on: causal relationships, patterns, key learnings, and actionable insights.
Keep it under 300 words.

MEMORIES:
%s`, topic, strings.Join(memories, "\n---\n"))

	model := h.model
	if model == "" {
		model = "llama3"
	}

	resp, err := h.provider.Chat(ctx, []providers.Message{
		{Role: "user", Content: prompt},
	}, nil, model, map[string]interface{}{
		"max_tokens":  512,
		"temperature": 0.3,
	})
	if err != nil {
		logger.WarnCF("hindsight", "Reflect LLM call failed", map[string]interface{}{"error": err.Error()})
		return h.vaultWriter.WriteMentalModel(topic, fmt.Sprintf("Reflection on '%s' pending (LLM unavailable).", topic))
	}

	return h.vaultWriter.WriteMentalModel(topic, resp.Content)
}

// searchVaultFiles does a basic keyword search across vault markdown files.
func (h *HindsightMemory) searchVaultFiles(query string, maxResults int) []string {
	var results []string
	queryWords := strings.Fields(strings.ToLower(query))
	if len(queryWords) == 0 {
		return results
	}

	dirs := []string{
		filepath.Join(h.vaultRoot, "world-facts"),
		filepath.Join(h.vaultRoot, "experiences"),
		filepath.Join(h.vaultRoot, "mental-models"),
	}

	for _, dir := range dirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
				continue
			}
			data, err := os.ReadFile(filepath.Join(dir, entry.Name()))
			if err != nil {
				continue
			}
			content := strings.ToLower(string(data))
			hits := 0
			for _, w := range queryWords {
				if strings.Contains(content, w) {
					hits++
				}
			}
			if hits > 0 && hits >= len(queryWords)/2 {
				// Extract first meaningful line as result
				for _, line := range strings.Split(string(data), "\n") {
					line = strings.TrimSpace(line)
					if line != "" && !strings.HasPrefix(line, "---") && !strings.HasPrefix(line, "#") {
						results = append(results, line)
						break
					}
				}
			}
			if len(results) >= maxResults {
				return results
			}
		}
	}
	return results
}
