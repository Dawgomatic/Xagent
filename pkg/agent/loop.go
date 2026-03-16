// Xagent - Ultra-lightweight personal AI agent
// Inspired by and based on nanobot: https://github.com/HKUDS/nanobot
// License: MIT
//
// Copyright (c) 2026 Xagent contributors

package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unicode/utf8"

	"github.com/Dawgomatic/Xagent/pkg/bus"
	"github.com/Dawgomatic/Xagent/pkg/config"
	"github.com/Dawgomatic/Xagent/pkg/constants"
	"github.com/Dawgomatic/Xagent/pkg/epoch"
	"github.com/Dawgomatic/Xagent/pkg/identity"
	"github.com/Dawgomatic/Xagent/pkg/logger"
	"github.com/Dawgomatic/Xagent/pkg/mcp"
	"github.com/Dawgomatic/Xagent/pkg/memory"
	"github.com/Dawgomatic/Xagent/pkg/providers"
	"github.com/Dawgomatic/Xagent/pkg/session"
	"github.com/Dawgomatic/Xagent/pkg/state"
	"github.com/Dawgomatic/Xagent/pkg/tools"
	"github.com/Dawgomatic/Xagent/pkg/utils"
	"github.com/Dawgomatic/Xagent/pkg/vault"
)

type AgentLoop struct {
	bus            *bus.MessageBus
	provider       providers.LLMProvider
	workspace      string
	model          string
	contextWindow  int // Maximum context window size in tokens
	maxIterations  int
	maxTokens      int                     // SWE100821: LLM max_tokens from config (was hard-coded)
	temperature    float64                 // SWE100821: LLM temperature from config (was hard-coded)
	messageTimeout time.Duration           // SWE100821: Per-message timeout to prevent one slow call blocking all
	identity       *identity.AgentIdentity // SWE100821: Unique agent identity + time tracking
	epoch          *epoch.Manager          // SWE100821: Epoch lifecycle (wake/sleep journaling)
	sessions       *session.SessionManager
	state          *state.Manager
	contextBuilder *ContextBuilder
	tools          *tools.ToolRegistry
	middleware     *tools.ToolMiddleware   // SWE100821: Tool middleware (caching, circuit breaker, analytics)
	planner        *Planner                // SWE100821: Plan-Act-Reflect loop
	compressor     *ContextCompressor      // SWE100821: Context compression for long sessions
	provenance     *ProvenanceTracker      // SWE100821: Provenance tracking per turn
	dream          *DreamMode              // SWE100821: Offline reflection during idle
	sleepManager   *SleepManager           // Phase 3: Continuous Improvement Sleep Cycle
	personality    *PersonalityTracker     // SWE100821: Personality evolution
	feedback       *tools.FeedbackTool     // OpenClaw-RL: User feedback for RL training
	vaultWriter    *vault.VaultWriter       // Obsidian knowledge vault
	hindsight      *memory.HindsightMemory // Hindsight learning memory
	semanticMemory *memory.SemanticMemory  // SWE100821: Vector-based semantic memory
	running        atomic.Bool
	summarizing    sync.Map // Tracks which sessions are currently being summarized
}

// processOptions configures how a message is processed
type processOptions struct {
	SessionKey      string // Session identifier for history/context
	Channel         string // Target channel for tool execution
	ChatID          string // Target chat ID for tool execution
	UserMessage     string // User message content (may include prefix)
	DefaultResponse string // Response when LLM returns empty
	EnableSummary   bool   // Whether to trigger summarization
	SendResponse    bool   // Whether to send response via bus
	NoHistory       bool   // If true, don't load session history (for heartbeat)
}

// createToolRegistry creates a tool registry with common tools.
// This is shared between main agent and subagents.
func createToolRegistry(workspace string, restrict bool, cfg *config.Config, msgBus *bus.MessageBus) (*tools.ToolRegistry, *tools.FeedbackTool) {
	registry := tools.NewToolRegistry()

	// File system tools
	registry.Register(tools.NewReadFileTool(workspace, restrict))
	registry.Register(tools.NewWriteFileTool(workspace, restrict))
	registry.Register(tools.NewListDirTool(workspace, restrict))
	registry.Register(tools.NewEditFileTool(workspace, restrict))
	registry.Register(tools.NewAppendFileTool(workspace, restrict))

	// Shell execution
	registry.Register(tools.NewExecTool(workspace, restrict))

	if searchTool := tools.NewWebSearchTool(tools.WebSearchToolOptions{
		BraveAPIKey:          cfg.Tools.Web.Brave.APIKey,
		BraveMaxResults:      cfg.Tools.Web.Brave.MaxResults,
		BraveEnabled:         cfg.Tools.Web.Brave.Enabled,
		DuckDuckGoMaxResults: cfg.Tools.Web.DuckDuckGo.MaxResults,
		DuckDuckGoEnabled:    cfg.Tools.Web.DuckDuckGo.Enabled,
	}); searchTool != nil {
		registry.Register(searchTool)
	}
	registry.Register(tools.NewWebFetchTool(50000))

	// Hardware tools (I2C, SPI) - Linux only, returns error on other platforms
	registry.Register(tools.NewI2CTool())
	registry.Register(tools.NewSPITool())

	// LLM hardware analysis and model recommendation
	registry.Register(tools.NewLLMCheckTool())

	// Message tool - available to both agent and subagent
	// Subagent uses it to communicate directly with user
	messageTool := tools.NewMessageTool()
	messageTool.SetSendCallback(func(channel, chatID, content string) error {
		msgBus.PublishOutbound(bus.OutboundMessage{
			Channel: channel,
			ChatID:  chatID,
			Content: content,
		})
		return nil
	})
	registry.Register(messageTool)

	// OpenClaw-RL: Feedback tool for user ratings
	feedbackTool := tools.NewFeedbackTool(workspace)
	registry.Register(feedbackTool)

	// Vision: Local image analysis via Ollama vision models
	registry.Register(tools.NewVisionTool(workspace))

	// Browser: Headless web automation
	registry.Register(tools.NewBrowserTool(workspace))

	// SWE100821: Phone access via ADB/libimobiledevice
	if cfg.Phone.Enabled {
		registry.Register(tools.NewPhoneTool(workspace, cfg.Phone))
	}

	return registry, feedbackTool
}

