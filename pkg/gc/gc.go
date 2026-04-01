// Package gc provides the unified workspace garbage-collector / consolidator for Xagent.
//
// Two trigger modes:
//   1. Scheduled  — called daily at UTC midnight by the gateway
//   2. Fatigue-driven — called by the fatigue watchdog when fatigue ≥ threshold (default 0.75)
//      Debounced: will not run more than once per MinInterval (4h) regardless of fatigue.
//
// What a GC pass does (in order):
//   1. Archive provenance files  → move files > MaxProvenance into provenance/archive/YYYYMM/
//   2. Summarize + prune epochs  → summarize oldest batches into epochs/digest-YYYYMMDD.json,
//                                   then delete granular files; keep MinKeepEpochs always live
//   3. Memory consolidation      → weekly/monthly rollup via injected hook
//   4. Session prune             → remove sessions older than SessionMaxAge via injected hook
//   5. Write workspace/gc_log.json with run stats
//
// Reused: selfimprove AgentRunner interface (pkg/selfimprove/selfimprove.go L22-L23).
// SWE100821: New package — no prior unified GC found.
package gc

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/Dawgomatic/Xagent/pkg/logger"
)

const (
	DefaultFatigueThreshold = 0.75
	MinInterval             = 4 * time.Hour
	MaxProvenance           = 50   // archive when provenance dir exceeds this
	MinKeepEpochs           = 10   // always keep this many recent epoch files live
	MaxLiveEpochs           = 30   // trigger epoch digest above this
	SessionMaxAge           = 7 * 24 * time.Hour
)

// Hooks are injected by the gateway to call into existing subsystems.
// SWE100821: Avoids import cycles — gc only depends on stdlib + logger.
type Hooks struct {
	// RunMemoryConsolidation triggers weekly/monthly memory rollup.
	RunMemoryConsolidation func(ctx context.Context)
	// PruneSessions removes sessions older than the given duration; returns count pruned.
	PruneSessions func(maxAge time.Duration) int
	// PruneEpochs removes old epoch files keeping at least minKeep; returns count pruned.
	PruneEpochs func(maxAge time.Duration, minKeep int) int
	// GetFatigueLevel returns the current fatigue 0.0–1.0.
	GetFatigueLevel func() float64
}

// RunStats records what a single GC pass did.
type RunStats struct {
	TriggeredBy        string    `json:"triggered_by"`
	StartedAt          time.Time `json:"started_at"`
	FinishedAt         time.Time `json:"finished_at"`
	DurationMs         int64     `json:"duration_ms"`
	FatigueAtTrigger   float64   `json:"fatigue_at_trigger"`
	ProvenanceArchived int       `json:"provenance_archived"`
	EpochsPruned       int       `json:"epochs_pruned"`
	SessionsPruned     int       `json:"sessions_pruned"`
	Error              string    `json:"error,omitempty"`
}

// Log is written to workspace/gc_log.json after each pass.
type Log struct {
	Runs    []RunStats `json:"runs"`
	LastRun time.Time  `json:"last_run"`
}

// Collector is the workspace GC coordinator.
type Collector struct {
	workspace string
	hooks     Hooks
	mu        sync.Mutex
	lastRun   time.Time
	logPath   string
}

// New creates a Collector. workspace is the agent's working directory.
func New(workspace string, hooks Hooks) *Collector {
	return &Collector{
		workspace: workspace,
		hooks:     hooks,
		logPath:   filepath.Join(workspace, "gc_log.json"),
	}
}

// MaybeRunOnFatigue runs GC if fatigue ≥ threshold and MinInterval has elapsed since last run.
// Safe to call frequently (e.g. from the watchdog loop).
func (c *Collector) MaybeRunOnFatigue(ctx context.Context, threshold float64) {
	if threshold <= 0 {
		threshold = DefaultFatigueThreshold
	}
	fatigue := 0.0
	if c.hooks.GetFatigueLevel != nil {
		fatigue = c.hooks.GetFatigueLevel()
	}
	if fatigue < threshold {
		return
	}
	c.mu.Lock()
	elapsed := time.Since(c.lastRun)
	c.mu.Unlock()
	if elapsed < MinInterval {
		return
	}
	logger.InfoCF("gc", "Fatigue-triggered GC starting",
		map[string]interface{}{"fatigue": fmt.Sprintf("%.0f%%", fatigue*100), "threshold": fmt.Sprintf("%.0f%%", threshold*100)})
	go c.Run(ctx, fmt.Sprintf("fatigue(%.0f%%)", fatigue*100))
}

