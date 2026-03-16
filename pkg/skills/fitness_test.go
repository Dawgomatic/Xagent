// SWE100821: Tests for skill fitness scoring — verifies recording, ranking,
// deprecation detection, and save/load persistence.
package skills

import (
	"os"
	"path/filepath"
	"testing"
)

// SWE100821: TestRecordUse_Success — 5 successes, expect TimesUsed=5, TimesSucceeded=5, Score>0
func TestRecordUse_Success(t *testing.T) {
	ft := NewFitnessTracker(t.TempDir())

	for i := 0; i < 5; i++ {
		ft.RecordUse("alpha", true) // SWE100821: record success
	}

	sf := ft.GetFitness("alpha")
	if sf == nil {
		t.Fatal("expected fitness entry for alpha")
	}
	if sf.TimesUsed != 5 {
		t.Errorf("TimesUsed = %d, want 5", sf.TimesUsed)
	}
	if sf.TimesSucceeded != 5 {
		t.Errorf("TimesSucceeded = %d, want 5", sf.TimesSucceeded)
	}
	if sf.Score <= 0 {
		t.Errorf("Score = %f, want > 0", sf.Score)
	}
}

// SWE100821: TestRecordUse_Failure — 3 failures, verify TimesFailed=3
func TestRecordUse_Failure(t *testing.T) {
	ft := NewFitnessTracker(t.TempDir())

	for i := 0; i < 3; i++ {
		ft.RecordUse("beta", false) // SWE100821: record failure
	}

	sf := ft.GetFitness("beta")
	if sf == nil {
		t.Fatal("expected fitness entry for beta")
	}
	if sf.TimesFailed != 3 {
		t.Errorf("TimesFailed = %d, want 3", sf.TimesFailed)
	}
	if sf.TimesUsed != 3 {
		t.Errorf("TimesUsed = %d, want 3", sf.TimesUsed)
	}
}

// SWE100821: TestGetTopSkills — 5 skills with varying success, verify top 3 order
func TestGetTopSkills(t *testing.T) {
	ft := NewFitnessTracker(t.TempDir())

	// SWE100821: Create skills with increasing success counts so scores differ
	skills := []struct {
		name     string
		success  int
		failures int
	}{
		{"worst", 1, 9},
		{"bad", 2, 8},
		{"mid", 5, 5},
		{"good", 8, 2},
		{"best", 10, 0},
	}

	for _, s := range skills {
		for i := 0; i < s.success; i++ {
			ft.RecordUse(s.name, true)
		}
		for i := 0; i < s.failures; i++ {
			ft.RecordUse(s.name, false)
		}
	}

	top := ft.GetTopSkills(3)
	if len(top) != 3 {
		t.Fatalf("GetTopSkills(3) returned %d, want 3", len(top))
	}
	// SWE100821: "best" must rank first — 100% success rate with 10 uses
	if top[0] != "best" {
		t.Errorf("top[0] = %q, want \"best\"", top[0])
	}
	if top[1] != "good" {
		t.Errorf("top[1] = %q, want \"good\"", top[1])
	}
}

// SWE100821: TestGetDeprecated — skill with 15 uses and only 1 success → low score → deprecated
func TestGetDeprecated(t *testing.T) {
	ft := NewFitnessTracker(t.TempDir())

	ft.RecordUse("deprecated_skill", true) // SWE100821: 1 success
	for i := 0; i < 30; i++ {
		ft.RecordUse("deprecated_skill", false) // SWE100821: 30 failures → score < 0.2
	}

	deprecated := ft.GetDeprecated()
	found := false
	for _, name := range deprecated {
		if name == "deprecated_skill" {
			found = true
		}
	}
	if !found {
		sf := ft.GetFitness("deprecated_skill")
		t.Errorf("expected deprecated_skill in deprecated list; score=%.4f, uses=%d", sf.Score, sf.TimesUsed)
	}
}

// SWE100821: TestSaveLoad — round-trip persistence to temp dir
func TestSaveLoad(t *testing.T) {
	dir := t.TempDir()
	ft1 := NewFitnessTracker(dir)

	for i := 0; i < 7; i++ {
		ft1.RecordUse("persist_skill", true)
	}
	ft1.RecordUse("persist_skill", false)

	if err := ft1.Save(); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	// SWE100821: Verify file exists on disk
	jsonPath := filepath.Join(dir, "skills", "fitness.json")
	if _, err := os.Stat(jsonPath); err != nil {
		t.Fatalf("fitness.json not found: %v", err)
	}

	// SWE100821: Load into a fresh tracker and compare
	ft2 := NewFitnessTracker(dir)
	sf := ft2.GetFitness("persist_skill")
	if sf == nil {
		t.Fatal("expected persist_skill after load")
	}
	if sf.TimesUsed != 8 {
		t.Errorf("loaded TimesUsed = %d, want 8", sf.TimesUsed)
	}
	if sf.TimesSucceeded != 7 {
		t.Errorf("loaded TimesSucceeded = %d, want 7", sf.TimesSucceeded)
	}
}
