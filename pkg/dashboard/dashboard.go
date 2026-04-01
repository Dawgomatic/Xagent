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

// SWE100821: Pre-compiled regex — was recompiling per /api/config request
var sensitiveConfigRe = regexp.MustCompile(`(?i)"(api_key|token|secret|password|app_secret|access_token|bot_token|app_token|channel_secret|channel_access_token|client_secret|encrypt_key|verification_token)":\s*"[^"]*"`)

// SWE100821: Regex for extracting [[wikilinks]] from vault notes (handles [[name]] and [[name|alias]])
var wikilinkRe = regexp.MustCompile(`\[\[([^\]\|]+)(?:\|[^\]]+)?\]\]`)

// Dashboard provides an embedded HTTP dashboard for agent introspection.
// SWE100821: Expanded with chat, vault graph, full subsystem browsing, and task queue.
type Dashboard struct {
	workspace      string
	vaultPath      string
	configPath     string
	metrics        *health.Metrics
	watchdog       *health.Watchdog
	model          string // SWE100821: Active LLM model name
	tier           string // SWE100821: Hardware tier
	startTime      time.Time
	chatHandler    func(ctx context.Context, message, sessionKey string) (string, error)
	chatMu         sync.Mutex
	chatReqs       map[string]*chatReq
	sensorProvider func() string      // SWE100821: Returns formatted sensor readings
	toolLister     func() []string    // SWE100821: Returns registered tool names
	fatigueFunc    func() float64     // SWE100821: Returns live fatigue level (0.0-1.0)
	taskLister     func() interface{}             // SWE100821: Returns serializable task list
	gcStatusFunc   func() map[string]interface{}  // SWE100821: Returns GC last run + fatigue stats
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

// SWE100821: SetWatchdog injects the subsystem watchdog for /api/watchdog status.
func (d *Dashboard) SetWatchdog(w *health.Watchdog) { d.watchdog = w }

// SWE100821: SetModelInfo sets the active model and hardware tier for the overview.
func (d *Dashboard) SetModelInfo(model, tier string) { d.model = model; d.tier = tier }

// SetChatHandler sets the function called to process chat messages.
// SWE100821: Signature matches agentLoop.ProcessDirect(ctx, content, sessionKey).
func (d *Dashboard) SetChatHandler(fn func(ctx context.Context, message, sessionKey string) (string, error)) {
	d.chatHandler = fn
}

// SWE100821: SetSensorProvider injects a function returning formatted sensor readings.
func (d *Dashboard) SetSensorProvider(fn func() string) { d.sensorProvider = fn }

// SWE100821: SetToolLister injects a function returning registered tool names.
func (d *Dashboard) SetToolLister(fn func() []string) { d.toolLister = fn }

// SWE100821: SetFatigueFunc injects a function returning live fatigue level (0.0-1.0).
func (d *Dashboard) SetFatigueFunc(fn func() float64) { d.fatigueFunc = fn }

	// SWE100821: SetTaskLister injects a function returning the serializable task list.
func (d *Dashboard) SetTaskLister(fn func() interface{}) { d.taskLister = fn }

// SWE100821: SetGCStatusFunc injects a function returning GC last-run time and fatigue.
func (d *Dashboard) SetGCStatusFunc(fn func() map[string]interface{}) { d.gcStatusFunc = fn }

// SetupRoutes registers all dashboard routes on the given mux.
// SWE100821: Includes original, memory, vault, detail, config, chat, and graph APIs.
func (d *Dashboard) SetupRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/dashboard", d.handleDashboardPage)
	// SWE100821: Browsers often request /dashboard/ — without this, net/http returns 404 for /dashboard/
	mux.HandleFunc("/dashboard/", d.handleDashboardPage)

	// Original APIs
	mux.HandleFunc("/api/state", d.handleState)
	mux.HandleFunc("/api/epochs", d.handleEpochs)
	mux.HandleFunc("/api/provenance", d.handleProvenance)
	mux.HandleFunc("/api/skills", d.handleSkills)
	// SWE100821: Serve individual skill SKILL.md content for dashboard detail view
	mux.HandleFunc("/api/skill/detail", d.handleSkillDetail)
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

	// SWE100821: Watchdog API — subsystem health status
	mux.HandleFunc("/api/watchdog", d.handleWatchdog)

	// SWE100821: New APIs for sensors, tools, goals
	mux.HandleFunc("/api/sensors", d.handleSensors)
	mux.HandleFunc("/api/tools", d.handleTools)
	mux.HandleFunc("/api/goals", d.handleGoals)

	// SWE100821: Chat API — async POST + polling GET
	mux.HandleFunc("/api/chat", d.handleChat)

	// SWE100821: Task queue API
	mux.HandleFunc("/api/tasks", d.handleTasks)
}

