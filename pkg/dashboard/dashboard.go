// SWE100821: Interactive cognitive dashboard — embedded web UI for full agent introspection.
// Serves a dark-themed tabbed SPA and JSON APIs for memory, vault, epochs, provenance,
// skills, config, cron, metrics, peers, chat, and vault knowledge graph.
// No external file dependencies; HTML lives in dashboard_html.go as a const string.

package dashboard

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/Dawgomatic/Xagent/pkg/health"
)

// SWE100821: Regex for extracting [[wikilinks]] from vault notes (handles [[name]] and [[name|alias]])
var wikilinkRe = regexp.MustCompile(`\[\[([^\]\|]+)(?:\|[^\]]+)?\]\]`)

// Dashboard provides an embedded HTTP dashboard for agent introspection.
// SWE100821: Expanded with chat, vault graph, and full subsystem browsing.
type Dashboard struct {
	workspace   string
	vaultPath   string
	configPath  string
	metrics     *health.Metrics
	startTime   time.Time
	chatHandler func(ctx context.Context, message, sessionKey string) (string, error)
	chatMu      sync.Mutex
	chatReqs    map[string]*chatReq
}

// chatReq tracks an in-flight chat request for async polling.
type chatReq struct {
	Response string    `json:"response,omitempty"`
	Error    string    `json:"error,omitempty"`
	Done     bool      `json:"done"`
	Created  time.Time `json:"-"`
}

// NewDashboard creates a dashboard rooted at the given workspace.
func NewDashboard(workspace string) *Dashboard {
	return &Dashboard{
		workspace: workspace,
		startTime: time.Now(),
		chatReqs:  make(map[string]*chatReq),
	}
}

// SetVaultPath configures the Obsidian vault path for browsing.
func (d *Dashboard) SetVaultPath(path string) { d.vaultPath = path }

// SetConfigPath sets the config.json path for sanitized viewing.
func (d *Dashboard) SetConfigPath(path string) { d.configPath = path }

// SetMetrics injects the health metrics pointer for live stats.
func (d *Dashboard) SetMetrics(m *health.Metrics) { d.metrics = m }

// SetChatHandler sets the function called to process chat messages.
// SWE100821: Signature matches agentLoop.ProcessDirect(ctx, content, sessionKey).
func (d *Dashboard) SetChatHandler(fn func(ctx context.Context, message, sessionKey string) (string, error)) {
	d.chatHandler = fn
}

// SetupRoutes registers all dashboard routes on the given mux.
// SWE100821: Includes original, memory, vault, detail, config, chat, and graph APIs.
func (d *Dashboard) SetupRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/dashboard", d.handleDashboardPage)

	// Original APIs
	mux.HandleFunc("/api/state", d.handleState)
	mux.HandleFunc("/api/epochs", d.handleEpochs)
	mux.HandleFunc("/api/provenance", d.handleProvenance)
	mux.HandleFunc("/api/skills", d.handleSkills)
	mux.HandleFunc("/api/peers", d.handlePeers)

	// Memory APIs
	mux.HandleFunc("/api/memory/longterm", d.handleMemoryLongterm)
	mux.HandleFunc("/api/memory/daily", d.handleMemoryDaily)
	mux.HandleFunc("/api/memory/consolidations", d.handleMemoryConsolidations)

	// Vault APIs
	mux.HandleFunc("/api/vault/tree", d.handleVaultTree)
	mux.HandleFunc("/api/vault/note", d.handleVaultNote)
	mux.HandleFunc("/api/vault/graph", d.handleVaultGraph)

	// Detail APIs
	mux.HandleFunc("/api/epoch/detail", d.handleEpochDetail)
	mux.HandleFunc("/api/provenance/detail", d.handleProvenanceDetail)

	// System APIs
	mux.HandleFunc("/api/config", d.handleConfig)
	mux.HandleFunc("/api/cron/jobs", d.handleCronJobs)
	mux.HandleFunc("/api/metrics", d.handleMetrics)

	// SWE100821: Chat API — async POST + polling GET
	mux.HandleFunc("/api/chat", d.handleChat)
}

