// Xagent - Ultra-lightweight personal AI agent
// Inspired by and based on nanobot: https://github.com/HKUDS/nanobot
// License: MIT
//
// Copyright (c) 2026 Xagent contributors

package main

import (
	"bufio"
	"context"
	"embed"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/chzyer/readline"
	"github.com/Dawgomatic/Xagent/pkg/agent"
	"github.com/Dawgomatic/Xagent/pkg/auth"
	"github.com/Dawgomatic/Xagent/pkg/bus"
	"github.com/Dawgomatic/Xagent/pkg/channels"
	"github.com/Dawgomatic/Xagent/pkg/config"
	"github.com/Dawgomatic/Xagent/pkg/cron"
	"github.com/Dawgomatic/Xagent/pkg/devices"
	"github.com/Dawgomatic/Xagent/pkg/health"
	"github.com/Dawgomatic/Xagent/pkg/heartbeat"
	"github.com/Dawgomatic/Xagent/pkg/hwprofile"
	"github.com/Dawgomatic/Xagent/pkg/logger"
	"github.com/Dawgomatic/Xagent/pkg/migrate"
	"github.com/Dawgomatic/Xagent/pkg/providers"
	"github.com/Dawgomatic/Xagent/pkg/skills"
	"github.com/Dawgomatic/Xagent/pkg/state"
	"github.com/Dawgomatic/Xagent/pkg/tools"
	"github.com/Dawgomatic/Xagent/pkg/llmcheck"
	"github.com/Dawgomatic/Xagent/pkg/upgrade"
	"github.com/Dawgomatic/Xagent/pkg/voice"
)

//go:generate cp -r ../../workspace .
//go:embed workspace
var embeddedFiles embed.FS

var (
	version   = "dev"
	gitCommit string
	buildTime string
	goVersion string
)

const logo = "🦞"

// formatVersion returns the version string with optional git commit
func formatVersion() string {
	v := version
	if gitCommit != "" {
		v += fmt.Sprintf(" (git: %s)", gitCommit)
	}
	return v
}

// formatBuildInfo returns build time and go version info
func formatBuildInfo() (build string, goVer string) {
	if buildTime != "" {
		build = buildTime
	}
	goVer = goVersion
	if goVer == "" {
		goVer = runtime.Version()
	}
	return
}

func printVersion() {
	fmt.Printf("%s xagent %s\n", logo, formatVersion())
	build, goVer := formatBuildInfo()
	if build != "" {
		fmt.Printf("  Build: %s\n", build)
	}
	if goVer != "" {
		fmt.Printf("  Go: %s\n", goVer)
	}
}

func copyDirectory(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}

		dstPath := filepath.Join(dst, relPath)

		if info.IsDir() {
			return os.MkdirAll(dstPath, info.Mode())
		}

		srcFile, err := os.Open(path)
		if err != nil {
			return err
		}
		defer srcFile.Close()

		dstFile, err := os.OpenFile(dstPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, info.Mode())
		if err != nil {
			return err
		}
		defer dstFile.Close()

		_, err = io.Copy(dstFile, srcFile)
		return err
	})
}

func main() {
	if len(os.Args) < 2 {
		printHelp()
		os.Exit(1)
	}

	command := os.Args[1]

	switch command {
	case "onboard":
		onboard()
	case "agent":
		agentCmd()
	case "gateway":
		gatewayCmd()
	case "status":
		statusCmd()
	case "hwprofile":
		hwprofileCmd() // SWE100821: CLI hardware detection
	case "migrate":
		migrateCmd()
	case "auth":
		authCmd()
	case "cron":
		cronCmd()
	case "skills":
		if len(os.Args) < 3 {
			skillsHelp()
			return
		}

		subcommand := os.Args[2]

		cfg, err := loadConfig()
		if err != nil {
			fmt.Printf("Error loading config: %v\n", err)
			os.Exit(1)
		}

		workspace := cfg.WorkspacePath()
		installer := skills.NewSkillInstaller(workspace)
		// 获取全局配置目录和内置 skills 目录
		globalDir := filepath.Dir(getConfigPath())
		globalSkillsDir := filepath.Join(globalDir, "skills")
		builtinSkillsDir := filepath.Join(globalDir, "xagent", "skills")
		skillsLoader := skills.NewSkillsLoader(workspace, globalSkillsDir, builtinSkillsDir)

		switch subcommand {
		case "list":
			skillsListCmd(skillsLoader)
		case "install":
			skillsInstallCmd(installer)
		case "create":
			skillsCreateCmd(workspace)
		case "remove", "uninstall":
			if len(os.Args) < 4 {
				fmt.Println("Usage: xagent skills remove <skill-name>")
				return
			}
			skillsRemoveCmd(installer, os.Args[3])
		case "install-builtin":
			skillsInstallBuiltinCmd(workspace)
		case "list-builtin":
			skillsListBuiltinCmd()
		case "search":
			skillsSearchCmd(installer)
		case "show":
			if len(os.Args) < 4 {
				fmt.Println("Usage: xagent skills show <skill-name>")
				return
			}
			skillsShowCmd(skillsLoader, os.Args[3])
		default:
			fmt.Printf("Unknown skills command: %s\n", subcommand)
			skillsHelp()
		}
	case "upgrade":
		upgradeCmd()
	case "llm-check":
		llmCheckCmd()
	case "version", "--version", "-v":
		printVersion()
	default:
		fmt.Printf("Unknown command: %s\n", command)
		printHelp()
		os.Exit(1)
	}
}