// handleDashboardPage serves the embedded interactive HTML dashboard.
func (d *Dashboard) handleDashboardPage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	// SWE100821: HTML is embedded at build time — avoid caching stale dashboard after upgrades
	w.Header().Set("Cache-Control", "no-store, must-revalidate")
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, dashboardHTML)
}

// --- Original APIs ---

func (d *Dashboard) handleState(w http.ResponseWriter, r *http.Request) {
	uptime := time.Since(d.startTime).Truncate(time.Second).String()
	// SWE100821: Use live fatigue from provider instead of hardcoded "low"
	fatigue := "low"
	if d.fatigueFunc != nil {
		f := d.fatigueFunc()
		switch {
		case f >= 0.8:
			fatigue = fmt.Sprintf("critical (%.0f%%)", f*100)
		case f >= 0.5:
			fatigue = fmt.Sprintf("high (%.0f%%)", f*100)
		case f >= 0.2:
			fatigue = fmt.Sprintf("medium (%.0f%%)", f*100)
		default:
			fatigue = fmt.Sprintf("low (%.0f%%)", f*100)
		}
	}
	messagesCount := 0

	// SWE100821: Use live model/tier from SetModelInfo, not epoch files
	model := d.model
	if model == "" {
		model = "unknown"
	}
	tier := d.tier
	if tier == "" {
		tier = "unknown"
	}

	// SWE100821: Get live fatigue from metrics if available
	if d.metrics != nil {
		snap := d.metrics.Snapshot()
		if total, ok := snap["messages_total"].(int64); ok {
			messagesCount = int(total)
		}
	}

	// Fallback to epoch files for message count if metrics not available
	if messagesCount == 0 {
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
				}
			}
		}
	}

	// SWE100821: Include live system time so dashboard can display it without a separate endpoint
	now := time.Now()
	zone, offsetSecs := now.Local().Zone()
	resp := map[string]interface{}{
		"uptime":         uptime,
		"fatigue":        fatigue,
		"messages_count": messagesCount,
		"model":          model,
		"tier":           tier,
		"current_time":   now.UTC().Format("2006-01-02 15:04:05 UTC"),
		"local_time":     now.Local().Format("2006-01-02 15:04:05 MST"),
		"timezone":       fmt.Sprintf("%s (UTC%+.1f)", zone, float64(offsetSecs)/3600.0),
		"unix":           now.UTC().Unix(),
	}
	// SWE100821: Inject GC status (last run, next scheduled) for dashboard overview
	if d.gcStatusFunc != nil {
		for k, v := range d.gcStatusFunc() {
			resp[k] = v
		}
	}
	writeJSON(w, resp)
}

func (d *Dashboard) handleEpochs(w http.ResponseWriter, r *http.Request) {
	writeFileList(w, filepath.Join(d.workspace, "epochs"), false, 0)
}

func (d *Dashboard) handleProvenance(w http.ResponseWriter, r *http.Request) {
	writeFileList(w, filepath.Join(d.workspace, "provenance"), false, 50)
}

