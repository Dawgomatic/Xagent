// SWE100821: Tests for hardware-aware task routing — verifies GPU keyword
// detection, local handling fallback, and peer selection by tier.
package agent2agent

import (
	"context"
	"testing"
)

// SWE100821: TestContainsGPUKeywords — positive and negative cases
func TestContainsGPUKeywords(t *testing.T) {
	cases := []struct {
		input string
		want  bool
	}{
		{"train with cuda", true},   // SWE100821: containsGPUKeywords expects pre-lowered input
		{"fine-tune the llm", true}, // SWE100821: contains "fine-tune"
		{"read a file", false},
		{"send an email", false},
		{"gpu inference needed", true},
	}
	for _, tc := range cases {
		got := containsGPUKeywords(tc.input)
		if got != tc.want {
			t.Errorf("containsGPUKeywords(%q) = %v, want %v", tc.input, got, tc.want)
		}
	}
}

// SWE100821: TestRoute_LocalHandling — high-tier agent with no peers handles locally
func TestRoute_LocalHandling(t *testing.T) {
	hub := NewA2AHub("local-agent")
	router := NewTaskRouter(hub, "high")

	_, routedTo, err := router.Route(context.Background(), "summarize this document")
	if err != nil {
		t.Fatalf("Route() error: %v", err)
	}
	if routedTo != "local" {
		t.Errorf("routedTo = %q, want \"local\"", routedTo)
	}
}

// SWE100821: TestBestPeerForTier — add peers to hub, verify correct selection
func TestBestPeerForTier(t *testing.T) {
	hub := NewA2AHub("local-agent")
	hub.RegisterPeer("peer-a", "http://10.0.0.1:8080") // SWE100821: default tier assumed "medium"
	hub.RegisterPeer("peer-b", "http://10.0.0.2:8080")

	router := NewTaskRouter(hub, "low")

	// SWE100821: BestPeerForTier("medium") should find a peer (default rank assumed medium)
	peerID, ok := router.BestPeerForTier("medium")
	if !ok {
		t.Fatal("expected to find a peer for tier medium")
	}
	if peerID != "peer-a" && peerID != "peer-b" {
		t.Errorf("unexpected peer %q", peerID)
	}

	// SWE100821: With no peers of tier "gpu", should not find one
	hub2 := NewA2AHub("solo")
	router2 := NewTaskRouter(hub2, "low")
	_, ok = router2.BestPeerForTier("gpu")
	if ok {
		t.Error("expected no peer for tier gpu when hub is empty")
	}
}