func printHelp() {
	fmt.Printf("%s xagent - Personal AI Assistant v%s\n\n", logo, version)
	fmt.Println("Usage: xagent <command>")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  onboard     Initialize xagent configuration and workspace")
	fmt.Println("  agent       Interact with the agent directly")
	fmt.Println("  auth        Manage authentication (login, logout, status)")
	fmt.Println("  gateway     Start xagent gateway")
	fmt.Println("  status      Show xagent status")
	fmt.Println("  hwprofile   Detect hardware and show compute tier + recommendations")
	fmt.Println("  cron        Manage scheduled tasks")
	fmt.Println("  migrate     Migrate from OpenClaw to Xagent")
	fmt.Println("  skills      Manage skills (install, list, remove, create)")
	fmt.Println("  upgrade     Self-upgrade Xagent, models, and skills")
	fmt.Println("  llm-check   Hardware analysis and optimal model recommendation")
	fmt.Println("  version     Show version information")
}

func upgradeCmd() {
	checkOnly := false
	modelOnly := false
	all := false

	for _, arg := range os.Args[2:] {
		switch arg {
		case "--check", "-c":
			checkOnly = true
		case "--model", "-m":
			modelOnly = true
		case "--all", "-a":
			all = true
		case "--help", "-h":
			fmt.Println("Usage: xagent upgrade [options]")
			fmt.Println()
			fmt.Println("Options:")
			fmt.Println("  --check   Check for updates without installing")
			fmt.Println("  --model   Update the Ollama model only")
			fmt.Println("  --all     Upgrade binary + model + skills archive")
			fmt.Println()
			fmt.Println("Examples:")
			fmt.Println("  xagent upgrade           Upgrade Xagent binary")
			fmt.Println("  xagent upgrade --check   Check if update is available")
			fmt.Println("  xagent upgrade --model   Pull latest Ollama model")
			fmt.Println("  xagent upgrade --all     Upgrade everything")
			return
		}
	}

	if modelOnly {
		model := ""
		cfg, err := loadConfig()
		if err == nil {
			model = cfg.Agents.Defaults.Model
		}
		if err := upgrade.UpgradeModel(model); err != nil {
			fmt.Printf("Model upgrade failed: %v\n", err)
			os.Exit(1)
		}
		return
	}

	fmt.Printf("Current version: %s\n", version)
	fmt.Println("Checking for updates...")

	result, err := upgrade.CheckLatest(version)
	if err != nil {
		fmt.Printf("Update check failed: %v\n", err)
		os.Exit(1)
	}

	if !result.UpdateAvail {
		fmt.Println("Already up to date")
		if all {
			upgrade.UpgradeModel("")
			binPath, _ := os.Executable()
			upgrade.UpgradeSkills(filepath.Dir(binPath))
		}
		return
	}

	fmt.Printf("Update available: %s -> %s\n", result.CurrentVersion, result.LatestVersion)
	if result.Release != nil && result.Release.PublishedAt != "" {
		fmt.Printf("Published: %s\n", result.Release.PublishedAt)
	}

	if checkOnly {
		return
	}

	if err := upgrade.DownloadAndReplace(result.Release, version); err != nil {
		fmt.Printf("Upgrade failed: %v\n", err)
		os.Exit(1)
	}

	if all {
		upgrade.UpgradeModel("")
		binPath, _ := os.Executable()
		upgrade.UpgradeSkills(filepath.Dir(binPath))
	}

	fmt.Println("\nRestart Xagent to use the new version:")
	fmt.Println("  sudo systemctl restart xagent-gateway")
}

