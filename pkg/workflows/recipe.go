// SWE100821: Composable workflows (recipes) — user-defined multi-step workflows
// as YAML files. Bridges the gap between simple cron jobs and full agent conversations.
// Each recipe has a trigger (cron, event, manual), steps (tool calls), and an
// optional synthesis step that summarizes all results.
//
// Example recipe (workspace/recipes/morning-briefing.yaml):
//
//   name: morning-briefing
//   trigger:
//     type: cron
//     schedule: "0 7 * * *"
//   steps:
//     - tool: web_search
//       args:
//         query: "top tech news today"
//     - tool: exec
//       args:
//         command: "cat /sys/class/thermal/thermal_zone0/temp"
//   synthesize: "Give me a morning briefing with the news and system temperature."
//   channel: telegram

package workflows

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Dawgomatic/Xagent/pkg/logger"
)

// Recipe defines a composable workflow.
type Recipe struct {
	Name        string       `json:"name" yaml:"name"`
	Description string       `json:"description,omitempty" yaml:"description"`
	Trigger     Trigger      `json:"trigger" yaml:"trigger"`
	Steps       []RecipeStep `json:"steps" yaml:"steps"`
	Synthesize  string       `json:"synthesize,omitempty" yaml:"synthesize"` // prompt for synthesis
	Channel     string       `json:"channel,omitempty" yaml:"channel"`      // target channel
	ChatID      string       `json:"chat_id,omitempty" yaml:"chat_id"`
	Enabled     bool         `json:"enabled" yaml:"enabled"`
}

// Trigger defines when a recipe should execute.
type Trigger struct {
	Type     string `json:"type" yaml:"type"`         // "cron", "event", "manual"
	Schedule string `json:"schedule,omitempty" yaml:"schedule"` // cron expression
	Event    string `json:"event,omitempty" yaml:"event"`       // event name (e.g., "device:usb:add")
}

// RecipeStep defines a single step in a recipe.
type RecipeStep struct {
	Tool      string                 `json:"tool" yaml:"tool"`
	Args      map[string]interface{} `json:"args,omitempty" yaml:"args"`
	Label     string                 `json:"label,omitempty" yaml:"label"`
	OnError   string                 `json:"on_error,omitempty" yaml:"on_error"` // "skip", "abort" (default: "abort")
}

// RecipeResult holds the output of a recipe execution.
type RecipeResult struct {
	RecipeName  string            `json:"recipe_name"`
	StartedAt   time.Time         `json:"started_at"`
	CompletedAt time.Time         `json:"completed_at"`
	StepResults []StepResult      `json:"step_results"`
	Synthesis   string            `json:"synthesis,omitempty"`
	Success     bool              `json:"success"`
	Error       string            `json:"error,omitempty"`
}

