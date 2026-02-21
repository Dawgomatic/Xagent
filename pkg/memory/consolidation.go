// SWE100821: Automatic memory consolidation — clusters and summarizes daily notes
// into weekly/monthly summaries, then archives the granular notes.
// Runs as a periodic cron job to prevent memory bloat while preserving knowledge.

package memory

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/Dawgomatic/Xagent/pkg/logger"
	"github.com/Dawgomatic/Xagent/pkg/providers"
)

// Consolidator manages periodic memory consolidation.
type Consolidator struct {
	workspace   string
	provider    providers.LLMProvider
	model       string
	memoryDir   string
	archiveDir  string
}

// NewConsolidator creates a memory consolidator for the given workspace.
func NewConsolidator(workspace string, provider providers.LLMProvider, model string) *Consolidator {
	memDir := filepath.Join(workspace, "memory")
	archDir := filepath.Join(memDir, "archive")
	os.MkdirAll(archDir, 0755)

	return &Consolidator{
		workspace:  workspace,
		provider:   provider,
		model:      model,
		memoryDir:  memDir,
		archiveDir: archDir,
	}
}

// ConsolidateWeekly collects daily notes from the past 7 days,
// summarizes them via LLM, writes a weekly summary, and archives the daily notes.
func (c *Consolidator) ConsolidateWeekly(ctx context.Context) error {
	now := time.Now()
	weekStart := now.AddDate(0, 0, -7)

	// Collect daily notes from the past week
	notes, files := c.collectDailyNotes(weekStart, now)
	if len(notes) == 0 {
		logger.InfoCF("consolidation", "No daily notes to consolidate", nil)
		return nil
	}

	// Summarize via LLM
	combined := strings.Join(notes, "\n\n---\n\n")
	summary, err := c.summarize(ctx, combined, "weekly")
	if err != nil {
		return fmt.Errorf("weekly consolidation failed: %w", err)
	}

	// Write weekly summary
	weekLabel := weekStart.Format("20060102") + "-" + now.Format("20060102")
	summaryFile := filepath.Join(c.memoryDir, "weekly", weekLabel+".md")
	os.MkdirAll(filepath.Dir(summaryFile), 0755)

	header := fmt.Sprintf("# Weekly Summary: %s to %s\n\n", weekStart.Format("2006-01-02"), now.Format("2006-01-02"))
	if err := os.WriteFile(summaryFile, []byte(header+summary), 0600); err != nil {
		return fmt.Errorf("failed to write weekly summary: %w", err)
	}

	// Archive the daily note files
	for _, f := range files {
		archivePath := filepath.Join(c.archiveDir, filepath.Base(f))
		os.Rename(f, archivePath)
	}

	logger.InfoCF("consolidation", "Weekly consolidation complete",
		map[string]interface{}{
			"notes_consolidated": len(files),
			"summary_file":      summaryFile,
		})

	return nil
}

// ConsolidateMonthly collects weekly summaries from the past month,
// produces a monthly summary, and archives the weekly files.
func (c *Consolidator) ConsolidateMonthly(ctx context.Context) error {
	weeklyDir := filepath.Join(c.memoryDir, "weekly")
	entries, err := os.ReadDir(weeklyDir)
	if err != nil || len(entries) < 4 {
		return nil // not enough weekly summaries yet
	}

	var notes []string
	var files []string
	cutoff := time.Now().AddDate(0, -1, 0)

	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		if info.ModTime().Before(cutoff) {
			data, err := os.ReadFile(filepath.Join(weeklyDir, e.Name()))
			if err == nil {
				notes = append(notes, string(data))
				files = append(files, filepath.Join(weeklyDir, e.Name()))
			}
		}
	}

	if len(notes) == 0 {
		return nil
	}

	combined := strings.Join(notes, "\n\n---\n\n")
	summary, err := c.summarize(ctx, combined, "monthly")
	if err != nil {
		return fmt.Errorf("monthly consolidation failed: %w", err)
	}

	monthLabel := time.Now().AddDate(0, -1, 0).Format("200601")
	summaryFile := filepath.Join(c.memoryDir, "monthly", monthLabel+".md")
	os.MkdirAll(filepath.Dir(summaryFile), 0755)

	header := fmt.Sprintf("# Monthly Summary: %s\n\n", time.Now().AddDate(0, -1, 0).Format("January 2006"))
	if err := os.WriteFile(summaryFile, []byte(header+summary), 0600); err != nil {
		return fmt.Errorf("failed to write monthly summary: %w", err)
	}

	// Append key insights to MEMORY.md
	memoryFile := filepath.Join(c.memoryDir, "MEMORY.md")
	f, err := os.OpenFile(memoryFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err == nil {
		insight := fmt.Sprintf("\n\n## Consolidated: %s\n%s\n", time.Now().AddDate(0, -1, 0).Format("January 2006"), summary)
		f.WriteString(insight)
		f.Close()
	}

	// Archive weekly files
	for _, fpath := range files {
		archivePath := filepath.Join(c.archiveDir, filepath.Base(fpath))
		os.Rename(fpath, archivePath)
	}

	logger.InfoCF("consolidation", "Monthly consolidation complete",
		map[string]interface{}{
			"weeks_consolidated": len(files),
			"summary_file":      summaryFile,
		})

	return nil
}

func (c *Consolidator) collectDailyNotes(from, to time.Time) ([]string, []string) {
	var notes []string
	var files []string

	for d := from; d.Before(to); d = d.AddDate(0, 0, 1) {
		dateStr := d.Format("20060102")
		monthDir := dateStr[:6]
		filePath := filepath.Join(c.memoryDir, monthDir, dateStr+".md")

		data, err := os.ReadFile(filePath)
		if err == nil {
			notes = append(notes, string(data))
			files = append(files, filePath)
		}
	}

	return notes, files
}

func (c *Consolidator) summarize(ctx context.Context, content, period string) (string, error) {
	prompt := fmt.Sprintf(`Summarize the following %s notes into a concise knowledge summary.
Focus on:
- Key facts learned
- Important decisions made
- Recurring topics or patterns
- Action items or open questions

Keep the summary to 200-400 words.

NOTES:
%s`, period, content)

	resp, err := c.provider.Chat(ctx, []providers.Message{
		{Role: "user", Content: prompt},
	}, nil, c.model, map[string]interface{}{
		"max_tokens":  1024,
		"temperature": 0.3,
	})
	if err != nil {
		return "", err
	}

	return resp.Content, nil
}

// ListConsolidations returns available weekly and monthly summaries.
func (c *Consolidator) ListConsolidations() map[string][]string {
	result := map[string][]string{
		"weekly":  {},
		"monthly": {},
	}

	for _, period := range []string{"weekly", "monthly"} {
		dir := filepath.Join(c.memoryDir, period)
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		sort.Slice(entries, func(i, j int) bool {
			return entries[i].Name() > entries[j].Name()
		})
		for _, e := range entries {
			if !e.IsDir() {
				result[period] = append(result[period], e.Name())
			}
		}
	}

	return result
}
