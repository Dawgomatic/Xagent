// hwprofile/hwprofile.go - SWE100821
// Runtime hardware fingerprinting and compute-tier classification.
// Reads Linux proc/sys to detect CPU, RAM, GPU, disk, and platform type.
// Zero external dependencies — pure Go + Linux kernel interfaces.
package hwprofile

import (
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

// Tier classifies overall compute capability.
type Tier string

const (
	TierMinimal Tier = "minimal" // RPi3, <=1 GB RAM
	TierLow     Tier = "low"     // RPi4, 2-4 GB RAM
	TierMid     Tier = "mid"     // x86_64, 8-16 GB RAM
	TierHigh    Tier = "high"    // x86_64, 32+ GB RAM, many cores
	TierGPU     Tier = "gpu"     // NVIDIA GPU detected with VRAM
)

// GPUInfo holds detected GPU details.
type GPUInfo struct {
	Detected bool   `json:"detected"`
	Name     string `json:"name,omitempty"`
	VRAM_MB  int    `json:"vram_mb,omitempty"` // dedicated VRAM in MB
	Driver   string `json:"driver,omitempty"`
	IsTegra  bool   `json:"is_tegra,omitempty"` // Jetson Xavier/Orin
}

// Profile is the runtime hardware fingerprint.
type Profile struct {
	Tier        Tier   `json:"tier"`
	Platform    string `json:"platform"` // xavier, rpi3, rpi4, x86_64
	Arch        string `json:"arch"`     // amd64, arm64
	CPUModel    string `json:"cpu_model"`
	CPUCores    int    `json:"cpu_cores"`    // logical cores
	CPUFreqMHz  int    `json:"cpu_freq_mhz"` // max freq, 0 if unknown
	RAMTotalMB  int    `json:"ram_total_mb"`
	RAMAvailMB  int    `json:"ram_avail_mb"`
	DiskFreeMB  int    `json:"disk_free_mb"`
	GPU         GPUInfo `json:"gpu"`
	DetectedAt  time.Time `json:"detected_at"`
}

// Recommendation holds tuning parameters derived from the hardware tier.
// SWE100821: Maps hardware capability to optimal system configuration.
type Recommendation struct {
	OllamaModel       string  `json:"ollama_model"`
	MaxTokens         int     `json:"max_tokens"`
	Temperature       float64 `json:"temperature"`
	MaxToolIterations int     `json:"max_tool_iterations"`
	MaxSubagents      int     `json:"max_subagents"`
	MessageTimeoutSec int     `json:"message_timeout_sec"`
	SessionPruneHours int     `json:"session_prune_hours"`
	BusBufferSize     int     `json:"bus_buffer_size"`
}

// cached singleton
var (
	cached   *Profile
	cacheMu  sync.Mutex
	cacheAge time.Time
	cacheTTL = 5 * time.Minute
)

// Detect performs hardware detection. Results are cached for 5 minutes.
// Safe to call frequently — returns cached Profile after first probe.
func Detect() *Profile {
	cacheMu.Lock()
	defer cacheMu.Unlock()

	if cached != nil && time.Since(cacheAge) < cacheTTL {
		return cached
	}

	p := &Profile{
		Arch:       runtime.GOARCH,
		CPUCores:   runtime.NumCPU(),
		DetectedAt: time.Now(),
	}

	p.Platform = detectPlatform()
	p.CPUModel = readCPUModel()
	p.CPUFreqMHz = readCPUMaxFreq()
	p.RAMTotalMB, p.RAMAvailMB = readMemInfo()
	p.DiskFreeMB = readDiskFree()
	p.GPU = detectGPU(p.Platform)
	p.Tier = classify(p)

	cached = p
	cacheAge = time.Now()
	return p
}

// Recommend returns tuning parameters for the detected hardware tier.
func (p *Profile) Recommend() Recommendation {
	switch p.Tier {
	case TierGPU:
		model := "llama3.1:8b"
		if p.GPU.VRAM_MB >= 16000 {
			model = "llama3.1:70b"
		} else if p.GPU.VRAM_MB >= 8000 {
			model = "llama3.1:8b"
		}
		return Recommendation{
			OllamaModel:       model,
			MaxTokens:         8192,
			Temperature:       0.7,
			MaxToolIterations: 25,
			MaxSubagents:      5,
			MessageTimeoutSec: 300,
			SessionPruneHours: 168, // 7 days
			BusBufferSize:     256,
		}
	case TierHigh:
		return Recommendation{
			OllamaModel:       "llama3.1:8b",
			MaxTokens:         8192,
			Temperature:       0.7,
			MaxToolIterations: 20,
			MaxSubagents:      4,
			MessageTimeoutSec: 300,
			SessionPruneHours: 168,
			BusBufferSize:     128,
		}
	case TierMid:
		return Recommendation{
			OllamaModel:       "llama3.1:8b",
			MaxTokens:         4096,
			Temperature:       0.7,
			MaxToolIterations: 15,
			MaxSubagents:      3,
			MessageTimeoutSec: 300,
			SessionPruneHours: 72,
			BusBufferSize:     64,
		}
	case TierLow:
		return Recommendation{
			OllamaModel:       "phi3:3.8b",
			MaxTokens:         2048,
			Temperature:       0.5,
			MaxToolIterations: 10,
			MaxSubagents:      1,
			MessageTimeoutSec: 600, // slower hardware, longer timeout
			SessionPruneHours: 24,
			BusBufferSize:     32,
		}
	default: // TierMinimal
		return Recommendation{
			OllamaModel:       "tinyllama:1.1b",
			MaxTokens:         1024,
			Temperature:       0.3,
			MaxToolIterations: 5,
			MaxSubagents:      1,
			MessageTimeoutSec: 900,
			SessionPruneHours: 12,
			BusBufferSize:     16,
		}
	}
}

// Summary returns a human-readable one-liner for logging.
func (p *Profile) Summary() string {
	gpu := "none"
	if p.GPU.Detected {
		gpu = fmt.Sprintf("%s (%dMB)", p.GPU.Name, p.GPU.VRAM_MB)
	}
	return fmt.Sprintf("tier=%s platform=%s cpu=%dx%dMHz ram=%dMB/%dMB disk=%dMB gpu=%s",
		p.Tier, p.Platform, p.CPUCores, p.CPUFreqMHz,
		p.RAMAvailMB, p.RAMTotalMB, p.DiskFreeMB, gpu)
}

// ---- internal detection helpers ----

// detectPlatform identifies Xavier, RPi3, RPi4, or x86_64.
// Reused pattern: start.sh L49-L62 (platform detection).
func detectPlatform() string {
	// Tegra (Jetson Xavier/Orin)
	if _, err := os.Stat("/etc/nv_tegra_release"); err == nil {
		return "xavier"
	}

	// Raspberry Pi via device-tree
	if data, err := os.ReadFile("/proc/device-tree/model"); err == nil {
		model := string(data)
		switch {
		case strings.Contains(model, "Raspberry Pi 4"):
			return "rpi4"
		case strings.Contains(model, "Raspberry Pi 3"):
			return "rpi3"
		case strings.Contains(model, "Raspberry Pi 5"):
			return "rpi5"
		case strings.Contains(model, "Raspberry Pi"):
			return "rpi"
		}
	}

	return "x86_64"
}

// readCPUModel reads first CPU model name from /proc/cpuinfo.
func readCPUModel() string {
	data, err := os.ReadFile("/proc/cpuinfo")
	if err != nil {
		return "unknown"
	}
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, "model name") || strings.HasPrefix(line, "Model") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				return strings.TrimSpace(parts[1])
			}
		}
	}
	return "unknown"
}

