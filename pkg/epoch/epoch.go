// SWE100821: Epoch lifecycle — the "day" above individual conversation sessions.
// Each agent boot is an epoch (wake). Graceful shutdown writes a journal (sleep).
// On next wake, the previous epoch's journal is loaded into context so the agent
// has continuity across restarts, like a human remembering yesterday.
//
// Storage: workspace/epochs/<session_id>.json  (one file per epoch)
// The most recent completed epoch is also symlinked as "last.json" for fast lookup.

package epoch

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/Dawgomatic/Xagent/pkg/identity"
	"github.com/Dawgomatic/Xagent/pkg/logger"
)

// Record is the journal for a single epoch (one boot-to-shutdown cycle).
type Record struct {
	SessionID    string     `json:"session_id"`
	AgentID      string     `json:"agent_id"`
	BootTime     time.Time  `json:"boot_time"`
	ShutdownTime *time.Time `json:"shutdown_time,omitempty"`
	Uptime       string     `json:"uptime,omitempty"`
	WakeNote     string     `json:"wake_note,omitempty"`
	Events       []Event    `json:"events,omitempty"`
	Reflection   string     `json:"reflection,omitempty"`
	Stats        EpochStats `json:"stats"`
}

// Event is a notable occurrence during an epoch.
type Event struct {
	Time    time.Time `json:"time"`
	Kind    string    `json:"kind"`
	Summary string    `json:"summary"`
}

// EpochStats captures quantitative data about the epoch.
type EpochStats struct {
	MessagesProcessed int     `json:"messages_processed"`
	ToolCalls         int     `json:"tool_calls"`
	SessionsActive    int     `json:"sessions_active"`
	FatigueLevel      float64 `json:"fatigue_level"` // 0.0 (fully rested) to 1.0 (exhausted)
	IsSleeping        bool    `json:"is_sleeping"`
}

// Manager handles epoch lifecycle operations.
type Manager struct {
	dir      string
	current  *Record
	identity *identity.AgentIdentity
	mu       sync.Mutex
}

// NewManager creates an epoch manager for the given workspace.
func NewManager(workspace string, id *identity.AgentIdentity) *Manager {
	dir := filepath.Join(workspace, "epochs")
	os.MkdirAll(dir, 0755)

	return &Manager{
		dir:      dir,
		identity: id,
	}
}

// Wake starts a new epoch. Loads the previous epoch's journal (if any)
// and returns it so the caller can inject it into the agent's context.
// This is the "waking up and remembering yesterday" step.
func (m *Manager) Wake() (*Record, error) {
	// SWE100821: Create current epoch record
	m.mu.Lock()
	m.current = &Record{
		SessionID: m.identity.SessionID,
		AgentID:   m.identity.AgentID,
		BootTime:  m.identity.BootTime,
		Events:    []Event{},
	}
	m.mu.Unlock()

	// SWE100821: Distinguish "no previous epochs" (expected) from real I/O errors.
	// Previously returned (nil, nil) for both, losing real errors silently.
	prev, err := m.LoadLast()
	if err != nil {
		if strings.Contains(err.Error(), "no previous epochs") {
			return nil, nil
		}
		logger.WarnCF("epoch", "Failed to load previous epoch", map[string]interface{}{"error": err.Error()})
		return nil, nil // non-fatal — agent can still start without prior epoch
	}
	return prev, nil
}

// RecordEvent logs a notable event during the current epoch.
func (m *Manager) RecordEvent(kind, summary string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.current == nil {
		return
	}
	m.current.Events = append(m.current.Events, Event{
		Time:    time.Now(),
		Kind:    kind,
		Summary: summary,
	})
}

// UpdateStats updates the current epoch's statistics.
func (m *Manager) UpdateStats(fn func(*EpochStats)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.current == nil {
		return
	}
	fn(&m.current.Stats)
}

// Sleep finalizes the current epoch — writes the journal to disk.
// This is the "going to sleep and writing in your diary" step.
func (m *Manager) Sleep(reflection string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.current == nil {
		return fmt.Errorf("epoch: no active epoch to sleep")
	}

	now := time.Now()
	m.current.ShutdownTime = &now
	m.current.Uptime = now.Sub(m.current.BootTime).Truncate(time.Second).String()
	m.current.Reflection = reflection

	// Write the epoch record
	path, err := m.saveCurrent()
	if err != nil {
		return fmt.Errorf("epoch: failed to save: %w", err)
	}
	// SWE100821: upgrade_period — audit trail for epoch journal on shutdown
	logger.InfoCF("upgrade_period", "epoch sleep journal written",
		map[string]interface{}{
			"path":                path,
			"session_id":          m.current.SessionID,
			"messages_processed":  m.current.Stats.MessagesProcessed,
			"tool_calls":          m.current.Stats.ToolCalls,
			"reflection_len":      len(reflection),
			"events_recorded":     len(m.current.Events),
		})

	return nil
}

