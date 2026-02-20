// SWE100821: Agent identity and time-tracking module.
// Generates a persistent AgentID (survives restarts) and a per-boot SessionID,
// both unique in space (crypto-random) and time (embedded timestamp).
// Tracks boot time and provides uptime queries.

package identity

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/uuid"
)

// AgentIdentity holds the unique identity and timing for an agent instance.
type AgentIdentity struct {
	AgentID   string    `json:"agent_id"`   // Persistent across restarts (created once)
	BirthTime time.Time `json:"birth_time"` // When this agent was first created
	SessionID string    `json:"-"`          // Unique per boot, not persisted
	BootTime  time.Time `json:"-"`          // When this session started

	mu       sync.RWMutex
	filePath string
}

// New loads or creates the agent identity for a given workspace.
// - AgentID + BirthTime are loaded from disk (or created on first run).
// - SessionID + BootTime are always fresh per boot.
func New(workspace string) *AgentIdentity {
	bootTime := time.Now()
	filePath := filepath.Join(workspace, "state", "identity.json")

	id := &AgentIdentity{
		SessionID: generateSessionID(bootTime),
		BootTime:  bootTime,
		filePath:  filePath,
	}

	if err := id.load(); err != nil || id.AgentID == "" {
		// SWE100821: First run — mint a new persistent AgentID
		id.AgentID = uuid.New().String()
		id.BirthTime = bootTime
		id.save()
	}

	return id
}

// Uptime returns the duration since this agent session booted.
func (id *AgentIdentity) Uptime() time.Duration {
	id.mu.RLock()
	defer id.mu.RUnlock()
	return time.Since(id.BootTime)
}

// Age returns the duration since this agent was first created.
func (id *AgentIdentity) Age() time.Duration {
	id.mu.RLock()
	defer id.mu.RUnlock()
	return time.Since(id.BirthTime)
}

// Summary returns a compact human-readable identity string.
func (id *AgentIdentity) Summary() string {
	id.mu.RLock()
	uptime := time.Since(id.BootTime).Truncate(time.Second)
	agentID := id.AgentID
	sessionID := id.SessionID
	bootTime := id.BootTime
	id.mu.RUnlock()
	return fmt.Sprintf("agent=%s session=%s boot=%s uptime=%s",
		agentID, sessionID,
		bootTime.Format("2006-01-02T15:04:05Z"),
		uptime)
}

// ForSystemPrompt returns identity + timing info formatted for the LLM context.
func (id *AgentIdentity) ForSystemPrompt() string {
	id.mu.RLock()
	agentID := id.AgentID
	sessionID := id.SessionID
	birthTime := id.BirthTime
	bootTime := id.BootTime
	uptime := time.Since(id.BootTime).Truncate(time.Second)
	id.mu.RUnlock()
	return fmt.Sprintf(
		"Agent ID: %s\nSession ID: %s\nBirth: %s\nBoot: %s\nUptime: %s",
		agentID,
		sessionID,
		birthTime.Format("2006-01-02 15:04:05"),
		bootTime.Format("2006-01-02 15:04:05"),
		uptime.String(),
	)
}

// generateSessionID creates a per-boot ID encoding timestamp + randomness.
// Format: <unix_ms_hex>-<8_random_bytes_hex>  (unique in space and time).
func generateSessionID(bootTime time.Time) string {
	tsHex := fmt.Sprintf("%x", bootTime.UnixMilli())
	randBytes := make([]byte, 8)
	if _, err := rand.Read(randBytes); err != nil {
		return fmt.Sprintf("%s-%d", tsHex, bootTime.UnixNano())
	}
	return tsHex + "-" + hex.EncodeToString(randBytes)
}

func (id *AgentIdentity) load() error {
	id.mu.Lock()
	defer id.mu.Unlock()

	data, err := os.ReadFile(id.filePath)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, id)
}

func (id *AgentIdentity) save() error {
	id.mu.Lock()
	defer id.mu.Unlock()

	dir := filepath.Dir(id.filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("identity: mkdir %s: %w", dir, err)
	}

	data, err := json.MarshalIndent(id, "", "  ")
	if err != nil {
		return fmt.Errorf("identity: marshal: %w", err)
	}

	tmpFile := id.filePath + ".tmp"
	if err := os.WriteFile(tmpFile, data, 0644); err != nil {
		return fmt.Errorf("identity: write tmp: %w", err)
	}
	if err := os.Rename(tmpFile, id.filePath); err != nil {
		os.Remove(tmpFile)
		return fmt.Errorf("identity: rename: %w", err)
	}
	return nil
}
