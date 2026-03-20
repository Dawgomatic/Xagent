package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/Dawgomatic/Xagent/pkg/agent"
	"github.com/Dawgomatic/Xagent/pkg/agent2agent"
	"github.com/Dawgomatic/Xagent/pkg/bus"
	"github.com/Dawgomatic/Xagent/pkg/channels"
	"github.com/Dawgomatic/Xagent/pkg/dashboard" // SWE100821: Cognitive dashboard
	"github.com/Dawgomatic/Xagent/pkg/devices"
	"github.com/Dawgomatic/Xagent/pkg/epoch"
	"github.com/Dawgomatic/Xagent/pkg/health"
	"github.com/Dawgomatic/Xagent/pkg/heartbeat"
	"github.com/Dawgomatic/Xagent/pkg/hwprofile"
	"github.com/Dawgomatic/Xagent/pkg/logger"
	"github.com/Dawgomatic/Xagent/pkg/providers"
	"github.com/Dawgomatic/Xagent/pkg/state"
	"github.com/Dawgomatic/Xagent/pkg/tools"
	"github.com/Dawgomatic/Xagent/pkg/voice"
)

func gatewayCmd() {
	// Check for --debug flag
	args := os.Args[2:]
	for _, arg := range args {
		if arg == "--debug" || arg == "-d" {
			logger.SetLevel(logger.DEBUG)
			fmt.Println("🔍 Debug mode enabled")
			break
		}
	}

	cfg, err := loadConfig()
	if err != nil {
		fmt.Printf("Error loading config: %v\n", err)
		os.Exit(1)
	}

	// SWE100821: Validate config on startup to surface misconfigurations early
	if warnings, valErr := cfg.Validate(); valErr != nil {
		fmt.Printf("❌ Config validation failed: %v\n", valErr)
		os.Exit(1)
	} else {
		for _, w := range warnings {
			fmt.Printf("⚠ Config warning: %s\n", w)
		}
	}

	// SWE100821: Autonomous hardware detection + adaptive scaling
	hwProfile := hwprofile.Detect()
	rec := hwProfile.Recommend()
	fmt.Printf("\n🔧 Hardware Profile: %s\n", hwProfile.Summary())
	fmt.Printf("  • Tier: %s → recommended provider: %s, model: %s\n", hwProfile.Tier, rec.Provider, rec.OllamaModel)

	// Auto-tune config from hardware profile (only if not explicitly overridden)
	if cfg.Agents.Defaults.MaxTokens == 8192 { // default value = not explicitly set
		cfg.Agents.Defaults.MaxTokens = rec.MaxTokens
		fmt.Printf("  • Auto-tuned max_tokens: %d\n", rec.MaxTokens)
	}
	if cfg.Agents.Defaults.MaxToolIterations == 20 { // default
		cfg.Agents.Defaults.MaxToolIterations = rec.MaxToolIterations
		fmt.Printf("  • Auto-tuned max_tool_iterations: %d\n", rec.MaxToolIterations)
	}

	// SWE100821: Auto-switch to local provider on embedded platforms when no provider is explicitly set
	if rec.Provider == "picolm" && cfg.Agents.Defaults.Provider == "" {
		cfg.Providers.PicoLM.Enabled = true
		cfg.Agents.Defaults.Provider = "picolm"
		cfg.Agents.Defaults.Model = "picolm-local"
		// Set KV cache path for system prompt reuse
		if cfg.Providers.PicoLM.CachePath == "" {
			cfg.Providers.PicoLM.CachePath = cfg.WorkspacePath() + "/picolm-system.kvc"
		}
		// Match thread count to detected cores
		if cfg.Providers.PicoLM.Threads <= 0 || cfg.Providers.PicoLM.Threads == 4 {
			cfg.Providers.PicoLM.Threads = hwProfile.CPUCores
		}
		fmt.Printf("  • Auto-switched to PicoLM (local-first, %d threads, KV cache enabled)\n", cfg.Providers.PicoLM.Threads)
	}

	provider, err := providers.CreateProvider(cfg)
	if err != nil {
		fmt.Printf("Error creating provider: %v\n", err)
		os.Exit(1)
	}

	msgBus := bus.NewMessageBus()
	agentLoop := agent.NewAgentLoop(cfg, msgBus, provider)

	// SWE100821: Disable planner on embedded to eliminate 2+ LLM calls per message
	if rec.DisablePlanner {
		agentLoop.DisablePlanner()
		fmt.Println("  • Planner disabled (embedded mode — single LLM call per message)")
		// SWE100821: Only enable compact prompt for PicoLM where prefill cost dominates.
		// Ollama models (llama3.1, phi3, etc.) need the full system prompt for proper
		// tool usage, identity, and response formatting.
		if cfg.Agents.Defaults.Provider == "picolm" {
			agentLoop.EnableCompactPrompt()
			fmt.Println("  • Compact prompt enabled (minimal system prompt for fast prefill)")
		}
	}

	// SWE100821: Start resource watcher — dynamically switch model when tier changes
	// Longer interval on embedded to reduce overhead (300s vs 60s)
	watchInterval := 60 * time.Second
	if rec.DisablePlanner {
		watchInterval = 300 * time.Second
	}
	stopWatch := hwprofile.WatchResources(watchInterval, func(old, cur *hwprofile.Profile) {
		logger.WarnCF("hwprofile", "Compute tier changed",
			map[string]interface{}{
				"old_tier":     string(old.Tier),
				"new_tier":     string(cur.Tier),
				"ram_avail_mb": cur.RAMAvailMB,
			})
		newRec := cur.Recommend()
		oldModel := agentLoop.GetModel()
		if newRec.OllamaModel != oldModel {
			agentLoop.SetModel(newRec.OllamaModel)
			logger.InfoCF("hwprofile", "Dynamic model switch",
				map[string]interface{}{
					"old_model": oldModel,
					"new_model": newRec.OllamaModel,
					"reason":    fmt.Sprintf("tier changed %s -> %s", old.Tier, cur.Tier),
				})
		}
	})
	defer stopWatch()

	// Print agent startup info
	fmt.Println("\n📦 Agent Status:")
	startupInfo := agentLoop.GetStartupInfo()
	toolsInfo := startupInfo["tools"].(map[string]interface{})
	skillsInfo := startupInfo["skills"].(map[string]interface{})
	identityInfo := startupInfo["identity"].(map[string]interface{})
	// SWE100821: Display unique agent identity + boot time at startup
	fmt.Printf("  • Agent ID:   %s\n", identityInfo["agent_id"])
	fmt.Printf("  • Session ID: %s\n", identityInfo["session_id"])
	fmt.Printf("  • Boot time:  %s\n", identityInfo["boot_time"])
	fmt.Printf("  • Tools: %d loaded\n", toolsInfo["count"])
	fmt.Printf("  • Skills: %d/%d available\n",
		skillsInfo["available"],
		skillsInfo["total"])

	// Log to file as well
	logger.InfoCF("agent", "Agent initialized",
		map[string]interface{}{
			"tools_count":      toolsInfo["count"],
			"skills_total":     skillsInfo["total"],
			"skills_available": skillsInfo["available"],
		})

	// SWE100821: Epoch lifecycle — wake up and remember the previous session
	epochManager := epoch.NewManager(cfg.WorkspacePath(), agentLoop.GetIdentity())
	prevEpoch, _ := epochManager.Wake()
	agentLoop.SetEpoch(epochManager)
	agentLoop.SetPreviousEpoch(prevEpoch)
	if prevEpoch != nil && prevEpoch.ShutdownTime != nil {
		fmt.Printf("  • Last epoch: %s (up %s, %d msgs)\n",
			prevEpoch.BootTime.Format("2006-01-02 15:04"),
			prevEpoch.Uptime,
			prevEpoch.Stats.MessagesProcessed)
	} else {
		fmt.Println("  • First epoch (no previous session)")
	}
	// Prune old epochs (keep last 30, delete anything older than 90 days)
	if pruned := epochManager.PruneOld(90*24*time.Hour, 30); pruned > 0 {
		logger.InfoCF("epoch", "Pruned old epochs", map[string]interface{}{"pruned": pruned})
	}

	// Setup cron tool and service
	cronService := setupCronTool(agentLoop, msgBus, cfg.WorkspacePath())

	heartbeatService := heartbeat.NewHeartbeatService(
		cfg.WorkspacePath(),
		cfg.Heartbeat.Interval,
		cfg.Heartbeat.Enabled,
	)
	heartbeatService.SetBus(msgBus)
	heartbeatService.SetHandler(func(prompt, channel, chatID string) *tools.ToolResult {
		// Use cli:direct as fallback if no valid channel
		if channel == "" || chatID == "" {
			channel, chatID = "cli", "direct"
		}
		// Use ProcessHeartbeat - no session history, each heartbeat is independent
		response, err := agentLoop.ProcessHeartbeat(context.Background(), prompt, channel, chatID)
		if err != nil {
			return tools.ErrorResult(fmt.Sprintf("Heartbeat error: %v", err))
		}
		if response == "HEARTBEAT_OK" {
			return tools.SilentResult("Heartbeat OK")
		}
		// For heartbeat, always return silent - the subagent result will be
		// sent to user via processSystemMessage when the async task completes
		return tools.SilentResult(response)
	})

	channelManager, err := channels.NewManager(cfg, msgBus)
	if err != nil {
		fmt.Printf("Error creating channel manager: %v\n", err)
		os.Exit(1)
	}

	var transcriber *voice.GroqTranscriber
	if cfg.Providers.Groq.APIKey != "" {
		transcriber = voice.NewGroqTranscriber(cfg.Providers.Groq.APIKey)
		logger.InfoC("voice", "Groq voice transcription enabled")
	}

	if transcriber != nil {
		if telegramChannel, ok := channelManager.GetChannel("telegram"); ok {
			if tc, ok := telegramChannel.(*channels.TelegramChannel); ok {
				tc.SetTranscriber(transcriber)
				logger.InfoC("voice", "Groq transcription attached to Telegram channel")
			}
		}
		if discordChannel, ok := channelManager.GetChannel("discord"); ok {
			if dc, ok := discordChannel.(*channels.DiscordChannel); ok {
				dc.SetTranscriber(transcriber)
				logger.InfoC("voice", "Groq transcription attached to Discord channel")
			}
		}
		if slackChannel, ok := channelManager.GetChannel("slack"); ok {
			if sc, ok := slackChannel.(*channels.SlackChannel); ok {
				sc.SetTranscriber(transcriber)
				logger.InfoC("voice", "Groq transcription attached to Slack channel")
			}
		}
	}

	enabledChannels := channelManager.GetEnabledChannels()
	if len(enabledChannels) > 0 {
		fmt.Printf("✓ Channels enabled: %s\n", enabledChannels)
	} else {
		fmt.Println("⚠ Warning: No channels enabled")
	}

	fmt.Printf("✓ Gateway started on %s:%d\n", cfg.Gateway.Host, cfg.Gateway.Port)

	// SWE100821: Health check server with Ollama readiness probe and metrics
	healthPort := cfg.Gateway.Port + 1
	var checkers []health.ReadinessChecker
	if cfg.Providers.VLLM.APIBase != "" {
		// Strip /v1 suffix for Ollama root ping
		ollamaBase := strings.TrimSuffix(cfg.Providers.VLLM.APIBase, "/v1")
		checkers = append(checkers, health.OllamaChecker(ollamaBase))
	}
	healthServer := health.NewServer(cfg.Gateway.Host, healthPort, checkers...)

	// SWE100821: Wire A2A protocol into gateway
	a2aHub := agent2agent.NewA2AHub(agentLoop.GetIdentity().AgentID)
	a2aHub.SetHandler(func(ctx context.Context, msg agent2agent.A2AMessage) (string, error) {
		return agentLoop.ProcessDirect(ctx, msg.Payload, "a2a:"+msg.FromAgentID)
	})
	healthServer.RegisterHandler("/a2a", a2aHub.HTTPHandler())
	fmt.Println("✓ A2A protocol enabled on /a2a")

	// SWE100821: Wire interactive cognitive dashboard into health server mux
	dash := dashboard.NewDashboard(cfg.WorkspacePath())
	// SWE100821: Pass vault path, config path, and metrics for interactive features
	if cfg.Vault.Enabled && cfg.Vault.Path != "" {
		vaultPath := cfg.Vault.Path
		if strings.HasPrefix(vaultPath, "~/") {
			if home, err := os.UserHomeDir(); err == nil {
				vaultPath = filepath.Join(home, vaultPath[2:])
			}
		}
		dash.SetVaultPath(vaultPath)
	}
	dash.SetConfigPath(getConfigPath())
	dash.SetMetrics(healthServer.GetMetrics())
	// SWE100821: Feed live model + tier to dashboard overview cards
	dash.SetModelInfo(cfg.Agents.Defaults.Model, string(hwProfile.Tier))
	// SWE100821: Wire metrics into agent loop so LLM calls, tool calls, and messages
	// are tracked and visible in the dashboard System tab.
	agentLoop.SetMetrics(healthServer.GetMetrics())
	// SWE100821: Wire chat handler — allows dashboard to send messages to the agent
	dash.SetChatHandler(func(ctx context.Context, message, sessionKey string) (string, error) {
		return agentLoop.ProcessDirect(ctx, message, sessionKey)
	})
	if mux := healthServer.GetMux(); mux != nil {
		dash.SetupRoutes(mux)
		fmt.Println("✓ Interactive dashboard enabled on /dashboard (chat, graph, memory, vault)")
	}

	healthServer.Start()
	healthServer.SetReady(true)
	fmt.Printf("✓ Health check server on %s:%d (/healthz, /readyz, /metricsz, /hwprofile, /a2a, /dashboard)\n", cfg.Gateway.Host, healthPort)

	fmt.Println("Press Ctrl+C to stop")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	epochManager.StartRolloverMonitor(ctx, 24*time.Hour)
	fmt.Println("✓ Epoch 24h rollover monitor started")

	if err := cronService.Start(); err != nil {
		fmt.Printf("Error starting cron service: %v\n", err)
	}
	fmt.Println("✓ Cron service started")

	if err := heartbeatService.Start(); err != nil {
		fmt.Printf("Error starting heartbeat service: %v\n", err)
	}
	fmt.Println("✓ Heartbeat service started")

	stateManager := state.NewManager(cfg.WorkspacePath())
	deviceService := devices.NewService(devices.Config{
		Enabled:    cfg.Devices.Enabled,
		MonitorUSB: cfg.Devices.MonitorUSB,
	}, stateManager)
	deviceService.SetBus(msgBus)
	if err := deviceService.Start(ctx); err != nil {
		fmt.Printf("Error starting device service: %v\n", err)
	} else if cfg.Devices.Enabled {
		fmt.Println("✓ Device event service started")
	}

	if err := channelManager.StartAll(ctx); err != nil {
		fmt.Printf("Error starting channels: %v\n", err)
	}

	go agentLoop.Run(ctx)

	// SWE100821: Start dream mode — autonomous reflection during idle periods.
	// After 2h idle, the agent reviews recent conversations, finds patterns,
	// and updates its world model. Runs every 12h.
	agentLoop.StartDreamMode(ctx)

	// SWE100821: Subsystem watchdog — monitors all services, auto-recovers external deps
	watchdog := health.NewWatchdog(30 * time.Second)

	// Gateway self-check (dashboard lives here)
	dashboardURL := fmt.Sprintf("http://127.0.0.1:%d/healthz", healthPort)
	watchdog.Register(health.HTTPChecker("dashboard", dashboardURL, nil))

	// Ollama LLM backend
	if cfg.Providers.VLLM.APIBase != "" {
		ollamaBase := strings.TrimSuffix(cfg.Providers.VLLM.APIBase, "/v1")
		watchdog.Register(health.HTTPChecker("ollama", ollamaBase, health.SystemdRecover("ollama")))
	}

	// Qdrant vector DB (if configured)
	if cfg.SemanticMemory.QdrantURL != "" {
		watchdog.Register(health.HTTPChecker("qdrant", cfg.SemanticMemory.QdrantURL, health.SystemdRecover("qdrant")))
	}

	// Heartbeat service
	watchdog.Register(health.CallbackChecker("heartbeat", func() error {
		if !heartbeatService.IsRunning() {
			return fmt.Errorf("heartbeat stopped")
		}
		return nil
	}))

	// Agent loop
	watchdog.Register(health.CallbackChecker("agent_loop", func() error {
		if !agentLoop.IsRunning() {
			return fmt.Errorf("agent loop stopped")
		}
		return nil
	}))

	// SWE100821: Cron scheduler
	watchdog.Register(health.CallbackChecker("cron", func() error {
		if !cronService.IsRunning() {
			return fmt.Errorf("cron stopped")
		}
		return nil
	}))

	// SWE100821: Sleep/fatigue manager
	watchdog.Register(health.CallbackChecker("sleep_fatigue", func() error {
		if !agentLoop.IsSleepRunning() {
			return fmt.Errorf("sleep manager stopped")
		}
		fatigue := agentLoop.GetFatigueLevel()
		if fatigue >= 0.9 {
			return fmt.Errorf("fatigue critical: %.0f%%", fatigue*100)
		}
		return nil
	}))

	// SWE100821: Channels (Discord, Telegram, etc.)
	watchdog.Register(health.CallbackChecker("channels", func() error {
		enabled := channelManager.GetEnabledChannels()
		if len(enabled) == 0 {
			return fmt.Errorf("no channels enabled")
		}
		return nil
	}))

	// SWE100821: Device service (USB monitoring)
	if cfg.Devices.Enabled {
		watchdog.Register(health.CallbackChecker("devices", func() error {
			if !deviceService.IsRunning() {
				return fmt.Errorf("device service stopped")
			}
			return nil
		}))
	}

	watchdog.Start()
	dash.SetWatchdog(watchdog)
	fmt.Println("✓ Subsystem watchdog started (30s interval)")

	// SWE100821: Handle both SIGINT (Ctrl+C) and SIGTERM (systemctl stop)
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	fmt.Println("\nShutting down...")
	healthServer.SetReady(false) // SWE100821: Signal not-ready before teardown

	// SWE100821: Epoch sleep — journal what happened this session before shutdown
	epochManager.UpdateStats(func(s *epoch.EpochStats) {
		s.SessionsActive = agentLoop.GetSessionStats()
	})
	uptime := agentLoop.GetIdentity().Uptime().Truncate(time.Second)
	reflection := fmt.Sprintf("Agent ran for %s. Shutting down gracefully.", uptime)
	if err := epochManager.Sleep(reflection); err != nil {
		logger.ErrorCF("epoch", "Failed to write epoch journal", map[string]interface{}{"error": err.Error()})
	} else {
		fmt.Println("✓ Epoch journal saved")
	}

	cancel()
	watchdog.Stop()
	deviceService.Stop()
	heartbeatService.Stop()
	cronService.Stop()
	agentLoop.Stop()
	channelManager.StopAll(ctx)
	healthServer.Stop()
	fmt.Println("✓ Gateway stopped")
}