func NewAgentLoop(cfg *config.Config, msgBus *bus.MessageBus, provider providers.LLMProvider) *AgentLoop {
	workspace := cfg.WorkspacePath()
	os.MkdirAll(workspace, 0755)

	restrict := cfg.Agents.Defaults.RestrictToWorkspace

	// Create tool registry for main agent
	toolsRegistry, feedbackTool := createToolRegistry(workspace, restrict, cfg, msgBus)

	// Create subagent manager with its own tool registry
	subagentManager := tools.NewSubagentManager(provider, cfg.Agents.Defaults.Model, workspace, msgBus)
	subagentTools, _ := createToolRegistry(workspace, restrict, cfg, msgBus)
	// Subagent doesn't need spawn/subagent tools to avoid recursion
	subagentManager.SetTools(subagentTools)

	// Register spawn tool (for main agent)
	spawnTool := tools.NewSpawnTool(subagentManager)
	toolsRegistry.Register(spawnTool)

	// Register subagent tool (synchronous execution)
	subagentTool := tools.NewSubagentTool(subagentManager)
	toolsRegistry.Register(subagentTool)

	sessionsManager := session.NewSessionManager(filepath.Join(workspace, "sessions"))

	// Create state manager for atomic state persistence
	stateManager := state.NewManager(workspace)

	// SWE100821: Initialize agent identity (unique in space and time) + boot-time tracking
	agentIdentity := identity.New(workspace)

	// SWE100821: Create context builder with semantic memory config
	contextBuilder := NewContextBuilder(workspace, cfg.SemanticMemory)
	contextBuilder.SetToolsRegistry(toolsRegistry)
	contextBuilder.SetIdentity(agentIdentity)

	// SWE100821: Register dynamic skill tools from SKILL.md frontmatter
	if contextBuilder.skillsLoader != nil {
		dynCount := registerDynamicSkillTools(contextBuilder.skillsLoader, toolsRegistry, workspace)
		if dynCount > 0 {
			logger.InfoCF("skills", "Registered dynamic skill tools", map[string]interface{}{"count": dynCount})
		}
	}

	// SWE100821: Config-driven MCP server initialization
	for _, serverCfg := range cfg.MCP.Servers {
		if !serverCfg.Enabled {
			continue
		}
		mcpCtx, mcpCancel := context.WithTimeout(context.Background(), 10*time.Second)
		transport, err := mcp.NewStdioTransport(serverCfg.Command, serverCfg.Args, nil)
		if err != nil {
			logger.WarnCF("mcp", "Failed to start MCP server",
				map[string]interface{}{"server": serverCfg.Name, "error": err.Error()})
			mcpCancel()
			continue
		}
		client := mcp.NewClient(serverCfg.Name, transport)
		count, err := registerMCPTools(mcpCtx, client, toolsRegistry)
		mcpCancel()
		if err != nil {
			logger.WarnCF("mcp", "MCP tool registration failed",
				map[string]interface{}{"server": serverCfg.Name, "error": err.Error()})
			client.Close()
			continue
		}
		logger.InfoCF("mcp", "MCP server registered",
			map[string]interface{}{"server": serverCfg.Name, "tools": count})
	}

	// SWE100821: Create tool middleware layer (caching, circuit breaker, analytics)
	toolMiddleware := tools.NewToolMiddleware(toolsRegistry)

	// SWE100821: Create planner for Plan-Act-Reflect loop
	planner := NewPlanner(provider, cfg.Agents.Defaults.Model, cfg.Agents.Defaults.MaxTokens, cfg.Agents.Defaults.Temperature)

	// SWE100821: Create context compressor for long sessions
	compressor := NewContextCompressor(provider, nil, cfg.Agents.Defaults.Model, "")

	// SWE100821: Create provenance tracker
	provenance := NewProvenanceTracker(workspace)

	// SWE100821: Create dream mode for offline reflection
	dreamMode := NewDreamMode(provider, cfg.Agents.Defaults.Model, workspace)

	// SWE100821: Create personality tracker
	personalityTracker := NewPersonalityTracker(workspace, provider, cfg.Agents.Defaults.Model)

	// Phase 3: Create Sleep Manager
	epochMgr := epoch.NewManager(workspace, agentIdentity)
	sleepManager := NewSleepManager(epochMgr, provider, msgBus, workspace, toolsRegistry)

	// SWE100821: Attach personality tracker to sleep manager for auto-analysis
	sleepManager.SetPersonality(personalityTracker)

	// SWE100821: Create semantic memory (Qdrant + Ollama embeddings) — config-driven
	semanticMem := memory.NewSemanticMemory(
		cfg.SemanticMemory.QdrantURL,
		cfg.SemanticMemory.OllamaURL,
		cfg.SemanticMemory.Collection,
		cfg.SemanticMemory.EmbedModel,
	)

	// Obsidian vault: create and initialize if enabled
	var vw *vault.VaultWriter
	var hm *memory.HindsightMemory
	if cfg.Vault.Enabled {
		vaultPath := cfg.Vault.Path
		if strings.HasPrefix(vaultPath, "~/") {
			if home, err := os.UserHomeDir(); err == nil {
				vaultPath = filepath.Join(home, vaultPath[2:])
			}
		}
		if vaultPath == "" {
			vaultPath = filepath.Join(workspace, "vault")
		}
		vw = vault.NewVaultWriter(vaultPath)
		if err := vw.Init(); err != nil {
			logger.WarnCF("vault", "Failed to initialize Obsidian vault",
				map[string]interface{}{"error": err.Error(), "path": vaultPath})
			vw = nil
		} else {
			hm = memory.NewHindsightMemory(vw, provider)
			// SWE100821: Wire semantic memory and vault root into hindsight for recall
			hm.SetSemanticMemory(semanticMem)
			hm.SetVaultRoot(vaultPath)
			hm.SetModel(cfg.Agents.Defaults.Model)
		}
	}

	return &AgentLoop{
		bus:            msgBus,
		provider:       provider,
		workspace:      workspace,
		model:          cfg.Agents.Defaults.Model,
		contextWindow:  cfg.Agents.Defaults.MaxTokens,
		maxIterations:  cfg.Agents.Defaults.MaxToolIterations,
		maxTokens:      cfg.Agents.Defaults.MaxTokens,   // SWE100821: from config, not hard-coded
		temperature:    cfg.Agents.Defaults.Temperature, // SWE100821: from config, not hard-coded
		messageTimeout: 5 * time.Minute,                 // SWE100821: per-message timeout
		identity:       agentIdentity,                   // SWE100821: unique identity + time tracking
		epoch:          epochMgr,                        // Epoch lifecycle (wake/sleep journaling)
		sessions:       sessionsManager,
		state:          stateManager,
		contextBuilder: contextBuilder,
		tools:          toolsRegistry,
		middleware:     toolMiddleware,     // SWE100821: middleware layer
		planner:        planner,            // SWE100821: Plan-Act-Reflect
		compressor:     compressor,         // SWE100821: context compression
		provenance:     provenance,         // SWE100821: provenance tracking
		dream:          dreamMode,          // SWE100821: dream mode
		sleepManager:   sleepManager,       // Phase 3: sleep cycle
		personality:    personalityTracker, // SWE100821: personality evolution
		feedback:       feedbackTool,       // OpenClaw-RL: feedback tool
		vaultWriter:    vw,                 // Obsidian knowledge vault
		hindsight:      hm,                 // Hindsight cognitive memory
		semanticMemory: semanticMem,        // SWE100821: semantic memory
		summarizing:    sync.Map{},
	}
}

