// SWE100821: Cognitive dashboard — embedded web UI for observing agent state.
// Serves a dark-themed HTML dashboard and JSON APIs for epochs, provenance, skills, peers.
// No external file dependencies; HTML is a const string.

package dashboard

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Dashboard provides an embedded HTTP dashboard for agent introspection.
type Dashboard struct {
	workspace string
	mux       *http.ServeMux
	startTime time.Time
}

// NewDashboard creates a dashboard rooted at the given workspace.
func NewDashboard(workspace string) *Dashboard {
	return &Dashboard{
		workspace: workspace,
		startTime: time.Now(),
	}
}

// SetupRoutes registers all dashboard routes on the given mux.
// SWE100821: Call before the health server starts.
func (d *Dashboard) SetupRoutes(mux *http.ServeMux) {
	d.mux = mux
	mux.HandleFunc("/dashboard", d.handleDashboardPage)
	mux.HandleFunc("/api/state", d.handleState)
	mux.HandleFunc("/api/epochs", d.handleEpochs)
	mux.HandleFunc("/api/provenance", d.handleProvenance)
	mux.HandleFunc("/api/skills", d.handleSkills)
	mux.HandleFunc("/api/peers", d.handlePeers)
}

// handleDashboardPage serves the embedded HTML dashboard.
func (d *Dashboard) handleDashboardPage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, dashboardHTML)
}

