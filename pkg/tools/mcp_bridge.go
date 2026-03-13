// MCP Bridge: Wraps MCP tools as native Xagent tools so they appear
// in the LLM's tool list alongside built-in tools.

package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Dawgomatic/Xagent/pkg/mcp"
)

// MCPBridgeTool wraps a single MCP tool as a native Xagent Tool.
type MCPBridgeTool struct {
	client     *mcp.Client
	mcpTool    mcp.MCPTool
	serverName string
}

// NewMCPBridgeTool creates an Xagent tool that delegates to an MCP server.
func NewMCPBridgeTool(client *mcp.Client, tool mcp.MCPTool) *MCPBridgeTool {
	return &MCPBridgeTool{
		client:     client,
		mcpTool:    tool,
		serverName: client.GetName(),
	}
}

func (t *MCPBridgeTool) Name() string {
	return fmt.Sprintf("mcp_%s_%s", t.serverName, t.mcpTool.Name)
}

func (t *MCPBridgeTool) Description() string {
	desc := t.mcpTool.Description
	if desc == "" {
		desc = fmt.Sprintf("MCP tool '%s' from server '%s'", t.mcpTool.Name, t.serverName)
	}
	return fmt.Sprintf("[MCP:%s] %s", t.serverName, desc)
}

func (t *MCPBridgeTool) Parameters() map[string]interface{} {
	if t.mcpTool.InputSchema != nil {
		return t.mcpTool.InputSchema
	}
	return map[string]interface{}{
		"type":       "object",
		"properties": map[string]interface{}{},
	}
}

func (t *MCPBridgeTool) Execute(ctx context.Context, args map[string]interface{}) *ToolResult {
	result, err := t.client.CallTool(ctx, t.mcpTool.Name, args)
	if err != nil {
		return &ToolResult{
			ForLLM:  fmt.Sprintf("MCP tool error: %v", err),
			IsError: true,
			Err:     err,
		}
	}

	if result.IsError {
		var texts []string
		for _, c := range result.Content {
			if c.Text != "" {
				texts = append(texts, c.Text)
			}
		}
		errMsg := strings.Join(texts, "\n")
		return &ToolResult{
			ForLLM:  fmt.Sprintf("MCP tool returned error: %s", errMsg),
			IsError: true,
		}
	}

	// Combine all text content
	var texts []string
	for _, c := range result.Content {
		if c.Text != "" {
			texts = append(texts, c.Text)
		}
	}

	output := strings.Join(texts, "\n")
	if output == "" {
		output = "Tool completed successfully (no output)"
	}

	return &ToolResult{
		ForLLM:  output,
		ForUser: output,
	}
}

// RegisterMCPTools connects to an MCP server and registers all discovered tools.
func RegisterMCPTools(registry *ToolRegistry, client *mcp.Client) (int, error) {
	ctx := context.Background()

	if err := client.Initialize(ctx); err != nil {
		return 0, fmt.Errorf("mcp init: %w", err)
	}

	tools, err := client.DiscoverTools(ctx)
	if err != nil {
		return 0, fmt.Errorf("mcp discover: %w", err)
	}

	for _, tool := range tools {
		bridge := NewMCPBridgeTool(client, tool)
		registry.Register(bridge)
	}

	return len(tools), nil
}

// MCPToolsJSON returns a JSON summary of all MCP tools for debugging.
func MCPToolsJSON(client *mcp.Client) string {
	tools := client.GetTools()
	data, _ := json.MarshalIndent(tools, "", "  ")
	return string(data)
}
