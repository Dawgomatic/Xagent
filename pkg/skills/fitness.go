// SWE100821: Skill fitness scoring — tracks usage, success rates, and recency
// to produce a composite fitness score for each skill. Enables automated
// deprecation and ranking of skills by effectiveness.
package skills

import (
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

// SkillFitness holds performance metrics for a single skill.
type SkillFitness struct {
	TimesUsed      int       `json:"times_used"`
	TimesSucceeded int       `json:"times_succeeded"`
	TimesFailed    int       `json:"times_failed"`
	LastUsed       time.Time `json:"last_used"`
	Score          float64   `json:"score"`
}

// FitnessTracker manages fitness data for all known skills.
type FitnessTracker struct {
	workspace string
	fitness   map[string]*SkillFitness
	mu        sync.RWMutex
}

// NewFitnessTracker creates a tracker and loads persisted data from workspace/skills/fitness.json.
func NewFitnessTracker(workspace string) *FitnessTracker {
	ft := &FitnessTracker{
		workspace: workspace,
		fitness:   make(map[string]*SkillFitness),
	}
	_ = ft.Load() // SWE100821: best-effort load; missing file is fine on first run.
	return ft
}

// RecordUse updates counters for a skill and recomputes its fitness score.
// SWE100821: Called after each skill invocation with the outcome.
func (ft *FitnessTracker) RecordUse(skillName string, succeeded bool) {
	ft.mu.Lock()
	defer ft.mu.Unlock()

	sf, ok := ft.fitness[skillName]
	if !ok {
		sf = &SkillFitness{}
		ft.fitness[skillName] = sf
	}

	sf.TimesUsed++
	if succeeded {
		sf.TimesSucceeded++
	} else {
		sf.TimesFailed++
	}
	sf.LastUsed = time.Now()
	sf.Score = computeScore(sf)
}

// GetFitness returns fitness data for a single skill, or nil if unknown.
func (ft *FitnessTracker) GetFitness(skillName string) *SkillFitness {
	ft.mu.RLock()
	defer ft.mu.RUnlock()

	sf, ok := ft.fitness[skillName]
	if !ok {
		return nil
	}
	cp := *sf
	return &cp
}

// GetTopSkills returns the top N skill names sorted by descending score.
func (ft *FitnessTracker) GetTopSkills(n int) []string {
	ft.mu.RLock()
	defer ft.mu.RUnlock()

	type entry struct {
		name  string
		score float64
	}
	entries := make([]entry, 0, len(ft.fitness))
	for name, sf := range ft.fitness {
		entries = append(entries, entry{name, sf.Score})
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].score > entries[j].score
	})

	if n > len(entries) {
		n = len(entries)
	}
	result := make([]string, n)
	for i := 0; i < n; i++ {
		result[i] = entries[i].name
	}
	return result
}

// GetDeprecated returns skill names with score < 0.2 and times_used > 10.
// SWE100821: Candidates for removal or retraining.
func (ft *FitnessTracker) GetDeprecated() []string {
	ft.mu.RLock()
	defer ft.mu.RUnlock()

	var deprecated []string
	for name, sf := range ft.fitness {
		if sf.Score < 0.2 && sf.TimesUsed > 10 {
			deprecated = append(deprecated, name)
		}
	}
	return deprecated
}

// Save persists fitness data to workspace/skills/fitness.json.
func (ft *FitnessTracker) Save() error {
	ft.mu.RLock()
	defer ft.mu.RUnlock()

	dir := filepath.Join(ft.workspace, "skills")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(ft.fitness, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "fitness.json"), data, 0644)
}

// Load reads fitness data from workspace/skills/fitness.json.
func (ft *FitnessTracker) Load() error {
	ft.mu.Lock()
	defer ft.mu.Unlock()

	data, err := os.ReadFile(filepath.Join(ft.workspace, "skills", "fitness.json"))
	if err != nil {
		return err
	}

	loaded := make(map[string]*SkillFitness)
	if err := json.Unmarshal(data, &loaded); err != nil {
		return err
	}
	ft.fitness = loaded
	return nil
}

// computeScore calculates: success_rate * log2(times_used + 1) * recency_bonus
// where recency_bonus decays from 1.0 with a 7-day half-life.
// SWE100821: Core fitness formula.
func computeScore(sf *SkillFitness) float64 {
	if sf.TimesUsed == 0 {
		return 0.0
	}
	successRate := float64(sf.TimesSucceeded) / float64(sf.TimesUsed)
	usageWeight := math.Log2(float64(sf.TimesUsed) + 1)

	daysSince := time.Since(sf.LastUsed).Hours() / 24.0
	halfLife := 7.0
	recencyBonus := math.Pow(0.5, daysSince/halfLife)

	return successRate * usageWeight * recencyBonus
}
