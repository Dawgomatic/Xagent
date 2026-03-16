// SWE100821: Tests for feedback tool — reward signal calculation, rating parsing,
// persistence round-trip, and summary generation.

package tools

import (
	"context"
	"strings"
	"testing"
)

// SWE100821: TestFeedbackTool_Name verifies tool identity
func TestFeedbackTool_Name(t *testing.T) {
	ft := NewFeedbackTool(t.TempDir())
	if ft.Name() != "feedback" {
		t.Errorf("expected 'feedback', got '%s'", ft.Name())
	}
}

// SWE100821: TestFeedbackTool_Execute_Good verifies positive rating
func TestFeedbackTool_Execute_Good(t *testing.T) {
	ft := NewFeedbackTool(t.TempDir())
	result := ft.Execute(context.Background(), map[string]interface{}{
		"rating":  "good",
		"comment": "great response",
	})
	if result.IsError {
		t.Fatalf("expected success, got error: %s", result.ForLLM)
	}
	if !strings.Contains(result.ForLLM, "Feedback recorded") {
		t.Errorf("expected confirmation in ForLLM, got: %s", result.ForLLM)
	}
}

// SWE100821: TestFeedbackTool_Execute_Bad verifies negative rating
func TestFeedbackTool_Execute_Bad(t *testing.T) {
	ft := NewFeedbackTool(t.TempDir())
	result := ft.Execute(context.Background(), map[string]interface{}{
		"rating": "bad",
	})
	if result.IsError {
		t.Fatalf("expected success, got error: %s", result.ForLLM)
	}
}

// SWE100821: TestGetRewardSignal_NoFeedback — empty file returns 0
func TestGetRewardSignal_NoFeedback(t *testing.T) {
	ft := NewFeedbackTool(t.TempDir())
	signal := ft.GetRewardSignal()
	if signal != 0.0 {
		t.Errorf("expected 0.0 for no feedback, got %f", signal)
	}
}

// SWE100821: TestGetRewardSignal_MixedFeedback — verifies calculation
func TestGetRewardSignal_MixedFeedback(t *testing.T) {
	dir := t.TempDir()
	ft := NewFeedbackTool(dir)
	ctx := context.Background()

	// 3 good, 1 bad, 1 neutral = (3-1)/5 = 0.4
	ft.Execute(ctx, map[string]interface{}{"rating": "good"})
	ft.Execute(ctx, map[string]interface{}{"rating": "good"})
	ft.Execute(ctx, map[string]interface{}{"rating": "good"})
	ft.Execute(ctx, map[string]interface{}{"rating": "bad"})
	ft.Execute(ctx, map[string]interface{}{"rating": "neutral"})

	signal := ft.GetRewardSignal()
	expected := 0.4
	if signal < expected-0.01 || signal > expected+0.01 {
		t.Errorf("expected reward signal ~%.1f, got %f", expected, signal)
	}
}

// SWE100821: TestGetRewardSignal_AllNegative — -1.0
func TestGetRewardSignal_AllNegative(t *testing.T) {
	dir := t.TempDir()
	ft := NewFeedbackTool(dir)
	ctx := context.Background()

	ft.Execute(ctx, map[string]interface{}{"rating": "bad"})
	ft.Execute(ctx, map[string]interface{}{"rating": "bad"})

	signal := ft.GetRewardSignal()
	if signal != -1.0 {
		t.Errorf("expected -1.0, got %f", signal)
	}
}

// SWE100821: TestGetFeedbackSummary — counts and comments
func TestGetFeedbackSummary(t *testing.T) {
	ft := NewFeedbackTool(t.TempDir())
	ctx := context.Background()

	ft.Execute(ctx, map[string]interface{}{"rating": "good", "comment": "nice work"})
	ft.Execute(ctx, map[string]interface{}{"rating": "bad", "comment": "too verbose"})

	summary := ft.GetFeedbackSummary()
	if !strings.Contains(summary, "1 positive") {
		t.Errorf("expected '1 positive' in summary, got: %s", summary)
	}
	if !strings.Contains(summary, "1 negative") {
		t.Errorf("expected '1 negative' in summary, got: %s", summary)
	}
	if !strings.Contains(summary, "nice work") {
		t.Errorf("expected comment in summary, got: %s", summary)
	}
}

// SWE100821: TestGetRecentFeedback — buffer returns entries
func TestGetRecentFeedback(t *testing.T) {
	ft := NewFeedbackTool(t.TempDir())
	ctx := context.Background()

	ft.Execute(ctx, map[string]interface{}{"rating": "good"})
	ft.Execute(ctx, map[string]interface{}{"rating": "bad"})

	entries := ft.GetRecentFeedback()
	if len(entries) != 2 {
		t.Errorf("expected 2 entries, got %d", len(entries))
	}
}
