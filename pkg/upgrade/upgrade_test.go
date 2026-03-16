// SWE100821: Tests for upgrade — verifies UpgradeFromCheckpoint with unreachable URL.
package upgrade

import (
	"strings"
	"testing"
)

// SWE100821: TestUpgradeFromCheckpoint_NoServer — unreachable URL returns error
func TestUpgradeFromCheckpoint_NoServer(t *testing.T) {
	dir := t.TempDir()

	// SWE100821: Use a URL that will refuse connection
	_, err := UpgradeFromCheckpoint("http://127.0.0.1:1", dir)
	if err == nil {
		t.Fatal("expected error for unreachable RL server")
	}
	if !strings.Contains(err.Error(), "fetching RL checkpoint") {
		t.Errorf("error = %q, want to contain 'fetching RL checkpoint'", err.Error())
	}
}
