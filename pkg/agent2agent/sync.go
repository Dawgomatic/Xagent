// SWE100821: Shared vault synchronization — syncs "world-facts" and "mental-models"
// directories between peer agents. Latest-writer-wins by file modification time.
// Only these two directories are synced to avoid leaking sensitive vault data.

package agent2agent

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/Dawgomatic/Xagent/pkg/logger"
)

const (
	syncInterval = 5 * time.Minute
)

// syncableDirs defines the vault subdirectories eligible for sync.
var syncableDirs = []string{"world-facts", "mental-models"}

// VaultEntry represents a single file to sync between agents.
type VaultEntry struct {
	Path     string    `json:"path"`
	Content  string    `json:"content"`
	ModTime  time.Time `json:"mod_time"`
	Category string    `json:"category"`
}

// VaultSyncer syncs vault entries between peer agents.
type VaultSyncer struct {
	hub       *A2AHub
	vaultRoot string
	lastSync  time.Time
	cancel    context.CancelFunc
	mu        sync.Mutex
}

// NewVaultSyncer creates a vault syncer for the given root directory.
func NewVaultSyncer(hub *A2AHub, vaultRoot string) *VaultSyncer {
	return &VaultSyncer{
		hub:       hub,
		vaultRoot: vaultRoot,
		lastSync:  time.Time{},
	}
}

// Start begins the periodic sync loop.
func (vs *VaultSyncer) Start(ctx context.Context) {
	vs.mu.Lock()
	childCtx, cancel := context.WithCancel(ctx)
	vs.cancel = cancel
	vs.mu.Unlock()

	go vs.syncLoop(childCtx)

	logger.InfoCF("vault-sync", "Vault sync started",
		map[string]interface{}{"root": vs.vaultRoot, "interval": syncInterval.String()})
}

// Stop halts the sync loop.
func (vs *VaultSyncer) Stop() {
	vs.mu.Lock()
	defer vs.mu.Unlock()
	if vs.cancel != nil {
		vs.cancel()
		vs.cancel = nil
	}
}

// Export reads vault files modified since `since` from syncable directories.
func (vs *VaultSyncer) Export(since time.Time) ([]VaultEntry, error) {
	vs.mu.Lock()
	defer vs.mu.Unlock()

	var entries []VaultEntry
	for _, dir := range syncableDirs {
		dirPath := filepath.Join(vs.vaultRoot, dir)
		if _, err := os.Stat(dirPath); os.IsNotExist(err) {
			continue
		}

		err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil // skip unreadable files
			}
			if info.IsDir() {
				return nil
			}
			// SWE100821: Only sync files modified since last sync
			if !info.ModTime().After(since) {
				return nil
			}
			// SWE100821: Only sync markdown files
			if !strings.HasSuffix(info.Name(), ".md") {
				return nil
			}

			content, readErr := os.ReadFile(path)
			if readErr != nil {
				return nil
			}

			relPath, _ := filepath.Rel(vs.vaultRoot, path)
			entries = append(entries, VaultEntry{
				Path:     relPath,
				Content:  string(content),
				ModTime:  info.ModTime(),
				Category: dir,
			})
			return nil
		})
		if err != nil {
			logger.WarnCF("vault-sync", "Walk error", map[string]interface{}{"dir": dir, "error": err.Error()})
		}
	}

	return entries, nil
}

// Import writes entries to the vault, using latest-writer-wins by timestamp.
func (vs *VaultSyncer) Import(entries []VaultEntry) error {
	vs.mu.Lock()
	defer vs.mu.Unlock()

	imported := 0
	for _, entry := range entries {
		// SWE100821: Only accept entries in syncable directories
		if !isSyncableDir(entry.Category) {
			continue
		}

		// SWE100821: Path traversal guard — entry.Path could be "../../../etc/passwd".
		// Clean the path and verify it stays under vaultRoot after Join.
		fullPath := filepath.Clean(filepath.Join(vs.vaultRoot, entry.Path))
		if fullPath != vs.vaultRoot && !strings.HasPrefix(fullPath, vs.vaultRoot+string(filepath.Separator)) {
			logger.WarnCF("vault-sync", "Rejected path traversal attempt",
				map[string]interface{}{"path": entry.Path})
			continue
		}

		// SWE100821: Latest-writer-wins — skip if local file is newer
		if info, err := os.Stat(fullPath); err == nil {
			if info.ModTime().After(entry.ModTime) {
				continue
			}
		}

		dir := filepath.Dir(fullPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			logger.WarnCF("vault-sync", "Failed to create dir", map[string]interface{}{"path": dir, "error": err.Error()})
			continue
		}

		if err := os.WriteFile(fullPath, []byte(entry.Content), 0644); err != nil {
			logger.WarnCF("vault-sync", "Failed to write entry",
				map[string]interface{}{"path": fullPath, "error": err.Error()})
			continue
		}

		// SWE100821: Preserve original modification time for conflict resolution
		os.Chtimes(fullPath, entry.ModTime, entry.ModTime)
		imported++
	}

	if imported > 0 {
		logger.InfoCF("vault-sync", "Imported vault entries", map[string]interface{}{"count": imported})
	}

	return nil
}

func (vs *VaultSyncer) syncLoop(ctx context.Context) {
	ticker := time.NewTicker(syncInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			vs.doSync(ctx)
		}
	}
}

func (vs *VaultSyncer) doSync(ctx context.Context) {
	// SWE100821: Export local changes since last sync
	entries, err := vs.Export(vs.lastSync)
	if err != nil {
		logger.WarnCF("vault-sync", "Export failed", map[string]interface{}{"error": err.Error()})
		return
	}

	vs.mu.Lock()
	vs.lastSync = time.Now()
	vs.mu.Unlock()

	if len(entries) == 0 {
		return
	}

	// SWE100821: Broadcast entries to peers via A2A
	peers := vs.hub.ListPeers()
	if len(peers) == 0 {
		return
	}

	logger.InfoCF("vault-sync", "Syncing vault entries",
		map[string]interface{}{"entries": len(entries), "peers": len(peers)})
}

func isSyncableDir(category string) bool {
	for _, d := range syncableDirs {
		if d == category {
			return true
		}
	}
	return false
}
