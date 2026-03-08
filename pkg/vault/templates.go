// Xagent - Ultra-lightweight personal AI agent
// Obsidian Knowledge Vault — note templates with YAML frontmatter
//
// Copyright (c) 2026 Xagent contributors
// License: MIT

package vault

import (
	"fmt"
	"strings"
)

// renderSessionNote creates a session note with YAML frontmatter and wikilinks.
func renderSessionNote(data SessionData, topics []string) string {
	var sb strings.Builder

	// --- YAML frontmatter ---
	sb.WriteString("---\n")
	sb.WriteString("tags: [session]\n")
	sb.WriteString(fmt.Sprintf("date: %s\n", data.Timestamp.Format("2006-01-02")))
	sb.WriteString(fmt.Sprintf("time: %s\n", data.Timestamp.Format("15:04:05")))
	sb.WriteString(fmt.Sprintf("channel: %s\n", data.Channel))
	sb.WriteString(fmt.Sprintf("session_key: %s\n", data.SessionKey))
	sb.WriteString(fmt.Sprintf("model: %s\n", data.Model))
	if len(data.ToolsUsed) > 0 {
		sb.WriteString(fmt.Sprintf("tools_used: [%s]\n", strings.Join(data.ToolsUsed, ", ")))
	}
	if len(topics) > 0 {
		sb.WriteString(fmt.Sprintf("topics: [%s]\n", strings.Join(topics, ", ")))
	}
	sb.WriteString(fmt.Sprintf("latency_ms: %d\n", data.LatencyMs))
	sb.WriteString(fmt.Sprintf("iterations: %d\n", data.Iterations))
	sb.WriteString("---\n\n")

	// --- Title ---
	sb.WriteString(fmt.Sprintf("# Session — %s\n\n", data.Timestamp.Format("2006-01-02 15:04")))

	// --- Context wikilinks ---
	dateStr := data.Timestamp.Format("2006-01-02")
	sb.WriteString(fmt.Sprintf("📅 [[%s]] · ", dateStr))
	if data.Channel != "" {
		sb.WriteString(fmt.Sprintf("📡 [[%s]] · ", data.Channel))
	}
	if data.Model != "" {
		sb.WriteString(fmt.Sprintf("🤖 [[%s]]", data.Model))
	}
	sb.WriteString("\n\n")

	// --- Tools ---
	if len(data.ToolsUsed) > 0 {
		sb.WriteString("## Tools Used\n\n")
		for _, tool := range data.ToolsUsed {
			sb.WriteString(fmt.Sprintf("- [[%s]]\n", tool))
		}
		sb.WriteString("\n")
	}

	// --- Skills ---
	if len(data.SkillsUsed) > 0 {
		sb.WriteString("## Skills\n\n")
		for _, skill := range data.SkillsUsed {
			sb.WriteString(fmt.Sprintf("- [[%s]]\n", skill))
		}
		sb.WriteString("\n")
	}

	// --- Topics ---
	if len(topics) > 0 {
		sb.WriteString("## Topics\n\n")
		for _, topic := range topics {
			sb.WriteString(fmt.Sprintf("- [[%s]]\n", topic))
		}
		sb.WriteString("\n")
	}

	// --- Conversation excerpt ---
	sb.WriteString("## Conversation\n\n")
	if data.UserMessage != "" {
		sb.WriteString("**User:**\n")
		sb.WriteString("> " + strings.ReplaceAll(truncStr(data.UserMessage, 300), "\n", "\n> "))
		sb.WriteString("\n\n")
	}
	if data.Response != "" {
		sb.WriteString("**Agent:**\n")
		sb.WriteString("> " + strings.ReplaceAll(truncStr(data.Response, 500), "\n", "\n> "))
		sb.WriteString("\n\n")
	}

	// --- Stats ---
	sb.WriteString("## Stats\n\n")
	sb.WriteString(fmt.Sprintf("| Metric | Value |\n"))
	sb.WriteString(fmt.Sprintf("|--------|-------|\n"))
	sb.WriteString(fmt.Sprintf("| Latency | %dms |\n", data.LatencyMs))
	sb.WriteString(fmt.Sprintf("| Iterations | %d |\n", data.Iterations))
	if data.PlanSteps > 0 {
		sb.WriteString(fmt.Sprintf("| Plan Steps | %d |\n", data.PlanSteps))
	}
	if len(data.MemoryHits) > 0 {
		sb.WriteString(fmt.Sprintf("| Memory Hits | %d |\n", len(data.MemoryHits)))
	}

	return sb.String()
}

// renderDreamNote creates a dream session note.
func renderDreamNote(data DreamData) string {
	var sb strings.Builder

	sb.WriteString("---\n")
	sb.WriteString("tags: [dream, reflection]\n")
	sb.WriteString(fmt.Sprintf("date: %s\n", data.Timestamp.Format("2006-01-02")))
	sb.WriteString(fmt.Sprintf("insights: %d\n", len(data.Insights)))
	sb.WriteString(fmt.Sprintf("patterns: %d\n", len(data.Patterns)))
	sb.WriteString("---\n\n")

	sb.WriteString(fmt.Sprintf("# 💭 Dream — %s\n\n", data.Timestamp.Format("2006-01-02 15:04")))

	dateStr := data.Timestamp.Format("2006-01-02")
	sb.WriteString(fmt.Sprintf("📅 [[%s]]\n\n", dateStr))

	if len(data.Patterns) > 0 {
		sb.WriteString("## Patterns Noticed\n\n")
		for _, p := range data.Patterns {
			sb.WriteString(fmt.Sprintf("- %s\n", p))
			// Auto-link topics within pattern text
			topics := ExtractTopics(p)
			if len(topics) > 0 {
				sb.WriteString(fmt.Sprintf("  → %s\n", BuildWikilinks(topics)))
			}
		}
		sb.WriteString("\n")
	}

	if len(data.Insights) > 0 {
		sb.WriteString("## Insights\n\n")
		for _, insight := range data.Insights {
			sb.WriteString(fmt.Sprintf("- %s\n", insight))
			topics := ExtractTopics(insight)
			if len(topics) > 0 {
				sb.WriteString(fmt.Sprintf("  → %s\n", BuildWikilinks(topics)))
			}
		}
		sb.WriteString("\n")
	}

	if len(data.Questions) > 0 {
		sb.WriteString("## Open Questions\n\n")
		for _, q := range data.Questions {
			sb.WriteString(fmt.Sprintf("- %s\n", q))
		}
	}

	return sb.String()
}

// renderDailyHeader creates the header for a new daily note.
func renderDailyHeader(dateStr string) string {
	return fmt.Sprintf(`---
tags: [daily]
date: %s
---
# %s

## Sessions

`, dateStr, dateStr)
}

// renderToolHeader creates the header for a new tool note.
func renderToolHeader(tool string) string {
	return fmt.Sprintf(`---
tags: [tool]
tool_name: %s
---
# Tool: %s

## Usage History

`, tool, tool)
}

// renderTopicHeader creates the header for a new topic note.
func renderTopicHeader(topic string) string {
	return fmt.Sprintf(`---
tags: [topic]
topic: %s
---
# %s

## Mentions

`, topic, strings.Title(topic))
}
