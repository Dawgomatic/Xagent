// SWE100821: Plan-Act-Reflect loop with chain-of-thought scratchpad.
// Upgrades the basic ReAct loop to generate a plan before acting, then
// reflect on each tool result against the plan. Replans on dead-ends.
//
// The scratchpad is internal reasoning that persists across tool calls
// but is NOT sent to the user — it acts as working memory.

package agent

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Dawgomatic/Xagent/pkg/logger"
	"github.com/Dawgomatic/Xagent/pkg/providers"
)

// PlanStep represents one step in the agent's execution plan.
type PlanStep struct {
	Index       int    `json:"index"`
	Description string `json:"description"`
	Tool        string `json:"tool,omitempty"` // expected tool, if known
	Status      string `json:"status"`         // pending, in_progress, completed, failed, skipped
}

// AgentPlan holds the multi-step plan and scratchpad for a single turn.
type AgentPlan struct {
	Goal       string      `json:"goal"`
	Steps      []PlanStep  `json:"steps"`
	Scratchpad string      `json:"scratchpad"` // chain-of-thought working memory
	CreatedAt  time.Time   `json:"created_at"`
	Replans    int         `json:"replans"`
}

// Planner generates and manages execution plans for the agent loop.
type Planner struct {
	provider    providers.LLMProvider
	model       string
	maxTokens   int
	temperature float64
}

// NewPlanner creates a planner using the given LLM provider.
func NewPlanner(provider providers.LLMProvider, model string, maxTokens int, temp float64) *Planner {
	return &Planner{
		provider:    provider,
		model:       model,
		maxTokens:   maxTokens,
		temperature: temp,
	}
}

// GeneratePlan asks the LLM to produce a multi-step plan for the user's request.
// Returns a structured plan that the agent loop can track progress against.
func (p *Planner) GeneratePlan(ctx context.Context, userMessage string, toolSummaries []string) (*AgentPlan, error) {
	toolList := strings.Join(toolSummaries, "\n")

	planPrompt := fmt.Sprintf(`You are a planning module. Given the user's request and available tools, create a concise step-by-step plan.

Available tools:
%s

User request: %s

Respond with ONLY a numbered plan (1-5 steps max). Each step should be one sentence.
If the request is simple (can be answered directly), respond with just:
1. Respond directly to the user

Format:
1. <step description>
2. <step description>
...`, toolList, userMessage)

	resp, err := p.provider.Chat(ctx, []providers.Message{
		{Role: "user", Content: planPrompt},
	}, nil, p.model, map[string]interface{}{
		"max_tokens":  512,
		"temperature": 0.3,
	})
	if err != nil {
		return nil, fmt.Errorf("plan generation failed: %w", err)
	}

	plan := &AgentPlan{
		Goal:      userMessage,
		Steps:     parsePlanSteps(resp.Content),
		CreatedAt: time.Now(),
	}

	logger.InfoCF("planner", "Plan generated",
		map[string]interface{}{
			"steps": len(plan.Steps),
			"goal":  userMessage[:minInt(len(userMessage), 80)],
		})

	return plan, nil
}

// Reflect asks the LLM to evaluate progress after a tool result.
// Returns updated scratchpad content and whether to replan.
func (p *Planner) Reflect(ctx context.Context, plan *AgentPlan, toolName, toolResult string) (scratchpad string, shouldReplan bool, err error) {
	reflectPrompt := fmt.Sprintf(`You are a reflection module. Evaluate the tool result against the plan.

Goal: %s
Current plan step: %s
Tool used: %s
Tool result (first 500 chars): %s

Scratchpad so far: %s

Respond in this format:
PROGRESS: <one sentence on what was accomplished>
SCRATCHPAD: <updated working notes — key findings, next considerations>
REPLAN: <yes/no — only yes if the current approach is clearly failing>`,
		plan.Goal,
		currentStepDescription(plan),
		toolName,
		truncate(toolResult, 500),
		plan.Scratchpad,
	)

	resp, err := p.provider.Chat(ctx, []providers.Message{
		{Role: "user", Content: reflectPrompt},
	}, nil, p.model, map[string]interface{}{
		"max_tokens":  256,
		"temperature": 0.2,
	})
	if err != nil {
		return plan.Scratchpad, false, err
	}

	content := resp.Content
	scratchpad = plan.Scratchpad
	shouldReplan = false

	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "SCRATCHPAD:") {
			scratchpad = strings.TrimPrefix(line, "SCRATCHPAD:")
			scratchpad = strings.TrimSpace(scratchpad)
		}
		if strings.HasPrefix(line, "REPLAN:") {
			val := strings.TrimSpace(strings.TrimPrefix(line, "REPLAN:"))
			shouldReplan = strings.EqualFold(val, "yes")
		}
	}

	return scratchpad, shouldReplan, nil
}

// ForSystemPrompt formats the current plan for injection into the LLM context.
func (plan *AgentPlan) ForSystemPrompt() string {
	if plan == nil || len(plan.Steps) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("## Current Plan\n\n")
	sb.WriteString(fmt.Sprintf("Goal: %s\n\n", plan.Goal))

	for _, step := range plan.Steps {
		marker := ""
		switch step.Status {
		case "completed":
			marker = ""
		case "in_progress":
			marker = ""
		case "failed":
			marker = ""
		case "skipped":
			marker = ""
		}
		sb.WriteString(fmt.Sprintf("%s %d. %s\n", marker, step.Index, step.Description))
	}

	if plan.Scratchpad != "" {
		sb.WriteString(fmt.Sprintf("\n### Working Notes\n%s\n", plan.Scratchpad))
	}

	return sb.String()
}

// AdvanceStep marks the current step as completed and moves to the next.
func (plan *AgentPlan) AdvanceStep() {
	for i := range plan.Steps {
		if plan.Steps[i].Status == "in_progress" {
			plan.Steps[i].Status = "completed"
			if i+1 < len(plan.Steps) {
				plan.Steps[i+1].Status = "in_progress"
			}
			return
		}
	}
}

// MarkCurrentFailed marks the current step as failed.
func (plan *AgentPlan) MarkCurrentFailed() {
	for i := range plan.Steps {
		if plan.Steps[i].Status == "in_progress" {
			plan.Steps[i].Status = "failed"
			return
		}
	}
}

// IsComplete returns true if all steps are completed or skipped.
func (plan *AgentPlan) IsComplete() bool {
	for _, step := range plan.Steps {
		if step.Status == "pending" || step.Status == "in_progress" {
			return false
		}
	}
	return true
}

func parsePlanSteps(content string) []PlanStep {
	var steps []PlanStep
	idx := 1
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// Strip leading number + period
		for i, c := range line {
			if c == '.' && i > 0 {
				line = strings.TrimSpace(line[i+1:])
				break
			}
			if c < '0' || c > '9' {
				break
			}
		}
		if line == "" {
			continue
		}
		status := "pending"
		if idx == 1 {
			status = "in_progress"
		}
		steps = append(steps, PlanStep{
			Index:       idx,
			Description: line,
			Status:      status,
		})
		idx++
	}
	return steps
}

func currentStepDescription(plan *AgentPlan) string {
	for _, step := range plan.Steps {
		if step.Status == "in_progress" {
			return step.Description
		}
	}
	if len(plan.Steps) > 0 {
		return plan.Steps[len(plan.Steps)-1].Description
	}
	return "unknown"
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
