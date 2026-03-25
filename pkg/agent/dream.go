// SWE100821: Dream mode — offline reflection during idle periods.
// When no messages arrive for a configurable idle period, the agent enters
// "dream mode": it reviews recent epoch journals and daily notes, identifies
// patterns and contradictions, generates insights, and writes them to memory.
// Optionally sends a proactive message: "While thinking, I realized..."
//
// This is genuinely unique — no other local agent framework does autonomous offline reflection.

package agent

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/Dawgomatic/Xagent/pkg/logger"
	"github.com/Dawgomatic/Xagent/pkg/providers"
)

// DreamMode manages autonomous offline reflection.
type DreamMode struct {
	provider    providers.LLMProvider
	model       string
	memory      *MemoryStore
	workspace   string
	idleTimeout time.Duration // how long to wait before dreaming
	interval    time.Duration // minimum time between dreams
	lastDream   time.Time
	lastMessage time.Time
	running     bool
	onInsight   func(insight string)     // callback to send proactive message
	onDream     func(result DreamResult) // callback for vault integration
	mu          sync.Mutex
	cancel      context.CancelFunc
}

// DreamResult contains the output of a dream session.
type DreamResult struct {
	Insights   []string  `json:"insights"`
	Patterns   []string  `json:"patterns"`
	Questions  []string  `json:"questions"` // open questions the agent identified
	DreamedAt  time.Time `json:"dreamed_at"`
	InputNotes int       `json:"input_notes"` // how many notes were reviewed
}

// NewDreamMode creates a dream mode manager.
func NewDreamMode(provider providers.LLMProvider, model, workspace string) *DreamMode {
	return &DreamMode{
		provider:    provider,
		model:       model,
		memory:      NewMemoryStore(workspace),
		workspace:   workspace,
		idleTimeout: 2 * time.Hour,
		interval:    12 * time.Hour,
	}
}

// SetIdleTimeout sets how long to wait without messages before dreaming.
func (dm *DreamMode) SetIdleTimeout(d time.Duration) {
	dm.mu.Lock()
	defer dm.mu.Unlock()
	dm.idleTimeout = d
}

// SetInterval sets the minimum time between dream sessions.
func (dm *DreamMode) SetInterval(d time.Duration) {
	dm.mu.Lock()
	defer dm.mu.Unlock()
	dm.interval = d
}

// SetInsightCallback registers a function to be called when a dream produces an insight.
func (dm *DreamMode) SetInsightCallback(fn func(insight string)) {
	dm.mu.Lock()
	defer dm.mu.Unlock()
	dm.onInsight = fn
}

// SetDreamCallback registers a function to be called with the full dream result.
// Used by the vault writer to create dream notes.
func (dm *DreamMode) SetDreamCallback(fn func(result DreamResult)) {
	dm.mu.Lock()
	defer dm.mu.Unlock()
	dm.onDream = fn
}

// RecordActivity updates the last-message timestamp to reset the idle timer.
func (dm *DreamMode) RecordActivity() {
	dm.mu.Lock()
	defer dm.mu.Unlock()
	dm.lastMessage = time.Now()
}

// Start begins the dream mode background loop.
func (dm *DreamMode) Start(ctx context.Context) {
	dm.mu.Lock()
	if dm.running {
		dm.mu.Unlock()
		return
	}
	dm.running = true
	dm.lastMessage = time.Now()
	ctx, dm.cancel = context.WithCancel(ctx)
	dm.mu.Unlock()

	go dm.loop(ctx)
}

// Stop terminates the dream mode loop.
func (dm *DreamMode) Stop() {
	dm.mu.Lock()
	defer dm.mu.Unlock()
	if dm.cancel != nil {
		dm.cancel()
	}
	dm.running = false
}

func (dm *DreamMode) loop(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Minute) // check every 5 min
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			dm.mu.Lock()
			idle := time.Since(dm.lastMessage)
			sinceLast := time.Since(dm.lastDream)
			shouldDream := idle >= dm.idleTimeout && sinceLast >= dm.interval
			dm.mu.Unlock()

			if shouldDream {
				dm.dream(ctx)
			}
		}
	}
}

