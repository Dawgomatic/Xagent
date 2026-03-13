// Vision tool: Local image understanding via Ollama multi-modal models.
// Supports llava, llama3.2-vision, and other vision-capable models.
// All processing is local — no cloud dependency.

package tools

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Dawgomatic/Xagent/pkg/logger"
)

// VisionTool analyzes images using local Ollama vision models.
type VisionTool struct {
	ollamaURL string
	model     string
	workspace string
}

// NewVisionTool creates a vision tool connected to local Ollama.
func NewVisionTool(workspace string) *VisionTool {
	return &VisionTool{
		ollamaURL: "http://localhost:11434",
		model:     "llava",
		workspace: workspace,
	}
}

func (t *VisionTool) Name() string {
	return "vision"
}

func (t *VisionTool) Description() string {
	return "Analyze an image using local AI vision. Describe what you see, read text, identify objects. Uses Ollama vision models locally — no cloud needed."
}

func (t *VisionTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"image_path": map[string]interface{}{
				"type":        "string",
				"description": "Absolute path to the image file to analyze",
			},
			"question": map[string]interface{}{
				"type":        "string",
				"description": "What to look for or ask about the image (default: 'Describe this image in detail')",
			},
			"model": map[string]interface{}{
				"type":        "string",
				"description": "Vision model to use (default: llava). Options: llava, llama3.2-vision",
			},
		},
		"required": []string{"image_path"},
	}
}

func (t *VisionTool) Execute(ctx context.Context, args map[string]interface{}) *ToolResult {
	imagePath, _ := args["image_path"].(string)
	question, _ := args["question"].(string)
	model, _ := args["model"].(string)

	if imagePath == "" {
		return &ToolResult{
			ForLLM:  "Error: image_path is required",
			IsError: true,
		}
	}

	if question == "" {
		question = "Describe this image in detail"
	}
	if model == "" {
		model = t.model
	}

	// Resolve relative paths
	if !filepath.IsAbs(imagePath) {
		imagePath = filepath.Join(t.workspace, imagePath)
	}

	// Read and encode the image
	imageData, err := os.ReadFile(imagePath)
	if err != nil {
		return &ToolResult{
			ForLLM:  fmt.Sprintf("Error reading image: %v", err),
			IsError: true,
			Err:     err,
		}
	}

	encoded := base64.StdEncoding.EncodeToString(imageData)

	// Call Ollama's /api/generate with images
	reqBody := map[string]interface{}{
		"model":  model,
		"prompt": question,
		"images": []string{encoded},
		"stream": false,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return &ToolResult{
			ForLLM:  fmt.Sprintf("Error encoding request: %v", err),
			IsError: true,
			Err:     err,
		}
	}

	client := &http.Client{Timeout: 120 * time.Second}
	req, err := http.NewRequestWithContext(ctx, "POST", t.ollamaURL+"/api/generate",
		strings.NewReader(string(jsonData)))
	if err != nil {
		return &ToolResult{
			ForLLM:  fmt.Sprintf("Error creating request: %v", err),
			IsError: true,
			Err:     err,
		}
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return &ToolResult{
			ForLLM:  fmt.Sprintf("Vision model not available (is Ollama running with %s?): %v", model, err),
			IsError: true,
			Err:     err,
		}
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return &ToolResult{
			ForLLM:  fmt.Sprintf("Error reading response: %v", err),
			IsError: true,
			Err:     err,
		}
	}

	if resp.StatusCode != http.StatusOK {
		return &ToolResult{
			ForLLM:  fmt.Sprintf("Vision API error (%d): %s", resp.StatusCode, string(body)),
			IsError: true,
		}
	}

	var result struct {
		Response string `json:"response"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return &ToolResult{
			ForLLM:  fmt.Sprintf("Error parsing response: %v", err),
			IsError: true,
			Err:     err,
		}
	}

	logger.InfoCF("vision", "Image analyzed", map[string]interface{}{
		"image":    filepath.Base(imagePath),
		"model":    model,
		"response": len(result.Response),
	})

	return &ToolResult{
		ForLLM:  result.Response,
		ForUser: fmt.Sprintf("🔍 Vision (%s): %s", model, result.Response),
	}
}
