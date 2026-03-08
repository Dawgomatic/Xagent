// Xagent - Ultra-lightweight personal AI agent
// Obsidian Knowledge Vault — core vault writer
// Creates Obsidian-compatible markdown notes with [[wikilinks]] for graph view.
//
// Copyright (c) 2026 Xagent contributors
// License: MIT

package vault

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/Dawgomatic/Xagent/pkg/logger"
)

// VaultWriter manages an Obsidian-compatible knowledge vault.
// Each agent turn produces session notes with [[wikilinks]] linking to
// tools, channels, models, topics, and daily notes — forming a rich graph.
type VaultWriter struct {
	root   string // e.g. ~/.xagent/vault
	mu     sync.Mutex
	topics *TopicRegistry
}

// SessionData contains the data needed to write a session note after a turn.
type SessionData struct {
	SessionKey  string
	Channel     string
	Model       string
	UserMessage string // first 300 chars
	Response    string // first 500 chars
	ToolsUsed   []string
	LatencyMs   int64
	Iterations  int
	PlanSteps   int
	SkillsUsed  []string
	MemoryHits  []string
	Timestamp   time.Time
}

// DreamData contains dream mode output for vault notes.
type DreamData struct {
	Insights  []string
	Patterns  []string
	Questions []string
	Timestamp time.Time
}

// PersonalityData contains personality changes for vault notes.
type PersonalityData struct {
	Trait  string
	Old    float64
	New    float64
	Reason string
	Date   time.Time
}

// NewVaultWriter creates a new vault writer at the given root path.
func NewVaultWriter(root string) *VaultWriter {
	return &VaultWriter{
		root:   root,
		topics: NewTopicRegistry(),
	}
}

// Init creates the vault directory structure and README.
func (v *VaultWriter) Init() error {
	v.mu.Lock()
	defer v.mu.Unlock()

	dirs := []string{
		"Sessions",
		"Daily",
		"Tools",
		"Topics",
		"Dreams",
		"Personality",
		"Channels",
		"Models",
	}

	for _, d := range dirs {
		if err := os.MkdirAll(filepath.Join(v.root, d), 0755); err != nil {
			return fmt.Errorf("creating vault dir %s: %w", d, err)
		}
	}

	// Write README if it doesn't exist
	readmePath := filepath.Join(v.root, "README.md")
	if _, err := os.Stat(readmePath); os.IsNotExist(err) {
		if err := os.WriteFile(readmePath, []byte(vaultReadme), 0644); err != nil {
			return fmt.Errorf("writing vault README: %w", err)
		}
	}

	logger.InfoCF("vault", "Obsidian vault initialized", map[string]interface{}{"path": v.root})
	return nil
}

// WriteSessionNote creates a session note with wikilinks to related entities.
func (v *VaultWriter) WriteSessionNote(data SessionData) error {
	v.mu.Lock()
	defer v.mu.Unlock()

	if data.Timestamp.IsZero() {
		data.Timestamp = time.Now()
	}

	// Extract topics from user message
	topics := ExtractTopics(data.UserMessage)

	// Build the note
	note := renderSessionNote(data, topics)

	// Write session note
	filename := sanitizeName(fmt.Sprintf("Session %s %s",
		data.Timestamp.Format("2006-01-02 15-04"),
		truncStr(data.SessionKey, 30)))
	sessionPath := filepath.Join(v.root, "Sessions", filename+".md")
	if err := os.WriteFile(sessionPath, []byte(note), 0644); err != nil {
		return err
	}

	// Update daily note
	dateStr := data.Timestamp.Format("2006-01-02")
	v.appendDailyEntry(dateStr, data, topics)

	// Update tool notes
	for _, tool := range data.ToolsUsed {
		v.updateToolNote(tool, data.SessionKey, data.Timestamp)
	}

	// Update topic notes
	for _, topic := range topics {
		v.updateTopicNote(topic, data.SessionKey, data.Timestamp, data.UserMessage)
	}

	// Update channel note
	if data.Channel != "" {
		v.updateChannelNote(data.Channel, data.SessionKey, data.Timestamp)
	}

	// Update model note
	if data.Model != "" {
		v.updateModelNote(data.Model, data.SessionKey, data.Timestamp)
	}

	return nil
}

