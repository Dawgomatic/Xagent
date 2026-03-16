// SWE100821: Tests for the sensor monitor — verifies creation, empty readings,
// and system prompt formatting with no data.
package sensors

import (
	"testing"
)

// SWE100821: TestNewSensorMonitor — create with 2 configs, verify not nil
func TestNewSensorMonitor(t *testing.T) {
	configs := []SensorConfig{
		{Name: "temp", Type: "temperature", Pin: 4, Interval: 10},
		{Name: "humidity", Type: "humidity", Pin: 5, Interval: 15},
	}

	sm := NewSensorMonitor(configs)
	if sm == nil {
		t.Fatal("NewSensorMonitor returned nil")
	}
	if len(sm.configs) != 2 {
		t.Errorf("len(configs) = %d, want 2", len(sm.configs))
	}
}

// SWE100821: TestGetLatest_Empty — no readings recorded, expect empty map
func TestGetLatest_Empty(t *testing.T) {
	sm := NewSensorMonitor([]SensorConfig{
		{Name: "temp", Type: "temperature"},
	})

	latest := sm.GetLatest()
	if len(latest) != 0 {
		t.Errorf("expected empty map, got %d entries", len(latest))
	}
}

// SWE100821: TestForSystemPrompt_NoReadings — verify empty string when no data
func TestForSystemPrompt_NoReadings(t *testing.T) {
	sm := NewSensorMonitor([]SensorConfig{
		{Name: "light", Type: "light"},
	})

	prompt := sm.ForSystemPrompt()
	if prompt != "" {
		t.Errorf("expected empty string, got %q", prompt)
	}
}
