// SWE100821: Wire orchestration DAG — exposes a "decompose" tool that uses the LLM
// to decompose a complex task into a DAG, then executes subtasks via subagents.
// Reused: pkg/orchestration/dag.go (TaskDAG, TaskNode, Execute)
// Reused: pkg/orchestration/aggregator.go (MergeDAGResults)
// Reused: pkg/orchestration/roles.go (GetRole for subagent specialization)

package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/Dawgomatic/Xagent/pkg/logger"
	"github.com/Dawgomatic/Xagent/pkg/orchestration"
	"github.com/Dawgomatic/Xagent/pkg/providers"
	"github.com/Dawgomatic/Xagent/pkg/tools"
)

// decomposeTool implements tools.Tool to decompose a task into a DAG and execute it.
type decomposeTool struct {
	provider providers.LLMProvider
	model    string
}

// NewDecomposeTool creates the "decompose" orchestration tool.
func NewDecomposeTool(provider providers.LLMProvider, model string) tools.Tool {
	return &decomposeTool{provider: provider, model: model}
}

func (t *decomposeTool) Name() string        { return "decompose" }
func (t *decomposeTool) Description() string {
	return "Decompose a complex task into parallel/sequential subtasks, execute via subagents, and return aggregated results."
}

func (t *decomposeTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"task": map[string]interface{}{
				"type":        "string",
				"description": "The complex task to decompose into subtasks",
			},
		},
		"required": []string{"task"},
	}
}

// dagSpec is the JSON structure the LLM produces for the DAG.
type dagSpec struct {
	Nodes []dagNodeSpec `json:"nodes"`
}

type dagNodeSpec struct {
	ID          string   `json:"id"`
	Description string   `json:"description"`
	Role        string   `json:"role"`
	DependsOn   []string `json:"depends_on"`
}

func (t *decomposeTool) Execute(ctx context.Context, args map[string]interface{}) *tools.ToolResult {
	task, ok := args["task"].(string)
	if !ok || task == "" {
		return tools.ErrorResult("task parameter is required")
	}

	// SWE100821: Ask LLM to produce a JSON DAG for the task
	prompt := fmt.Sprintf(`Decompose this task into 2-6 subtasks suitable for parallel/sequential execution.
Return ONLY valid JSON (no markdown fencing) matching this schema:
{"nodes":[{"id":"t1","description":"...","role":"researcher|coder|reviewer|planner|sysadmin","depends_on":[]}]}

Available roles: researcher (web search), coder (file ops), reviewer (read-only analysis), planner (decomposition), sysadmin (system checks).

Task: %s`, task)

	resp, err := t.provider.Chat(ctx, []providers.Message{
		{Role: "user", Content: prompt},
	}, nil, t.model, map[string]interface{}{
		"max_tokens":  1024,
		"temperature": 0.3,
	})
	if err != nil {
		return tools.ErrorResult(fmt.Sprintf("LLM decomposition failed: %v", err))
	}

	// SWE100821: Parse DAG spec from LLM response
	var spec dagSpec
	if err := json.Unmarshal([]byte(resp.Content), &spec); err != nil {
		return tools.ErrorResult(fmt.Sprintf("Failed to parse DAG JSON: %v\nRaw: %s", err, resp.Content))
	}
	if len(spec.Nodes) == 0 {
		return tools.ErrorResult("LLM returned empty DAG")
	}

	// SWE100821: Build TaskDAG from spec
	dagID := fmt.Sprintf("dag-%d", time.Now().UnixMilli())
	dag := orchestration.NewTaskDAG(dagID, task)

	for _, ns := range spec.Nodes {
		node := &orchestration.TaskNode{
			ID:          ns.ID,
			Description: ns.Description,
			Role:        ns.Role,
			DependsOn:   ns.DependsOn,
		}
		if err := dag.AddNode(node); err != nil {
			return tools.ErrorResult(fmt.Sprintf("DAG build error: %v", err))
		}
	}

	// SWE100821: Execute DAG with LLM-based executor (each node runs as mini subagent)
	executor := func(execCtx context.Context, node *orchestration.TaskNode) (string, error) {
		role := orchestration.GetRole(node.Role)
		sysPrompt := "Complete this subtask concisely."
		if role != nil {
			sysPrompt = role.SystemPrompt
		}

		subResp, subErr := t.provider.Chat(execCtx, []providers.Message{
			{Role: "system", Content: sysPrompt},
			{Role: "user", Content: node.Description},
		}, nil, t.model, map[string]interface{}{
			"max_tokens":  2048,
			"temperature": 0.4,
		})
		if subErr != nil {
			return "", subErr
		}
		return subResp.Content, nil
	}

	if err := dag.Execute(ctx, executor, 3); err != nil {
		logger.WarnCF("decompose", "DAG execution error", map[string]interface{}{"error": err.Error()})
		return tools.ErrorResult(fmt.Sprintf("DAG execution failed: %v\n\n%s", err, dag.Summary()))
	}

	// SWE100821: Aggregate results via orchestration.Aggregator
	agg := orchestration.NewAggregator(t.provider, t.model)
	merged, err := agg.MergeDAGResults(ctx, dag)
	if err != nil {
		return &tools.ToolResult{ForLLM: dag.Summary(), IsError: false}
	}

	return &tools.ToolResult{
		ForLLM:  fmt.Sprintf("## Decomposition Results\n\n%s\n\n---\n%s", merged, dag.Summary()),
		IsError: false,
	}
}
