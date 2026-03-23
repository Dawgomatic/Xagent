// SWE100821: Skill auto-discovery — when the agent encounters a task it can't
// handle well, search the 10,000+ skill archive for relevant skills and suggest
// installation. Turns failures into learning opportunities.

package skills

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/Dawgomatic/Xagent/pkg/logger"
)

//go:embed catalog.json
var embeddedCatalogData []byte

// SkillCatalogEntry represents a skill in the searchable archive.
type SkillCatalogEntry struct {
	Owner       string   `json:"owner"`
	Name        string   `json:"slug"`
	Description string   `json:"description"`
	Tags        []string `json:"tags"`
	Category    string   `json:"category"`
	Source      string   `json:"source"`
	Path        string   `json:"path"`
	Score       float64  `json:"-"` // search relevance score
}

// AutoDiscoverer searches the skill archive for relevant skills.
type AutoDiscoverer struct {
	installDir     string // where to install discovered skills
	catalog        []SkillCatalogEntry
	installedNames map[string]bool
}

// NewAutoDiscoverer creates a skill auto-discoverer.
// Now natively baked in without manual json loading or reference folders.
func NewAutoDiscoverer(workspace string) *AutoDiscoverer {
	ad := &AutoDiscoverer{
		installDir:     filepath.Join(workspace, "skills"),
		installedNames: make(map[string]bool),
	}
	ad.loadCatalog()
	ad.loadInstalled()
	return ad
}

// Search finds skills relevant to the given query/task description.
// Returns up to maxResults skills, sorted by relevance.
func (ad *AutoDiscoverer) Search(query string, maxResults int) []SkillCatalogEntry {
	if len(ad.catalog) == 0 {
		return nil
	}

	queryLower := strings.ToLower(query)
	queryWords := strings.Fields(queryLower)

	var scored []SkillCatalogEntry
	for _, skill := range ad.catalog {
		// Skip already-installed skills
		if ad.installedNames[skill.Name] {
			continue
		}

		score := ad.relevanceScore(skill, queryWords, queryLower)
		if score > 0 {
			entry := skill
			entry.Score = score
			scored = append(scored, entry)
		}
	}

	sort.Slice(scored, func(i, j int) bool {
		return scored[i].Score > scored[j].Score
	})

	if len(scored) > maxResults {
		scored = scored[:maxResults]
	}

	return scored
}

// SuggestForError searches for skills that might help with a tool error.
// Returns a formatted suggestion string or empty if none found.
func (ad *AutoDiscoverer) SuggestForError(toolName, errorMsg string) string {
	query := fmt.Sprintf("%s %s", toolName, errorMsg)
	results := ad.Search(query, 3)
	if len(results) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString(" These skills might help:\n")
	for _, r := range results {
		sb.WriteString(fmt.Sprintf("- **%s**: %s (tags: %s)\n",
			r.Name, r.Description, strings.Join(r.Tags, ", ")))
	}
	// SWE100821: Direct agent to use the skills tool, not the CLI
	sb.WriteString("\nInstall with: skills(action=\"install\", name=\"owner/slug\")")

	return sb.String()
}

// SuggestForTopic proactively suggests skills related to a conversation topic.
func (ad *AutoDiscoverer) SuggestForTopic(topic string) []SkillCatalogEntry {
	return ad.Search(topic, 3)
}

// SWE100821: Updated prompt to instruct agent to actively USE the skills tool, not just suggest.
func (ad *AutoDiscoverer) ForSystemPrompt() string {
	if len(ad.catalog) == 0 {
		return ""
	}

	installed := len(ad.installedNames)

	return fmt.Sprintf(`## Skills — Your Extensible Capabilities

You have a **skills** tool with access to %d+ installable skills (%d currently installed).

**USE the skills tool proactively:**
- When you encounter an unfamiliar task, SEARCH for relevant skills: skills(action="search", query="<topic>")
- When a search finds something useful, INSTALL it: skills(action="install", name="owner/slug")
- Check what you already have: skills(action="list")
- After a tool error, search for skills that handle that domain
- When the user asks about capabilities you lack, search for a skill first before saying you cannot

**When to search skills:**
- Hardware interaction (sensors, GPIO, camera, audio)
- Specialized domains (security, networking, databases, ML)
- Automation tasks (monitoring, scheduling, data pipelines)
- Any task where your built-in tools feel insufficient

Do NOT tell the user to install skills manually — you can do it yourself with the skills tool.`, len(ad.catalog), installed)
}

func (ad *AutoDiscoverer) relevanceScore(skill SkillCatalogEntry, queryWords []string, queryLower string) float64 {
	score := 0.0
	nameLower := strings.ToLower(skill.Name)
	descLower := strings.ToLower(skill.Description)

	// Exact name match
	if strings.Contains(queryLower, nameLower) {
		score += 5.0
	}

	// Word matches in name, description, tags
	for _, word := range queryWords {
		if len(word) < 3 {
			continue
		}
		if strings.Contains(nameLower, word) {
			score += 2.0
		}
		if strings.Contains(descLower, word) {
			score += 1.0
		}
		for _, tag := range skill.Tags {
			if strings.Contains(strings.ToLower(tag), word) {
				score += 1.5
			}
		}
	}

	return score
}

func (ad *AutoDiscoverer) loadCatalog() {
	if err := json.Unmarshal(embeddedCatalogData, &ad.catalog); err != nil {
		logger.WarnCF("skills", "Failed to parse embedded skill catalog",
			map[string]interface{}{"error": err.Error()})
		return
	}

	logger.InfoCF("skills", "Embedded native skill catalog loaded",
		map[string]interface{}{"skills": len(ad.catalog)})
}

// SWE100821: Walk 2 levels deep to match archive <author>/<skill> layout
func (ad *AutoDiscoverer) loadInstalled() {
	entries, err := os.ReadDir(ad.installDir)
	if err != nil {
		return
	}
	for _, e := range entries {
		if !e.IsDir() || strings.HasPrefix(e.Name(), ".") {
			continue
		}
		dirPath := filepath.Join(ad.installDir, e.Name())

		// Level 1: flat skill
		if _, err := os.Stat(filepath.Join(dirPath, "SKILL.md")); err == nil {
			ad.installedNames[e.Name()] = true
			continue
		}

		// Level 2: author/skill
		subEntries, err := os.ReadDir(dirPath)
		if err != nil {
			continue
		}
		for _, se := range subEntries {
			if se.IsDir() {
				if _, err := os.Stat(filepath.Join(dirPath, se.Name(), "SKILL.md")); err == nil {
					ad.installedNames[se.Name()] = true
				}
			}
		}
	}
}