// LoadLast returns the most recently completed epoch record.
func (m *Manager) LoadLast() (*Record, error) {
	entries, err := os.ReadDir(m.dir)
	if err != nil {
		return nil, err
	}

	// SWE100821: Filter to timestamped .json files only (YYYYMMDD-HHMMSS-*.json).
	// Non-timestamped files like "last.json" would sort lexically higher than digits,
	// causing the wrong file to be loaded as the most recent epoch.
	var jsonFiles []string
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		// Epoch files start with a date: 20YYMMDD-
		if len(e.Name()) >= 9 && e.Name()[0:2] == "20" {
			jsonFiles = append(jsonFiles, e.Name())
		}
	}

	if len(jsonFiles) == 0 {
		return nil, fmt.Errorf("epoch: no previous epochs")
	}

	sort.Sort(sort.Reverse(sort.StringSlice(jsonFiles)))

	// Load the most recent one
	data, err := os.ReadFile(filepath.Join(m.dir, jsonFiles[0]))
	if err != nil {
		return nil, err
	}

	var rec Record
	if err := json.Unmarshal(data, &rec); err != nil {
		return nil, err
	}

	return &rec, nil
}

// ForSystemPrompt formats the previous epoch for injection into the LLM context.
// Returns empty string if no previous epoch exists.
func ForSystemPrompt(prev *Record) string {
	if prev == nil {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("## Previous Session (Last Epoch)\n")
	sb.WriteString(fmt.Sprintf("Session ID: %s\n", prev.SessionID))
	sb.WriteString(fmt.Sprintf("Boot: %s\n", prev.BootTime.Format("2006-01-02 15:04:05")))

	if prev.ShutdownTime != nil {
		sb.WriteString(fmt.Sprintf("Shutdown: %s\n", prev.ShutdownTime.Format("2006-01-02 15:04:05")))
	}
	if prev.Uptime != "" {
		sb.WriteString(fmt.Sprintf("Duration: %s\n", prev.Uptime))
	}

	sb.WriteString(fmt.Sprintf("Messages: %d | Tool calls: %d | Active sessions: %d\n",
		prev.Stats.MessagesProcessed,
		prev.Stats.ToolCalls,
		prev.Stats.SessionsActive))
	sb.WriteString(fmt.Sprintf("Woke up with Fatigue Level: %.2f\n", prev.Stats.FatigueLevel))

	if len(prev.Events) > 0 {
		sb.WriteString("\nNotable events:\n")
		limit := 10
		if len(prev.Events) < limit {
			limit = len(prev.Events)
		}
		for _, ev := range prev.Events[:limit] {
			sb.WriteString(fmt.Sprintf("- [%s] %s: %s\n",
				ev.Time.Format("15:04"), ev.Kind, ev.Summary))
		}
		if len(prev.Events) > 10 {
			sb.WriteString(fmt.Sprintf("... and %d more events\n", len(prev.Events)-10))
		}
	}

	if prev.Reflection != "" {
		sb.WriteString(fmt.Sprintf("\nReflection: %s\n", prev.Reflection))
	}

	return sb.String()
}

// GetCurrent returns the current epoch record (may be nil if Wake not called).
func (m *Manager) GetCurrent() *Record {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.current
}

// saveCurrent writes the current epoch record to disk.
// Filename: <boot_timestamp>-<session_id_prefix>.json for chronological sorting.
// Returns the written file path, or empty string if nothing to save.
func (m *Manager) saveCurrent() (string, error) {
	if m.current == nil {
		return "", nil
	}

	// Filename uses boot timestamp for natural sort order
	ts := m.current.BootTime.Format("20060102-150405")
	sessionPrefix := m.current.SessionID
	if len(sessionPrefix) > 16 {
		sessionPrefix = sessionPrefix[:16]
	}
	filename := fmt.Sprintf("%s-%s.json", ts, sessionPrefix)

	data, err := json.MarshalIndent(m.current, "", "  ")
	if err != nil {
		return "", err
	}

	filePath := filepath.Join(m.dir, filename)
	tmpFile := filePath + ".tmp"
	if err := os.WriteFile(tmpFile, data, 0644); err != nil {
		return "", err
	}
	if err := os.Rename(tmpFile, filePath); err != nil {
		os.Remove(tmpFile)
		return "", err
	}

	return filePath, nil
}

// PruneOld removes epoch records older than maxAge, keeping at least minKeep.
func (m *Manager) PruneOld(maxAge time.Duration, minKeep int) int {
	entries, err := os.ReadDir(m.dir)
	if err != nil {
		return 0
	}

	var jsonFiles []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".json") {
			jsonFiles = append(jsonFiles, e.Name())
		}
	}

	sort.Strings(jsonFiles)

	if len(jsonFiles) <= minKeep {
		return 0
	}

	cutoff := time.Now().Add(-maxAge)
	pruned := 0
	candidates := jsonFiles[:len(jsonFiles)-minKeep]

	for _, f := range candidates {
		info, err := os.Stat(filepath.Join(m.dir, f))
		if err != nil {
			continue
		}
		if info.ModTime().Before(cutoff) {
			full := filepath.Join(m.dir, f)
			if err := os.Remove(full); err != nil {
				logger.WarnCF("upgrade_period", "failed to prune old epoch file",
					map[string]interface{}{"file": f, "error": err.Error()})
				continue
			}
			pruned++
			// SWE100821: upgrade_period — log each pruned epoch file
			logger.InfoCF("upgrade_period", "pruned old epoch file",
				map[string]interface{}{"file": f, "mod_time": info.ModTime().Format(time.RFC3339)})
		}
	}

	return pruned
}

