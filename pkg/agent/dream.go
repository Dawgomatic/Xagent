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

	// Gather material to reflect on
	recentNotes := dm.memory.GetRecentDailyNotes(7)
	longTerm := dm.memory.ReadLongTerm()

	if recentNotes == "" && longTerm == "" {
		logger.InfoCF("dream", "Nothing to dream about (no notes or memory)", nil)
		return
	}

	material := ""
	noteCount := 0
	if recentNotes != "" {
		material += "## Recent Daily Notes (last 7 days)\n\n" + recentNotes
		noteCount = strings.Count(recentNotes, "# 20") // rough count by date headers
	}
	if longTerm != "" {
		material += "\n\n## Long-term Memory\n\n" + longTerm
	}

	// SWE100821: Ask the LLM to reflect
	dreamPrompt := fmt.Sprintf(`You are in dream mode — a quiet time for autonomous reflection.

Review the following notes and memories. Think deeply about:
1. Patterns or recurring themes across days
2. Contradictions or inconsistencies in information
3. Questions that remain unanswered
4. Connections between different topics
5. Insights that synthesize multiple observations

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

Be concise. Focus on genuinely novel observations.

MATERIAL:
%s`, truncate(material, 3000))

	resp, err := dm.provider.Chat(ctx, []providers.Message{
		{Role: "user", Content: dreamPrompt},
	}, nil, dm.model, map[string]interface{}{
		"max_tokens":  1024,
		"temperature": 0.8, // SWE100821: higher temperature for creative reflection
	})

	if err != nil {
		logger.WarnCF("dream", "Dream reflection failed", map[string]interface{}{"error": err.Error()})
		return
	}

	result := parseDreamResult(resp.Content, noteCount)

	// Write insights to daily notes
	if len(result.Insights) > 0 || len(result.Patterns) > 0 {
		var dreamNote strings.Builder
		dreamNote.WriteString("## 💭 Dream Mode Reflection\n\n")

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

		dm.memory.AppendToday(dreamNote.String())
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
		cb(fmt.Sprintf("💭 While reflecting during idle time, I noticed: %s", result.Insights[0]))
	}

	logger.InfoCF("dream", "Dream mode complete",
		map[string]interface{}{
			"patterns":  len(result.Patterns),
			"insights":  len(result.Insights),
			"questions": len(result.Questions),
		})
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