// readCPUMaxFreq reads the maximum CPU frequency from cpufreq sysfs.
func readCPUMaxFreq() int {
	// Try cpufreq first (reports in kHz)
	paths := []string{
		"/sys/devices/system/cpu/cpu0/cpufreq/cpuinfo_max_freq",
		"/sys/devices/system/cpu/cpu0/cpufreq/scaling_max_freq",
	}
	for _, p := range paths {
		data, err := os.ReadFile(p)
		if err == nil {
			khz, err := strconv.Atoi(strings.TrimSpace(string(data)))
			if err == nil {
				return khz / 1000 // kHz → MHz
			}
		}
	}

	// Fallback: parse /proc/cpuinfo for "cpu MHz"
	data, err := os.ReadFile("/proc/cpuinfo")
	if err != nil {
		return 0
	}
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, "cpu MHz") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				mhz, err := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
				if err == nil {
					return int(math.Round(mhz))
				}
			}
		}
	}
	return 0
}

// readMemInfo reads total and available RAM from /proc/meminfo.
func readMemInfo() (totalMB, availMB int) {
	data, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		return 0, 0
	}
	for _, line := range strings.Split(string(data), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		kb, _ := strconv.Atoi(fields[1])
		switch fields[0] {
		case "MemTotal:":
			totalMB = kb / 1024
		case "MemAvailable:":
			availMB = kb / 1024
		}
	}
	return
}