func (al *AgentLoop) Run(ctx context.Context) error {
	al.running.Store(true)

	if al.sleepManager != nil {
		al.sleepManager.Start(ctx)
	}

	for al.running.Load() {
		select {
		case <-ctx.Done():
			return nil
		default:
			msg, ok := al.bus.ConsumeInbound(ctx)
			if !ok {
				continue
			}

			response, err := al.processMessage(ctx, msg)
			if err != nil {
				response = fmt.Sprintf("Error processing message: %v", err)
			}

			if response != "" {
				// Check if the message tool already sent a response during this round.
				// If so, skip publishing to avoid duplicate messages to the user.
				alreadySent := false
				if tool, ok := al.tools.Get("message"); ok {
					if mt, ok := tool.(*tools.MessageTool); ok {
						alreadySent = mt.HasSentInRound()
					}
				}

				if !alreadySent {
					al.bus.PublishOutbound(bus.OutboundMessage{
						Channel: msg.Channel,
						ChatID:  msg.ChatID,
						Content: response,
					})
				}
			}
		}
	}

	return nil
}

func (al *AgentLoop) Stop() {
	al.running.Store(false)
	if al.sleepManager != nil {
		al.sleepManager.Stop()
	}
}

func (al *AgentLoop) RegisterTool(tool tools.Tool) {
	al.tools.Register(tool)
}

// RecordLastChannel records the last active channel for this workspace.
// This uses the atomic state save mechanism to prevent data loss on crash.
func (al *AgentLoop) RecordLastChannel(channel string) error {
	return al.state.SetLastChannel(channel)
}

// RecordLastChatID records the last active chat ID for this workspace.
// This uses the atomic state save mechanism to prevent data loss on crash.
func (al *AgentLoop) RecordLastChatID(chatID string) error {
	return al.state.SetLastChatID(chatID)
}

func (al *AgentLoop) ProcessDirect(ctx context.Context, content, sessionKey string) (string, error) {
	return al.ProcessDirectWithChannel(ctx, content, sessionKey, "cli", "direct")
}

func (al *AgentLoop) ProcessDirectWithChannel(ctx context.Context, content, sessionKey, channel, chatID string) (string, error) {
	msg := bus.InboundMessage{
		Channel:    channel,
		SenderID:   "cron",
		ChatID:     chatID,
		Content:    content,
		SessionKey: sessionKey,
	}

	return al.processMessage(ctx, msg)
}

// ProcessHeartbeat processes a heartbeat request without session history.
// Each heartbeat is independent and doesn't accumulate context.
func (al *AgentLoop) ProcessHeartbeat(ctx context.Context, content, channel, chatID string) (string, error) {
	return al.runAgentLoop(ctx, processOptions{
		SessionKey:      "heartbeat",
		Channel:         channel,
		ChatID:          chatID,
		UserMessage:     content,
		DefaultResponse: "I've completed processing but have no response to give.",
		EnableSummary:   false,
		SendResponse:    false,
		NoHistory:       true, // Don't load session history for heartbeat
	})
}

func (al *AgentLoop) processMessage(ctx context.Context, msg bus.InboundMessage) (string, error) {
	// SWE100821: Per-message timeout so one slow call can't block all users
	msgCtx, msgCancel := context.WithTimeout(ctx, al.messageTimeout)
	defer msgCancel()
	ctx = msgCtx

	// Add message preview to log (show full content for error messages)
	var logContent string
	if strings.Contains(msg.Content, "Error:") || strings.Contains(msg.Content, "error") {
		logContent = msg.Content // Full content for errors
	} else {
		logContent = utils.Truncate(msg.Content, 80)
	}
	// SWE100821: Include request_id for end-to-end tracing
	logger.InfoCF("agent", fmt.Sprintf("[%s] Processing message from %s:%s: %s", msg.RequestID, msg.Channel, msg.SenderID, logContent),
		map[string]interface{}{
			"request_id":  msg.RequestID,
			"channel":     msg.Channel,
			"chat_id":     msg.ChatID,
			"sender_id":   msg.SenderID,
			"session_key": msg.SessionKey,
		})

	// Route system messages to processSystemMessage
	if msg.Channel == "system" {
		return al.processSystemMessage(ctx, msg)
	}

	// Process as user message
	return al.runAgentLoop(ctx, processOptions{
		SessionKey:      msg.SessionKey,
		Channel:         msg.Channel,
		ChatID:          msg.ChatID,
		UserMessage:     msg.Content,
		DefaultResponse: "I've completed processing but have no response to give.",
		EnableSummary:   true,
		SendResponse:    false,
	})
}

