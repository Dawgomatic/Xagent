package vault

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestVaultInit(t *testing.T) {
	dir := t.TempDir()
	vw := NewVaultWriter(dir)

	if err := vw.Init(); err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	// Check all directories created
	expectedDirs := []string{"Sessions", "Daily", "Tools", "Topics", "Dreams", "Personality", "Channels", "Models", "World", "Experiences", "MentalModels"}
	for _, d := range expectedDirs {
		path := filepath.Join(dir, d)
		info, err := os.Stat(path)
		if err != nil {
			t.Errorf("expected dir %s to exist: %v", d, err)
			continue
		}
		if !info.IsDir() {
			t.Errorf("expected %s to be a directory", d)
		}
	}

	// Check README created
	readme := filepath.Join(dir, "README.md")
	data, err := os.ReadFile(readme)
	if err != nil {
		t.Fatal("README.md not created")
	}
	if !strings.Contains(string(data), "Xagent Knowledge Vault") {
		t.Error("README doesn't contain expected title")
	}
}

func TestWriteSessionNote(t *testing.T) {
	dir := t.TempDir()
	vw := NewVaultWriter(dir)
	vw.Init()

	data := SessionData{
		SessionKey:  "telegram:12345",
		Channel:     "telegram",
		Model:       "qwen3-4b",
		UserMessage: "How do I use docker compose with GPU support?",
		Response:    "You can use the deploy section with nvidia runtime...",
		ToolsUsed:   []string{"shell", "read_file"},
		LatencyMs:   1234,
		Iterations:  2,
		Timestamp:   time.Date(2026, 3, 8, 14, 30, 0, 0, time.UTC),
	}

	if err := vw.WriteSessionNote(data); err != nil {
		t.Fatalf("WriteSessionNote failed: %v", err)
	}

	// Check session note exists
	sessions, _ := os.ReadDir(filepath.Join(dir, "Sessions"))
	if len(sessions) == 0 {
		t.Fatal("no session notes created")
	}

	content, _ := os.ReadFile(filepath.Join(dir, "Sessions", sessions[0].Name()))
	note := string(content)

	// Verify frontmatter
	if !strings.Contains(note, "tags: [session]") {
		t.Error("missing session tag in frontmatter")
	}
	if !strings.Contains(note, "channel: telegram") {
		t.Error("missing channel in frontmatter")
	}
	if !strings.Contains(note, "model: qwen3-4b") {
		t.Error("missing model in frontmatter")
	}

	// Verify wikilinks
	if !strings.Contains(note, "[[telegram]]") {
		t.Error("missing channel wikilink")
	}
	if !strings.Contains(note, "[[qwen3-4b]]") {
		t.Error("missing model wikilink")
	}
	if !strings.Contains(note, "[[shell]]") {
		t.Error("missing tool wikilink for shell")
	}
	if !strings.Contains(note, "[[read_file]]") {
		t.Error("missing tool wikilink for read_file")
	}

	// Verify topics extracted
	if !strings.Contains(note, "[[docker]]") {
		t.Error("missing topic wikilink for docker")
	}

	// Check daily note created
	dailyPath := filepath.Join(dir, "Daily", "2026-03-08.md")
	if _, err := os.Stat(dailyPath); err != nil {
		t.Error("daily note not created")
	}

	// Check tool notes created
	shellPath := filepath.Join(dir, "Tools", "shell.md")
	if _, err := os.Stat(shellPath); err != nil {
		t.Error("shell tool note not created")
	}
	toolContent, _ := os.ReadFile(shellPath)
	if !strings.Contains(string(toolContent), "tags: [tool]") {
		t.Error("tool note missing frontmatter")
	}

	// Check topic notes created
	dockerPath := filepath.Join(dir, "Topics", "docker.md")
	if _, err := os.Stat(dockerPath); err != nil {
		t.Error("docker topic note not created")
	}

	// Check channel note created
	channelPath := filepath.Join(dir, "Channels", "telegram.md")
	if _, err := os.Stat(channelPath); err != nil {
		t.Error("telegram channel note not created")
	}

	// Check model note created
	modelPath := filepath.Join(dir, "Models", "qwen3-4b.md")
	if _, err := os.Stat(modelPath); err != nil {
		t.Error("qwen3-4b model note not created")
	}
}

