// SWE100821: Provenance tracking — records the full lineage of every agent response.
// For each turn, tracks which tools were called, which skills contributed,
// which memories were recalled, and which provider/model generated the response.
// Enables debugging, trust, and transparency for a local-first agent.

package agent

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// ProvenanceRecord captures the full lineage of a single agent turn.
type ProvenanceRecord struct {
	TurnID      string             `json:"turn_id"`
	Timestamp   time.Time          `json:"timestamp"`
	SessionKey  string             `json:"session_key"`
	Channel     string             `json:"channel"`
	UserMessage string             `json:"user_message"` // first 200 chars
	Provider    string             `json:"provider"`
	Model       string             `json:"model"`
	ToolsCalled []ToolProvenance   `json:"tools_called"`
	SkillsUsed  []string           `json:"skills_used"`
	MemoryHits  []string           `json:"memory_hits,omitempty"`
	PlanSteps   int                `json:"plan_steps,omitempty"`
	Iterations  int                `json:"iterations"`
	LatencyMs   int64              `json:"latency_ms"`
	TokensUsed  int                `json:"tokens_used,omitempty"`
}

// ToolProvenance records a single tool invocation within a turn.
type ToolProvenance struct {
	Name      string `json:"name"`
	Success   bool   `json:"success"`
	LatencyMs int64  `json:"latency_ms"`
}

// ProvenanceTracker manages provenance records for the agent.
type ProvenanceTracker struct {
	dir     string
	current *ProvenanceRecord
	mu      sync.Mutex
}

// NewProvenanceTracker creates a tracker storing records in the given workspace.
func NewProvenanceTracker(workspace string) *ProvenanceTracker {
	dir := filepath.Join(workspace, "provenance")
	os.MkdirAll(dir, 0755)
	return &ProvenanceTracker{dir: dir}
}

// StartTurn begins tracking a new turn.
func (pt *ProvenanceTracker) StartTurn(turnID, sessionKey, channel, userMessage, model string) {
	pt.mu.Lock()
	defer pt.mu.Unlock()

	msgPreview := userMessage
	if len(msgPreview) > 200 {
		msgPreview = msgPreview[:200]
	}

	pt.current = &ProvenanceRecord{
		TurnID:      turnID,
		Timestamp:   time.Now(),
		SessionKey:  sessionKey,
		Channel:     channel,
		UserMessage: msgPreview,
		Model:       model,
		ToolsCalled: make([]ToolProvenance, 0),
		SkillsUsed:  make([]string, 0),
	}
}

// RecordToolCall logs a tool invocation.
func (pt *ProvenanceTracker) RecordToolCall(name string, success bool, latencyMs int64) {
	pt.mu.Lock()
	defer pt.mu.Unlock()
	if pt.current == nil {
		return
	}
	pt.current.ToolsCalled = append(pt.current.ToolsCalled, ToolProvenance{
		Name:      name,
		Success:   success,
		LatencyMs: latencyMs,
	})
}

// RecordSkills logs which skills contributed to this turn.
func (pt *ProvenanceTracker) RecordSkills(skills []string) {
	pt.mu.Lock()
	defer pt.mu.Unlock()
	if pt.current == nil {
		return
	}
	pt.current.SkillsUsed = skills
}

// RecordMemoryHits logs which memories were recalled.
func (pt *ProvenanceTracker) RecordMemoryHits(hits []string) {
	pt.mu.Lock()
	defer pt.mu.Unlock()
	if pt.current == nil {
		return
	}
	pt.current.MemoryHits = hits
}

// SetIterations sets the final iteration count.
func (pt *ProvenanceTracker) SetIterations(n int) {
	pt.mu.Lock()
	defer pt.mu.Unlock()
	if pt.current == nil {
		return
	}
	pt.current.Iterations = n
}

// SetPlanSteps sets the plan step count.
func (pt *ProvenanceTracker) SetPlanSteps(n int) {
	pt.mu.Lock()
	defer pt.mu.Unlock()
	if pt.current == nil {
		return
	}
	pt.current.PlanSteps = n
}

// FinishTurn finalizes and persists the provenance record.
func (pt *ProvenanceTracker) FinishTurn() error {
	pt.mu.Lock()
	defer pt.mu.Unlock()
	if pt.current == nil {
		return nil
	}

	pt.current.LatencyMs = time.Since(pt.current.Timestamp).Milliseconds()

	// Write to daily provenance file (append JSONL)
	dateStr := time.Now().Format("20060102")
	filePath := filepath.Join(pt.dir, dateStr+".jsonl")

	data, err := json.Marshal(pt.current)
	if err != nil {
		return fmt.Errorf("provenance marshal failed: %w", err)
	}

	f, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return fmt.Errorf("provenance file open failed: %w", err)
	}
	defer f.Close()

	if _, err := f.Write(append(data, '\n')); err != nil {
		return fmt.Errorf("provenance write failed: %w", err)
	}

	pt.current = nil
	return nil
}

// GetCurrent returns the current in-progress provenance record (for system prompt).
func (pt *ProvenanceTracker) GetCurrent() *ProvenanceRecord {
	pt.mu.Lock()
	defer pt.mu.Unlock()
	return pt.current
}

// PruneOld removes provenance files older than maxAge.
func (pt *ProvenanceTracker) PruneOld(maxAge time.Duration) int {
	entries, err := os.ReadDir(pt.dir)
	if err != nil {
		return 0
	}
	cutoff := time.Now().Add(-maxAge)
	pruned := 0
	for _, e := range entries {
		info, err := e.Info()
		if err != nil {
			continue
		}
		if info.ModTime().Before(cutoff) {
			os.Remove(filepath.Join(pt.dir, e.Name()))
			pruned++
		}
	}
	return pruned
}
