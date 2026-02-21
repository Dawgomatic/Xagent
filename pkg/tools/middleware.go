// SWE100821: Tool middleware layer — pre/post hooks, result caching, and circuit breaker.
// Wraps the ToolRegistry to add cross-cutting concerns without modifying individual tools.

package tools

import (
	"context"
	"crypto/sha256"
	"fmt"
	"sync"
	"time"

	"github.com/Dawgomatic/Xagent/pkg/logger"
)

// MiddlewareHook runs before or after tool execution.
// Pre-hooks can block execution by returning a non-nil ToolResult.
// Post-hooks can modify or observe results.
type MiddlewareHook func(toolName string, args map[string]interface{}, result *ToolResult) *ToolResult

// CircuitState tracks per-tool failure state for the circuit breaker.
type CircuitState struct {
	Failures     int
	LastFailure  time.Time
	Open         bool
	OpenUntil    time.Time
}

// CacheEntry stores a cached tool result with expiration.
type CacheEntry struct {
	Result    *ToolResult
	ExpiresAt time.Time
}

// ToolMiddleware wraps a ToolRegistry with hooks, caching, and circuit breaking.
type ToolMiddleware struct {
	registry       *ToolRegistry
	preHooks       []MiddlewareHook
	postHooks      []MiddlewareHook
	cache          map[string]*CacheEntry
	cacheMu        sync.RWMutex
	cacheTTL       time.Duration
	cacheableTools map[string]bool // SWE100821: tools whose results are deterministic and cacheable
	circuits       map[string]*CircuitState
	circuitMu      sync.RWMutex
	failThreshold  int
	cooldown       time.Duration
	analytics      map[string]*ToolAnalytics
	analyticsMu    sync.RWMutex
}

// ToolAnalytics tracks per-tool usage statistics for tool-use learning.
type ToolAnalytics struct {
	TotalCalls   int           `json:"total_calls"`
	Successes    int           `json:"successes"`
	Failures     int           `json:"failures"`
	TotalLatency time.Duration `json:"total_latency_ms"`
	LastUsed     time.Time     `json:"last_used"`
}

// NewToolMiddleware creates middleware wrapping the given registry.
func NewToolMiddleware(registry *ToolRegistry) *ToolMiddleware {
	return &ToolMiddleware{
		registry:       registry,
		preHooks:       make([]MiddlewareHook, 0),
		postHooks:      make([]MiddlewareHook, 0),
		cache:          make(map[string]*CacheEntry),
		cacheTTL:       5 * time.Minute,
		cacheableTools: map[string]bool{"read_file": true, "list_dir": true, "web_fetch": true},
		circuits:       make(map[string]*CircuitState),
		failThreshold:  3,
		cooldown:       30 * time.Second,
		analytics:      make(map[string]*ToolAnalytics),
	}
}

// AddPreHook registers a hook that runs before tool execution.
func (m *ToolMiddleware) AddPreHook(hook MiddlewareHook) {
	m.preHooks = append(m.preHooks, hook)
}

// AddPostHook registers a hook that runs after tool execution.
func (m *ToolMiddleware) AddPostHook(hook MiddlewareHook) {
	m.postHooks = append(m.postHooks, hook)
}

// SetCacheTTL sets the default cache TTL for cacheable tools.
func (m *ToolMiddleware) SetCacheTTL(ttl time.Duration) {
	m.cacheTTL = ttl
}

// SetCacheableTools overrides which tools have cacheable results.
func (m *ToolMiddleware) SetCacheableTools(tools []string) {
	m.cacheableTools = make(map[string]bool, len(tools))
	for _, t := range tools {
		m.cacheableTools[t] = true
	}
}

// Execute runs a tool through the middleware chain: pre-hooks → cache → circuit → execute → post-hooks.
func (m *ToolMiddleware) Execute(ctx context.Context, name string, args map[string]interface{}, channel, chatID string, cb AsyncCallback) *ToolResult {
	// SWE100821: Pre-hooks can block execution
	for _, hook := range m.preHooks {
		if blocked := hook(name, args, nil); blocked != nil {
			return blocked
		}
	}

	// SWE100821: Circuit breaker check
	if m.isCircuitOpen(name) {
		logger.WarnCF("middleware", "Circuit breaker open, skipping tool",
			map[string]interface{}{"tool": name})
		return ErrorResult(fmt.Sprintf("Tool '%s' temporarily disabled due to repeated failures. Try again shortly.", name))
	}

	// SWE100821: Cache lookup for deterministic tools
	if m.cacheableTools[name] {
		cacheKey := m.cacheKey(name, args)
		if cached := m.getCache(cacheKey); cached != nil {
			logger.DebugCF("middleware", "Cache hit", map[string]interface{}{"tool": name})
			return cached
		}
	}

	// Execute through registry
	start := time.Now()
	result := m.registry.ExecuteWithContext(ctx, name, args, channel, chatID, cb)
	duration := time.Since(start)

	// SWE100821: Update analytics
	m.recordAnalytics(name, result, duration)

	// SWE100821: Update circuit breaker
	if result.IsError {
		m.recordFailure(name)
	} else {
		m.resetCircuit(name)
	}

	// SWE100821: Cache result if tool is cacheable and succeeded
	if m.cacheableTools[name] && !result.IsError {
		cacheKey := m.cacheKey(name, args)
		m.setCache(cacheKey, result)
	}

	// SWE100821: Post-hooks can observe/modify results
	for _, hook := range m.postHooks {
		if modified := hook(name, args, result); modified != nil {
			result = modified
		}
	}

	return result
}

