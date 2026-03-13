package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/Dawgomatic/Xagent/pkg/upgrade"
)

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