// handleDashboardPage serves the embedded interactive HTML dashboard.
func (d *Dashboard) handleDashboardPage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, dashboardHTML)
}

// --- Original APIs ---

func (d *Dashboard) handleState(w http.ResponseWriter, r *http.Request) {
	uptime := time.Since(d.startTime).Truncate(time.Second).String()
	fatigue := "low"
	model := "unknown"
	tier := "unknown"
	messagesCount := 0

	epochDir := filepath.Join(d.workspace, "epochs")
	if entries, err := os.ReadDir(epochDir); err == nil && len(entries) > 0 {
		sort.Slice(entries, func(i, j int) bool { return entries[i].Name() > entries[j].Name() })
		latest := filepath.Join(epochDir, entries[0].Name())
		if data, err := os.ReadFile(latest); err == nil {
			var ep map[string]interface{}
			if json.Unmarshal(data, &ep) == nil {
				if stats, ok := ep["stats"].(map[string]interface{}); ok {
					if mc, ok := stats["messages_processed"].(float64); ok {
						messagesCount = int(mc)
					}
				}
				if m, ok := ep["model"].(string); ok && m != "" {
					model = m
				}
			}
		}
	}

	writeJSON(w, map[string]interface{}{
		"uptime":         uptime,
		"fatigue":        fatigue,
		"messages_count": messagesCount,
		"model":          model,
		"tier":           tier,
	})
}

func (d *Dashboard) handleEpochs(w http.ResponseWriter, r *http.Request) {
	writeFileList(w, filepath.Join(d.workspace, "epochs"), false, 0)
}

func (d *Dashboard) handleProvenance(w http.ResponseWriter, r *http.Request) {
	writeFileList(w, filepath.Join(d.workspace, "provenance"), false, 50)
}

func (d *Dashboard) handleSkills(w http.ResponseWriter, r *http.Request) {
	skillsDir := filepath.Join(d.workspace, "skills")
	entries, err := os.ReadDir(skillsDir)
	if err != nil {
		writeJSON(w, []interface{}{})
		return
	}

	type skillEntry struct {
		Name    string `json:"name"`
		IsDir   bool   `json:"is_dir"`
		ModTime string `json:"mod_time"`
	}

	result := make([]skillEntry, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() || strings.HasSuffix(e.Name(), ".yaml") || strings.HasSuffix(e.Name(), ".yml") || strings.HasSuffix(e.Name(), ".md") {
			info, _ := e.Info()
			mt := ""
			if info != nil {
				mt = info.ModTime().Format(time.RFC3339)
			}
			result = append(result, skillEntry{Name: e.Name(), IsDir: e.IsDir(), ModTime: mt})
		}
	}
	writeJSON(w, result)
}

func (d *Dashboard) handlePeers(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, []interface{}{})
}

// --- Memory APIs ---

