package agent

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/Dawgomatic/Xagent/pkg/config"
	"github.com/Dawgomatic/Xagent/pkg/epoch"
	"github.com/Dawgomatic/Xagent/pkg/identity"
	"github.com/Dawgomatic/Xagent/pkg/logger"
	"github.com/Dawgomatic/Xagent/pkg/providers"
	"github.com/Dawgomatic/Xagent/pkg/skills"
	"github.com/Dawgomatic/Xagent/pkg/tools"
)

type ContextBuilder struct {
	workspace      string
	skillsLoader   *skills.SkillsLoader
	memory         *MemoryStore
	tools          *tools.ToolRegistry     // Direct reference to tool registry
	identity       *identity.AgentIdentity // SWE100821: Agent identity + time tracking
	prevEpoch      *epoch.Record           // SWE100821: Previous epoch for wake-up recall
	autoDiscoverer *skills.AutoDiscoverer  // SWE100821: Skill auto-discovery
	bootstrapCache map[string]string       // Cache for AGENTS.md, SOUL.md, etc.
	bootstrapMTime map[string]time.Time    // MTime for cache invalidation
	compactPrompt  bool                    // SWE100821: Minimal system prompt for embedded/PicoLM
}
// SWE100821: Removed duplicate semanticMemory field — only AgentLoop's instance
// is used for search/store. This one was never read, just wasting a Qdrant probe.

func getGlobalConfigDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".xagent")
}

func NewContextBuilder(workspace string, _ ...config.SemanticMemoryConfig) *ContextBuilder {
	// builtin skills: skills directory in current project
	wd, _ := os.Getwd()
	builtinSkillsDir := filepath.Join(wd, "skills")
	// SWE100821: Handle cloned skill archives where structure is skills/skills/<author>/<skill>/
	// If skills/skills/ exists and contains dirs, use the inner dir as the real root so the
	// 2-level scanner sees <author>/<skill>/SKILL.md directly.
	innerSkills := filepath.Join(builtinSkillsDir, "skills")
	if info, err := os.Stat(innerSkills); err == nil && info.IsDir() {
		builtinSkillsDir = innerSkills
	}
	globalSkillsDir := filepath.Join(getGlobalConfigDir(), "skills")

	// SWE100821: Initialize skill auto-discoverer
	autoDisc := skills.NewAutoDiscoverer(workspace)

	return &ContextBuilder{
		workspace:      workspace,
		skillsLoader:   skills.NewSkillsLoader(workspace, globalSkillsDir, builtinSkillsDir),
		memory:         NewMemoryStore(workspace),
		autoDiscoverer: autoDisc,
		bootstrapCache: make(map[string]string),
		bootstrapMTime: make(map[string]time.Time),
	}
}

// SetToolsRegistry sets the tools registry for dynamic tool summary generation.
func (cb *ContextBuilder) SetToolsRegistry(registry *tools.ToolRegistry) {
	cb.tools = registry
}

// SetIdentity sets the agent identity for system prompt injection.
// SWE100821: Unique agent identity + time tracking in system prompt.
func (cb *ContextBuilder) SetIdentity(id *identity.AgentIdentity) {
	cb.identity = id
}

// SetPreviousEpoch stores the last epoch for injection into the system prompt.
// SWE100821: Wake-up recall — the agent remembers what happened last session.
func (cb *ContextBuilder) SetPreviousEpoch(rec *epoch.Record) {
	cb.prevEpoch = rec
}

