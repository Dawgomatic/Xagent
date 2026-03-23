// SWE100821: Temporal memory index — time-aware memory retrieval.
// Stores timestamped topic entries and resolves natural-language temporal
// references ("yesterday", "last week") into filtered result sets.
// Persists to workspace/state/temporal_index.json, capped at 10k entries.

package memory

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

const maxTemporalEntries = 10000

// TemporalEntry is a single timestamped memory item with topic tags.
type TemporalEntry struct {
	Timestamp  time.Time `json:"timestamp"`
	TopicTags  []string  `json:"topic_tags"`
	SessionKey string    `json:"session_key"`
	Summary    string    `json:"summary"`
	Source     string    `json:"source"`
}

// TemporalIndex provides time-range and topic-filtered memory queries.
type TemporalIndex struct {
	workspace string
	entries   []TemporalEntry
	mu        sync.RWMutex
}

// NewTemporalIndex creates a TemporalIndex, loading existing state from disk.
func NewTemporalIndex(workspace string) *TemporalIndex {
	ti := &TemporalIndex{
		workspace: workspace,
		entries:   make([]TemporalEntry, 0),
	}
	// SWE100821: Best-effort load; fresh index if file missing
	_ = ti.Load()
	return ti
}

// Add appends an entry, prunes if over cap, and persists to disk.
func (ti *TemporalIndex) Add(entry TemporalEntry) {
	ti.mu.Lock()
	defer ti.mu.Unlock()

	ti.entries = append(ti.entries, entry)

	// SWE100821: Prune oldest entries when cap exceeded
	if len(ti.entries) > maxTemporalEntries {
		sort.Slice(ti.entries, func(i, j int) bool {
			return ti.entries[i].Timestamp.Before(ti.entries[j].Timestamp)
		})
		ti.entries = ti.entries[len(ti.entries)-maxTemporalEntries:]
	}

	_ = ti.saveLocked()
}

// Query returns entries within [since, until] optionally filtered by topic substring.
func (ti *TemporalIndex) Query(since, until time.Time, topicFilter string) []TemporalEntry {
	ti.mu.RLock()
	defer ti.mu.RUnlock()

	lowerFilter := strings.ToLower(topicFilter)
	var results []TemporalEntry
	for _, e := range ti.entries {
		if e.Timestamp.Before(since) || e.Timestamp.After(until) {
			continue
		}
		if lowerFilter != "" && !topicMatches(e.TopicTags, lowerFilter) {
			continue
		}
		results = append(results, e)
	}
	return results
}

// SWE100821: hasTemporalRef checks if a query contains a time-related phrase.
var temporalKeywords = []string{
	"yesterday", "today", "last week", "this week", "last month", "this month",
	"earlier", "before", "remember when", "previously", "ago", "recent",
}

func hasTemporalRef(lower string) bool {
	for _, kw := range temporalKeywords {
		if strings.Contains(lower, kw) {
			return true
		}
	}
	return false
}

// ForSystemPrompt parses temporal references in query and returns formatted results.
// Recognises: "yesterday", "today", "last week", "this week", "last month", "this month".
func (ti *TemporalIndex) ForSystemPrompt(query string) string {
	lower := strings.ToLower(query)
	now := time.Now()
	var since, until time.Time

	switch {
	case strings.Contains(lower, "yesterday"):
		y := now.AddDate(0, 0, -1)
		since = startOfDay(y)
		until = endOfDay(y)
	case strings.Contains(lower, "today"):
		since = startOfDay(now)
		until = endOfDay(now)
	case strings.Contains(lower, "last week"):
		since = startOfDay(now.AddDate(0, 0, -7))
		until = endOfDay(now.AddDate(0, 0, -1))
	case strings.Contains(lower, "this week"):
		weekday := int(now.Weekday())
		if weekday == 0 {
			weekday = 7
		}
		since = startOfDay(now.AddDate(0, 0, -(weekday - 1)))
		until = endOfDay(now)
	case strings.Contains(lower, "last month"):
		firstThisMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
		since = firstThisMonth.AddDate(0, -1, 0)
		until = firstThisMonth.Add(-time.Second)
	case strings.Contains(lower, "this month"):
		since = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
		until = endOfDay(now)
	default:
		since = startOfDay(now.AddDate(0, 0, -7))
		until = endOfDay(now)
	}

	results := ti.Query(since, until, "")
	if len(results) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("## Temporal Memory (%s → %s)\n\n",
		since.Format("2006-01-02"), until.Format("2006-01-02")))
	for _, e := range results {
		sb.WriteString(fmt.Sprintf("- [%s] %s (tags: %s)\n",
			e.Timestamp.Format("2006-01-02 15:04"),
			e.Summary,
			strings.Join(e.TopicTags, ", ")))
	}
	return sb.String()
}

// SWE100821: ForContext returns temporal memory only when relevant.
// If the query has a temporal reference ("yesterday", "last week", etc.), returns
// full temporal recall. Otherwise returns a compact recent-activity summary (last 24h,
// capped at 5 entries) so the agent has ambient continuity without flooding context.
func (ti *TemporalIndex) ForContext(query string) string {
	lower := strings.ToLower(query)

	if hasTemporalRef(lower) {
		return ti.ForSystemPrompt(query)
	}

	now := time.Now()
	results := ti.Query(startOfDay(now.AddDate(0, 0, -1)), endOfDay(now), "")
	if len(results) == 0 {
		return ""
	}

	cap := 5
	if len(results) > cap {
		results = results[len(results)-cap:]
	}

	var sb strings.Builder
	sb.WriteString("## Recent Activity\n\n")
	for _, e := range results {
		sb.WriteString(fmt.Sprintf("- [%s] %s\n",
			e.Timestamp.Format("15:04"), e.Summary))
	}
	return sb.String()
}

// Save persists the index to disk as JSON.
func (ti *TemporalIndex) Save() error {
	ti.mu.RLock()
	defer ti.mu.RUnlock()
	return ti.saveLocked()
}

// Load reads the index from disk.
func (ti *TemporalIndex) Load() error {
	ti.mu.Lock()
	defer ti.mu.Unlock()

	p := ti.filePath()
	data, err := os.ReadFile(p)
	if err != nil {
		return err
	}
	var entries []TemporalEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		return err
	}
	ti.entries = entries
	return nil
}

// --- internal helpers ---

func (ti *TemporalIndex) filePath() string {
	return filepath.Join(ti.workspace, "state", "temporal_index.json")
}

// saveLocked writes entries to disk; caller must hold at least RLock.
func (ti *TemporalIndex) saveLocked() error {
	dir := filepath.Dir(ti.filePath())
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(ti.entries, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(ti.filePath(), data, 0644)
}

func topicMatches(tags []string, filter string) bool {
	for _, t := range tags {
		if strings.Contains(strings.ToLower(t), filter) {
			return true
		}
	}
	return false
}

func startOfDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}

func endOfDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 23, 59, 59, 999999999, t.Location())
}