func llmCheckCmd() {
	subcmd := "check"
	if len(os.Args) >= 3 {
		subcmd = os.Args[2]
	}

	switch subcmd {
	case "hw-detect":
		hw, err := llmcheck.DetectHardware()
		if err != nil {
			fmt.Printf("Hardware detection failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("=== Hardware Profile ===")
		fmt.Printf("CPU:      %s (%d cores)\n", hw.CPU.Brand, hw.CPU.Cores)
		fmt.Printf("Memory:   %.1f GB total, %.1f GB available\n", hw.Memory.TotalGB, hw.Memory.AvailableGB)
		if hw.GPU.Model != "None" {
			fmt.Printf("GPU:      %s (%s, %d MB VRAM)\n", hw.GPU.Model, hw.GPU.Vendor, hw.GPU.VRAM_MB)
		} else {
			fmt.Println("GPU:      None (CPU inference)")
		}
		fmt.Printf("Backend:  %s\n", hw.BackendID)
		fmt.Printf("Tier:     %s\n", hw.Tier)
		fmt.Printf("Arch:     %s/%s\n", hw.OS, hw.Arch)

	case "check":
		fmt.Println("Analyzing hardware and scoring models...")
		result, err := llmcheck.Analyze(llmcheck.AnalysisOptions{
			UseCase: flagOrDefault(os.Args[2:], "--use-case", "general"),
		})
		if err != nil {
			fmt.Printf("Analysis failed: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("\n=== Hardware: %s ===\n", result.Hardware.Summary())
		if result.Ollama.Available {
			fmt.Printf("Ollama:   v%s (running)\n", result.Ollama.Version)
		} else {
			fmt.Printf("Ollama:   not running (%s)\n", result.Ollama.Error)
		}

		if result.TopPick != nil {
			fmt.Printf("\nTop Pick: %s (score %.1f)\n", result.TopPick.Model.Name, result.TopPick.Score.FinalScore)
		}

		fmt.Printf("\n--- Compatible (%d models) ---\n", len(result.Compatible))
		limit := 10
		if limit > len(result.Compatible) {
			limit = len(result.Compatible)
		}
		for _, r := range result.Compatible[:limit] {
			fmt.Println(llmcheck.FormatRecommendation(r))
		}

		if len(result.Marginal) > 0 {
			fmt.Printf("\n--- Marginal (%d models, may swap) ---\n", len(result.Marginal))
			for _, r := range result.Marginal {
				fmt.Println(llmcheck.FormatRecommendation(r))
			}
		}

	case "recommend":
		category := flagOrDefault(os.Args[2:], "--category", "general")
		if len(os.Args) >= 4 && !strings.HasPrefix(os.Args[3], "--") {
			category = os.Args[3]
		}

		hw, err := llmcheck.DetectHardware()
		if err != nil {
			fmt.Printf("Hardware detection failed: %v\n", err)
			os.Exit(1)
		}

		recs := llmcheck.Recommend(category, hw)
		fmt.Printf("=== Top %d for '%s' on %s ===\n\n", len(recs), category, hw.Tier)
		for i, r := range recs {
			fmt.Printf("%2d. %s\n", i+1, llmcheck.FormatRecommendation(r))
		}

	case "installed":
		hw, err := llmcheck.DetectHardware()
		if err != nil {
			fmt.Printf("Hardware detection failed: %v\n", err)
			os.Exit(1)
		}

		useCase := flagOrDefault(os.Args[2:], "--use-case", "general")
		recs, err := llmcheck.RankInstalled(hw, useCase)
		if err != nil {
			fmt.Printf("Failed: %v\n", err)
			os.Exit(1)
		}

		if len(recs) == 0 {
			fmt.Println("No models installed in Ollama.")
			return
		}

		fmt.Printf("=== Installed models ranked for '%s' ===\n\n", useCase)
		for i, r := range recs {
			fmt.Printf("%2d. %s\n", i+1, llmcheck.FormatRecommendation(r))
		}

	case "pull":
		if len(os.Args) < 4 {
			fmt.Println("Usage: xagent llm-check pull <model-name>")
			return
		}
		modelName := os.Args[3]
		fmt.Printf("Pulling %s...\n", modelName)
		client := llmcheck.NewOllamaClient()
		err := client.PullModel(modelName, func(p llmcheck.PullProgress) {
			if p.Total > 0 {
				pct := float64(p.Completed) / float64(p.Total) * 100
				fmt.Printf("\r  %s: %.0f%%", p.Status, pct)
			} else {
				fmt.Printf("\r  %s", p.Status)
			}
		})
		fmt.Println()
		if err != nil {
			fmt.Printf("Pull failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Successfully pulled %s\n", modelName)

	case "benchmark":
		if len(os.Args) < 4 {
			fmt.Println("Usage: xagent llm-check benchmark <model-name>")
			return
		}
		modelName := os.Args[3]
		fmt.Printf("Benchmarking %s...\n", modelName)
		client := llmcheck.NewOllamaClient()
		result, err := client.Benchmark(modelName)
		if err != nil {
			fmt.Printf("Benchmark failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("\n=== Benchmark: %s ===\n", result.Model)
		fmt.Printf("  Tokens/sec:     %.1f\n", result.TokensPerSecond)
		fmt.Printf("  Total time:     %.0f ms\n", result.TotalDurationMs)
		fmt.Printf("  Load time:      %.0f ms\n", result.LoadDurationMs)
		fmt.Printf("  Response tokens: %d\n", result.ResponseTokens)

	default:
		fmt.Println("Usage: xagent llm-check <command>")
		fmt.Println()
		fmt.Println("Commands:")
		fmt.Println("  hw-detect    Detect hardware (CPU, GPU, RAM, backend)")
		fmt.Println("  check        Full analysis: score all models for your hardware")
		fmt.Println("  recommend    Top picks by category (general, coding, reasoning, vision)")
		fmt.Println("  installed    Rank your installed Ollama models")
		fmt.Println("  pull         Pull a model from Ollama registry")
		fmt.Println("  benchmark    Benchmark an installed model")
		fmt.Println()
		fmt.Println("Options:")
		fmt.Println("  --use-case <category>   Scoring category (general, coding, reasoning, chat, vision)")
		fmt.Println("  --category <category>   For recommend: filter by category")
	}
}

func flagOrDefault(args []string, flag, def string) string {
	for i, a := range args {
		if a == flag && i+1 < len(args) {
			return args[i+1]
		}
	}
	return def
}

func onboard() {
	configPath := getConfigPath()

	if _, err := os.Stat(configPath); err == nil {
		fmt.Printf("Config already exists at %s\n", configPath)
		fmt.Print("Overwrite? (y/n): ")
		var response string
		fmt.Scanln(&response)
		if response != "y" {
			fmt.Println("Aborted.")
			return
		}
	}

	cfg := config.DefaultConfig()
	if err := config.SaveConfig(configPath, cfg); err != nil {
		fmt.Printf("Error saving config: %v\n", err)
		os.Exit(1)
	}

	workspace := cfg.WorkspacePath()
	createWorkspaceTemplates(workspace)

	fmt.Printf("%s xagent is ready!\n", logo)
	fmt.Println("\nNext steps:")
	fmt.Println("  1. Add your API key to", configPath)
	fmt.Println("     Get one at: https://openrouter.ai/keys")
	fmt.Println("  2. Chat: xagent agent -m \"Hello!\"")
}

func copyEmbeddedToTarget(targetDir string) error {
	// Ensure target directory exists
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return fmt.Errorf("Failed to create target directory: %w", err)
	}

	// Walk through all files in embed.FS
	err := fs.WalkDir(embeddedFiles, "workspace", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if d.IsDir() {
			return nil
		}

		// Read embedded file
		data, err := embeddedFiles.ReadFile(path)
		if err != nil {
			return fmt.Errorf("Failed to read embedded file %s: %w", path, err)
		}

		new_path, err := filepath.Rel("workspace", path)
		if err != nil {
			return fmt.Errorf("Failed to get relative path for %s: %v\n", path, err)
		}

		// Build target file path
		targetPath := filepath.Join(targetDir, new_path)

		// Ensure target file's directory exists
		if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
			return fmt.Errorf("Failed to create directory %s: %w", filepath.Dir(targetPath), err)
		}

		// Write file
		if err := os.WriteFile(targetPath, data, 0644); err != nil {
			return fmt.Errorf("Failed to write file %s: %w", targetPath, err)
		}

		return nil
	})

	return err
}

func createWorkspaceTemplates(workspace string) {
	err := copyEmbeddedToTarget(workspace)
	if err != nil {
		fmt.Printf("Error copying workspace templates: %v\n", err)
	}
}

func migrateCmd() {
	if len(os.Args) > 2 && (os.Args[2] == "--help" || os.Args[2] == "-h") {
		migrateHelp()
		return
	}

	opts := migrate.Options{}

	args := os.Args[2:]
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--dry-run":
			opts.DryRun = true
		case "--config-only":
			opts.ConfigOnly = true
		case "--workspace-only":
			opts.WorkspaceOnly = true
		case "--force":
			opts.Force = true
		case "--refresh":
			opts.Refresh = true
		case "--openclaw-home":
			if i+1 < len(args) {
				opts.OpenClawHome = args[i+1]
				i++
			}
		case "--xagent-home":
			if i+1 < len(args) {
				opts.XagentHome = args[i+1]
				i++
			}
		default:
			fmt.Printf("Unknown flag: %s\n", args[i])
			migrateHelp()
			os.Exit(1)
		}
	}

	result, err := migrate.Run(opts)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	if !opts.DryRun {
		migrate.PrintSummary(result)
	}
}

func migrateHelp() {
	fmt.Println("\nMigrate from OpenClaw to Xagent")
	fmt.Println()
	fmt.Println("Usage: xagent migrate [options]")
	fmt.Println()
	fmt.Println("Options:")
	fmt.Println("  --dry-run          Show what would be migrated without making changes")
	fmt.Println("  --refresh          Re-sync workspace files from OpenClaw (repeatable)")
	fmt.Println("  --config-only      Only migrate config, skip workspace files")
	fmt.Println("  --workspace-only   Only migrate workspace files, skip config")
	fmt.Println("  --force            Skip confirmation prompts")
	fmt.Println("  --openclaw-home    Override OpenClaw home directory (default: ~/.openclaw)")
	fmt.Println("  --xagent-home    Override Xagent home directory (default: ~/.xagent)")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  xagent migrate              Detect and migrate from OpenClaw")
	fmt.Println("  xagent migrate --dry-run    Show what would be migrated")
	fmt.Println("  xagent migrate --refresh    Re-sync workspace files")
	fmt.Println("  xagent migrate --force      Migrate without confirmation")
}

func agentCmd() {
	message := ""
	sessionKey := "cli:default"

	args := os.Args[2:]
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--debug", "-d":
			logger.SetLevel(logger.DEBUG)
			fmt.Println("🔍 Debug mode enabled")
		case "-m", "--message":
			if i+1 < len(args) {
				message = args[i+1]
				i++
			}
		case "-s", "--session":
			if i+1 < len(args) {
				sessionKey = args[i+1]
				i++
			}
		}
	}

	cfg, err := loadConfig()
	if err != nil {
		fmt.Printf("Error loading config: %v\n", err)
		os.Exit(1)
	}

	provider, err := providers.CreateProvider(cfg)
	if err != nil {
		fmt.Printf("Error creating provider: %v\n", err)
		os.Exit(1)
	}

	msgBus := bus.NewMessageBus()
	agentLoop := agent.NewAgentLoop(cfg, msgBus, provider)

	// Print agent startup info (only for interactive mode)
	startupInfo := agentLoop.GetStartupInfo()
	logger.InfoCF("agent", "Agent initialized",
		map[string]interface{}{
			"tools_count":      startupInfo["tools"].(map[string]interface{})["count"],
			"skills_total":     startupInfo["skills"].(map[string]interface{})["total"],
			"skills_available": startupInfo["skills"].(map[string]interface{})["available"],
		})

	if message != "" {
		ctx := context.Background()
		response, err := agentLoop.ProcessDirect(ctx, message, sessionKey)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("\n%s %s\n", logo, response)
	} else {
		fmt.Printf("%s Interactive mode (Ctrl+C to exit)\n\n", logo)
		interactiveMode(agentLoop, sessionKey)
	}
}

