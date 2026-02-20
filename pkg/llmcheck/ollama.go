package llmcheck

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

type OllamaClient struct {
	BaseURL    string
	HTTPClient *http.Client
}

func NewOllamaClient() *OllamaClient {
	base := os.Getenv("OLLAMA_HOST")
	if base == "" {
		base = "http://localhost:11434"
	}
	if !strings.HasPrefix(base, "http://") && !strings.HasPrefix(base, "https://") {
		base = "http://" + base
	}
	base = strings.TrimRight(base, "/")

	return &OllamaClient{
		BaseURL: base,
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

type OllamaVersion struct {
	Available bool   `json:"available"`
	Version   string `json:"version"`
	Error     string `json:"error,omitempty"`
}

func (c *OllamaClient) CheckAvailability() OllamaVersion {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", c.BaseURL+"/api/version", nil)
	if err != nil {
		return OllamaVersion{Error: err.Error()}
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return OllamaVersion{Error: fmt.Sprintf("Ollama not running at %s: %s", c.BaseURL, err.Error())}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return OllamaVersion{Error: fmt.Sprintf("HTTP %d from Ollama", resp.StatusCode)}
	}

	var data struct {
		Version string `json:"version"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return OllamaVersion{Available: true, Version: "unknown"}
	}

	return OllamaVersion{Available: true, Version: data.Version}
}

type OllamaModel struct {
	Name       string    `json:"name"`
	Model      string    `json:"model"`
	ModifiedAt time.Time `json:"modified_at"`
	Size       int64     `json:"size"`
	Family     string    `json:"family"`
	ParamSize  string    `json:"parameter_size"`
	QuantLevel string    `json:"quantization_level"`
}

type ollamaTagsResponse struct {
	Models []struct {
		Name       string    `json:"name"`
		Model      string    `json:"model"`
		ModifiedAt time.Time `json:"modified_at"`
		Size       int64     `json:"size"`
		Details    struct {
			Family           string `json:"family"`
			ParameterSize    string `json:"parameter_size"`
			QuantizationLevel string `json:"quantization_level"`
		} `json:"details"`
	} `json:"models"`
}

func (c *OllamaClient) ListModels() ([]OllamaModel, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", c.BaseURL+"/api/tags", nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ollama not available: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d from Ollama", resp.StatusCode)
	}

	var data ollamaTagsResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}

	models := make([]OllamaModel, 0, len(data.Models))
	for _, m := range data.Models {
		models = append(models, OllamaModel{
			Name:       m.Name,
			Model:      m.Model,
			ModifiedAt: m.ModifiedAt,
			Size:       m.Size,
			Family:     m.Details.Family,
			ParamSize:  m.Details.ParameterSize,
			QuantLevel: m.Details.QuantizationLevel,
		})
	}
	return models, nil
}

type RunningModel struct {
	Name      string    `json:"name"`
	Model     string    `json:"model"`
	Size      int64     `json:"size"`
	ExpiresAt time.Time `json:"expires_at"`
}

func (c *OllamaClient) ListRunning() ([]RunningModel, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", c.BaseURL+"/api/ps", nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ollama not available: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, nil
	}

	var data struct {
		Models []RunningModel `json:"models"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, nil
	}
	return data.Models, nil
}

type PullProgress struct {
	Status    string `json:"status"`
	Digest    string `json:"digest"`
	Total     int64  `json:"total"`
	Completed int64  `json:"completed"`
}

func (c *OllamaClient) PullModel(name string, onProgress func(PullProgress)) error {
	body := fmt.Sprintf(`{"name":"%s","stream":true}`, name)

	req, err := http.NewRequest("POST", c.BaseURL+"/api/pull", strings.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 0}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("pull failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("pull HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	decoder := json.NewDecoder(resp.Body)
	for {
		var p PullProgress
		if err := decoder.Decode(&p); err != nil {
			if err == io.EOF {
				break
			}
			return err
		}
		if onProgress != nil {
			onProgress(p)
		}
		if p.Status == "success" {
			break
		}
	}
	return nil
}

func (c *OllamaClient) DeleteModel(name string) error {
	body := fmt.Sprintf(`{"name":"%s"}`, name)

	req, err := http.NewRequest("DELETE", c.BaseURL+"/api/delete", strings.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("delete failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("delete HTTP %d", resp.StatusCode)
	}
	return nil
}

type BenchmarkResult struct {
	Model            string  `json:"model"`
	TotalDurationMs  float64 `json:"total_duration_ms"`
	LoadDurationMs   float64 `json:"load_duration_ms"`
	PromptTokens     int     `json:"prompt_tokens"`
	ResponseTokens   int     `json:"response_tokens"`
	TokensPerSecond  float64 `json:"tokens_per_second"`
}

func (c *OllamaClient) Benchmark(modelName string) (*BenchmarkResult, error) {
	body := fmt.Sprintf(`{"model":"%s","prompt":"Explain what a neural network is in exactly 3 sentences.","stream":false}`, modelName)

	req, err := http.NewRequest("POST", c.BaseURL+"/api/generate", strings.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("benchmark failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("benchmark HTTP %d", resp.StatusCode)
	}

	var data struct {
		TotalDuration    int64 `json:"total_duration"`
		LoadDuration     int64 `json:"load_duration"`
		PromptEvalCount  int   `json:"prompt_eval_count"`
		EvalCount        int   `json:"eval_count"`
		EvalDuration     int64 `json:"eval_duration"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}

	tps := 0.0
	if data.EvalDuration > 0 {
		tps = float64(data.EvalCount) / (float64(data.EvalDuration) / 1e9)
	}

	return &BenchmarkResult{
		Model:           modelName,
		TotalDurationMs: float64(data.TotalDuration) / 1e6,
		LoadDurationMs:  float64(data.LoadDuration) / 1e6,
		PromptTokens:    data.PromptEvalCount,
		ResponseTokens:  data.EvalCount,
		TokensPerSecond: tps,
	}, nil
}
