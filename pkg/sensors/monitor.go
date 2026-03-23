// SWE100821: Perception subsystem — multi-source sensor monitor.
// Discovers and polls all available sensor sources (phone, host, I2C, USB).
// Each source runs in its own goroutine with independent failure tracking and retry.
// Readings are buffered per-sensor and formatted for injection into the agent's system prompt.

package sensors

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/Dawgomatic/Xagent/pkg/logger"
)

// SensorConfig defines a single sensor to monitor (legacy config-based sensors).
type SensorConfig struct {
	Name       string  `json:"name" yaml:"name"`
	Type       string  `json:"type" yaml:"type"`
	Pin        int     `json:"pin" yaml:"pin"`
	Interval   int     `json:"interval" yaml:"interval"`
	AlertAbove float64 `json:"alert_above" yaml:"alert_above"`
	AlertBelow float64 `json:"alert_below" yaml:"alert_below"`
}

// SensorReading is a single timestamped sensor measurement.
type SensorReading struct {
	Name      string    `json:"name"`
	Value     float64   `json:"value"`
	Unit      string    `json:"unit"`
	Timestamp time.Time `json:"timestamp"`
}

// MessagePublisher is a minimal interface for publishing alerts.
type MessagePublisher interface {
	PublishOutbound(msg interface{})
}

// sourceState tracks per-source health for graceful retry.
type sourceState struct {
	source       SensorSource
	failures     int
	lastFailure  time.Time
	lastSuccess  time.Time
	backoffUntil time.Time
	disabled     bool // SWE100821: permanently disabled after repeated failures
}

// SensorMonitor polls discovered sensor sources, buffers readings, and fires alerts.
type SensorMonitor struct {
	sources  []*sourceState
	readings map[string][]SensorReading
	bus      MessagePublisher
	mu       sync.RWMutex
	running  bool
	cancel   context.CancelFunc

	// SWE100821: Alert thresholds — keyed by sensor name prefix
	thresholds map[string]threshold

	// SWE100821: Re-discovery interval — detect hot-plugged sensors
	rediscoverInterval time.Duration

	// SWE100821: Camera source reference for event-triggered captures
	cameraSource *CameraSource
	triggerCtx   context.Context // parent context for event-triggered goroutines
}

type threshold struct {
	above float64
	below float64
}

const maxReadingsPerSensor = 60

// SWE100821: NewSensorMonitor creates a monitor. Configs param kept for API compat
// but unused — thresholds are hardcoded defaults. Call DiscoverAndStart to begin polling.
func NewSensorMonitor(_ []SensorConfig) *SensorMonitor {
	return &SensorMonitor{
		readings:           make(map[string][]SensorReading),
		thresholds:         defaultThresholds(),
		rediscoverInterval: 5 * time.Minute,
	}
}

// SWE100821: Sensible defaults — alert when things are actually interesting
func defaultThresholds() map[string]threshold {
	return map[string]threshold{
		"host/":             {above: 85, below: 0},   // CPU/GPU temp > 85°C
		"phone/battery":     {above: 0, below: 15},    // battery < 15%
		"phone/temperature": {above: 45, below: 0},    // phone overheating
		"host/ram_used":     {above: 90, below: 0},    // RAM > 90%
		"host/disk_root":    {above: 90, below: 0},    // disk > 90%
		"host/load_1m":      {above: 8, below: 0},     // high load
	}
}

// SetBus attaches a message publisher for threshold alerts.
func (sm *SensorMonitor) SetBus(bus MessagePublisher) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.bus = bus
}

// AddSource adds a sensor source manually (for testing or non-discovered sources).
func (sm *SensorMonitor) AddSource(src SensorSource) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.sources = append(sm.sources, &sourceState{source: src})
}

// SWE100821: GetCameraSource returns the discovered camera source (or nil).
// Used by cmd_gateway to create the CameraTool.
func (sm *SensorMonitor) GetCameraSource() *CameraSource {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.cameraSource
}