func (d *Dashboard) handleMemoryLongterm(w http.ResponseWriter, r *http.Request) {
	memFile := filepath.Join(d.workspace, "memory", "MEMORY.md")

	switch r.Method {
	case http.MethodGet:
		data, err := os.ReadFile(memFile)
		if err != nil {
			writeJSON(w, map[string]string{"content": "", "error": "not found"})
			return
		}
		writeJSON(w, map[string]string{"content": string(data)})

	case http.MethodPut:
		body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
		if err != nil {
			http.Error(w, "read error", http.StatusBadRequest)
			return
		}
		var payload struct {
			Content string `json:"content"`
		}
		if err := json.Unmarshal(body, &payload); err != nil {
			http.Error(w, "invalid JSON", http.StatusBadRequest)
			return
		}
		os.MkdirAll(filepath.Dir(memFile), 0755)
		if err := os.WriteFile(memFile, []byte(payload.Content), 0644); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, map[string]string{"status": "saved"})

	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (d *Dashboard) handleMemoryDaily(w http.ResponseWriter, r *http.Request) {
	memDir := filepath.Join(d.workspace, "memory")
	file := r.URL.Query().Get("file")

	if file != "" {
		clean := filepath.Clean(file)
		if strings.Contains(clean, "..") {
			http.Error(w, "invalid path", http.StatusBadRequest)
			return
		}
		data, err := os.ReadFile(filepath.Join(memDir, clean))
		if err != nil {
			writeJSON(w, map[string]string{"content": "", "error": "not found"})
			return
		}
		writeJSON(w, map[string]string{"path": clean, "content": string(data)})
		return
	}

	type dailyNote struct {
		Path    string `json:"path"`
		Name    string `json:"name"`
		ModTime string `json:"mod_time"`
	}

	var notes []dailyNote
	entries, err := os.ReadDir(memDir)
	if err != nil {
		writeJSON(w, []interface{}{})
		return
	}

	for _, e := range entries {
		if !e.IsDir() || len(e.Name()) != 6 || !isDigits(e.Name()) {
			continue
		}
		subDir := filepath.Join(memDir, e.Name())
		subs, err := os.ReadDir(subDir)
		if err != nil {
			continue
		}
		for _, s := range subs {
			if s.IsDir() || !strings.HasSuffix(s.Name(), ".md") {
				continue
			}
			info, _ := s.Info()
			mt := ""
			if info != nil {
				mt = info.ModTime().Format(time.RFC3339)
			}
			notes = append(notes, dailyNote{
				Path:    filepath.Join(e.Name(), s.Name()),
				Name:    s.Name(),
				ModTime: mt,
			})
		}
	}

	sort.Slice(notes, func(i, j int) bool { return notes[i].Path > notes[j].Path })
	if notes == nil {
		notes = []dailyNote{}
	}
	writeJSON(w, notes)
}

func (d *Dashboard) handleMemoryConsolidations(w http.ResponseWriter, r *http.Request) {
	type consolidation struct {
		Type    string `json:"type"`
		Name    string `json:"name"`
		ModTime string `json:"mod_time"`
	}

	file := r.URL.Query().Get("file")
	if file != "" {
		clean := filepath.Clean(file)
		if strings.Contains(clean, "..") {
			http.Error(w, "invalid path", http.StatusBadRequest)
			return
		}
		data, err := os.ReadFile(filepath.Join(d.workspace, "memory", clean))
		if err != nil {
			writeJSON(w, map[string]string{"error": "not found"})
			return
		}
		writeJSON(w, map[string]string{"content": string(data)})
		return
	}

	var result []consolidation
	for _, sub := range []string{"weekly", "monthly"} {
		dir := filepath.Join(d.workspace, "memory", sub)
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if e.IsDir() {
				continue
			}
			info, _ := e.Info()
			mt := ""
			if info != nil {
				mt = info.ModTime().Format(time.RFC3339)
			}
			result = append(result, consolidation{Type: sub, Name: e.Name(), ModTime: mt})
		}
	}

	if result == nil {
		result = []consolidation{}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ModTime > result[j].ModTime })
	writeJSON(w, result)
}

// --- Vault APIs ---

