// SWE100821: Memory importance scoring — multi-factor ranking for memory retrieval.
// Scores memories by emotional salience, novelty, recency decay, and reference count.
// Mimics human memory consolidation: important memories persist, trivial ones fade.

package memory

import (
	"math"
	"strings"
	"time"
)

// ScoredMemory wraps a memory point with its computed importance score.
type ScoredMemory struct {
	Point           MemoryPoint
	ImportanceScore float64
	RecencyScore    float64
	SalienceScore   float64
	NoveltyScore    float64
	ReferenceScore  float64
}

// MemoryScorer computes importance scores for memory points.
type MemoryScorer struct {
	RecencyHalfLife  time.Duration // how quickly recency decays (default: 7 days)
	SalienceKeywords []string     // words that indicate emotional salience
	Weights          ScoreWeights
}

// ScoreWeights controls the relative importance of each scoring factor.
type ScoreWeights struct {
	Recency   float64
	Salience  float64
	Novelty   float64
	Reference float64
	Semantic  float64 // vector similarity score from Qdrant
}

// DefaultScorer returns a scorer with sensible defaults.
func DefaultScorer() *MemoryScorer {
	return &MemoryScorer{
		RecencyHalfLife: 7 * 24 * time.Hour,
		SalienceKeywords: []string{
			"important", "critical", "urgent", "error", "fail", "success",
			"remember", "never", "always", "warning", "love", "hate",
			"frustrated", "happy", "angry", "thank", "please",
			"password", "secret", "key", "deadline",
		},
		Weights: ScoreWeights{
			Recency:   0.25,
			Salience:  0.20,
			Novelty:   0.15,
			Reference: 0.10,
			Semantic:  0.30,
		},
	}
}

// Score computes the composite importance score for a memory point.
// referenceCount is how many times this memory has been recalled before.
// uniqueTermRatio is the fraction of terms in this memory not seen in other recent memories.
func (ms *MemoryScorer) Score(point MemoryPoint, referenceCount int, uniqueTermRatio float64) ScoredMemory {
	recency := ms.computeRecency(point.Created)
	salience := ms.computeSalience(point.Text)
	novelty := uniqueTermRatio
	reference := ms.computeReference(referenceCount)

	composite := ms.Weights.Recency*recency +
		ms.Weights.Salience*salience +
		ms.Weights.Novelty*novelty +
		ms.Weights.Reference*reference +
		ms.Weights.Semantic*point.Score

	return ScoredMemory{
		Point:           point,
		ImportanceScore: composite,
		RecencyScore:    recency,
		SalienceScore:   salience,
		NoveltyScore:    novelty,
		ReferenceScore:  reference,
	}
}

// SWE100821: Exponential recency decay — half-life model
func (ms *MemoryScorer) computeRecency(created time.Time) float64 {
	age := time.Since(created)
	return math.Exp(-0.693 * float64(age) / float64(ms.RecencyHalfLife))
}

// SWE100821: Keyword-based salience detection
func (ms *MemoryScorer) computeSalience(text string) float64 {
	lower := strings.ToLower(text)
	hits := 0
	for _, kw := range ms.SalienceKeywords {
		if strings.Contains(lower, kw) {
			hits++
		}
	}
	// Saturate at 1.0, with diminishing returns
	return 1.0 - math.Exp(-0.5*float64(hits))
}

// SWE100821: Log-scaled reference count — frequently recalled memories are more important
func (ms *MemoryScorer) computeReference(count int) float64 {
	if count <= 0 {
		return 0
	}
	return math.Min(1.0, math.Log2(float64(count+1))/5.0)
}

// RankMemories scores and sorts memories by importance (descending).
func (ms *MemoryScorer) RankMemories(points []MemoryPoint, referenceCounts map[uint64]int) []ScoredMemory {
	scored := make([]ScoredMemory, 0, len(points))
	for _, p := range points {
		refCount := referenceCounts[p.ID]
		scored = append(scored, ms.Score(p, refCount, 0.5)) // default novelty
	}

	// Sort by importance descending
	for i := 0; i < len(scored); i++ {
		for j := i + 1; j < len(scored); j++ {
			if scored[j].ImportanceScore > scored[i].ImportanceScore {
				scored[i], scored[j] = scored[j], scored[i]
			}
		}
	}

	return scored
}