func (al *AgentLoop) processSystemMessage(ctx context.Context, msg bus.InboundMessage) (string, error) {
	// Verify this is a system message
	if msg.Channel != "system" {
		return "", fmt.Errorf("processSystemMessage called with non-system message channel: %s", msg.Channel)
	}

	logger.InfoCF("agent", "Processing system message",
		map[string]interface{}{
			"sender_id": msg.SenderID,
			"chat_id":   msg.ChatID,
		})

	// Parse origin channel from chat_id (format: "channel:chat_id")
	var originChannel string
	if idx := strings.Index(msg.ChatID, ":"); idx > 0 {
		originChannel = msg.ChatID[:idx]
	} else {
		// Fallback
		originChannel = "cli"
	}

	// Extract subagent result from message content
	// Format: "Task 'label' completed.\n\nResult:\n<actual content>"
	content := msg.Content
	if idx := strings.Index(content, "Result:\n"); idx >= 0 {
		content = content[idx+8:] // Extract just the result part
	}

	// Skip internal channels - only log, don't send to user
	if constants.IsInternalChannel(originChannel) {
		logger.InfoCF("agent", "Subagent completed (internal channel)",
			map[string]interface{}{
				"sender_id":   msg.SenderID,
				"content_len": len(content),
				"channel":     originChannel,
			})
		return "", nil
	}

	// Agent acts as dispatcher only - subagent handles user interaction via message tool
	// Don't forward result here, subagent should use message tool to communicate with user
	logger.InfoCF("agent", "Subagent completed",
		map[string]interface{}{
			"sender_id":   msg.SenderID,
			"channel":     originChannel,
			"content_len": len(content),
		})

	// Agent only logs, does not respond to user
	return "", nil
}