func TestWriteDreamNote(t *testing.T) {
	dir := t.TempDir()
	vw := NewVaultWriter(dir)
	vw.Init()

	data := DreamData{
		Insights:  []string{"Users prefer concise docker commands", "GPU usage patterns are cyclic"},
		Patterns:  []string{"Docker questions cluster on Fridays"},
		Questions: []string{"Why does the agent avoid using kubernetes?"},
		Timestamp: time.Date(2026, 3, 8, 3, 0, 0, 0, time.UTC),
	}

	if err := vw.WriteDreamNote(data); err != nil {
		t.Fatalf("WriteDreamNote failed: %v", err)
	}

	dreams, _ := os.ReadDir(filepath.Join(dir, "Dreams"))
	if len(dreams) == 0 {
		t.Fatal("no dream notes created")
	}

	content, _ := os.ReadFile(filepath.Join(dir, "Dreams", dreams[0].Name()))
	note := string(content)

	if !strings.Contains(note, "tags: [dream, reflection]") {
		t.Error("missing dream tags")
	}
	if !strings.Contains(note, " Dream") {
		t.Error("missing dream title")
	}
	if !strings.Contains(note, "docker") {
		t.Error("missing docker topic from insight")
	}
}

func TestExtractTopics(t *testing.T) {
	tests := []struct {
		message  string
		expected []string
	}{
		{"How do I set up docker?", []string{"docker"}},
		{"Configure kubernetes cluster with GPU", []string{"kubernetes", "gpu"}},
		{"", nil},
		{"Hello world", nil}, // no tech topics
		{"Use python with pytorch for training", []string{"python", "pytorch", "training"}},
	}

	for _, tt := range tests {
		topics := ExtractTopics(tt.message)
		for _, exp := range tt.expected {
			found := false
			for _, got := range topics {
				if got == exp {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("ExtractTopics(%q): expected topic %q, got %v", tt.message, exp, topics)
			}
		}
	}
}

func TestBuildWikilinks(t *testing.T) {
	result := BuildWikilinks([]string{"docker", "kubernetes"})
	if result != "[[docker]] [[kubernetes]]" {
		t.Errorf("expected '[[docker]] [[kubernetes]]', got '%s'", result)
	}

	empty := BuildWikilinks(nil)
	if empty != "" {
		t.Errorf("expected empty string, got '%s'", empty)
	}
}

func TestSanitizeName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"simple", "simple"},
		{"has:colons", "has_colons"},
		{"has/slashes", "has_slashes"},
		{"has<angle>brackets", "has_angle_brackets"},
	}

	for _, tt := range tests {
		got := sanitizeName(tt.input)
		if got != tt.expected {
			t.Errorf("sanitizeName(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestPersonalityChange(t *testing.T) {
	dir := t.TempDir()
	vw := NewVaultWriter(dir)
	vw.Init()

	err := vw.WritePersonalityChange(PersonalityData{
		Trait:  "verbosity",
		Old:    0.5,
		New:    0.3,
		Reason: "User prefers terse responses",
		Date:   time.Now(),
	})
	if err != nil {
		t.Fatalf("WritePersonalityChange failed: %v", err)
	}

	path := filepath.Join(dir, "Personality", "Personality Evolution.md")
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal("personality note not created")
	}

	if !strings.Contains(string(content), "verbosity") {
		t.Error("personality note missing trait name")
	}
	if !strings.Contains(string(content), "0.50 → 0.30") {
		t.Error("personality note missing trait values")
	}
}
