// SWE100821: Tests for Plan-Act-Reflect planner (parsePlanSteps, AdvanceStep, etc.)

package agent

import (
	"context"
	"strings"
	"testing"
)

// SWE100821: Tests for Plan-Act-Reflect planner
func TestParsePlanSteps(t *testing.T) {
	input := "1. Read the file\n2. Parse contents\n3. Return result"
	steps := parsePlanSteps(input)
	if len(steps) != 3 {
		t.Fatalf("expected 3 steps, got %d", len(steps))
	}
	// SWE100821: first step should auto-start
	if steps[0].Status != "in_progress" {
		t.Errorf("first step should be in_progress, got %s", steps[0].Status)
	}
	if steps[1].Status != "pending" {
		t.Errorf("second step should be pending, got %s", steps[1].Status)
	}
	if steps[0].Index != 1 || steps[2].Index != 3 {
		t.Errorf("unexpected indices: %d, %d", steps[0].Index, steps[2].Index)
	}
}

// SWE100821: verify AdvanceStep transitions in_progress→completed and promotes next
func TestAdvanceStep(t *testing.T) {
	plan := &AgentPlan{
		Steps: []PlanStep{
			{Index: 1, Status: "in_progress"},
			{Index: 2, Status: "pending"},
			{Index: 3, Status: "pending"},
		},
	}
	plan.AdvanceStep()
	if plan.Steps[0].Status != "completed" {
		t.Errorf("step 1 should be completed, got %s", plan.Steps[0].Status)
	}
	if plan.Steps[1].Status != "in_progress" {
		t.Errorf("step 2 should be in_progress, got %s", plan.Steps[1].Status)
	}
}

// SWE100821: verify MarkCurrentFailed sets in_progress step to failed
func TestMarkCurrentFailed(t *testing.T) {
	plan := &AgentPlan{
		Steps: []PlanStep{
			{Index: 1, Status: "completed"},
			{Index: 2, Status: "in_progress"},
		},
	}
	plan.MarkCurrentFailed()
	if plan.Steps[1].Status != "failed" {
		t.Errorf("step 2 should be failed, got %s", plan.Steps[1].Status)
	}
}

// SWE100821: verify IsComplete returns true only when all steps done
func TestIsComplete(t *testing.T) {
	plan := &AgentPlan{
		Steps: []PlanStep{
			{Index: 1, Status: "completed"},
			{Index: 2, Status: "completed"},
		},
	}
	if !plan.IsComplete() {
		t.Error("expected plan to be complete")
	}

	plan.Steps[1].Status = "pending"
	if plan.IsComplete() {
		t.Error("expected plan to NOT be complete with pending step")
	}
}

// SWE100821: verify ForSystemPrompt produces expected sections
func TestForSystemPrompt(t *testing.T) {
	plan := &AgentPlan{
		Goal: "Test the system",
		Steps: []PlanStep{
			{Index: 1, Description: "Read files", Status: "completed"},
			{Index: 2, Description: "Write tests", Status: "in_progress"},
		},
		Scratchpad: "Found 3 files",
	}
	output := plan.ForSystemPrompt()
	if output == "" {
		t.Error("expected non-empty prompt")
	}
	if !strings.Contains(output, "Test the system") {
		t.Error("expected goal in prompt")
	}
	if !strings.Contains(output, "Working Notes") {
		t.Error("expected scratchpad section")
	}
}

// SWE100821: verify Reflect with mock returns no replan
func TestReflect(t *testing.T) {
	p := NewPlanner(&mockProvider{}, "test", 512, 0.3)
	ctx := context.Background()
	plan := &AgentPlan{
		Goal:  "test goal",
		Steps: []PlanStep{{Index: 1, Description: "step 1", Status: "in_progress"}},
	}
	_, shouldReplan, err := p.Reflect(ctx, plan, "exec", "success output")
	if err != nil {
		t.Fatalf("Reflect failed: %v", err)
	}
	// SWE100821: mockProvider returns "Mock response" — no REPLAN: yes line
	if shouldReplan {
		t.Error("should not replan with mock response")
	}
}

// SWE100821: nil plan should return empty string
func TestForSystemPrompt_NilPlan(t *testing.T) {
	var plan *AgentPlan
	output := plan.ForSystemPrompt()
	if output != "" {
		t.Errorf("nil plan should return empty string, got %s", output)
	}
}

// SWE100821: empty-step plan should return empty string
func TestForSystemPrompt_EmptySteps(t *testing.T) {
	plan := &AgentPlan{Goal: "test", Steps: []PlanStep{}}
	output := plan.ForSystemPrompt()
	if output != "" {
		t.Errorf("empty steps should return empty string, got %s", output)
	}
}

// SWE100821: AdvanceStep on last step should mark completed without panic
func TestAdvanceStep_LastStep(t *testing.T) {
	plan := &AgentPlan{
		Steps: []PlanStep{
			{Index: 1, Status: "in_progress"},
		},
	}
	plan.AdvanceStep()
	if plan.Steps[0].Status != "completed" {
		t.Errorf("last step should be completed, got %s", plan.Steps[0].Status)
	}
}
