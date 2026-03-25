// SWE100821: Daily vault consolidation — moves full session note bodies to Sessions/Archive/YYYY-MM-DD/
// and replaces originals with short stubs + wikilinks so the Obsidian graph stays connected
// while on-disk size and node body size shrink. Original data is preserved under Archive/.
package vault

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Dawgomatic/Xagent/pkg/logger"
)

const (
	consolidatedFrontmatterKey = "xagent_consolidated: true"
	minSessionBytes            = 512 // skip tiny files (already compact)
)

// ConsolidateSessionsForDate moves session notes for the given calendar day (UTC) into
// Sessions/Archive/YYYY-MM-DD/ and writes stub files at the original paths.
// Reused: vault graph links to session filenames; stubs keep the same basename.
func ConsolidateSessionsForDate(root string, day time.Time) (n int, err error) {
	dateStr := day.UTC().Format("2006-01-02")
	sessionsDir := filepath.Join(root, "Sessions")
	if st, e := os.Stat(sessionsDir); e != nil || !st.IsDir() {
		if e != nil && os.IsNotExist(e) {
			return 0, nil
		}
		if e != nil {
			return 0, e
		}
		return 0, nil
	}

	archiveDir := filepath.Join(sessionsDir, "Archive", dateStr)
	if err := os.MkdirAll(archiveDir, 0755); err != nil {
		return 0, fmt.Errorf("vault consolidate: mkdir archive: %w", err)
	}
	rollupDir := filepath.Join(root, "Daily", "Consolidated")
	if err := os.MkdirAll(rollupDir, 0755); err != nil {
		return 0, fmt.Errorf("vault consolidate: mkdir rollup: %w", err)
	}

	entries, err := os.ReadDir(sessionsDir)
	if err != nil {
		return 0, err
	}

	prefix := "Session " + dateStr
	var rollupLines []string

	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.HasSuffix(strings.ToLower(name), ".md") {
			continue
		}
		if !strings.HasPrefix(name, prefix) {
			continue
		}

		src := filepath.Join(sessionsDir, name)
		st, statErr := os.Stat(src)
		if statErr != nil || st.Size() < minSessionBytes {
			continue
		}

		body, readErr := os.ReadFile(src)
		if readErr != nil {
			return n, readErr
		}
		if fileLooksConsolidated(body) {
			continue
		}

		dst := filepath.Join(archiveDir, name)
		if err := os.Rename(src, dst); err != nil {
			return n, fmt.Errorf("vault consolidate: rename %s: %w", name, err)
		}

		stub := buildSessionStub(name, dateStr, extractTitleLine(body))
		writeErr := os.WriteFile(filepath.Join(sessionsDir, name), []byte(stub), 0644)
		if writeErr != nil {
			// best-effort rollback: move archive back
			_ = os.Rename(dst, src)
			return n, fmt.Errorf("vault consolidate: write stub %s: %w", name, writeErr)
		}

		n++
		base := strings.TrimSuffix(name, ".md")
		linkArchive := fmt.Sprintf("[[Sessions/Archive/%s/%s]]", dateStr, base)
		rollupLines = append(rollupLines, fmt.Sprintf("- Stub [[%s]] → full %s", base, linkArchive))
	}

	if len(rollupLines) > 0 {
		rollupPath := filepath.Join(rollupDir, dateStr+" rollup.md")
		rollup := buildRollup(dateStr, rollupLines, len(rollupLines))
		wErr := os.WriteFile(rollupPath, []byte(rollup), 0644)
		if wErr != nil {
			logger.WarnCF("vault", "Failed to write consolidation rollup", map[string]interface{}{"error": wErr.Error()})
		}
	}

	return n, nil
}

// ConsolidateYesterdayUTC consolidates session notes for yesterday (UTC calendar day).
func ConsolidateYesterdayUTC(root string) (int, error) {
	y := time.Now().UTC().AddDate(0, 0, -1)
	return ConsolidateSessionsForDate(root, y)
}

// StartDailyArchiveScheduler runs ConsolidateYesterdayUTC once per day at hourUTC (0–23, UTC).
// SWE100821: Waits until the next occurrence of that hour, then repeats every 24h until ctx done.
func StartDailyArchiveScheduler(ctx context.Context, root string, hourUTC int) {
	if hourUTC < 0 || hourUTC > 23 {
		hourUTC = 4
	}
	go func() {
		for {
			wait := durationUntilNextHourUTC(hourUTC)
			select {
			case <-ctx.Done():
				return
			case <-time.After(wait):
			}
			n, err := ConsolidateYesterdayUTC(root)
			if err != nil {
				logger.WarnCF("vault", "Daily consolidation failed", map[string]interface{}{"error": err.Error()})
			} else if n > 0 {
				logger.InfoCF("vault", "Daily vault consolidation complete", map[string]interface{}{
					"archived_notes": n,
					"vault":          root,
				})
			}
		}
	}()
}

func durationUntilNextHourUTC(hour int) time.Duration {
	now := time.Now().UTC()
	next := time.Date(now.Year(), now.Month(), now.Day(), hour, 0, 0, 0, time.UTC)
	if !next.After(now) {
		next = next.Add(24 * time.Hour)
	}
	return next.Sub(now)
}

func fileLooksConsolidated(body []byte) bool {
	s := bufio.NewScanner(strings.NewReader(string(body)))
	n := 0
	for s.Scan() && n < 48 {
		n++
		if strings.Contains(s.Text(), consolidatedFrontmatterKey) {
			return true
		}
	}
	return false
}

func extractTitleLine(body []byte) string {
	s := bufio.NewScanner(strings.NewReader(string(body)))
	for s.Scan() {
		line := strings.TrimSpace(s.Text())
		if strings.HasPrefix(line, "# ") {
			return strings.TrimPrefix(line, "# ")
		}
	}
	return ""
}

func buildSessionStub(filename, dateStr, titleHint string) string {
	base := strings.TrimSuffix(filename, ".md")
	title := titleHint
	if title == "" {
		title = base
	}
	archLink := fmt.Sprintf("[[Sessions/Archive/%s/%s]]", dateStr, base)
	return fmt.Sprintf(`---
tags: [session, consolidated]
date: %s
%s
---

# %s

**Archived** — full conversation (preserved): %s

*This stub keeps the same filename so incoming [[wikilinks]] from Daily/Tools/Topics still resolve. The graph stays connected; open the archive link for the full text.*

`, dateStr, consolidatedFrontmatterKey, title, archLink)
}

func buildRollup(dateStr string, lines []string, count int) string {
	var b strings.Builder
	b.WriteString("---\ntags: [consolidated, daily, vault]\n")
	b.WriteString(fmt.Sprintf("date: %s\n---\n\n", dateStr))
	b.WriteString(fmt.Sprintf("# Vault consolidation — %s\n\n", dateStr))
	b.WriteString(fmt.Sprintf("**Session notes archived (bodies moved to `Sessions/Archive/%s/`):** %d\n\n", dateStr, count))
	b.WriteString(strings.Join(lines, "\n"))
	b.WriteString("\n")
	return b.String()
}