// Rollover finalizes the current epoch and starts a new one transparently.
func (m *Manager) Rollover(reflection string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.current == nil {
		return fmt.Errorf("epoch: no active epoch to rollover")
	}

	now := time.Now()
	m.current.ShutdownTime = &now
	m.current.Uptime = now.Sub(m.current.BootTime).Truncate(time.Second).String()
	m.current.Reflection = reflection

	// Write the old epoch record to disk
	path, err := m.saveCurrent()
	if err != nil {
		return fmt.Errorf("epoch: failed to save during rollover: %w", err)
	}
	oldSession := m.current.SessionID
	oldBoot := m.current.BootTime
	oldUptime := m.current.Uptime
	// SWE100821: upgrade_period — rollover completes prior epoch on disk
	logger.InfoCF("upgrade_period", "epoch rollover persisted completed epoch",
		map[string]interface{}{
			"path":         path,
			"session_id":   oldSession,
			"boot_time":    oldBoot.Format(time.RFC3339),
			"uptime":       oldUptime,
			"reflection":   reflection,
			"msg_count":    m.current.Stats.MessagesProcessed,
			"tool_calls":   m.current.Stats.ToolCalls,
		})

	// Carry over state to the new epoch journal
	newRecord := &Record{
		SessionID: m.identity.SessionID,
		AgentID:   m.identity.AgentID,
		BootTime:  now,
		Events:    []Event{},
		Stats: EpochStats{
			FatigueLevel: m.current.Stats.FatigueLevel,
			IsSleeping:   m.current.Stats.IsSleeping,
		},
	}

	m.current = newRecord
	// SWE100821: upgrade_period — new epoch is in-memory until next sleep/rollover
	logger.InfoCF("upgrade_period", "epoch rollover started new epoch",
		map[string]interface{}{
			"session_id":      newRecord.SessionID,
			"boot_time":       newRecord.BootTime.Format(time.RFC3339),
			"carried_fatigue": newRecord.Stats.FatigueLevel,
		})
	return nil
}

// StartRolloverMonitor begins a background check to automatically rollover epochs
// older than the specified interval (e.g., 24 hours).
func (m *Manager) StartRolloverMonitor(ctx context.Context, interval time.Duration) {
	go func() {
		ticker := time.NewTicker(15 * time.Minute)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				m.mu.Lock()
				if m.current == nil {
					m.mu.Unlock()
					continue
				}
				uptime := time.Since(m.current.BootTime)
				m.mu.Unlock()

				if uptime >= interval {
					logger.InfoCF("epoch", fmt.Sprintf("Epoch uptime %s exceeded interval %s. Rolling over.", uptime.Truncate(time.Minute), interval), nil)
					if err := m.Rollover(fmt.Sprintf("Automatic %s rollover.", interval)); err != nil {
						logger.ErrorCF("epoch", "Failed to rollover epoch", map[string]interface{}{"error": err.Error()})
					}
				}
			}
		}
	}()
}
