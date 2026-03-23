package skills

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// SWE100821: maxSkillsInPrompt caps how many skills are listed individually in
// the system prompt. Beyond this, only a count is shown + auto-discovery hint.
// Prevents blowing context window with 11K+ skill entries.
// SWE100821: Lowered from 50 → 20 to keep system prompt under ~4K tokens on embedded devices.
// Remaining skills are still discoverable via auto-discovery and `xagent skills list`.
const maxSkillsInPrompt = 20

type SkillMetadata struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type SkillInfo struct {
	Name        string `json:"name"`
	Path        string `json:"path"`
	Source      string `json:"source"`
	Description string `json:"description"`
}

type SkillsLoader struct {
	workspace       string
	workspaceSkills string // workspace skills (project-level)
	globalSkills    string // global skills (~/.xagent/skills)
	builtinSkills   string // built-in skills
}

func NewSkillsLoader(workspace string, globalSkills string, builtinSkills string) *SkillsLoader {
	return &SkillsLoader{
		workspace:       workspace,
		workspaceSkills: filepath.Join(workspace, "skills"),
		globalSkills:    globalSkills, // ~/.xagent/skills
		builtinSkills:   builtinSkills,
	}
}

// ListSkills returns all discovered skills across workspace, global, and builtin roots.
// SWE100821: Walks up to 2 levels deep to handle both flat (<skill>/SKILL.md) and
// archive-style (<author>/<skill>/SKILL.md) directory layouts.
func (sl *SkillsLoader) ListSkills() []SkillInfo {
	seen := make(map[string]bool)
	skills := make([]SkillInfo, 0)

	sl.scanRoot(sl.workspaceSkills, "workspace", seen, &skills)
	sl.scanRoot(sl.globalSkills, "global", seen, &skills)
	sl.scanRoot(sl.builtinSkills, "builtin", seen, &skills)

	return skills
}

// scanRoot scans a skill root directory for SKILL.md files up to 2 levels deep.
// Handles both flat (root/<skill>/SKILL.md) and nested (root/<author>/<skill>/SKILL.md).
// Uses _meta.json for fast metadata when available to avoid reading every SKILL.md.
func (sl *SkillsLoader) scanRoot(root, source string, seen map[string]bool, skills *[]SkillInfo) {
	if root == "" {
		return
	}
	dirs, err := os.ReadDir(root)
	if err != nil {
		return
	}

	for _, d := range dirs {
		if !d.IsDir() || strings.HasPrefix(d.Name(), ".") {
			continue
		}
		dirPath := filepath.Join(root, d.Name())

		// Level 1: check for SKILL.md directly (flat layout)
		skillFile := filepath.Join(dirPath, "SKILL.md")
		if _, err := os.Stat(skillFile); err == nil {
			sl.addSkill(d.Name(), skillFile, source, seen, skills)
			continue
		}

		// Level 2: walk subdirectories (archive <author>/<skill> layout)
		subDirs, err := os.ReadDir(dirPath)
		if err != nil {
			continue
		}
		for _, sd := range subDirs {
			if !sd.IsDir() || strings.HasPrefix(sd.Name(), ".") {
				continue
			}
			subSkillFile := filepath.Join(dirPath, sd.Name(), "SKILL.md")
			if _, err := os.Stat(subSkillFile); err == nil {
				// SWE100821: Use author/skill as the canonical name so skills
				// from different authors with the same slug don't collide.
				canonicalName := d.Name() + "/" + sd.Name()
				sl.addSkill(canonicalName, subSkillFile, source, seen, skills)
			}
		}
	}
}

// addSkill appends a skill if not already seen (workspace > global > builtin priority).
func (sl *SkillsLoader) addSkill(name, skillFile, source string, seen map[string]bool, skills *[]SkillInfo) {
	if seen[name] {
		return
	}
	seen[name] = true

	info := SkillInfo{
		Name:   name,
		Path:   skillFile,
		Source: source,
	}

	// SWE100821: Try _meta.json first (pre-parsed, fast) before reading full SKILL.md
	metaPath := filepath.Join(filepath.Dir(skillFile), "_meta.json")
	if metaData, err := os.ReadFile(metaPath); err == nil {
		var meta struct {
			Slug        string `json:"slug"`
			Description string `json:"description"`
		}
		if json.Unmarshal(metaData, &meta) == nil && meta.Description != "" {
			info.Description = meta.Description
			*skills = append(*skills, info)
			return
		}
	}

	metadata := sl.getSkillMetadata(skillFile)
	if metadata != nil {
		info.Description = metadata.Description
	}
	*skills = append(*skills, info)
}

// LoadSkill loads a skill's content by name. Accepts both flat names ("my-skill")
// and author-qualified names ("author/skill-name").
func (sl *SkillsLoader) LoadSkill(name string) (string, bool) {
	// SWE100821: Reject path traversal in skill names — LLM could request "../../../etc/passwd"
	if strings.Contains(name, "..") || filepath.IsAbs(name) {
		return "", false
	}

	for _, root := range []string{sl.workspaceSkills, sl.globalSkills, sl.builtinSkills} {
		if root == "" {
			continue
		}
		skillFile := filepath.Clean(filepath.Join(root, name, "SKILL.md"))
		absRoot := filepath.Clean(root)
		if !strings.HasPrefix(skillFile, absRoot+string(filepath.Separator)) {
			continue
		}
		if content, err := os.ReadFile(skillFile); err == nil {
			return sl.stripFrontmatter(string(content)), true
		}
	}

	// Fallback: if name has no slash, search 2 levels deep by slug match
	if !strings.Contains(name, "/") {
		for _, root := range []string{sl.workspaceSkills, sl.globalSkills, sl.builtinSkills} {
			if root == "" {
				continue
			}
			if found, content := sl.findSkillBySlug(root, name); found {
				return content, true
			}
		}
	}

	return "", false
}