func interactiveMode(agentLoop *agent.AgentLoop, sessionKey string) {
	prompt := fmt.Sprintf("%s You: ", logo)

	rl, err := readline.NewEx(&readline.Config{
		Prompt:          prompt,
		HistoryFile:     filepath.Join(os.TempDir(), ".xagent_history"),
		HistoryLimit:    100,
		InterruptPrompt: "^C",
		EOFPrompt:       "exit",
	})

	if err != nil {
		fmt.Printf("Error initializing readline: %v\n", err)
		fmt.Println("Falling back to simple input mode...")
		simpleInteractiveMode(agentLoop, sessionKey)
		return
	}
	defer rl.Close()

	for {
		line, err := rl.Readline()
		if err != nil {
			if err == readline.ErrInterrupt || err == io.EOF {
				fmt.Println("\nGoodbye!")
				return
			}
			fmt.Printf("Error reading input: %v\n", err)
			continue
		}

		input := strings.TrimSpace(line)
		if input == "" {
			continue
		}

		if input == "exit" || input == "quit" {
			fmt.Println("Goodbye!")
			return
		}

		ctx := context.Background()
		response, err := agentLoop.ProcessDirect(ctx, input, sessionKey)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			continue
		}

		fmt.Printf("\n%s %s\n\n", logo, response)
	}
}

