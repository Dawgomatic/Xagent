package skills

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type SkillInstaller struct {
	workspace string
}

type AvailableSkill struct {
	Name        string   `json:"name"`
	Repository  string   `json:"repository"`
	Description string   `json:"description"`
	Author      string   `json:"author"`
	Tags        []string `json:"tags"`
}

type BuiltinSkill struct {
	Name    string `json:"name"`
	Path    string `json:"path"`
	Enabled bool   `json:"enabled"`
}

func NewSkillInstaller(workspace string) *SkillInstaller {
	return &SkillInstaller{
		workspace: workspace,
	}
}

// SWE100821: Install skills into <author>/<skill>/ layout for consistency with archive
func (si *SkillInstaller) InstallFromGitHub(ctx context.Context, repo string) error {
	parts := strings.SplitN(repo, "/", 2)
	var skillDir string
	if len(parts) == 2 {
		skillDir = filepath.Join(si.workspace, "skills", parts[0], parts[1])
	} else {
		skillDir = filepath.Join(si.workspace, "skills", filepath.Base(repo))
	}

	if _, err := os.Stat(skillDir); err == nil {
		return fmt.Errorf("skill '%s' already exists", filepath.Base(repo))
	}

	url := fmt.Sprintf("https://raw.githubusercontent.com/%s/main/SKILL.md", repo)

	client := &http.Client{Timeout: 15 * time.Second}
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to fetch skill: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("failed to fetch skill: HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	if err := os.MkdirAll(skillDir, 0755); err != nil {
		return fmt.Errorf("failed to create skill directory: %w", err)
	}

	skillPath := filepath.Join(skillDir, "SKILL.md")
	if err := os.WriteFile(skillPath, body, 0644); err != nil {
		return fmt.Errorf("failed to write skill file: %w", err)
	}

	return nil
}

// SWE100821: Uninstall accepts both flat ("skill") and qualified ("author/skill") names
func (si *SkillInstaller) Uninstall(skillName string) error {
	skillDir := filepath.Join(si.workspace, "skills", skillName)

	if _, err := os.Stat(skillDir); os.IsNotExist(err) {
		return fmt.Errorf("skill '%s' not found", skillName)
	}

	if err := os.RemoveAll(skillDir); err != nil {
		return fmt.Errorf("failed to remove skill: %w", err)
	}

	// SWE100821: Clean up empty author dir after removing last skill
	if strings.Contains(skillName, "/") {
		authorDir := filepath.Dir(skillDir)
		if entries, err := os.ReadDir(authorDir); err == nil && len(entries) == 0 {
			os.Remove(authorDir)
		}
	}

	return nil
}

func (si *SkillInstaller) ListAvailableSkills(ctx context.Context) ([]AvailableSkill, error) {
	url := "https://raw.githubusercontent.com/sipeed/xagent-skills/main/skills.json"

	client := &http.Client{Timeout: 15 * time.Second}
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch skills list: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("failed to fetch skills list: HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var skills []AvailableSkill
	if err := json.Unmarshal(body, &skills); err != nil {
		return nil, fmt.Errorf("failed to parse skills list: %w", err)
	}

	return skills, nil
}

// SWE100821: ListBuiltinSkills walks up to 2 levels deep, consistent with loader
func (si *SkillInstaller) ListBuiltinSkills() []BuiltinSkill {
	builtinSkillsDir := filepath.Join(filepath.Dir(si.workspace), "xagent", "skills")

	var skills []BuiltinSkill
	si.walkBuiltinDir(builtinSkillsDir, &skills)
	return skills
}

func (si *SkillInstaller) walkBuiltinDir(root string, skills *[]BuiltinSkill) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return
	}

	for _, entry := range entries {
		if !entry.IsDir() || strings.HasPrefix(entry.Name(), ".") {
			continue
		}
		dirPath := filepath.Join(root, entry.Name())
		skillFile := filepath.Join(dirPath, "SKILL.md")

		if _, err := os.Stat(skillFile); err == nil {
			skill := BuiltinSkill{
				Name:    entry.Name(),
				Path:    skillFile,
				Enabled: true,
			}
			*skills = append(*skills, skill)
			fmt.Printf("  ✓  %s\n", entry.Name())
		} else {
			si.walkBuiltinDir(dirPath, skills)
		}
	}
}
