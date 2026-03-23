// SWE100821: Phone sensor source — reads sensor data from a USB-attached Android phone via ADB.
// Polls: accelerometer, light, pressure, proximity, step counter, battery, screen state.
// Graceful failure: if ADB disconnects, marks unavailable and retries on next poll.

package sensors

import (
	"context"
	"fmt"
	"math"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"
)

// PhoneSource reads sensors from an Android phone via ADB shell commands.
type PhoneSource struct {
	adbPath   string
	serial    string // empty = auto-detect first device
	available bool
	mu        sync.RWMutex
	lastCheck time.Time
	failures  int
}

// NewPhoneSource creates a phone sensor source. Pass empty serial for auto-detect.
func NewPhoneSource(adbPath, serial string) *PhoneSource {
	if adbPath == "" {
		adbPath = "adb"
	}
	return &PhoneSource{adbPath: adbPath, serial: serial}
}

func (ps *PhoneSource) Name() string { return "phone" }

func (ps *PhoneSource) Interval() time.Duration { return 30 * time.Second }

func (ps *PhoneSource) Available() bool {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	// Rate-limit availability checks to every 30s
	if time.Since(ps.lastCheck) < 30*time.Second {
		return ps.available
	}
	ps.lastCheck = time.Now()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	out, err := ps.adbCmd(ctx, "get-state")
	ps.available = err == nil && strings.TrimSpace(out) == "device"
	if ps.available {
		ps.failures = 0
	}
	return ps.available
}

func (ps *PhoneSource) Poll(ctx context.Context) ([]SensorReading, error) {
	if !ps.Available() {
		return nil, fmt.Errorf("phone not connected via ADB")
	}

	now := time.Now()
	var readings []SensorReading
	var errs []string

	// SWE100821: Run sensor reads in parallel — each is independent, total poll time = max(reads)
	type result struct {
		readings []SensorReading
		err      error
	}
	ch := make(chan result, 5)

	go func() { r, e := ps.readBattery(ctx, now); ch <- result{r, e} }()
	go func() { r, e := ps.readLightSensor(ctx, now); ch <- result{r, e} }()
	go func() { r, e := ps.readAccelerometer(ctx, now); ch <- result{r, e} }()
	go func() { r, e := ps.readScreenState(ctx, now); ch <- result{r, e} }()
	go func() { r, e := ps.readPressure(ctx, now); ch <- result{r, e} }()

	for i := 0; i < 5; i++ {
		res := <-ch
		if res.err != nil {
			errs = append(errs, res.err.Error())
			continue
		}
		readings = append(readings, res.readings...)
	}

	ps.mu.Lock()
	if len(readings) == 0 && len(errs) > 0 {
		ps.failures++
		// Back off after repeated failures
		if ps.failures > 5 {
			ps.available = false
		}
	} else {
		ps.failures = 0
	}
	ps.mu.Unlock()

	if len(errs) > 0 && len(readings) == 0 {
		return nil, fmt.Errorf("all phone sensors failed: %s", strings.Join(errs, "; "))
	}
	return readings, nil
}

func (ps *PhoneSource) readBattery(ctx context.Context, now time.Time) ([]SensorReading, error) {
	out, err := ps.shell(ctx, "dumpsys battery")
	if err != nil {
		return nil, err
	}
	var readings []SensorReading
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "level:") {
			if v, err := strconv.ParseFloat(strings.TrimSpace(strings.TrimPrefix(line, "level:")), 64); err == nil {
				readings = append(readings, SensorReading{Name: "phone/battery", Value: v, Unit: "%", Timestamp: now})
			}
		}
		if strings.HasPrefix(line, "temperature:") {
			if v, err := strconv.ParseFloat(strings.TrimSpace(strings.TrimPrefix(line, "temperature:")), 64); err == nil {
				readings = append(readings, SensorReading{Name: "phone/temperature", Value: v / 10.0, Unit: "°C", Timestamp: now})
			}
		}
	}
	return readings, nil
}

