package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/Dawgomatic/Xagent/pkg/llmcheck"
)

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
