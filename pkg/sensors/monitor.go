// SWE100821: Embodied cognition — sensor monitor for physical-world awareness.
// Reads from configurable sensors (stub for now), tracks rolling buffers,
// checks thresholds, and formats readings for the agent's system prompt.

package sensors

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"
)

// SensorConfig defines a single sensor to monitor.
type SensorConfig struct {
	Name       string  `json:"name" yaml:"name"`
	Type       string  `json:"type" yaml:"type"`             // "temperature", "humidity", "light", etc.
	Pin        int     `json:"pin" yaml:"pin"`                // GPIO/I2C pin
	Interval   int     `json:"interval" yaml:"interval"`      // polling interval in seconds
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
// SWE100821: Avoids importing the bus package to keep sensors decoupled.
type MessagePublisher interface {
	PublishOutbound(msg interface{})
}

// SensorMonitor polls configured sensors, buffers readings, and fires alerts.
type SensorMonitor struct {
	configs  []SensorConfig
	readings map[string][]SensorReading
	bus      MessagePublisher
	mu       sync.RWMutex
	running  bool
	cancel   context.CancelFunc
}

const maxReadingsPerSensor = 60

// NewSensorMonitor creates a monitor for the given sensor configs.
func NewSensorMonitor(configs []SensorConfig) *SensorMonitor {
	return &SensorMonitor{
		configs:  configs,
		readings: make(map[string][]SensorReading, len(configs)),
	}
}

// SetBus attaches a message publisher for threshold alerts.
func (sm *SensorMonitor) SetBus(bus MessagePublisher) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	// SWE100821: Late-bind bus so monitor can be created before bus is ready
	sm.bus = bus
}

// Start begins polling each sensor in its own goroutine.
// Returns immediately; cancel the context or call Stop() to shut down.
func (sm *SensorMonitor) Start(ctx context.Context) {
	sm.mu.Lock()
	if sm.running {
		sm.mu.Unlock()
		return
	}
	sm.running = true
	ctx, sm.cancel = context.WithCancel(ctx)
	sm.mu.Unlock()

	for _, cfg := range sm.configs {
		go sm.pollSensor(ctx, cfg)
	}
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
func (sm *SensorMonitor) ForSystemPrompt() string {
	latest := sm.GetLatest()
	if len(latest) == 0 {
		return ""
	}

	// SWE100821: Compact format to minimize token usage in system prompt
	var b strings.Builder
	b.WriteString("[Sensors]\n")
	for _, r := range latest {
		fmt.Fprintf(&b, "  %s: %.1f %s (at %s)\n",
			r.Name, r.Value, r.Unit, r.Timestamp.Format("15:04:05"))
	}
	return b.String()
}

// pollSensor reads a single sensor at the configured interval.
func (sm *SensorMonitor) pollSensor(ctx context.Context, cfg SensorConfig) {
	interval := time.Duration(cfg.Interval) * time.Second
	if interval <= 0 {
		interval = 10 * time.Second
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			reading := sm.readSensor(cfg)
			sm.storeReading(reading)
			sm.checkThresholds(cfg, reading)
		}
	}
}

// readSensor returns a reading from the sensor.
// SWE100821: Stub implementation — real I2C/GPIO reads are platform-specific.
func (sm *SensorMonitor) readSensor(cfg SensorConfig) SensorReading {
	unit := unitForType(cfg.Type)
	return SensorReading{
		Name:      cfg.Name,
		Value:     0,
		Unit:      unit,
		Timestamp: time.Now(),
	}
}

// storeReading appends a reading, keeping a rolling buffer of maxReadingsPerSensor.
func (sm *SensorMonitor) storeReading(r SensorReading) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	buf := sm.readings[r.Name]
	buf = append(buf, r)
	// SWE100821: Rolling buffer — drop oldest when at capacity
	if len(buf) > maxReadingsPerSensor {
		buf = buf[len(buf)-maxReadingsPerSensor:]
	}
	sm.readings[r.Name] = buf
}

// checkThresholds publishes an alert if the reading crosses configured bounds.
func (sm *SensorMonitor) checkThresholds(cfg SensorConfig, r SensorReading) {
	sm.mu.RLock()
	pub := sm.bus
	sm.mu.RUnlock()

	if pub == nil {
		return
	}

	// SWE100821: Fire alert when value is outside [AlertBelow, AlertAbove]
	if cfg.AlertAbove > 0 && r.Value > cfg.AlertAbove {
		pub.PublishOutbound(map[string]interface{}{
			"type":    "sensor_alert",
			"sensor":  r.Name,
			"value":   r.Value,
			"unit":    r.Unit,
			"message": fmt.Sprintf("%s reading %.1f %s exceeds upper threshold %.1f", r.Name, r.Value, r.Unit, cfg.AlertAbove),
		})
	}
	if cfg.AlertBelow != 0 && r.Value < cfg.AlertBelow {
		pub.PublishOutbound(map[string]interface{}{
			"type":    "sensor_alert",
			"sensor":  r.Name,
			"value":   r.Value,
			"unit":    r.Unit,
			"message": fmt.Sprintf("%s reading %.1f %s below lower threshold %.1f", r.Name, r.Value, r.Unit, cfg.AlertBelow),
		})
	}
}

// unitForType returns a default unit string for common sensor types.
func unitForType(sensorType string) string {
	switch sensorType {
	case "temperature":
		return "°C"
	case "humidity":
		return "%"
	case "light":
		return "lux"
	case "pressure":
		return "hPa"
	case "distance":
		return "cm"
	default:
		return ""
	}
}
