package upgrade

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const upgradeCheckInterval = 7 * 24 * time.Hour

func ShouldCheckForUpgrade(workspace string) bool {
	logFile := filepath.Join(workspace, "upgrade_check.log")
	info, err := os.Stat(logFile)
	if err != nil {
		return true
	}
	return time.Since(info.ModTime()) > upgradeCheckInterval
}

func RecordUpgradeCheck(workspace string) error {
	logFile := filepath.Join(workspace, "upgrade_check.log")
	content := fmt.Sprintf("last_check=%s\n", time.Now().Format(time.RFC3339))
	return os.WriteFile(logFile, []byte(content), 0644)
}
