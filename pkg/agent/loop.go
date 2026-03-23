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
	"github.com/Dawgomatic/Xagent/pkg/health"
	"github.com/Dawgomatic/Xagent/pkg/epoch"
	"github.com/Dawgomatic/Xagent/pkg/identity"
	"github.com/Dawgomatic/Xagent/pkg/logger"
	"github.com/Dawgomatic/Xagent/pkg/mcp"
	"github.com/Dawgomatic/Xagent/pkg/memory"
	"github.com/Dawgomatic/Xagent/pkg/providers"
	"github.com/Dawgomatic/Xagent/pkg/session"
	"github.com/Dawgomatic/Xagent/pkg/skills"
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
	plannerDisabled bool                   // SWE100821: Skip planner on embedded/edge hw
	compressor     *ContextCompressor      // SWE100821: Context compression for long sessions
	provenance     *ProvenanceTracker      // SWE100821: Provenance tracking per turn
	dream          *DreamMode              // SWE100821: Offline reflection during idle
	sleepManager   *SleepManager           // Phase 3: Continuous Improvement Sleep Cycle
	personality    *PersonalityTracker     // SWE100821: Personality evolution
	feedback       *tools.FeedbackTool     // OpenClaw-RL: User feedback for RL training
	vaultWriter    *vault.VaultWriter       // Obsidian knowledge vault
	hindsight      *memory.HindsightMemory // Hindsight learning memory
	semanticMemory *memory.SemanticMemory  // SWE100821: Vector-based semantic memory
	temporalIndex  *memory.TemporalIndex   // SWE100821: Time-aware memory retrieval
	consolidator   *memory.Consolidator    // SWE100821: Weekly/monthly memory rollup
	metrics        *health.Metrics         // SWE100821: Live metrics for dashboard System tab
	perception     PerceptionProvider      // SWE100821: Ambient sensor data for system prompt
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
	MaxIterations   int    // SWE100821: Override max iterations (0 = use default)
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
	// SWE100821: GOALS.md project/goal tracking (list, add, update, review)
	registry.Register(tools.NewGoalsTool(workspace))

	// SWE100821: Shell execution — configurable network/script access
	execTool := tools.NewExecToolWithConfig(workspace, restrict,
		cfg.Tools.Exec.AllowNetwork, cfg.Tools.Exec.AllowScripts, cfg.Tools.Exec.DenyCommands)
	if cfg.Tools.Exec.TimeoutSecs > 0 {
		execTool.SetTimeout(time.Duration(cfg.Tools.Exec.TimeoutSecs) * time.Second)
	}
	registry.Register(execTool)

	if searchTool := tools.NewWebSearchTool(tools.WebSearchToolOptions{
		BraveAPIKey:          cfg.Tools.Web.Brave.APIKey,
		BraveMaxResults:      cfg.Tools.Web.Brave.MaxResults,
		BraveEnabled:         cfg.Tools.Web.Brave.Enabled,
		DuckDuckGoMaxResults: cfg.Tools.Web.DuckDuckGo.MaxResults,
		DuckDuckGoEnabled:    cfg.Tools.Web.DuckDuckGo.Enabled,
	}); searchTool != nil {
		registry.Register(searchTool)
	}
	// SWE100821: web_fetch removed — fetch tool is a superset (GET + headers, same HTML stripping)

	// Hardware tools (I2C, SPI, USB) - Linux only, returns error on other platforms
	registry.Register(tools.NewI2CTool())
	registry.Register(tools.NewSPITool())
	// SWE100821: USB device enumeration so agent can see what's physically connected
	registry.Register(tools.NewUSBTool())

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

	// SWE100821: Vision — configurable model (moondream on embedded, llava on desktop)
	registry.Register(tools.NewVisionTool(workspace, cfg.Tools.Vision.Model, cfg.Tools.Vision.OllamaURL))

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

	// SWE100821: Register skills tool so the agent can search the 10K+ catalog
	// and install skills at runtime instead of hallucinating skill names.
	skillInstaller := skills.NewSkillInstaller(workspace)
	toolsRegistry.Register(tools.NewSkillsTool(
		contextBuilder.autoDiscoverer,
		skillInstaller,
		contextBuilder.skillsLoader,
	))

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
	// SWE100821: Use the configured model for sleep subagent, not hardcoded "llama3"
	sleepManager.SetModel(cfg.Agents.Defaults.Model)

	// SWE100821: Attach personality tracker to sleep manager for auto-analysis
	sleepManager.SetPersonality(personalityTracker)

	// SWE100821: Create semantic memory (Qdrant + Ollama embeddings) — config-driven
	semanticMem := memory.NewSemanticMemory(
		cfg.SemanticMemory.QdrantURL,
		cfg.SemanticMemory.OllamaURL,
		cfg.SemanticMemory.Collection,
		cfg.SemanticMemory.EmbedModel,
	)

	// SWE100821: Temporal memory + consolidation — time-aware recall + periodic rollup
	temporalIdx := memory.NewTemporalIndex(workspace)
	consolidator := memory.NewConsolidator(workspace, provider, cfg.Agents.Defaults.Model)

	// SWE100821: Register fetch tool (GoalsTool already registered in createToolRegistry)
	toolsRegistry.Register(tools.NewFetchTool())

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
		contextWindow:  resolveContextWindow(cfg.Agents.Defaults.ContextWindow, cfg.Agents.Defaults.MaxTokens),
		maxIterations:  cfg.Agents.Defaults.MaxToolIterations,
		maxTokens:      cfg.Agents.Defaults.MaxTokens,   // SWE100821: from config, not hard-coded
		temperature:    cfg.Agents.Defaults.Temperature, // SWE100821: from config, not hard-coded
		messageTimeout: resolveMessageTimeout(cfg.Agents.Defaults.MessageTimeoutSecs), // SWE100821: configurable per-message timeout
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
		temporalIndex:  temporalIdx,        // SWE100821: temporal memory
		consolidator:   consolidator,       // SWE100821: memory consolidation
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

