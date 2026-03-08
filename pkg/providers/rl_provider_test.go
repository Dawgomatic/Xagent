package providers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRLProvider_SetSessionContext(t *testing.T) {
	p := NewRLProvider("http://localhost:30000/v1", "test-key", "qwen3-4b")

	// Default context
	ctx := p.GetSessionContext()
	if ctx.SessionID != "unknown" {
		t.Errorf("expected default session_id 'unknown', got %s", ctx.SessionID)
	}
	if ctx.TurnType != RLTurnSide {
		t.Errorf("expected default turn_type 'side', got %s", ctx.TurnType)
	}

	// Set custom context
	p.SetSessionContext(&RLSessionContext{
		SessionID:   "telegram:12345",
		TurnType:    RLTurnMain,
		SessionDone: true,
	})

	ctx = p.GetSessionContext()
	if ctx.SessionID != "telegram:12345" {
		t.Errorf("expected session_id 'telegram:12345', got %s", ctx.SessionID)
	}
	if ctx.TurnType != RLTurnMain {
		t.Errorf("expected turn_type 'main', got %s", ctx.TurnType)
	}
	if !ctx.SessionDone {
		t.Error("expected session_done=true")
	}
}

func TestRLProvider_HeadersSentCorrectly(t *testing.T) {
	// Track headers received by the mock server
	var receivedHeaders http.Header

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedHeaders = r.Header.Clone()

		// Return a valid OpenAI-compatible response
		response := map[string]interface{}{
			"choices": []map[string]interface{}{
				{
					"message": map[string]interface{}{
						"content": "Hello from RL server",
					},
					"finish_reason": "stop",
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	p := NewRLProvider(server.URL, "test-api-key", "qwen3-4b")
	p.SetSessionContext(&RLSessionContext{
		SessionID:   "test-session-42",
		TurnType:    RLTurnMain,
		SessionDone: false,
	})

	messages := []Message{
		{Role: "user", Content: "Hello"},
	}

	resp, err := p.Chat(context.Background(), messages, nil, "", nil)
	if err != nil {
		t.Fatalf("Chat failed: %v", err)
	}

	// Verify response
	if resp.Content != "Hello from RL server" {
		t.Errorf("expected content 'Hello from RL server', got %s", resp.Content)
	}

	// Verify RL headers
	if receivedHeaders.Get("X-Session-Id") != "test-session-42" {
		t.Errorf("expected X-Session-Id 'test-session-42', got %s", receivedHeaders.Get("X-Session-Id"))
	}
	if receivedHeaders.Get("X-Turn-Type") != "main" {
		t.Errorf("expected X-Turn-Type 'main', got %s", receivedHeaders.Get("X-Turn-Type"))
	}
	if receivedHeaders.Get("X-Session-Done") != "" {
		t.Errorf("expected no X-Session-Done header when false, got %s", receivedHeaders.Get("X-Session-Done"))
	}

	// Verify auth header
	if receivedHeaders.Get("Authorization") != "Bearer test-api-key" {
		t.Errorf("expected Authorization 'Bearer test-api-key', got %s", receivedHeaders.Get("Authorization"))
	}
}

func TestRLProvider_SessionDoneHeader(t *testing.T) {
	var receivedHeaders http.Header

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedHeaders = r.Header.Clone()
		response := map[string]interface{}{
			"choices": []map[string]interface{}{
				{"message": map[string]interface{}{"content": "bye"}, "finish_reason": "stop"},
			},
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	p := NewRLProvider(server.URL, "", "qwen3-4b")
	p.SetSessionContext(&RLSessionContext{
		SessionID:   "session-done-test",
		TurnType:    RLTurnMain,
		SessionDone: true,
	})

	_, err := p.Chat(context.Background(), []Message{{Role: "user", Content: "bye"}}, nil, "", nil)
	if err != nil {
		t.Fatalf("Chat failed: %v", err)
	}

	if receivedHeaders.Get("X-Session-Done") != "true" {
		t.Errorf("expected X-Session-Done 'true', got '%s'", receivedHeaders.Get("X-Session-Done"))
	}
}

func TestRLProvider_SideTurnType(t *testing.T) {
	var receivedHeaders http.Header

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedHeaders = r.Header.Clone()
		response := map[string]interface{}{
			"choices": []map[string]interface{}{
				{"message": map[string]interface{}{"content": "tool result"}, "finish_reason": "stop"},
			},
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	p := NewRLProvider(server.URL, "", "qwen3-4b")
	p.SetSessionContext(&RLSessionContext{
		SessionID: "heartbeat-session",
		TurnType:  RLTurnSide,
	})

	_, err := p.Chat(context.Background(), []Message{{Role: "user", Content: "heartbeat check"}}, nil, "", nil)
	if err != nil {
		t.Fatalf("Chat failed: %v", err)
	}

	if receivedHeaders.Get("X-Turn-Type") != "side" {
		t.Errorf("expected X-Turn-Type 'side', got '%s'", receivedHeaders.Get("X-Turn-Type"))
	}
}

func TestRLProvider_ToolCallsParsed(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := map[string]interface{}{
			"choices": []map[string]interface{}{
				{
					"message": map[string]interface{}{
						"content": "",
						"tool_calls": []map[string]interface{}{
							{
								"id":   "call_123",
								"type": "function",
								"function": map[string]interface{}{
									"name":      "read_file",
									"arguments": `{"path": "/tmp/test.txt"}`,
								},
							},
						},
					},
					"finish_reason": "tool_calls",
				},
			},
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	p := NewRLProvider(server.URL, "", "qwen3-4b")
	p.SetSessionContext(&RLSessionContext{SessionID: "tool-test", TurnType: RLTurnMain})

	resp, err := p.Chat(context.Background(), []Message{{Role: "user", Content: "read file"}}, nil, "", nil)
	if err != nil {
		t.Fatalf("Chat failed: %v", err)
	}

	if len(resp.ToolCalls) != 1 {
		t.Fatalf("expected 1 tool call, got %d", len(resp.ToolCalls))
	}

	tc := resp.ToolCalls[0]
	if tc.ID != "call_123" {
		t.Errorf("expected tool call ID 'call_123', got %s", tc.ID)
	}
	if tc.Name != "read_file" {
		t.Errorf("expected tool name 'read_file', got %s", tc.Name)
	}
	if tc.Arguments["path"] != "/tmp/test.txt" {
		t.Errorf("expected path arg '/tmp/test.txt', got %v", tc.Arguments["path"])
	}
}

func TestRLProvider_GetDefaultModel(t *testing.T) {
	p := NewRLProvider("http://localhost:30000/v1", "", "qwen3-8b")
	if p.GetDefaultModel() != "qwen3-8b" {
		t.Errorf("expected default model 'qwen3-8b', got %s", p.GetDefaultModel())
	}
}

func TestRLProvider_EmptyServerURL(t *testing.T) {
	p := NewRLProvider("", "", "qwen3-4b")
	_, err := p.Chat(context.Background(), []Message{{Role: "user", Content: "hello"}}, nil, "", nil)
	if err == nil {
		t.Error("expected error for empty server URL")
	}
}

func TestRLProvider_503RetryOnWeightUpdate(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if calls <= 2 {
			w.WriteHeader(503)
			w.Write([]byte(`{"detail":"submission paused for weight update"}`))
			return
		}
		response := map[string]interface{}{
			"choices": []map[string]interface{}{
				{"message": map[string]interface{}{"content": "success after retry"}, "finish_reason": "stop"},
			},
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	p := NewRLProvider(server.URL, "", "qwen3-4b")
	p.SetSessionContext(&RLSessionContext{SessionID: "retry-test", TurnType: RLTurnMain})

	resp, err := p.Chat(context.Background(), []Message{{Role: "user", Content: "hello"}}, nil, "", nil)
	if err != nil {
		t.Fatalf("expected success after retry, got error: %v", err)
	}

	if resp.Content != "success after retry" {
		t.Errorf("expected 'success after retry', got '%s'", resp.Content)
	}

	if calls != 3 {
		t.Errorf("expected 3 calls (2 retries + 1 success), got %d", calls)
	}
}
