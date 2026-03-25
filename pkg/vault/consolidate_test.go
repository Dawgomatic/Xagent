package vault

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestConsolidateSessionsForDate moves large session files to Archive and leaves stubs.
func TestConsolidateSessionsForDate(t *testing.T) {
	root := t.TempDir()
	vw := NewVaultWriter(root)
	if err := vw.Init(); err != nil {
		t.Fatal(err)
	}
	day := time.Date(2026, 1, 14, 12, 0, 0, 0, time.UTC)
	name := "Session 2026-01-14 15-04 test-key.md"
	body := strings.Repeat("x", 600) + "\n# Session title\n\nbody"
	path := filepath.Join(root, "Sessions", name)
	if err := os.WriteFile(path, []byte(body), 0644); err != nil {
		t.Fatal(err)
	}

	n, err := ConsolidateSessionsForDate(root, day)
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("expected 1 consolidated, got %d", n)
	}

	arch := filepath.Join(root, "Sessions", "Archive", "2026-01-14", name)
	if _, err := os.Stat(arch); err != nil {
		t.Fatalf("archive missing: %v", err)
	}
	stub, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(stub), consolidatedFrontmatterKey) {
		t.Fatal("stub missing consolidation marker")
	}
	if !strings.Contains(string(stub), "Sessions/Archive/2026-01-14") {
		t.Fatal("stub missing archive wikilink path")
	}
}

func TestConsolidateSessionsForDate_SkipsAlreadyStub(t *testing.T) {
	root := t.TempDir()
	vw := NewVaultWriter(root)
	if err := vw.Init(); err != nil {
		t.Fatal(err)
	}
	day := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	name := "Session 2026-02-01 10-00 x.md"
	stub := "---\n" + consolidatedFrontmatterKey + "\n---\n"
	path := filepath.Join(root, "Sessions", name)
	if err := os.WriteFile(path, []byte(stub+strings.Repeat("z", 600)), 0644); err != nil {
		t.Fatal(err)
	}
	n, err := ConsolidateSessionsForDate(root, day)
	if err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Fatalf("expected 0, got %d", n)
	}
}

func TestDurationUntilNextHourUTC(t *testing.T) {
	d := durationUntilNextHourUTC(12)
	if d <= 0 || d > 25*time.Hour {
		t.Fatalf("unexpected duration: %v", d)
	}
}
