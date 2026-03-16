// SWE100821: Tests for the interactive cognitive dashboard — verifies HTML,
// state JSON, skills, memory, vault, epoch detail, provenance detail, and config endpoints.
package dashboard

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func setupDashboard(t *testing.T) (*Dashboard, *http.ServeMux) {
	t.Helper()
	dir := t.TempDir()
	d := NewDashboard(dir)
	mux := http.NewServeMux()
	d.SetupRoutes(mux)
	return d, mux
}

func get(mux *http.ServeMux, path string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, path, nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
}

// SWE100821: TestDashboard_HTMLEndpoint — GET /dashboard returns 200 + HTML
func TestDashboard_HTMLEndpoint(t *testing.T) {
	_, mux := setupDashboard(t)
	rec := get(mux, "/dashboard")

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rec.Code)
	}
	ct := rec.Header().Get("Content-Type")
	if !strings.Contains(ct, "text/html") {
		t.Errorf("Content-Type = %q, want text/html", ct)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "Xagent") {
		t.Error("expected 'Xagent' in dashboard HTML")
	}
	if !strings.Contains(body, "page-memory") {
		t.Error("expected memory page in interactive dashboard")
	}
	if !strings.Contains(body, "page-vault") {
		t.Error("expected vault page in interactive dashboard")
	}
}

// SWE100821: TestDashboard_StateEndpoint — GET /api/state returns 200 + JSON
func TestDashboard_StateEndpoint(t *testing.T) {
	_, mux := setupDashboard(t)
	rec := get(mux, "/api/state")

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rec.Code)
	}

	var state map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&state); err != nil {
		t.Fatalf("decoding JSON: %v", err)
	}
	for _, key := range []string{"uptime", "fatigue", "messages_count", "model", "tier"} {
		if _, ok := state[key]; !ok {
			t.Errorf("missing key %q in state response", key)
		}
	}
}

// SWE100821: TestDashboard_SkillsEndpoint — create temp skills dir, verify response
func TestDashboard_SkillsEndpoint(t *testing.T) {
	dir := t.TempDir()
	skillsDir := filepath.Join(dir, "skills")
	os.MkdirAll(skillsDir, 0755)
	os.MkdirAll(filepath.Join(skillsDir, "git-ops"), 0755)
	os.WriteFile(filepath.Join(skillsDir, "deploy.yaml"), []byte("name: deploy"), 0644)

	d := NewDashboard(dir)
	mux := http.NewServeMux()
	d.SetupRoutes(mux)

	rec := get(mux, "/api/skills")
	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rec.Code)
	}

	var items []map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&items); err != nil {
		t.Fatalf("decoding JSON: %v", err)
	}
	if len(items) < 2 {
		t.Errorf("expected at least 2 skills, got %d", len(items))
	}
}

// SWE100821: TestDashboard_MemoryLongterm — GET/PUT cycle for MEMORY.md
func TestDashboard_MemoryLongterm(t *testing.T) {
	dir := t.TempDir()
	memDir := filepath.Join(dir, "memory")
	os.MkdirAll(memDir, 0755)
	os.WriteFile(filepath.Join(memDir, "MEMORY.md"), []byte("# Test Memory"), 0644)

	d := NewDashboard(dir)
	mux := http.NewServeMux()
	d.SetupRoutes(mux)

	// GET
	rec := get(mux, "/api/memory/longterm")
	if rec.Code != http.StatusOK {
		t.Fatalf("GET status = %d", rec.Code)
	}
	var resp map[string]string
	json.NewDecoder(rec.Body).Decode(&resp)
	if resp["content"] != "# Test Memory" {
		t.Errorf("content = %q, want '# Test Memory'", resp["content"])
	}

	// PUT
	body := strings.NewReader(`{"content":"# Updated"}`)
	req := httptest.NewRequest(http.MethodPut, "/api/memory/longterm", body)
	req.Header.Set("Content-Type", "application/json")
	rec2 := httptest.NewRecorder()
	mux.ServeHTTP(rec2, req)
	if rec2.Code != http.StatusOK {
		t.Fatalf("PUT status = %d", rec2.Code)
	}

	// Verify write
	data, _ := os.ReadFile(filepath.Join(memDir, "MEMORY.md"))
	if string(data) != "# Updated" {
		t.Errorf("file content = %q, want '# Updated'", string(data))
	}
}

// SWE100821: TestDashboard_VaultTree — vault tree returns structure
func TestDashboard_VaultTree(t *testing.T) {
	vaultDir := t.TempDir()
	os.MkdirAll(filepath.Join(vaultDir, "Sessions"), 0755)
	os.WriteFile(filepath.Join(vaultDir, "Sessions", "test.md"), []byte("# Note"), 0644)

	d := NewDashboard(t.TempDir())
	d.SetVaultPath(vaultDir)
	mux := http.NewServeMux()
	d.SetupRoutes(mux)

	rec := get(mux, "/api/vault/tree")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}

	var result map[string]interface{}
	json.NewDecoder(rec.Body).Decode(&result)
	tree, ok := result["tree"].([]interface{})
	if !ok || len(tree) == 0 {
		t.Error("expected non-empty vault tree")
	}
}