// runAgentLoop is the core message processing logic.
// It handles context building, LLM calls, tool execution, and response handling.
// SWE100821: Now integrates Plan-Act-Reflect, provenance tracking, personality observation, and dream mode.
func (al *AgentLoop) runAgentLoop(ctx context.Context, opts processOptions) (string, error) {
	// 0. Record last channel for heartbeat notifications (skip internal channels)
	if opts.Channel != "" && opts.ChatID != "" {
		// Don't record internal channels (cli, system, subagent)
		if !constants.IsInternalChannel(opts.Channel) {
			channelKey := fmt.Sprintf("%s:%s", opts.Channel, opts.ChatID)
			if err := al.RecordLastChannel(channelKey); err != nil {
				logger.WarnCF("agent", "Failed to record last channel: %v", map[string]interface{}{"error": err.Error()})
			}
		}
	}

	// SWE100821: Start provenance tracking for this turn
	turnID := fmt.Sprintf("%s-%d", opts.SessionKey, time.Now().UnixMilli())
	al.provenance.StartTurn(turnID, opts.SessionKey, opts.Channel, opts.UserMessage, al.model)

	// SWE100821: Record activity to reset dream mode idle timer
	if al.dream != nil {
		al.dream.RecordActivity()
	}

	// Phase 3: Increment Biological Fatigue (records activity and handles dynamic wake up)
	if al.sleepManager != nil && !opts.NoHistory {
		al.sleepManager.RecordActivity(0) // Tool calls updated after iteration completes
	}

	// 1. Update tool contexts
	al.updateToolContexts(opts.Channel, opts.ChatID)

	// OpenClaw-RL: Set session context on feedback tool and RL provider
	if al.feedback != nil {
		al.feedback.SetSessionID(opts.SessionKey)
	}
	if rlProvider, ok := al.provider.(*providers.RLProvider); ok {
		turnType := providers.RLTurnMain
		if opts.NoHistory || constants.IsInternalChannel(opts.Channel) {
			turnType = providers.RLTurnSide
		}
		rlProvider.SetSessionContext(&providers.RLSessionContext{
			SessionID:   opts.SessionKey,
			TurnType:    turnType,
			SessionDone: false,
		})
	}

	// 2. Build messages (skip history for heartbeat)
	var history []providers.Message
	var summary string
	if !opts.NoHistory {
		history = al.sessions.GetHistory(opts.SessionKey)
		summary = al.sessions.GetSummary(opts.SessionKey)
	}

	// SWE100821: Compress history when it exceeds threshold to preserve context window
	if al.compressor != nil && len(history) > 20 {
		compressed, recent, compErr := al.compressor.CompressHistory(ctx, history)
		if compErr == nil && compressed != "" {
			if summary != "" {
				summary += "\n\n" + compressed
			} else {
				summary = compressed
			}
			history = recent
		}
	}

	messages := al.contextBuilder.BuildMessages(
		history,
		summary,
		opts.UserMessage,
		nil,
		opts.Channel,
		opts.ChatID,
	)

	// SWE100821: Generate execution plan (Plan phase of Plan-Act-Reflect)
	var plan *AgentPlan
	if al.planner != nil && !opts.NoHistory {
		toolSummaries := al.tools.GetSummaries()
		var err error
		plan, err = al.planner.GeneratePlan(ctx, opts.UserMessage, toolSummaries)
		if err != nil {
			logger.WarnCF("planner", "Plan generation failed, proceeding without plan",
				map[string]interface{}{"error": err.Error()})
		} else if plan != nil {
			al.provenance.SetPlanSteps(len(plan.Steps))
		}
	}

	// SWE100821: Inject plan into context if generated
	if plan != nil {
		planContext := plan.ForSystemPrompt()
		if planContext != "" && len(messages) > 0 {
			messages[0].Content += "\n\n" + planContext
		}
	}

	// SWE100821: Inject personality adaptations into context
	if al.personality != nil {
		personalityContext := al.personality.ForSystemPrompt()
		if personalityContext != "" && len(messages) > 0 {
			messages[0].Content += "\n\n" + personalityContext
		}
	}

	// SWE100821: Inject tool-use learning hints into context
	if al.middleware != nil {
		hints := al.middleware.GetToolHints()
		if hints != "" && len(messages) > 0 {
			messages[0].Content += "\n\n## Tool Performance Hints\n" + hints
		}
	}

	// OpenClaw-RL: Inject user feedback summary into context
	if al.feedback != nil {
		feedbackSummary := al.feedback.GetFeedbackSummary()
		if feedbackSummary != "" && len(messages) > 0 {
			messages[0].Content += "\n\n" + feedbackSummary
		}
	}

	// 3. Save user message to session
	al.sessions.AddMessage(opts.SessionKey, "user", opts.UserMessage)

	// 4. Run LLM iteration loop
	// SWE100821: Pass plan into iteration loop for Plan-Act-Reflect
	finalContent, iteration, err := al.runLLMIteration(ctx, messages, opts, plan)
	if err != nil {
		return "", err
	}

	// 5. Handle empty response
	if finalContent == "" {
		finalContent = opts.DefaultResponse
	}

	// 6. Save final assistant message to session
	al.sessions.AddMessage(opts.SessionKey, "assistant", finalContent)
	al.sessions.Save(opts.SessionKey)

	// Hindsight Memory: Retain facts and experiences
	if al.hindsight != nil && !opts.NoHistory {
		if err := al.hindsight.Retain(ctx, opts.UserMessage, "User Prompt: "+opts.SessionKey); err != nil {
			logger.WarnCF("hindsight", "Failed to retain user message", map[string]interface{}{"error": err.Error()})
		}
		if err := al.hindsight.Retain(ctx, finalContent, "Agent Response: "+opts.SessionKey); err != nil {
			logger.WarnCF("hindsight", "Failed to retain assistant memory", map[string]interface{}{"error": err.Error()})
		}
	}

	// 7. Optional: summarization
	if opts.EnableSummary {
		al.maybeSummarize(opts.SessionKey)
	}

	// 8. Optional: send response via bus
	if opts.SendResponse {
		al.bus.PublishOutbound(bus.OutboundMessage{
			Channel: opts.Channel,
			ChatID:  opts.ChatID,
			Content: finalContent,
		})
	}

	// 9. Log response
	responsePreview := utils.Truncate(finalContent, 120)
	logger.InfoCF("agent", fmt.Sprintf("Response: %s", responsePreview),
		map[string]interface{}{
			"session_key":  opts.SessionKey,
			"iterations":   iteration,
			"final_length": len(finalContent),
		})

	// 10. SWE100821: Record epoch event + stats for wake/sleep journaling
	if al.epoch != nil {
		msgPreview := utils.Truncate(opts.UserMessage, 60)
		al.epoch.RecordEvent("message", fmt.Sprintf("[%s] %s", opts.Channel, msgPreview))
		al.epoch.UpdateStats(func(s *epoch.EpochStats) {
			s.MessagesProcessed++
			s.ToolCalls += iteration - 1 // iterations beyond the first are tool-call rounds
		})
	}

	// Phase 3: Track biological fatigue (add fatigue for tool iterations)
	if al.sleepManager != nil && !opts.NoHistory && iteration > 1 {
		al.sleepManager.RecordActivity(iteration - 1)
	}

	// 11. SWE100821: Finalize provenance tracking
	al.provenance.SetIterations(iteration)
	if err := al.provenance.FinishTurn(); err != nil {
		logger.WarnCF("provenance", "Failed to save provenance",
			map[string]interface{}{"error": err.Error()})
	}

	// 12. SWE100821: Feed personality tracker
	if al.personality != nil {
		al.personality.Observe(len(opts.UserMessage), len(finalContent), nil)
	}

	// 13. Obsidian vault: write session note with wikilinks
	if al.vaultWriter != nil {
		prov := al.provenance.GetCurrent()
		vaultData := vault.SessionData{
			SessionKey:  opts.SessionKey,
			Channel:     opts.Channel,
			Model:       al.model,
			UserMessage: opts.UserMessage,
			Response:    finalContent,
			LatencyMs:   time.Since(time.Now()).Milliseconds(), // approximate
			Iterations:  iteration,
		}
		if prov != nil {
			vaultData.LatencyMs = prov.LatencyMs
			for _, tc := range prov.ToolsCalled {
				vaultData.ToolsUsed = append(vaultData.ToolsUsed, tc.Name)
			}
			vaultData.SkillsUsed = prov.SkillsUsed
			vaultData.MemoryHits = prov.MemoryHits
			vaultData.PlanSteps = prov.PlanSteps
		}
		if err := al.vaultWriter.WriteSessionNote(vaultData); err != nil {
			logger.WarnCF("vault", "Failed to write session note",
				map[string]interface{}{"error": err.Error()})
		}
	}

	// 14. SWE100821: Auto-store to semantic memory for vector recall
	if al.semanticMemory != nil && al.semanticMemory.IsAvailable() && !opts.NoHistory {
		go func() {
			storeCtx, storeCancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer storeCancel()
			summary := fmt.Sprintf("User asked: %s. Agent responded: %s",
				utils.Truncate(opts.UserMessage, 200),
				utils.Truncate(finalContent, 200))
			if err := al.semanticMemory.StoreConversationSummary(storeCtx, opts.SessionKey, summary); err != nil {
				logger.DebugCF("memory", "Semantic auto-store failed", map[string]interface{}{"error": err.Error()})
			}
		}()
	}

	return finalContent, nil
}

