package llmcheck

import (
	"math"
	"strings"
)

type CategoryWeights struct {
	Q float64 // Quality
	S float64 // Speed
	F float64 // Fit
	C float64 // Context
}

var WeightPresets = map[string]CategoryWeights{
	"general":    {Q: 0.40, S: 0.35, F: 0.15, C: 0.10},
	"coding":     {Q: 0.55, S: 0.20, F: 0.15, C: 0.10},
	"reasoning":  {Q: 0.60, S: 0.15, F: 0.10, C: 0.15},
	"chat":       {Q: 0.40, S: 0.40, F: 0.15, C: 0.05},
	"creative":   {Q: 0.50, S: 0.25, F: 0.15, C: 0.10},
	"embeddings": {Q: 0.30, S: 0.50, F: 0.15, C: 0.05},
	"vision":     {Q: 0.50, S: 0.25, F: 0.15, C: 0.10},
	"fast":       {Q: 0.25, S: 0.55, F: 0.15, C: 0.05},
	"quality":    {Q: 0.65, S: 0.10, F: 0.15, C: 0.10},
}

var familyQuality = map[string]float64{
	"qwen2.5": 95, "qwen2": 90, "llama3.3": 95, "llama3.2": 92,
	"llama3.1": 90, "llama3": 88, "deepseek-v3": 96, "deepseek-v2.5": 94,
	"deepseek-coder-v2": 92, "deepseek-r1": 96, "gemma2": 90, "gemma": 82,
	"phi-4": 92, "phi-3.5": 88, "phi-3": 85, "phi-2": 75,
	"mistral-large": 94, "mistral": 85, "mixtral": 88,
	"command-r": 90, "command-r-plus": 93,
	"qwen2.5-coder": 96, "codellama": 82, "starcoder2": 85,
	"deepseek-coder": 88, "codegemma": 80, "granite-code": 78,
	"yi": 85, "yi-coder": 88, "openchat": 78, "neural-chat": 75,
	"zephyr": 80, "openhermes": 82, "nous-hermes": 82,
	"dolphin": 80, "orca": 78,
	"llava": 82, "llava-llama3": 85, "llava-phi3": 80,
	"bakllava": 78, "moondream": 75,
	"nomic-embed-text": 85, "mxbai-embed-large": 88,
	"all-minilm": 80, "snowflake-arctic-embed": 85,
	"solar": 82, "falcon": 75, "vicuna": 72, "wizardlm": 78,
	"aya": 85, "smollm": 70, "tinyllama": 65,
}

var quantPenalties = map[string]float64{
	"FP16": 0, "F16": 0, "Q8_0": 2, "Q6_K": 4,
	"Q5_K_M": 6, "Q5_K_S": 7, "Q5_0": 8,
	"Q4_K_M": 10, "Q4_K_S": 11, "Q4_0": 12,
	"Q3_K_M": 16, "Q3_K_S": 18, "Q3_K_L": 15,
	"IQ4_XS": 11, "IQ4_NL": 10,
	"IQ3_XXS": 20, "IQ3_XS": 18, "IQ3_S": 17,
	"IQ2_XS": 25, "IQ2_XXS": 28, "Q2_K": 22, "Q2_K_S": 24,
}

var taskBonuses = map[string]map[string]float64{
	"coding": {
		"qwen2.5-coder": 15, "deepseek-coder": 12, "deepseek-coder-v2": 15,
		"codellama": 10, "starcoder2": 12, "codegemma": 8, "yi-coder": 10, "granite-code": 8,
	},
	"reasoning": {
		"deepseek-r1": 15, "qwen2.5": 10, "llama3.3": 10, "phi-4": 12,
		"command-r-plus": 10, "mistral-large": 10,
	},
	"chat": {
		"llama3.2": 10, "mistral": 8, "gemma2": 8, "openchat": 10,
		"neural-chat": 8, "dolphin": 8,
	},
	"vision": {
		"llava": 15, "llava-llama3": 18, "llava-phi3": 15, "bakllava": 12, "moondream": 10,
	},
	"embeddings": {
		"nomic-embed-text": 15, "mxbai-embed-large": 18, "all-minilm": 12,
		"snowflake-arctic-embed": 15,
	},
}

var backendSpeed = map[string]float64{
	"cuda_h100": 120, "cuda_a100": 90, "cuda_4090": 70, "cuda_4080": 55,
	"cuda_3090": 50, "cuda_3080": 40, "cuda_3070": 32, "cuda_3060": 25,
	"cuda_2080": 28, "cuda_p100": 30, "cuda_default": 30,
	"rocm_mi300": 100, "rocm_mi250": 70, "rocm_7900xtx": 55,
	"rocm_7900xt": 45, "rocm_7800xt": 38, "rocm_6900xt": 35, "rocm_default": 30,
	"metal_m4_ultra": 75, "metal_m4_max": 60, "metal_m4_pro": 45, "metal_m4": 35,
	"metal_m3_ultra": 65, "metal_m3_max": 50, "metal_m3_pro": 40, "metal_m3": 30,
	"metal_m2_ultra": 55, "metal_m2_max": 45, "metal_m2_pro": 35, "metal_m2": 28,
	"metal_m1_ultra": 45, "metal_m1_max": 38, "metal_m1_pro": 30, "metal_m1": 22,
	"metal_default": 30,
	"intel_arc_a770": 30, "intel_arc_a750": 25, "intel_arc_default": 20,
	"cpu_avx512_amx": 12, "cpu_avx512": 8, "cpu_avx2": 5,
	"cpu_neon": 4, "cpu_avx": 3, "cpu_default": 2,
}

