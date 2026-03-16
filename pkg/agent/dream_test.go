// SWE100821: Tests for dream mode parsing (parseDreamResult, parseWorldModelUpdates, updateWorldModel).

package agent

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// SWE100821: parseDreamResult extracts patterns, insights, questions from structured LLM output
func TestParseDreamResult(t *testing.T) {
	input := `PATTERNS:
- Users ask about Go testing frequently
- Config questions spike on Mondays

INSIGHTS:
- The agent could pre-load testing docs on startup
- Monday spikes correlate with deployment cycles

QUESTIONS:
- Why does the user avoid shell tools?
- Is there a preferred test framework?`

	result := parseDreamResult(input, 5)

	if len(result.Patterns) != 2 {
		t.Fatalf("expected 2 patterns, got %d", len(result.Patterns))
	}
	if !strings.Contains(result.Patterns[0], "Go testing") {
		t.Errorf("expected first pattern about Go testing, got %q", result.Patterns[0])
	}

	if len(result.Insights) != 2 {
		t.Fatalf("expected 2 insights, got %d", len(result.Insights))
	}
	if !strings.Contains(result.Insights[1], "Monday") {
		t.Errorf("expected second insight about Monday, got %q", result.Insights[1])
	}

	if len(result.Questions) != 2 {
		t.Fatalf("expected 2 questions, got %d", len(result.Questions))
	}
	if result.InputNotes != 5 {
		t.Errorf("expected InputNotes=5, got %d", result.InputNotes)
	}
}

// SWE100821: parseDreamResult with empty input returns empty slices
func TestParseDreamResult_Empty(t *testing.T) {
	result := parseDreamResult("", 0)
	if len(result.Patterns) != 0 || len(result.Insights) != 0 || len(result.Questions) != 0 {
		t.Error("expected empty result for empty input")
	}
}

// SWE100821: parseWorldModelUpdates extracts ADD/UPDATE/DEPRECATE lines
func TestParseWorldModelUpdates(t *testing.T) {
	input := `PATTERNS:
- some pattern

WORLD_MODEL_UPDATES:
ADD: Users prefer terse responses
UPDATE: Shell tool usage is increasing -> Shell is the primary tool
DEPRECATE: Verbose explanations are preferred

Some other text`

	updates := parseWorldModelUpdates(input)
	if !strings.Contains(updates, "ADD: Users prefer terse responses") {
		t.Errorf("expected ADD line, got %q", updates)
	}
	if !strings.Contains(updates, "UPDATE: Shell tool usage") {
		t.Errorf("expected UPDATE line, got %q", updates)
	}
	if !strings.Contains(updates, "DEPRECATE: Verbose") {
		t.Errorf("expected DEPRECATE line, got %q", updates)
	}
}

// SWE100821: parseWorldModelUpdates returns empty for input with no WORLD_MODEL_UPDATES section
func TestParseWorldModelUpdates_NoSection(t *testing.T) {
	input := "PATTERNS:\n- p1\nINSIGHTS:\n- i1"
	updates := parseWorldModelUpdates(input)
	if updates != "" {
		t.Errorf("expected empty updates for input without section, got %q", updates)
	}
}

// SWE100821: updateWorldModel creates/appends to WORLD_MODEL.md in temp directory
func TestUpdateWorldModel(t *testing.T) {
	dir := t.TempDir()
	content := "ADD: New causal belief\nUPDATE: Old -> New\n"

	// SWE100821: first write — creates the file
	if err := updateWorldModel(dir, content); err != nil {
		t.Fatalf("updateWorldModel failed: %v", err)
	}

	wmPath := filepath.Join(dir, "WORLD_MODEL.md")
	data, err := os.ReadFile(wmPath)
	if err != nil {
		t.Fatalf("WORLD_MODEL.md not created: %v", err)
	}
	if !strings.Contains(string(data), "ADD: New causal belief") {
		t.Errorf("expected ADD line in file, got %q", string(data))
	}
	if !strings.Contains(string(data), "Dream Update") {
		t.Errorf("expected 'Dream Update' header in file, got %q", string(data))
	}

	// SWE100821: second write — appends to existing
	content2 := "DEPRECATE: Stale belief\n"
	if err := updateWorldModel(dir, content2); err != nil {
		t.Fatalf("second updateWorldModel failed: %v", err)
	}

	data2, err := os.ReadFile(wmPath)
	if err != nil {
		t.Fatalf("failed to read updated WORLD_MODEL.md: %v", err)
	}
	if !strings.Contains(string(data2), "ADD: New causal belief") {
		t.Error("original content should still be present after append")
	}
	if !strings.Contains(string(data2), "DEPRECATE: Stale belief") {
		t.Error("appended content should be present")
	}
}
