package llmcheck

import "strings"

type ModelVariant struct {
	Name       string  `json:"name"`
	Family     string  `json:"family"`
	ParamsB    float64 `json:"params_b"`
	Quant      string  `json:"quant"`
	SizeGB     float64 `json:"size_gb"`
	ContextLen int     `json:"context_len"`
	Category   string  `json:"category"`
	OllamaTag  string  `json:"ollama_tag"`
}

func EstimateSizeGB(paramsB float64, quant string) float64 {
	q := normalizeQuant(quant)
	switch {
	case q == "FP16" || q == "F16":
		return paramsB * 2.0
	case startsWith(q, "Q8"):
		return paramsB * 1.0
	case startsWith(q, "Q6"):
		return paramsB * 0.75
	case startsWith(q, "Q5"):
		return paramsB * 0.6
	case startsWith(q, "Q4") || startsWith(q, "IQ4"):
		return paramsB * 0.5
	case startsWith(q, "Q3") || startsWith(q, "IQ3"):
		return paramsB * 0.4
	case startsWith(q, "Q2") || startsWith(q, "IQ2"):
		return paramsB * 0.3
	default:
		return paramsB * 0.5
	}
}

func (m *ModelVariant) EffectiveSize() float64 {
	if m.SizeGB > 0 {
		return m.SizeGB
	}
	return EstimateSizeGB(m.ParamsB, m.Quant)
}