// SWE100821: handleSkills walks both flat (skills/<name>/SKILL.md) and
// author-qualified (skills/<author>/<skill>/SKILL.md) layouts.
func (d *Dashboard) handleSkills(w http.ResponseWriter, r *http.Request) {
	type skillEntry struct {
		Author      string `json:"author"`
		Name        string `json:"name"`
		Path        string `json:"path"`
		Description string `json:"description,omitempty"`
		ModTime     string `json:"mod_time,omitempty"`
	}

	var result []skillEntry
	seen := map[string]bool{}

	roots := []string{
		filepath.Join(d.workspace, "skills"),
		filepath.Join(d.workspace, "skills", "skills"),
	}

	for _, root := range roots {
		entries, err := os.ReadDir(root)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if !e.IsDir() || strings.HasPrefix(e.Name(), ".") {
				continue
			}
			dirPath := filepath.Join(root, e.Name())

			// Check flat layout: skills/<name>/SKILL.md
			flatMD := filepath.Join(dirPath, "SKILL.md")
			if _, err := os.Stat(flatMD); err == nil {
				path := e.Name()
				if seen[path] {
					continue
				}
				seen[path] = true
				entry := skillEntry{Author: "local", Name: e.Name(), Path: path}
				if info, err := e.Info(); err == nil {
					entry.ModTime = info.ModTime().Format(time.RFC3339)
				}
				if data, err := os.ReadFile(flatMD); err == nil {
					entry.Description = extractFrontmatterField(string(data), "description")
				}
				result = append(result, entry)
				continue
			}

			// Check author layout: skills/<author>/<skill>/SKILL.md
			subEntries, err := os.ReadDir(dirPath)
			if err != nil {
				continue
			}
			for _, sub := range subEntries {
				if !sub.IsDir() || strings.HasPrefix(sub.Name(), ".") {
					continue
				}
				subMD := filepath.Join(dirPath, sub.Name(), "SKILL.md")
				if _, err := os.Stat(subMD); err != nil {
					continue
				}
				path := e.Name() + "/" + sub.Name()
				if seen[path] {
					continue
				}
				seen[path] = true
				entry := skillEntry{Author: e.Name(), Name: sub.Name(), Path: path}
				if info, err := sub.Info(); err == nil {
					entry.ModTime = info.ModTime().Format(time.RFC3339)
				}

				metaPath := filepath.Join(dirPath, sub.Name(), "_meta.json")
				if metaData, err := os.ReadFile(metaPath); err == nil {
					var meta struct {
						DisplayName string `json:"displayName"`
					}
					if json.Unmarshal(metaData, &meta) == nil && meta.DisplayName != "" {
						entry.Description = meta.DisplayName
					}
				}
				if entry.Description == "" {
					if data, err := os.ReadFile(subMD); err == nil {
						entry.Description = extractFrontmatterField(string(data), "description")
					}
				}
				result = append(result, entry)
			}
		}
	}

	if result == nil {
		result = []skillEntry{}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Path < result[j].Path })
	writeJSON(w, result)
}

// SWE100821: handleSkillDetail returns the full SKILL.md content for a given skill path.
// GET /api/skill/detail?path=skill-name  (flat) or ?path=author/skill-name (qualified)
func (d *Dashboard) handleSkillDetail(w http.ResponseWriter, r *http.Request) {
	skillPath := r.URL.Query().Get("path")
	if skillPath == "" {
		http.Error(w, "missing ?path=author/skill", http.StatusBadRequest)
		return
	}

	clean := filepath.Clean(skillPath)
	if strings.Contains(clean, "..") {
		http.Error(w, "invalid path", http.StatusBadRequest)
		return
	}

	// Search in multiple roots, both flat and archive layout
	roots := []string{
		filepath.Join(d.workspace, "skills"),
		filepath.Join(d.workspace, "skills", "skills"),
	}

	for _, root := range roots {
		mdPath := filepath.Join(root, clean, "SKILL.md")
		data, err := os.ReadFile(mdPath)
		if err != nil {
			continue
		}
		info, _ := os.Stat(mdPath)
		mt := ""
		if info != nil {
			mt = info.ModTime().Format(time.RFC3339)
		}
		writeJSON(w, map[string]string{
			"path":     clean,
			"content":  string(data),
			"mod_time": mt,
		})
		return
	}

	writeJSON(w, map[string]string{"error": "skill not found", "path": clean})
}

