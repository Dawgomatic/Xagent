// SWE100821: Dynamic tool loading from skills — lets skills register their own tools
// at runtime. A "weather" skill can register a "weather_fetch" tool with its own
// schema, rather than going through generic exec calls. Makes skills first-class
// citizens in the tool system.
//
// Skills declare tools in their SKILL.md frontmatter:
//   ---
//   tools:
//     - name: weather_fetch
//       description: Fetch weather for a location
//       command: "curl -s 'wttr.in/{location}?format=3'"
//       parameters:
//         location: { type: string, description: City name }
//   ---

package skills

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/Dawgomatic/Xagent/pkg/logger"
)

// SkillToolDef is a tool definition declared by a skill in its frontmatter.
type SkillToolDef struct {
	Name        string                        `json:"name" yaml:"name"`
	Description string                        `json:"description" yaml:"description"`
	Command     string                        `json:"command" yaml:"command"` // shell command template with {param} placeholders
	Parameters  map[string]SkillToolParamDef  `json:"parameters" yaml:"parameters"`
	Timeout     int                           `json:"timeout,omitempty" yaml:"timeout"` // seconds
	SkillName   string                        `json:"skill_name" yaml:"-"`
}

// SkillToolParamDef defines a parameter for a skill-defined tool.
type SkillToolParamDef struct {
	Type        string `json:"type" yaml:"type"`
	Description string `json:"description" yaml:"description"`
	Required    bool   `json:"required,omitempty" yaml:"required"`
	Default     string `json:"default,omitempty" yaml:"default"`
}

// DynamicTool wraps a SkillToolDef as a runnable tool.
type DynamicTool struct {
	def       SkillToolDef
	workspace string
}

// NewDynamicTool creates a tool from a skill tool definition.
func NewDynamicTool(def SkillToolDef, workspace string) *DynamicTool {
	return &DynamicTool{def: def, workspace: workspace}
}

// Name returns the tool name.
func (dt *DynamicTool) Name() string { return dt.def.Name }

// Description returns the tool description, including the source skill.
func (dt *DynamicTool) Description() string {
	return fmt.Sprintf("%s (from skill: %s)", dt.def.Description, dt.def.SkillName)
}

// Parameters returns the tool parameter schema.
func (dt *DynamicTool) Parameters() map[string]interface{} {
	props := make(map[string]interface{})
	required := make([]string, 0)

	for name, param := range dt.def.Parameters {
		props[name] = map[string]interface{}{
			"type":        param.Type,
			"description": param.Description,
		}
		if param.Required {
			required = append(required, name)
		}
	}

	return map[string]interface{}{
		"type":       "object",
		"properties": props,
		"required":   required,
	}
}

// Execute runs the skill tool command with parameter substitution.
func (dt *DynamicTool) Execute(ctx context.Context, args map[string]interface{}) *DynamicToolResult {
	command := dt.def.Command

	// Substitute parameters into command template
	for name, param := range dt.def.Parameters {
		placeholder := "{" + name + "}"
		value, ok := args[name].(string)
		if !ok {
			if param.Default != "" {
				value = param.Default
			} else if param.Required {
				return &DynamicToolResult{
					ForLLM:  fmt.Sprintf("Required parameter '%s' missing", name),
					IsError: true,
				}
			}
		}
		command = strings.ReplaceAll(command, placeholder, value)
	}

	timeout := 30 * time.Second
	if dt.def.Timeout > 0 {
		timeout = time.Duration(dt.def.Timeout) * time.Second
	}

	cmdCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.CommandContext(cmdCtx, "sh", "-c", command)
	cmd.Dir = dt.workspace

	output, err := cmd.CombinedOutput()

	if err != nil {
		if cmdCtx.Err() == context.DeadlineExceeded {
			return &DynamicToolResult{
				ForLLM:  fmt.Sprintf("Skill tool '%s' timed out after %v", dt.def.Name, timeout),
				IsError: true,
			}
		}
		return &DynamicToolResult{
			ForLLM:  fmt.Sprintf("Skill tool '%s' error: %v\nOutput: %s", dt.def.Name, err, string(output)),
			IsError: true,
		}
	}

	result := string(output)
	if len(result) > 10000 {
		result = result[:10000] + fmt.Sprintf("\n... (truncated, %d more chars)", len(result)-10000)
	}

	logger.InfoCF("skill-tool", "Dynamic tool executed",
		map[string]interface{}{
			"tool":       dt.def.Name,
			"skill":      dt.def.SkillName,
			"output_len": len(result),
		})

	return &DynamicToolResult{
		ForLLM:  result,
		IsError: false,
	}
}

// DynamicToolResult is the result from a skill-defined tool.
type DynamicToolResult struct {
	ForLLM  string
	IsError bool
}

// ExtractToolDefs extracts tool definitions from a skill's metadata.
// The metadata is parsed from the skill's SKILL.md frontmatter.
func ExtractToolDefs(skillName string, metadata map[string]interface{}) []SkillToolDef {
	toolsRaw, ok := metadata["tools"]
	if !ok {
		return nil
	}

	toolsList, ok := toolsRaw.([]interface{})
	if !ok {
		return nil
	}

	var defs []SkillToolDef
	for _, t := range toolsList {
		toolMap, ok := t.(map[string]interface{})
		if !ok {
			continue
		}

		def := SkillToolDef{SkillName: skillName}
		if name, ok := toolMap["name"].(string); ok {
			def.Name = name
		}
		if desc, ok := toolMap["description"].(string); ok {
			def.Description = desc
		}
		if cmd, ok := toolMap["command"].(string); ok {
			def.Command = cmd
		}

		if params, ok := toolMap["parameters"].(map[string]interface{}); ok {
			def.Parameters = make(map[string]SkillToolParamDef)
			for pName, pVal := range params {
				pMap, ok := pVal.(map[string]interface{})
				if !ok {
					continue
				}
				pd := SkillToolParamDef{}
				if t, ok := pMap["type"].(string); ok {
					pd.Type = t
				}
				if d, ok := pMap["description"].(string); ok {
					pd.Description = d
				}
				if r, ok := pMap["required"].(bool); ok {
					pd.Required = r
				}
				def.Parameters[pName] = pd
			}
		}

		if def.Name != "" && def.Command != "" {
			defs = append(defs, def)
		}
	}

	return defs
}
