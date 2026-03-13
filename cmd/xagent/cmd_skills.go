package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Dawgomatic/Xagent/pkg/skills"
)

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