// StartDreamMode starts the dream mode background loop.
// SWE100821: Call this after the agent loop is running.
func (al *AgentLoop) StartDreamMode(ctx context.Context) {
	if al.dream == nil {
		return
	}
	al.dream.SetInsightCallback(func(insight string) {
		// SWE100821: Send dream insight to last active channel
		channelKey := al.state.GetLastChannel()
		if channelKey == "" {
			return
		}
		parts := strings.SplitN(channelKey, ":", 2)
		if len(parts) != 2 {
			return
		}
		al.bus.PublishOutbound(bus.OutboundMessage{
			Channel: parts[0],
			ChatID:  parts[1],
			Content: insight,
		})
	})

	// Obsidian vault: write dream notes when dream mode produces results
	if al.vaultWriter != nil {
		al.dream.SetDreamCallback(func(result DreamResult) {
			if err := al.vaultWriter.WriteDreamNote(vault.DreamData{
				Insights:  result.Insights,
				Patterns:  result.Patterns,
				Questions: result.Questions,
				Timestamp: result.DreamedAt,
			}); err != nil {
				logger.WarnCF("vault", "Failed to write dream note",
					map[string]interface{}{"error": err.Error()})
			}
		})
	}

	al.dream.Start(ctx)
}

// GetMiddleware returns the tool middleware for external configuration.
// SWE100821: Allows gateway to add approval hooks, etc.
func (al *AgentLoop) GetMiddleware() *tools.ToolMiddleware {
	return al.middleware
}

// runLLMIteration executes the LLM call loop with tool handling.
// SWE100821: Now integrates Plan-Act-Reflect — reflects after each tool call,
// advances/fails plan steps, and replans on dead-ends.
func (al *AgentLoop) runLLMIteration(ctx context.Context, messages []providers.Message, opts processOptions, plan *AgentPlan) (string, int, error) {
	iteration := 0
	var finalContent string

	for iteration < al.maxIterations {
		iteration++

		logger.DebugCF("agent", "LLM iteration",
			map[string]interface{}{
				"iteration": iteration,
				"max":       al.maxIterations,
			})

		// Build tool definitions
		providerToolDefs := al.tools.ToProviderDefs()

		// Log LLM request details
		logger.DebugCF("agent", "LLM request",
			map[string]interface{}{
				"iteration":         iteration,
				"model":             al.model,
				"messages_count":    len(messages),
				"tools_count":       len(providerToolDefs),
				"max_tokens":        al.maxTokens,
				"temperature":       al.temperature,
				"system_prompt_len": len(messages[0].Content),
			})

		// Log full messages (detailed)
		logger.DebugCF("agent", "Full LLM request",
			map[string]interface{}{
				"iteration":     iteration,
				"messages_json": formatMessagesForLog(messages),
				"tools_json":    formatToolsForLog(providerToolDefs),
			})

		// SWE100821: Use config values instead of hard-coded 8192/0.7
		response, err := al.provider.Chat(ctx, messages, providerToolDefs, al.model, map[string]interface{}{
			"max_tokens":  al.maxTokens,
			"temperature": al.temperature,
		})

		if err != nil {
			logger.ErrorCF("agent", "LLM call failed",
				map[string]interface{}{
					"iteration": iteration,
					"error":     err.Error(),
				})
			return "", iteration, fmt.Errorf("LLM call failed: %w", err)
		}

		// Check if no tool calls - we're done
		if len(response.ToolCalls) == 0 {
			finalContent = response.Content
			logger.InfoCF("agent", "LLM response without tool calls (direct answer)",
				map[string]interface{}{
					"iteration":     iteration,
					"content_chars": len(finalContent),
				})
			break
		}

		// Log tool calls
		toolNames := make([]string, 0, len(response.ToolCalls))
		for _, tc := range response.ToolCalls {
			toolNames = append(toolNames, tc.Name)
		}
		logger.InfoCF("agent", "LLM requested tool calls",
			map[string]interface{}{
				"tools":     toolNames,
				"count":     len(response.ToolCalls),
				"iteration": iteration,
			})

		// Build assistant message with tool calls
		assistantMsg := providers.Message{
			Role:    "assistant",
			Content: response.Content,
		}
		for _, tc := range response.ToolCalls {
			argumentsJSON, _ := json.Marshal(tc.Arguments)
			assistantMsg.ToolCalls = append(assistantMsg.ToolCalls, providers.ToolCall{
				ID:   tc.ID,
				Type: "function",
				Function: &providers.FunctionCall{
					Name:      tc.Name,
					Arguments: string(argumentsJSON),
				},
			})
		}
		messages = append(messages, assistantMsg)

		// Save assistant message with tool calls to session
		al.sessions.AddFullMessage(opts.SessionKey, assistantMsg)

		// Execute tool calls
		for _, tc := range response.ToolCalls {
			// Log tool call with arguments preview
			argsJSON, _ := json.Marshal(tc.Arguments)
			argsPreview := utils.Truncate(string(argsJSON), 200)
			logger.InfoCF("agent", fmt.Sprintf("Tool call: %s(%s)", tc.Name, argsPreview),
				map[string]interface{}{
					"tool":      tc.Name,
					"iteration": iteration,
				})

			// SWE100821: Track tool execution timing for provenance
			start := time.Now()

			// Create async callback for tools that implement AsyncTool
			asyncCallback := func(callbackCtx context.Context, result *tools.ToolResult) {
				if !result.Silent && result.ForUser != "" {
					logger.InfoCF("agent", "Async tool completed, agent will handle notification",
						map[string]interface{}{
							"tool":        tc.Name,
							"content_len": len(result.ForUser),
						})
				}
			}

			// SWE100821: Execute through middleware layer (caching, circuit breaker, analytics)
			var toolResult *tools.ToolResult
			if al.middleware != nil {
				toolResult = al.middleware.Execute(ctx, tc.Name, tc.Arguments, opts.Channel, opts.ChatID, asyncCallback)
			} else {
				toolResult = al.tools.ExecuteWithContext(ctx, tc.Name, tc.Arguments, opts.Channel, opts.ChatID, asyncCallback)
			}

			// SWE100821: Record tool call in provenance
			toolLatency := time.Since(start)
			al.provenance.RecordToolCall(tc.Name, !toolResult.IsError, toolLatency.Milliseconds())

			// Send ForUser content to user immediately if not Silent
			if !toolResult.Silent && toolResult.ForUser != "" && opts.SendResponse {
				al.bus.PublishOutbound(bus.OutboundMessage{
					Channel: opts.Channel,
					ChatID:  opts.ChatID,
					Content: toolResult.ForUser,
				})
				logger.DebugCF("agent", "Sent tool result to user",
					map[string]interface{}{
						"tool":        tc.Name,
						"content_len": len(toolResult.ForUser),
					})
			}

			// Determine content for LLM based on tool result
			contentForLLM := toolResult.ForLLM
			if contentForLLM == "" && toolResult.Err != nil {
				contentForLLM = toolResult.Err.Error()
			}

			toolResultMsg := providers.Message{
				Role:       "tool",
				Content:    contentForLLM,
				ToolCallID: tc.ID,
			}
			messages = append(messages, toolResultMsg)

			// Save tool result message to session
			al.sessions.AddFullMessage(opts.SessionKey, toolResultMsg)

			// SWE100821: Plan-Act-Reflect — reflect after each tool call
			if plan != nil && al.planner != nil {
				if toolResult.IsError {
					plan.MarkCurrentFailed()
					logger.InfoCF("planner", "Plan step failed",
						map[string]interface{}{"tool": tc.Name, "step": currentStepDescription(plan)})
				} else {
					plan.AdvanceStep()
				}

				scratchpad, shouldReplan, reflectErr := al.planner.Reflect(ctx, plan, tc.Name, contentForLLM)
				if reflectErr != nil {
					logger.WarnCF("planner", "Reflection failed", map[string]interface{}{"error": reflectErr.Error()})
				} else {
					plan.Scratchpad = scratchpad

					if shouldReplan && plan.Replans < 2 {
						plan.Replans++
						logger.InfoCF("planner", "Replanning", map[string]interface{}{"replan_count": plan.Replans})
						toolSummaries := al.tools.GetSummaries()
						newPlan, replanErr := al.planner.GeneratePlan(ctx, plan.Goal+"\n\nPrevious approach notes: "+scratchpad, toolSummaries)
						if replanErr == nil && newPlan != nil {
							newPlan.Replans = plan.Replans
							newPlan.Scratchpad = scratchpad
							plan = newPlan
						}
					}
				}

				// SWE100821: Update plan context in system prompt for next iteration
				planContext := plan.ForSystemPrompt()
				if planContext != "" && len(messages) > 0 {
					// Replace existing plan section or append
					sysContent := messages[0].Content
					if idx := strings.Index(sysContent, "## Current Plan"); idx >= 0 {
						messages[0].Content = sysContent[:idx] + planContext
					} else {
						messages[0].Content += "\n\n" + planContext
					}
				}
			}
		}

		// SWE100821: Early exit if plan is fully complete
		if plan != nil && plan.IsComplete() {
			logger.InfoCF("planner", "Plan complete, finishing iteration loop", nil)
		}
	}

	return finalContent, iteration, nil
}

