// SWE100821: Tests for the voice loop — verifies construction and agent binding.
// Does not test audio I/O (requires hardware).
package voice

import (
	"context"
	"testing"
)

// SWE100821: stubAgent implements AgentProcessor for test use
type stubAgent struct{}

func (s *stubAgent) ProcessDirect(_ context.Context, _, _ string) (string, error) {
	return "ok", nil
}

// SWE100821: TestNewVoiceLoop — verify creation without panic
func TestNewVoiceLoop(t *testing.T) {
	dir := t.TempDir()
	transcriber := NewGroqTranscriber("") // SWE100821: empty key — won't call API
	vl := NewVoiceLoop(transcriber, dir)

	if vl == nil {
		t.Fatal("NewVoiceLoop returned nil")
	}
	if vl.transcriber == nil {
		t.Error("transcriber should be set")
	}
	if vl.tts == nil {
		t.Error("tts should be set")
	}
}

// SWE100821: TestSetAgent — verify agent can be set without error
func TestSetAgent(t *testing.T) {
	dir := t.TempDir()
	transcriber := NewGroqTranscriber("")
	vl := NewVoiceLoop(transcriber, dir)

	vl.SetAgent(&stubAgent{})

	vl.mu.Lock()
	hasAgent := vl.agentLoop != nil
	vl.mu.Unlock()

	if !hasAgent {
		t.Error("expected agentLoop to be set after SetAgent")
	}
}