var CuratedCatalog = []ModelVariant{
	// General / Chat
	{Name: "llama3.2:1b", Family: "llama3.2", ParamsB: 1, Quant: "Q4_K_M", SizeGB: 0.7, ContextLen: 131072, Category: "general", OllamaTag: "llama3.2:1b"},
	{Name: "llama3.2:3b", Family: "llama3.2", ParamsB: 3, Quant: "Q4_K_M", SizeGB: 2.0, ContextLen: 131072, Category: "general", OllamaTag: "llama3.2:3b"},
	{Name: "llama3.1:8b", Family: "llama3.1", ParamsB: 8, Quant: "Q4_K_M", SizeGB: 4.7, ContextLen: 131072, Category: "general", OllamaTag: "llama3.1:8b"},
	{Name: "llama3.3:70b", Family: "llama3.3", ParamsB: 70, Quant: "Q4_K_M", SizeGB: 43, ContextLen: 131072, Category: "general", OllamaTag: "llama3.3:70b-instruct-q4_K_M"},

	// Qwen general
	{Name: "qwen2.5:0.5b", Family: "qwen2.5", ParamsB: 0.5, Quant: "Q4_K_M", SizeGB: 0.4, ContextLen: 32768, Category: "general", OllamaTag: "qwen2.5:0.5b"},
	{Name: "qwen2.5:1.5b", Family: "qwen2.5", ParamsB: 1.5, Quant: "Q4_K_M", SizeGB: 1.0, ContextLen: 32768, Category: "general", OllamaTag: "qwen2.5:1.5b"},
	{Name: "qwen2.5:3b", Family: "qwen2.5", ParamsB: 3, Quant: "Q4_K_M", SizeGB: 1.9, ContextLen: 32768, Category: "general", OllamaTag: "qwen2.5:3b"},
	{Name: "qwen2.5:7b", Family: "qwen2.5", ParamsB: 7, Quant: "Q4_K_M", SizeGB: 4.7, ContextLen: 131072, Category: "general", OllamaTag: "qwen2.5:7b"},
	{Name: "qwen2.5:14b", Family: "qwen2.5", ParamsB: 14, Quant: "Q4_K_M", SizeGB: 9.0, ContextLen: 131072, Category: "general", OllamaTag: "qwen2.5:14b"},
	{Name: "qwen2.5:32b", Family: "qwen2.5", ParamsB: 32, Quant: "Q4_K_M", SizeGB: 20, ContextLen: 131072, Category: "general", OllamaTag: "qwen2.5:32b"},

	// Reasoning
	{Name: "deepseek-r1:1.5b", Family: "deepseek-r1", ParamsB: 1.5, Quant: "Q4_K_M", SizeGB: 1.1, ContextLen: 65536, Category: "reasoning", OllamaTag: "deepseek-r1:1.5b"},
	{Name: "deepseek-r1:7b", Family: "deepseek-r1", ParamsB: 7, Quant: "Q4_K_M", SizeGB: 4.7, ContextLen: 65536, Category: "reasoning", OllamaTag: "deepseek-r1:7b"},
	{Name: "deepseek-r1:14b", Family: "deepseek-r1", ParamsB: 14, Quant: "Q4_K_M", SizeGB: 9.0, ContextLen: 65536, Category: "reasoning", OllamaTag: "deepseek-r1:14b"},
	{Name: "deepseek-r1:32b", Family: "deepseek-r1", ParamsB: 32, Quant: "Q4_K_M", SizeGB: 20, ContextLen: 65536, Category: "reasoning", OllamaTag: "deepseek-r1:32b"},
	{Name: "deepseek-r1:70b", Family: "deepseek-r1", ParamsB: 70, Quant: "Q4_K_M", SizeGB: 43, ContextLen: 65536, Category: "reasoning", OllamaTag: "deepseek-r1:70b"},
	{Name: "phi4:14b", Family: "phi-4", ParamsB: 14, Quant: "Q4_K_M", SizeGB: 9.1, ContextLen: 16384, Category: "reasoning", OllamaTag: "phi4:14b"},

	// Coding
	{Name: "qwen2.5-coder:1.5b", Family: "qwen2.5-coder", ParamsB: 1.5, Quant: "Q4_K_M", SizeGB: 1.0, ContextLen: 32768, Category: "coding", OllamaTag: "qwen2.5-coder:1.5b"},
	{Name: "qwen2.5-coder:3b", Family: "qwen2.5-coder", ParamsB: 3, Quant: "Q4_K_M", SizeGB: 1.9, ContextLen: 32768, Category: "coding", OllamaTag: "qwen2.5-coder:3b"},
	{Name: "qwen2.5-coder:7b", Family: "qwen2.5-coder", ParamsB: 7, Quant: "Q4_K_M", SizeGB: 4.7, ContextLen: 131072, Category: "coding", OllamaTag: "qwen2.5-coder:7b"},
	{Name: "qwen2.5-coder:14b", Family: "qwen2.5-coder", ParamsB: 14, Quant: "Q4_K_M", SizeGB: 9.0, ContextLen: 131072, Category: "coding", OllamaTag: "qwen2.5-coder:14b"},
	{Name: "qwen2.5-coder:32b", Family: "qwen2.5-coder", ParamsB: 32, Quant: "Q4_K_M", SizeGB: 20, ContextLen: 131072, Category: "coding", OllamaTag: "qwen2.5-coder:32b"},
	{Name: "deepseek-coder-v2:16b", Family: "deepseek-coder-v2", ParamsB: 16, Quant: "Q4_K_M", SizeGB: 8.9, ContextLen: 131072, Category: "coding", OllamaTag: "deepseek-coder-v2:16b"},
	{Name: "starcoder2:3b", Family: "starcoder2", ParamsB: 3, Quant: "Q4_K_M", SizeGB: 1.7, ContextLen: 16384, Category: "coding", OllamaTag: "starcoder2:3b"},
	{Name: "starcoder2:7b", Family: "starcoder2", ParamsB: 7, Quant: "Q4_K_M", SizeGB: 4.0, ContextLen: 16384, Category: "coding", OllamaTag: "starcoder2:7b"},

	// Phi (lightweight)
	{Name: "phi3:3.8b", Family: "phi-3", ParamsB: 3.8, Quant: "Q4_K_M", SizeGB: 2.3, ContextLen: 4096, Category: "general", OllamaTag: "phi3:3.8b"},
	{Name: "phi3.5:3.8b", Family: "phi-3.5", ParamsB: 3.8, Quant: "Q4_K_M", SizeGB: 2.2, ContextLen: 131072, Category: "general", OllamaTag: "phi3.5:3.8b"},

	// Gemma
	{Name: "gemma2:2b", Family: "gemma2", ParamsB: 2, Quant: "Q4_K_M", SizeGB: 1.6, ContextLen: 8192, Category: "general", OllamaTag: "gemma2:2b"},
	{Name: "gemma2:9b", Family: "gemma2", ParamsB: 9, Quant: "Q4_K_M", SizeGB: 5.4, ContextLen: 8192, Category: "general", OllamaTag: "gemma2:9b"},
	{Name: "gemma2:27b", Family: "gemma2", ParamsB: 27, Quant: "Q4_K_M", SizeGB: 16, ContextLen: 8192, Category: "general", OllamaTag: "gemma2:27b"},

	// Mistral
	{Name: "mistral:7b", Family: "mistral", ParamsB: 7, Quant: "Q4_K_M", SizeGB: 4.1, ContextLen: 32768, Category: "general", OllamaTag: "mistral:7b"},
	{Name: "mixtral:8x7b", Family: "mixtral", ParamsB: 47, Quant: "Q4_K_M", SizeGB: 26, ContextLen: 32768, Category: "general", OllamaTag: "mixtral:8x7b"},

	// Vision
	{Name: "llava:7b", Family: "llava", ParamsB: 7, Quant: "Q4_K_M", SizeGB: 4.7, ContextLen: 4096, Category: "vision", OllamaTag: "llava:7b"},
	{Name: "llava:13b", Family: "llava", ParamsB: 13, Quant: "Q4_K_M", SizeGB: 8.0, ContextLen: 4096, Category: "vision", OllamaTag: "llava:13b"},
	{Name: "moondream:1.8b", Family: "moondream", ParamsB: 1.8, Quant: "Q4_K_M", SizeGB: 1.0, ContextLen: 2048, Category: "vision", OllamaTag: "moondream:1.8b"},

	// Embeddings
	{Name: "nomic-embed-text", Family: "nomic-embed-text", ParamsB: 0.14, Quant: "F16", SizeGB: 0.27, ContextLen: 8192, Category: "embeddings", OllamaTag: "nomic-embed-text"},
	{Name: "mxbai-embed-large", Family: "mxbai-embed-large", ParamsB: 0.33, Quant: "F16", SizeGB: 0.67, ContextLen: 512, Category: "embeddings", OllamaTag: "mxbai-embed-large"},
	{Name: "all-minilm", Family: "all-minilm", ParamsB: 0.023, Quant: "F16", SizeGB: 0.045, ContextLen: 256, Category: "embeddings", OllamaTag: "all-minilm"},

	// Tiny / ultra-low
	{Name: "tinyllama:1.1b", Family: "tinyllama", ParamsB: 1.1, Quant: "Q4_K_M", SizeGB: 0.6, ContextLen: 2048, Category: "general", OllamaTag: "tinyllama:1.1b"},
	{Name: "smollm:135m", Family: "smollm", ParamsB: 0.135, Quant: "Q4_K_M", SizeGB: 0.1, ContextLen: 2048, Category: "general", OllamaTag: "smollm:135m"},
}

func normalizeQuant(q string) string {
	return strings.ToUpper(strings.TrimSpace(q))
}

func startsWith(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}
