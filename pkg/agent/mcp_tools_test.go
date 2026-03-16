// SWE100821: Tests for MCP tool adapter — verifies name prefixing, parameter
// passthrough, and execute delegation with mock transport.

package agent

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/Dawgomatic/Xagent/pkg/mcp"
	"github.com/Dawgomatic/Xagent/pkg/tools"
)

// SWE100821: mockTransport implements mcp.Transport for testing
type mockTransport struct {
	responses [][]byte
	idx       int
	sent      [][]byte
}

func (mt *mockTransport) Send(_ context.Context, data []byte) error {
	mt.sent = append(mt.sent, data)
	return nil
}

func (mt *mockTransport) Receive(_ context.Context) ([]byte, error) {
	if mt.idx >= len(mt.responses) {
		return nil, context.Canceled
	}
	resp := mt.responses[mt.idx]
	mt.idx++
	return resp, nil
}

func (mt *mockTransport) Close() error { return nil }

func makeJSONRPCResp(id int64, result interface{}) []byte {
	res, _ := json.Marshal(result)
	resp := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      id,
		"result":  json.RawMessage(res),
	}
	data, _ := json.Marshal(resp)
	return data
}

// SWE100821: TestMCPToolAdapter_Name verifies server prefix is applied
func TestMCPToolAdapter_Name(t *testing.T) {
	adapter := &mcpToolAdapter{
		mcpTool: mcp.MCPTool{Name: "search"},
		prefix:  "brave",
	}
	if adapter.Name() != "brave_search" {
		t.Errorf("expected 'brave_search', got '%s'", adapter.Name())
	}
}

// SWE100821: TestMCPToolAdapter_Description with fallback
func TestMCPToolAdapter_Description(t *testing.T) {
	adapter := &mcpToolAdapter{
		mcpTool: mcp.MCPTool{Name: "search", Description: ""},
		prefix:  "brave",
	}
	desc := adapter.Description()
	if !strings.Contains(desc, "brave") {
		t.Errorf("fallback description should contain prefix, got: %s", desc)
	}

	adapter2 := &mcpToolAdapter{
		mcpTool: mcp.MCPTool{Name: "search", Description: "Web search tool"},
		prefix:  "brave",
	}
	if adapter2.Description() != "Web search tool" {
		t.Errorf("expected explicit description, got: %s", adapter2.Description())
	}
}

// SWE100821: TestMCPToolAdapter_Parameters with and without schema
func TestMCPToolAdapter_Parameters(t *testing.T) {
	adapter := &mcpToolAdapter{
		mcpTool: mcp.MCPTool{Name: "search", InputSchema: nil},
		prefix:  "brave",
	}
	params := adapter.Parameters()
	if params["type"] != "object" {
		t.Errorf("nil schema should return default object schema")
	}

	schema := map[string]interface{}{"type": "object", "properties": map[string]interface{}{"q": "string"}}
	adapter2 := &mcpToolAdapter{
		mcpTool: mcp.MCPTool{Name: "search", InputSchema: schema},
		prefix:  "brave",
	}
	if adapter2.Parameters()["properties"] == nil {
		t.Error("schema should pass through")
	}
}

// SWE100821: TestMCPToolAdapter_Execute with mock transport
func TestMCPToolAdapter_Execute(t *testing.T) {
	callToolResult := mcp.CallToolResult{
		Content: []mcp.ContentItem{{Type: "text", Text: "search result"}},
		IsError: false,
	}
	transport := &mockTransport{
		responses: [][]byte{makeJSONRPCResp(1, callToolResult)},
	}
	client := mcp.NewClient("test-server", transport)

	adapter := &mcpToolAdapter{
		mcpClient: client,
		mcpTool:   mcp.MCPTool{Name: "search"},
		prefix:    "test-server",
	}
	result := adapter.Execute(context.Background(), map[string]interface{}{"q": "hello"})
	if result.IsError {
		t.Errorf("expected success, got error: %s", result.ForLLM)
	}
	if !strings.Contains(result.ForLLM, "search result") {
		t.Errorf("expected 'search result' in output, got: %s", result.ForLLM)
	}
}

// SWE100821: TestMCPToolAdapter_ExecuteError verifies error propagation
func TestMCPToolAdapter_ExecuteError(t *testing.T) {
	callToolResult := mcp.CallToolResult{
		Content: []mcp.ContentItem{{Type: "text", Text: "not found"}},
		IsError: true,
	}
	transport := &mockTransport{
		responses: [][]byte{makeJSONRPCResp(1, callToolResult)},
	}
	client := mcp.NewClient("test-server", transport)

	adapter := &mcpToolAdapter{
		mcpClient: client,
		mcpTool:   mcp.MCPTool{Name: "search"},
		prefix:    "test-server",
	}
	result := adapter.Execute(context.Background(), map[string]interface{}{})
	if !result.IsError {
		t.Error("expected error result for MCP error response")
	}
}

// SWE100821: TestRegisterMCPTools verifies discovery + registration
func TestRegisterMCPTools(t *testing.T) {
	initResult := mcp.InitializeResult{
		ProtocolVersion: "2024-11-05",
		ServerInfo:      mcp.ServerInfo{Name: "test", Version: "0.1"},
	}
	listResult := mcp.ListToolsResult{
		Tools: []mcp.MCPTool{
			{Name: "tool1", Description: "T1"},
			{Name: "tool2", Description: "T2"},
		},
	}
	transport := &mockTransport{
		responses: [][]byte{
			makeJSONRPCResp(1, initResult),
			makeJSONRPCResp(3, listResult),
		},
	}
	client := mcp.NewClient("test-srv", transport)
	registry := tools.NewToolRegistry()

	count, err := registerMCPTools(context.Background(), client, registry)
	if err != nil {
		t.Fatalf("registerMCPTools failed: %v", err)
	}
	if count != 2 {
		t.Errorf("expected 2 tools registered, got %d", count)
	}
}
