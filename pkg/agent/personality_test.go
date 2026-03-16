// SWE100821: Tests for personality tracker (Observe, Analyze, ForSystemPrompt, profile round-trip).

package agent

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// SWE100821: Observe accumulates observations and flushes at 100
func TestObserve_CountAndFlush(t *testing.T) {
	pt := NewPersonalityTracker(t.TempDir(), &mockProvider{}, "mock")

	for i := 0; i < 110; i++ {
		pt.Observe(50, 200, []string{"shell"})
	}
	// SWE100821: flush fires at 101 (→50), then 9 more are added = 59
	if len(pt.observations) != 59 {
		t.Errorf("expected 59 observations after flush, got %d", len(pt.observations))
	}
}

// SWE100821: Analyze with <10 observations returns profile without changes
func TestAnalyze_InsufficientData(t *testing.T) {
	pt := NewPersonalityTracker(t.TempDir(), &mockProvider{}, "mock")

	for i := 0; i < 5; i++ {
		pt.Observe(50, 200, nil)
	}

	profile, err := pt.Analyze(context.Background())
	if err != nil {
		t.Fatalf("Analyze failed: %v", err)
	}
	// SWE100821: not enough data → no adaptations generated
	if len(profile.Adaptations) != 0 {
		t.Errorf("expected 0 adaptations with insufficient data, got %d", len(profile.Adaptations))
	}
}

// SWE100821: Analyze with sufficient data computes verbosity
func TestAnalyze_Verbosity(t *testing.T) {
	dir := t.TempDir()
	pt := NewPersonalityTracker(dir, &mockProvider{}, "mock")

	// SWE100821: short user messages → "short" preferred length
	for i := 0; i < 15; i++ {
		pt.Observe(30, 100, []string{"exec"})
	}

	profile, err := pt.Analyze(context.Background())
	if err != nil {
		t.Fatalf("Analyze failed: %v", err)
	}
	if profile.PreferredLen != "short" {
		t.Errorf("expected preferred_len 'short' for avg user len <50, got %q", profile.PreferredLen)
	}
	// SWE100821: verbosity = avgAgentLen/500 = 100/500 = 0.2
	v := profile.Traits["verbosity"]
	if v < 0.1 || v > 0.3 {
		t.Errorf("expected verbosity ~0.2, got %f", v)
	}
}

// SWE100821: ForSystemPrompt returns empty string with no data
func TestForSystemPrompt_NoData(t *testing.T) {
	pt := NewPersonalityTracker(t.TempDir(), &mockProvider{}, "mock")
	output := pt.ForSystemPrompt()
	if output != "" {
		t.Errorf("expected empty prompt with no data, got %q", output)
	}
}

// SWE100821: loadProfile/saveProfile round-trip preserves data
func TestProfileRoundTrip(t *testing.T) {
	dir := t.TempDir()
	pt := NewPersonalityTracker(dir, &mockProvider{}, "mock")

	original := &PersonalityProfile{
		Traits:       map[string]float64{"verbosity": 0.7, "formality": 0.3, "creativity": 0.5},
		PreferredLen: "long",
		TotalSamples: 42,
	}

	pt.saveProfile(original)

	// SWE100821: verify file was created
	profilePath := filepath.Join(dir, "state", "personality.json")
	data, err := os.ReadFile(profilePath)
	if err != nil {
		t.Fatalf("profile file not created: %v", err)
	}

	var loaded PersonalityProfile
	if err := json.Unmarshal(data, &loaded); err != nil {
		t.Fatalf("failed to unmarshal profile: %v", err)
	}
	if loaded.TotalSamples != 42 {
		t.Errorf("expected TotalSamples=42, got %d", loaded.TotalSamples)
	}
	if loaded.Traits["verbosity"] != 0.7 {
		t.Errorf("expected verbosity=0.7, got %f", loaded.Traits["verbosity"])
	}

	// SWE100821: verify loadProfile reads it back correctly
	reloaded := pt.loadProfile()
	if reloaded.PreferredLen != "long" {
		t.Errorf("expected preferred_len 'long' after reload, got %q", reloaded.PreferredLen)
	}
}