func (cb *ContextBuilder) getIdentity() string {
	now := time.Now().Format("2006-01-02 15:04 (Monday)")
	workspacePath, _ := filepath.Abs(filepath.Join(cb.workspace))
	runtimeStr := fmt.Sprintf("%s %s, Go %s", runtime.GOOS, runtime.GOARCH, runtime.Version())

	// SWE100821: Inject agent identity + time tracking into system prompt
	identitySection := ""
	if cb.identity != nil {
		identitySection = fmt.Sprintf(`## Agent Identity
%s
`, cb.identity.ForSystemPrompt())
	}

	// SWE100821: Tool summaries removed from system prompt — native tool definitions
	// (via API tool_calls schema) are sufficient and less likely to diverge.
	// Phone workflow removed — phone tool description already covers valid actions.
	return fmt.Sprintf(`# Agent

You are an autonomous AI agent.

## Current Time
%s

%s## Runtime
%s

## Workspace
Your workspace is at: %s
- Memory: %s/memory/MEMORY.md
- Daily Notes: %s/memory/YYYYMM/YYYYMMDD.md
- Skills: %s/skills/{skill-name}/SKILL.md

## Rules

1. **Use tools when needed** — call the appropriate tool. Do NOT pretend to perform actions.
2. **Never give up** — if a tool fails, try a different approach. Keep trying until the task is complete.
3. **Respond with text after completing** — summarize what you did. Do NOT keep calling tools after the task is done.
4. **Plain text responses** — never wrap your final response in JSON or XML.
5. **Memory** — write important information to %s/memory/MEMORY.md`,
		now, identitySection, runtimeStr, workspacePath, workspacePath, workspacePath, workspacePath, workspacePath)
}

// SWE100821: buildToolsSection removed — tool definitions are sent via native API
// tool schema (ToolToSchema in base.go), not duplicated as text in system prompt.

// SWE100821: Enable compact prompt mode — strips system prompt to <200 chars
// for PicoLM/embedded devices where prefill cost dominates latency.
func (cb *ContextBuilder) SetCompactPrompt(enabled bool) {
	cb.compactPrompt = enabled
}

func (cb *ContextBuilder) BuildSystemPrompt() string {
	// SWE100821: Compact mode — minimal prompt for embedded inference (PicoLM/TinyLlama).
	// Full system prompt (~3000 chars / ~750 tokens) causes 3+ min prefill on ARM.
	// This keeps it under ~200 chars / ~50 tokens for sub-15s responses.
	if cb.compactPrompt {
		return "You are xagent, a concise AI assistant. Answer directly. Be brief."
	}

	parts := []string{}

	// Core identity section
	parts = append(parts, cb.getIdentity())

	// Bootstrap files
	bootstrapContent := cb.LoadBootstrapFiles()
	if bootstrapContent != "" {
		parts = append(parts, bootstrapContent)
	}

	// Skills - show summary, AI can read full content with read_file tool
	skillsSummary := cb.skillsLoader.BuildSkillsSummary()
	if skillsSummary != "" {
		parts = append(parts, fmt.Sprintf(`# Skills

The following skills extend your capabilities. To use a skill, read its SKILL.md file using the read_file tool.

%s`, skillsSummary))
	}

	// SWE100821: Previous epoch recall — "what happened last session"
	epochContext := epoch.ForSystemPrompt(cb.prevEpoch)
	if epochContext != "" {
		parts = append(parts, "# Previous Session\n\n"+epochContext)
	}

	// SWE100821: GetMemoryContext() already includes its own "# Memory" header
	memoryContext := cb.memory.GetMemoryContext()
	if memoryContext != "" {
		parts = append(parts, memoryContext)
	}

	// SWE100821: Skill auto-discovery prompt
	if cb.autoDiscoverer != nil {
		discoverPrompt := cb.autoDiscoverer.ForSystemPrompt()
		if discoverPrompt != "" {
			parts = append(parts, discoverPrompt)
		}
	}

	// Join with "---" separator
	return strings.Join(parts, "\n\n---\n\n")
}

func (cb *ContextBuilder) LoadBootstrapFiles() string {
	bootstrapFiles := []string{
		"AGENTS.md",
		"SOUL.md",
		"USER.md",
		"IDENTITY.md",
	}

	var sb strings.Builder
	for _, filename := range bootstrapFiles {
		filePath := filepath.Join(cb.workspace, filename)

		info, err := os.Stat(filePath)
		if err != nil {
			continue // File doesn't exist
		}

		// Check cache
		mtime := info.ModTime()
		cachedMTime, cached := cb.bootstrapMTime[filename]
		var content string

		if cached && mtime.Equal(cachedMTime) {
			content = cb.bootstrapCache[filename]
		} else {
			if data, err := os.ReadFile(filePath); err == nil {
				content = string(data)
				cb.bootstrapCache[filename] = content
				cb.bootstrapMTime[filename] = mtime
			}
		}

		if content != "" {
			sb.WriteString(fmt.Sprintf("## %s\n\n%s\n\n", filename, content))
		}
	}

	return sb.String()
}