var quantSpeedMult = map[string]float64{
	"FP16": 0.5, "F16": 0.5, "Q8_0": 0.7, "Q6_K": 0.85,
	"Q5_K_M": 0.92, "Q5_K_S": 0.92, "Q5_0": 0.92,
	"Q4_K_M": 1.0, "Q4_K_S": 1.0, "Q4_0": 1.05,
	"Q3_K_M": 1.15, "Q3_K_S": 1.15, "Q3_K_L": 1.1,
	"IQ4_XS": 1.02, "IQ4_NL": 1.0,
	"IQ3_XXS": 1.2, "IQ3_XS": 1.18, "IQ3_S": 1.15,
	"IQ2_XS": 1.25, "IQ2_XXS": 1.28, "Q2_K": 1.22, "Q2_K_S": 1.25,
}

type ScoreBreakdown struct {
	Quality    float64 `json:"quality"`
	Speed      float64 `json:"speed"`
	Fit        float64 `json:"fit"`
	Context    float64 `json:"context"`
	FinalScore float64 `json:"final_score"`
	EstTPS     float64 `json:"est_tps"`
}

func ScoreModel(model *ModelVariant, hw *HardwareProfile, useCase string, targetCtx int) ScoreBreakdown {
	if targetCtx <= 0 {
		targetCtx = 8192
	}

	weights, ok := WeightPresets[useCase]
	if !ok {
		weights = WeightPresets["general"]
	}

	Q := qualityScore(model, useCase)
	estTPS := estimateTPS(model, hw)
	S := speedScore(estTPS, 20)
	F := fitScore(model, hw)
	C := contextScore(model, targetCtx)

	final := Q*weights.Q + S*weights.S + F*weights.F + C*weights.C

	return ScoreBreakdown{
		Quality:    Q,
		Speed:      S,
		Fit:        F,
		Context:    C,
		FinalScore: final,
		EstTPS:     estTPS,
	}
}

func qualityScore(m *ModelVariant, useCase string) float64 {
	base, ok := familyQuality[m.Family]
	if !ok {
		base = 70
	}

	// Parameter bonus
	switch {
	case m.ParamsB >= 65:
		base += 15
	case m.ParamsB >= 30:
		base += 12
	case m.ParamsB >= 13:
		base += 8
	case m.ParamsB >= 6:
		base += 5
	case m.ParamsB >= 3:
		base += 2
	}

	// Quantization penalty
	q := normalizeQuant(m.Quant)
	if penalty, ok := quantPenalties[q]; ok {
		base -= penalty
	}

	// Task bonus
	if bonuses, ok := taskBonuses[useCase]; ok {
		if bonus, ok := bonuses[m.Family]; ok {
			base += bonus
		}
	}

	return clamp(base, 0, 100)
}

func estimateTPS(m *ModelVariant, hw *HardwareProfile) float64 {
	baseTPS, ok := backendSpeed[hw.BackendID]
	if !ok {
		baseTPS = backendSpeed["cpu_default"]
	}

	q := normalizeQuant(m.Quant)
	qMult := 1.0
	if mult, ok := quantSpeedMult[q]; ok {
		qMult = mult
	}

	// Scale by model size relative to 7B baseline with diminishing returns
	sizeRatio := m.ParamsB / 7.0
	if sizeRatio < 0.1 {
		sizeRatio = 0.1
	}
	sizeMult := 1.0 / math.Pow(sizeRatio, 0.7)

	return baseTPS * qMult * sizeMult
}

func speedScore(estTPS, targetTPS float64) float64 {
	if targetTPS <= 0 {
		targetTPS = 20
	}

	ratio := estTPS / targetTPS
	switch {
	case ratio >= 3.0:
		return 100
	case ratio >= 2.0:
		return 90 + (ratio-2.0)*10
	case ratio >= 1.0:
		return 70 + (ratio-1.0)*20
	case ratio >= 0.5:
		return 30 + (ratio-0.5)*80
	default:
		return ratio * 60
	}
}

func fitScore(m *ModelVariant, hw *HardwareProfile) float64 {
	effectiveMem := hw.EffectiveMemoryGB()
	if effectiveMem <= 0 {
		return 50
	}

	modelSize := m.EffectiveSize()
	usage := modelSize / effectiveMem

	switch {
	case usage <= 0.70:
		return 100
	case usage <= 0.85:
		return 90 + (0.85-usage)/0.15*10
	case usage <= 1.0:
		return 70 + (1.0-usage)/0.15*20
	case usage <= 1.2:
		return 30 + (1.2-usage)/0.2*40
	default:
		return math.Max(0, 30-(usage-1.2)*50)
	}
}

func contextScore(m *ModelVariant, targetCtx int) float64 {
	if targetCtx <= 0 {
		return 85
	}

	ratio := float64(m.ContextLen) / float64(targetCtx)
	switch {
	case ratio >= 2.0:
		return 100
	case ratio >= 1.0:
		return 85 + (ratio-1.0)*15
	case ratio >= 0.5:
		return 50 + (ratio-0.5)*70
	default:
		return ratio * 100
	}
}

func clamp(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// FitsInMemory returns true if the model can run on this hardware
func FitsInMemory(m *ModelVariant, hw *HardwareProfile) bool {
	return m.EffectiveSize() <= hw.EffectiveMemoryGB()*1.05
}

// MatchesCategory returns true if the model family has a task bonus for the given use case,
// or if it's a general model and the category is general.
func MatchesCategory(m *ModelVariant, category string) bool {
	if category == "" || category == "general" {
		return true
	}
	if strings.EqualFold(m.Category, category) {
		return true
	}
	if bonuses, ok := taskBonuses[category]; ok {
		if _, ok := bonuses[m.Family]; ok {
			return true
		}
	}
	return false
}