func simpleInteractiveMode(agentLoop *agent.AgentLoop, sessionKey string) {
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print(fmt.Sprintf("%s You: ", logo))
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				fmt.Println("\nGoodbye!")
				return
			}
			fmt.Printf("Error reading input: %v\n", err)
			continue
		}

		input := strings.TrimSpace(line)
		if input == "" {
			continue
		}

		if input == "exit" || input == "quit" {
			fmt.Println("Goodbye!")
			return
		}

		ctx := context.Background()
		response, err := agentLoop.ProcessDirect(ctx, input, sessionKey)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			continue
		}

		fmt.Printf("\n%s %s\n\n", logo, response)
	}
}

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
				"old_tier": string(old.Tier),
				"new_tier": string(cur.Tier),
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
	cancel()
	deviceService.Stop()
	heartbeatService.Stop()
	cronService.Stop()
	agentLoop.Stop()
	channelManager.StopAll(ctx)
	healthServer.Stop()
	fmt.Println("✓ Gateway stopped")
}

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
		hasZhipu := cfg.Providers.Zhipu.APIKey != ""
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
		fmt.Println("Zhipu API:", status(hasZhipu))
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

// hwprofileCmd detects hardware and prints compute tier + recommendations.
// SWE100821: Autonomous hardware detection for scalable high→low compute.
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

func authCmd() {
	if len(os.Args) < 3 {
		authHelp()
		return
	}

	switch os.Args[2] {
	case "login":
		authLoginCmd()
	case "logout":
		authLogoutCmd()
	case "status":
		authStatusCmd()
	default:
		fmt.Printf("Unknown auth command: %s\n", os.Args[2])
		authHelp()
	}
}

func authHelp() {
	fmt.Println("\nAuth commands:")
	fmt.Println("  login       Login via OAuth or paste token")
	fmt.Println("  logout      Remove stored credentials")
	fmt.Println("  status      Show current auth status")
	fmt.Println()
	fmt.Println("Login options:")
	fmt.Println("  --provider <name>    Provider to login with (openai, anthropic)")
	fmt.Println("  --device-code        Use device code flow (for headless environments)")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  xagent auth login --provider openai")
	fmt.Println("  xagent auth login --provider openai --device-code")
	fmt.Println("  xagent auth login --provider anthropic")
	fmt.Println("  xagent auth logout --provider openai")
	fmt.Println("  xagent auth status")
}

func authLoginCmd() {
	provider := ""
	useDeviceCode := false

	args := os.Args[3:]
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--provider", "-p":
			if i+1 < len(args) {
				provider = args[i+1]
				i++
			}
		case "--device-code":
			useDeviceCode = true
		}
	}

	if provider == "" {
		fmt.Println("Error: --provider is required")
		fmt.Println("Supported providers: openai, anthropic")
		return
	}

	switch provider {
	case "openai":
		authLoginOpenAI(useDeviceCode)
	case "anthropic":
		authLoginPasteToken(provider)
	default:
		fmt.Printf("Unsupported provider: %s\n", provider)
		fmt.Println("Supported providers: openai, anthropic")
	}
}

func authLoginOpenAI(useDeviceCode bool) {
	cfg := auth.OpenAIOAuthConfig()

	var cred *auth.AuthCredential
	var err error

	if useDeviceCode {
		cred, err = auth.LoginDeviceCode(cfg)
	} else {
		cred, err = auth.LoginBrowser(cfg)
	}

	if err != nil {
		fmt.Printf("Login failed: %v\n", err)
		os.Exit(1)
	}

	if err := auth.SetCredential("openai", cred); err != nil {
		fmt.Printf("Failed to save credentials: %v\n", err)
		os.Exit(1)
	}

	appCfg, err := loadConfig()
	if err == nil {
		appCfg.Providers.OpenAI.AuthMethod = "oauth"
		if err := config.SaveConfig(getConfigPath(), appCfg); err != nil {
			fmt.Printf("Warning: could not update config: %v\n", err)
		}
	}

	fmt.Println("Login successful!")
	if cred.AccountID != "" {
		fmt.Printf("Account: %s\n", cred.AccountID)
	}
}

func authLoginPasteToken(provider string) {
	cred, err := auth.LoginPasteToken(provider, os.Stdin)
	if err != nil {
		fmt.Printf("Login failed: %v\n", err)
		os.Exit(1)
	}

	if err := auth.SetCredential(provider, cred); err != nil {
		fmt.Printf("Failed to save credentials: %v\n", err)
		os.Exit(1)
	}

	appCfg, err := loadConfig()
	if err == nil {
		switch provider {
		case "anthropic":
			appCfg.Providers.Anthropic.AuthMethod = "token"
		case "openai":
			appCfg.Providers.OpenAI.AuthMethod = "token"
		}
		if err := config.SaveConfig(getConfigPath(), appCfg); err != nil {
			fmt.Printf("Warning: could not update config: %v\n", err)
		}
	}

	fmt.Printf("Token saved for %s!\n", provider)
}