// handleState returns current agent state as JSON.
func (d *Dashboard) handleState(w http.ResponseWriter, r *http.Request) {
	// SWE100821: Aggregate state from workspace files
	uptime := time.Since(d.startTime).Truncate(time.Second).String()

	fatigue := "low"
	model := "unknown"
	tier := "unknown"
	messagesCount := 0

	// Read current epoch for stats
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

// handleEpochs lists epoch files from workspace/epochs/.
func (d *Dashboard) handleEpochs(w http.ResponseWriter, r *http.Request) {
	// SWE100821: List epoch journal files
	epochDir := filepath.Join(d.workspace, "epochs")
	entries, err := os.ReadDir(epochDir)
	if err != nil {
		writeJSON(w, []interface{}{})
		return
	}

	sort.Slice(entries, func(i, j int) bool { return entries[i].Name() > entries[j].Name() })

	type epochEntry struct {
		Name    string `json:"name"`
		ModTime string `json:"mod_time"`
	}

	result := make([]epochEntry, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		result = append(result, epochEntry{
			Name:    e.Name(),
			ModTime: info.ModTime().Format(time.RFC3339),
		})
	}
	writeJSON(w, result)
}

// handleProvenance returns the last 50 provenance entries.
func (d *Dashboard) handleProvenance(w http.ResponseWriter, r *http.Request) {
	// SWE100821: Tail provenance log
	provDir := filepath.Join(d.workspace, "provenance")
	entries, err := os.ReadDir(provDir)
	if err != nil {
		writeJSON(w, []interface{}{})
		return
	}

	sort.Slice(entries, func(i, j int) bool { return entries[i].Name() > entries[j].Name() })

	const maxEntries = 50
	type provEntry struct {
		Name    string `json:"name"`
		ModTime string `json:"mod_time"`
	}

	result := make([]provEntry, 0, maxEntries)
	for i, e := range entries {
		if i >= maxEntries {
			break
		}
		if e.IsDir() {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		result = append(result, provEntry{
			Name:    e.Name(),
			ModTime: info.ModTime().Format(time.RFC3339),
		})
	}
	writeJSON(w, result)
}

// handleSkills lists installed skills from workspace/skills/.
func (d *Dashboard) handleSkills(w http.ResponseWriter, r *http.Request) {
	// SWE100821: Enumerate skill directories
	skillsDir := filepath.Join(d.workspace, "skills")
	entries, err := os.ReadDir(skillsDir)
	if err != nil {
		writeJSON(w, []interface{}{})
		return
	}

	names := make([]string, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() || strings.HasSuffix(e.Name(), ".yaml") || strings.HasSuffix(e.Name(), ".yml") {
			names = append(names, e.Name())
		}
	}
	writeJSON(w, names)
}

// handlePeers is a placeholder that returns an empty array.
func (d *Dashboard) handlePeers(w http.ResponseWriter, r *http.Request) {
	// SWE100821: Placeholder — future A2A peer discovery
	writeJSON(w, []interface{}{})
}

func writeJSON(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(v)
}

// SWE100821: Embedded HTML dashboard — dark-themed, auto-refresh, CSS grid, vanilla JS.
const dashboardHTML = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>Xagent Dashboard</title>
<style>
*{margin:0;padding:0;box-sizing:border-box}
body{font-family:'Segoe UI',system-ui,sans-serif;background:#0d1117;color:#c9d1d9;min-height:100vh}
header{background:#161b22;border-bottom:1px solid #30363d;padding:16px 24px;display:flex;align-items:center;gap:12px}
header h1{font-size:1.25rem;font-weight:600;color:#58a6ff}
header .dot{width:10px;height:10px;border-radius:50%;background:#3fb950;animation:pulse 2s infinite}
@keyframes pulse{0%,100%{opacity:1}50%{opacity:.4}}
main{max-width:1200px;margin:0 auto;padding:24px}
.cards{display:grid;grid-template-columns:repeat(auto-fit,minmax(200px,1fr));gap:16px;margin-bottom:32px}
.card{background:#161b22;border:1px solid #30363d;border-radius:8px;padding:20px}
.card .label{font-size:.75rem;color:#8b949e;text-transform:uppercase;letter-spacing:.05em;margin-bottom:4px}
.card .value{font-size:1.5rem;font-weight:600;color:#f0f6fc}
.section{margin-bottom:24px}
.section h2{font-size:1rem;color:#8b949e;margin-bottom:12px;border-bottom:1px solid #21262d;padding-bottom:8px}
.list{background:#161b22;border:1px solid #30363d;border-radius:8px;max-height:300px;overflow-y:auto}
.list-item{padding:10px 16px;border-bottom:1px solid #21262d;font-size:.875rem;display:flex;justify-content:space-between}
.list-item:last-child{border-bottom:none}
.list-item .time{color:#8b949e;font-size:.75rem}
.empty{color:#484f58;padding:16px;text-align:center;font-style:italic}
footer{text-align:center;color:#484f58;font-size:.75rem;padding:16px}
</style>
</head>
<body>
<header><div class="dot"></div><h1>Xagent Cognitive Dashboard</h1></header>
<main>
<div class="cards">
  <div class="card"><div class="label">Uptime</div><div class="value" id="uptime">—</div></div>
  <div class="card"><div class="label">Fatigue</div><div class="value" id="fatigue">—</div></div>
  <div class="card"><div class="label">Messages</div><div class="value" id="messages">—</div></div>
  <div class="card"><div class="label">Model</div><div class="value" id="model">—</div></div>
  <div class="card"><div class="label">Tier</div><div class="value" id="tier">—</div></div>
</div>
<div class="section"><h2>Epochs</h2><div class="list" id="epochs"><div class="empty">Loading...</div></div></div>
<div class="section"><h2>Provenance</h2><div class="list" id="provenance"><div class="empty">Loading...</div></div></div>
<div class="section"><h2>Skills</h2><div class="list" id="skills"><div class="empty">Loading...</div></div></div>
<div class="section"><h2>Peers</h2><div class="list" id="peers"><div class="empty">Loading...</div></div></div>
</main>
<footer>Xagent Cognitive Dashboard — auto-refreshes every 10s</footer>
<script>
async function load(url){try{const r=await fetch(url);return await r.json()}catch{return null}}
function renderList(id,items,fn){
  const el=document.getElementById(id);
  if(!items||items.length===0){el.innerHTML='<div class="empty">None</div>';return}
  el.innerHTML=items.map(fn).join('')
}
async function refresh(){
  const s=await load('/api/state');
  if(s){
    document.getElementById('uptime').textContent=s.uptime||'—';
    document.getElementById('fatigue').textContent=s.fatigue||'—';
    document.getElementById('messages').textContent=s.messages_count!=null?s.messages_count:'—';
    document.getElementById('model').textContent=s.model||'—';
    document.getElementById('tier').textContent=s.tier||'—';
  }
  const ep=await load('/api/epochs');
  renderList('epochs',ep,e=>'<div class="list-item"><span>'+e.name+'</span><span class="time">'+e.mod_time+'</span></div>');
  const pr=await load('/api/provenance');
  renderList('provenance',pr,e=>'<div class="list-item"><span>'+e.name+'</span><span class="time">'+e.mod_time+'</span></div>');
  const sk=await load('/api/skills');
  renderList('skills',sk,e=>'<div class="list-item"><span>'+e+'</span></div>');
  const pe=await load('/api/peers');
  renderList('peers',pe,e=>'<div class="list-item"><span>'+JSON.stringify(e)+'</span></div>');
}
refresh();
setInterval(refresh,10000);
</script>
</body>
</html>`