// readDiskFree returns available disk space (MB) for the working directory.
func readDiskFree() int {
	wd, err := os.Getwd()
	if err != nil {
		wd = "/"
	}
	var stat syscall.Statfs_t
	if err := syscall.Statfs(wd, &stat); err != nil {
		return 0
	}
	// Available blocks * block size → bytes → MB
	return int(stat.Bavail * uint64(stat.Bsize) / (1024 * 1024))
}

// detectGPU probes for NVIDIA GPU via nvidia-smi or Tegra sysfs.
func detectGPU(platform string) GPUInfo {
	info := GPUInfo{}

	// Tegra (Jetson): shared memory, no discrete VRAM
	if platform == "xavier" {
		info.Detected = true
		info.IsTegra = true
		info.Driver = "tegra"
		info.Name = readTegraGPUName()
		// Tegra uses unified memory; report shared pool from meminfo
		totalMB, _ := readMemInfo()
		info.VRAM_MB = totalMB // shared
		return info
	}

	// nvidia-smi for discrete GPUs
	out, err := exec.Command("nvidia-smi",
		"--query-gpu=name,memory.total,driver_version",
		"--format=csv,noheader,nounits").Output()
	if err != nil {
		return info // no GPU
	}

	line := strings.TrimSpace(string(out))
	// May have multiple GPUs; take first
	lines := strings.SplitN(line, "\n", 2)
	parts := strings.Split(lines[0], ", ")
	if len(parts) >= 3 {
		info.Detected = true
		info.Name = strings.TrimSpace(parts[0])
		vram, _ := strconv.Atoi(strings.TrimSpace(parts[1]))
		info.VRAM_MB = vram
		info.Driver = strings.TrimSpace(parts[2])
	}

	return info
}

// readTegraGPUName reads the Tegra GPU description from /proc.
func readTegraGPUName() string {
	// Try common Jetson sysfs paths
	paths := []string{
		"/sys/devices/gpu.0/devfreq/17000000.ga10b/device/of_node/name",
		"/proc/device-tree/gpu/compatible",
	}
	for _, p := range paths {
		data, err := os.ReadFile(p)
		if err == nil {
			return strings.TrimSpace(strings.ReplaceAll(string(data), "\x00", " "))
		}
	}
	return "Tegra GPU"
}

// classify maps raw metrics to a compute tier.
func classify(p *Profile) Tier {
	// GPU always wins if present
	if p.GPU.Detected && !p.GPU.IsTegra && p.GPU.VRAM_MB > 0 {
		return TierGPU
	}

	// Tegra with lots of unified memory → GPU tier
	if p.GPU.IsTegra && p.RAMTotalMB >= 8000 {
		return TierGPU
	}

	// RAM-based tiers
	switch {
	case p.RAMTotalMB >= 32000:
		return TierHigh
	case p.RAMTotalMB >= 8000:
		return TierMid
	case p.RAMTotalMB >= 2000:
		return TierLow
	default:
		return TierMinimal
	}
}

