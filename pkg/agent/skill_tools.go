// SWE100821: Adapts skill-defined DynamicTool to the standard tools.Tool interface.
// Bridges the skills package into the tool registry so SKILL.md-declared tools
// are first-class citizens in the agent's tool system.

package agent

import (
	"context"

	"github.com/Dawgomatic/Xagent/pkg/skills"
	"github.com/Dawgomatic/Xagent/pkg/tools"
)

// skillToolAdapter wraps a skills.DynamicTool to satisfy tools.Tool.
type skillToolAdapter struct {
	dt *skills.DynamicTool
}

func (a *skillToolAdapter) Name() string                       { return a.dt.Name() }
func (a *skillToolAdapter) Description() string                { return a.dt.Description() }
func (a *skillToolAdapter) Parameters() map[string]interface{} { return a.dt.Parameters() }

func (a *skillToolAdapter) Execute(ctx context.Context, args map[string]interface{}) *tools.ToolResult {
	result := a.dt.Execute(ctx, args)
	return &tools.ToolResult{
		ForLLM:  result.ForLLM,
		IsError: result.IsError,
	}
}

// registerDynamicSkillTools extracts tool definitions from loaded skills and
// registers them in the tool registry.
func registerDynamicSkillTools(loader *skills.SkillsLoader, registry *tools.ToolRegistry, workspace string) int {
	allSkills := loader.ListSkills()
	registered := 0

	for _, si := range allSkills {
		metadata := loader.GetRawMetadata(si.Path)
		if metadata == nil {
			continue
		}
		defs := skills.ExtractToolDefs(si.Name, metadata)
		for _, def := range defs {
			adapter := &skillToolAdapter{
				dt: skills.NewDynamicTool(def, workspace),
			}
			registry.Register(adapter)
			registered++
		}
	}

	return registered
}