func (d *Dashboard) handleVaultTree(w http.ResponseWriter, r *http.Request) {
	if d.vaultPath == "" {
		writeJSON(w, map[string]interface{}{"error": "vault not configured", "tree": []interface{}{}})
		return
	}

	type treeNode struct {
		Name     string     `json:"name"`
		Path     string     `json:"path"`
		IsDir    bool       `json:"is_dir"`
		Children []treeNode `json:"children,omitempty"`
	}

	var walk func(dir string, depth int) []treeNode
	walk = func(dir string, depth int) []treeNode {
		if depth > 3 {
			return nil
		}
		entries, err := os.ReadDir(dir)
		if err != nil {
			return nil
		}
		var nodes []treeNode
		for _, e := range entries {
			if strings.HasPrefix(e.Name(), ".") {
				continue
			}
			rel, _ := filepath.Rel(d.vaultPath, filepath.Join(dir, e.Name()))
			node := treeNode{Name: e.Name(), Path: rel, IsDir: e.IsDir()}
			if e.IsDir() {
				node.Children = walk(filepath.Join(dir, e.Name()), depth+1)
			}
			nodes = append(nodes, node)
		}
		sort.Slice(nodes, func(i, j int) bool {
			if nodes[i].IsDir != nodes[j].IsDir {
				return nodes[i].IsDir
			}
			return nodes[i].Name < nodes[j].Name
		})
		return nodes
	}

	tree := walk(d.vaultPath, 0)
	if tree == nil {
		tree = []treeNode{}
	}
	writeJSON(w, map[string]interface{}{"root": d.vaultPath, "tree": tree})
}

func (d *Dashboard) handleVaultNote(w http.ResponseWriter, r *http.Request) {
	if d.vaultPath == "" {
		writeJSON(w, map[string]string{"error": "vault not configured"})
		return
	}
	path := r.URL.Query().Get("path")
	if path == "" {
		http.Error(w, "missing ?path=", http.StatusBadRequest)
		return
	}

	clean := filepath.Clean(path)
	if strings.Contains(clean, "..") {
		http.Error(w, "invalid path", http.StatusBadRequest)
		return
	}

	fullPath := filepath.Join(d.vaultPath, clean)
	data, err := os.ReadFile(fullPath)
	if err != nil {
		writeJSON(w, map[string]string{"error": "not found", "path": clean})
		return
	}

	info, _ := os.Stat(fullPath)
	mt := ""
	if info != nil {
		mt = info.ModTime().Format(time.RFC3339)
	}

	writeJSON(w, map[string]string{
		"path":     clean,
		"content":  string(data),
		"mod_time": mt,
	})
}

// handleVaultGraph scans all vault .md files, extracts [[wikilinks]], and returns
// a graph of nodes and edges for force-directed visualization.
// SWE100821: Capped at 2000 files to bound memory and response time.
func (d *Dashboard) handleVaultGraph(w http.ResponseWriter, r *http.Request) {
	if d.vaultPath == "" {
		writeJSON(w, map[string]interface{}{"error": "vault not configured", "nodes": []interface{}{}, "edges": []interface{}{}})
		return
	}

	type graphNode struct {
		ID    string `json:"id"`
		Name  string `json:"name"`
		Group string `json:"group"`
		Size  int    `json:"size"`
	}
	type graphEdge struct {
		Source string `json:"source"`
		Target string `json:"target"`
	}

	// Pass 1: collect all .md files and build a name→path lookup
	nameLookup := map[string]string{} // lowercase(basename_no_ext) → relative_path
	fileContents := map[string]string{}
	fileCount := 0
	const maxFiles = 2000

	filepath.WalkDir(d.vaultPath, func(path string, entry os.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return nil
		}
		if fileCount >= maxFiles {
			return filepath.SkipAll
		}
		if strings.HasPrefix(entry.Name(), ".") || !strings.HasSuffix(entry.Name(), ".md") {
			return nil
		}

		rel, _ := filepath.Rel(d.vaultPath, path)
		base := strings.TrimSuffix(entry.Name(), ".md")
		nameLookup[strings.ToLower(base)] = rel

		data, err := os.ReadFile(path)
		if err == nil {
			fileContents[rel] = string(data)
		}
		fileCount++
		return nil
	})

	// Pass 2: build nodes and edges
	connectionCount := map[string]int{}
	var edges []graphEdge
	edgeSet := map[string]bool{} // deduplicate edges

	for rel, content := range fileContents {
		matches := wikilinkRe.FindAllStringSubmatch(content, -1)
		for _, m := range matches {
			linkName := strings.TrimSpace(m[1])
			targetPath, ok := nameLookup[strings.ToLower(linkName)]
			if !ok {
				continue
			}
			if targetPath == rel {
				continue // skip self-links
			}

			edgeKey := rel + "→" + targetPath
			if edgeSet[edgeKey] {
				continue
			}
			edgeSet[edgeKey] = true
			edges = append(edges, graphEdge{Source: rel, Target: targetPath})
			connectionCount[rel]++
			connectionCount[targetPath]++
		}
	}

	// Build node list from all files that participate in at least one edge,
	// plus all files regardless (so orphans are visible too)
	nodeSet := map[string]bool{}
	var nodes []graphNode

	addNode := func(rel string) {
		if nodeSet[rel] {
			return
		}
		nodeSet[rel] = true
		parts := strings.SplitN(rel, string(os.PathSeparator), 2)
		group := "Other"
		if len(parts) >= 2 {
			group = parts[0]
		}
		name := strings.TrimSuffix(filepath.Base(rel), ".md")
		size := connectionCount[rel]
		if size < 1 {
			size = 1
		}
		nodes = append(nodes, graphNode{ID: rel, Name: name, Group: group, Size: size})
	}

	for rel := range fileContents {
		addNode(rel)
	}

	if nodes == nil {
		nodes = []graphNode{}
	}
	if edges == nil {
		edges = []graphEdge{}
	}

	writeJSON(w, map[string]interface{}{
		"nodes": nodes,
		"edges": edges,
	})
}