// findSkillBySlug searches 2 levels deep for a skill matching the given slug name.
func (sl *SkillsLoader) findSkillBySlug(root, slug string) (bool, string) {
	dirs, err := os.ReadDir(root)
	if err != nil {
		return false, ""
	}
	for _, d := range dirs {
		if !d.IsDir() {
			continue
		}
		skillFile := filepath.Join(root, d.Name(), slug, "SKILL.md")
		if content, err := os.ReadFile(skillFile); err == nil {
			return true, sl.stripFrontmatter(string(content))
		}
	}
	return false, ""
}

func (sl *SkillsLoader) LoadSkillsForContext(skillNames []string) string {
	if len(skillNames) == 0 {
		return ""
	}

	var parts []string
	for _, name := range skillNames {
		content, ok := sl.LoadSkill(name)
		if ok {
			parts = append(parts, fmt.Sprintf("### Skill: %s\n\n%s", name, content))
		}
	}

	return strings.Join(parts, "\n\n---\n\n")
}

// BuildSkillsSummary returns a compact skill summary for the system prompt.
// SWE100821: Capped to maxSkillsInPrompt to prevent context explosion with 11K+ skills.
// Beyond the cap, shows count + auto-discovery hint instead of individual listings.
func (sl *SkillsLoader) BuildSkillsSummary() string {
	allSkills := sl.ListSkills()
	if len(allSkills) == 0 {
		return ""
	}

	var lines []string
	lines = append(lines, "<skills>")

	displayCount := len(allSkills)
	if displayCount > maxSkillsInPrompt {
		displayCount = maxSkillsInPrompt
	}

	for i := 0; i < displayCount; i++ {
		s := allSkills[i]
		escapedName := escapeXML(s.Name)
		escapedDesc := escapeXML(s.Description)
		escapedPath := escapeXML(s.Path)

		lines = append(lines, "  <skill>")
		lines = append(lines, fmt.Sprintf("    <name>%s</name>", escapedName))
		lines = append(lines, fmt.Sprintf("    <description>%s</description>", escapedDesc))
		lines = append(lines, fmt.Sprintf("    <location>%s</location>", escapedPath))
		lines = append(lines, fmt.Sprintf("    <source>%s</source>", s.Source))
		lines = append(lines, "  </skill>")
	}

	// SWE100821: Show overflow count so the agent knows more exist
	if len(allSkills) > maxSkillsInPrompt {
		lines = append(lines, fmt.Sprintf("  <!-- %d more skills available. Use skill auto-discovery or 'xagent skills list' to find them. -->",
			len(allSkills)-maxSkillsInPrompt))
	}

	lines = append(lines, "</skills>")

	return strings.Join(lines, "\n")
}

func (sl *SkillsLoader) getSkillMetadata(skillPath string) *SkillMetadata {
	content, err := os.ReadFile(skillPath)
	if err != nil {
		return nil
	}

	frontmatter := sl.extractFrontmatter(string(content))
	if frontmatter == "" {
		return &SkillMetadata{
			Name: filepath.Base(filepath.Dir(skillPath)),
		}
	}

	// Try JSON first (for backward compatibility)
	var jsonMeta struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	if err := json.Unmarshal([]byte(frontmatter), &jsonMeta); err == nil {
		return &SkillMetadata{
			Name:        jsonMeta.Name,
			Description: jsonMeta.Description,
		}
	}

	// Fall back to simple YAML parsing
	yamlMeta := sl.parseSimpleYAML(frontmatter)
	return &SkillMetadata{
		Name:        yamlMeta["name"],
		Description: yamlMeta["description"],
	}
}

// parseSimpleYAML parses simple key: value YAML format
// Example: name: github\n description: "..."
func (sl *SkillsLoader) parseSimpleYAML(content string) map[string]string {
	result := make(map[string]string)

	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, ":", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			// Remove quotes if present
			value = strings.Trim(value, "\"'")
			result[key] = value
		}
	}

	return result
}

func (sl *SkillsLoader) extractFrontmatter(content string) string {
	// (?s) enables DOTALL mode so . matches newlines
	// Match first ---, capture everything until next --- on its own line
	re := regexp.MustCompile(`(?s)^---\n(.*)\n---`)
	match := re.FindStringSubmatch(content)
	if len(match) > 1 {
		return match[1]
	}
	return ""
}

func (sl *SkillsLoader) stripFrontmatter(content string) string {
	re := regexp.MustCompile(`^---\n.*?\n---\n`)
	return re.ReplaceAllString(content, "")
}

// GetRawMetadata returns the parsed YAML frontmatter of a skill as a map.
// SWE100821: Used by dynamic tool registration to extract tool definitions.
func (sl *SkillsLoader) GetRawMetadata(skillPath string) map[string]interface{} {
	content, err := os.ReadFile(skillPath)
	if err != nil {
		return nil
	}

	fm := sl.extractFrontmatter(string(content))
	if fm == "" {
		return nil
	}

	// Parse YAML-like frontmatter into a map
	result := make(map[string]interface{})
	yamlMap := sl.parseSimpleYAML(fm)
	for k, v := range yamlMap {
		result[k] = v
	}
	return result
}

func escapeXML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	return s
}
