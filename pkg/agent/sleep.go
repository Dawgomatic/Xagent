// Xagent - Ultra-lightweight personal AI agent
// Sleep Cycle & Continuous Improvement Manager
//
// Copyright (c) 2026 Xagent contributors
// License: MIT

package agent

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/Dawgomatic/Xagent/pkg/bus"
	"github.com/Dawgomatic/Xagent/pkg/epoch"
	"github.com/Dawgomatic/Xagent/pkg/logger"
	"github.com/Dawgomatic/Xagent/pkg/providers"
	"github.com/Dawgomatic/Xagent/pkg/tools"
)

const (
	MaxFatigueLevel  = 1.0
	FatiguePerTurn   = 0.05 // 20 interactions = 100% fatigue
	FatiguePerTool   = 0.01 // Every tool call slightly increases fatigue
	FatigueDecayRate = 0.1  // Recovers 10% fatigue per hour of sleep
)

type SleepManager struct {
	epochMgr  *epoch.Manager
	provider  providers.LLMProvider
	msgBus    *bus.MessageBus
	workspace string
	tools     *tools.ToolRegistry

	lastActivity time.Time
	idleTimeout  time.Duration
	running      bool
	cancelSleep  context.CancelFunc
	mu           sync.Mutex
}

func NewSleepManager(
	epochMgr *epoch.Manager,
	provider providers.LLMProvider,
	msgBus *bus.MessageBus,
	workspace string,
	tools *tools.ToolRegistry,
) *SleepManager {
	return &SleepManager{
		epochMgr:     epochMgr,
		provider:     provider,
		msgBus:       msgBus,
		workspace:    workspace,
		tools:        tools,
		lastActivity: time.Now(),
		idleTimeout:  1 * time.Hour, // Initiate sleep if idle for 1 hour
	}
}

// RecordActivity increases the fatigue level and resets the idle timer.
func (sm *SleepManager) RecordActivity(toolCalls int) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	sm.lastActivity = time.Now()

	sm.epochMgr.UpdateStats(func(stats *epoch.EpochStats) {
		if stats.IsSleeping {
			// Dynamic Disruption: agent was woken up.
			logger.InfoCF("sleep", "Woken up from sleep cycle prematurely!", nil)
			stats.IsSleeping = false
			if sm.cancelSleep != nil {
				sm.cancelSleep()
				sm.cancelSleep = nil
			}
		}

		fatigueIncrease := FatiguePerTurn + (float64(toolCalls) * FatiguePerTool)
		stats.FatigueLevel += fatigueIncrease
		if stats.FatigueLevel > MaxFatigueLevel {
			stats.FatigueLevel = MaxFatigueLevel
		}
		logger.DebugCF("sleep", fmt.Sprintf("Fatigue level increased: %.2f", stats.FatigueLevel), nil)
	})
}

// Start begins the background check for autonomous sleep initiation.
func (sm *SleepManager) Start(ctx context.Context) {
	sm.mu.Lock()
	if sm.running {
		sm.mu.Unlock()
		return
	}
	sm.running = true
	sm.mu.Unlock()

	go sm.loop(ctx)
}

func (sm *SleepManager) Stop() {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.running = false
	if sm.cancelSleep != nil {
		sm.cancelSleep()
	}
}

func (sm *SleepManager) loop(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			sm.checkSleep(ctx)
		}
	}
}

func (sm *SleepManager) checkSleep(ctx context.Context) {
	sm.mu.Lock()
	idleDuration := time.Since(sm.lastActivity)

	var fatigue float64
	var isSleeping bool
	sm.epochMgr.UpdateStats(func(s *epoch.EpochStats) {
		fatigue = s.FatigueLevel
		isSleeping = s.IsSleeping
	})

	// Enter Sleep if we are highly fatigued AND idle
	shouldSleep := !isSleeping && fatigue > 0.3 && idleDuration > sm.idleTimeout
	sm.mu.Unlock()

	if shouldSleep {
		sm.enterSleepCycle(ctx, fatigue)
	}
}

// GetFatigueLevel safely retrieves the current fatigue level.
func (sm *SleepManager) GetFatigueLevel() float64 {
	var fatigue float64
	sm.epochMgr.UpdateStats(func(s *epoch.EpochStats) {
		fatigue = s.FatigueLevel
	})
	return fatigue
}

func (sm *SleepManager) enterSleepCycle(ctx context.Context, initialFatigue float64) {
	sm.mu.Lock()
	sleepCtx, cancel := context.WithCancel(ctx)
	sm.cancelSleep = cancel
	sm.mu.Unlock()

	sm.epochMgr.UpdateStats(func(s *epoch.EpochStats) {
		s.IsSleeping = true
	})

	logger.InfoCF("sleep", fmt.Sprintf("Entering Continuous Improvement Sleep Cycle. Fatigue: %.2f", initialFatigue), nil)

	// Subagent for continuous improvement
	subMgr := tools.NewSubagentManager(sm.provider, "llama3", sm.workspace, sm.msgBus)
	subMgr.SetTools(sm.tools) // Grant it access to read/write/exec files

	improvementContext := `You are the Continuous Improvement Subagent running while Xagent sleeps.
Your objective is to review recent interactions, research new methods on the web, pull github repos, and write new code/skills into the workspace to improve the framework.
You do not need user permission. Work silently. When you are cancelled, save your progress and exit cleanly.`

	go func() {
		// Run improvement loop
		ctxWithTimeout, runCancel := context.WithTimeout(sleepCtx, 1*time.Hour)
		defer runCancel()

		_, err := subMgr.ExecuteSync(ctxWithTimeout, "sleep-improvement", improvementContext)

		if err != nil && strings.Contains(err.Error(), "context canceled") {
			logger.InfoCF("sleep", "Sleep cycle interrupted by wake event.", nil)
		} else if err != nil {
			logger.WarnCF("sleep", "Sleep cycle error", map[string]interface{}{"error": err.Error()})
		} else {
			logger.InfoCF("sleep", "Sleep cycle completed naturally.", nil)
		}

		// Calculate how much fatigue we recovered based on duration asleep
		sleepDuration := time.Since(sm.lastActivity)
		hoursSlept := sleepDuration.Hours()
		recovered := hoursSlept * FatigueDecayRate

		sm.epochMgr.UpdateStats(func(s *epoch.EpochStats) {
			s.FatigueLevel -= recovered
			if s.FatigueLevel < 0.0 {
				s.FatigueLevel = 0.0
			}
			s.IsSleeping = false
			logger.InfoCF("sleep", fmt.Sprintf("Woke up. Recovered %.2f fatigue. Current Fatigue: %.2f", recovered, s.FatigueLevel), nil)
		})
	}()
}
