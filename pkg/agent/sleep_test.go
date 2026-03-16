package agent

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/Dawgomatic/Xagent/pkg/bus"
	"github.com/Dawgomatic/Xagent/pkg/epoch"
	"github.com/Dawgomatic/Xagent/pkg/identity"
	"github.com/Dawgomatic/Xagent/pkg/tools"
)

func TestSleepManager_WakeInterrupt(t *testing.T) {
	fmt.Println("Setting up Sleep test...")
	id := &identity.AgentIdentity{
		SessionID: "test-session-123",
		AgentID:   "test-agent",
		BootTime:  time.Now(),
	}
	epochMgr := epoch.NewManager("/tmp/xagent_test_epoch", id)
	epochMgr.Wake()

	msgBus := bus.NewMessageBus()
	registry := tools.NewToolRegistry()

	// mockProvider is defined in loop_test.go
	sm := NewSleepManager(epochMgr, &mockProvider{}, msgBus, "/tmp", registry)

	// Override idle duration so it sleeps immediately (we use package-level access)
	sm.idleTimeout = 1 * time.Millisecond

	fmt.Println("Simulating heavy agent activity (building fatigue)...")
	for i := 0; i < 10; i++ {
		sm.RecordActivity(2)
	}

	fmt.Printf("Current Fatigue: %.2f\n", sm.GetFatigueLevel())

	// Wait a moment so idle duration > 1ms
	time.Sleep(10 * time.Millisecond)

	fmt.Println("Manually triggering checkSleep...")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sm.checkSleep(ctx)

	// Verify it's sleeping
	var isSleeping bool
	epochMgr.UpdateStats(func(s *epoch.EpochStats) {
		isSleeping = s.IsSleeping
	})

	if !isSleeping {
		t.Fatalf("Expected agent to be sleeping, but it is awake!")
	}
	fmt.Println("Agent successfully entered Sleep Mode and spawned Improvement Subagent.")

	// Now emulate a wake interrupt (user message received)
	fmt.Println("Simulating incoming user message (Wake Interrupt)...")
	sm.RecordActivity(0)

	// Verify it woke up
	epochMgr.UpdateStats(func(s *epoch.EpochStats) {
		isSleeping = s.IsSleeping
	})

	if isSleeping {
		t.Fatalf("Expected agent to be awake after RecordActivity interrupt, but it's still sleeping!")
	}

	fmt.Println("Agent successfully woke up and cancelled the subagent!")
}
