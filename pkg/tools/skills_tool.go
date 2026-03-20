// SWE100821: Skills tool — lets the agent search the 10K+ embedded skill catalog
// and install skills at runtime. Without this, the agent knows skills exist but
// has no way to look them up or install them.
package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/Dawgomatic/Xagent/pkg/skills"
)

// SkillsTool exposes skill catalog search and install to the agent.
type SkillsTool struct {
	discoverer *skills.AutoDiscoverer
	installer  *skills.SkillInstaller
	loader     *skills.SkillsLoader
}

// NewSkillsTool creates a skills tool with catalog search and install capabilities.
func NewSkillsTool(discoverer *skills.AutoDiscoverer, installer *skills.SkillInstaller, loader *skills.SkillsLoader) *SkillsTool {
	return &SkillsTool{
		discoverer: discoverer,
		installer:  installer,
		loader:     loader,
	}
}

func (t *SkillsTool) Name() string { return "skills" }

func (t *SkillsTool) Description() string {
	return "Search the skill catalog (10,000+ skills) and install new skills. Actions: search (find skills by keyword/topic), install (install a skill by owner/name), list (show installed skills)."
}

func (t *SkillsTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"action": map[string]interface{}{
				"type":        "string",
				"enum":        []string{"search", "install", "list"},
				"description": "Action: search (find skills by query), install (install a skill), list (show installed)",
			},
			"query": map[string]interface{}{
				"type":        "string",
				"description": "Search query — topic, keyword, or task description. Required for search.",
			},
			"name": map[string]interface{}{
				"type":        "string",
				"description": "Skill name to install (format: 'owner/slug', e.g. '0xbeekeeper/security'). Required for install.",
			},
		},
		"required": []string{"action"},
	}
}

func (t *SkillsTool) Execute(ctx context.Context, args map[string]interface{}) *ToolResult {
	action, _ := args["action"].(string)

	switch action {
	case "search":
		return t.search(args)
	case "install":
		return t.install(ctx, args)
	case "list":
		return t.list()
	default:
		return ErrorResult(fmt.Sprintf("Unknown action: %s. Use search, install, or list.", action))
	}
}

func (t *SkillsTool) search(args map[string]interface{}) *ToolResult {
	query, _ := args["query"].(string)
	if query == "" {
		return ErrorResult("query parameter required for search action")
	}

	if t.discoverer == nil {
		return ErrorResult("Skill catalog not available")
	}

	results := t.discoverer.Search(query, 10)
	if len(results) == 0 {
		return NewToolResult(fmt.Sprintf("No skills found matching '%s'.", query))
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Found %d skills matching '%s':\n\n", len(results), query))
	for i, r := range results {
		installName := r.Name
		if r.Owner != "" {
			installName = r.Owner + "/" + r.Name
		}
		sb.WriteString(fmt.Sprintf("%d. **%s** (%s)\n", i+1, r.Name, installName))
		if r.Description != "" {
			sb.WriteString(fmt.Sprintf("   %s\n", r.Description))
		}
		if len(r.Tags) > 0 {
			sb.WriteString(fmt.Sprintf("   Tags: %s\n", strings.Join(r.Tags, ", ")))
		}
		sb.WriteString("\n")
	}
	sb.WriteString("Install with: skills({action: 'install', name: 'owner/slug'})")

	return NewToolResult(sb.String())
}

func (t *SkillsTool) install(ctx context.Context, args map[string]interface{}) *ToolResult {
	name, _ := args["name"].(string)
	if name == "" {
		return ErrorResult("name parameter required for install (format: 'owner/slug')")
	}

	if t.installer == nil {
		return ErrorResult("Skill installer not available")
	}

	if err := t.installer.InstallFromGitHub(ctx, name); err != nil {
		return ErrorResult(fmt.Sprintf("Failed to install skill '%s': %v", name, err))
	}

	return NewToolResult(fmt.Sprintf("Skill '%s' installed successfully. It is now available for use.", name))
}

func (t *SkillsTool) list() *ToolResult {
	if t.loader == nil {
		return ErrorResult("Skills loader not available")
	}

	installed := t.loader.ListSkills()
	if len(installed) == 0 {
		return NewToolResult("No skills installed.")
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("%d skills installed:\n\n", len(installed)))
	for _, s := range installed {
		line := fmt.Sprintf("- %s", s.Name)
		if s.Description != "" {
			desc := s.Description
			if len(desc) > 80 {
				desc = desc[:80] + "..."
			}
			line += fmt.Sprintf(": %s", desc)
		}
		line += fmt.Sprintf(" [%s]\n", s.Source)
		sb.WriteString(line)
	}

	return NewToolResult(sb.String())
}
