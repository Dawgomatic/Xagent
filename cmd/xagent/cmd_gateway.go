package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/Dawgomatic/Xagent/pkg/agent"
	"github.com/Dawgomatic/Xagent/pkg/bus"
	"github.com/Dawgomatic/Xagent/pkg/channels"
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
	fmt.Printf("  • Tier: %s → recommended model: %s\n", hwProfile.Tier, rec.OllamaModel)

	// Auto-tune config from hardware profile (only if not explicitly overridden)
	if cfg.Agents.Defaults.MaxTokens == 8192 { // default value = not explicitly set
		cfg.Agents.Defaults.MaxTokens = rec.MaxTokens
		fmt.Printf("  • Auto-tuned max_tokens: %d\n", rec.MaxTokens)
	}
	if cfg.Agents.Defaults.MaxToolIterations == 20 { // default
		cfg.Agents.Defaults.MaxToolIterations = rec.MaxToolIterations
		fmt.Printf("  • Auto-tuned max_tool_iterations: %d\n", rec.MaxToolIterations)
	}

	// Start resource watcher for dynamic tier changes (e.g., RAM pressure)
	stopWatch := hwprofile.WatchResources(60*time.Second, func(old, cur *hwprofile.Profile) {
		logger.WarnCF("hwprofile", "Compute tier changed",
			map[string]interface{}{
				"old_tier":     string(old.Tier),
				"new_tier":     string(cur.Tier),
				"ram_avail_mb": cur.RAMAvailMB,
			})
	})
	defer stopWatch()

	provider, err := providers.CreateProvider(cfg)
	if err != nil {
		fmt.Printf("Error creating provider: %v\n", err)
		os.Exit(1)
	}

	msgBus := bus.NewMessageBus()
	agentLoop := agent.NewAgentLoop(cfg, msgBus, provider)

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
	healthServer.Start()
	healthServer.SetReady(true)
	fmt.Printf("✓ Health check server on %s:%d (/healthz, /readyz, /metricsz, /hwprofile)\n", cfg.Gateway.Host, healthPort)

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
	deviceService.Stop()
	heartbeatService.Stop()
	cronService.Stop()
	agentLoop.Stop()
	channelManager.StopAll(ctx)
	healthServer.Stop()
	fmt.Println("✓ Gateway stopped")
}
