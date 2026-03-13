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
	Name        string   `json:"slug"` // Mapped to raw dict 'slug' keys
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
	sb.WriteString("\nInstall with: `xagent skills install <name>`")

	return sb.String()
}

// SuggestForTopic proactively suggests skills related to a conversation topic.
func (ad *AutoDiscoverer) SuggestForTopic(topic string) []SkillCatalogEntry {
	return ad.Search(topic, 3)
}

// ForSystemPrompt returns a skills-discovery section for the system prompt,
// informing the agent that it can search for skills.
func (ad *AutoDiscoverer) ForSystemPrompt() string {
	if len(ad.catalog) == 0 {
		return ""
	}
	return fmt.Sprintf(`## Skill Discovery

You have access to a catalog of %d+ skills. If you encounter a task you cannot handle well,
suggest relevant skills to the user. Skills can be searched by topic, tool name, or error message.
The user can install skills with: xagent skills install <name>`, len(ad.catalog))
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

func (ad *AutoDiscoverer) loadInstalled() {
	entries, err := os.ReadDir(ad.installDir)
	if err != nil {
		return
	}
	for _, e := range entries {
		if e.IsDir() {
			ad.installedNames[e.Name()] = true
		}
	}
}
