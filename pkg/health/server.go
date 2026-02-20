// health/server.go - SWE100821
// Lightweight HTTP health check server for systemd liveness/readiness probes.
// Provides /healthz (liveness), /readyz (readiness), and /metricsz (basic metrics).
package health

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Dawgomatic/Xagent/pkg/hwprofile"
	"github.com/Dawgomatic/Xagent/pkg/logger"
)

// ReadinessChecker is called by /readyz to verify external deps (e.g., Ollama).
type ReadinessChecker func() error

// Metrics tracks basic request-level counters for observability.
// SWE100821: Lightweight alternative to Prometheus for embedded systems.
type Metrics struct {
	mu              sync.Mutex
	MessagesTotal   int64         `json:"messages_total"`
	MessagesErrored int64         `json:"messages_errored"`
	LLMCallsTotal   int64         `json:"llm_calls_total"`
	LLMCallsFailed  int64         `json:"llm_calls_failed"`
	LLMLatencySum   time.Duration `json:"-"` // sum for avg calculation
	ToolCallsTotal  int64         `json:"tool_calls_total"`
}

// Snapshot returns a JSON-serializable copy of current metrics.
func (m *Metrics) Snapshot() map[string]interface{} {
	m.mu.Lock()
	defer m.mu.Unlock()
	avgLatencyMs := float64(0)
	if m.LLMCallsTotal > 0 {
		avgLatencyMs = float64(m.LLMLatencySum.Milliseconds()) / float64(m.LLMCallsTotal)
	}
	return map[string]interface{}{
		"messages_total":     m.MessagesTotal,
		"messages_errored":   m.MessagesErrored,
		"llm_calls_total":    m.LLMCallsTotal,
		"llm_calls_failed":   m.LLMCallsFailed,
		"llm_avg_latency_ms": avgLatencyMs,
		"tool_calls_total":   m.ToolCallsTotal,
	}
}

// IncMessage increments message counter.
func (m *Metrics) IncMessage()      { atomic.AddInt64(&m.MessagesTotal, 1) }
// IncMessageError increments errored message counter.
func (m *Metrics) IncMessageError() { atomic.AddInt64(&m.MessagesErrored, 1) }
// IncToolCall increments tool call counter.
func (m *Metrics) IncToolCall()     { atomic.AddInt64(&m.ToolCallsTotal, 1) }

// RecordLLMCall records an LLM call with its latency.
func (m *Metrics) RecordLLMCall(latency time.Duration, failed bool) {
	atomic.AddInt64(&m.LLMCallsTotal, 1)
	if failed {
		atomic.AddInt64(&m.LLMCallsFailed, 1)
	}
	m.mu.Lock()
	m.LLMLatencySum += latency
	m.mu.Unlock()
}

// Server exposes /healthz, /readyz, and /metricsz on a dedicated port.
type Server struct {
	httpServer *http.Server
	ready      atomic.Bool
	startTime  time.Time
	checkers   []ReadinessChecker
	metrics    *Metrics
}

// NewServer creates a health server on host:port.
// Checkers are optional functions called by /readyz.
func NewServer(host string, port int, checkers ...ReadinessChecker) *Server {
	s := &Server{
		startTime: time.Now(),
		checkers:  checkers,
		metrics:   &Metrics{},
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", s.handleHealthz)
	mux.HandleFunc("/readyz", s.handleReadyz)
	mux.HandleFunc("/metricsz", s.handleMetricsz)
	// SWE100821: Hardware profile endpoint for autonomous compute-tier detection
	mux.HandleFunc("/hwprofile", s.handleHWProfile)

	s.httpServer = &http.Server{
		Addr:         fmt.Sprintf("%s:%d", host, port),
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
	}

	return s
}

// GetMetrics returns the metrics instance for external callers to record events.
func (s *Server) GetMetrics() *Metrics {
	return s.metrics
}

// SetReady marks the service as ready to serve traffic.
func (s *Server) SetReady(ready bool) {
	s.ready.Store(ready)
}

// Start begins serving in a goroutine. Returns immediately.
func (s *Server) Start() {
	go func() {
		logger.InfoCF("health", "Health server listening", map[string]interface{}{
			"addr": s.httpServer.Addr,
		})
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.ErrorCF("health", "Health server error", map[string]interface{}{
				"error": err.Error(),
			})
		}
	}()
}

// Stop gracefully shuts down the health server.
func (s *Server) Stop() {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	s.httpServer.Shutdown(ctx)
}

// handleHealthz returns 200 if process is alive (liveness probe).
func (s *Server) handleHealthz(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "ok",
		"uptime": time.Since(s.startTime).String(),
	})
}

// handleReadyz returns 200 if all readiness checks pass, 503 otherwise.
func (s *Server) handleReadyz(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if !s.ready.Load() {
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]string{"status": "not_ready"})
		return
	}

	// Run readiness checkers
	for _, check := range s.checkers {
		if err := check(); err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"status": "unhealthy",
				"error":  err.Error(),
			})
			return
		}
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ready"})
}

// handleMetricsz returns basic operational metrics as JSON.
// SWE100821: Lightweight metrics for embedded/edge systems without Prometheus.
func (s *Server) handleMetricsz(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	snap := s.metrics.Snapshot()
	snap["uptime_seconds"] = int(time.Since(s.startTime).Seconds())
	// SWE100821: Include hardware tier + available RAM in metrics for monitoring
	p := hwprofile.Detect()
	snap["hw_tier"] = string(p.Tier)
	snap["hw_ram_avail_mb"] = p.RAMAvailMB
	snap["hw_cpu_cores"] = p.CPUCores
	json.NewEncoder(w).Encode(snap)
}

// handleHWProfile returns the full hardware fingerprint and tuning recommendations.
// SWE100821: Autonomous hardware detection for scalable high→low compute.
func (s *Server) handleHWProfile(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	profile := hwprofile.Detect()
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(profile.AsMap())
}

// OllamaChecker returns a ReadinessChecker that pings Ollama at the given base URL.
// SWE100821: Verifies the LLM backend is reachable before marking ready.
func OllamaChecker(apiBase string) ReadinessChecker {
	return func() error {
		client := &http.Client{Timeout: 3 * time.Second}
		// Ollama exposes a root endpoint that returns "Ollama is running"
		resp, err := client.Get(apiBase)
		if err != nil {
			return fmt.Errorf("ollama unreachable at %s: %w", apiBase, err)
		}
		resp.Body.Close()
		if resp.StatusCode >= 500 {
			return fmt.Errorf("ollama returned status %d", resp.StatusCode)
		}
		return nil
	}
}