// BuildMessages constructs the message array for LLM calls.
// SWE100821: Stable system prompt — the system message is STATIC (identity, bootstrap,
// skills, memory, epoch) so Anthropic/OpenRouter can cache it across calls.
// All volatile per-turn data (semantic recall, channel, summary, plan, personality,
// tool hints, feedback) is injected via a separate "user" context preamble so it
// does NOT invalidate the cached prefix. See Nanobot agent/context.py L83-90.
func (cb *ContextBuilder) BuildMessages(history []providers.Message, summary string, currentMessage string, media []string, channel, chatID string, volatileContext ...string) []providers.Message {
	messages := []providers.Message{}

	systemPrompt := cb.BuildSystemPrompt()

	logger.DebugCF("agent", "System prompt built",
		map[string]interface{}{
			"total_chars":   len(systemPrompt),
			"total_lines":   strings.Count(systemPrompt, "\n") + 1,
			"section_count": strings.Count(systemPrompt, "\n\n---\n\n") + 1,
		})

	if logger.GetLevel() <= logger.DEBUG {
		preview := systemPrompt
		if len(preview) > 500 {
			preview = preview[:500] + "... (truncated)"
		}
		logger.DebugCF("agent", "System prompt preview",
			map[string]interface{}{"preview": preview})
	}

	// Diegox-17: prevent orphaned tool messages from breaking LLM
	for len(history) > 0 && (history[0].Role == "tool") {
		logger.DebugCF("agent", "Removing orphaned tool message from history to prevent LLM error",
			map[string]interface{}{"role": history[0].Role})
		history = history[1:]
	}

	messages = append(messages, providers.Message{
		Role:    "system",
		Content: systemPrompt,
	})

	// SWE100821: Build volatile context preamble — kept OUT of system prompt for cache stability.
	var volatile strings.Builder
	if summary != "" {
		volatile.WriteString("## Summary of Previous Conversation\n\n")
		volatile.WriteString(summary)
		volatile.WriteString("\n\n")
	}
	if channel != "" && chatID != "" {
		volatile.WriteString(fmt.Sprintf("## Current Session\nChannel: %s\nChat ID: %s\n\n", channel, chatID))
	}
	for _, vc := range volatileContext {
		if vc != "" {
			volatile.WriteString(vc)
			volatile.WriteString("\n\n")
		}
	}
	if volatile.Len() > 0 {
		messages = append(messages, providers.Message{
			Role:    "user",
			Content: strings.TrimSpace(volatile.String()),
		})
		// LLM expects alternating user/assistant — add ack so history starts clean
		messages = append(messages, providers.Message{
			Role:    "assistant",
			Content: "Understood, I have the updated context.",
		})
	}

	messages = append(messages, history...)

	messages = append(messages, providers.Message{
		Role:    "user",
		Content: currentMessage,
	})

	return messages
}

func (cb *ContextBuilder) AddToolResult(messages []providers.Message, toolCallID, toolName, result string) []providers.Message {
	messages = append(messages, providers.Message{
		Role:       "tool",
		Content:    result,
		ToolCallID: toolCallID,
	})
	return messages
}

func (cb *ContextBuilder) AddAssistantMessage(messages []providers.Message, content string, toolCalls []map[string]interface{}) []providers.Message {
	msg := providers.Message{
		Role:    "assistant",
		Content: content,
	}
	// Always add assistant message, whether or not it has tool calls
	messages = append(messages, msg)
	return messages
}


// GetSkillsInfo returns information about loaded skills.
func (cb *ContextBuilder) GetSkillsInfo() map[string]interface{} {
	allSkills := cb.skillsLoader.ListSkills()
	skillNames := make([]string, 0, len(allSkills))
	for _, s := range allSkills {
		skillNames = append(skillNames, s.Name)
	}
	return map[string]interface{}{
		"total":     len(allSkills),
		"available": len(allSkills),
		"names":     skillNames,
	}
}