func (dm *DreamMode) dream(ctx context.Context) {
	logger.InfoCF("dream", "Entering dream mode (offline reflection)", nil)

	// SWE100821: Read WORLD_MODEL.md if it exists for causal-belief integration
	worldModelPath := filepath.Join(dm.workspace, "WORLD_MODEL.md")
	worldModelContent := ""
	if data, err := os.ReadFile(worldModelPath); err == nil {
		worldModelContent = string(data)
	}

	// Gather material to reflect on
	recentNotes := dm.memory.GetRecentDailyNotes(7)
	longTerm := dm.memory.ReadLongTerm()

	// SWE100821: Also pull goals and recent vault experiences for richer dream material.
	// Previously only used daily notes + MEMORY.md + WORLD_MODEL.md, which was often thin.
	goalsContent := ""
	if data, err := os.ReadFile(filepath.Join(dm.workspace, "GOALS.md")); err == nil && len(data) > 0 {
		goalsContent = string(data)
	}
	vaultExperiences := ""
	expDir := filepath.Join(dm.workspace, "..", "vault", "Experiences")
	if entries, err := os.ReadDir(expDir); err == nil && len(entries) > 0 {
		var expBuf strings.Builder
		count := 0
		for i := len(entries) - 1; i >= 0 && count < 5; i-- {
			if entries[i].IsDir() {
				continue
			}
			if data, err := os.ReadFile(filepath.Join(expDir, entries[i].Name())); err == nil {
				expBuf.WriteString(string(data))
				expBuf.WriteString("\n---\n")
				count++
			}
		}
		vaultExperiences = expBuf.String()
	}

	if recentNotes == "" && longTerm == "" && worldModelContent == "" && goalsContent == "" {
		logger.InfoCF("dream", "Nothing to dream about (no notes or memory)", nil)
		dm.mu.Lock()
		dm.lastDream = time.Now()
		dm.mu.Unlock()
		return
	}

	var matBuilder strings.Builder
	noteCount := 0
	if recentNotes != "" {
		matBuilder.WriteString("## Recent Daily Notes (last 7 days)\n\n")
		matBuilder.WriteString(recentNotes)
		noteCount = strings.Count(recentNotes, "# 20")
	}
	if longTerm != "" {
		matBuilder.WriteString("\n\n## Long-term Memory\n\n")
		matBuilder.WriteString(longTerm)
	}
	if worldModelContent != "" {
		matBuilder.WriteString("\n\n## Current World Model\n\n")
		matBuilder.WriteString(worldModelContent)
	}
	// SWE100821: Goals give the dream mode awareness of the agent's self-directed projects
	if goalsContent != "" {
		matBuilder.WriteString("\n\n## Current Goals & Projects\n\n")
		matBuilder.WriteString(goalsContent)
	}
	// SWE100821: Recent experiences provide concrete interaction context for pattern finding
	if vaultExperiences != "" {
		matBuilder.WriteString("\n\n## Recent Experiences (last 5)\n\n")
		matBuilder.WriteString(vaultExperiences)
	}
	material := matBuilder.String()

	// SWE100821: Ask the LLM to reflect, including world model update instructions
	dreamPrompt := fmt.Sprintf(`You are in dream mode — a quiet time for autonomous reflection.

Review the following notes and memories. Think deeply about:
1. Patterns or recurring themes across days
2. Contradictions or inconsistencies in information
3. Questions that remain unanswered
4. Connections between different topics
5. Insights that synthesize multiple observations

Also update the WORLD_MODEL.md with new causal beliefs, updated models, and deprecated beliefs. Format as:
WORLD_MODEL_UPDATES:
ADD: <new causal belief or model>
UPDATE: <existing belief> -> <revised belief>
DEPRECATE: <belief that is no longer supported by evidence>

Respond in this format:

PATTERNS:
- <pattern 1>
- <pattern 2>

INSIGHTS:
- <insight 1>
- <insight 2>

QUESTIONS:
- <open question 1>
- <open question 2>

WORLD_MODEL_UPDATES:
ADD: ...
UPDATE: ...
DEPRECATE: ...

Be concise. Focus on genuinely novel observations.

MATERIAL:
%s`, truncate(material, 3000))

	// SWE100821: Bounded timeout for dream LLM call — prevents hung provider from blocking forever
	dreamCtx, dreamCancel := context.WithTimeout(ctx, 5*time.Minute)
	defer dreamCancel()

	resp, err := dm.provider.Chat(dreamCtx, []providers.Message{
		{Role: "user", Content: dreamPrompt},
	}, nil, dm.model, map[string]interface{}{
		"max_tokens":  1536,
		"temperature": 0.8,
	})

	if err != nil {
		logger.WarnCF("dream", "Dream reflection failed", map[string]interface{}{"error": err.Error()})
		// SWE100821: Update lastDream on failure to prevent retry spam every 5 min
		dm.mu.Lock()
		dm.lastDream = time.Now()
		dm.mu.Unlock()
		return
	}

	result := parseDreamResult(resp.Content, noteCount)

	// SWE100821: Parse and persist WORLD_MODEL updates from dream output
	wmUpdates := parseWorldModelUpdates(resp.Content)
	if wmUpdates != "" {
		if err := updateWorldModel(dm.workspace, wmUpdates); err != nil {
			logger.WarnCF("dream", "Failed to update WORLD_MODEL.md", map[string]interface{}{"error": err.Error()})
		} else {
			logger.InfoCF("dream", "Updated WORLD_MODEL.md with dream insights", nil)
		}
	}

	// Write insights to daily notes
	if len(result.Insights) > 0 || len(result.Patterns) > 0 {
		var dreamNote strings.Builder
		dreamNote.WriteString("##  Dream Mode Reflection\n\n")

		if len(result.Patterns) > 0 {
			dreamNote.WriteString("### Patterns\n")
			for _, p := range result.Patterns {
				dreamNote.WriteString("- " + p + "\n")
			}
			dreamNote.WriteString("\n")
		}

		if len(result.Insights) > 0 {
			dreamNote.WriteString("### Insights\n")
			for _, i := range result.Insights {
				dreamNote.WriteString("- " + i + "\n")
			}
			dreamNote.WriteString("\n")
		}

		if len(result.Questions) > 0 {
			dreamNote.WriteString("### Open Questions\n")
			for _, q := range result.Questions {
				dreamNote.WriteString("- " + q + "\n")
			}
		}

		note := dreamNote.String()
		if err := dm.memory.AppendToday(note); err != nil {
			logger.WarnCF("dream", "Failed to append dream daily note", map[string]interface{}{"error": err.Error()})
		} else {
			// SWE100821: upgrade_period — dream idle reflection wrote to disk
			logger.InfoCF("upgrade_period", "dream daily note append",
				map[string]interface{}{
					"path":          dm.memory.TodayNotePath(),
					"bytes_appended": len(note),
				})
		}
	}

	// Optionally notify user of the most interesting insight
	dm.mu.Lock()
	dm.lastDream = time.Now()
	cb := dm.onInsight
	dreamCb := dm.onDream
	dm.mu.Unlock()

	// Notify vault of dream results
	if dreamCb != nil {
		dreamCb(result)
	}

	if cb != nil && len(result.Insights) > 0 {
		cb(fmt.Sprintf(" While reflecting during idle time, I noticed: %s", result.Insights[0]))
	}

	logger.InfoCF("dream", "Dream mode complete",
		map[string]interface{}{
			"patterns":  len(result.Patterns),
			"insights":  len(result.Insights),
			"questions": len(result.Questions),
		})
}