// extractFrontmatterField pulls a single field from YAML frontmatter (--- delimited).
func extractFrontmatterField(content, field string) string {
	if !strings.HasPrefix(content, "---") {
		return ""
	}
	end := strings.Index(content[3:], "---")
	if end < 0 {
		return ""
	}
	fm := content[3 : 3+end]
	for _, line := range strings.Split(fm, "\n") {
		line = strings.TrimSpace(line)
		prefix := field + ":"
		if strings.HasPrefix(line, prefix) {
			return strings.TrimSpace(strings.TrimPrefix(line, prefix))
		}
	}
	return ""
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
// SWE100821: Supports query params for filtering:
//   - min_conn=N  — only include nodes with >= N connections (default 1, hides orphans)
//   - group=X     — only include nodes from this group (empty = all)
//   - exclude=X,Y — comma-separated groups to exclude (e.g. "Sessions")
//   - max_nodes=N — cap total nodes (default 200, sorted by connection count)
func (d *Dashboard) handleVaultGraph(w http.ResponseWriter, r *http.Request) {
	if d.vaultPath == "" {
		writeJSON(w, map[string]interface{}{"error": "vault not configured", "nodes": []interface{}{}, "edges": []interface{}{}})
		return
	}

	// SWE100821: Parse filter params
	minConn := 1
	if v := r.URL.Query().Get("min_conn"); v != "" {
		if n, err := fmt.Sscanf(v, "%d", &minConn); n != 1 || err != nil {
			minConn = 1
		}
	}
	maxNodes := 200
	if v := r.URL.Query().Get("max_nodes"); v != "" {
		if n, err := fmt.Sscanf(v, "%d", &maxNodes); n != 1 || err != nil {
			maxNodes = 200
		}
	}
	filterGroup := r.URL.Query().Get("group")
	excludeGroups := map[string]bool{}
	if v := r.URL.Query().Get("exclude"); v != "" {
		for _, g := range strings.Split(v, ",") {
			excludeGroups[strings.TrimSpace(g)] = true
		}
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
	nameLookup := map[string]string{}
	fileContents := map[string]string{}
	fileGroups := map[string]string{}
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

		parts := strings.SplitN(rel, string(os.PathSeparator), 2)
		group := "Other"
		if len(parts) >= 2 {
			group = parts[0]
		}
		fileGroups[rel] = group

		data, readErr := os.ReadFile(path)
		if readErr == nil {
			fileContents[rel] = string(data)
		}
		fileCount++
		return nil
	})

	// Pass 2: build edges and count connections
	connectionCount := map[string]int{}
	type rawEdge struct{ src, dst string }
	var allEdges []rawEdge
	edgeSet := map[string]bool{}

	for rel, content := range fileContents {
		matches := wikilinkRe.FindAllStringSubmatch(content, -1)
		for _, m := range matches {
			linkName := strings.TrimSpace(m[1])
			targetPath, ok := nameLookup[strings.ToLower(linkName)]
			if !ok || targetPath == rel {
				continue
			}
			edgeKey := rel + "→" + targetPath
			if edgeSet[edgeKey] {
				continue
			}
			edgeSet[edgeKey] = true
			allEdges = append(allEdges, rawEdge{rel, targetPath})
			connectionCount[rel]++
			connectionCount[targetPath]++
		}
	}

	// SWE100821: Pass 3 — filter and rank nodes
	type rankedNode struct {
		rel   string
		group string
		conns int
	}
	var candidates []rankedNode
	for rel := range fileContents {
		group := fileGroups[rel]
		conns := connectionCount[rel]
		if conns < minConn {
			continue
		}
		if filterGroup != "" && group != filterGroup {
			continue
		}
		if excludeGroups[group] {
			continue
		}
		candidates = append(candidates, rankedNode{rel, group, conns})
	}
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].conns > candidates[j].conns
	})
	if len(candidates) > maxNodes {
		candidates = candidates[:maxNodes]
	}

	// Build final node set
	nodeSet := map[string]bool{}
	var nodes []graphNode
	for _, c := range candidates {
		nodeSet[c.rel] = true
		name := strings.TrimSuffix(filepath.Base(c.rel), ".md")
		nodes = append(nodes, graphNode{ID: c.rel, Name: name, Group: c.group, Size: c.conns})
	}

	// Only include edges where both endpoints survived filtering
	var edges []graphEdge
	for _, e := range allEdges {
		if nodeSet[e.src] && nodeSet[e.dst] {
			edges = append(edges, graphEdge{Source: e.src, Target: e.dst})
		}
	}

	if nodes == nil {
		nodes = []graphNode{}
	}
	if edges == nil {
		edges = []graphEdge{}
	}

	writeJSON(w, map[string]interface{}{
		"nodes":       nodes,
		"edges":       edges,
		"total_files": fileCount,
		"filtered":    len(candidates) < len(fileContents),
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
	// SWE100821: use pre-compiled sensitiveConfigRe
	redacted := sensitiveConfigRe.ReplaceAllStringFunc(string(data), func(match string) string {
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

// --- Watchdog API ---

// SWE100821: handleWatchdog returns subsystem health status for the dashboard.
func (d *Dashboard) handleWatchdog(w http.ResponseWriter, r *http.Request) {
	if d.watchdog == nil {
		writeJSON(w, map[string]interface{}{"subsystems": []interface{}{}, "all_healthy": true})
		return
	}
	statuses := d.watchdog.GetStatus()
	writeJSON(w, map[string]interface{}{
		"subsystems":  statuses,
		"all_healthy": d.watchdog.AllHealthy(),
	})
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

// --- Sensors API ---

// SWE100821: handleSensors returns live sensor readings from the perception subsystem.
func (d *Dashboard) handleSensors(w http.ResponseWriter, r *http.Request) {
	if d.sensorProvider == nil {
		writeJSON(w, map[string]interface{}{"readings": "", "available": false})
		return
	}
	readings := d.sensorProvider()
	writeJSON(w, map[string]interface{}{
		"readings":  readings,
		"available": readings != "",
	})
}

// --- Tools API ---

// SWE100821: handleTools returns the list of registered tool names.
func (d *Dashboard) handleTools(w http.ResponseWriter, r *http.Request) {
	if d.toolLister == nil {
		writeJSON(w, []string{})
		return
	}
	names := d.toolLister()
	sort.Strings(names)
	writeJSON(w, names)
}

// --- Goals API ---

// SWE100821: handleGoals returns the GOALS.md file content.
func (d *Dashboard) handleGoals(w http.ResponseWriter, r *http.Request) {
	goalsPath := filepath.Join(d.workspace, "GOALS.md")
	data, err := os.ReadFile(goalsPath)
	if err != nil {
		writeJSON(w, map[string]interface{}{"content": "", "exists": false})
		return
	}
	writeJSON(w, map[string]interface{}{
		"content": string(data),
		"exists":  true,
	})
}

// --- Tasks API ---

// SWE100821: handleTasks returns the full task list for the dashboard task tab.
func (d *Dashboard) handleTasks(w http.ResponseWriter, r *http.Request) {
	if d.taskLister == nil {
		writeJSON(w, []interface{}{})
		return
	}
	writeJSON(w, d.taskLister())
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
