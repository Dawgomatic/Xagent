// SWE100821: Tests for skill composition suggestions — verifies co-occurrence
// detection from provenance JSONL logs.
package skills

import (
	"os"
	"path/filepath"
	"testing"
)

// SWE100821: TestSuggestCompositions — mock provenance JSONL with co-occurring
// skills appearing >3 times, verify suggestion returned.
func TestSuggestCompositions(t *testing.T) {
	dir := t.TempDir()
	provPath := filepath.Join(dir, "provenance.jsonl")

	// SWE100821: 4 sessions with skills "git-ops" and "code-review" co-occurring → threshold >3
	lines := ""
	for i := 0; i < 4; i++ {
		lines += `{"session_id":"s` + string(rune('0'+i)) + `","skills":["git-ops","code-review","deploy"]}` + "\n"
	}
	if err := os.WriteFile(provPath, []byte(lines), 0644); err != nil {
		t.Fatalf("writing provenance: %v", err)
	}

	sc := NewSkillComposer(dir)
	suggestions := sc.SuggestCompositions(provPath)

	if len(suggestions) == 0 {
		t.Fatal("expected at least one composition suggestion")
	}

	// SWE100821: Verify at least the git-ops + code-review pair is suggested
	found := false
	for _, pair := range suggestions {
		if len(pair) == 2 {
			a, b := pair[0], pair[1]
			if (a == "code-review" && b == "git-ops") || (a == "git-ops" && b == "code-review") {
				found = true
			}
		}
	}
	if !found {
		t.Errorf("expected git-ops + code-review pair in suggestions, got %v", suggestions)
	}
}

// SWE100821: TestSuggestCompositions_NoCoPairs — no co-occurrences above threshold
func TestSuggestCompositions_NoCoPairs(t *testing.T) {
	dir := t.TempDir()
	provPath := filepath.Join(dir, "provenance.jsonl")

	// SWE100821: Each session uses a different single skill → no pairs
	lines := `{"session_id":"s1","skills":["alpha"]}
{"session_id":"s2","skills":["beta"]}
{"session_id":"s3","skills":["gamma"]}
`
	if err := os.WriteFile(provPath, []byte(lines), 0644); err != nil {
		t.Fatalf("writing provenance: %v", err)
	}

	sc := NewSkillComposer(dir)
	suggestions := sc.SuggestCompositions(provPath)

	if len(suggestions) != 0 {
		t.Errorf("expected 0 suggestions for non-co-occurring skills, got %d: %v", len(suggestions), suggestions)
	}
}
