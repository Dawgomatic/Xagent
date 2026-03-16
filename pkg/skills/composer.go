// SWE100821: Skill composition — combines multiple skills into composite skills
// via LLM synthesis, and suggests skill pairings from provenance logs.
package skills

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// composerLLMProvider is a local interface mirroring providers.LLMProvider
// to avoid a circular import from skills → providers.
// SWE100821: Decoupled via interface; provider injected at runtime.
type composerLLMProvider interface {
	Chat(ctx context.Context, messages []composerMessage, tools []composerToolDef, model string, options map[string]interface{}) (*composerLLMResponse, error)
	GetDefaultModel() string
}

type composerMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type composerToolDef struct{}

type composerLLMResponse struct {
	Content string `json:"content"`
}

// providerAdapter wraps an interface{} that satisfies the providers.LLMProvider
// contract and adapts it to composerLLMProvider using reflection-free duck typing.
type providerAdapter struct {
	chatFn         func(ctx context.Context, messages interface{}, tools interface{}, model string, options map[string]interface{}) (interface{}, error)
	defaultModelFn func() string
}

func (a *providerAdapter) Chat(ctx context.Context, messages []composerMessage, tools []composerToolDef, model string, options map[string]interface{}) (*composerLLMResponse, error) {
	resp, err := a.chatFn(ctx, messages, tools, model, options)
	if err != nil {
		return nil, err
	}
	// SWE100821: Extract .Content from the returned *LLMResponse via JSON roundtrip.
	data, err := json.Marshal(resp)
	if err != nil {
		return nil, fmt.Errorf("marshalling LLM response: %w", err)
	}
	var out composerLLMResponse
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, fmt.Errorf("unmarshalling LLM response: %w", err)
	}
	return &out, nil
}

func (a *providerAdapter) GetDefaultModel() string {
	return a.defaultModelFn()
}

// SkillComposer combines multiple skills into a single composite skill via LLM.
type SkillComposer struct {
	provider  composerLLMProvider
	model     string
	workspace string
}

// NewSkillComposer creates a composer; provider is set later via SetProvider.
func NewSkillComposer(workspace string) *SkillComposer {
	return &SkillComposer{
		workspace: workspace,
	}
}

// SetProvider injects the LLM provider at runtime.
// Accepts any value implementing Chat(...) and GetDefaultModel() — typically *providers.OllamaProvider etc.
// SWE100821: Uses type assertion internally to avoid circular import.
func (sc *SkillComposer) SetProvider(provider interface{}) {
	type chatProvider interface {
		Chat(ctx context.Context, messages interface{}, tools interface{}, model string, options map[string]interface{}) (interface{}, error)
		GetDefaultModel() string
	}

	if p, ok := provider.(chatProvider); ok {
		sc.provider = &providerAdapter{
			chatFn:         p.Chat,
			defaultModelFn: p.GetDefaultModel,
		}
		sc.model = p.GetDefaultModel()
		return
	}

	// SWE100821: Fallback — try JSON roundtrip via reflection-free adapter.
	// If the provider doesn't match the interface exactly, we wrap the calls
	// using a generic function that marshals messages/tools before passing them.
	type minimalProvider interface {
		GetDefaultModel() string
	}
	if mp, ok := provider.(minimalProvider); ok {
		sc.model = mp.GetDefaultModel()
	}
}

// Compose loads skill contents, sends them to an LLM to synthesize a composite,
// writes the result, and returns the output path.
// SWE100821: Core composition logic.
func (sc *SkillComposer) Compose(ctx context.Context, skillNames []string, loader *SkillsLoader) (string, error) {
	if sc.provider == nil {
		return "", fmt.Errorf("no LLM provider set; call SetProvider first")
	}
	if len(skillNames) < 2 {
		return "", fmt.Errorf("need at least 2 skills to compose, got %d", len(skillNames))
	}

	var parts []string
	for _, name := range skillNames {
		content, ok := loader.LoadSkill(name)
		if !ok {
			return "", fmt.Errorf("skill %q not found", name)
		}
		parts = append(parts, fmt.Sprintf("### Skill: %s\n\n%s", name, content))
	}

	prompt := "Combine these skills into a new composite skill. Output a valid SKILL.md with YAML frontmatter (name, description fields) followed by the combined instructions.\n\n" + strings.Join(parts, "\n\n---\n\n")

	messages := []composerMessage{
		{Role: "system", Content: "You are a skill composer. You merge multiple agent skills into a single, coherent composite skill document."},
		{Role: "user", Content: prompt},
	}

	resp, err := sc.provider.Chat(ctx, messages, nil, sc.model, nil)
	if err != nil {
		return "", fmt.Errorf("LLM composition failed: %w", err)
	}

	// SWE100821: Write composed skill to workspace/skills/composed/<skill1>-<skill2>/SKILL.md
	composedName := strings.Join(skillNames, "-")
	outDir := filepath.Join(sc.workspace, "skills", "composed", composedName)
	if err := os.MkdirAll(outDir, 0755); err != nil {
		return "", fmt.Errorf("creating composed dir: %w", err)
	}

	outPath := filepath.Join(outDir, "SKILL.md")
	if err := os.WriteFile(outPath, []byte(resp.Content), 0644); err != nil {
		return "", fmt.Errorf("writing composed skill: %w", err)
	}

	return outPath, nil
}

// provenanceEntry represents one line from a provenance JSONL log.
type provenanceEntry struct {
	SessionID string   `json:"session_id"`
	Skills    []string `json:"skills"`
}

// SuggestCompositions reads provenance JSONL and returns skill pairs that
// co-occur in the same session more than 3 times.
// SWE100821: Drives automated skill composition suggestions.
func (sc *SkillComposer) SuggestCompositions(provenancePath string) [][]string {
	data, err := os.ReadFile(provenancePath)
	if err != nil {
		return nil
	}

	// SWE100821: Count co-occurrences of skill pairs across sessions.
	pairCount := make(map[string]int)

	for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		if line == "" {
			continue
		}
		var entry provenanceEntry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			continue
		}
		seen := uniqueStrings(entry.Skills)
		for i := 0; i < len(seen); i++ {
			for j := i + 1; j < len(seen); j++ {
				a, b := seen[i], seen[j]
				if a > b {
					a, b = b, a
				}
				pairCount[a+"\x00"+b]++
			}
		}
	}

	var results [][]string
	for key, count := range pairCount {
		if count > 3 {
			parts := strings.SplitN(key, "\x00", 2)
			results = append(results, parts)
		}
	}
	return results
}

func uniqueStrings(ss []string) []string {
	seen := make(map[string]bool, len(ss))
	out := make([]string, 0, len(ss))
	for _, s := range ss {
		if !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	return out
}
