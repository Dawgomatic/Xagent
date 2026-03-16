// SWE100821: Adapts MCP server tools to the standard tools.Tool interface.
// Bridges discovered MCP tools into the agent tool registry so they behave
// identically to built-in tools. Pattern reused from skill_tools.go.

package agent

import (
	"context"
	"fmt"
	"strings"

	"github.com/Dawgomatic/Xagent/pkg/mcp"
	"github.com/Dawgomatic/Xagent/pkg/tools"
)

// mcpToolAdapter wraps an MCP tool as a tools.Tool.
// Reused: pkg/agent/skill_tools.go L15-L29 — same adapter pattern.
type mcpToolAdapter struct {
	mcpClient *mcp.Client
	mcpTool   mcp.MCPTool
	prefix    string // server name prefix for disambiguation
}

func (a *mcpToolAdapter) Name() string {
	// SWE100821: Prefix with server name to avoid collisions between MCP servers
	return a.prefix + "_" + a.mcpTool.Name
}

func (a *mcpToolAdapter) Description() string {
	desc := a.mcpTool.Description
	if desc == "" {
		desc = fmt.Sprintf("MCP tool %s from server %s", a.mcpTool.Name, a.prefix)
	}
	return desc
}

func (a *mcpToolAdapter) Parameters() map[string]interface{} {
	if a.mcpTool.InputSchema != nil {
		return a.mcpTool.InputSchema
	}
	return map[string]interface{}{
		"type":       "object",
		"properties": map[string]interface{}{},
	}
}

// Execute calls the MCP server's tool and returns the result.
func (a *mcpToolAdapter) Execute(ctx context.Context, args map[string]interface{}) *tools.ToolResult {
	// SWE100821: Delegate to MCP client CallTool
	result, err := a.mcpClient.CallTool(ctx, a.mcpTool.Name, args)
	if err != nil {
		return tools.ErrorResult(fmt.Sprintf("MCP tool %s failed: %v", a.mcpTool.Name, err))
	}

	if result.IsError {
		var errText strings.Builder
		for _, c := range result.Content {
			if c.Text != "" {
				errText.WriteString(c.Text)
				errText.WriteString("\n")
			}
		}
		return tools.ErrorResult(errText.String())
	}

	var sb strings.Builder
	for _, c := range result.Content {
		if c.Text != "" {
			sb.WriteString(c.Text)
			sb.WriteString("\n")
		}
	}
	return &tools.ToolResult{
		ForLLM:  sb.String(),
		IsError: false,
	}
}

// registerMCPTools connects to a single MCP server, discovers its tools,
// and registers them in the tool registry. Returns the number of tools registered.
func registerMCPTools(ctx context.Context, client *mcp.Client, registry *tools.ToolRegistry) (int, error) {
	// SWE100821: Initialize handshake
	if err := client.Initialize(ctx); err != nil {
		return 0, fmt.Errorf("initialize %s: %w", client.GetName(), err)
	}

	// SWE100821: Discover available tools
	mcpTools, err := client.DiscoverTools(ctx)
	if err != nil {
		return 0, fmt.Errorf("discover tools %s: %w", client.GetName(), err)
	}

	registered := 0
	for _, mt := range mcpTools {
		adapter := &mcpToolAdapter{
			mcpClient: client,
			mcpTool:   mt,
			prefix:    client.GetName(),
		}
		registry.Register(adapter)
		registered++
	}

	return registered, nil
}