// DiscoverAndStart runs auto-discovery then starts polling all found sources.
func (sm *SensorMonitor) DiscoverAndStart(ctx context.Context) {
	sm.mu.Lock()
	if sm.running {
		sm.mu.Unlock()
		return
	}
	sm.running = true
	ctx, sm.cancel = context.WithCancel(ctx)
	sm.triggerCtx = ctx // SWE100821: store for event-triggered goroutines
	sm.mu.Unlock()

	// Initial discovery
	sm.runDiscovery()

	// Start polling each source
	sm.mu.RLock()
	for _, ss := range sm.sources {
		go sm.pollSource(ctx, ss)
	}
	sm.mu.RUnlock()

	// Periodic re-discovery for hot-plugged devices
	go sm.rediscoveryLoop(ctx)
}

// Start begins polling (backward-compatible with old API — uses discovery internally).
func (sm *SensorMonitor) Start(ctx context.Context) {
	sm.DiscoverAndStart(ctx)
}

// Stop halts all sensor polling goroutines.
func (sm *SensorMonitor) Stop() {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	if sm.cancel != nil {
		sm.cancel()
	}
	sm.running = false
}

// GetLatest returns the most recent reading for each sensor.
func (sm *SensorMonitor) GetLatest() map[string]SensorReading {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	result := make(map[string]SensorReading, len(sm.readings))
	for name, buf := range sm.readings {
		if len(buf) > 0 {
			result[name] = buf[len(buf)-1]
		}
	}
	return result
}

// ForSystemPrompt formats current readings as a text block for the agent's context.
// Groups readings by source for readability. Omits stale data (>5 min old).
func (sm *SensorMonitor) ForSystemPrompt() string {
	latest := sm.GetLatest()
	if len(latest) == 0 {
		return ""
	}

	// SWE100821: Drop stale readings — if a source died, don't show its last values forever.
	// Camera ambient polls every 60min, event captures are ad-hoc — allow 90min staleness
	// for camera readings, 5min for everything else.
	cutoff := time.Now().Add(-5 * time.Minute)
	cameraCutoff := time.Now().Add(-90 * time.Minute)
	for k, r := range latest {
		// SWE100821: Use longer staleness window for camera readings
		c := cutoff
		if strings.HasPrefix(k, "camera/") {
			c = cameraCutoff
		}
		if r.Timestamp.Before(c) {
			delete(latest, k)
		}
	}
	if len(latest) == 0 {
		return ""
	}

	// Group by source prefix (before '/')
	groups := make(map[string][]SensorReading)
	for _, r := range latest {
		prefix := r.Name
		if i := strings.Index(r.Name, "/"); i >= 0 {
			prefix = r.Name[:i]
		}
		groups[prefix] = append(groups[prefix], r)
	}

	// Sort groups for deterministic output
	var sortedGroups []string
	for g := range groups {
		sortedGroups = append(sortedGroups, g)
	}
	sort.Strings(sortedGroups)

	var b strings.Builder
	b.WriteString("[Perception — Sensor Readings]\n")
	for _, group := range sortedGroups {
		readings := groups[group]
		sort.Slice(readings, func(i, j int) bool { return readings[i].Name < readings[j].Name })
		b.WriteString(fmt.Sprintf("  %s:\n", group))
		for _, r := range readings {
			shortName := r.Name
			if i := strings.Index(r.Name, "/"); i >= 0 {
				shortName = r.Name[i+1:]
			}
			// SWE100821: Text readings (camera scene, status labels) use Unit as the value;
			// numeric readings show Value + Unit normally.
			if shortName == "scene" || r.Unit == "active" || r.Unit == "present" || r.Unit == "on" || r.Unit == "off" {
				b.WriteString(fmt.Sprintf("    %s: %s\n", shortName, r.Unit))
			} else {
				b.WriteString(fmt.Sprintf("    %s: %.1f %s\n", shortName, r.Value, r.Unit))
			}
		}
	}

	// SWE100821: Add source health summary
	sm.mu.RLock()
	var downSources []string
	for _, ss := range sm.sources {
		if ss.failures > 3 {
			downSources = append(downSources, ss.source.Name())
		}
	}
	sm.mu.RUnlock()
	if len(downSources) > 0 {
		b.WriteString(fmt.Sprintf("  [!] Sources with issues: %s\n", strings.Join(downSources, ", ")))
	}

	return b.String()
}