// --- Detail APIs ---

func (d *Dashboard) handleEpochDetail(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("name")
	if name == "" {
		http.Error(w, "missing ?name=", http.StatusBadRequest)
		return
	}
	clean := filepath.Base(name)
	data, err := os.ReadFile(filepath.Join(d.workspace, "epochs", clean))
	if err != nil {
		writeJSON(w, map[string]string{"error": "not found"})
		return
	}
	var parsed interface{}
	if json.Unmarshal(data, &parsed) == nil {
		writeJSON(w, parsed)
	} else {
		writeJSON(w, map[string]string{"raw": string(data)})
	}
}

func (d *Dashboard) handleProvenanceDetail(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("name")
	if name == "" {
		http.Error(w, "missing ?name=", http.StatusBadRequest)
		return
	}
	clean := filepath.Base(name)
	data, err := os.ReadFile(filepath.Join(d.workspace, "provenance", clean))
	if err != nil {
		writeJSON(w, map[string]string{"error": "not found"})
		return
	}
	var records []interface{}
	for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var obj interface{}
		if json.Unmarshal([]byte(line), &obj) == nil {
			records = append(records, obj)
		}
	}
	if records == nil {
		records = []interface{}{}
	}
	writeJSON(w, records)
}

// --- Config / System APIs ---

func (d *Dashboard) handleConfig(w http.ResponseWriter, r *http.Request) {
	if d.configPath == "" {
		writeJSON(w, map[string]string{"error": "config path not set"})
		return
	}
	data, err := os.ReadFile(d.configPath)
	if err != nil {
		writeJSON(w, map[string]string{"error": "cannot read config"})
		return
	}
	sensitivePattern := regexp.MustCompile(`(?i)"(api_key|token|secret|password|app_secret|access_token|bot_token|app_token|channel_secret|channel_access_token|client_secret|encrypt_key|verification_token)":\s*"[^"]*"`)
	redacted := sensitivePattern.ReplaceAllStringFunc(string(data), func(match string) string {
		parts := strings.SplitN(match, ":", 2)
		if len(parts) == 2 && strings.TrimSpace(parts[1]) != `""` {
			return parts[0] + `: "***REDACTED***"`
		}
		return match
	})
	var parsed interface{}
	if json.Unmarshal([]byte(redacted), &parsed) == nil {
		writeJSON(w, parsed)
	} else {
		writeJSON(w, map[string]string{"raw": redacted})
	}
}