func authLogoutCmd() {
	provider := ""

	args := os.Args[3:]
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--provider", "-p":
			if i+1 < len(args) {
				provider = args[i+1]
				i++
			}
		}
	}

	if provider != "" {
		if err := auth.DeleteCredential(provider); err != nil {
			fmt.Printf("Failed to remove credentials: %v\n", err)
			os.Exit(1)
		}

		appCfg, err := loadConfig()
		if err == nil {
			switch provider {
			case "openai":
				appCfg.Providers.OpenAI.AuthMethod = ""
			case "anthropic":
				appCfg.Providers.Anthropic.AuthMethod = ""
			}
			config.SaveConfig(getConfigPath(), appCfg)
		}

		fmt.Printf("Logged out from %s\n", provider)
	} else {
		if err := auth.DeleteAllCredentials(); err != nil {
			fmt.Printf("Failed to remove credentials: %v\n", err)
			os.Exit(1)
		}

		appCfg, err := loadConfig()
		if err == nil {
			appCfg.Providers.OpenAI.AuthMethod = ""
			appCfg.Providers.Anthropic.AuthMethod = ""
			config.SaveConfig(getConfigPath(), appCfg)
		}

		fmt.Println("Logged out from all providers")
	}
}

func authStatusCmd() {
	store, err := auth.LoadStore()
	if err != nil {
		fmt.Printf("Error loading auth store: %v\n", err)
		return
	}

	if len(store.Credentials) == 0 {
		fmt.Println("No authenticated providers.")
		fmt.Println("Run: xagent auth login --provider <name>")
		return
	}

	fmt.Println("\nAuthenticated Providers:")
	fmt.Println("------------------------")
	for provider, cred := range store.Credentials {
		status := "active"
		if cred.IsExpired() {
			status = "expired"
		} else if cred.NeedsRefresh() {
			status = "needs refresh"
		}

		fmt.Printf("  %s:\n", provider)
		fmt.Printf("    Method: %s\n", cred.AuthMethod)
		fmt.Printf("    Status: %s\n", status)
		if cred.AccountID != "" {
			fmt.Printf("    Account: %s\n", cred.AccountID)
		}
		if !cred.ExpiresAt.IsZero() {
			fmt.Printf("    Expires: %s\n", cred.ExpiresAt.Format("2006-01-02 15:04"))
		}
	}
}

func getConfigPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".xagent", "config.json")
}

func setupCronTool(agentLoop *agent.AgentLoop, msgBus *bus.MessageBus, workspace string) *cron.CronService {
	cronStorePath := filepath.Join(workspace, "cron", "jobs.json")

	// Create cron service
	cronService := cron.NewCronService(cronStorePath, nil)

	// Create and register CronTool
	cronTool := tools.NewCronTool(cronService, agentLoop, msgBus, workspace)
	agentLoop.RegisterTool(cronTool)

	// Set the onJob handler
	cronService.SetOnJob(func(job *cron.CronJob) (string, error) {
		result := cronTool.ExecuteJob(context.Background(), job)
		return result, nil
	})

	return cronService
}

func loadConfig() (*config.Config, error) {
	return config.LoadConfig(getConfigPath())
}

func cronCmd() {
	if len(os.Args) < 3 {
		cronHelp()
		return
	}

	subcommand := os.Args[2]

	// Load config to get workspace path
	cfg, err := loadConfig()
	if err != nil {
		fmt.Printf("Error loading config: %v\n", err)
		return
	}

	cronStorePath := filepath.Join(cfg.WorkspacePath(), "cron", "jobs.json")

	switch subcommand {
	case "list":
		cronListCmd(cronStorePath)
	case "add":
		cronAddCmd(cronStorePath)
	case "remove":
		if len(os.Args) < 4 {
			fmt.Println("Usage: xagent cron remove <job_id>")
			return
		}
		cronRemoveCmd(cronStorePath, os.Args[3])
	case "enable":
		cronEnableCmd(cronStorePath, false)
	case "disable":
		cronEnableCmd(cronStorePath, true)
	default:
		fmt.Printf("Unknown cron command: %s\n", subcommand)
		cronHelp()
	}
}

func cronHelp() {
	fmt.Println("\nCron commands:")
	fmt.Println("  list              List all scheduled jobs")
	fmt.Println("  add              Add a new scheduled job")
	fmt.Println("  remove <id>       Remove a job by ID")
	fmt.Println("  enable <id>      Enable a job")
	fmt.Println("  disable <id>     Disable a job")
	fmt.Println()
	fmt.Println("Add options:")
	fmt.Println("  -n, --name       Job name")
	fmt.Println("  -m, --message    Message for agent")
	fmt.Println("  -e, --every      Run every N seconds")
	fmt.Println("  -c, --cron       Cron expression (e.g. '0 9 * * *')")
	fmt.Println("  -d, --deliver     Deliver response to channel")
	fmt.Println("  --to             Recipient for delivery")
	fmt.Println("  --channel        Channel for delivery")
}

func cronListCmd(storePath string) {
	cs := cron.NewCronService(storePath, nil)
	jobs := cs.ListJobs(true) // Show all jobs, including disabled

	if len(jobs) == 0 {
		fmt.Println("No scheduled jobs.")
		return
	}

	fmt.Println("\nScheduled Jobs:")
	fmt.Println("----------------")
	for _, job := range jobs {
		var schedule string
		if job.Schedule.Kind == "every" && job.Schedule.EveryMS != nil {
			schedule = fmt.Sprintf("every %ds", *job.Schedule.EveryMS/1000)
		} else if job.Schedule.Kind == "cron" {
			schedule = job.Schedule.Expr
		} else {
			schedule = "one-time"
		}

		nextRun := "scheduled"
		if job.State.NextRunAtMS != nil {
			nextTime := time.UnixMilli(*job.State.NextRunAtMS)
			nextRun = nextTime.Format("2006-01-02 15:04")
		}

		status := "enabled"
		if !job.Enabled {
			status = "disabled"
		}

		fmt.Printf("  %s (%s)\n", job.Name, job.ID)
		fmt.Printf("    Schedule: %s\n", schedule)
		fmt.Printf("    Status: %s\n", status)
		fmt.Printf("    Next run: %s\n", nextRun)
	}
}

