// Xagent - Ultra-lightweight personal AI agent
// Inspired by and based on nanobot: https://github.com/HKUDS/nanobot
// License: MIT
//
// Copyright (c) 2026 Xagent contributors

package main

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/Dawgomatic/Xagent/pkg/skills"
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

const logo = ""

// formatVersion returns the version string with optional git commit

// formatBuildInfo returns build time and go version info

// hwprofileCmd detects hardware and prints compute tier + recommendations.
// SWE100821: Autonomous hardware detection for scalable high→low compute.

func formatVersion() string {
	v := version
	if gitCommit != "" {
		v += fmt.Sprintf(" (git: %s)", gitCommit)
	}
	return v
}

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
