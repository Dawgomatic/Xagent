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
	"github.com/Dawgomatic/Xagent/pkg/memory"
	"github.com/Dawgomatic/Xagent/pkg/providers"
	"github.com/Dawgomatic/Xagent/pkg/skills"
	"github.com/Dawgomatic/Xagent/pkg/tools"
)

type ContextBuilder struct {
	workspace      string
	skillsLoader   *skills.SkillsLoader
	memory         *MemoryStore
	semanticMemory *memory.SemanticMemory  // SWE100821: Vector-based semantic memory
	tools          *tools.ToolRegistry     // Direct reference to tool registry
	identity       *identity.AgentIdentity // SWE100821: Agent identity + time tracking
	prevEpoch      *epoch.Record           // SWE100821: Previous epoch for wake-up recall
	autoDiscoverer *skills.AutoDiscoverer  // SWE100821: Skill auto-discovery
	bootstrapCache map[string]string       // Cache for AGENTS.md, SOUL.md, etc.
	bootstrapMTime map[string]time.Time    // MTime for cache invalidation
	compactPrompt  bool                    // SWE100821: Minimal system prompt for embedded/PicoLM
}

func getGlobalConfigDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".xagent")
}

func NewContextBuilder(workspace string, smCfg ...config.SemanticMemoryConfig) *ContextBuilder {
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

	// SWE100821: Initialize semantic memory (Qdrant + Ollama embeddings) — config-driven
	var qdrantURL, ollamaURL, collection, embedModel string
	if len(smCfg) > 0 {
		qdrantURL = smCfg[0].QdrantURL
		ollamaURL = smCfg[0].OllamaURL
		collection = smCfg[0].Collection
		embedModel = smCfg[0].EmbedModel
	}
	semanticMem := memory.NewSemanticMemory(qdrantURL, ollamaURL, collection, embedModel)

	// SWE100821: Initialize skill auto-discoverer
	autoDisc := skills.NewAutoDiscoverer(workspace)

	return &ContextBuilder{
		workspace:      workspace,
		skillsLoader:   skills.NewSkillsLoader(workspace, globalSkillsDir, builtinSkillsDir),
		memory:         NewMemoryStore(workspace),
		semanticMemory: semanticMem,
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

	toolsSection := cb.buildToolsSection()

	return fmt.Sprintf(`# xagent 

You are xagent, a helpful AI assistant.

## Current Time
%s

%s## Runtime
%s

## Workspace
Your workspace is at: %s
- Memory: %s/memory/MEMORY.md
- Daily Notes: %s/memory/YYYYMM/YYYYMMDD.md
- Skills: %s/skills/{skill-name}/SKILL.md

%s

## Important Rules

1. **Use tools when needed** - When you need to perform an action (schedule reminders, send messages, execute commands, etc.), call the appropriate tool. Do NOT pretend to do it.

2. **Respond with text after getting results** - Once you have the information from a tool call, respond directly to the user with a clear text answer. Do NOT keep calling tools after you have what you need.

3. **Be helpful and concise** - Summarize tool results for the user in a clear response.

4. **Memory** - When remembering something, write to %s/memory/MEMORY.md`,
		now, identitySection, runtimeStr, workspacePath, workspacePath, workspacePath, workspacePath, toolsSection, workspacePath)
}

func (cb *ContextBuilder) buildToolsSection() string {
	if cb.tools == nil {
		return ""
	}

	summaries := cb.tools.GetSummaries()
	if len(summaries) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("## Available Tools\n\n")
	// SWE100821: Balanced instruction — use tools when needed, but respond after
	sb.WriteString("Use tools to perform actions. After getting results, respond to the user with a clear text answer.\n\n")
	for _, s := range summaries {
		sb.WriteString(s)
		sb.WriteString("\n")
	}

	return sb.String()
}

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

	// Memory context
	memoryContext := cb.memory.GetMemoryContext()
	if memoryContext != "" {
		parts = append(parts, "# Memory\n\n"+memoryContext)
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

func (cb *ContextBuilder) loadSkills() string {
	allSkills := cb.skillsLoader.ListSkills()
	if len(allSkills) == 0 {
		return ""
	}

	var skillNames []string
	for _, s := range allSkills {
		skillNames = append(skillNames, s.Name)
	}

	content := cb.skillsLoader.LoadSkillsForContext(skillNames)
	if content == "" {
		return ""
	}

	return "# Skill Definitions\n\n" + content
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