func cronAddCmd(storePath string) {
	name := ""
	message := ""
	var everySec *int64
	cronExpr := ""
	deliver := false
	channel := ""
	to := ""

	args := os.Args[3:]
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-n", "--name":
			if i+1 < len(args) {
				name = args[i+1]
				i++
			}
		case "-m", "--message":
			if i+1 < len(args) {
				message = args[i+1]
				i++
			}
		case "-e", "--every":
			if i+1 < len(args) {
				var sec int64
				fmt.Sscanf(args[i+1], "%d", &sec)
				everySec = &sec
				i++
			}
		case "-c", "--cron":
			if i+1 < len(args) {
				cronExpr = args[i+1]
				i++
			}
		case "-d", "--deliver":
			deliver = true
		case "--to":
			if i+1 < len(args) {
				to = args[i+1]
				i++
			}
		case "--channel":
			if i+1 < len(args) {
				channel = args[i+1]
				i++
			}
		}
	}

	if name == "" {
		fmt.Println("Error: --name is required")
		return
	}

	if message == "" {
		fmt.Println("Error: --message is required")
		return
	}

	if everySec == nil && cronExpr == "" {
		fmt.Println("Error: Either --every or --cron must be specified")
		return
	}

	var schedule cron.CronSchedule
	if everySec != nil {
		everyMS := *everySec * 1000
		schedule = cron.CronSchedule{
			Kind:    "every",
			EveryMS: &everyMS,
		}
	} else {
		schedule = cron.CronSchedule{
			Kind: "cron",
			Expr: cronExpr,
		}
	}

	cs := cron.NewCronService(storePath, nil)
	job, err := cs.AddJob(name, schedule, message, deliver, channel, to)
	if err != nil {
		fmt.Printf("Error adding job: %v\n", err)
		return
	}

	fmt.Printf("✓ Added job '%s' (%s)\n", job.Name, job.ID)
}

func cronRemoveCmd(storePath, jobID string) {
	cs := cron.NewCronService(storePath, nil)
	if cs.RemoveJob(jobID) {
		fmt.Printf("✓ Removed job %s\n", jobID)
	} else {
		fmt.Printf("✗ Job %s not found\n", jobID)
	}
}

func cronEnableCmd(storePath string, disable bool) {
	if len(os.Args) < 4 {
		fmt.Println("Usage: xagent cron enable/disable <job_id>")
		return
	}

	jobID := os.Args[3]
	cs := cron.NewCronService(storePath, nil)
	enabled := !disable

	job := cs.EnableJob(jobID, enabled)
	if job != nil {
		status := "enabled"
		if disable {
			status = "disabled"
		}
		fmt.Printf("✓ Job '%s' %s\n", job.Name, status)
	} else {
		fmt.Printf("✗ Job %s not found\n", jobID)
	}
}

func skillsHelp() {
	fmt.Println("\nSkills commands:")
	fmt.Println("  list                    List installed skills")
	fmt.Println("  create <name>           Create a new skill from template")
	fmt.Println("  install <repo>          Install skill from GitHub")
	fmt.Println("  install-builtin         Install all builtin skills to workspace")
	fmt.Println("  list-builtin            List available builtin skills")
	fmt.Println("  remove <name>           Remove installed skill")
	fmt.Println("  search                  Search available skills")
	fmt.Println("  show <name>             Show skill details")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  xagent skills list")
	fmt.Println("  xagent skills create my-new-skill")
	fmt.Println("  xagent skills install sipeed/xagent-skills/weather")
	fmt.Println("  xagent skills install-builtin")
	fmt.Println("  xagent skills remove weather")
}

