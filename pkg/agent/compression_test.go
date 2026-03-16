// SWE100821: Tests for context compression (CompressHistory).

package agent

import (
	"context"
	"testing"

	"github.com/Dawgomatic/Xagent/pkg/providers"
)

// SWE100821: short history (≤ recentWindowSize) returns all messages untouched
func TestCompressHistory_ShortHistory(t *testing.T) {
	cc := NewContextCompressor(&mockProvider{}, nil, "mock", "")
	cc.SetRecentWindowSize(6)

	msgs := []providers.Message{
		{Role: "user", Content: "hello"},
		{Role: "assistant", Content: "hi"},
	}

	compressed, recent, err := cc.CompressHistory(context.Background(), msgs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if compressed != "" {
		t.Errorf("expected empty compressed, got %q", compressed)
	}
	if len(recent) != len(msgs) {
		t.Errorf("expected %d recent messages, got %d", len(msgs), len(recent))
	}
}

// SWE100821: long history splits into compressed + recent
func TestCompressHistory_LongHistory(t *testing.T) {
	cc := NewContextCompressor(&mockProvider{}, nil, "mock", "")
	cc.SetRecentWindowSize(2)

	msgs := []providers.Message{
		{Role: "user", Content: "msg1"},
		{Role: "assistant", Content: "resp1"},
		{Role: "user", Content: "msg2"},
		{Role: "assistant", Content: "resp2"},
		{Role: "user", Content: "msg3"},
		{Role: "assistant", Content: "resp3"},
	}

	compressed, recent, err := cc.CompressHistory(context.Background(), msgs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// SWE100821: mockProvider returns "Mock response" as compressed summary
	if compressed == "" {
		t.Error("expected non-empty compressed output for long history")
	}
	if len(recent) != 2 {
		t.Errorf("expected 2 recent messages, got %d", len(recent))
	}
	if recent[0].Content != "msg3" {
		t.Errorf("expected recent[0] to be msg3, got %q", recent[0].Content)
	}
}

// SWE100821: system messages in older portion should be skipped during compression
func TestCompressHistory_SkipsSystemMessages(t *testing.T) {
	cc := NewContextCompressor(&mockProvider{}, nil, "mock", "")
	cc.SetRecentWindowSize(1)

	msgs := []providers.Message{
		{Role: "system", Content: "You are a helpful assistant"},
		{Role: "system", Content: "Additional system instructions"},
		{Role: "user", Content: "hello"},
	}

	compressed, recent, err := cc.CompressHistory(context.Background(), msgs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// SWE100821: only system messages in older portion → nothing to compress
	if compressed != "" {
		t.Errorf("expected empty compressed when only system msgs in older portion, got %q", compressed)
	}
	if len(recent) != 1 {
		t.Errorf("expected 1 recent message, got %d", len(recent))
	}
}

// SWE100821: exact window size boundary — no compression needed
func TestCompressHistory_ExactWindowSize(t *testing.T) {
	cc := NewContextCompressor(&mockProvider{}, nil, "mock", "")
	cc.SetRecentWindowSize(3)

	msgs := []providers.Message{
		{Role: "user", Content: "a"},
		{Role: "assistant", Content: "b"},
		{Role: "user", Content: "c"},
	}

	compressed, recent, err := cc.CompressHistory(context.Background(), msgs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if compressed != "" {
		t.Errorf("expected empty compressed at exact window size, got %q", compressed)
	}
	if len(recent) != 3 {
		t.Errorf("expected 3 recent messages, got %d", len(recent))
	}
}