// SWE100821: Expose running state for watchdog subsystem monitoring.
func (al *AgentLoop) IsRunning() bool {
	return al.running.Load()
}

// SWE100821: Expose internal subsystem health for watchdog monitoring.
func (al *AgentLoop) IsSleepRunning() bool {
	return al.sleepManager != nil && al.sleepManager.IsRunning()
}

func (al *AgentLoop) GetFatigueLevel() float64 {
	if al.sleepManager != nil {
		return al.sleepManager.GetFatigueLevel()
	}
	return 0
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
		MaxIterations:   8,    // SWE100821: Cap heartbeat — prevents runaway phone/tool spam
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

	// SWE100821: Increment message counter for dashboard metrics
	if al.metrics != nil {
		al.metrics.IncMessage()
	}

	// Route system messages to processSystemMessage
	if msg.Channel == "system" {
		return al.processSystemMessage(ctx, msg)
	}

	// SWE100821: Context compaction keeps messages bounded — no hard iteration cap needed
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
	// SWE100821: Track turn start for accurate latency measurement
	turnStart := time.Now()
	_ = turnStart // used in vault session note below

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

	// SWE100821: Parallel context building (from Hindsight retrieval.py pattern).
	// Semantic search (Qdrant HTTP, ~50-200ms) runs concurrently with history load
	// so context build time = max(semantic, history) instead of sum.
	var history []providers.Message
	var summary string
	var semanticContext string

	var ctxWg sync.WaitGroup
	if !opts.NoHistory {
		ctxWg.Add(1)
		go func() {
			defer ctxWg.Done()
			history = al.sessions.GetHistory(opts.SessionKey)
			summary = al.sessions.GetSummary(opts.SessionKey)
		}()
	}
	if al.semanticMemory != nil && al.semanticMemory.IsAvailable() && !opts.NoHistory {
		ctxWg.Add(1)
		go func() {
			defer ctxWg.Done()
			smCtx, smCancel := context.WithTimeout(ctx, 3*time.Second)
			defer smCancel()
			semanticContext = al.semanticMemory.ForSystemPrompt(smCtx, opts.UserMessage, 5)
		}()
	}
	ctxWg.Wait()

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

	// SWE100821: Generate execution plan (Plan phase of Plan-Act-Reflect)
	var plan *AgentPlan
	if al.planner != nil && !opts.NoHistory && !al.plannerDisabled {
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

	// SWE100821: Collect volatile context that changes per-turn.
	// Passed as extra args to BuildMessages so the system prompt stays STATIC
	// for prompt caching (Anthropic/OpenRouter). From Nanobot context.py pattern.
	var volatileCtx []string
	if plan != nil {
		if pc := plan.ForSystemPrompt(); pc != "" {
			volatileCtx = append(volatileCtx, pc)
		}
	}
	if al.personality != nil {
		if pc := al.personality.ForSystemPrompt(); pc != "" {
			volatileCtx = append(volatileCtx, pc)
		}
	}
	if al.middleware != nil {
		if hints := al.middleware.GetToolHints(); hints != "" {
			volatileCtx = append(volatileCtx, "## Tool Performance Hints\n"+hints)
		}
	}
	if al.feedback != nil {
		if fs := al.feedback.GetFeedbackSummary(); fs != "" {
			volatileCtx = append(volatileCtx, fs)
		}
	}
	if semanticContext != "" {
		volatileCtx = append(volatileCtx, semanticContext)
	}
	// SWE100821: Inject temporal memory — time-aware recall for "yesterday", "last week", etc.
	if al.temporalIndex != nil {
		if tc := al.temporalIndex.ForContext(opts.UserMessage); tc != "" {
			volatileCtx = append(volatileCtx, tc)
		}
	}
	// SWE100821: Inject fatigue level into context so the model is aware of its energy state.
	// Previously fatigue was tracked but never shown to the LLM — purely decorative.
	if al.sleepManager != nil {
		fatigue := al.sleepManager.GetFatigueLevel()
		if fatigue > 0.3 {
			fatigueDesc := "moderate"
			if fatigue > 0.7 {
				fatigueDesc = "high"
			}
			volatileCtx = append(volatileCtx, fmt.Sprintf(
				"## Energy Status\nYour fatigue level is %s (%.0f%%). Consider being more concise in responses.",
				fatigueDesc, fatigue*100))
		}
	}

	// SWE100821: Inject ambient sensor readings into agent context
	if al.perception != nil {
		if pc := al.perception.ForSystemPrompt(); pc != "" {
			volatileCtx = append(volatileCtx, pc)
		}
	}

	messages := al.contextBuilder.BuildMessages(
		history,
		summary,
		opts.UserMessage,
		nil,
		opts.Channel,
		opts.ChatID,
		volatileCtx...,
	)

	// 3. Save user message to session
	al.sessions.AddMessage(opts.SessionKey, "user", opts.UserMessage)

	// 4. Run LLM iteration loop
	// SWE100821: Pass plan into iteration loop for Plan-Act-Reflect
	finalContent, iteration, err := al.runLLMIteration(ctx, messages, opts, plan)
	if err != nil {
		return "", err
	}

	// SWE100821: Strip JSON wrappers from responses — llama3.1:8b sometimes wraps
	// its text response as {"type":"text","data":"actual response"} instead of just text.
	finalContent = unwrapJSONResponse(finalContent)

	// 5. Handle empty response
	if finalContent == "" {
		finalContent = opts.DefaultResponse
	}

	// 6. Save final assistant message to session
	al.sessions.AddMessage(opts.SessionKey, "assistant", finalContent)
	// SWE100821: Check save error — silent failures can lose the last turn on crash
	if err := al.sessions.Save(opts.SessionKey); err != nil {
		logger.WarnCF("session", "Failed to persist session",
			map[string]interface{}{"session": opts.SessionKey, "error": err.Error()})
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

	// SWE100821: Parallel post-response operations (from Hindsight retrieval.py pattern).
	// Response is already sent — run all bookkeeping concurrently to reduce
	// post-response latency from sum(ops) to max(ops).
	var postWg sync.WaitGroup

	// Hindsight Memory: Retain facts and experiences
	if al.hindsight != nil && !opts.NoHistory {
		postWg.Add(1)
		go func() {
			defer postWg.Done()
			if err := al.hindsight.Retain(ctx, opts.UserMessage, "User Prompt: "+opts.SessionKey); err != nil {
				logger.WarnCF("hindsight", "Failed to retain user message", map[string]interface{}{"error": err.Error()})
			}
			if err := al.hindsight.Retain(ctx, finalContent, "Agent Response: "+opts.SessionKey); err != nil {
				logger.WarnCF("hindsight", "Failed to retain assistant memory", map[string]interface{}{"error": err.Error()})
			}
		}()
	}

	// 10. SWE100821: Record epoch event + stats for wake/sleep journaling
	if al.epoch != nil {
		postWg.Add(1)
		go func() {
			defer postWg.Done()
			msgPreview := utils.Truncate(opts.UserMessage, 60)
			al.epoch.RecordEvent("message", fmt.Sprintf("[%s] %s", opts.Channel, msgPreview))
			al.epoch.UpdateStats(func(s *epoch.EpochStats) {
				s.MessagesProcessed++
				s.ToolCalls += iteration - 1
			})
		}()
	}

	// Phase 3: Track biological fatigue (add fatigue for tool iterations)
	if al.sleepManager != nil && !opts.NoHistory && iteration > 1 {
		al.sleepManager.RecordActivity(iteration - 1)
	}

	// 11. SWE100821: Finalize provenance tracking
	al.provenance.SetIterations(iteration)
	postWg.Add(1)
	go func() {
		defer postWg.Done()
		if err := al.provenance.FinishTurn(); err != nil {
			logger.WarnCF("provenance", "Failed to save provenance",
				map[string]interface{}{"error": err.Error()})
		}
	}()

	// 12. SWE100821: Feed personality tracker with tool names from provenance
	if al.personality != nil {
		var toolNames []string
		if prov := al.provenance.GetCurrent(); prov != nil {
			for _, tc := range prov.ToolsCalled {
				toolNames = append(toolNames, tc.Name)
			}
		}
		al.personality.Observe(len(opts.UserMessage), len(finalContent), toolNames)
	}

	// SWE100821: Record conversation in temporal memory for time-based recall
	if al.temporalIndex != nil && !opts.NoHistory {
		postWg.Add(1)
		go func() {
			defer postWg.Done()
			summary := utils.Truncate(opts.UserMessage, 120) + " → " + utils.Truncate(finalContent, 120)
			al.temporalIndex.Add(memory.TemporalEntry{
				Timestamp:  time.Now(),
				TopicTags:  extractTopicTags(opts.UserMessage),
				SessionKey: opts.SessionKey,
				Summary:    summary,
				Source:      opts.Channel,
			})
			// SWE100821: Persist temporal index to disk so data survives restarts
			if err := al.temporalIndex.Save(); err != nil {
				logger.WarnCF("temporal", "Failed to save temporal index",
					map[string]interface{}{"error": err.Error()})
			}
		}()
	}

	// SWE100821: Write daily note after each conversation turn.
	// Previously daily notes were only written from dream mode, creating a
	// chicken-and-egg problem: dreams need notes, but notes were only written by dreams.
	if !opts.NoHistory && finalContent != "" {
		postWg.Add(1)
		go func() {
			defer postWg.Done()
			noteEntry := fmt.Sprintf("## %s [%s]\n\n**User**: %s\n\n**Agent**: %s\n",
				time.Now().Format("15:04"),
				opts.Channel,
				utils.Truncate(opts.UserMessage, 200),
				utils.Truncate(finalContent, 300))
			if err := al.contextBuilder.memory.AppendToday(noteEntry); err != nil {
				logger.WarnCF("memory", "Failed to write daily note",
					map[string]interface{}{"error": err.Error()})
			}
		}()
	}

	// 13. Obsidian vault: write session note with wikilinks
	if al.vaultWriter != nil {
		postWg.Add(1)
		go func() {
			defer postWg.Done()
			prov := al.provenance.GetCurrent()
			vaultData := vault.SessionData{
				SessionKey:  opts.SessionKey,
				Channel:     opts.Channel,
				Model:       al.model,
				UserMessage: opts.UserMessage,
				Response:    finalContent,
				LatencyMs:   time.Since(turnStart).Milliseconds(),
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
		}()
	}

	// 14. SWE100821: Auto-store to semantic memory for vector recall
	if al.semanticMemory != nil && al.semanticMemory.IsAvailable() && !opts.NoHistory {
		postWg.Add(1)
		go func() {
			defer postWg.Done()
			storeCtx, storeCancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer storeCancel()
			storeSummary := fmt.Sprintf("User asked: %s. Agent responded: %s",
				utils.Truncate(opts.UserMessage, 200),
				utils.Truncate(finalContent, 200))
			if err := al.semanticMemory.StoreConversationSummary(storeCtx, opts.SessionKey, storeSummary); err != nil {
				logger.DebugCF("memory", "Semantic auto-store failed", map[string]interface{}{"error": err.Error()})
			}
		}()
	}

	postWg.Wait()

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
	// SWE100821: Also trigger hindsight reflection and memory consolidation after dreaming
	al.dream.SetDreamCallback(func(result DreamResult) {
		if al.vaultWriter != nil {
			if err := al.vaultWriter.WriteDreamNote(vault.DreamData{
				Insights:  result.Insights,
				Patterns:  result.Patterns,
				Questions: result.Questions,
				Timestamp: result.DreamedAt,
			}); err != nil {
				logger.WarnCF("vault", "Failed to write dream note",
					map[string]interface{}{"error": err.Error()})
			}
		}

		// SWE100821: Hindsight reflect — synthesize mental models from dream topics
		if len(result.Patterns) > 0 || len(result.Insights) > 0 {
			var topics []string
			for _, p := range result.Patterns {
				topics = append(topics, p)
			}
			if len(topics) > 3 {
				topics = topics[:3]
			}
			// SWE100821: Bounded context for post-dream work — prevents hung LLM from blocking forever
			reflectCtx, reflectCancel := context.WithTimeout(context.Background(), 5*time.Minute)
			go func() {
				defer reflectCancel()
				al.RunHindsightReflect(reflectCtx, topics)
			}()
		}

		// SWE100821: Memory consolidation — roll up daily notes into weekly/monthly
		consolidateCtx, consolidateCancel := context.WithTimeout(context.Background(), 5*time.Minute)
		go func() {
			defer consolidateCancel()
			al.RunConsolidation(consolidateCtx)
		}()
	})

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
	// SWE100821: Track last tool call signature to detect infinite loops.
	var lastToolSig string
	repeatCount := 0
	// SWE100821: Track tool calls and results for fallback response synthesis.
	// When the model exhausts iterations without producing text, we build a
	// summary from what actually happened instead of "no response to give."
	type toolLogEntry struct {
		Name   string
		Result string
		IsErr  bool
	}
	var toolLog []toolLogEntry

	// SWE100821: Use per-request cap if set (e.g. heartbeat), else global default
	maxIter := al.maxIterations
	if opts.MaxIterations > 0 {
		maxIter = opts.MaxIterations
	}

	for iteration < maxIter {
		iteration++

		logger.DebugCF("agent", "LLM iteration",
			map[string]interface{}{
				"iteration": iteration,
				"max":       maxIter,
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

		// SWE100821: Guard debug formatting — formatMessages/formatTools allocate heavily
		if logger.GetLevel() <= logger.DEBUG {
			logger.DebugCF("agent", "Full LLM request",
				map[string]interface{}{
					"iteration":     iteration,
					"messages_json": formatMessagesForLog(messages),
					"tools_json":    formatToolsForLog(providerToolDefs),
				})
		}

		// SWE100821: Use config values instead of hard-coded 8192/0.7
		llmStart := time.Now()
		response, err := al.provider.Chat(ctx, messages, providerToolDefs, al.model, map[string]interface{}{
			"max_tokens":  al.maxTokens,
			"temperature": al.temperature,
		})
		llmLatency := time.Since(llmStart)

		// SWE100821: Record LLM call metrics for dashboard System tab
		if al.metrics != nil {
			al.metrics.RecordLLMCall(llmLatency, err != nil)
		}

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
			// SWE100821: qwen2.5-coder emits tool calls as plain text content
			// ({"name":"skills","arguments":{...}}) instead of using the tool_calls
			// field. Try to parse content as a tool call and re-inject it.
			if parsed := parseContentAsToolCall(response.Content); parsed != nil {
				response.ToolCalls = []providers.ToolCall{*parsed}
				logger.InfoCF("agent", "Recovered tool call from content JSON",
					map[string]interface{}{"tool": parsed.Name, "iteration": iteration})
			} else {
				finalContent = response.Content
				logger.InfoCF("agent", "LLM response without tool calls (direct answer)",
					map[string]interface{}{
						"iteration":     iteration,
						"content_chars": len(finalContent),
					})
				break
			}
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

		// SWE100821: Detect tool call loops — if the model calls the same tool(s) with
		// the same args 3+ times in a row, break the loop and return whatever content
		// we have. Small models get stuck in infinite tool-call repetition.
		currentSig := ""
		for _, tc := range response.ToolCalls {
			argsJSON, _ := json.Marshal(tc.Arguments)
			currentSig += tc.Name + ":" + string(argsJSON) + ";"
		}
		if currentSig == lastToolSig {
			repeatCount++
			// SWE100821: Only break after 5 identical calls (was 3).
			// For phone tasks, small models need more attempts.
			if repeatCount >= 4 {
				logger.WarnCF("agent", "Tool call loop detected, breaking",
					map[string]interface{}{
						"tools":     toolNames,
						"repeats":   repeatCount + 1,
						"iteration": iteration,
					})
				if finalContent == "" && response.Content != "" {
					finalContent = response.Content
				}
				break
			}
		} else {
			repeatCount = 0
		}
		lastToolSig = currentSig

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

		// SWE100821: Intercept "response-as-tool-call" — llama3.1:8b never generates
		// plain text. It wraps every response as a tool call with a single text arg
		// (e.g. skills({response: "Here are the results..."})). Detect this and
		// capture the text as finalContent instead of executing the broken tool call.
		if captured := interceptResponseToolCall(response.ToolCalls); captured != "" {
			finalContent = captured
			logger.InfoCF("agent", "Intercepted response-as-tool-call",
				map[string]interface{}{"len": len(captured), "iteration": iteration})
			// Still need to add tool result messages so the conversation stays valid
			for _, tc := range response.ToolCalls {
				messages = append(messages, providers.Message{
					Role:       "tool",
					Content:    "OK",
					ToolCallID: tc.ID,
				})
			}
			break
		}

		// SWE100821: Execute tool calls in parallel (from PicoClaw toolloop.go pattern).
		// Tools within a single LLM response are independent — run concurrently,
		// reducing latency from sum(tools) to max(tools).
		type parallelToolResult struct {
			result  *tools.ToolResult
			latency time.Duration
		}
		parallelResults := make([]parallelToolResult, len(response.ToolCalls))

		// Log all tool calls before execution
		for _, tc := range response.ToolCalls {
			argsJSON, _ := json.Marshal(tc.Arguments)
			argsPreview := utils.Truncate(string(argsJSON), 200)
			logger.InfoCF("agent", fmt.Sprintf("Tool call: %s(%s)", tc.Name, argsPreview),
				map[string]interface{}{
					"tool":      tc.Name,
					"iteration": iteration,
				})
		}

		var toolWg sync.WaitGroup
		for i, tc := range response.ToolCalls {
			toolWg.Add(1)
			go func(idx int, tc providers.ToolCall) {
				defer toolWg.Done()
				start := time.Now()

				asyncCallback := func(callbackCtx context.Context, result *tools.ToolResult) {
					if !result.Silent && result.ForUser != "" {
						logger.InfoCF("agent", "Async tool completed, agent will handle notification",
							map[string]interface{}{
								"tool":        tc.Name,
								"content_len": len(result.ForUser),
							})
					}
				}

				var toolResult *tools.ToolResult
				if al.middleware != nil {
					toolResult = al.middleware.Execute(ctx, tc.Name, tc.Arguments, opts.Channel, opts.ChatID, asyncCallback)
				} else {
					toolResult = al.tools.ExecuteWithContext(ctx, tc.Name, tc.Arguments, opts.Channel, opts.ChatID, asyncCallback)
				}

				parallelResults[idx] = parallelToolResult{
					result:  toolResult,
					latency: time.Since(start),
				}
			}(i, tc)
		}
		toolWg.Wait()

		// Process results in order after all tools complete
		for i, tc := range response.ToolCalls {
			toolResult := parallelResults[i].result
			toolLatency := parallelResults[i].latency

			// SWE100821: Increment tool call counter for dashboard System tab
			if al.metrics != nil {
				al.metrics.IncToolCall()
			}
			al.provenance.RecordToolCall(tc.Name, !toolResult.IsError, toolLatency.Milliseconds())

			// SWE100821: Accumulate tool log for fallback response synthesis
			logResult := toolResult.ForLLM
			if len(logResult) > 200 {
				logResult = logResult[:200] + "..."
			}
			toolLog = append(toolLog, toolLogEntry{Name: tc.Name, Result: logResult, IsErr: toolResult.IsError})

			// SWE100821: Capture message tool content so dashboard chat gets a response.
			// llama3.1:8b uses varied param names — check all.
			if tc.Name == "message" {
				for _, key := range []string{"content", "message", "text", "data", "response"} {
					if contentVal, ok := tc.Arguments[key]; ok {
						if contentStr, ok := contentVal.(string); ok && contentStr != "" {
							finalContent = contentStr
							break
						}
					}
				}
			}

			// SWE100821: Small models (llama3.1:8b) wrap text responses as tool calls
			// with {type:"text", data:"..."} instead of just returning text. Catch this
			// pattern and extract the text as the final response.
			if typeVal, ok := tc.Arguments["type"]; ok {
				if typeStr, ok := typeVal.(string); ok && typeStr == "text" {
					if dataVal, ok := tc.Arguments["data"]; ok {
						if dataStr, ok := dataVal.(string); ok && dataStr != "" {
							finalContent = dataStr
							logger.InfoCF("agent", "Captured text-wrapped tool call as response",
								map[string]interface{}{"tool": tc.Name, "len": len(dataStr)})
						}
					}
				}
			}

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

			contentForLLM := toolResult.ForLLM
			if contentForLLM == "" && toolResult.Err != nil {
				contentForLLM = toolResult.Err.Error()
			}

			// SWE100821: Self-correction — when a tool call fails with "Unknown action",
			// append valid actions so the model corrects itself instead of looping.
			if toolResult.IsError && strings.Contains(contentForLLM, "Unknown action") {
				if toolDef, ok := al.tools.Get(tc.Name); ok {
					params := toolDef.Parameters()
					if props, ok := params["properties"].(map[string]interface{}); ok {
						if actionProp, ok := props["action"].(map[string]interface{}); ok {
							if enum, ok := actionProp["enum"].([]string); ok {
								contentForLLM += fmt.Sprintf("\n\nValid actions for %s: %s. Try a different approach — take a screenshot first to see the screen, then use tap/swipe/text to interact.", tc.Name, strings.Join(enum, ", "))
							}
						}
					}
				}
			}

			// SWE100821: Suggest skills when a tool call fails — agent can install them itself
			if toolResult.IsError && al.contextBuilder.autoDiscoverer != nil {
				if suggestion := al.contextBuilder.autoDiscoverer.SuggestForError(tc.Name, contentForLLM); suggestion != "" {
					contentForLLM += "\n\n" + suggestion
				}
			}

			toolResultMsg := providers.Message{
				Role:       "tool",
				Content:    contentForLLM,
				ToolCallID: tc.ID,
			}
			messages = append(messages, toolResultMsg)

			al.sessions.AddFullMessage(opts.SessionKey, toolResultMsg)

			// SWE100821: Plan-Act-Reflect — reflect after each tool result (sequential)
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

				planContext := plan.ForSystemPrompt()
				if planContext != "" && len(messages) > 0 {
					sysContent := messages[0].Content
					if idx := strings.Index(sysContent, "## Current Plan"); idx >= 0 {
						messages[0].Content = sysContent[:idx] + planContext
					} else {
						messages[0].Content += "\n\n" + planContext
					}
				}
			}
		}

		// SWE100821: Break when plan is complete — stops wasted iterations
		if plan != nil && plan.IsComplete() {
			logger.InfoCF("planner", "Plan complete, exiting iteration loop", nil)
			if finalContent == "" && response != nil && response.Content != "" {
				finalContent = response.Content
			}
			break
		}

		// SWE100821: Context window compaction — prevents context overflow on small models.
		// Every 8 iterations, compress older messages into a summary.
		// Keeps: preamble (system + volatile context/ack), recent messages, and a compressed summary.
		const compactEvery = 8
		const keepRecent = 6

		if iteration > 0 && iteration%compactEvery == 0 && len(messages) > keepRecent+4 {
			// SWE100821: Find preamble end dynamically — BuildMessages layout is:
			// [system, volatile_user?, volatile_ack?, ...history..., current_user, ...iteration_msgs]
			// oldStart must skip past all preamble messages to avoid compacting them.
			oldStart := 1 // skip system (idx 0)
			for oldStart < len(messages) {
				m := messages[oldStart]
				if m.Role == "user" && m.Content == "Understood, I have the updated context." {
					oldStart++
					continue
				}
				if m.Role == "assistant" && m.Content == "Understood, I have the updated context." {
					oldStart++
					continue
				}
				// Check for volatile context preamble (contains session/summary markers)
				if m.Role == "user" && (strings.Contains(m.Content, "## Summary of Previous") ||
					strings.Contains(m.Content, "## Current Session") ||
					strings.Contains(m.Content, "## Recent Activity")) {
					oldStart++
					continue
				}
				break
			}

			oldEnd := len(messages) - keepRecent
			if oldEnd > oldStart {
				var compactSummary strings.Builder
				compactSummary.WriteString("Previous actions summary:\n")
				for j := oldStart; j < oldEnd; j++ {
					m := messages[j]
					switch m.Role {
					case "assistant":
						if len(m.ToolCalls) > 0 {
							for _, tc := range m.ToolCalls {
								name := tc.Name
								if tc.Function != nil {
									name = tc.Function.Name
								}
								compactSummary.WriteString(fmt.Sprintf("- Called %s\n", name))
							}
						} else if m.Content != "" {
							trunc := []rune(m.Content)
							if len(trunc) > 100 {
								compactSummary.WriteString(fmt.Sprintf("- Said: %s...\n", string(trunc[:100])))
							} else {
								compactSummary.WriteString(fmt.Sprintf("- Said: %s\n", string(trunc)))
							}
						}
					case "tool":
						trunc := []rune(m.Content)
						if len(trunc) > 150 {
							compactSummary.WriteString(fmt.Sprintf("  Result: %s...\n", string(trunc[:150])))
						} else {
							compactSummary.WriteString(fmt.Sprintf("  Result: %s\n", string(trunc)))
						}
					}
				}

				compacted := make([]providers.Message, 0, oldStart+1+keepRecent)
				compacted = append(compacted, messages[:oldStart]...) // preamble
				compacted = append(compacted, providers.Message{
					Role:    "assistant",
					Content: compactSummary.String(),
				})
				compacted = append(compacted, messages[len(messages)-keepRecent:]...)
				oldLen := len(messages)
				messages = compacted

				logger.InfoCF("agent", "Context compacted",
					map[string]interface{}{
						"iteration":    iteration,
						"old_messages": oldLen,
						"new_messages": len(messages),
						"preamble":     oldStart,
					})
			}
		}
	}

	// SWE100821: If iterations exhausted with no text response, build a summary
	// from the tool call log instead of returning empty (which becomes
	// "I've completed processing but have no response to give.")
	if finalContent == "" && len(toolLog) > 0 {
		var sb strings.Builder
		sb.WriteString("Here's what I did:\n")
		for _, entry := range toolLog {
			status := "OK"
			if entry.IsErr {
				status = "failed"
			}
			sb.WriteString(fmt.Sprintf("- %s: %s", entry.Name, status))
			if entry.Result != "" && !entry.IsErr {
				sb.WriteString(fmt.Sprintf(" — %s", entry.Result))
			} else if entry.IsErr && entry.Result != "" {
				sb.WriteString(fmt.Sprintf(" (%s)", entry.Result))
			}
			sb.WriteString("\n")
		}
		finalContent = sb.String()
		logger.InfoCF("agent", "Built fallback response from tool log",
			map[string]interface{}{"tool_calls": len(toolLog), "len": len(finalContent)})
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

// DisablePlanner turns off the Plan-Act-Reflect loop to reduce LLM calls.
// SWE100821: On embedded/edge hardware, each LLM call costs 10-60s; the planner
// adds 2+ extra calls per message. Disabling it cuts response time by 40-60%.
func (al *AgentLoop) DisablePlanner() {
	al.plannerDisabled = true
}

// SWE100821: SetMetrics injects the health metrics pointer so the agent loop
// can increment message, LLM call, and tool call counters for the dashboard.
func (al *AgentLoop) SetMetrics(m *health.Metrics) {
	al.metrics = m
}

// SWE100821: PerceptionProvider abstracts the sensor monitor for system prompt injection.
type PerceptionProvider interface {
	ForSystemPrompt() string
}

// SWE100821: SetPerception injects the sensor monitor so readings appear in the agent's context.
func (al *AgentLoop) SetPerception(p PerceptionProvider) {
	al.perception = p
}

// SWE100821: EnableCompactPrompt strips the system prompt to ~50 tokens for
// PicoLM/embedded devices. Full prompt (~750 tokens) causes 3+ min prefill on ARM.
func (al *AgentLoop) EnableCompactPrompt() {
	al.contextBuilder.SetCompactPrompt(true)
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

// SWE100821: PruneSessions removes sessions older than maxAge from memory and disk.
func (al *AgentLoop) PruneSessions(maxAge time.Duration) int {
	return al.sessions.PruneStale(maxAge)
}

// SWE100821: interceptResponseToolCall detects when a small model wraps its text
// response as a tool call argument. Returns the extracted text, or "" if this
// looks like a legitimate tool call.
// Pattern: tool({response: "long text..."}) or tool({message: "long text..."})
// with no valid action/required params = the model is trying to respond, not act.
func interceptResponseToolCall(toolCalls []providers.ToolCall) string {
	if len(toolCalls) != 1 {
		return ""
	}
	tc := toolCalls[0]
	args := tc.Arguments

	// If it has a valid "action" param, it's a real tool call
	if _, hasAction := args["action"]; hasAction {
		return ""
	}

	// Check for response-like keys with substantial text content
	for _, key := range []string{"response", "message", "data", "text", "content", "answer", "result"} {
		if val, ok := args[key]; ok {
			if str, ok := val.(string); ok && len(str) > 20 {
				return str
			}
		}
	}

	return ""
}

// SWE100821: parseContentAsToolCall detects when a model emits a tool call as
// plain text content (e.g. {"name":"skills","arguments":{"action":"search",...}}).
// Returns a reconstructed ToolCall if the pattern matches, nil otherwise.
func parseContentAsToolCall(content string) *providers.ToolCall {
	content = strings.TrimSpace(content)
	if !strings.HasPrefix(content, "{") {
		return nil
	}
	var parsed struct {
		Name      string                 `json:"name"`
		Arguments map[string]interface{} `json:"arguments"`
	}
	if err := json.Unmarshal([]byte(content), &parsed); err != nil {
		return nil
	}
	if parsed.Name == "" || parsed.Arguments == nil {
		return nil
	}
	argsJSON, _ := json.Marshal(parsed.Arguments)
	return &providers.ToolCall{
		ID:   fmt.Sprintf("recovered-%d", time.Now().UnixNano()),
		Name: parsed.Name,
		Arguments: parsed.Arguments,
		Function: &providers.FunctionCall{
			Name:      parsed.Name,
			Arguments: string(argsJSON),
		},
	}
}

// SWE100821: unwrapJSONResponse strips JSON wrappers that small models add.
// Handles: {"data":"text"}, {"content":"text"}, {"name":"tool","arguments":{"response":"text"}}
func unwrapJSONResponse(s string) string {
	s = strings.TrimSpace(s)
	if !strings.HasPrefix(s, "{") || !strings.HasSuffix(s, "}") {
		return s
	}
	var wrapper map[string]interface{}
	if err := json.Unmarshal([]byte(s), &wrapper); err != nil {
		return s
	}
	// Pattern: {"name":"...", "arguments":{"response":"text"}} — tool-call-as-response
	if args, ok := wrapper["arguments"].(map[string]interface{}); ok {
		for _, key := range []string{"response", "message", "text", "content", "answer", "result", "data"} {
			if val, ok := args[key]; ok {
				if str, ok := val.(string); ok && len(str) > 5 {
					return str
				}
			}
		}
	}
	// Pattern: {"data":"text"} or {"content":"text"}
	for _, key := range []string{"data", "content", "text", "message", "response"} {
		if val, ok := wrapper[key]; ok {
			if str, ok := val.(string); ok && str != "" {
				return str
			}
		}
	}
	return s
}

// SWE100821: resolveMessageTimeout converts config seconds to Duration with sensible defaults.
// 0 → 15 minutes (safe for ARM/embedded), explicit value used as-is.
func resolveMessageTimeout(secs int) time.Duration {
	if secs > 0 {
		return time.Duration(secs) * time.Second
	}
	return 15 * time.Minute
}

// SWE100821: resolveContextWindow separates context window (model capacity) from max_tokens (output cap).
// Priority: explicit context_window > max_tokens*4 heuristic > 8192 default.
func resolveContextWindow(contextWindow, maxTokens int) int {
	if contextWindow > 0 {
		return contextWindow
	}
	if maxTokens > 0 {
		return maxTokens * 4
	}
	return 8192
}

// SWE100821: strings.Builder — was using result += (O(n²) allocation)
func formatMessagesForLog(messages []providers.Message) string {
	if len(messages) == 0 {
		return "[]"
	}
	var sb strings.Builder
	sb.Grow(len(messages) * 120)
	sb.WriteString("[\n")
	for i, msg := range messages {
		fmt.Fprintf(&sb, "  [%d] Role: %s\n", i, msg.Role)
		if len(msg.ToolCalls) > 0 {
			sb.WriteString("  ToolCalls:\n")
			for _, tc := range msg.ToolCalls {
				fmt.Fprintf(&sb, "    - ID: %s, Type: %s, Name: %s\n", tc.ID, tc.Type, tc.Name)
				if tc.Function != nil {
					fmt.Fprintf(&sb, "      Arguments: %s\n", utils.Truncate(tc.Function.Arguments, 200))
				}
			}
		}
		if msg.Content != "" {
			fmt.Fprintf(&sb, "  Content: %s\n", utils.Truncate(msg.Content, 200))
		}
		if msg.ToolCallID != "" {
			fmt.Fprintf(&sb, "  ToolCallID: %s\n", msg.ToolCallID)
		}
		sb.WriteByte('\n')
	}
	sb.WriteByte(']')
	return sb.String()
}

// SWE100821: strings.Builder — was using result += (O(n²) allocation)
func formatToolsForLog(tools []providers.ToolDefinition) string {
	if len(tools) == 0 {
		return "[]"
	}
	var sb strings.Builder
	sb.Grow(len(tools) * 100)
	sb.WriteString("[\n")
	for i, tool := range tools {
		fmt.Fprintf(&sb, "  [%d] Type: %s, Name: %s\n", i, tool.Type, tool.Function.Name)
		fmt.Fprintf(&sb, "      Description: %s\n", tool.Function.Description)
		if len(tool.Function.Parameters) > 0 {
			fmt.Fprintf(&sb, "      Parameters: %s\n", utils.Truncate(fmt.Sprintf("%v", tool.Function.Parameters), 200))
		}
	}
	sb.WriteByte(']')
	return sb.String()
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

	// SWE100821: Include tool messages in summarization — condense to "Tool: name → result_preview"
	// so the summary retains what actions were taken and their outcomes.
	for _, m := range toSummarize {
		msgTokens := len(m.Content) / 4
		if msgTokens > maxMessageTokens {
			omitted = true
			continue
		}
		switch m.Role {
		case "user", "assistant":
			validMessages = append(validMessages, m)
		case "tool":
			condensed := m.Content
			if len(condensed) > 200 {
				condensed = condensed[:200] + "..."
			}
			validMessages = append(validMessages, providers.Message{
				Role:    "user",
				Content: fmt.Sprintf("[Tool Result] %s", condensed),
			})
		}
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
		if err := al.sessions.Save(sessionKey); err != nil {
			logger.WarnCF("session", "Failed to save summarized session",
				map[string]interface{}{"session": sessionKey, "error": err.Error()})
		}
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

// SWE100821: GetToolNames returns all registered tool names for dashboard display.
func (al *AgentLoop) GetToolNames() []string {
	if al.tools == nil {
		return nil
	}
	return al.tools.List()
}

// SWE100821: extractTopicTags pulls simple topic keywords from a message for temporal indexing.
// Lightweight — no LLM call. Extracts nouns/keywords longer than 3 chars.
func extractTopicTags(msg string) []string {
	words := strings.Fields(strings.ToLower(msg))
	seen := make(map[string]bool)
	var tags []string
	stopWords := map[string]bool{
		"the": true, "and": true, "for": true, "are": true, "but": true,
		"not": true, "you": true, "all": true, "can": true, "her": true,
		"was": true, "one": true, "our": true, "out": true, "has": true,
		"have": true, "that": true, "this": true, "with": true, "from": true,
		"they": true, "been": true, "said": true, "each": true, "which": true,
		"their": true, "will": true, "other": true, "about": true, "many": true,
		"then": true, "them": true, "these": true, "some": true, "would": true,
		"make": true, "like": true, "what": true, "when": true, "your": true,
		"could": true, "there": true, "into": true, "just": true, "also": true,
	}
	for _, w := range words {
		w = strings.Trim(w, ".,!?;:'\"()-[]{}") 
		if len(w) <= 3 || stopWords[w] || seen[w] {
			continue
		}
		seen[w] = true
		tags = append(tags, w)
		if len(tags) >= 5 {
			break
		}
	}
	return tags
}

// SWE100821: RunConsolidation triggers memory consolidation (weekly + monthly rollup).
// Called from gateway cron or dream mode.
func (al *AgentLoop) RunConsolidation(ctx context.Context) {
	if al.consolidator == nil {
		return
	}
	if err := al.consolidator.ConsolidateWeekly(ctx); err != nil {
		logger.WarnCF("consolidation", "Weekly consolidation failed", map[string]interface{}{"error": err.Error()})
	}
	if err := al.consolidator.ConsolidateMonthly(ctx); err != nil {
		logger.WarnCF("consolidation", "Monthly consolidation failed", map[string]interface{}{"error": err.Error()})
	}
}

// SWE100821: RunHindsightReflect reflects on key topics from dream insights.
// Called after dream mode produces results to synthesize mental models.
func (al *AgentLoop) RunHindsightReflect(ctx context.Context, topics []string) {
	if al.hindsight == nil {
		return
	}
	for _, topic := range topics {
		if err := al.hindsight.Reflect(ctx, topic); err != nil {
			logger.WarnCF("hindsight", "Reflection failed", map[string]interface{}{
				"topic": topic, "error": err.Error(),
			})
		}
	}
}

// SWE100821: SendProactive sends a proactive message to the last active channel.
// Used by dream mode, sensor alerts, and idle curiosity.
func (al *AgentLoop) SendProactive(content string) {
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
		Content: content,
	})
}