// SWE100821: TestDashboard_EpochDetail — returns epoch JSON content
func TestDashboard_EpochDetail(t *testing.T) {
	dir := t.TempDir()
	epochDir := filepath.Join(dir, "epochs")
	os.MkdirAll(epochDir, 0755)
	os.WriteFile(filepath.Join(epochDir, "2025-01-01.json"), []byte(`{"boot_time":"2025-01-01T00:00:00Z"}`), 0644)

	d := NewDashboard(dir)
	mux := http.NewServeMux()
	d.SetupRoutes(mux)

	rec := get(mux, "/api/epoch/detail?name=2025-01-01.json")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}

	var result map[string]interface{}
	json.NewDecoder(rec.Body).Decode(&result)
	if result["boot_time"] != "2025-01-01T00:00:00Z" {
		t.Errorf("unexpected epoch content: %v", result)
	}
}

// SWE100821: TestDashboard_Metrics — returns metrics JSON
func TestDashboard_Metrics(t *testing.T) {
	_, mux := setupDashboard(t)
	rec := get(mux, "/api/metrics")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	var result map[string]interface{}
	json.NewDecoder(rec.Body).Decode(&result)
	if _, ok := result["uptime_seconds"]; !ok {
		t.Error("expected uptime_seconds in metrics")
	}
}

// SWE100821: TestDashboard_PathTraversal — ensure path traversal is blocked
func TestDashboard_PathTraversal(t *testing.T) {
	_, mux := setupDashboard(t)
	rec := get(mux, "/api/memory/daily?file=../../etc/passwd")
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for path traversal, got %d", rec.Code)
	}
}

// SWE100821: TestDashboard_ChatEndpoint — POST sends a message, GET polls for response
func TestDashboard_ChatEndpoint(t *testing.T) {
	d, mux := setupDashboard(t)
	d.SetChatHandler(func(ctx context.Context, message, sessionKey string) (string, error) {
		return "echo: " + message, nil
	})

	// POST a message
	body := strings.NewReader(`{"message":"hello","session_key":"test-session"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/chat", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("POST /api/chat status = %d", rec.Code)
	}

	var postResp map[string]string
	json.NewDecoder(rec.Body).Decode(&postResp)
	id := postResp["id"]
	if id == "" {
		t.Fatal("expected non-empty id from POST /api/chat")
	}

	// Poll until done (max 3s)
	for i := 0; i < 30; i++ {
		pollRec := get(mux, "/api/chat?id="+id)
		var pollResp map[string]interface{}
		json.NewDecoder(pollRec.Body).Decode(&pollResp)
		if done, ok := pollResp["done"].(bool); ok && done {
			resp, _ := pollResp["response"].(string)
			if resp != "echo: hello" {
				t.Errorf("response = %q, want 'echo: hello'", resp)
			}
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatal("chat response never completed")
}

// SWE100821: TestDashboard_ChatNoHandler — returns error when no handler is set
func TestDashboard_ChatNoHandler(t *testing.T) {
	_, mux := setupDashboard(t)
	body := strings.NewReader(`{"message":"hello"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/chat", body)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	var resp map[string]string
	json.NewDecoder(rec.Body).Decode(&resp)
	if resp["error"] != "chat not available" {
		t.Errorf("expected 'chat not available', got %q", resp["error"])
	}
}

// SWE100821: TestDashboard_VaultGraph — returns nodes and edges from vault wikilinks
func TestDashboard_VaultGraph(t *testing.T) {
	vaultDir := t.TempDir()
	os.MkdirAll(filepath.Join(vaultDir, "Sessions"), 0755)
	os.MkdirAll(filepath.Join(vaultDir, "Daily"), 0755)

	// Session note that links to a daily note
	os.WriteFile(filepath.Join(vaultDir, "Sessions", "Session 2025-01-01.md"),
		[]byte("# Session\n[[2025-01-01]]\n[[docker]]"), 0644)
	os.WriteFile(filepath.Join(vaultDir, "Daily", "2025-01-01.md"),
		[]byte("# Daily\n[[Session 2025-01-01]]"), 0644)

	d := NewDashboard(t.TempDir())
	d.SetVaultPath(vaultDir)
	mux := http.NewServeMux()
	d.SetupRoutes(mux)

	rec := get(mux, "/api/vault/graph")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}

	var result map[string]interface{}
	json.NewDecoder(rec.Body).Decode(&result)

	nodes, _ := result["nodes"].([]interface{})
	edges, _ := result["edges"].([]interface{})

	if len(nodes) < 2 {
		t.Errorf("expected at least 2 nodes, got %d", len(nodes))
	}
	if len(edges) < 1 {
		t.Errorf("expected at least 1 edge, got %d", len(edges))
	}
}
