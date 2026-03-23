// fetch.go implements the "fetch" tool — HTTP GET/HEAD helpers for the agent to retrieve page text
// (HTML stripped via regexp, JSON/plain passthrough) or inspect response headers, with optional length cap.
//
// Reused: pkg/tools/web.go L15-L47 — userAgent string; L21-L26 — scriptRe, styleRe, stripTagsRe, whitespaceRe for HTML stripping.

package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/Dawgomatic/Xagent/pkg/logger"
)

const fetchDefaultMaxLen = 5000

// FetchTool performs bounded HTTP fetches for LLM consumption (body text or headers only).
type FetchTool struct {
	httpClient *http.Client // SWE100821: injectable client for tests and timeout tuning
}

// NewFetchTool builds a FetchTool with a default net/http client (60s timeout, redirect cap).
func NewFetchTool() *FetchTool {
	// SWE100821: mirror web.go fetchHTTPClient — dedicated Client on struct per requirements
	return &FetchTool{
		httpClient: newFetchToolHTTPClient(),
	}
}

func fetchToolCheckRedirect(req *http.Request, via []*http.Request) error {
	if len(via) >= 5 {
		return fmt.Errorf("stopped after 5 redirects")
	}
	return nil
}

// newFetchToolHTTPClient returns a client configured like fetchHTTPClient in web.go.
func newFetchToolHTTPClient() *http.Client {
	return &http.Client{
		Timeout: 60 * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:        10,
			MaxIdleConnsPerHost: 5,
			IdleConnTimeout:     90 * time.Second,
			DisableCompression:  false,
			TLSHandshakeTimeout: 15 * time.Second,
		},
		CheckRedirect: fetchToolCheckRedirect,
	}
}

func (t *FetchTool) Name() string {
	return "fetch"
}

func (t *FetchTool) Description() string {
	return `Fetch a URL over HTTP(S). Action "get" returns response body as readable text (HTML tags stripped with simple regexp; JSON pretty-printed when applicable). Action "headers" returns response headers only (HEAD request).`
}

func (t *FetchTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"action": map[string]interface{}{
				"type":        "string",
				"description": `Required. "get" to fetch body text, "headers" for response headers only`,
				"enum":        []string{"get", "headers"},
			},
			"url": map[string]interface{}{
				"type":        "string",
				"description": "HTTP or HTTPS URL (required for get and headers)",
			},
			"max_length": map[string]interface{}{
				"type":        "integer",
				"description": "Max runes of body text for action get",
				"default":     fetchDefaultMaxLen,
				"minimum":     1.0,
			},
		},
		"required": []string{"action", "url"},
	}
}

func (t *FetchTool) Execute(ctx context.Context, args map[string]interface{}) *ToolResult {
	action, _ := args["action"].(string)
	action = strings.ToLower(strings.TrimSpace(action))
	urlStr, _ := args["url"].(string)
	urlStr = strings.TrimSpace(urlStr)

	if action == "" {
		return ErrorResult("action is required (get or headers)")
	}
	if urlStr == "" {
		return ErrorResult("url is required")
	}

	parsed, err := url.Parse(urlStr)
	if err != nil {
		return ErrorResult(fmt.Sprintf("invalid URL: %v", err))
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return ErrorResult("only http/https URLs are allowed")
	}
	if parsed.Host == "" {
		return ErrorResult("missing host in URL")
	}

	client := t.httpClient
	if client == nil {
		client = newFetchToolHTTPClient()
	}

	switch action {
	case "headers":
		return t.doHeaders(ctx, client, urlStr)
	case "get":
		maxLen := fetchArgMaxLength(args)
		return t.doGet(ctx, client, urlStr, maxLen)
	default:
		return ErrorResult(fmt.Sprintf("unknown action %q (use get or headers)", action))
	}
}

func fetchArgMaxLength(args map[string]interface{}) int {
	// SWE100821: LLM tool args often arrive as float64
	switch v := args["max_length"].(type) {
	case float64:
		if int(v) > 0 {
			return int(v)
		}
	case int:
		if v > 0 {
			return v
		}
	case int64:
		if v > 0 {
			return int(v)
		}
	case json.Number:
		if n, err := v.Int64(); err == nil && n > 0 {
			return int(n)
		}
	}
	return fetchDefaultMaxLen
}

func (t *FetchTool) doHeaders(ctx context.Context, client *http.Client, urlStr string) *ToolResult {
	req, err := http.NewRequestWithContext(ctx, http.MethodHead, urlStr, nil)
	if err != nil {
		return ErrorResult(fmt.Sprintf("failed to create request: %v", err))
	}
	req.Header.Set("User-Agent", userAgent)

	resp, err := client.Do(req)
	if err != nil {
		logger.ErrorF("fetch tool HEAD failed", map[string]interface{}{"url": urlStr, "err": err.Error()})
		return ErrorResult(fmt.Sprintf("request failed: %v", err))
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)

	out := formatResponseLine(resp) + "\n"
	for _, line := range sortedHeaderLines(resp.Header) {
		out += line + "\n"
	}
	return NewToolResult(strings.TrimSpace(out))
}