// StepResult holds the output of a single recipe step.
type StepResult struct {
	Tool    string `json:"tool"`
	Label   string `json:"label"`
	Output  string `json:"output"`
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

// ToolExecutor executes a single tool call and returns the result.
type ToolExecutor func(ctx context.Context, toolName string, args map[string]interface{}) (output string, err error)

// Synthesizer generates a synthesis of step results.
type Synthesizer func(ctx context.Context, prompt string, stepOutputs []string) (string, error)

// RecipeEngine loads and executes workflow recipes.
type RecipeEngine struct {
	recipeDir   string
	recipes     map[string]*Recipe
	executor    ToolExecutor
	synthesizer Synthesizer
	onResult    func(result RecipeResult) // callback for completed recipes
}

// NewRecipeEngine creates a recipe engine loading from the given directory.
func NewRecipeEngine(workspace string) *RecipeEngine {
	recipeDir := filepath.Join(workspace, "recipes")
	os.MkdirAll(recipeDir, 0755)

	engine := &RecipeEngine{
		recipeDir: recipeDir,
		recipes:   make(map[string]*Recipe),
	}
	engine.loadRecipes()
	return engine
}

// SetExecutor sets the tool execution function.
func (re *RecipeEngine) SetExecutor(fn ToolExecutor) {
	re.executor = fn
}

// SetSynthesizer sets the synthesis function.
func (re *RecipeEngine) SetSynthesizer(fn Synthesizer) {
	re.synthesizer = fn
}

// SetResultCallback sets the function called when a recipe completes.
func (re *RecipeEngine) SetResultCallback(fn func(RecipeResult)) {
	re.onResult = fn
}

// Execute runs a recipe by name.
func (re *RecipeEngine) Execute(ctx context.Context, name string) (*RecipeResult, error) {
	recipe, ok := re.recipes[name]
	if !ok {
		return nil, fmt.Errorf("recipe %q not found", name)
	}

	if !recipe.Enabled {
		return nil, fmt.Errorf("recipe %q is disabled", name)
	}

	return re.executeRecipe(ctx, recipe)
}

// ExecuteRecipe runs a recipe directly.
func (re *RecipeEngine) executeRecipe(ctx context.Context, recipe *Recipe) (*RecipeResult, error) {
	result := &RecipeResult{
		RecipeName: recipe.Name,
		StartedAt:  time.Now(),
		Success:    true,
	}

	logger.InfoCF("workflow", "Executing recipe",
		map[string]interface{}{"recipe": recipe.Name, "steps": len(recipe.Steps)})

	var stepOutputs []string

	for i, step := range recipe.Steps {
		if re.executor == nil {
			result.Success = false
			result.Error = "no tool executor configured"
			break
		}

		output, err := re.executor(ctx, step.Tool, step.Args)

		stepResult := StepResult{
			Tool:    step.Tool,
			Label:   step.Label,
			Output:  output,
			Success: err == nil,
		}

		if err != nil {
			stepResult.Error = err.Error()
			onError := step.OnError
			if onError == "" {
				onError = "abort"
			}

			if onError == "abort" {
				result.Success = false
				result.Error = fmt.Sprintf("step %d (%s) failed: %v", i+1, step.Tool, err)
				result.StepResults = append(result.StepResults, stepResult)
				break
			}
			// "skip" — continue to next step
		}

		result.StepResults = append(result.StepResults, stepResult)
		if err == nil {
			stepOutputs = append(stepOutputs, fmt.Sprintf("[%s] %s", step.Tool, output))
		}
	}

	// Synthesis step
	if result.Success && recipe.Synthesize != "" && re.synthesizer != nil && len(stepOutputs) > 0 {
		synthesis, err := re.synthesizer(ctx, recipe.Synthesize, stepOutputs)
		if err != nil {
			logger.WarnCF("workflow", "Synthesis failed",
				map[string]interface{}{"recipe": recipe.Name, "error": err.Error()})
		} else {
			result.Synthesis = synthesis
		}
	}

	result.CompletedAt = time.Now()

	if re.onResult != nil {
		re.onResult(*result)
	}

	logger.InfoCF("workflow", "Recipe execution complete",
		map[string]interface{}{
			"recipe":  recipe.Name,
			"success": result.Success,
			"steps":   len(result.StepResults),
			"elapsed": result.CompletedAt.Sub(result.StartedAt).String(),
		})

	return result, nil
}

// ListRecipes returns all loaded recipes.
func (re *RecipeEngine) ListRecipes() []*Recipe {
	recipes := make([]*Recipe, 0, len(re.recipes))
	for _, r := range re.recipes {
		recipes = append(recipes, r)
	}
	return recipes
}

// GetCronRecipes returns recipes with cron triggers.
func (re *RecipeEngine) GetCronRecipes() []*Recipe {
	var cron []*Recipe
	for _, r := range re.recipes {
		if r.Trigger.Type == "cron" && r.Enabled {
			cron = append(cron, r)
		}
	}
	return cron
}

// GetEventRecipes returns recipes triggered by the given event.
func (re *RecipeEngine) GetEventRecipes(event string) []*Recipe {
	var matching []*Recipe
	for _, r := range re.recipes {
		if r.Trigger.Type == "event" && r.Enabled && matchEvent(r.Trigger.Event, event) {
			matching = append(matching, r)
		}
	}
	return matching
}

// Reload reloads recipes from disk.
func (re *RecipeEngine) Reload() {
	re.recipes = make(map[string]*Recipe)
	re.loadRecipes()
}

func (re *RecipeEngine) loadRecipes() {
	entries, err := os.ReadDir(re.recipeDir)
	if err != nil {
		return
	}

	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.HasSuffix(name, ".json") {
			continue
		}

		data, err := os.ReadFile(filepath.Join(re.recipeDir, name))
		if err != nil {
			continue
		}

		var recipe Recipe
		if err := json.Unmarshal(data, &recipe); err != nil {
			logger.WarnCF("workflow", "Failed to parse recipe",
				map[string]interface{}{"file": name, "error": err.Error()})
			continue
		}

		re.recipes[recipe.Name] = &recipe
	}

	logger.InfoCF("workflow", "Recipes loaded",
		map[string]interface{}{"count": len(re.recipes)})
}

func matchEvent(pattern, event string) bool {
	if pattern == event {
		return true
	}
	// Support wildcard: "device:usb:*" matches "device:usb:add"
	if strings.HasSuffix(pattern, "*") {
		prefix := strings.TrimSuffix(pattern, "*")
		return strings.HasPrefix(event, prefix)
	}
	return false
}