func (ps *PhoneSource) readLightSensor(ctx context.Context, now time.Time) ([]SensorReading, error) {
	// SWE100821: dumpsys sensorservice gives last known values for on-change sensors
	out, err := ps.shell(ctx, "dumpsys sensorservice | grep -A 3 'light'")
	if err != nil {
		return nil, err
	}
	// Parse "last : <value>" pattern
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if strings.Contains(line, "last") && strings.Contains(line, "<") {
			// Format: "last : <value>"
			start := strings.Index(line, "<")
			end := strings.Index(line, ">")
			if start >= 0 && end > start {
				valStr := line[start+1 : end]
				parts := strings.Fields(valStr)
				if len(parts) > 0 {
					if v, err := strconv.ParseFloat(parts[0], 64); err == nil {
						return []SensorReading{{Name: "phone/light", Value: v, Unit: "lux", Timestamp: now}}, nil
					}
				}
			}
		}
	}
	return nil, fmt.Errorf("light sensor data not found")
}

func (ps *PhoneSource) readAccelerometer(ctx context.Context, now time.Time) ([]SensorReading, error) {
	out, err := ps.shell(ctx, "dumpsys sensorservice | grep -A 3 -i 'accelerometer'")
	if err != nil {
		return nil, err
	}
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if strings.Contains(line, "last") && strings.Contains(line, "<") {
			start := strings.Index(line, "<")
			end := strings.Index(line, ">")
			if start >= 0 && end > start {
				valStr := line[start+1 : end]
				parts := strings.Fields(valStr)
				if len(parts) >= 3 {
					x, _ := strconv.ParseFloat(strings.TrimSuffix(parts[0], ","), 64)
					y, _ := strconv.ParseFloat(strings.TrimSuffix(parts[1], ","), 64)
					z, _ := strconv.ParseFloat(strings.TrimSuffix(parts[2], ","), 64)
					magnitude := math.Sqrt(x*x + y*y + z*z)
					// Classify motion state from acceleration magnitude (9.8 = resting)
					state := "stationary"
					if math.Abs(magnitude-9.8) > 2.0 {
						state = "moving"
					}
					return []SensorReading{
						{Name: "phone/motion", Value: magnitude, Unit: "m/s² (" + state + ")", Timestamp: now},
					}, nil
				}
			}
		}
	}
	return nil, fmt.Errorf("accelerometer data not found")
}

func (ps *PhoneSource) readScreenState(ctx context.Context, now time.Time) ([]SensorReading, error) {
	out, err := ps.shell(ctx, "dumpsys power | grep 'mWakefulness\\|mScreenOn\\|Display Power'")
	if err != nil {
		return nil, err
	}
	screenOn := 0.0
	for _, line := range strings.Split(out, "\n") {
		if strings.Contains(line, "Awake") || strings.Contains(line, "mScreenOn=true") {
			screenOn = 1.0
		}
	}
	label := "off"
	if screenOn > 0 {
		label = "on"
	}
	return []SensorReading{{Name: "phone/screen", Value: screenOn, Unit: label, Timestamp: now}}, nil
}

func (ps *PhoneSource) readPressure(ctx context.Context, now time.Time) ([]SensorReading, error) {
	out, err := ps.shell(ctx, "dumpsys sensorservice | grep -A 3 -i 'pressure'")
	if err != nil {
		return nil, err
	}
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if strings.Contains(line, "last") && strings.Contains(line, "<") {
			start := strings.Index(line, "<")
			end := strings.Index(line, ">")
			if start >= 0 && end > start {
				valStr := line[start+1 : end]
				parts := strings.Fields(valStr)
				if len(parts) > 0 {
					if v, err := strconv.ParseFloat(parts[0], 64); err == nil && v > 800 && v < 1200 {
						return []SensorReading{{Name: "phone/pressure", Value: v, Unit: "hPa", Timestamp: now}}, nil
					}
				}
			}
		}
	}
	return nil, fmt.Errorf("pressure sensor data not found")
}

func (ps *PhoneSource) shell(ctx context.Context, cmd string) (string, error) {
	args := []string{}
	if ps.serial != "" {
		args = append(args, "-s", ps.serial)
	}
	args = append(args, "shell", cmd)
	return ps.run(ctx, args...)
}

func (ps *PhoneSource) adbCmd(ctx context.Context, cmd string) (string, error) {
	args := []string{}
	if ps.serial != "" {
		args = append(args, "-s", ps.serial)
	}
	args = append(args, cmd)
	return ps.run(ctx, args...)
}

func (ps *PhoneSource) run(ctx context.Context, args ...string) (string, error) {
	c := exec.CommandContext(ctx, ps.adbPath, args...)
	out, err := c.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("adb %s: %w", strings.Join(args, " "), err)
	}
	return string(out), nil
}
