// SWE100821: Tests for temporal memory index — time-range queries, topic filtering,
// persistence round-trip, natural-language temporal resolution, and pruning.

package memory

import (
	"strings"
	"testing"
	"time"
)

// SWE100821: TestAdd_AndQuery — add 5 entries over 7 days, query full range
func TestAdd_AndQuery(t *testing.T) {
	dir := t.TempDir()
	ti := NewTemporalIndex(dir)
	now := time.Now()

	for i := 0; i < 5; i++ {
		ti.Add(TemporalEntry{
			Timestamp:  now.Add(-time.Duration(i) * 24 * time.Hour),
			TopicTags:  []string{"test"},
			SessionKey: "sess",
			Summary:    "entry",
			Source:      "test",
		})
	}

	since := now.Add(-8 * 24 * time.Hour)
	until := now.Add(time.Hour)
	results := ti.Query(since, until, "")
	if len(results) != 5 {
		t.Errorf("expected 5 entries, got %d", len(results))
	}
}

// SWE100821: TestQuery_TopicFilter — filter by topic tag
func TestQuery_TopicFilter(t *testing.T) {
	dir := t.TempDir()
	ti := NewTemporalIndex(dir)
	now := time.Now()

	ti.Add(TemporalEntry{Timestamp: now, TopicTags: []string{"coding"}, Summary: "wrote tests"})
	ti.Add(TemporalEntry{Timestamp: now, TopicTags: []string{"fitness"}, Summary: "ran 5k"})
	ti.Add(TemporalEntry{Timestamp: now, TopicTags: []string{"coding", "review"}, Summary: "code review"})

	since := now.Add(-time.Hour)
	until := now.Add(time.Hour)

	codingResults := ti.Query(since, until, "coding")
	if len(codingResults) != 2 {
		t.Errorf("expected 2 coding entries, got %d", len(codingResults))
	}

	fitnessResults := ti.Query(since, until, "fitness")
	if len(fitnessResults) != 1 {
		t.Errorf("expected 1 fitness entry, got %d", len(fitnessResults))
	}
}

// SWE100821: TestSaveLoad — round-trip persistence to temp file
func TestSaveLoad(t *testing.T) {
	dir := t.TempDir()
	ti := NewTemporalIndex(dir)
	now := time.Now()

	ti.Add(TemporalEntry{
		Timestamp:  now,
		TopicTags:  []string{"persist"},
		SessionKey: "s1",
		Summary:    "saved entry",
		Source:      "test",
	})

	if err := ti.Save(); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// SWE100821: Load into fresh index and verify
	ti2 := NewTemporalIndex(dir)
	results := ti2.Query(now.Add(-time.Hour), now.Add(time.Hour), "")
	if len(results) != 1 {
		t.Errorf("expected 1 entry after reload, got %d", len(results))
	}
	if results[0].Summary != "saved entry" {
		t.Errorf("expected summary 'saved entry', got '%s'", results[0].Summary)
	}
}

// SWE100821: TestForSystemPrompt_Yesterday — entry timestamped yesterday appears in "yesterday" query
func TestForSystemPrompt_Yesterday(t *testing.T) {
	dir := t.TempDir()
	ti := NewTemporalIndex(dir)

	yesterday := time.Now().Add(-24 * time.Hour)
	ti.Add(TemporalEntry{
		Timestamp: yesterday,
		TopicTags: []string{"deploy"},
		Summary:   "deployed v2.0",
		Source:     "test",
	})

	output := ti.ForSystemPrompt("what happened yesterday")
	if output == "" {
		t.Error("expected non-empty output for 'yesterday' query")
	}
	if !strings.Contains(output, "deployed v2.0") {
		t.Errorf("expected output to contain 'deployed v2.0', got: %s", output)
	}
}

// SWE100821: TestForSystemPrompt_NoTemporalRef — query without temporal ref uses 7-day default
// Adding no entries within that range should return empty.
func TestForSystemPrompt_NoTemporalRef(t *testing.T) {
	dir := t.TempDir()
	ti := NewTemporalIndex(dir)

	// Add an entry from 30 days ago — outside the 7-day default window
	ti.Add(TemporalEntry{
		Timestamp: time.Now().Add(-30 * 24 * time.Hour),
		TopicTags: []string{"old"},
		Summary:   "ancient event",
		Source:     "test",
	})

	output := ti.ForSystemPrompt("hello")
	if output != "" {
		t.Errorf("expected empty for 'hello' with no recent entries, got: %s", output)
	}
}

// SWE100821: TestPrune — adding entries beyond cap triggers pruning.
// Uses direct slice manipulation to avoid 10k disk writes.
func TestPrune(t *testing.T) {
	dir := t.TempDir()
	ti := NewTemporalIndex(dir)
	now := time.Now()

	// SWE100821: Pre-fill entries directly to avoid O(n) disk I/O per Add
	ti.mu.Lock()
	for i := 0; i < maxTemporalEntries; i++ {
		ti.entries = append(ti.entries, TemporalEntry{
			Timestamp: now.Add(-time.Duration(i) * time.Second),
			TopicTags: []string{"bulk"},
			Summary:   "entry",
			Source:     "test",
		})
	}
	ti.mu.Unlock()

	// Add one more via public API to trigger prune
	ti.Add(TemporalEntry{
		Timestamp: now.Add(time.Second),
		TopicTags: []string{"overflow"},
		Summary:   "overflow entry",
		Source:     "test",
	})

	since := now.Add(-time.Duration(maxTemporalEntries+10) * time.Second)
	until := now.Add(time.Hour)
	results := ti.Query(since, until, "")
	if len(results) > maxTemporalEntries {
		t.Errorf("expected at most %d entries after prune, got %d", maxTemporalEntries, len(results))
	}
}
