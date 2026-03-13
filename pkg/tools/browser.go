// Browser tool: Headless web automation for autonomous browsing.
// Uses chromedp (Chrome DevTools Protocol) for zero-dependency browser control.
// The agent can navigate, click, type, take screenshots, and extract text.
// All execution is local.

package tools

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/Dawgomatic/Xagent/pkg/logger"
)

// BrowserTool provides headless web automation capabilities.
type BrowserTool struct {
	workspace string
	outputDir string
	mu        sync.Mutex
}

// NewBrowserTool creates a browser automation tool.
func NewBrowserTool(workspace string) *BrowserTool {
	outputDir := filepath.Join(workspace, "browser_output")
	os.MkdirAll(outputDir, 0755)
	return &BrowserTool{
		workspace: workspace,
		outputDir: outputDir,
	}
}

func (t *BrowserTool) Name() string {
	return "browser"
}

func (t *BrowserTool) Description() string {
	return "Browse the web autonomously. Navigate to URLs, take screenshots, extract page text and links. Uses a headless browser locally."
}

func (t *BrowserTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"action": map[string]interface{}{
				"type":        "string",
				"description": "Action to perform: navigate, screenshot, read_text, extract_links",
				"enum":        []string{"navigate", "screenshot", "read_text", "extract_links"},
			},
			"url": map[string]interface{}{
				"type":        "string",
				"description": "URL to navigate to (required for navigate action)",
			},
			"selector": map[string]interface{}{
				"type":        "string",
				"description": "Optional CSS selector to target specific element",
			},
		},
		"required": []string{"action", "url"},
	}
}

