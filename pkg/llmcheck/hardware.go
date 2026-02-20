package llmcheck

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"strconv"
	"strings"
)

type CPUInfo struct {
	Brand         string  `json:"brand"`
	Cores         int     `json:"cores"`
	PhysicalCores int     `json:"physical_cores"`
	SpeedMHz      float64 `json:"speed_mhz"`
	Architecture  string  `json:"architecture"`
	HasAVX2       bool    `json:"has_avx2"`
	HasAVX512     bool    `json:"has_avx512"`
	HasNEON       bool    `json:"has_neon"`
}

type GPUInfo struct {
	Model    string  `json:"model"`
	Vendor   string  `json:"vendor"`
	VRAM_MB  int     `json:"vram_mb"`
	Backend  string  `json:"backend"`
	GPUCount int     `json:"gpu_count"`
}

type MemoryInfo struct {
	TotalGB     float64 `json:"total_gb"`
	FreeGB      float64 `json:"free_gb"`
	AvailableGB float64 `json:"available_gb"`
}

type HardwareProfile struct {
	CPU       CPUInfo    `json:"cpu"`
	GPU       GPUInfo    `json:"gpu"`
	Memory    MemoryInfo `json:"memory"`
	OS        string     `json:"os"`
	Arch      string     `json:"arch"`
	Tier      string     `json:"tier"`
	BackendID string     `json:"backend_id"`
}

func DetectHardware() (*HardwareProfile, error) {
	hw := &HardwareProfile{
		OS:   runtime.GOOS,
		Arch: runtime.GOARCH,
	}

	hw.CPU = detectCPU()
	hw.Memory = detectMemory()
	hw.GPU = detectGPU()
	hw.BackendID = resolveBackendID(hw)
	hw.Tier = classifyTier(hw)

	return hw, nil
}

func detectCPU() CPUInfo {
	info := CPUInfo{
		Brand:         "Unknown",
		Cores:         runtime.NumCPU(),
		PhysicalCores: runtime.NumCPU(),
		Architecture:  runtime.GOARCH,
	}

	if runtime.GOOS == "linux" {
		f, err := os.Open("/proc/cpuinfo")
		if err != nil {
			return info
		}
		defer f.Close()

		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			line := scanner.Text()
			if strings.HasPrefix(line, "model name") {
				parts := strings.SplitN(line, ":", 2)
				if len(parts) == 2 {
					info.Brand = strings.TrimSpace(parts[1])
				}
			}
		}

		flagsData, err := os.ReadFile("/proc/cpuinfo")
		if err == nil {
			content := string(flagsData)
			if strings.Contains(content, "avx512") {
				info.HasAVX512 = true
				info.HasAVX2 = true
			} else if strings.Contains(content, "avx2") {
				info.HasAVX2 = true
			}
		}
	}

	if runtime.GOARCH == "arm64" || runtime.GOARCH == "arm" {
		info.HasNEON = true
	}

	return info
}

func detectMemory() MemoryInfo {
	mem := MemoryInfo{}

	if runtime.GOOS == "linux" {
		f, err := os.Open("/proc/meminfo")
		if err != nil {
			return mem
		}
		defer f.Close()

		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			line := scanner.Text()
			if strings.HasPrefix(line, "MemTotal:") {
				mem.TotalGB = parseMemInfoKB(line)
			} else if strings.HasPrefix(line, "MemFree:") {
				mem.FreeGB = parseMemInfoKB(line)
			} else if strings.HasPrefix(line, "MemAvailable:") {
				mem.AvailableGB = parseMemInfoKB(line)
			}
		}
	}

	return mem
}

func parseMemInfoKB(line string) float64 {
	re := regexp.MustCompile(`(\d+)`)
	match := re.FindString(line)
	if match == "" {
		return 0
	}
	kb, err := strconv.ParseFloat(match, 64)
	if err != nil {
		return 0
	}
	return kb / (1024 * 1024)
}

func detectGPU() GPUInfo {
	gpu := GPUInfo{
		Model:    "None",
		Vendor:   "none",
		Backend:  "cpu",
		GPUCount: 0,
	}

	if tryNvidia(&gpu) {
		return gpu
	}
	if tryROCm(&gpu) {
		return gpu
	}
	if tryJetson(&gpu) {
		return gpu
	}

	return gpu
}

func tryNvidia(gpu *GPUInfo) bool {
	out, err := exec.Command("nvidia-smi",
		"--query-gpu=name,memory.total,count",
		"--format=csv,noheader,nounits").Output()
	if err != nil {
		return false
	}

	line := strings.TrimSpace(string(out))
	parts := strings.Split(line, ", ")
	if len(parts) >= 2 {
		gpu.Model = strings.TrimSpace(parts[0])
		gpu.Vendor = "nvidia"
		gpu.Backend = "cuda"
		if vram, err := strconv.Atoi(strings.TrimSpace(parts[1])); err == nil {
			gpu.VRAM_MB = vram
		}
		gpu.GPUCount = 1
		if len(parts) >= 3 {
			if c, err := strconv.Atoi(strings.TrimSpace(parts[2])); err == nil {
				gpu.GPUCount = c
			}
		}
		gpu.Backend = classifyNvidiaBackend(gpu.Model, gpu.VRAM_MB)
		return true
	}
	return false
}