// GetAnalytics returns usage analytics for all tools.
func (m *ToolMiddleware) GetAnalytics() map[string]*ToolAnalytics {
	m.analyticsMu.RLock()
	defer m.analyticsMu.RUnlock()
	cp := make(map[string]*ToolAnalytics, len(m.analytics))
	for k, v := range m.analytics {
		clone := *v
		cp[k] = &clone
	}
	return cp
}

// GetToolHints returns learning-based hints for the system prompt.
// Identifies which tools succeed/fail for various patterns.
func (m *ToolMiddleware) GetToolHints() string {
	m.analyticsMu.RLock()
	defer m.analyticsMu.RUnlock()

	var hints string
	for name, a := range m.analytics {
		if a.TotalCalls < 3 {
			continue
		}
		rate := float64(a.Successes) / float64(a.TotalCalls) * 100
		avgLatency := time.Duration(0)
		if a.TotalCalls > 0 {
			avgLatency = a.TotalLatency / time.Duration(a.TotalCalls)
		}
		if rate < 70 {
			hints += fmt.Sprintf("- %s: %.0f%% success rate (%d calls, avg %dms) — consider alternatives\n",
				name, rate, a.TotalCalls, avgLatency.Milliseconds())
		}
	}
	return hints
}

func (m *ToolMiddleware) cacheKey(name string, args map[string]interface{}) string {
	h := sha256.New()
	h.Write([]byte(name))
	for k, v := range args {
		h.Write([]byte(fmt.Sprintf("%s=%v", k, v)))
	}
	return fmt.Sprintf("%x", h.Sum(nil))[:16]
}

func (m *ToolMiddleware) getCache(key string) *ToolResult {
	m.cacheMu.RLock()
	defer m.cacheMu.RUnlock()
	entry, ok := m.cache[key]
	if !ok || time.Now().After(entry.ExpiresAt) {
		return nil
	}
	return entry.Result
}

func (m *ToolMiddleware) setCache(key string, result *ToolResult) {
	m.cacheMu.Lock()
	defer m.cacheMu.Unlock()
	m.cache[key] = &CacheEntry{
		Result:    result,
		ExpiresAt: time.Now().Add(m.cacheTTL),
	}
	// SWE100821: Evict expired entries if cache grows large
	if len(m.cache) > 200 {
		now := time.Now()
		for k, v := range m.cache {
			if now.After(v.ExpiresAt) {
				delete(m.cache, k)
			}
		}
	}
}

func (m *ToolMiddleware) isCircuitOpen(name string) bool {
	m.circuitMu.RLock()
	defer m.circuitMu.RUnlock()
	cs, ok := m.circuits[name]
	if !ok {
		return false
	}
	if cs.Open && time.Now().After(cs.OpenUntil) {
		return false // cooldown expired, allow retry
	}
	return cs.Open
}

func (m *ToolMiddleware) recordFailure(name string) {
	m.circuitMu.Lock()
	defer m.circuitMu.Unlock()
	cs, ok := m.circuits[name]
	if !ok {
		cs = &CircuitState{}
		m.circuits[name] = cs
	}
	cs.Failures++
	cs.LastFailure = time.Now()
	if cs.Failures >= m.failThreshold {
		cs.Open = true
		cs.OpenUntil = time.Now().Add(m.cooldown)
		logger.WarnCF("middleware", "Circuit breaker opened",
			map[string]interface{}{"tool": name, "failures": cs.Failures})
	}
}

func (m *ToolMiddleware) resetCircuit(name string) {
	m.circuitMu.Lock()
	defer m.circuitMu.Unlock()
	if cs, ok := m.circuits[name]; ok {
		cs.Failures = 0
		cs.Open = false
	}
}

func (m *ToolMiddleware) recordAnalytics(name string, result *ToolResult, duration time.Duration) {
	m.analyticsMu.Lock()
	defer m.analyticsMu.Unlock()
	a, ok := m.analytics[name]
	if !ok {
		a = &ToolAnalytics{}
		m.analytics[name] = a
	}
	a.TotalCalls++
	a.TotalLatency += duration
	a.LastUsed = time.Now()
	if result.IsError {
		a.Failures++
	} else {
		a.Successes++
	}
}

// ClearCache clears all cached tool results.
func (m *ToolMiddleware) ClearCache() {
	m.cacheMu.Lock()
	defer m.cacheMu.Unlock()
	m.cache = make(map[string]*CacheEntry)
}