func skillsCreateCmd(workspace string) {
	if len(os.Args) < 4 {
		fmt.Println("Usage: xagent skills create <skill-name> [--description \"...\"]")
		fmt.Println("Example: xagent skills create systemd-manager")
		return
	}

	name := os.Args[3]
	description := ""

	for i := 4; i < len(os.Args); i++ {
		if (os.Args[i] == "--description" || os.Args[i] == "-d") && i+1 < len(os.Args) {
			description = os.Args[i+1]
			i++
		}
	}

	skillDir := filepath.Join(workspace, "skills", name)
	if _, err := os.Stat(skillDir); err == nil {
		fmt.Printf("Skill '%s' already exists at %s\n", name, skillDir)
		return
	}

	if err := os.MkdirAll(skillDir, 0755); err != nil {
		fmt.Printf("Error creating directory: %v\n", err)
		os.Exit(1)
	}

	if description == "" {
		description = fmt.Sprintf("TODO: Describe what %s does and when to use it.", name)
	}

	words := strings.Split(name, "-")
	for i, w := range words {
		if len(w) > 0 {
			words[i] = strings.ToUpper(w[:1]) + w[1:]
		}
	}
	title := strings.Join(words, " ")

	content := fmt.Sprintf(`---
name: %s
description: %s
---

# %s

## Quick Start

TODO: Add instructions here.
`, name, description, title)

	skillPath := filepath.Join(skillDir, "SKILL.md")
	if err := os.WriteFile(skillPath, []byte(content), 0644); err != nil {
		fmt.Printf("Error writing SKILL.md: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Created skill: %s\n", skillDir)
	fmt.Printf("  %s/SKILL.md\n", name)
	fmt.Println()
	fmt.Printf("Edit %s to add your skill instructions.\n", skillPath)
	fmt.Println("The skill will be auto-discovered by Xagent.")
}

func skillsListCmd(loader *skills.SkillsLoader) {
	allSkills := loader.ListSkills()

	if len(allSkills) == 0 {
		fmt.Println("No skills installed.")
		return
	}

	fmt.Println("\nInstalled Skills:")
	fmt.Println("------------------")
	for _, skill := range allSkills {
		fmt.Printf("  ✓ %s (%s)\n", skill.Name, skill.Source)
		if skill.Description != "" {
			fmt.Printf("    %s\n", skill.Description)
		}
	}
}

func skillsInstallCmd(installer *skills.SkillInstaller) {
	if len(os.Args) < 4 {
		fmt.Println("Usage: xagent skills install <github-repo>")
		fmt.Println("Example: xagent skills install sipeed/xagent-skills/weather")
		return
	}

	repo := os.Args[3]
	fmt.Printf("Installing skill from %s...\n", repo)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := installer.InstallFromGitHub(ctx, repo); err != nil {
		fmt.Printf("✗ Failed to install skill: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✓ Skill '%s' installed successfully!\n", filepath.Base(repo))
}

func skillsRemoveCmd(installer *skills.SkillInstaller, skillName string) {
	fmt.Printf("Removing skill '%s'...\n", skillName)

	if err := installer.Uninstall(skillName); err != nil {
		fmt.Printf("✗ Failed to remove skill: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✓ Skill '%s' removed successfully!\n", skillName)
}

func skillsInstallBuiltinCmd(workspace string) {
	builtinSkillsDir := "./xagent/skills"
	workspaceSkillsDir := filepath.Join(workspace, "skills")

	fmt.Printf("Copying builtin skills to workspace...\n")

	skillsToInstall := []string{
		"weather",
		"news",
		"stock",
		"calculator",
	}

	for _, skillName := range skillsToInstall {
		builtinPath := filepath.Join(builtinSkillsDir, skillName)
		workspacePath := filepath.Join(workspaceSkillsDir, skillName)

		if _, err := os.Stat(builtinPath); err != nil {
			fmt.Printf("⊘ Builtin skill '%s' not found: %v\n", skillName, err)
			continue
		}

		if err := os.MkdirAll(workspacePath, 0755); err != nil {
			fmt.Printf("✗ Failed to create directory for %s: %v\n", skillName, err)
			continue
		}

		if err := copyDirectory(builtinPath, workspacePath); err != nil {
			fmt.Printf("✗ Failed to copy %s: %v\n", skillName, err)
		}
	}

	fmt.Println("\n✓ All builtin skills installed!")
	fmt.Println("Now you can use them in your workspace.")
}

func skillsListBuiltinCmd() {
	cfg, err := loadConfig()
	if err != nil {
		fmt.Printf("Error loading config: %v\n", err)
		return
	}
	builtinSkillsDir := filepath.Join(filepath.Dir(cfg.WorkspacePath()), "xagent", "skills")

	fmt.Println("\nAvailable Builtin Skills:")
	fmt.Println("-----------------------")

	entries, err := os.ReadDir(builtinSkillsDir)
	if err != nil {
		fmt.Printf("Error reading builtin skills: %v\n", err)
		return
	}

	if len(entries) == 0 {
		fmt.Println("No builtin skills available.")
		return
	}

	for _, entry := range entries {
		if entry.IsDir() {
			skillName := entry.Name()
			skillFile := filepath.Join(builtinSkillsDir, skillName, "SKILL.md")

			description := "No description"
			if _, err := os.Stat(skillFile); err == nil {
				data, err := os.ReadFile(skillFile)
				if err == nil {
					content := string(data)
					if idx := strings.Index(content, "\n"); idx > 0 {
						firstLine := content[:idx]
						if strings.Contains(firstLine, "description:") {
							descLine := strings.Index(content[idx:], "\n")
							if descLine > 0 {
								description = strings.TrimSpace(content[idx+descLine : idx+descLine])
							}
						}
					}
				}
			}
			status := "✓"
			fmt.Printf("  %s  %s\n", status, entry.Name())
			if description != "" {
				fmt.Printf("     %s\n", description)
			}
		}
	}
}

func skillsSearchCmd(installer *skills.SkillInstaller) {
	fmt.Println("Searching for available skills...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	availableSkills, err := installer.ListAvailableSkills(ctx)
	if err != nil {
		fmt.Printf("✗ Failed to fetch skills list: %v\n", err)
		return
	}

	if len(availableSkills) == 0 {
		fmt.Println("No skills available.")
		return
	}

	fmt.Printf("\nAvailable Skills (%d):\n", len(availableSkills))
	fmt.Println("--------------------")
	for _, skill := range availableSkills {
		fmt.Printf("  📦 %s\n", skill.Name)
		fmt.Printf("     %s\n", skill.Description)
		fmt.Printf("     Repo: %s\n", skill.Repository)
		if skill.Author != "" {
			fmt.Printf("     Author: %s\n", skill.Author)
		}
		if len(skill.Tags) > 0 {
			fmt.Printf("     Tags: %v\n", skill.Tags)
		}
		fmt.Println()
	}
}

func skillsShowCmd(loader *skills.SkillsLoader, skillName string) {
	content, ok := loader.LoadSkill(skillName)
	if !ok {
		fmt.Printf("✗ Skill '%s' not found\n", skillName)
		return
	}

	fmt.Printf("\n📦 Skill: %s\n", skillName)
	fmt.Println("----------------------")
	fmt.Println(content)
}
