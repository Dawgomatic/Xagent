// SWE100821: Subagent result aggregation — synthesizes results from multiple
// parallel subagents into a unified, coherent response.
// When multiple subagents run concurrently, the parent agent receives all
// results and generates a single synthesized answer.

package orchestration

import (
	"context"
	"fmt"
	"strings"

	"github.com/Dawgomatic/Xagent/pkg/providers"
)

// SubagentResult holds the output from a single subagent.
type SubagentResult struct {
	Role    string `json:"role"`
	Label   string `json:"label"`
	Content string `json:"content"`
	Success bool   `json:"success"`
}

// Aggregator synthesizes multiple subagent results into one response.
type Aggregator struct {
	provider providers.LLMProvider
	model    string
}

// NewAggregator creates a result aggregator.
func NewAggregator(provider providers.LLMProvider, model string) *Aggregator {
	return &Aggregator{provider: provider, model: model}
}

// Synthesize combines multiple subagent results into a single coherent response.
func (a *Aggregator) Synthesize(ctx context.Context, goal string, results []SubagentResult) (string, error) {
	if len(results) == 0 {
		return "", fmt.Errorf("no results to synthesize")
	}

	if len(results) == 1 {
		return results[0].Content, nil
	}

	var sb strings.Builder
	for i, r := range results {
		status := "✅"
		if !r.Success {
			status = "❌"
		}
		sb.WriteString(fmt.Sprintf("### Result %d (%s %s: %s)\n%s\n\n",
			i+1, status, r.Role, r.Label, r.Content))
	}

	prompt := fmt.Sprintf(`You are synthesizing results from multiple specialist agents working on the same goal.

Goal: %s

Results from specialists:
%s

Create a single, coherent response that:
1. Combines complementary information from all successful results
2. Notes any conflicts or contradictions between results
3. Prioritizes the most relevant and accurate information
4. Maintains a clear, organized structure

Do NOT repeat the raw results — synthesize them into one answer.`, goal, sb.String())

	resp, err := a.provider.Chat(ctx, []providers.Message{
		{Role: "user", Content: prompt},
	}, nil, a.model, map[string]interface{}{
		"max_tokens":  2048,
		"temperature": 0.3,
	})
	if err != nil {
		// Fallback: concatenate results
		var fallback strings.Builder
		for _, r := range results {
			if r.Success {
				fallback.WriteString(fmt.Sprintf("## %s (%s)\n%s\n\n", r.Label, r.Role, r.Content))
			}
		}
		return fallback.String(), nil
	}

	return resp.Content, nil
}

// MergeDAGResults aggregates results from a completed TaskDAG.
func (a *Aggregator) MergeDAGResults(ctx context.Context, dag *TaskDAG) (string, error) {
	var results []SubagentResult

	dag.mu.RLock()
	for _, node := range dag.Nodes {
		results = append(results, SubagentResult{
			Role:    node.Role,
			Label:   node.ID,
			Content: node.Result,
			Success: node.Status == "completed",
		})
	}
	dag.mu.RUnlock()

	return a.Synthesize(ctx, dag.Goal, results)
}