// WriteDreamNote creates a dream session note in the vault.
func (v *VaultWriter) WriteDreamNote(data DreamData) error {
	v.mu.Lock()
	defer v.mu.Unlock()

	if data.Timestamp.IsZero() {
		data.Timestamp = time.Now()
	}

	note := renderDreamNote(data)
	filename := sanitizeName(fmt.Sprintf("Dream %s", data.Timestamp.Format("2006-01-02 15-04")))
	dreamPath := filepath.Join(v.root, "Dreams", filename+".md")
	if err := os.WriteFile(dreamPath, []byte(note), 0644); err != nil {
		return err
	}

	// Link dream from daily note
	dateStr := data.Timestamp.Format("2006-01-02")
	dailyPath := filepath.Join(v.root, "Daily", dateStr+".md")
	appendToFile(dailyPath, fmt.Sprintf("\n- 💭 [[%s]] — %d insights, %d patterns\n",
		filename, len(data.Insights), len(data.Patterns)))

	return nil
}

// WritePersonalityChange logs a personality evolution event.
func (v *VaultWriter) WritePersonalityChange(data PersonalityData) error {
	v.mu.Lock()
	defer v.mu.Unlock()

	evolutionPath := filepath.Join(v.root, "Personality", "Personality Evolution.md")

	// Create file with frontmatter if new
	if _, err := os.Stat(evolutionPath); os.IsNotExist(err) {
		header := "---\ntags: [personality, evolution]\n---\n# Personality Evolution\n\nTimeline of how the agent's personality has adapted based on interactions.\n\n"
		os.WriteFile(evolutionPath, []byte(header), 0644)
	}

	entry := fmt.Sprintf("## %s — %s\n- **%s**: %.2f → %.2f\n- Reason: %s\n\n",
		data.Date.Format("2006-01-02"),
		data.Trait,
		data.Trait,
		data.Old, data.New,
		data.Reason)

	return appendToFile(evolutionPath, entry)
}

// appendDailyEntry adds a session entry to the daily note.
func (v *VaultWriter) appendDailyEntry(dateStr string, data SessionData, topics []string) {
	dailyPath := filepath.Join(v.root, "Daily", dateStr+".md")

	// Create daily note if it doesn't exist
	if _, err := os.Stat(dailyPath); os.IsNotExist(err) {
		header := renderDailyHeader(dateStr)
		os.WriteFile(dailyPath, []byte(header), 0644)
	}

	// Build entry with wikilinks
	toolLinks := make([]string, len(data.ToolsUsed))
	for i, t := range data.ToolsUsed {
		toolLinks[i] = "[[" + t + "]]"
	}

	topicLinks := make([]string, len(topics))
	for i, t := range topics {
		topicLinks[i] = "[[" + t + "]]"
	}

	sessionName := sanitizeName(fmt.Sprintf("Session %s %s",
		data.Timestamp.Format("2006-01-02 15-04"),
		truncStr(data.SessionKey, 30)))

	entry := fmt.Sprintf("- %s [[%s]] via [[%s]] | %s | %dms",
		data.Timestamp.Format("15:04"),
		sessionName,
		data.Channel,
		strings.Join(toolLinks, " "),
		data.LatencyMs)

	if len(topicLinks) > 0 {
		entry += " | " + strings.Join(topicLinks, " ")
	}
	entry += "\n"

	appendToFile(dailyPath, entry)
}

// updateToolNote creates or updates a tool's note with session references.
func (v *VaultWriter) updateToolNote(tool, sessionKey string, ts time.Time) {
	toolPath := filepath.Join(v.root, "Tools", sanitizeName(tool)+".md")

	if _, err := os.Stat(toolPath); os.IsNotExist(err) {
		header := renderToolHeader(tool)
		os.WriteFile(toolPath, []byte(header), 0644)
	}

	sessionName := sanitizeName(fmt.Sprintf("Session %s %s",
		ts.Format("2006-01-02 15-04"),
		truncStr(sessionKey, 30)))

	entry := fmt.Sprintf("- %s [[%s]]\n", ts.Format("2006-01-02 15:04"), sessionName)
	appendToFile(toolPath, entry)
}

