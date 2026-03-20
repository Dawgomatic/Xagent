// SWE100821: Interactive dashboard HTML — tabbed SPA with chat interface, vault graph
// visualization (force-directed), memory editor, vault browser, epoch/provenance detail
// views, config viewer, and system metrics.
// Embedded as a Go const string; no build step or external assets required.

package dashboard

const dashboardHTML = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>Xagent Dashboard</title>
<style>
*{margin:0;padding:0;box-sizing:border-box}
:root{--bg:#0d1117;--surface:#161b22;--border:#30363d;--text:#c9d1d9;--text-dim:#8b949e;--text-bright:#f0f6fc;--accent:#58a6ff;--accent-hover:#79c0ff;--green:#3fb950;--orange:#d29922;--red:#f85149;--purple:#a371f7;--cyan:#79c0ff;--sidebar-w:220px}
body{font-family:'Segoe UI',system-ui,-apple-system,sans-serif;background:var(--bg);color:var(--text);min-height:100vh;display:flex}
#sidebar{width:var(--sidebar-w);min-height:100vh;background:var(--surface);border-right:1px solid var(--border);display:flex;flex-direction:column;position:fixed;top:0;left:0;z-index:10}
#sidebar .logo{padding:16px 20px;border-bottom:1px solid var(--border);display:flex;align-items:center;gap:10px}
#sidebar .logo .dot{width:10px;height:10px;border-radius:50%;background:var(--green);animation:pulse 2s infinite}
#sidebar .logo h1{font-size:1rem;font-weight:600;color:var(--accent)}
@keyframes pulse{0%,100%{opacity:1}50%{opacity:.4}}
.nav-item{padding:10px 20px;cursor:pointer;display:flex;align-items:center;gap:10px;font-size:.875rem;color:var(--text-dim);border-left:3px solid transparent;transition:all .15s}
.nav-item:hover{background:rgba(88,166,255,.08);color:var(--text)}
.nav-item.active{background:rgba(88,166,255,.12);color:var(--accent);border-left-color:var(--accent)}
.nav-icon{width:20px;text-align:center;font-size:1rem}
#main{margin-left:var(--sidebar-w);flex:1;min-height:100vh}
#main header{background:var(--surface);border-bottom:1px solid var(--border);padding:14px 24px;display:flex;align-items:center;justify-content:space-between}
#main header h2{font-size:1.1rem;font-weight:600;color:var(--text-bright)}
#main header .meta{font-size:.75rem;color:var(--text-dim)}
.page{display:none;padding:24px;animation:fadeIn .2s}
.page.active{display:block}
@keyframes fadeIn{from{opacity:0}to{opacity:1}}
.cards{display:grid;grid-template-columns:repeat(auto-fit,minmax(180px,1fr));gap:14px;margin-bottom:24px}
.card{background:var(--surface);border:1px solid var(--border);border-radius:8px;padding:16px}
.card .label{font-size:.7rem;color:var(--text-dim);text-transform:uppercase;letter-spacing:.05em;margin-bottom:4px}
.card .value{font-size:1.4rem;font-weight:600;color:var(--text-bright)}
.card .value.sm{font-size:1rem}
.section{margin-bottom:20px}
.section h3{font-size:.9rem;color:var(--text-dim);margin-bottom:10px;border-bottom:1px solid var(--border);padding-bottom:6px}
.list{background:var(--surface);border:1px solid var(--border);border-radius:8px;max-height:350px;overflow-y:auto}
.list-item{padding:10px 14px;border-bottom:1px solid var(--border);font-size:.8rem;display:flex;justify-content:space-between;align-items:center;cursor:pointer;transition:background .1s}
.list-item:last-child{border-bottom:none}
.list-item:hover{background:rgba(88,166,255,.06)}
.list-item .time{color:var(--text-dim);font-size:.7rem}
.empty{color:var(--text-dim);padding:16px;text-align:center;font-style:italic;font-size:.85rem}
.split{display:grid;grid-template-columns:280px 1fr;gap:16px;min-height:500px}
.split .panel{background:var(--surface);border:1px solid var(--border);border-radius:8px;overflow:hidden}
.panel-header{padding:10px 14px;background:rgba(255,255,255,.02);border-bottom:1px solid var(--border);font-size:.8rem;font-weight:600;color:var(--text-dim);display:flex;justify-content:space-between;align-items:center}
.panel-body{padding:0;overflow-y:auto;max-height:calc(100vh - 200px)}
.panel-body.pad{padding:14px}
.tree-item{padding:6px 14px;font-size:.8rem;cursor:pointer;display:flex;align-items:center;gap:6px;border-bottom:1px solid rgba(48,54,61,.5)}
.tree-item:hover{background:rgba(88,166,255,.06)}
.tree-item.active{background:rgba(88,166,255,.12);color:var(--accent)}
.tree-children{padding-left:16px}
.tree-toggle{cursor:pointer;user-select:none;width:14px;font-size:.7rem;color:var(--text-dim)}
.content-view{white-space:pre-wrap;font-family:'Cascadia Code','Fira Code',monospace;font-size:.8rem;line-height:1.6;color:var(--text);padding:14px;min-height:200px}
.editor{width:100%;min-height:400px;background:var(--bg);color:var(--text);border:1px solid var(--border);border-radius:6px;padding:12px;font-family:'Cascadia Code','Fira Code',monospace;font-size:.8rem;line-height:1.5;resize:vertical}
.btn{padding:8px 16px;border-radius:6px;border:1px solid var(--border);background:var(--surface);color:var(--text);cursor:pointer;font-size:.8rem;transition:all .15s}
.btn:hover{border-color:var(--accent);color:var(--accent)}
.btn-primary{background:rgba(88,166,255,.15);border-color:var(--accent);color:var(--accent)}
.btn-primary:hover{background:rgba(88,166,255,.25)}
.btn-sm{padding:4px 10px;font-size:.75rem}
.toolbar{display:flex;gap:8px;align-items:center;margin-bottom:12px}
.status-msg{font-size:.75rem;color:var(--green);margin-left:8px}
.json-view{white-space:pre-wrap;font-family:'Cascadia Code','Fira Code',monospace;font-size:.78rem;line-height:1.5;color:var(--text);padding:14px;background:var(--bg);border-radius:6px;border:1px solid var(--border);max-height:600px;overflow-y:auto}
.json-key{color:var(--accent)}
.json-str{color:var(--green)}
.json-num{color:var(--orange)}
.json-bool{color:var(--red)}
.json-null{color:var(--text-dim)}
.detail-container{background:var(--surface);border:1px solid var(--border);border-radius:8px;margin-top:12px;overflow:hidden}
.detail-header{padding:12px 16px;background:rgba(255,255,255,.02);border-bottom:1px solid var(--border);display:flex;justify-content:space-between;align-items:center}
.detail-header h4{font-size:.9rem;color:var(--text-bright)}
.detail-close{cursor:pointer;color:var(--text-dim);font-size:1.2rem}
.detail-close:hover{color:var(--red)}
.detail-body{padding:16px;max-height:500px;overflow-y:auto}
.tab-bar{display:flex;gap:0;margin-bottom:16px;border-bottom:1px solid var(--border)}
.tab{padding:8px 16px;cursor:pointer;font-size:.8rem;color:var(--text-dim);border-bottom:2px solid transparent;transition:all .15s}
.tab:hover{color:var(--text)}
.tab.active{color:var(--accent);border-bottom-color:var(--accent)}
.tab-panel{display:none}
.tab-panel.active{display:block}

/* Chat styles */
.chat-wrap{display:flex;flex-direction:column;height:calc(100vh - 140px)}
.chat-messages{flex:1;overflow-y:auto;padding:16px;display:flex;flex-direction:column;gap:12px}
.chat-msg{max-width:75%;padding:10px 14px;border-radius:12px;font-size:.85rem;line-height:1.5;white-space:pre-wrap;word-wrap:break-word}
.chat-msg.user{align-self:flex-end;background:rgba(88,166,255,.18);border:1px solid rgba(88,166,255,.3);color:var(--text-bright)}
.chat-msg.agent{align-self:flex-start;background:var(--surface);border:1px solid var(--border);color:var(--text)}
.chat-msg.system{align-self:center;color:var(--text-dim);font-style:italic;font-size:.75rem;padding:4px 12px}
.chat-input-bar{display:flex;gap:8px;padding:12px 0 0 0;border-top:1px solid var(--border)}
.chat-input{flex:1;background:var(--bg);border:1px solid var(--border);border-radius:8px;padding:10px 14px;color:var(--text);font-size:.85rem;font-family:inherit;resize:none;min-height:44px;max-height:120px}
.chat-input:focus{outline:none;border-color:var(--accent)}
.chat-send{padding:10px 20px;border-radius:8px;border:none;background:var(--accent);color:#fff;font-weight:600;cursor:pointer;font-size:.85rem;transition:opacity .15s}
.chat-send:hover{opacity:.85}
.chat-send:disabled{opacity:.4;cursor:not-allowed}

/* Graph styles */
.graph-wrap{position:relative;height:calc(100vh - 140px);background:var(--bg);border:1px solid var(--border);border-radius:8px;overflow:hidden}
#graph-canvas{width:100%;height:100%;cursor:grab}
#graph-canvas:active{cursor:grabbing}
.graph-legend{position:absolute;top:12px;right:12px;background:rgba(22,27,34,.92);border:1px solid var(--border);border-radius:8px;padding:10px 14px;font-size:.7rem;max-height:280px;overflow-y:auto}
.graph-legend-item{display:flex;align-items:center;gap:6px;margin-bottom:4px;cursor:pointer}
.graph-legend-dot{width:10px;height:10px;border-radius:50%;flex-shrink:0}
.graph-legend-item.dimmed{opacity:.3}
.graph-controls{position:absolute;top:12px;left:12px;display:flex;gap:6px}
.graph-info{position:absolute;bottom:12px;left:12px;background:rgba(22,27,34,.92);border:1px solid var(--border);border-radius:8px;padding:10px 14px;font-size:.75rem;max-width:350px;max-height:200px;overflow-y:auto;display:none}
.graph-stats{position:absolute;bottom:12px;right:12px;font-size:.7rem;color:var(--text-dim);background:rgba(22,27,34,.8);padding:4px 10px;border-radius:4px}

@media(max-width:900px){
  #sidebar{width:60px}
  #sidebar .logo h1,.nav-item span:not(.nav-icon){display:none}
  #sidebar .logo{padding:16px 10px;justify-content:center}
  .nav-item{padding:12px 10px;justify-content:center}
  #main{margin-left:60px}
  .split{grid-template-columns:1fr}
}
</style>
</head>
<body>

<div id="sidebar">
  <div class="logo"><div class="dot"></div><h1>Xagent</h1></div>
  <div class="nav-item active" onclick="navigate('overview')"><span class="nav-icon">&#9673;</span><span>Overview</span></div>
  <div class="nav-item" onclick="navigate('chat')"><span class="nav-icon">&#9993;</span><span>Chat</span></div>
  <div class="nav-item" onclick="navigate('memory')"><span class="nav-icon">&#9881;</span><span>Memory</span></div>
  <div class="nav-item" onclick="navigate('vault')"><span class="nav-icon">&#9830;</span><span>Vault</span></div>
  <div class="nav-item" onclick="navigate('graph')"><span class="nav-icon">&#10040;</span><span>Graph</span></div>
  <div class="nav-item" onclick="navigate('epochs')"><span class="nav-icon">&#8634;</span><span>Epochs</span></div>
  <div class="nav-item" onclick="navigate('provenance')"><span class="nav-icon">&#8618;</span><span>Provenance</span></div>
  <div class="nav-item" onclick="navigate('skills')"><span class="nav-icon">&#9733;</span><span>Skills</span></div>
  <div class="nav-item" onclick="navigate('config')"><span class="nav-icon">&#9881;</span><span>Config</span></div>
  <div class="nav-item" onclick="navigate('system')"><span class="nav-icon">&#9636;</span><span>System</span></div>
</div>

<div id="main">
<header>
  <h2 id="page-title">Overview</h2>
  <div class="meta" id="header-meta">auto-refreshes every 10s</div>
</header>

<!-- OVERVIEW -->
<div class="page active" id="page-overview">
  <div class="cards">
    <div class="card"><div class="label">Uptime</div><div class="value" id="ov-uptime">--</div></div>
    <div class="card"><div class="label">Fatigue</div><div class="value" id="ov-fatigue">--</div></div>
    <div class="card"><div class="label">Messages</div><div class="value" id="ov-messages">--</div></div>
    <div class="card"><div class="label">Model</div><div class="value sm" id="ov-model">--</div></div>
    <div class="card"><div class="label">Tier</div><div class="value" id="ov-tier">--</div></div>
    <div class="card"><div class="label">LLM Calls</div><div class="value" id="ov-llm">--</div></div>
    <div class="card"><div class="label">Tool Calls</div><div class="value" id="ov-tools">--</div></div>
    <div class="card"><div class="label">Avg Latency</div><div class="value sm" id="ov-latency">--</div></div>
  </div>
  <div style="display:grid;grid-template-columns:1fr 1fr;gap:16px">
    <div class="section"><h3>Recent Epochs</h3><div class="list" id="ov-epochs"><div class="empty">Loading...</div></div></div>
    <div class="section"><h3>Recent Provenance</h3><div class="list" id="ov-prov"><div class="empty">Loading...</div></div></div>
  </div>
  <div style="display:grid;grid-template-columns:1fr 1fr;gap:16px">
    <div class="section"><h3>Skills</h3><div class="list" id="ov-skills"><div class="empty">Loading...</div></div></div>
    <div class="section"><h3>Peers</h3><div class="list" id="ov-peers"><div class="empty">Loading...</div></div></div>
  </div>
</div>

<!-- CHAT -->
<div class="page" id="page-chat">
  <div class="chat-wrap">
    <div class="chat-messages" id="chat-messages">
      <div class="chat-msg system">Send a message to interact with the agent. Session persists while this page is open.</div>
    </div>
    <div class="chat-input-bar">
      <textarea class="chat-input" id="chat-input" placeholder="Type a message..." rows="1" onkeydown="chatKeyDown(event)"></textarea>
      <button class="chat-send" id="chat-send" onclick="sendChat()">Send</button>
    </div>
  </div>
</div>

<!-- MEMORY -->
<div class="page" id="page-memory">
  <div class="tab-bar">
    <div class="tab active" onclick="memTab('longterm')">Long-Term</div>
    <div class="tab" onclick="memTab('daily')">Daily Notes</div>
    <div class="tab" onclick="memTab('consolidations')">Consolidations</div>
  </div>
  <div class="tab-panel active" id="mem-longterm">
    <div class="toolbar">
      <button class="btn btn-primary" onclick="saveMemory()">Save</button>
      <button class="btn" onclick="loadMemoryLongterm()">Reload</button>
      <span class="status-msg" id="mem-status"></span>
    </div>
    <textarea class="editor" id="mem-editor" placeholder="Loading MEMORY.md..."></textarea>
  </div>
  <div class="tab-panel" id="mem-daily">
    <div class="split">
      <div class="panel"><div class="panel-header">Daily Notes</div><div class="panel-body" id="mem-daily-list"><div class="empty">Loading...</div></div></div>
      <div class="panel"><div class="panel-header" id="mem-daily-title">Select a note</div><div class="panel-body pad"><div class="content-view" id="mem-daily-content">Click a note on the left to view its contents.</div></div></div>
    </div>
  </div>
  <div class="tab-panel" id="mem-consolidations">
    <div class="split">
      <div class="panel"><div class="panel-header">Consolidations</div><div class="panel-body" id="mem-cons-list"><div class="empty">Loading...</div></div></div>
      <div class="panel"><div class="panel-header" id="mem-cons-title">Select a consolidation</div><div class="panel-body pad"><div class="content-view" id="mem-cons-content">Click a consolidation on the left to view.</div></div></div>
    </div>
  </div>
</div>

<!-- VAULT -->
<div class="page" id="page-vault">
  <div class="split">
    <div class="panel"><div class="panel-header">Vault Tree<button class="btn btn-sm" onclick="loadVaultTree()" style="margin-left:auto">Refresh</button></div><div class="panel-body" id="vault-tree"><div class="empty">Loading...</div></div></div>
    <div class="panel"><div class="panel-header" id="vault-note-title">Select a note</div><div class="panel-body pad"><div class="content-view" id="vault-note-content">Click a file in the tree to view.</div></div></div>
  </div>
</div>

<!-- GRAPH -->
<div class="page" id="page-graph" style="padding:0">
  <div class="graph-wrap">
    <canvas id="graph-canvas"></canvas>
    <div class="graph-controls" style="display:flex;gap:6px;align-items:center;flex-wrap:wrap">
      <button class="btn btn-sm" onclick="graphReset()">Reset</button>
      <button class="btn btn-sm" onclick="graphZoom(1.3)">+</button>
      <button class="btn btn-sm" onclick="graphZoom(0.7)">-</button>
      <span style="color:var(--text-dim);font-size:.7rem;margin-left:8px">Min links</span>
      <input id="graph-min-conn" type="number" min="0" max="50" value="1" style="width:48px;background:var(--bg-tertiary);border:1px solid var(--border);color:var(--text);border-radius:4px;padding:2px 4px;font-size:.7rem" onchange="reloadGraph()">
      <span style="color:var(--text-dim);font-size:.7rem">Max nodes</span>
      <input id="graph-max-nodes" type="number" min="10" max="1000" value="150" style="width:56px;background:var(--bg-tertiary);border:1px solid var(--border);color:var(--text);border-radius:4px;padding:2px 4px;font-size:.7rem" onchange="reloadGraph()">
      <input id="graph-search" type="text" placeholder="Search nodes..." style="width:120px;background:var(--bg-tertiary);border:1px solid var(--border);color:var(--text);border-radius:4px;padding:2px 6px;font-size:.7rem" oninput="filterGraphNodes()">
    </div>
    <div class="graph-legend" id="graph-legend"></div>
    <div class="graph-info" id="graph-info"></div>
    <div class="graph-stats" id="graph-stats">Loading graph...</div>
  </div>
</div>

<!-- EPOCHS -->
<div class="page" id="page-epochs">
  <div class="section"><h3>Epoch History</h3><div class="list" id="epochs-list" style="max-height:none"><div class="empty">Loading...</div></div></div>
  <div id="epoch-detail"></div>
</div>

<!-- PROVENANCE -->
<div class="page" id="page-provenance">
  <div class="section"><h3>Provenance Log</h3><div class="list" id="prov-list" style="max-height:none"><div class="empty">Loading...</div></div></div>
  <div id="prov-detail"></div>
</div>

<!-- SKILLS -->
<div class="page" id="page-skills">
  <div class="section">
    <h3>Installed Skills</h3>
    <div class="toolbar">
      <input type="text" id="skill-search" class="chat-input" placeholder="Filter skills..." style="max-width:300px;min-height:36px;max-height:36px">
      <span id="skill-count" style="font-size:.75rem;color:var(--text-dim)"></span>
    </div>
    <div class="split">
      <div class="panel">
        <div class="panel-header">Skills <span id="skill-list-count"></span></div>
        <div class="panel-body" id="skills-list" style="max-height:calc(100vh - 260px)"><div class="empty">Loading...</div></div>
      </div>
      <div class="panel">
        <div class="panel-header" id="skill-detail-header">Select a skill</div>
        <div class="panel-body pad" id="skill-detail-body" style="max-height:calc(100vh - 260px)">
          <div class="empty">Click a skill to view its contents</div>
        </div>
      </div>
    </div>
  </div>
</div>

<!-- CONFIG -->
<div class="page" id="page-config">
  <div class="section"><h3>Agent Configuration (secrets redacted)</h3><div class="json-view" id="config-view"><div class="empty">Loading...</div></div></div>
</div>

<!-- SYSTEM -->
<div class="page" id="page-system">
  <div class="cards" id="sys-cards">
    <div class="card"><div class="label">Uptime (s)</div><div class="value" id="sys-uptime">--</div></div>
    <div class="card"><div class="label">Messages</div><div class="value" id="sys-msgs">--</div></div>
    <div class="card"><div class="label">Errors</div><div class="value" id="sys-errs">--</div></div>
    <div class="card"><div class="label">LLM Calls</div><div class="value" id="sys-llm">--</div></div>
    <div class="card"><div class="label">LLM Failures</div><div class="value" id="sys-llm-fail">--</div></div>
    <div class="card"><div class="label">Avg LLM Latency</div><div class="value sm" id="sys-lat">--</div></div>
    <div class="card"><div class="label">Tool Calls</div><div class="value" id="sys-tool">--</div></div>
  </div>
  <div class="section" style="margin-bottom:16px">
    <h3>Subsystem Watchdog</h3>
    <div id="sys-watchdog" style="display:grid;grid-template-columns:repeat(auto-fill,minmax(220px,1fr));gap:10px"><div class="empty">Loading...</div></div>
  </div>
  <div style="display:grid;grid-template-columns:1fr 1fr;gap:16px">
    <div class="section"><h3>Cron Jobs</h3><div class="list" id="sys-cron"><div class="empty">Loading...</div></div></div>
    <div class="section"><h3>A2A Peers</h3><div class="list" id="sys-peers"><div class="empty">Loading...</div></div></div>
  </div>
</div>

</div>

<script>
// ===== Navigation =====
var PAGES = ['overview','chat','memory','vault','graph','epochs','provenance','skills','config','system'];
var TITLES = {overview:'Overview',chat:'Chat',memory:'Memory System',vault:'Obsidian Vault',graph:'Knowledge Graph',epochs:'Epoch History',provenance:'Provenance Log',skills:'Skills',config:'Configuration',system:'System & Metrics'};
var currentPage = 'overview';

function navigate(page) {
  document.querySelectorAll('.page').forEach(function(p){ p.classList.remove('active'); });
  document.querySelectorAll('.nav-item').forEach(function(n){ n.classList.remove('active'); });
  var el = document.getElementById('page-' + page);
  if (el) el.classList.add('active');
  var idx = PAGES.indexOf(page);
  var navItems = document.querySelectorAll('.nav-item');
  if (idx >= 0 && navItems[idx]) navItems[idx].classList.add('active');
  document.getElementById('page-title').textContent = TITLES[page] || page;
  currentPage = page;
  loadPage(page);
  if (page === 'graph') resizeGraphCanvas();
}

function loadPage(page) {
  var loaders = {overview:loadOverview,chat:function(){},memory:loadMemoryPage,vault:loadVaultTree,graph:loadGraph,epochs:loadEpochsList,provenance:loadProvList,skills:loadSkillsGrid,config:loadConfig,system:loadSystem};
  if (loaders[page]) loaders[page]();
}

function api(url, opts) { return fetch(url, opts).then(function(r){ return r.json(); }).catch(function(){ return null; }); }
function setText(id, val) { var el = document.getElementById(id); if (el) el.textContent = val != null ? val : '--'; }
function esc(s) { if (s == null) return ''; var d = document.createElement('div'); d.appendChild(document.createTextNode(String(s))); return d.innerHTML; }
function fmtTime(t) { if (!t) return ''; try { return new Date(t).toLocaleString(); } catch(e) { return t; } }
function renderListItems(id, items, fn) { var el = document.getElementById(id); if (!el) return; if (!items || items.length===0){el.innerHTML='<div class="empty">None</div>';return;} el.innerHTML=items.map(fn).join(''); }

// ===== Overview =====
function loadOverview() {
  api('/api/state').then(function(s){if(!s)return;setText('ov-uptime',s.uptime);setText('ov-fatigue',s.fatigue);setText('ov-messages',s.messages_count!=null?s.messages_count:'--');setText('ov-model',s.model);setText('ov-tier',s.tier);});
  api('/api/metrics').then(function(m){if(!m)return;setText('ov-llm',m.llm_calls_total!=null?m.llm_calls_total:'--');setText('ov-tools',m.tool_calls_total!=null?m.tool_calls_total:'--');setText('ov-latency',m.llm_avg_latency_ms!=null?Math.round(m.llm_avg_latency_ms)+'ms':'--');});
  api('/api/epochs').then(function(items){renderListItems('ov-epochs',items,function(e){return '<div class="list-item"><span>'+esc(e.name)+'</span><span class="time">'+fmtTime(e.mod_time)+'</span></div>';});});
  api('/api/provenance').then(function(items){renderListItems('ov-prov',items,function(e){return '<div class="list-item"><span>'+esc(e.name)+'</span><span class="time">'+fmtTime(e.mod_time)+'</span></div>';});});
  api('/api/skills').then(function(items){renderListItems('ov-skills',items,function(e){var n=e.path||e.name||JSON.stringify(e);return '<div class="list-item" style="cursor:pointer" onclick="navigate(\'skills\')"><span>'+esc(n)+'</span>'+(e.description?'<span class="time">'+esc(e.description)+'</span>':'')+'</div>';});});
  api('/api/peers').then(function(items){renderListItems('ov-peers',items,function(e){return '<div class="list-item"><span>'+esc(JSON.stringify(e))+'</span></div>';});});
}

// ===== Chat =====
var chatSessionKey = 'dashboard-' + Date.now();
var chatBusy = false;

function chatKeyDown(e) { if (e.key === 'Enter' && !e.shiftKey) { e.preventDefault(); sendChat(); } }

function sendChat() {
  if (chatBusy) return;
  var input = document.getElementById('chat-input');
  var msg = input.value.trim();
  if (!msg) return;
  input.value = '';
  appendChatMsg('user', msg);
  chatBusy = true;
  document.getElementById('chat-send').disabled = true;
  appendChatMsg('system', 'Thinking...');

  fetch('/api/chat', {method:'POST', headers:{'Content-Type':'application/json'}, body:JSON.stringify({message:msg, session_key:chatSessionKey})})
    .then(function(r){return r.json();})
    .then(function(d){
      if (d.error) { removeLast('system'); appendChatMsg('agent','Error: '+d.error); chatBusy=false; document.getElementById('chat-send').disabled=false; return; }
      pollChat(d.id);
    })
    .catch(function(e){ removeLast('system'); appendChatMsg('agent','Network error'); chatBusy=false; document.getElementById('chat-send').disabled=false; });
}

function pollChat(id) {
  api('/api/chat?id='+encodeURIComponent(id)).then(function(d) {
    if (!d) { removeLast('system'); appendChatMsg('agent','Poll error'); chatBusy=false; document.getElementById('chat-send').disabled=false; return; }
    if (!d.done) { setTimeout(function(){ pollChat(id); }, 1000); return; }
    removeLast('system');
    if (d.error && d.error !== '') { appendChatMsg('agent','Error: '+d.error); }
    else { appendChatMsg('agent', d.response || '(empty response)'); }
    chatBusy = false;
    document.getElementById('chat-send').disabled = false;
  });
}

function appendChatMsg(type, text) {
  var el = document.getElementById('chat-messages');
  var div = document.createElement('div');
  div.className = 'chat-msg ' + type;
  div.textContent = text;
  el.appendChild(div);
  el.scrollTop = el.scrollHeight;
}

function removeLast(type) {
  var el = document.getElementById('chat-messages');
  var msgs = el.querySelectorAll('.chat-msg.' + type);
  if (msgs.length > 0) msgs[msgs.length - 1].remove();
}

// ===== Memory =====
function memTab(tab) {
  document.querySelectorAll('#page-memory .tab').forEach(function(t){t.classList.remove('active');});
  document.querySelectorAll('#page-memory .tab-panel').forEach(function(p){p.classList.remove('active');});
  var tabs=['longterm','daily','consolidations']; var idx=tabs.indexOf(tab);
  var tabEls=document.querySelectorAll('#page-memory .tab'); if(tabEls[idx])tabEls[idx].classList.add('active');
  var panel=document.getElementById('mem-'+tab); if(panel)panel.classList.add('active');
  if(tab==='longterm')loadMemoryLongterm(); if(tab==='daily')loadMemoryDaily(); if(tab==='consolidations')loadMemoryConsolidations();
}
function loadMemoryPage(){loadMemoryLongterm();}
function loadMemoryLongterm(){api('/api/memory/longterm').then(function(d){if(!d)return;document.getElementById('mem-editor').value=d.content||'';});}
function saveMemory(){
  fetch('/api/memory/longterm',{method:'PUT',headers:{'Content-Type':'application/json'},body:JSON.stringify({content:document.getElementById('mem-editor').value})})
    .then(function(r){return r.json();}).then(function(d){var el=document.getElementById('mem-status');el.textContent=d.status==='saved'?'Saved!':'Error';setTimeout(function(){el.textContent='';},2000);});
}
function loadMemoryDaily(){api('/api/memory/daily').then(function(items){if(!items||items.length===0){document.getElementById('mem-daily-list').innerHTML='<div class="empty">No daily notes</div>';return;}document.getElementById('mem-daily-list').innerHTML=items.map(function(e){return '<div class="tree-item" onclick="viewDailyNote(\''+esc(e.path)+'\')">'+esc(e.name)+'<span class="time" style="margin-left:auto">'+fmtTime(e.mod_time)+'</span></div>';}).join('');});}
function viewDailyNote(path){document.getElementById('mem-daily-title').textContent=path;api('/api/memory/daily?file='+encodeURIComponent(path)).then(function(d){document.getElementById('mem-daily-content').textContent=d&&d.content?d.content:'(empty)';});}
function loadMemoryConsolidations(){api('/api/memory/consolidations').then(function(items){if(!items||items.length===0){document.getElementById('mem-cons-list').innerHTML='<div class="empty">No consolidations</div>';return;}document.getElementById('mem-cons-list').innerHTML=items.map(function(e){var fp=e.type+'/'+e.name;return '<div class="tree-item" onclick="viewConsolidation(\''+esc(fp)+'\',\''+esc(e.type+': '+e.name)+'\')"><span style="color:var(--accent)">['+esc(e.type)+']</span>&nbsp;'+esc(e.name)+'<span class="time" style="margin-left:auto">'+fmtTime(e.mod_time)+'</span></div>';}).join('');});}
function viewConsolidation(path,title){document.getElementById('mem-cons-title').textContent=title;api('/api/memory/consolidations?file='+encodeURIComponent(path)).then(function(d){document.getElementById('mem-cons-content').textContent=d&&d.content?d.content:'(empty)';});}

// ===== Vault =====
function loadVaultTree(){api('/api/vault/tree').then(function(d){if(!d||d.error){document.getElementById('vault-tree').innerHTML='<div class="empty">'+(d&&d.error?esc(d.error):'Error')+'</div>';return;}document.getElementById('vault-tree').innerHTML=renderTree(d.tree||[]);});}
function renderTree(nodes){if(!nodes||nodes.length===0)return '<div class="empty">Empty</div>';var h='';for(var i=0;i<nodes.length;i++){var n=nodes[i];if(n.is_dir){var cid='vt-'+n.path.replace(/[^a-zA-Z0-9]/g,'_');h+='<div class="tree-item" onclick="toggleTree(\''+cid+'\',this)"><span class="tree-toggle">&#9654;</span>'+esc(n.name)+'/</div><div class="tree-children" id="'+cid+'" style="display:none">';if(n.children)h+=renderTree(n.children);h+='</div>';}else{h+='<div class="tree-item" onclick="viewVaultNote(\''+esc(n.path)+'\',this)">'+esc(n.name)+'</div>';}}return h;}
function toggleTree(id,el){var ch=document.getElementById(id);if(!ch)return;var s=ch.style.display!=='none';ch.style.display=s?'none':'block';var t=el.querySelector('.tree-toggle');if(t)t.innerHTML=s?'&#9654;':'&#9660;';}
function viewVaultNote(path,el){document.querySelectorAll('#vault-tree .tree-item').forEach(function(t){t.classList.remove('active');});if(el)el.classList.add('active');document.getElementById('vault-note-title').textContent=path;api('/api/vault/note?path='+encodeURIComponent(path)).then(function(d){document.getElementById('vault-note-content').textContent=d&&!d.error?d.content||'(empty)':d&&d.error?d.error:'Error';});}

// ===== Graph (Force-Directed) =====
var GN=[], GE=[], gAlpha=1, gScale=1, gPanX=0, gPanY=0;
var gCanvas, gCtx, gW, gH, gDragNode=null, gPanning=false, gMX=0, gMY=0, gHover=null, gSelected=null;
var gAnimId=null, gLoaded=false;
var gHiddenGroups={};
var GROUP_COLORS={Sessions:'#58a6ff',Daily:'#3fb950',Tools:'#d29922',Topics:'#a371f7',Dreams:'#79c0ff',Channels:'#e3b341',Models:'#f85149',World:'#56d364',Experiences:'#db61a2',MentalModels:'#bc8cff',Personality:'#f0883e',Other:'#8b949e'};

var gSearchTerm='';
function loadGraph() {
  if (gLoaded) return;
  reloadGraph();
}
function reloadGraph() {
  var mc=document.getElementById('graph-min-conn');
  var mn=document.getElementById('graph-max-nodes');
  var minConn=mc?mc.value:'1';
  var maxNodes=mn?mn.value:'150';
  var url='/api/vault/graph?min_conn='+minConn+'&max_nodes='+maxNodes;
  api(url).then(function(d) {
    if (!d || d.error) { document.getElementById('graph-stats').textContent = d&&d.error?d.error:'Error'; return; }
    // SWE100821: Assign initial positions by group for clustering
    var groupAngles={};var gi=0;
    var groups={};(d.nodes||[]).forEach(function(n){groups[n.group]=true;});
    var gKeys=Object.keys(groups);
    gKeys.forEach(function(g,i){groupAngles[g]=2*Math.PI*i/gKeys.length;});
    GN = (d.nodes||[]).map(function(n){ var a=groupAngles[n.group]||0; var spread=150+Math.random()*100; return {id:n.id,name:n.name,group:n.group,size:n.size,x:Math.cos(a)*spread+(Math.random()-0.5)*80,y:Math.sin(a)*spread+(Math.random()-0.5)*80,vx:0,vy:0}; });
    GE = (d.edges||[]).map(function(e){ return {source:findNode(e.source),target:findNode(e.target)}; }).filter(function(e){return e.source!=null&&e.target!=null;});
    var statsText=GN.length+' nodes, '+GE.length+' edges';
    if(d.total_files)statsText+=' (of '+d.total_files+' files)';
    document.getElementById('graph-stats').textContent = statsText;
    buildLegend();
    gLoaded = true;
    gAlpha = 1;
    stepGraph();
  });
}
function filterGraphNodes(){
  var el=document.getElementById('graph-search');
  gSearchTerm=el?el.value.toLowerCase():'';
  drawGraph();
}

function findNode(id){for(var i=0;i<GN.length;i++){if(GN[i].id===id)return i;}return null;}

function buildLegend(){
  var groups={};
  for(var i=0;i<GN.length;i++){var g=GN[i].group;if(!groups[g])groups[g]=0;groups[g]++;}
  var html='';
  var keys=Object.keys(groups).sort();
  for(var k=0;k<keys.length;k++){
    var g=keys[k]; var c=GROUP_COLORS[g]||'#8b949e';
    var dimmed=gHiddenGroups[g]?' dimmed':'';
    html+='<div class="graph-legend-item'+dimmed+'" onclick="toggleGroup(\''+esc(g)+'\')">'+'<div class="graph-legend-dot" style="background:'+c+'"></div>'+esc(g)+' ('+groups[g]+')</div>';
  }
  document.getElementById('graph-legend').innerHTML=html;
}

function toggleGroup(g){
  gHiddenGroups[g]=!gHiddenGroups[g];
  buildLegend();
  drawGraph();
}

function resizeGraphCanvas(){
  gCanvas=document.getElementById('graph-canvas');
  if(!gCanvas)return;
  gCtx=gCanvas.getContext('2d');
  var wrap=gCanvas.parentElement;
  gW=wrap.clientWidth; gH=wrap.clientHeight;
  gCanvas.width=gW; gCanvas.height=gH;
  if(gLoaded)drawGraph();
}

function stepGraph(){
  if(gAlpha<0.001){drawGraph();return;}
  gAlpha*=0.985;
  var n=GN.length;
  // SWE100821: Compute group centroids for clustering force
  var groupCX={},groupCY={},groupCnt={};
  for(var i=0;i<n;i++){
    if(gHiddenGroups[GN[i].group])continue;
    var g=GN[i].group;
    if(!groupCnt[g]){groupCX[g]=0;groupCY[g]=0;groupCnt[g]=0;}
    groupCX[g]+=GN[i].x;groupCY[g]+=GN[i].y;groupCnt[g]++;
  }
  for(var g in groupCnt){groupCX[g]/=groupCnt[g];groupCY[g]/=groupCnt[g];}
  // Repulsion (all pairs) — stronger for cross-group
  for(var i=0;i<n;i++){
    if(gHiddenGroups[GN[i].group])continue;
    for(var j=i+1;j<n;j++){
      if(gHiddenGroups[GN[j].group])continue;
      var dx=GN[j].x-GN[i].x, dy=GN[j].y-GN[i].y;
      var dist=Math.sqrt(dx*dx+dy*dy)||1;
      var repStr=GN[i].group===GN[j].group?400:800;
      var f=-repStr/(dist*dist)*gAlpha;
      var fx=f*dx/dist, fy=f*dy/dist;
      GN[i].vx+=fx; GN[i].vy+=fy;
      GN[j].vx-=fx; GN[j].vy-=fy;
    }
  }
  // Attraction (edges) — tighter for same-group
  for(var e=0;e<GE.length;e++){
    var si=GE[e].source, ti=GE[e].target;
    if(si==null||ti==null)continue;
    var s=GN[si], t=GN[ti];
    if(gHiddenGroups[s.group]||gHiddenGroups[t.group])continue;
    var dx=t.x-s.x, dy=t.y-s.y;
    var dist=Math.sqrt(dx*dx+dy*dy)||1;
    var idealLen=s.group===t.group?50:120;
    var f=0.025*(dist-idealLen)*gAlpha;
    var fx=f*dx/dist, fy=f*dy/dist;
    s.vx+=fx; s.vy+=fy;
    t.vx-=fx; t.vy-=fy;
  }
  // SWE100821: Group clustering — gently pull nodes toward their group centroid
  for(var i=0;i<n;i++){
    if(gHiddenGroups[GN[i].group]||gDragNode===i)continue;
    var g=GN[i].group;
    if(groupCnt[g]>1){
      var cx=groupCX[g],cy=groupCY[g];
      GN[i].vx+=(cx-GN[i].x)*0.005*gAlpha;
      GN[i].vy+=(cy-GN[i].y)*0.005*gAlpha;
    }
  }
  // Centering + damping + integrate
  for(var i=0;i<n;i++){
    if(gHiddenGroups[GN[i].group])continue;
    if(gDragNode===i)continue;
    GN[i].vx-=GN[i].x*0.008*gAlpha;
    GN[i].vy-=GN[i].y*0.008*gAlpha;
    GN[i].vx*=0.85; GN[i].vy*=0.85;
    GN[i].x+=GN[i].vx; GN[i].y+=GN[i].vy;
  }
  drawGraph();
  gAnimId=requestAnimationFrame(stepGraph);
}

function drawGraph(){
  if(!gCtx)return;
  gCtx.clearRect(0,0,gW,gH);
  gCtx.save();
  gCtx.translate(gW/2, gH/2);
  gCtx.scale(gScale, gScale);
  gCtx.translate(gPanX, gPanY);

  // SWE100821: Compute max connections for relative sizing
  var maxConn=1;
  for(var i=0;i<GN.length;i++){if(GN[i].size>maxConn)maxConn=GN[i].size;}

  // SWE100821: Build search match set
  var hasSearch=gSearchTerm.length>0;
  var searchMatch={};
  if(hasSearch){for(var i=0;i<GN.length;i++){if(GN[i].name.toLowerCase().indexOf(gSearchTerm)>=0)searchMatch[i]=true;}}

  // Edges — opacity scales with whether endpoints are selected/searched
  for(var e=0;e<GE.length;e++){
    var si=GE[e].source, ti=GE[e].target;
    if(si==null||ti==null)continue;
    var s=GN[si], t=GN[ti];
    if(gHiddenGroups[s.group]||gHiddenGroups[t.group])continue;
    gCtx.beginPath();
    gCtx.moveTo(s.x,s.y);
    gCtx.lineTo(t.x,t.y);
    if(gSelected!=null&&(si===gSelected||ti===gSelected)){
      gCtx.strokeStyle='rgba(88,166,255,0.6)';
      gCtx.lineWidth=1.5;
    } else if(hasSearch&&(searchMatch[si]||searchMatch[ti])){
      gCtx.strokeStyle='rgba(88,166,255,0.3)';
      gCtx.lineWidth=1;
    } else {
      var edgeAlpha=hasSearch?0.08:(gSelected!=null?0.12:0.25);
      gCtx.strokeStyle='rgba(48,54,61,'+edgeAlpha+')';
      gCtx.lineWidth=0.5;
    }
    gCtx.stroke();
  }

  // Nodes — size by relative connection count, dim non-matches during search
  for(var i=0;i<GN.length;i++){
    var nd=GN[i];
    if(gHiddenGroups[nd.group])continue;
    var r=3+Math.sqrt(nd.size/maxConn)*12;
    var col=GROUP_COLORS[nd.group]||'#8b949e';
    var dimmed=hasSearch&&!searchMatch[i];
    gCtx.beginPath();
    gCtx.arc(nd.x, nd.y, r, 0, Math.PI*2);
    gCtx.globalAlpha=dimmed?0.15:1;
    gCtx.fillStyle=col;
    if(i===gSelected){gCtx.fillStyle='#fff';gCtx.lineWidth=2;gCtx.strokeStyle=col;gCtx.stroke();}
    else if(hasSearch&&searchMatch[i]){gCtx.lineWidth=2;gCtx.strokeStyle='#fff';gCtx.stroke();}
    else if(i===gHover){gCtx.lineWidth=1.5;gCtx.strokeStyle='#fff';gCtx.stroke();}
    gCtx.fill();
    gCtx.globalAlpha=1;
  }

  // Labels — show for hovered, selected, search matches with high connections, and top nodes when zoomed
  var labelNodes=[];
  if(gHover!=null)labelNodes.push(gHover);
  if(gSelected!=null&&gSelected!==gHover)labelNodes.push(gSelected);
  if(hasSearch){for(var k in searchMatch){labelNodes.push(parseInt(k));}}
  else if(gScale>1.5){for(var i=0;i<GN.length;i++){if(!gHiddenGroups[GN[i].group]&&GN[i].size>=maxConn*0.3)labelNodes.push(i);}}
  for(var li=0;li<labelNodes.length;li++){
    var idx=labelNodes[li];
    if(idx==null||gHiddenGroups[GN[idx].group])continue;
    var nd=GN[idx];
    var lr=3+Math.sqrt(nd.size/maxConn)*12;
    gCtx.font=(idx===gSelected||idx===gHover?'bold ':'')+' 11px sans-serif';
    gCtx.fillStyle='#f0f6fc';
    gCtx.textAlign='center';
    gCtx.fillText(nd.name, nd.x, nd.y - lr - 5);
  }

  gCtx.restore();
}

function graphScreenToWorld(sx,sy){return {x:(sx-gW/2)/gScale-gPanX, y:(sy-gH/2)/gScale-gPanY};}
function graphNodeAt(wx,wy){
  for(var i=GN.length-1;i>=0;i--){
    if(gHiddenGroups[GN[i].group])continue;
    var dx=GN[i].x-wx, dy=GN[i].y-wy;
    var r=Math.min(3+GN[i].size*1.5,16)+4;
    if(dx*dx+dy*dy<r*r)return i;
  }
  return null;
}

function graphReset(){gScale=1;gPanX=0;gPanY=0;gAlpha=1;gSelected=null;gHiddenGroups={};gSearchTerm='';gLoaded=false;var s=document.getElementById('graph-search');if(s)s.value='';buildLegend();reloadGraph();}
function graphZoom(factor){gScale*=factor;if(gScale<0.1)gScale=0.1;if(gScale>10)gScale=10;drawGraph();}

// Graph mouse events
function initGraphEvents(){
  gCanvas=document.getElementById('graph-canvas');
  if(!gCanvas)return;
  gCanvas.addEventListener('mousedown',function(e){
    var r=gCanvas.getBoundingClientRect();
    var sx=e.clientX-r.left, sy=e.clientY-r.top;
    var w=graphScreenToWorld(sx,sy);
    var hit=graphNodeAt(w.x,w.y);
    if(hit!=null){gDragNode=hit;gAlpha=Math.max(gAlpha,0.1);stepGraph();}
    else{gPanning=true;}
    gMX=e.clientX;gMY=e.clientY;
  });
  gCanvas.addEventListener('mousemove',function(e){
    var r=gCanvas.getBoundingClientRect();
    var sx=e.clientX-r.left, sy=e.clientY-r.top;
    if(gDragNode!=null){
      var dx=(e.clientX-gMX)/gScale, dy=(e.clientY-gMY)/gScale;
      GN[gDragNode].x+=dx; GN[gDragNode].y+=dy;
      GN[gDragNode].vx=0; GN[gDragNode].vy=0;
      gMX=e.clientX;gMY=e.clientY;
      drawGraph();
    } else if(gPanning){
      gPanX+=(e.clientX-gMX)/gScale; gPanY+=(e.clientY-gMY)/gScale;
      gMX=e.clientX;gMY=e.clientY;
      drawGraph();
    } else {
      var w=graphScreenToWorld(sx,sy);
      var prev=gHover;
      gHover=graphNodeAt(w.x,w.y);
      if(gHover!==prev)drawGraph();
      gCanvas.style.cursor=gHover!=null?'pointer':'grab';
    }
  });
  gCanvas.addEventListener('mouseup',function(e){
    if(gDragNode!=null){
      var r=gCanvas.getBoundingClientRect();
      var sx=e.clientX-r.left, sy=e.clientY-r.top;
      var w=graphScreenToWorld(sx,sy);
      var hit=graphNodeAt(w.x,w.y);
      if(hit===gDragNode){
        gSelected=hit;
        showGraphNodeInfo(GN[hit]);
      }
    }
    gDragNode=null;gPanning=false;
  });
  gCanvas.addEventListener('wheel',function(e){
    e.preventDefault();
    var factor=e.deltaY<0?1.1:0.9;
    graphZoom(factor);
  },{passive:false});
  gCanvas.addEventListener('dblclick',function(e){
    if(gSelected!=null&&GN[gSelected]){
      navigate('vault');
      setTimeout(function(){viewVaultNote(GN[gSelected].id,null);},200);
    }
  });
}

function showGraphNodeInfo(nd){
  var info=document.getElementById('graph-info');
  var conns=0;
  var linked=[];
  for(var e=0;e<GE.length;e++){
    var si=GE[e].source,ti=GE[e].target;
    if(si!=null&&GN[si].id===nd.id&&ti!=null){conns++;linked.push(GN[ti].name);}
    if(ti!=null&&GN[ti].id===nd.id&&si!=null){conns++;linked.push(GN[si].name);}
  }
  info.style.display='block';
  info.innerHTML='<strong style="color:var(--text-bright)">'+esc(nd.name)+'</strong><br>'+
    '<span style="color:'+(GROUP_COLORS[nd.group]||'#8b949e')+'">'+esc(nd.group)+'</span> &middot; '+conns+' connections<br>'+
    '<span style="color:var(--text-dim);font-size:.7rem">'+esc(nd.id)+'</span><br>'+
    (linked.length>0?'<div style="margin-top:6px;color:var(--text-dim)">Linked: '+linked.slice(0,10).map(function(l){return esc(l);}).join(', ')+(linked.length>10?' ...':'')+'</div>':'')+
    '<div style="margin-top:6px;color:var(--accent);cursor:pointer" onclick="navigate(\'vault\');setTimeout(function(){viewVaultNote(\''+esc(nd.id)+'\',null);},200);">Open in Vault &rarr;</div>';
}

// ===== Epochs =====
function loadEpochsList(){api('/api/epochs').then(function(items){renderListItems('epochs-list',items,function(e){return '<div class="list-item" onclick="viewEpoch(\''+esc(e.name)+'\')"><span>'+esc(e.name)+'</span><span class="time">'+fmtTime(e.mod_time)+'</span></div>';});});}
function viewEpoch(name){api('/api/epoch/detail?name='+encodeURIComponent(name)).then(function(d){if(!d)return;document.getElementById('epoch-detail').innerHTML='<div class="detail-container"><div class="detail-header"><h4>'+esc(name)+'</h4><span class="detail-close" onclick="this.closest(\'.detail-container\').remove()">&#10005;</span></div><div class="detail-body"><div class="json-view">'+syntaxHL(d)+'</div></div></div>';});}

// ===== Provenance =====
function loadProvList(){api('/api/provenance').then(function(items){renderListItems('prov-list',items,function(e){return '<div class="list-item" onclick="viewProv(\''+esc(e.name)+'\')"><span>'+esc(e.name)+'</span><span class="time">'+fmtTime(e.mod_time)+'</span></div>';});});}
function viewProv(name){api('/api/provenance/detail?name='+encodeURIComponent(name)).then(function(recs){if(!recs)return;var h='<div class="detail-container"><div class="detail-header"><h4>'+esc(name)+' ('+recs.length+' records)</h4><span class="detail-close" onclick="this.closest(\'.detail-container\').remove()">&#10005;</span></div><div class="detail-body">';for(var i=0;i<recs.length;i++){h+='<div style="margin-bottom:12px;padding-bottom:12px;border-bottom:1px solid var(--border)"><div class="json-view" style="max-height:200px">'+syntaxHL(recs[i])+'</div></div>';}h+='</div></div>';document.getElementById('prov-detail').innerHTML=h;});}

// ===== Skills (SWE100821: split-panel with search, detail view) =====
var _skillsData = [];
function loadSkillsGrid(){
  api('/api/skills').then(function(items){
    _skillsData = items || [];
    renderSkillsList(_skillsData);
    document.getElementById('skill-count').textContent = _skillsData.length + ' skills';
  });
  var searchEl = document.getElementById('skill-search');
  if (searchEl && !searchEl._bound) {
    searchEl._bound = true;
    searchEl.addEventListener('input', function(){
      var q = this.value.toLowerCase();
      var filtered = _skillsData.filter(function(s){
        return (s.path||'').toLowerCase().indexOf(q) >= 0 ||
               (s.description||'').toLowerCase().indexOf(q) >= 0 ||
               (s.author||'').toLowerCase().indexOf(q) >= 0 ||
               (s.name||'').toLowerCase().indexOf(q) >= 0;
      });
      renderSkillsList(filtered);
    });
  }
}
function renderSkillsList(items){
  var el = document.getElementById('skills-list');
  document.getElementById('skill-list-count').textContent = '(' + items.length + ')';
  if(!items||items.length===0){el.innerHTML='<div class="empty">No skills found</div>';return;}
  var grouped = {};
  items.forEach(function(s){
    var a = s.author || 'unknown';
    if(!grouped[a]) grouped[a] = [];
    grouped[a].push(s);
  });
  var authors = Object.keys(grouped).sort();
  var h = '';
  authors.forEach(function(author){
    h += '<div style="padding:6px 14px;font-size:.7rem;color:var(--text-dim);background:rgba(255,255,255,.02);border-bottom:1px solid var(--border);text-transform:uppercase;letter-spacing:.05em">' + esc(author) + ' (' + grouped[author].length + ')</div>';
    grouped[author].forEach(function(s){
      h += '<div class="list-item skill-item" data-path="' + esc(s.path) + '" onclick="viewSkill(\'' + esc(s.path) + '\')">';
      h += '<div style="display:flex;flex-direction:column;gap:2px"><span style="color:var(--text-bright)">' + esc(s.name) + '</span>';
      if(s.description) h += '<span style="font-size:.7rem;color:var(--text-dim)">' + esc(s.description) + '</span>';
      h += '</div></div>';
    });
  });
  el.innerHTML = h;
}
function viewSkill(path){
  document.querySelectorAll('.skill-item').forEach(function(el){
    el.classList.toggle('active', el.getAttribute('data-path') === path);
  });
  document.getElementById('skill-detail-header').textContent = path;
  document.getElementById('skill-detail-body').innerHTML = '<div class="empty">Loading...</div>';
  api('/api/skill/detail?path=' + encodeURIComponent(path)).then(function(d){
    if(!d || d.error){
      document.getElementById('skill-detail-body').innerHTML = '<div class="empty">' + esc(d && d.error || 'Not found') + '</div>';
      return;
    }
    var content = d.content || '';
    var body = content;
    if(body.indexOf('---') === 0){
      var end = body.indexOf('---', 3);
      if(end > 0) body = body.substring(end + 3).trim();
    }
    var html = '<div style="margin-bottom:12px;display:flex;gap:8px;align-items:center">';
    html += '<span style="font-size:.7rem;color:var(--text-dim)">Modified: ' + fmtTime(d.mod_time) + '</span>';
    html += '</div>';
    html += '<div class="content-view" style="white-space:pre-wrap;font-size:.8rem;line-height:1.6">' + renderSkillMD(body) + '</div>';
    document.getElementById('skill-detail-body').innerHTML = html;
  });
}
var BT3 = String.fromCharCode(96,96,96);
var BT1 = String.fromCharCode(96);
function renderSkillMD(md){
  var lines = md.split('\n');
  var out = '';
  var inCode = false;
  for(var i=0;i<lines.length;i++){
    var L = lines[i];
    if(L.trim().indexOf(BT3) === 0){
      if(inCode){out += '</code></pre>'; inCode=false;}
      else{out += '<pre style="background:var(--bg);border:1px solid var(--border);border-radius:6px;padding:12px;overflow-x:auto;margin:8px 0"><code>'; inCode=true;}
      continue;
    }
    if(inCode){out += esc(L) + '\n'; continue;}
    if(L.match(/^### /)){out += '<h4 style="color:var(--accent);margin:12px 0 4px;font-size:.85rem">' + esc(L.substring(4)) + '</h4>'; continue;}
    if(L.match(/^## /)){out += '<h3 style="color:var(--text-bright);margin:14px 0 6px;font-size:.95rem">' + esc(L.substring(3)) + '</h3>'; continue;}
    if(L.match(/^# /)){out += '<h2 style="color:var(--text-bright);margin:16px 0 8px;font-size:1.1rem">' + esc(L.substring(2)) + '</h2>'; continue;}
    if(L.match(/^- /)){out += '<div style="padding-left:16px;margin:2px 0"><span style="color:var(--accent);margin-right:6px">&#8226;</span>' + escInline(L.substring(2)) + '</div>'; continue;}
    if(L.trim() === ''){out += '<br>'; continue;}
    out += '<p style="margin:4px 0">' + escInline(L) + '</p>';
  }
  if(inCode) out += '</code></pre>';
  return out;
}
function escInline(s){
  s = esc(s);
  var codeRe = new RegExp(BT1 + '([^' + BT1 + ']+)' + BT1, 'g');
  s = s.replace(codeRe, '<code style="background:var(--bg);padding:1px 5px;border-radius:3px;font-size:.8em;color:var(--cyan)">$1</code>');
  s = s.replace(/\*\*([^*]+)\*\*/g, '<strong style="color:var(--text-bright)">$1</strong>');
  return s;
}

// ===== Config =====
function loadConfig(){api('/api/config').then(function(d){if(!d){document.getElementById('config-view').textContent='Error';return;}document.getElementById('config-view').innerHTML=syntaxHL(d);});}

// ===== System =====
function loadSystem(){
  api('/api/metrics').then(function(m){if(!m)return;setText('sys-uptime',m.uptime_seconds!=null?m.uptime_seconds:'--');setText('sys-msgs',m.messages_total!=null?m.messages_total:'--');setText('sys-errs',m.messages_errored!=null?m.messages_errored:'--');setText('sys-llm',m.llm_calls_total!=null?m.llm_calls_total:'--');setText('sys-llm-fail',m.llm_calls_failed!=null?m.llm_calls_failed:'--');setText('sys-lat',m.llm_avg_latency_ms!=null?Math.round(m.llm_avg_latency_ms)+'ms':'--');setText('sys-tool',m.tool_calls_total!=null?m.tool_calls_total:'--');});
  api('/api/watchdog').then(function(d){var el=document.getElementById('sys-watchdog');if(!d||!d.subsystems||d.subsystems.length===0){el.innerHTML='<div class="empty">No subsystems registered</div>';return;}el.innerHTML=d.subsystems.map(function(s){var col=s.status==='up'?'#4ade80':s.status==='down'?'#f87171':'#facc15';var icon=s.status==='up'?'&#9679;':s.status==='down'?'&#9888;':'&#63;';var since=s.since?new Date(s.since).toLocaleTimeString():'--';var info=s.status==='down'&&s.message?'<div style="font-size:11px;color:#f87171;margin-top:4px">'+esc(s.message)+'</div>':'';var restarts=s.restarts>0?'<div style="font-size:11px;color:#facc15;margin-top:2px">Restarts: '+s.restarts+'</div>':'';return '<div style="background:#1e293b;border-left:3px solid '+col+';border-radius:6px;padding:12px"><div style="display:flex;align-items:center;gap:6px"><span style="color:'+col+';font-size:16px">'+icon+'</span><span style="font-weight:600;font-size:14px">'+esc(s.name)+'</span></div><div style="font-size:12px;color:#94a3b8;margin-top:4px">Since '+since+'</div>'+info+restarts+'</div>';}).join('');});
  api('/api/cron/jobs').then(function(items){if(!items||!Array.isArray(items)||items.length===0){document.getElementById('sys-cron').innerHTML='<div class="empty">No cron jobs</div>';return;}document.getElementById('sys-cron').innerHTML=items.map(function(j){var n=j.name||j.id||JSON.stringify(j);var s=j.schedule||j.cron||'';var en=j.enabled!==false?'active':'disabled';return '<div class="list-item"><span>'+esc(n)+'</span><span class="time">'+esc(s)+' ['+en+']</span></div>';}).join('');});
  api('/api/peers').then(function(items){renderListItems('sys-peers',items,function(e){return '<div class="list-item"><span>'+esc(JSON.stringify(e))+'</span></div>';});});
}

// ===== JSON Syntax Highlighting =====
function syntaxHL(obj){
  var j=JSON.stringify(obj,null,2);if(!j)return '';j=esc(j);
  j=j.replace(/"([^"]+)":/g,'<span class="json-key">"$1"</span>:');
  j=j.replace(/: "([^"]*)"/g,': <span class="json-str">"$1"</span>');
  j=j.replace(/: (\d+\.?\d*)/g,': <span class="json-num">$1</span>');
  j=j.replace(/: (true|false)/g,': <span class="json-bool">$1</span>');
  j=j.replace(/: (null)/g,': <span class="json-null">$1</span>');
  return j;
}

// ===== Init =====
loadOverview();
initGraphEvents();
window.addEventListener('resize', function(){ if(currentPage==='graph') resizeGraphCanvas(); });
setInterval(function(){ if(currentPage==='overview')loadOverview(); if(currentPage==='system')loadSystem(); },10000);
</script>
</body>
</html>` + ""