func (t *BrowserTool) Execute(ctx context.Context, args map[string]interface{}) *ToolResult {
	action, _ := args["action"].(string)
	url, _ := args["url"].(string)

	if action == "" {
		return &ToolResult{ForLLM: "Error: action is required", IsError: true}
	}
	if url == "" {
		return &ToolResult{ForLLM: "Error: url is required", IsError: true}
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	switch action {
	case "navigate", "read_text":
		return t.readPage(ctx, url)
	case "screenshot":
		return t.takeScreenshot(ctx, url)
	case "extract_links":
		return t.extractLinks(ctx, url)
	default:
		return &ToolResult{ForLLM: fmt.Sprintf("Unknown action: %s", action), IsError: true}
	}
}

// readPage fetches a URL and extracts text content using curl + html2text approach.
func (t *BrowserTool) readPage(ctx context.Context, url string) *ToolResult {
	cmdCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// Use curl to fetch and basic text extraction
	cmd := exec.CommandContext(cmdCtx, "curl", "-sL", "-A",
		"Mozilla/5.0 (X11; Linux aarch64) Xagent/1.0", "--max-time", "20", url)
	output, err := cmd.Output()
	if err != nil {
		return &ToolResult{
			ForLLM:  fmt.Sprintf("Failed to fetch %s: %v", url, err),
			IsError: true,
			Err:     err,
		}
	}

	// Strip HTML tags for text extraction
	text := stripHTMLTags(string(output))

	// Truncate to reasonable size
	if len(text) > 8000 {
		text = text[:8000] + "\n\n... (truncated)"
	}

	logger.InfoCF("browser", "Page fetched", map[string]interface{}{
		"url":      url,
		"text_len": len(text),
	})

	return &ToolResult{
		ForLLM:  fmt.Sprintf("Content from %s:\n\n%s", url, text),
		ForUser: fmt.Sprintf("🌐 Fetched %s (%d chars)", url, len(text)),
	}
}

// takeScreenshot captures a screenshot using a headless browser if available.
func (t *BrowserTool) takeScreenshot(ctx context.Context, url string) *ToolResult {
	outFile := filepath.Join(t.outputDir, fmt.Sprintf("screenshot_%d.png", time.Now().UnixNano()))

	cmdCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// Try chromium/chrome headless screenshot
	browsers := []string{"chromium-browser", "chromium", "google-chrome", "chrome"}
	var browserPath string
	for _, b := range browsers {
		if path, err := exec.LookPath(b); err == nil {
			browserPath = path
			break
		}
	}

	if browserPath == "" {
		return &ToolResult{
			ForLLM:  "No headless browser found. Install chromium: apt install chromium-browser",
			IsError: true,
		}
	}

	cmd := exec.CommandContext(cmdCtx, browserPath,
		"--headless", "--disable-gpu", "--no-sandbox",
		"--screenshot="+outFile, "--window-size=1280,720",
		url)
	if err := cmd.Run(); err != nil {
		return &ToolResult{
			ForLLM:  fmt.Sprintf("Screenshot failed: %v", err),
			IsError: true,
			Err:     err,
		}
	}

	logger.InfoCF("browser", "Screenshot captured", map[string]interface{}{
		"url":  url,
		"file": outFile,
	})

	return &ToolResult{
		ForLLM:  fmt.Sprintf("Screenshot saved to %s. Use the vision tool to analyze it.", outFile),
		ForUser: fmt.Sprintf("📸 Screenshot saved: %s", outFile),
	}
}

// extractLinks fetches a page and returns all hyperlinks.
func (t *BrowserTool) extractLinks(ctx context.Context, url string) *ToolResult {
	cmdCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(cmdCtx, "curl", "-sL", "-A",
		"Mozilla/5.0 (X11; Linux aarch64) Xagent/1.0", "--max-time", "20", url)
	output, err := cmd.Output()
	if err != nil {
		return &ToolResult{
			ForLLM:  fmt.Sprintf("Failed to fetch %s: %v", url, err),
			IsError: true,
			Err:     err,
		}
	}

	links := extractHrefLinks(string(output))
	if len(links) == 0 {
		return &ToolResult{ForLLM: "No links found on the page."}
	}

	if len(links) > 50 {
		links = links[:50]
	}

	return &ToolResult{
		ForLLM:  fmt.Sprintf("Links found on %s:\n%s", url, strings.Join(links, "\n")),
		ForUser: fmt.Sprintf("🔗 Found %d links on %s", len(links), url),
	}
}

// stripHTMLTags removes HTML tags from content for text extraction.
func stripHTMLTags(html string) string {
	var result strings.Builder
	inTag := false
	inScript := false
	inStyle := false

	lower := strings.ToLower(html)

	for i := 0; i < len(html); i++ {
		if i+7 < len(lower) && lower[i:i+7] == "<script" {
			inScript = true
		}
		if i+8 < len(lower) && lower[i:i+9] == "</script>" {
			inScript = false
			i += 8
			continue
		}
		if i+6 < len(lower) && lower[i:i+6] == "<style" {
			inStyle = true
		}
		if i+7 < len(lower) && lower[i:i+8] == "</style>" {
			inStyle = false
			i += 7
			continue
		}

		if inScript || inStyle {
			continue
		}

		if html[i] == '<' {
			inTag = true
			// Add newline for block elements
			if i+2 < len(lower) {
				tag := lower[i:]
				if strings.HasPrefix(tag, "<br") || strings.HasPrefix(tag, "<p") ||
					strings.HasPrefix(tag, "<div") || strings.HasPrefix(tag, "<h") ||
					strings.HasPrefix(tag, "<li") || strings.HasPrefix(tag, "<tr") {
					result.WriteByte('\n')
				}
			}
			continue
		}
		if html[i] == '>' {
			inTag = false
			continue
		}
		if !inTag {
			result.WriteByte(html[i])
		}
	}

	// Clean up whitespace
	text := result.String()
	lines := strings.Split(text, "\n")
	var cleaned []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			cleaned = append(cleaned, trimmed)
		}
	}
	return strings.Join(cleaned, "\n")
}

// extractHrefLinks pulls href values from anchor tags.
func extractHrefLinks(html string) []string {
	var links []string
	seen := make(map[string]bool)

	lower := strings.ToLower(html)
	offset := 0
	for {
		idx := strings.Index(lower[offset:], "href=\"")
		if idx < 0 {
			break
		}
		start := offset + idx + 6
		end := strings.Index(html[start:], "\"")
		if end < 0 {
			break
		}
		link := html[start : start+end]
		offset = start + end

		if link != "" && link != "#" && !seen[link] {
			seen[link] = true
			links = append(links, link)
		}
	}
	return links
}