// updateToolContexts updates the context for tools that need channel/chatID info.
func (al *AgentLoop) updateToolContexts(channel, chatID string) {
	// Use ContextualTool interface instead of type assertions
	if tool, ok := al.tools.Get("message"); ok {
		if mt, ok := tool.(tools.ContextualTool); ok {
			mt.SetContext(channel, chatID)
		}
	}
	if tool, ok := al.tools.Get("spawn"); ok {
		if st, ok := tool.(tools.ContextualTool); ok {
			st.SetContext(channel, chatID)
		}
	}
	if tool, ok := al.tools.Get("subagent"); ok {
		if st, ok := tool.(tools.ContextualTool); ok {
			st.SetContext(channel, chatID)
		}
	}
}

// maybeSummarize triggers summarization if the session history exceeds thresholds.
func (al *AgentLoop) maybeSummarize(sessionKey string) {
	newHistory := al.sessions.GetHistory(sessionKey)
	tokenEstimate := al.estimateTokens(newHistory)
	threshold := al.contextWindow * 75 / 100

	if len(newHistory) > 20 || tokenEstimate > threshold {
		if _, loading := al.summarizing.LoadOrStore(sessionKey, true); !loading {
			go func() {
				defer al.summarizing.Delete(sessionKey)
				al.summarizeSession(sessionKey)
			}()
		}
	}
}

// GetStartupInfo returns information about loaded tools and skills for logging.
func (al *AgentLoop) GetStartupInfo() map[string]interface{} {
	info := make(map[string]interface{})

	// Tools info
	tools := al.tools.List()
	info["tools"] = map[string]interface{}{
		"count": len(tools),
		"names": tools,
	}

	// Skills info
	info["skills"] = al.contextBuilder.GetSkillsInfo()

	// SWE100821: Agent identity + time tracking
	info["identity"] = map[string]interface{}{
		"agent_id":   al.identity.AgentID,
		"session_id": al.identity.SessionID,
		"boot_time":  al.identity.BootTime.Format(time.RFC3339),
		"birth_time": al.identity.BirthTime.Format(time.RFC3339),
	}

	return info
}

// GetIdentity returns the agent's identity for external consumers.
// SWE100821: Exposes identity for status/health endpoints.
func (al *AgentLoop) GetIdentity() *identity.AgentIdentity {
	return al.identity
}

// SetModel dynamically switches the LLM model used by the agent.
// SWE100821: Called when hardware tier changes to adapt to available resources.
func (al *AgentLoop) SetModel(model string) {
	al.model = model
}

// GetModel returns the current model name.
func (al *AgentLoop) GetModel() string {
	return al.model
}

// SetEpoch attaches the epoch manager so the agent can journal events.
// SWE100821: Epoch lifecycle (wake/sleep journaling).
func (al *AgentLoop) SetEpoch(em *epoch.Manager) {
	al.epoch = em
}

