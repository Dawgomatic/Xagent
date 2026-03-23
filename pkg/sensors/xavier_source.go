// SWE100821: Xavier/host sensor source — reads thermals, GPU temp, CPU usage, RAM, disk.
// Works on any Linux host, with extra Tegra-specific thermal zones on Jetson Xavier.
// Graceful failure: individual reads can fail without killing the whole source.

package sensors

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// XavierSource reads system sensors from the host Linux machine via sysfs and /proc.
type XavierSource struct {
	thermalZones []string // discovered thermal zone paths
}

// NewXavierSource creates a host sensor source.
func NewXavierSource() *XavierSource {
	return &XavierSource{}
}

func (xs *XavierSource) Name() string { return "host" }

func (xs *XavierSource) Interval() time.Duration { return 30 * time.Second }

func (xs *XavierSource) Available() bool {
	return runtime.GOOS == "linux"
}

func (xs *XavierSource) Poll(ctx context.Context) ([]SensorReading, error) {
	if !xs.Available() {
		return nil, fmt.Errorf("host sensors only available on Linux")
	}

	now := time.Now()
	var readings []SensorReading

	// Discover thermal zones on first poll
	if len(xs.thermalZones) == 0 {
		xs.discoverThermalZones()
	}

	// Thermal zones (CPU, GPU, board temps)
	for _, zone := range xs.thermalZones {
		if r, err := xs.readThermalZone(zone, now); err == nil {
			readings = append(readings, r)
		}
	}

	// RAM usage
	if r, err := xs.readMemory(now); err == nil {
		readings = append(readings, r...)
	}

	// Disk usage
	if r, err := xs.readDisk(now); err == nil {
		readings = append(readings, r)
	}

	// CPU load average
	if r, err := xs.readLoadAvg(now); err == nil {
		readings = append(readings, r)
	}

	// GPU utilization (Tegra)
	if r, err := xs.readTegraGPU(now); err == nil {
		readings = append(readings, r)
	}

	if len(readings) == 0 {
		return nil, fmt.Errorf("no host sensor readings available")
	}
	return readings, nil
}

func (xs *XavierSource) discoverThermalZones() {
	matches, err := filepath.Glob("/sys/class/thermal/thermal_zone*/type")
	if err != nil {
		return
	}
	for _, typePath := range matches {
		data, err := os.ReadFile(typePath)
		if err != nil {
			continue
		}
		name := strings.TrimSpace(string(data))
		// Only include useful zones, skip generic ones
		if isUsefulThermalZone(name) {
			zonePath := filepath.Dir(typePath)
			xs.thermalZones = append(xs.thermalZones, zonePath)
		}
	}
}

func isUsefulThermalZone(name string) bool {
	lower := strings.ToLower(name)
	useful := []string{"cpu", "gpu", "board", "soc", "tboard", "tdiode", "pmic", "thermal", "iwlwifi"}
	for _, u := range useful {
		if strings.Contains(lower, u) {
			return true
		}
	}
	return false
}

func (xs *XavierSource) readThermalZone(zonePath string, now time.Time) (SensorReading, error) {
	typeData, err := os.ReadFile(filepath.Join(zonePath, "type"))
	if err != nil {
		return SensorReading{}, err
	}
	tempData, err := os.ReadFile(filepath.Join(zonePath, "temp"))
	if err != nil {
		return SensorReading{}, err
	}

	name := strings.TrimSpace(string(typeData))
	tempStr := strings.TrimSpace(string(tempData))
	tempMilliC, err := strconv.ParseFloat(tempStr, 64)
	if err != nil {
		return SensorReading{}, err
	}

	return SensorReading{
		Name:      "host/" + name,
		Value:     tempMilliC / 1000.0,
		Unit:      "°C",
		Timestamp: now,
	}, nil
}

func (xs *XavierSource) readMemory(now time.Time) ([]SensorReading, error) {
	data, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		return nil, err
	}

	var totalKB, availKB float64
	for _, line := range strings.Split(string(data), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		v, _ := strconv.ParseFloat(fields[1], 64)
		switch fields[0] {
		case "MemTotal:":
			totalKB = v
		case "MemAvailable:":
			availKB = v
		}
	}
	if totalKB == 0 {
		return nil, fmt.Errorf("could not parse meminfo")
	}

	usedPct := (1.0 - availKB/totalKB) * 100.0
	return []SensorReading{
		{Name: "host/ram_used", Value: usedPct, Unit: fmt.Sprintf("%% (%.0fMB/%.0fMB)", (totalKB-availKB)/1024, totalKB/1024), Timestamp: now},
	}, nil
}

func (xs *XavierSource) readDisk(now time.Time) (SensorReading, error) {
	// Reused: pkg/hwprofile/hwprofile.go L312-L324 — same syscall.Statfs pattern
	var stat syscall.Statfs_t
	if err := syscall.Statfs("/", &stat); err != nil {
		return SensorReading{}, err
	}
	total := float64(stat.Blocks) * float64(stat.Bsize)
	free := float64(stat.Bavail) * float64(stat.Bsize)
	if total == 0 {
		return SensorReading{}, fmt.Errorf("zero total disk")
	}
	usedPct := (1.0 - free/total) * 100.0
	return SensorReading{
		Name:      "host/disk_root",
		Value:     usedPct,
		Unit:      fmt.Sprintf("%% (%.1fGB free)", free/1e9),
		Timestamp: now,
	}, nil
}

func (xs *XavierSource) readLoadAvg(now time.Time) (SensorReading, error) {
	data, err := os.ReadFile("/proc/loadavg")
	if err != nil {
		return SensorReading{}, err
	}
	fields := strings.Fields(string(data))
	if len(fields) < 1 {
		return SensorReading{}, fmt.Errorf("empty loadavg")
	}
	load, err := strconv.ParseFloat(fields[0], 64)
	if err != nil {
		return SensorReading{}, err
	}
	return SensorReading{Name: "host/load_1m", Value: load, Unit: "", Timestamp: now}, nil
}

func (xs *XavierSource) readTegraGPU(now time.Time) (SensorReading, error) {
	// Tegra GPU load is in /sys/devices/gpu.0/load (or similar)
	paths := []string{
		"/sys/devices/gpu.0/load",
		"/sys/devices/57000000.gpu/load",
		"/sys/devices/17000000.gv11b/load",
	}
	for _, p := range paths {
		data, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		v, err := strconv.ParseFloat(strings.TrimSpace(string(data)), 64)
		if err != nil {
			continue
		}
		// Tegra reports load as 0-1000
		return SensorReading{Name: "host/gpu_load", Value: v / 10.0, Unit: "%", Timestamp: now}, nil
	}
	return SensorReading{}, fmt.Errorf("no Tegra GPU load path found")
}
