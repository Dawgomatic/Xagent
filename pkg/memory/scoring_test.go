// SWE100821: Tests for memory importance scoring system.
// Covers recency decay, salience detection, reference scaling, composite scoring, and ranking.

package memory

import (
	"math"
	"testing"
	"time"
)

// SWE100821: TestComputeRecency — recent point ~1.0, 30-day-old < 0.1
func TestComputeRecency(t *testing.T) {
	scorer := DefaultScorer()

	recentPt := MemoryPoint{Created: time.Now(), Text: "recent"}
	recentScored := scorer.Score(recentPt, 0, 0)
	if recentScored.RecencyScore < 0.95 {
		t.Errorf("expected recent recency near 1.0, got %f", recentScored.RecencyScore)
	}

	oldPt := MemoryPoint{Created: time.Now().Add(-30 * 24 * time.Hour), Text: "old"}
	oldScored := scorer.Score(oldPt, 0, 0)
	if oldScored.RecencyScore >= 0.1 {
		t.Errorf("expected 30-day-old recency < 0.1, got %f", oldScored.RecencyScore)
	}
}

// SWE100821: TestComputeSalience — keyword-heavy text scores higher
func TestComputeSalience(t *testing.T) {
	scorer := DefaultScorer()

	highPt := MemoryPoint{Created: time.Now(), Text: "critical error found"}
	highScored := scorer.Score(highPt, 0, 0)

	lowPt := MemoryPoint{Created: time.Now(), Text: "hello world"}
	lowScored := scorer.Score(lowPt, 0, 0)

	if highScored.SalienceScore <= lowScored.SalienceScore {
		t.Errorf("expected 'critical error found' salience (%f) > 'hello world' salience (%f)",
			highScored.SalienceScore, lowScored.SalienceScore)
	}
}

// SWE100821: TestComputeReference — 0 refs = 0, 10 refs > 5 refs
func TestComputeReference(t *testing.T) {
	scorer := DefaultScorer()
	pt := MemoryPoint{Created: time.Now(), Text: "test"}

	zeroRef := scorer.Score(pt, 0, 0)
	if zeroRef.ReferenceScore != 0 {
		t.Errorf("expected 0 refs = 0, got %f", zeroRef.ReferenceScore)
	}

	fiveRef := scorer.Score(pt, 5, 0)
	tenRef := scorer.Score(pt, 10, 0)
	if tenRef.ReferenceScore <= fiveRef.ReferenceScore {
		t.Errorf("expected 10 refs (%f) > 5 refs (%f)", tenRef.ReferenceScore, fiveRef.ReferenceScore)
	}
}

// SWE100821: TestScore_Composite — verify weighted sum with known values
func TestScore_Composite(t *testing.T) {
	scorer := &MemoryScorer{
		RecencyHalfLife:  7 * 24 * time.Hour,
		SalienceKeywords: []string{"critical"},
		Weights: ScoreWeights{
			Recency:   0.25,
			Salience:  0.20,
			Novelty:   0.15,
			Reference: 0.10,
			Semantic:  0.30,
		},
	}

	pt := MemoryPoint{
		Created: time.Now(),
		Text:    "critical alert",
		Score:   0.9,
	}

	scored := scorer.Score(pt, 5, 0.7)

	// Composite = 0.25*recency + 0.20*salience + 0.15*0.7 + 0.10*ref + 0.30*0.9
	// All sub-scores are bounded [0,1], so composite should be reasonable
	if scored.ImportanceScore <= 0 || scored.ImportanceScore > 1.0 {
		t.Errorf("expected composite in (0, 1.0], got %f", scored.ImportanceScore)
	}

	// Verify composite matches manual calculation
	expected := scorer.Weights.Recency*scored.RecencyScore +
		scorer.Weights.Salience*scored.SalienceScore +
		scorer.Weights.Novelty*0.7 +
		scorer.Weights.Reference*scored.ReferenceScore +
		scorer.Weights.Semantic*0.9

	if math.Abs(scored.ImportanceScore-expected) > 1e-9 {
		t.Errorf("composite mismatch: got %f, expected %f", scored.ImportanceScore, expected)
	}
}

// SWE100821: TestRankMemories — verify sorted descending by ImportanceScore
func TestRankMemories(t *testing.T) {
	scorer := DefaultScorer()
	now := time.Now()

	points := []MemoryPoint{
		{ID: 1, Created: now.Add(-10 * 24 * time.Hour), Text: "old boring"},
		{ID: 2, Created: now, Text: "critical error urgent"},
		{ID: 3, Created: now.Add(-2 * 24 * time.Hour), Text: "somewhat recent"},
		{ID: 4, Created: now.Add(-30 * 24 * time.Hour), Text: "ancient"},
		{ID: 5, Created: now, Text: "important success always remember"},
	}

	refs := map[uint64]int{
		1: 0,
		2: 3,
		3: 1,
		4: 0,
		5: 8,
	}

	ranked := scorer.RankMemories(points, refs)

	if len(ranked) != 5 {
		t.Fatalf("expected 5 results, got %d", len(ranked))
	}

	for i := 1; i < len(ranked); i++ {
		if ranked[i].ImportanceScore > ranked[i-1].ImportanceScore {
			t.Errorf("rank[%d] score %f > rank[%d] score %f — not sorted descending",
				i, ranked[i].ImportanceScore, i-1, ranked[i-1].ImportanceScore)
		}
	}
}

// SWE100821: TestDefaultScorer — verify weights sum to ~1.0
func TestDefaultScorer(t *testing.T) {
	scorer := DefaultScorer()
	w := scorer.Weights
	sum := w.Recency + w.Salience + w.Novelty + w.Reference + w.Semantic
	if math.Abs(sum-1.0) > 0.01 {
		t.Errorf("expected weights to sum to ~1.0, got %f", sum)
	}
}
