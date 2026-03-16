// SWE100821: Tests for vault synchronization — verifies export, import,
// and latest-writer-wins conflict resolution.
package agent2agent

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// SWE100821: TestExport — create temp vault with world-facts/test.md, export since epoch
func TestExport(t *testing.T) {
	dir := t.TempDir()
	wfDir := filepath.Join(dir, "world-facts")
	if err := os.MkdirAll(wfDir, 0755); err != nil {
		t.Fatal(err)
	}
	content := "# Earth\nThird planet from the sun."
	if err := os.WriteFile(filepath.Join(wfDir, "test.md"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	hub := NewA2AHub("exporter")
	vs := NewVaultSyncer(hub, dir)

	// SWE100821: Export since epoch → should capture everything
	entries, err := vs.Export(time.Time{})
	if err != nil {
		t.Fatalf("Export() error: %v", err)
	}
	if len(entries) == 0 {
		t.Fatal("expected at least 1 entry from export")
	}

	found := false
	for _, e := range entries {
		if e.Path == filepath.Join("world-facts", "test.md") {
			found = true
			if e.Content != content {
				t.Errorf("content mismatch: got %q", e.Content)
			}
			if e.Category != "world-facts" {
				t.Errorf("category = %q, want \"world-facts\"", e.Category)
			}
		}
	}
	if !found {
		t.Error("expected world-facts/test.md in exported entries")
	}
}

// SWE100821: TestImport — import entries to temp dir, verify files written
func TestImport(t *testing.T) {
	dir := t.TempDir()
	hub := NewA2AHub("importer")
	vs := NewVaultSyncer(hub, dir)

	entries := []VaultEntry{
		{
			Path:     filepath.Join("world-facts", "imported.md"),
			Content:  "# Imported Fact\nThis was synced.",
			ModTime:  time.Now(),
			Category: "world-facts",
		},
	}

	if err := vs.Import(entries); err != nil {
		t.Fatalf("Import() error: %v", err)
	}

	// SWE100821: Verify file written to disk
	data, err := os.ReadFile(filepath.Join(dir, "world-facts", "imported.md"))
	if err != nil {
		t.Fatalf("reading imported file: %v", err)
	}
	if string(data) != entries[0].Content {
		t.Errorf("imported content = %q, want %q", string(data), entries[0].Content)
	}
}

// SWE100821: TestImport_LatestWriterWins — local file newer than import entry → not overwritten
func TestImport_LatestWriterWins(t *testing.T) {
	dir := t.TempDir()
	hub := NewA2AHub("winner")
	vs := NewVaultSyncer(hub, dir)

	localPath := filepath.Join(dir, "world-facts", "conflict.md")
	if err := os.MkdirAll(filepath.Dir(localPath), 0755); err != nil {
		t.Fatal(err)
	}
	localContent := "# Local Version\nI am newer."
	if err := os.WriteFile(localPath, []byte(localContent), 0644); err != nil {
		t.Fatal(err)
	}

	// SWE100821: Import entry is older than the local file
	entries := []VaultEntry{
		{
			Path:     filepath.Join("world-facts", "conflict.md"),
			Content:  "# Remote Version\nI am older.",
			ModTime:  time.Now().Add(-1 * time.Hour), // 1 hour in the past
			Category: "world-facts",
		},
	}

	if err := vs.Import(entries); err != nil {
		t.Fatalf("Import() error: %v", err)
	}

	// SWE100821: Local file should NOT be overwritten
	data, err := os.ReadFile(localPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != localContent {
		t.Errorf("local file was overwritten; got %q, want %q", string(data), localContent)
	}
}
