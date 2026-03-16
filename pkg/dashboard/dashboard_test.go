// SWE100821: Tests for the cognitive dashboard — verifies HTML, state JSON,
// and skills list endpoints via httptest.
package dashboard

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// SWE100821: TestDashboard_HTMLEndpoint — GET /dashboard returns 200 + HTML
func TestDashboard_HTMLEndpoint(t *testing.T) {
	d := NewDashboard(t.TempDir())
	mux := http.NewServeMux()
	d.SetupRoutes(mux)

	req := httptest.NewRequest(http.MethodGet, "/dashboard", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

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
}

// SWE100821: TestDashboard_StateEndpoint — GET /api/state returns 200 + JSON
func TestDashboard_StateEndpoint(t *testing.T) {
	d := NewDashboard(t.TempDir())
	mux := http.NewServeMux()
	d.SetupRoutes(mux)

	req := httptest.NewRequest(http.MethodGet, "/api/state", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rec.Code)
	}

	var state map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&state); err != nil {
		t.Fatalf("decoding JSON: %v", err)
	}
	// SWE100821: Verify expected keys present
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

	// SWE100821: Create a fake skill directory and a yaml file
	os.MkdirAll(filepath.Join(skillsDir, "git-ops"), 0755)
	os.WriteFile(filepath.Join(skillsDir, "deploy.yaml"), []byte("name: deploy"), 0644)

	d := NewDashboard(dir)
	mux := http.NewServeMux()
	d.SetupRoutes(mux)

	req := httptest.NewRequest(http.MethodGet, "/api/skills", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rec.Code)
	}

	var names []string
	if err := json.NewDecoder(rec.Body).Decode(&names); err != nil {
		t.Fatalf("decoding JSON: %v", err)
	}
	if len(names) < 2 {
		t.Errorf("expected at least 2 skills, got %d: %v", len(names), names)
	}
}