func classifyNvidiaBackend(model string, vramMB int) string {
	m := strings.ToLower(model)
	switch {
	case strings.Contains(m, "h100"):
		return "cuda_h100"
	case strings.Contains(m, "a100"):
		return "cuda_a100"
	case strings.Contains(m, "4090"):
		return "cuda_4090"
	case strings.Contains(m, "4080"):
		return "cuda_4080"
	case strings.Contains(m, "3090"):
		return "cuda_3090"
	case strings.Contains(m, "3080"):
		return "cuda_3080"
	case strings.Contains(m, "3070"):
		return "cuda_3070"
	case strings.Contains(m, "3060"):
		return "cuda_3060"
	case strings.Contains(m, "2080"):
		return "cuda_2080"
	case strings.Contains(m, "p100") || strings.Contains(m, "tesla"):
		return "cuda_p100"
	default:
		return "cuda_default"
	}
}

func tryROCm(gpu *GPUInfo) bool {
	out, err := exec.Command("rocm-smi", "--showproductname", "--csv").Output()
	if err != nil {
		return false
	}

	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(lines) >= 2 {
		gpu.Model = strings.TrimSpace(lines[1])
		gpu.Vendor = "amd"
		gpu.Backend = "rocm_default"
		gpu.GPUCount = 1

		m := strings.ToLower(gpu.Model)
		switch {
		case strings.Contains(m, "mi300"):
			gpu.Backend = "rocm_mi300"
		case strings.Contains(m, "mi250"):
			gpu.Backend = "rocm_mi250"
		case strings.Contains(m, "7900 xtx"):
			gpu.Backend = "rocm_7900xtx"
		case strings.Contains(m, "7900 xt"):
			gpu.Backend = "rocm_7900xt"
		case strings.Contains(m, "7800 xt"):
			gpu.Backend = "rocm_7800xt"
		}
		return true
	}
	return false
}

func tryJetson(gpu *GPUInfo) bool {
	data, err := os.ReadFile("/sys/bus/platform/drivers/tegra-mc/memory-bandwidth")
	if err != nil {
		data, err = os.ReadFile("/proc/device-tree/model")
	}
	if err != nil {
		return false
	}

	model := strings.TrimSpace(string(data))
	if strings.Contains(strings.ToLower(model), "jetson") || strings.Contains(strings.ToLower(model), "xavier") {
		gpu.Model = model
		gpu.Vendor = "nvidia"
		gpu.Backend = "cuda_default"
		gpu.GPUCount = 1
		return true
	}
	return false
}

func resolveBackendID(hw *HardwareProfile) string {
	if hw.GPU.Backend != "cpu" {
		return hw.GPU.Backend
	}

	switch {
	case hw.CPU.HasAVX512:
		return "cpu_avx512"
	case hw.CPU.HasAVX2:
		return "cpu_avx2"
	case hw.CPU.HasNEON:
		return "cpu_neon"
	default:
		return "cpu_default"
	}
}

func classifyTier(hw *HardwareProfile) string {
	mem := hw.Memory.TotalGB
	vram := float64(hw.GPU.VRAM_MB) / 1024.0

	effectiveMem := mem
	if vram > 0 {
		effectiveMem = vram
	}

	switch {
	case effectiveMem >= 80:
		return "ultra_high"
	case effectiveMem >= 48:
		return "very_high"
	case effectiveMem >= 24:
		return "high"
	case effectiveMem >= 16:
		return "medium_high"
	case effectiveMem >= 8:
		return "medium"
	case effectiveMem >= 4:
		return "medium_low"
	case effectiveMem >= 2:
		return "low"
	default:
		return "ultra_low"
	}
}

func (hw *HardwareProfile) EffectiveMemoryGB() float64 {
	vram := float64(hw.GPU.VRAM_MB) / 1024.0
	if vram > 0 {
		return vram
	}
	return hw.Memory.TotalGB
}

func (hw *HardwareProfile) Summary() string {
	gpu := hw.GPU.Model
	if gpu == "None" || gpu == "" {
		gpu = "CPU only"
	}
	return fmt.Sprintf("%s | %dC | %.0fGB RAM | %s (%s) | tier=%s",
		hw.CPU.Brand, hw.CPU.Cores, hw.Memory.TotalGB,
		gpu, hw.BackendID, hw.Tier)
}