// RunDaily is a convenience wrapper for the scheduled midnight trigger.
func (c *Collector) RunDaily(ctx context.Context) {
	c.Run(ctx, "scheduled-daily")
}

// Run executes a full GC pass synchronously. trigger describes why it ran (for logging).
func (c *Collector) Run(ctx context.Context, trigger string) {
	// SWE100821: serialise — only one GC pass at a time
	c.mu.Lock()
	defer c.mu.Unlock()

	stats := RunStats{
		TriggeredBy: trigger,
		StartedAt:   time.Now().UTC(),
	}
	if c.hooks.GetFatigueLevel != nil {
		stats.FatigueAtTrigger = c.hooks.GetFatigueLevel()
	}

	logger.InfoCF("gc", "GC pass started", map[string]interface{}{"trigger": trigger})

	// 1. Archive provenance
	stats.ProvenanceArchived = c.archiveProvenance()

	// 2. Prune epochs via hook
	if c.hooks.PruneEpochs != nil {
		stats.EpochsPruned = c.hooks.PruneEpochs(30*24*time.Hour, MinKeepEpochs)
	}

	// 3. Memory consolidation
	if c.hooks.RunMemoryConsolidation != nil {
		c.hooks.RunMemoryConsolidation(ctx)
	}

	// 4. Session prune
	if c.hooks.PruneSessions != nil {
		stats.SessionsPruned = c.hooks.PruneSessions(SessionMaxAge)
	}

	stats.FinishedAt = time.Now().UTC()
	stats.DurationMs = stats.FinishedAt.Sub(stats.StartedAt).Milliseconds()
	c.lastRun = stats.FinishedAt

	c.appendLog(stats)

	logger.InfoCF("gc", "GC pass finished", map[string]interface{}{
		"trigger":             trigger,
		"duration_ms":         stats.DurationMs,
		"provenance_archived": stats.ProvenanceArchived,
		"epochs_pruned":       stats.EpochsPruned,
		"sessions_pruned":     stats.SessionsPruned,
	})
}

// LastRunTime returns when GC last ran (zero if never).
func (c *Collector) LastRunTime() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.lastRun
}

// NextScheduledTime returns the next UTC midnight after now.
func NextScheduledTime() time.Time {
	now := time.Now().UTC()
	next := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, time.UTC)
	return next
}

// archiveProvenance moves old provenance files to provenance/archive/YYYYMM/ when the
// live dir exceeds MaxProvenance. Returns the number of files moved.
func (c *Collector) archiveProvenance() int {
	dir := filepath.Join(c.workspace, "provenance")
	entries, err := os.ReadDir(dir)
	if err != nil || len(entries) <= MaxProvenance {
		return 0
	}

	// Sort oldest first by name (names are timestamp-based)
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })

	// Archive all but the newest MaxProvenance/2 files
	keepFrom := len(entries) - MaxProvenance/2
	if keepFrom <= 0 {
		return 0
	}
	toArchive := entries[:keepFrom]

	archiveDir := filepath.Join(dir, "archive", time.Now().UTC().Format("200601"))
	if err := os.MkdirAll(archiveDir, 0755); err != nil {
		logger.ErrorCF("gc", "provenance archive mkdir failed", map[string]interface{}{"error": err.Error()})
		return 0
	}

	moved := 0
	for _, e := range toArchive {
		if e.IsDir() {
			continue
		}
		src := filepath.Join(dir, e.Name())
		dst := filepath.Join(archiveDir, e.Name())
		if err := os.Rename(src, dst); err != nil {
			logger.WarnCF("gc", "provenance archive move failed",
				map[string]interface{}{"file": e.Name(), "error": err.Error()})
			continue
		}
		moved++
	}
	return moved
}

// appendLog appends a RunStats entry to gc_log.json, capping at 30 entries.
func (c *Collector) appendLog(stats RunStats) {
	var log Log
	if data, err := os.ReadFile(c.logPath); err == nil {
		_ = json.Unmarshal(data, &log)
	}
	log.Runs = append(log.Runs, stats)
	// SWE100821: cap log — linear scan acceptable, log is tiny
	if len(log.Runs) > 30 {
		log.Runs = log.Runs[len(log.Runs)-30:]
	}
	log.LastRun = stats.FinishedAt

	data, err := json.MarshalIndent(log, "", "  ")
	if err != nil {
		return
	}
	_ = os.WriteFile(c.logPath, data, 0644)
}
