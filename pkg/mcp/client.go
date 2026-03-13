// MCP (Model Context Protocol) client for Xagent.
// Connects to local MCP servers via stdio or HTTP SSE, discovers tools,
// and bridges them into the Xagent tool registry.
// All communication is local — no cloud dependency.

package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/Dawgomatic/Xagent/pkg/logger"
)

// JSON-RPC 2.0 types
type jsonRPCRequest struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      int64       `json:"id"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params,omitempty"`
}

type jsonRPCResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      int64           `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *jsonRPCError   `json:"error,omitempty"`
}

type jsonRPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// MCP protocol types

type ServerInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type InitializeResult struct {
	ProtocolVersion string     `json:"protocolVersion"`
	ServerInfo      ServerInfo `json:"serverInfo"`
}

type MCPTool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	InputSchema map[string]interface{} `json:"inputSchema"`
}

type ListToolsResult struct {
	Tools []MCPTool `json:"tools"`
}

type MCPResource struct {
	URI         string `json:"uri"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	MimeType    string `json:"mimeType,omitempty"`
}

type ListResourcesResult struct {
	Resources []MCPResource `json:"resources"`
}

type ContentItem struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

type CallToolResult struct {
	Content []ContentItem `json:"content"`
	IsError bool          `json:"isError,omitempty"`
}

type ReadResourceResult struct {
	Contents []ContentItem `json:"contents"`
}

// Transport is the interface for MCP communication channels.
type Transport interface {
	Send(ctx context.Context, data []byte) error
	Receive(ctx context.Context) ([]byte, error)
	Close() error
}

// Client is an MCP client that connects to a single MCP server.
type Client struct {
	name       string
	transport  Transport
	nextID     atomic.Int64
	tools      []MCPTool
	resources  []MCPResource
	serverInfo ServerInfo
	mu         sync.RWMutex
}

// NewClient creates a new MCP client with the given transport.
func NewClient(name string, transport Transport) *Client {
	return &Client{
		name:      name,
		transport: transport,
	}
}

// Initialize performs the MCP handshake with the server.
func (c *Client) Initialize(ctx context.Context) error {
	params := map[string]interface{}{
		"protocolVersion": "2024-11-05",
		"capabilities":    map[string]interface{}{},
		"clientInfo": map[string]interface{}{
			"name":    "xagent",
			"version": "1.0.0",
		},
	}

	var result InitializeResult
	if err := c.call(ctx, "initialize", params, &result); err != nil {
		return fmt.Errorf("mcp initialize failed: %w", err)
	}

	c.mu.Lock()
	c.serverInfo = result.ServerInfo
	c.mu.Unlock()

	logger.InfoCF("mcp", "Connected to MCP server",
		map[string]interface{}{
			"server":   result.ServerInfo.Name,
			"version":  result.ServerInfo.Version,
			"protocol": result.ProtocolVersion,
		})

	// Send initialized notification
	_ = c.notify(ctx, "notifications/initialized", nil)

	return nil
}

// DiscoverTools fetches the list of available tools from the server.
func (c *Client) DiscoverTools(ctx context.Context) ([]MCPTool, error) {
	var result ListToolsResult
	if err := c.call(ctx, "tools/list", nil, &result); err != nil {
		return nil, fmt.Errorf("mcp tools/list failed: %w", err)
	}

	c.mu.Lock()
	c.tools = result.Tools
	c.mu.Unlock()

	logger.InfoCF("mcp", "Discovered MCP tools",
		map[string]interface{}{
			"server": c.name,
			"count":  len(result.Tools),
		})

	return result.Tools, nil
}

// DiscoverResources fetches available resources from the server.
func (c *Client) DiscoverResources(ctx context.Context) ([]MCPResource, error) {
	var result ListResourcesResult
	if err := c.call(ctx, "resources/list", nil, &result); err != nil {
		return nil, fmt.Errorf("mcp resources/list failed: %w", err)
	}

	c.mu.Lock()
	c.resources = result.Resources
	c.mu.Unlock()

	return result.Resources, nil
}

// CallTool invokes a tool on the MCP server.
func (c *Client) CallTool(ctx context.Context, name string, arguments map[string]interface{}) (*CallToolResult, error) {
	params := map[string]interface{}{
		"name":      name,
		"arguments": arguments,
	}

	var result CallToolResult
	if err := c.call(ctx, "tools/call", params, &result); err != nil {
		return nil, fmt.Errorf("mcp tools/call %s failed: %w", name, err)
	}

	return &result, nil
}

// ReadResource reads a resource from the MCP server.
func (c *Client) ReadResource(ctx context.Context, uri string) (*ReadResourceResult, error) {
	params := map[string]interface{}{
		"uri": uri,
	}

	var result ReadResourceResult
	if err := c.call(ctx, "resources/read", params, &result); err != nil {
		return nil, fmt.Errorf("mcp resources/read %s failed: %w", uri, err)
	}

	return &result, nil
}

// GetTools returns the cached tool list.
func (c *Client) GetTools() []MCPTool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.tools
}

// GetName returns the client name.
func (c *Client) GetName() string {
	return c.name
}

// Close shuts down the transport.
func (c *Client) Close() error {
	return c.transport.Close()
}

func (c *Client) call(ctx context.Context, method string, params interface{}, result interface{}) error {
	id := c.nextID.Add(1)

	req := jsonRPCRequest{
		JSONRPC: "2.0",
		ID:      id,
		Method:  method,
		Params:  params,
	}

	data, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}

	if err := c.transport.Send(ctx, data); err != nil {
		return fmt.Errorf("send request: %w", err)
	}

	respData, err := c.transport.Receive(ctx)
	if err != nil {
		return fmt.Errorf("receive response: %w", err)
	}

	var resp jsonRPCResponse
	if err := json.Unmarshal(respData, &resp); err != nil {
		return fmt.Errorf("unmarshal response: %w", err)
	}

	if resp.Error != nil {
		return fmt.Errorf("rpc error %d: %s", resp.Error.Code, resp.Error.Message)
	}

	if result != nil && resp.Result != nil {
		if err := json.Unmarshal(resp.Result, result); err != nil {
			return fmt.Errorf("unmarshal result: %w", err)
		}
	}

	return nil
}

func (c *Client) notify(ctx context.Context, method string, params interface{}) error {
	req := jsonRPCRequest{
		JSONRPC: "2.0",
		Method:  method,
		Params:  params,
	}

	data, err := json.Marshal(req)
	if err != nil {
		return err
	}

	return c.transport.Send(ctx, data)
}