// SourceSummary returns a human-readable summary of discovered sources and their health.
func (sm *SensorMonitor) SourceSummary() string {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	if len(sm.sources) == 0 {
		return "No sensor sources discovered"
	}

	var parts []string
	for _, ss := range sm.sources {
		status := "ok"
		if ss.failures > 0 {
			status = fmt.Sprintf("%d failures", ss.failures)
		}
		if !ss.backoffUntil.IsZero() && time.Now().Before(ss.backoffUntil) {
			status = "backing off"
		}
		parts = append(parts, fmt.Sprintf("%s(%s)", ss.source.Name(), status))
	}
	return strings.Join(parts, ", ")
}

// pollSource reads from a single source at its preferred interval with retry/backoff.
func (sm *SensorMonitor) pollSource(ctx context.Context, ss *sourceState) {
	interval := ss.source.Interval()
	if interval <= 0 {
		interval = 30 * time.Second
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// SWE100821: Skip permanently disabled sources
			sm.mu.RLock()
			disabled := ss.disabled
			backoff := ss.backoffUntil
			sm.mu.RUnlock()
			if disabled {
				return
			}
			if !backoff.IsZero() && time.Now().Before(backoff) {
				continue
			}

			pollCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
			readings, err := ss.source.Poll(pollCtx)
			cancel()

			sm.mu.Lock()
			if err != nil {
				ss.failures++
				ss.lastFailure = time.Now()

				// SWE100821: After 5 consecutive failures, disable the source to stop log spam.
				// Empty I2C buses and unplugged sensors will never recover on their own.
				if ss.failures >= 5 {
					ss.disabled = true
					sm.mu.Unlock()
					logger.InfoCF("perception", "Source disabled after repeated failures", map[string]interface{}{
						"source":   ss.source.Name(),
						"failures": ss.failures,
					})
					return
				}

				backoffDur := time.Duration(1<<min(ss.failures, 4)) * 30 * time.Second
				if backoffDur > 5*time.Minute {
					backoffDur = 5 * time.Minute
				}
				ss.backoffUntil = time.Now().Add(backoffDur)
				sm.mu.Unlock()

				if ss.failures <= 3 {
					logger.WarnCF("perception", "Source poll failed", map[string]interface{}{
						"source":   ss.source.Name(),
						"failures": ss.failures,
						"backoff":  backoffDur.String(),
						"error":    err.Error(),
					})
				}
				continue
			}

			// Success — reset failure tracking
			ss.failures = 0
			ss.lastSuccess = time.Now()
			ss.backoffUntil = time.Time{}
			sm.mu.Unlock()

			// Store readings, check thresholds, and evaluate camera triggers
			for _, r := range readings {
				sm.storeReading(r)
				sm.checkThresholds(r)
				sm.checkCameraTriggers(r) // SWE100821: event-driven camera captures
			}
		}
	}
}

func (sm *SensorMonitor) runDiscovery() {
	result := Discover()
	sm.mu.Lock()
	defer sm.mu.Unlock()

	// Add newly discovered sources (avoid duplicates by name)
	existing := make(map[string]bool, len(sm.sources))
	for _, ss := range sm.sources {
		existing[ss.source.Name()] = true
	}
	for _, src := range result.Sources {
		if !existing[src.Name()] {
			sm.sources = append(sm.sources, &sourceState{source: src})
			logger.InfoCF("perception", "New source added", map[string]interface{}{"source": src.Name()})
		}
		// SWE100821: Capture typed reference to camera source for event triggers
		if cs, ok := src.(*CameraSource); ok && sm.cameraSource == nil {
			sm.cameraSource = cs
		}
	}
}

func (sm *SensorMonitor) rediscoveryLoop(ctx context.Context) {
	ticker := time.NewTicker(sm.rediscoverInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			oldCount := len(sm.sources)
			sm.runDiscovery()

			sm.mu.RLock()
			newCount := len(sm.sources)
			sm.mu.RUnlock()

			if newCount > oldCount {
				// Start polling new sources
				sm.mu.RLock()
				for _, ss := range sm.sources[oldCount:] {
					go sm.pollSource(ctx, ss)
				}
				sm.mu.RUnlock()
				logger.InfoCF("perception", "Hot-plug: new sources started", map[string]interface{}{
					"new": newCount - oldCount,
				})
			}
		}
	}
}