// SetPreviousEpoch injects the last epoch into the system prompt context.
// SWE100821: Wake-up recall — agent remembers what happened last session.
func (al *AgentLoop) SetPreviousEpoch(rec *epoch.Record) {
	al.contextBuilder.SetPreviousEpoch(rec)
}

// GetSessionStats returns counts useful for epoch journaling.
// SWE100821: Feeds epoch stats at sleep time.
func (al *AgentLoop) GetSessionStats() (sessions int) {
	allSessions := al.sessions.GetAllKeys()
	return len(allSessions)
}

// formatMessagesForLog formats messages for logging
func formatMessagesForLog(messages []providers.Message) string {
	if len(messages) == 0 {
		return "[]"
	}

	var result string
	result += "[\n"
	for i, msg := range messages {
		result += fmt.Sprintf("  [%d] Role: %s\n", i, msg.Role)
		if msg.ToolCalls != nil && len(msg.ToolCalls) > 0 {
			result += "  ToolCalls:\n"
			for _, tc := range msg.ToolCalls {
				result += fmt.Sprintf("    - ID: %s, Type: %s, Name: %s\n", tc.ID, tc.Type, tc.Name)
				if tc.Function != nil {
					result += fmt.Sprintf("      Arguments: %s\n", utils.Truncate(tc.Function.Arguments, 200))
				}
			}
		}
		if msg.Content != "" {
			content := utils.Truncate(msg.Content, 200)
			result += fmt.Sprintf("  Content: %s\n", content)
		}
		if msg.ToolCallID != "" {
			result += fmt.Sprintf("  ToolCallID: %s\n", msg.ToolCallID)
		}
		result += "\n"
	}
	result += "]"
	return result
}

// formatToolsForLog formats tool definitions for logging
func formatToolsForLog(tools []providers.ToolDefinition) string {
	if len(tools) == 0 {
		return "[]"
	}

	var result string
	result += "[\n"
	for i, tool := range tools {
		result += fmt.Sprintf("  [%d] Type: %s, Name: %s\n", i, tool.Type, tool.Function.Name)
		result += fmt.Sprintf("      Description: %s\n", tool.Function.Description)
		if len(tool.Function.Parameters) > 0 {
			result += fmt.Sprintf("      Parameters: %s\n", utils.Truncate(fmt.Sprintf("%v", tool.Function.Parameters), 200))
		}
	}
	result += "]"
	return result
}

// summarizeSession summarizes the conversation history for a session.
func (al *AgentLoop) summarizeSession(sessionKey string) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	history := al.sessions.GetHistory(sessionKey)
	summary := al.sessions.GetSummary(sessionKey)

	// Keep last 4 messages for continuity
	if len(history) <= 4 {
		return
	}

	toSummarize := history[:len(history)-4]

	// Oversized Message Guard
	// Skip messages larger than 50% of context window to prevent summarizer overflow
	maxMessageTokens := al.contextWindow / 2
	validMessages := make([]providers.Message, 0)
	omitted := false

	for _, m := range toSummarize {
		if m.Role != "user" && m.Role != "assistant" {
			continue
		}
		// Estimate tokens for this message
		msgTokens := len(m.Content) / 4
		if msgTokens > maxMessageTokens {
			omitted = true
			continue
		}
		validMessages = append(validMessages, m)
	}

	if len(validMessages) == 0 {
		return
	}

	// Multi-Part Summarization
	// Split into two parts if history is significant
	var finalSummary string
	if len(validMessages) > 10 {
		mid := len(validMessages) / 2
		part1 := validMessages[:mid]
		part2 := validMessages[mid:]

		s1, _ := al.summarizeBatch(ctx, part1, "")
		s2, _ := al.summarizeBatch(ctx, part2, "")

		// Merge them
		mergePrompt := fmt.Sprintf("Merge these two conversation summaries into one cohesive summary:\n\n1: %s\n\n2: %s", s1, s2)
		resp, err := al.provider.Chat(ctx, []providers.Message{{Role: "user", Content: mergePrompt}}, nil, al.model, map[string]interface{}{
			"max_tokens":  1024,
			"temperature": 0.3,
		})
		if err == nil {
			finalSummary = resp.Content
		} else {
			finalSummary = s1 + " " + s2
		}
	} else {
		finalSummary, _ = al.summarizeBatch(ctx, validMessages, summary)
	}

	if omitted && finalSummary != "" {
		finalSummary += "\n[Note: Some oversized messages were omitted from this summary for efficiency.]"
	}

	if finalSummary != "" {
		al.sessions.SetSummary(sessionKey, finalSummary)
		al.sessions.TruncateHistory(sessionKey, 4)
		al.sessions.Save(sessionKey)
	}
}

// summarizeBatch summarizes a batch of messages.
func (al *AgentLoop) summarizeBatch(ctx context.Context, batch []providers.Message, existingSummary string) (string, error) {
	prompt := "Provide a concise summary of this conversation segment, preserving core context and key points.\n"
	if existingSummary != "" {
		prompt += "Existing context: " + existingSummary + "\n"
	}
	prompt += "\nCONVERSATION:\n"
	for _, m := range batch {
		prompt += fmt.Sprintf("%s: %s\n", m.Role, m.Content)
	}

	response, err := al.provider.Chat(ctx, []providers.Message{{Role: "user", Content: prompt}}, nil, al.model, map[string]interface{}{
		"max_tokens":  1024,
		"temperature": 0.3,
	})
	if err != nil {
		return "", err
	}
	return response.Content, nil
}

// estimateTokens estimates the number of tokens in a message list.
// Uses rune count instead of byte length so that CJK and other multi-byte
// characters are not over-counted (a Chinese character is 3 bytes but roughly
// one token).
func (al *AgentLoop) estimateTokens(messages []providers.Message) int {
	total := 0
	for _, m := range messages {
		total += utf8.RuneCountInString(m.Content) / 3
	}
	return total
}