// AsMap returns the profile as a map[string]interface{} for JSON endpoints.
func (p *Profile) AsMap() map[string]interface{} {
	rec := p.Recommend()
	return map[string]interface{}{
		"tier":     string(p.Tier),
		"platform": p.Platform,
		"arch":     p.Arch,
		"cpu": map[string]interface{}{
			"model":    p.CPUModel,
			"cores":    p.CPUCores,
			"freq_mhz": p.CPUFreqMHz,
		},
		"memory": map[string]interface{}{
			"total_mb":     p.RAMTotalMB,
			"available_mb": p.RAMAvailMB,
		},
		"disk": map[string]interface{}{
			"free_mb": p.DiskFreeMB,
		},
		"gpu": map[string]interface{}{
			"detected": p.GPU.Detected,
			"name":     p.GPU.Name,
			"vram_mb":  p.GPU.VRAM_MB,
			"driver":   p.GPU.Driver,
			"is_tegra": p.GPU.IsTegra,
		},
		"recommendation": map[string]interface{}{
			"ollama_model":        rec.OllamaModel,
			"max_tokens":          rec.MaxTokens,
			"temperature":         rec.Temperature,
			"max_tool_iterations": rec.MaxToolIterations,
			"max_subagents":       rec.MaxSubagents,
			"message_timeout_sec": rec.MessageTimeoutSec,
			"session_prune_hours": rec.SessionPruneHours,
			"bus_buffer_size":     rec.BusBufferSize,
		},
		"detected_at": p.DetectedAt.Format(time.RFC3339),
	}
}

// InvalidateCache forces the next Detect() call to re-probe hardware.
// Useful after hot-plug events or resource changes.
func InvalidateCache() {
	cacheMu.Lock()
	defer cacheMu.Unlock()
	cached = nil
}

// ModelForVRAM returns the best Ollama model name for the given VRAM budget.
// SWE100821: Fine-grained model selection for GPU tiers.
func ModelForVRAM(vramMB int) string {
	switch {
	case vramMB >= 48000:
		return "llama3.1:70b"
	case vramMB >= 24000:
		return "llama3.1:70b-q4_0" // quantized 70B
	case vramMB >= 16000:
		return "llama3.1:8b"
	case vramMB >= 8000:
		return "llama3.1:8b"
	case vramMB >= 4000:
		return "phi3:3.8b"
	default:
		return "tinyllama:1.1b"
	}
}

// WatchResources starts a goroutine that periodically refreshes the profile.
// Calls onChange when the tier changes (e.g., RAM pressure).
// Returns a stop function.
// SWE100821: Enables dynamic scaling under memory pressure.
func WatchResources(interval time.Duration, onChange func(old, new *Profile)) func() {
	stop := make(chan struct{})
	go func() {
		prev := Detect()
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-stop:
				return
			case <-ticker.C:
				InvalidateCache()
				curr := Detect()
				if curr.Tier != prev.Tier && onChange != nil {
					onChange(prev, curr)
				}
				prev = curr
			}
		}
	}()
	return func() { close(stop) }
}

// FindOllamaModels returns the list of locally available Ollama models.
// SWE100821: Used to verify the recommended model is actually pulled.
func FindOllamaModels() []string {
	out, err := exec.Command("ollama", "list").Output()
	if err != nil {
		return nil
	}
	var models []string
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		fields := strings.Fields(line)
		if len(fields) > 0 && fields[0] != "NAME" {
			models = append(models, fields[0])
		}
	}
	return models
}

// BestAvailableModel returns the highest-capability model that is both
// recommended for this tier AND already pulled in Ollama.
// Falls back to the first available model, or the tier's recommendation.
func (p *Profile) BestAvailableModel() string {
	rec := p.Recommend()
	available := FindOllamaModels()
	if len(available) == 0 {
		return rec.OllamaModel
	}

	// Check if recommended model is available
	for _, m := range available {
		if m == rec.OllamaModel {
			return rec.OllamaModel
		}
	}

	// Priority list from largest to smallest
	priority := []string{
		"llama3.1:70b", "llama3.1:70b-q4_0", "llama3.1:8b",
		"phi3:3.8b", "tinyllama:1.1b",
	}
	for _, want := range priority {
		for _, have := range available {
			if have == want {
				return have
			}
		}
	}

	// Fallback: first available model
	return available[0]
}

// ProfilePath returns ~/.xagent/hwprofile.json for persisting profiles.
func ProfilePath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".xagent", "hwprofile.json")
}