// SWE100821: checkCameraTriggers fires an event-driven camera capture when sensor
// readings indicate an interesting environmental change. Runs in a goroutine to avoid
// blocking the polling loop. The 5-min cooldown lives in CameraSource.EventCapture.
func (sm *SensorMonitor) checkCameraTriggers(r SensorReading) {
	sm.mu.RLock()
	cs := sm.cameraSource
	triggerCtx := sm.triggerCtx
	sm.mu.RUnlock()

	if cs == nil || triggerCtx == nil {
		return
	}

	trigger := false

	switch r.Name {
	case "phone/motion":
		// Trigger when motion state changes significantly (delta > 2 m/s²)
		sm.mu.RLock()
		prev := sm.previousReading(r.Name)
		sm.mu.RUnlock()
		if prev != nil {
			delta := r.Value - prev.Value
			if delta > 2.0 || delta < -2.0 {
				trigger = true
				logger.InfoCF("perception", "Camera trigger: motion change", map[string]interface{}{
					"prev": prev.Value, "now": r.Value,
				})
			}
		}

	case "phone/light":
		// Trigger on >50% relative change in light level
		sm.mu.RLock()
		prev := sm.previousReading(r.Name)
		sm.mu.RUnlock()
		if prev != nil && prev.Value > 0 {
			ratio := r.Value / prev.Value
			if ratio > 1.5 || ratio < 0.5 {
				trigger = true
				logger.InfoCF("perception", "Camera trigger: light change", map[string]interface{}{
					"prev": prev.Value, "now": r.Value,
				})
			}
		}

	case "phone/screen":
		// Trigger on any screen state change (someone picked up / put down phone)
		sm.mu.RLock()
		prev := sm.previousReading(r.Name)
		sm.mu.RUnlock()
		if prev != nil && prev.Value != r.Value {
			trigger = true
			logger.InfoCF("perception", "Camera trigger: screen state change", map[string]interface{}{
				"prev": prev.Value, "now": r.Value,
			})
		}
	}

	if !trigger {
		return
	}

	go func() {
		readings, ok := cs.EventCapture(triggerCtx)
		if !ok {
			return
		}
		for _, er := range readings {
			sm.storeReading(er)
		}
	}()
}

// SWE100821: previousReading returns the second-to-last reading for a sensor (for delta checks).
// Must be called with sm.mu held (RLock).
func (sm *SensorMonitor) previousReading(name string) *SensorReading {
	buf := sm.readings[name]
	if len(buf) < 2 {
		return nil
	}
	r := buf[len(buf)-2]
	return &r
}

// storeReading appends a reading, keeping a rolling buffer.
func (sm *SensorMonitor) storeReading(r SensorReading) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	buf := sm.readings[r.Name]
	buf = append(buf, r)
	if len(buf) > maxReadingsPerSensor {
		buf = buf[len(buf)-maxReadingsPerSensor:]
	}
	sm.readings[r.Name] = buf
}

// checkThresholds publishes an alert if the reading crosses configured bounds.
func (sm *SensorMonitor) checkThresholds(r SensorReading) {
	sm.mu.RLock()
	pub := sm.bus
	sm.mu.RUnlock()

	if pub == nil {
		return
	}

	for prefix, t := range sm.thresholds {
		if !strings.HasPrefix(r.Name, prefix) && r.Name != strings.TrimSuffix(prefix, "/") {
			continue
		}
		if t.above > 0 && r.Value > t.above {
			pub.PublishOutbound(map[string]interface{}{
				"type":    "sensor_alert",
				"sensor":  r.Name,
				"value":   r.Value,
				"unit":    r.Unit,
				"message": fmt.Sprintf("ALERT: %s = %.1f %s (threshold: >%.1f)", r.Name, r.Value, r.Unit, t.above),
			})
		}
		if t.below > 0 && r.Value < t.below {
			pub.PublishOutbound(map[string]interface{}{
				"type":    "sensor_alert",
				"sensor":  r.Name,
				"value":   r.Value,
				"unit":    r.Unit,
				"message": fmt.Sprintf("ALERT: %s = %.1f %s (threshold: <%.1f)", r.Name, r.Value, r.Unit, t.below),
			})
		}
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// unitForType returns a default unit string for common sensor types.
// SWE100821: unitForType removed — dead code, never called from any source file.
