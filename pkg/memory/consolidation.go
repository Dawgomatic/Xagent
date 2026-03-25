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

	// SWE100821: upgrade_period — start of scheduled weekly rollup
	logger.InfoCF("upgrade_period", "consolidation weekly started",
		map[string]interface{}{
			"window_from": weekStart.Format(time.RFC3339),
			"window_to":   now.Format(time.RFC3339),
		})

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
	written := header + summary
	if err := os.WriteFile(summaryFile, []byte(written), 0600); err != nil {
		return fmt.Errorf("failed to write weekly summary: %w", err)
	}
	logger.InfoCF("upgrade_period", "weekly summary file written",
		map[string]interface{}{"path": summaryFile, "bytes": len(written)})

	// SWE100821: Archive daily note files — log rename errors instead of ignoring
	archiveOK := 0
	for _, f := range files {
		archivePath := filepath.Join(c.archiveDir, filepath.Base(f))
		if err := os.Rename(f, archivePath); err != nil {
			logger.WarnCF("consolidation", "Failed to archive daily note",
				map[string]interface{}{"file": f, "error": err.Error()})
		} else {
			archiveOK++
		}
	}

	logger.InfoCF("consolidation", "Weekly consolidation complete",
		map[string]interface{}{
			"notes_consolidated": len(files),
			"notes_archived":    archiveOK,
			"summary_file":      summaryFile,
		})
	// SWE100821: upgrade_period — mirror for grep/metrics
	logger.InfoCF("upgrade_period", "consolidation weekly finished",
		map[string]interface{}{
			"notes_consolidated": len(files),
			"notes_archived":     archiveOK,
			"summary_file":       summaryFile,
		})

	return nil
}

// ConsolidateMonthly collects weekly summaries from the past month,
// produces a monthly summary, and archives the weekly files.
func (c *Consolidator) ConsolidateMonthly(ctx context.Context) error {
	weeklyDir := filepath.Join(c.memoryDir, "weekly")
	entries, err := os.ReadDir(weeklyDir)
	if err != nil {
		if os.IsNotExist(err) {
			logger.InfoCF("upgrade_period", "consolidation monthly skipped: no weekly directory yet", nil)
		} else {
			logger.WarnCF("upgrade_period", "consolidation monthly: read weekly dir failed",
				map[string]interface{}{"error": err.Error(), "dir": weeklyDir})
		}
		return nil
	}
	if len(entries) < 4 {
		// SWE100821: upgrade_period — explicit skip (was silent)
		logger.InfoCF("upgrade_period", "consolidation monthly skipped: need at least 4 weekly files",
			map[string]interface{}{"weekly_file_count": len(entries)})
		return nil
	}

	logger.InfoCF("upgrade_period", "consolidation monthly started",
		map[string]interface{}{"weekly_candidates": len(entries)})

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
		logger.InfoCF("upgrade_period", "consolidation monthly skipped: no stale weekly files to roll up", nil)
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
	monthlyBody := header + summary
	if err := os.WriteFile(summaryFile, []byte(monthlyBody), 0600); err != nil {
		return fmt.Errorf("failed to write monthly summary: %w", err)
	}
	logger.InfoCF("upgrade_period", "monthly summary file written",
		map[string]interface{}{"path": summaryFile, "bytes": len(monthlyBody)})

	// SWE100821: Append key insights to MEMORY.md — propagate write errors
	memoryFile := filepath.Join(c.memoryDir, "MEMORY.md")
	if f, fErr := os.OpenFile(memoryFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600); fErr != nil {
		logger.WarnCF("consolidation", "Failed to open MEMORY.md for append",
			map[string]interface{}{"error": fErr.Error()})
	} else {
		insight := fmt.Sprintf("\n\n## Consolidated: %s\n%s\n", time.Now().AddDate(0, -1, 0).Format("January 2006"), summary)
		if _, wErr := f.WriteString(insight); wErr != nil {
			logger.WarnCF("consolidation", "Failed to write to MEMORY.md",
				map[string]interface{}{"error": wErr.Error()})
		} else {
			// SWE100821: upgrade_period — long-term memory append during monthly rollup
			logger.InfoCF("upgrade_period", "MEMORY.md appended from monthly consolidation",
				map[string]interface{}{
					"path":        memoryFile,
					"bytes_added": len(insight),
				})
		}
		f.Close()
	}

	// SWE100821: Archive weekly files — log rename errors
	archiveOKM := 0
	for _, fpath := range files {
		archivePath := filepath.Join(c.archiveDir, filepath.Base(fpath))
		if err := os.Rename(fpath, archivePath); err != nil {
			logger.WarnCF("consolidation", "Failed to archive weekly note",
				map[string]interface{}{"file": fpath, "error": err.Error()})
		} else {
			archiveOKM++
		}
	}

	logger.InfoCF("consolidation", "Monthly consolidation complete",
		map[string]interface{}{
			"weeks_consolidated": len(files),
			"weeks_archived":    archiveOKM,
			"summary_file":      summaryFile,
		})
	logger.InfoCF("upgrade_period", "consolidation monthly finished",
		map[string]interface{}{
			"weeks_consolidated": len(files),
			"weeks_archived":     archiveOKM,
			"summary_file":       summaryFile,
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