// SWE100821: parseWorldModelUpdates extracts ADD/UPDATE/DEPRECATE lines from dream output.
// Tolerates prose between directives — LLMs often add explanatory text.
func parseWorldModelUpdates(content string) string {
	var updates strings.Builder
	inSection := false
	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "WORLD_MODEL_UPDATES:") {
			inSection = true
			continue
		}
		if !inSection {
			continue
		}
		// Accept directive lines regardless of surrounding prose
		if strings.HasPrefix(trimmed, "ADD:") || strings.HasPrefix(trimmed, "UPDATE:") || strings.HasPrefix(trimmed, "DEPRECATE:") {
			updates.WriteString(trimmed + "\n")
		}
		// Stop on next section header (e.g., PATTERNS:, INSIGHTS:)
		if strings.HasSuffix(trimmed, ":") && !strings.HasPrefix(trimmed, "ADD:") &&
			!strings.HasPrefix(trimmed, "UPDATE:") && !strings.HasPrefix(trimmed, "DEPRECATE:") &&
			!strings.HasPrefix(trimmed, "WORLD_MODEL_UPDATES:") && len(trimmed) > 2 {
			break
		}
	}
	return updates.String()
}

// SWE100821: updateWorldModel reads WORLD_MODEL.md, appends new content under today's date, and writes it back.
func updateWorldModel(workspace, content string) error {
	wmPath := filepath.Join(workspace, "WORLD_MODEL.md")

	existing := ""
	if data, err := os.ReadFile(wmPath); err == nil {
		existing = string(data)
	}

	header := fmt.Sprintf("\n\n## %s — Dream Update\n\n", time.Now().Format("2006-01-02"))
	updated := existing + header + content

	if err := os.MkdirAll(filepath.Dir(wmPath), 0755); err != nil {
		return err
	}
	if err := os.WriteFile(wmPath, []byte(updated), 0644); err != nil {
		return err
	}
	// SWE100821: upgrade_period — world model file touched during dream
	logger.InfoCF("upgrade_period", "WORLD_MODEL.md updated from dream",
		map[string]interface{}{
			"path":            wmPath,
			"total_bytes":     len(updated),
			"directive_bytes": len(content),
		})
	return nil
}

func parseDreamResult(content string, noteCount int) DreamResult {
	result := DreamResult{
		DreamedAt:  time.Now(),
		InputNotes: noteCount,
	}

	section := ""
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(line, "PATTERNS:"):
			section = "patterns"
		case strings.HasPrefix(line, "INSIGHTS:"):
			section = "insights"
		case strings.HasPrefix(line, "QUESTIONS:"):
			section = "questions"
		case strings.HasPrefix(line, "- "):
			item := strings.TrimPrefix(line, "- ")
			switch section {
			case "patterns":
				result.Patterns = append(result.Patterns, item)
			case "insights":
				result.Insights = append(result.Insights, item)
			case "questions":
				result.Questions = append(result.Questions, item)
			}
		}
	}

	return result
}
