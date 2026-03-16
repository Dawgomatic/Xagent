// SWE100821: Tests for skillToolAdapter (bridges skills.DynamicTool → tools.Tool).

package agent

import (
	"context"
	"testing"

	"github.com/Dawgomatic/Xagent/pkg/skills"
)

// SWE100821: verify skillToolAdapter delegates Name/Description/Parameters correctly
func TestSkillToolAdapter_Metadata(t *testing.T) {
	def := skills.SkillToolDef{
		Name:        "test_tool",
		Description: "A test tool",
		Command:     "echo hello",
		SkillName:   "test-skill",
		Parameters: map[string]skills.SkillToolParamDef{
			"arg1": {Type: "string", Description: "first arg", Required: true},
		},
	}
	adapter := &skillToolAdapter{
		dt: skills.NewDynamicTool(def, t.TempDir()),
	}

	if adapter.Name() != "test_tool" {
		t.Errorf("expected name 'test_tool', got %q", adapter.Name())
	}
	if adapter.Description() == "" {
		t.Error("expected non-empty description")
	}

	params := adapter.Parameters()
	if params == nil {
		t.Fatal("expected non-nil parameters")
	}
	props, ok := params["properties"].(map[string]interface{})
	if !ok {
		t.Fatal("expected 'properties' in parameters")
	}
	if _, ok := props["arg1"]; !ok {
		t.Error("expected 'arg1' in properties")
	}
}

// SWE100821: verify skillToolAdapter.Execute runs and returns a ToolResult
func TestSkillToolAdapter_Execute(t *testing.T) {
	def := skills.SkillToolDef{
		Name:      "echo_tool",
		Command:   "echo 'skill output'",
		SkillName: "test-skill",
	}
	adapter := &skillToolAdapter{
		dt: skills.NewDynamicTool(def, t.TempDir()),
	}

	result := adapter.Execute(context.Background(), map[string]interface{}{})
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	// SWE100821: echo command should succeed
	if result.IsError {
		t.Errorf("expected no error, got error: %s", result.ForLLM)
	}
	if result.ForLLM == "" {
		t.Error("expected non-empty ForLLM content")
	}
}

// SWE100821: verify missing required param produces error
func TestSkillToolAdapter_MissingRequiredParam(t *testing.T) {
	def := skills.SkillToolDef{
		Name:      "param_tool",
		Command:   "echo {name}",
		SkillName: "test-skill",
		Parameters: map[string]skills.SkillToolParamDef{
			"name": {Type: "string", Description: "name", Required: true},
		},
	}
	adapter := &skillToolAdapter{
		dt: skills.NewDynamicTool(def, t.TempDir()),
	}

	result := adapter.Execute(context.Background(), map[string]interface{}{})
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	// SWE100821: missing required param should produce error
	if !result.IsError {
		t.Error("expected error for missing required parameter")
	}
}