func (t *FetchTool) doGet(ctx context.Context, client *http.Client, urlStr string, maxLen int) *ToolResult {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, urlStr, nil)
	if err != nil {
		return ErrorResult(fmt.Sprintf("failed to create request: %v", err))
	}
	req.Header.Set("User-Agent", userAgent)

	resp, err := client.Do(req)
	if err != nil {
		logger.ErrorF("fetch tool GET failed", map[string]interface{}{"url": urlStr, "err": err.Error()})
		return ErrorResult(fmt.Sprintf("request failed: %v", err))
	}
	defer resp.Body.Close()

	// SWE100821: Cap response body at 2MB to prevent memory exhaustion from large/malicious responses
	const maxBodySize = 2 * 1024 * 1024
	limited := io.LimitReader(resp.Body, maxBodySize+1)
	body, err := io.ReadAll(limited)
	if err != nil {
		return ErrorResult(fmt.Sprintf("failed to read response: %v", err))
	}
	if len(body) > maxBodySize {
		body = body[:maxBodySize]
	}

	ct := resp.Header.Get("Content-Type")
	text, mode := fetchBodyToText(body, ct)

	truncated := false
	if maxLen > 0 && utf8.RuneCountInString(text) > maxLen {
		text = truncateRunes(text, maxLen)
		truncated = true
	}

	var b strings.Builder
	fmt.Fprintf(&b, "status: %d\n", resp.StatusCode)
	fmt.Fprintf(&b, "content-type: %s\n", ct)
	fmt.Fprintf(&b, "mode: %s\n", mode)
	fmt.Fprintf(&b, "truncated: %v\n", truncated)
	b.WriteString("---\n")
	b.WriteString(text)

	return NewToolResult(b.String())
}

// fetchBodyToText converts body bytes to a string for the LLM. Reused regexes: web.go scriptRe/styleRe/stripTagsRe.
func fetchBodyToText(body []byte, contentType string) (string, string) {
	ctLower := strings.ToLower(contentType)

	if strings.Contains(ctLower, "application/json") || strings.Contains(ctLower, "+json") {
		var data interface{}
		if err := json.Unmarshal(body, &data); err == nil {
			formatted, err := json.MarshalIndent(data, "", "  ")
			if err == nil {
				return string(formatted), "json"
			}
		}
		return string(body), "json-raw"
	}

	s := string(body)
	if strings.Contains(ctLower, "text/html") || looksLikeHTML(s) {
		return fetchStripHTML(s), "html-text"
	}

	if strings.HasPrefix(ctLower, "text/") || strings.Contains(ctLower, "charset=") {
		return strings.TrimSpace(s), "text"
	}

	return strings.TrimSpace(s), "raw"
}

func looksLikeHTML(s string) bool {
	t := strings.TrimSpace(s)
	if len(t) == 0 {
		return false
	}
	lower := strings.ToLower(t)
	return strings.HasPrefix(lower, "<!doctype") || strings.HasPrefix(lower, "<html")
}

// fetchStripHTML removes script/style blocks and tags; collapses whitespace. Reused: pkg/tools/web.go L21-L26.
func fetchStripHTML(html string) string {
	result := scriptRe.ReplaceAllLiteralString(html, "")
	result = styleRe.ReplaceAllLiteralString(result, "")
	result = stripTagsRe.ReplaceAllLiteralString(result, "")
	result = strings.TrimSpace(result)
	result = whitespaceRe.ReplaceAllLiteralString(result, " ")

	lines := strings.Split(result, "\n")
	var clean []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			clean = append(clean, line)
		}
	}
	return strings.Join(clean, "\n")
}

func formatResponseLine(resp *http.Response) string {
	proto := resp.Proto
	if proto == "" {
		proto = "HTTP/1.x"
	}
	return fmt.Sprintf("%s %s", proto, resp.Status)
}

func sortedHeaderLines(h http.Header) []string {
	keys := make([]string, 0, len(h))
	for k := range h {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var out []string
	for _, k := range keys {
		vals := append([]string(nil), h[k]...)
		sort.Strings(vals)
		for _, v := range vals {
			out = append(out, fmt.Sprintf("%s: %s", k, v))
		}
	}
	return out
}

func truncateRunes(s string, max int) string {
	if max <= 0 {
		return ""
	}
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return string(runes[:max])
}