// updateTopicNote creates or updates a topic note.
func (v *VaultWriter) updateTopicNote(topic, sessionKey string, ts time.Time, context string) {
	topicPath := filepath.Join(v.root, "Topics", sanitizeName(topic)+".md")

	if _, err := os.Stat(topicPath); os.IsNotExist(err) {
		header := renderTopicHeader(topic)
		os.WriteFile(topicPath, []byte(header), 0644)
	}

	sessionName := sanitizeName(fmt.Sprintf("Session %s %s",
		ts.Format("2006-01-02 15-04"),
		truncStr(sessionKey, 30)))

	snippet := truncStr(context, 100)
	entry := fmt.Sprintf("- %s [[%s]] — %s\n", ts.Format("2006-01-02"), sessionName, snippet)
	appendToFile(topicPath, entry)
}

// updateChannelNote creates or updates a channel note.
func (v *VaultWriter) updateChannelNote(channel, sessionKey string, ts time.Time) {
	channelPath := filepath.Join(v.root, "Channels", sanitizeName(channel)+".md")

	if _, err := os.Stat(channelPath); os.IsNotExist(err) {
		header := fmt.Sprintf("---\ntags: [channel]\n---\n# Channel: %s\n\n## Sessions\n\n", channel)
		os.WriteFile(channelPath, []byte(header), 0644)
	}

	sessionName := sanitizeName(fmt.Sprintf("Session %s %s",
		ts.Format("2006-01-02 15-04"),
		truncStr(sessionKey, 30)))

	entry := fmt.Sprintf("- %s [[%s]]\n", ts.Format("2006-01-02 15:04"), sessionName)
	appendToFile(channelPath, entry)
}

// updateModelNote creates or updates a model note.
func (v *VaultWriter) updateModelNote(model, sessionKey string, ts time.Time) {
	modelPath := filepath.Join(v.root, "Models", sanitizeName(model)+".md")

	if _, err := os.Stat(modelPath); os.IsNotExist(err) {
		header := fmt.Sprintf("---\ntags: [model]\n---\n# Model: %s\n\n## Sessions\n\n", model)
		os.WriteFile(modelPath, []byte(header), 0644)
	}

	sessionName := sanitizeName(fmt.Sprintf("Session %s %s",
		ts.Format("2006-01-02 15-04"),
		truncStr(sessionKey, 30)))

	entry := fmt.Sprintf("- %s [[%s]]\n", ts.Format("2006-01-02 15:04"), sessionName)
	appendToFile(modelPath, entry)
}

// --- Utility functions ---

var unsafeChars = regexp.MustCompile(`[<>:"/\\|?*\x00-\x1f]`)

// sanitizeName makes a string safe for use as an Obsidian filename.
func sanitizeName(name string) string {
	name = unsafeChars.ReplaceAllString(name, "_")
	name = strings.TrimSpace(name)
	if len(name) > 100 {
		name = name[:100]
	}
	return name
}

func truncStr(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen]
}

func appendToFile(path string, content string) error {
	dir := filepath.Dir(path)
	os.MkdirAll(dir, 0755)

	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.WriteString(content)
	return err
}

const vaultReadme = `---
tags: [index]
---
# Xagent Knowledge Vault

This vault is automatically maintained by **xagent**. Open it in [Obsidian](https://obsidian.md) and use the **Graph View** (Ctrl/Cmd+G) to explore how the agent's knowledge connects.

## Folders

| Folder | Contents |
|--------|----------|
| Sessions/ | One note per conversation turn — hub nodes linking everything |
| Daily/ | Daily aggregates — timeline spine of the graph |
| Tools/ | One note per tool — shows usage patterns |
| Topics/ | Extracted topics — shows recurring themes |
| Dreams/ | Dream mode reflections — autonomous insights |
| Personality/ | Personality evolution timeline |
| Channels/ | Communication channels (telegram, discord, etc.) |
| Models/ | LLM models used |

## Graph Tips

- **Color by folder** to see node types at a glance
- **Filter by tag** (e.g. #session, #tool, #topic) to focus
- Highly connected nodes are the most important concepts
- Orphan nodes indicate one-off interactions
`
