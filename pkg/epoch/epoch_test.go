package epoch

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Dawgomatic/Xagent/pkg/identity"
)

func TestRollover(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "epoch_test_")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	id := &identity.AgentIdentity{
		SessionID: "test-session",
		AgentID:   "test-agent",
		BootTime:  time.Now(),
	}

	mgr := NewManager(tmpDir, id)
	_, err = mgr.Wake()
	if err != nil {
		t.Fatal(err)
	}

	// Record an event and update stats
	mgr.RecordEvent("TEST", "Test event")
	mgr.UpdateStats(func(s *EpochStats) {
		s.FatigueLevel = 0.5
		s.IsSleeping = true
	})

	// Wait a tiny bit to avoid timestamp collisions on some systems
	time.Sleep(10 * time.Millisecond)

	err = mgr.Rollover("test rollover")
	if err != nil {
		t.Fatal(err)
	}

	// Verify current record is new and carried over stats
	current := mgr.GetCurrent()
	if current == nil {
		t.Fatal("Current epoch is nil")
	}

	if current.Stats.FatigueLevel != 0.5 {
		t.Errorf("Expected FatigueLevel 0.5, got %v", current.Stats.FatigueLevel)
	}

	if !current.Stats.IsSleeping {
		t.Errorf("Expected IsSleeping to be true")
	}

	if len(current.Events) != 0 {
		t.Errorf("Expected 0 events in new epoch, got %v", len(current.Events))
	}

	// Verify old epoch was saved
	entries, err := os.ReadDir(filepath.Join(tmpDir, "epochs"))
	if err != nil {
		t.Fatal(err)
	}

	if len(entries) != 1 {
		t.Errorf("Expected 1 saved epoch file, got %v", len(entries))
	}
}
