package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/Dawgomatic/Xagent/pkg/auth"
	"github.com/Dawgomatic/Xagent/pkg/hwprofile"
	"github.com/Dawgomatic/Xagent/pkg/identity"
)

func statusCmd() {
	cfg, err := loadConfig()
	if err != nil {
		fmt.Printf("Error loading config: %v\n", err)
		return
	}

	configPath := getConfigPath()

	fmt.Printf("%s xagent Status\n", logo)
	fmt.Printf("Version: %s\n", formatVersion())
	build, _ := formatBuildInfo()
	if build != "" {
		fmt.Printf("Build: %s\n", build)
	}
	// SWE100821: Show hardware tier in status output
	hwp := hwprofile.Detect()
	fmt.Printf("Hardware: %s (tier=%s, %dMB RAM)\n", hwp.Platform, hwp.Tier, hwp.RAMTotalMB)

	// SWE100821: Show persistent agent identity + birth time
	agentID := identity.New(cfg.WorkspacePath())
	fmt.Printf("Agent ID: %s\n", agentID.AgentID)
	fmt.Printf("Birth: %s (age: %s)\n", agentID.BirthTime.Format("2006-01-02 15:04:05"), agentID.Age().Truncate(time.Second))
	fmt.Println()

	if _, err := os.Stat(configPath); err == nil {
		fmt.Println("Config:", configPath, "✓")
	} else {
		fmt.Println("Config:", configPath, "✗")
	}

	workspace := cfg.WorkspacePath()
	if _, err := os.Stat(workspace); err == nil {
		fmt.Println("Workspace:", workspace, "✓")
	} else {
		fmt.Println("Workspace:", workspace, "✗")
	}

	if _, err := os.Stat(configPath); err == nil {
		fmt.Printf("Model: %s\n", cfg.Agents.Defaults.Model)

		hasOpenRouter := cfg.Providers.OpenRouter.APIKey != ""
		hasAnthropic := cfg.Providers.Anthropic.APIKey != ""
		hasOpenAI := cfg.Providers.OpenAI.APIKey != ""
		hasGemini := cfg.Providers.Gemini.APIKey != ""
		// SWE100821: Zhipu provider removed (Chinese service)
		hasNvidia := cfg.Providers.Nvidia.APIKey != ""
		hasGroq := cfg.Providers.Groq.APIKey != ""
		hasVLLM := cfg.Providers.VLLM.APIBase != ""

		status := func(enabled bool) string {
			if enabled {
				return "✓"
			}
			return "not set"
		}
		fmt.Println("OpenRouter API:", status(hasOpenRouter))
		fmt.Println("Anthropic API:", status(hasAnthropic))
		fmt.Println("OpenAI API:", status(hasOpenAI))
		fmt.Println("Gemini API:", status(hasGemini))
		fmt.Println("Nvidia API:", status(hasNvidia))
		fmt.Println("Groq API:", status(hasGroq))
		if hasVLLM {
			fmt.Printf("vLLM/Local: ✓ %s\n", cfg.Providers.VLLM.APIBase)
		} else {
			fmt.Println("vLLM/Local: not set")
		}

		store, _ := auth.LoadStore()
		if store != nil && len(store.Credentials) > 0 {
			fmt.Println("\nOAuth/Token Auth:")
			for provider, cred := range store.Credentials {
				status := "authenticated"
				if cred.IsExpired() {
					status = "expired"
				} else if cred.NeedsRefresh() {
					status = "needs refresh"
				}
				fmt.Printf("  %s (%s): %s\n", provider, cred.AuthMethod, status)
			}
		}
	}
}

func hwprofileCmd() {
	p := hwprofile.Detect()
	rec := p.Recommend()

	fmt.Printf("%s Hardware Profile\n", logo)
	fmt.Println()
	fmt.Printf("Platform:    %s (%s)\n", p.Platform, p.Arch)
	fmt.Printf("CPU:         %s (%d cores, %d MHz)\n", p.CPUModel, p.CPUCores, p.CPUFreqMHz)
	fmt.Printf("RAM:         %d MB total, %d MB available\n", p.RAMTotalMB, p.RAMAvailMB)
	fmt.Printf("Disk free:   %d MB\n", p.DiskFreeMB)

	if p.GPU.Detected {
		fmt.Printf("GPU:         %s (%d MB VRAM, driver: %s)\n", p.GPU.Name, p.GPU.VRAM_MB, p.GPU.Driver)
	} else {
		fmt.Println("GPU:         not detected")
	}

	fmt.Println()
	fmt.Printf("Compute Tier: %s\n", p.Tier)
	fmt.Println()
	fmt.Println("Recommendations:")
	fmt.Printf("  Model:              %s\n", rec.OllamaModel)
	fmt.Printf("  Max tokens:         %d\n", rec.MaxTokens)
	fmt.Printf("  Temperature:        %.1f\n", rec.Temperature)
	fmt.Printf("  Max tool iters:     %d\n", rec.MaxToolIterations)
	fmt.Printf("  Max subagents:      %d\n", rec.MaxSubagents)
	fmt.Printf("  Message timeout:    %ds\n", rec.MessageTimeoutSec)
	fmt.Printf("  Session prune:      %dh\n", rec.SessionPruneHours)

	// Check which Ollama models are available
	models := hwprofile.FindOllamaModels()
	if len(models) > 0 {
		fmt.Println()
		fmt.Printf("Ollama models available: %s\n", strings.Join(models, ", "))
		best := p.BestAvailableModel()
		fmt.Printf("Best available model:    %s\n", best)
	} else {
		fmt.Println()
		fmt.Println("Ollama: not running or no models pulled")
	}
}