func (d *Dashboard) handleCronJobs(w http.ResponseWriter, r *http.Request) {
	data, err := os.ReadFile(filepath.Join(d.workspace, "cron", "jobs.json"))
	if err != nil {
		writeJSON(w, []interface{}{})
		return
	}
	var parsed interface{}
	if json.Unmarshal(data, &parsed) == nil {
		writeJSON(w, parsed)
	} else {
		writeJSON(w, []interface{}{})
	}
}

func (d *Dashboard) handleMetrics(w http.ResponseWriter, r *http.Request) {
	result := map[string]interface{}{
		"uptime_seconds": int(time.Since(d.startTime).Seconds()),
	}
	if d.metrics != nil {
		snap := d.metrics.Snapshot()
		for k, v := range snap {
			result[k] = v
		}
	}
	writeJSON(w, result)
}

// --- Chat API ---

// handleChat processes chat messages asynchronously.
// POST {message, session_key} → {id} (spawns goroutine, returns immediately)
// GET  ?id=xxx             → {done, response, error}
// SWE100821: Async polling avoids health server's 5s WriteTimeout constraint.
func (d *Dashboard) handleChat(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		if d.chatHandler == nil {
			writeJSON(w, map[string]string{"error": "chat not available"})
			return
		}
		body, err := io.ReadAll(io.LimitReader(r.Body, 1<<16))
		if err != nil {
			http.Error(w, "read error", http.StatusBadRequest)
			return
		}
		var payload struct {
			Message    string `json:"message"`
			SessionKey string `json:"session_key"`
		}
		if err := json.Unmarshal(body, &payload); err != nil || payload.Message == "" {
			http.Error(w, "invalid request — need {message}", http.StatusBadRequest)
			return
		}
		if payload.SessionKey == "" {
			payload.SessionKey = "dashboard:direct"
		}

		id := fmt.Sprintf("%d", time.Now().UnixNano())
		req := &chatReq{Created: time.Now()}

		d.chatMu.Lock()
		// SWE100821: Purge stale requests older than 5 minutes
		for k, v := range d.chatReqs {
			if time.Since(v.Created) > 5*time.Minute {
				delete(d.chatReqs, k)
			}
		}
		d.chatReqs[id] = req
		d.chatMu.Unlock()

		go func() {
			resp, err := d.chatHandler(context.Background(), payload.Message, payload.SessionKey)
			d.chatMu.Lock()
			if err != nil {
				req.Error = err.Error()
			}
			req.Response = resp
			req.Done = true
			d.chatMu.Unlock()
		}()

		writeJSON(w, map[string]string{"id": id})

	case http.MethodGet:
		id := r.URL.Query().Get("id")
		if id == "" {
			http.Error(w, "missing ?id=", http.StatusBadRequest)
			return
		}
		d.chatMu.Lock()
		req, ok := d.chatReqs[id]
		d.chatMu.Unlock()
		if !ok {
			writeJSON(w, map[string]interface{}{"error": "unknown request", "done": true})
			return
		}
		d.chatMu.Lock()
		writeJSON(w, map[string]interface{}{
			"done":     req.Done,
			"response": req.Response,
			"error":    req.Error,
		})
		d.chatMu.Unlock()

	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

// --- Helpers ---

func writeJSON(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(v)
}

func writeFileList(w http.ResponseWriter, dir string, dirsOnly bool, maxEntries int) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		writeJSON(w, []interface{}{})
		return
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name() > entries[j].Name() })

	type fileEntry struct {
		Name    string `json:"name"`
		ModTime string `json:"mod_time"`
	}

	var result []fileEntry
	for _, e := range entries {
		if dirsOnly && !e.IsDir() {
			continue
		}
		if !dirsOnly && e.IsDir() {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		result = append(result, fileEntry{
			Name:    e.Name(),
			ModTime: info.ModTime().Format(time.RFC3339),
		})
		if maxEntries > 0 && len(result) >= maxEntries {
			break
		}
	}
	if result == nil {
		result = []fileEntry{}
	}
	writeJSON(w, result)
}

func isDigits(s string) bool {
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}
