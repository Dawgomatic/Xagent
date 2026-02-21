package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Dawgomatic/Xagent/pkg/llmcheck"
)

type LLMCheckTool struct{}

func NewLLMCheckTool() *LLMCheckTool {
	return &LLMCheckTool{}
}

func (t *LLMCheckTool) Name() string {
	return "llm_check"
}

func (t *LLMCheckTool) Description() string {
	return "Analyze hardware and recommend optimal LLM models. Actions: hw-detect (detect hardware), check (full analysis), recommend (top picks by category: general/coding/reasoning/chat/vision), installed (rank installed models), pull (download model), benchmark (test model speed)."
}

func (t *LLMCheckTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"action": map[string]interface{}{
				"type":        "string",
				"description": "Action to perform: hw-detect, check, recommend, installed, pull, benchmark",
				"enum":        []string{"hw-detect", "check", "recommend", "installed", "pull", "benchmark"},
			},
			"category": map[string]interface{}{
				"type":        "string",
				"description": "Use case category for scoring (general, coding, reasoning, chat, vision, fast, quality)",
			},
			"model": map[string]interface{}{
				"type":        "string",
				"description": "Model name for pull/benchmark actions",
			},
		},
		"required": []string{"action"},
	}
}

func (t *LLMCheckTool) Execute(ctx context.Context, args map[string]interface{}) *ToolResult {
	action, _ := args["action"].(string)
	category, _ := args["category"].(string)
	model, _ := args["model"].(string)

	if category == "" {
		category = "general"
	}

	switch action {
	case "hw-detect":
		return t.hwDetect()
	case "check":
		return t.check(category)
	case "recommend":
		return t.recommend(category)
	case "installed":
		return t.installed(category)
	case "pull":
		if model == "" {
			return ErrorResult("model name required for pull action")
		}
		return t.pull(model)
	case "benchmark":
		if model == "" {
			return ErrorResult("model name required for benchmark action")
		}
		return t.benchmark(model)
	default:
		return ErrorResult(fmt.Sprintf("unknown action: %s", action))
	}
}

func (t *LLMCheckTool) hwDetect() *ToolResult {
	hw, err := llmcheck.DetectHardware()
	if err != nil {
		return ErrorResult(fmt.Sprintf("hardware detection failed: %v", err))
	}
	data, _ := json.MarshalIndent(hw, "", "  ")
	return NewToolResult(string(data))
}

func (t *LLMCheckTool) check(category string) *ToolResult {
	result, err := llmcheck.Analyze(llmcheck.AnalysisOptions{UseCase: category})
	if err != nil {
		return ErrorResult(fmt.Sprintf("analysis failed: %v", err))
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Hardware: %s\n", result.Hardware.Summary()))
	if result.Ollama.Available {
		sb.WriteString(fmt.Sprintf("Ollama: v%s (running)\n", result.Ollama.Version))
	} else {
		sb.WriteString(fmt.Sprintf("Ollama: not running\n"))
	}

	if result.TopPick != nil {
		sb.WriteString(fmt.Sprintf("\nTop Pick: %s (score %.1f, ~%.0f TPS, %.1fGB)\n",
			result.TopPick.Model.Name, result.TopPick.Score.FinalScore,
			result.TopPick.Score.EstTPS, result.TopPick.Model.EffectiveSize()))
	}

	sb.WriteString(fmt.Sprintf("\nCompatible models (%d):\n", len(result.Compatible)))
	limit := 10
	if limit > len(result.Compatible) {
		limit = len(result.Compatible)
	}
	for _, r := range result.Compatible[:limit] {
		sb.WriteString(fmt.Sprintf("  %s\n", llmcheck.FormatRecommendation(r)))
	}

	if len(result.Marginal) > 0 {
		sb.WriteString(fmt.Sprintf("\nMarginal models (%d):\n", len(result.Marginal)))
		for _, r := range result.Marginal {
			sb.WriteString(fmt.Sprintf("  %s\n", llmcheck.FormatRecommendation(r)))
		}
	}

	return NewToolResult(sb.String())
}

func (t *LLMCheckTool) recommend(category string) *ToolResult {
	hw, err := llmcheck.DetectHardware()
	if err != nil {
		return ErrorResult(fmt.Sprintf("hardware detection failed: %v", err))
	}

	recs := llmcheck.Recommend(category, hw)
	if len(recs) == 0 {
		return NewToolResult("No compatible models found for this hardware and category.")
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Top %d for '%s' on %s:\n\n", len(recs), category, hw.Tier))
	for i, r := range recs {
		sb.WriteString(fmt.Sprintf("%d. %s\n", i+1, llmcheck.FormatRecommendation(r)))
	}
	return NewToolResult(sb.String())
}

func (t *LLMCheckTool) installed(category string) *ToolResult {
	hw, err := llmcheck.DetectHardware()
	if err != nil {
		return ErrorResult(fmt.Sprintf("hardware detection failed: %v", err))
	}

	recs, err := llmcheck.RankInstalled(hw, category)
	if err != nil {
		return ErrorResult(fmt.Sprintf("failed: %v", err))
	}

	if len(recs) == 0 {
		return NewToolResult("No models installed in Ollama.")
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Installed models ranked for '%s':\n\n", category))
	for i, r := range recs {
		sb.WriteString(fmt.Sprintf("%d. %s\n", i+1, llmcheck.FormatRecommendation(r)))
	}
	return NewToolResult(sb.String())
}

func (t *LLMCheckTool) pull(model string) *ToolResult {
	client := llmcheck.NewOllamaClient()
	status := client.CheckAvailability()
	if !status.Available {
		return ErrorResult(fmt.Sprintf("Ollama not available: %s", status.Error))
	}

	var lastStatus string
	err := client.PullModel(model, func(p llmcheck.PullProgress) {
		lastStatus = p.Status
	})
	if err != nil {
		return ErrorResult(fmt.Sprintf("pull failed: %v", err))
	}
	_ = lastStatus
	return NewToolResult(fmt.Sprintf("Successfully pulled model: %s", model))
}

func (t *LLMCheckTool) benchmark(model string) *ToolResult {
	client := llmcheck.NewOllamaClient()
	result, err := client.Benchmark(model)
	if err != nil {
		return ErrorResult(fmt.Sprintf("benchmark failed: %v", err))
	}

	data, _ := json.MarshalIndent(result, "", "  ")
	return NewToolResult(string(data))
}
