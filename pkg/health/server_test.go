// SWE100821: Tests for the health server — verifies liveness, readiness,
// metrics, and custom handler registration via httptest.
package health

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// SWE100821: helper — creates a Server and returns its internal mux for httptest
func testMux(t *testing.T) (*Server, *http.ServeMux) {
	t.Helper()
	s := NewServer("127.0.0.1", 0) // SWE100821: port 0 — not actually listening
	mux, ok := s.httpServer.Handler.(*http.ServeMux)
	if !ok {
		t.Fatal("expected *http.ServeMux from server handler")
	}
	return s, mux
}

// SWE100821: TestHealthz — GET /healthz returns 200 with status=ok
func TestHealthz(t *testing.T) {
	_, mux := testMux(t)

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rec.Code)
	}
	var body map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decoding: %v", err)
	}
	if body["status"] != "ok" {
		t.Errorf("status = %v, want \"ok\"", body["status"])
	}
}

// SWE100821: TestReadyz_Ready — set ready=true, verify 200
func TestReadyz_Ready(t *testing.T) {
	s, mux := testMux(t)
	s.SetReady(true) // SWE100821: mark ready

	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rec.Code)
	}
}

// SWE100821: TestReadyz_NotReady — default (not ready), verify 503
func TestReadyz_NotReady(t *testing.T) {
	_, mux := testMux(t)

	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want 503", rec.Code)
	}
}

// SWE100821: TestMetricsz — verify JSON with expected keys
func TestMetricsz(t *testing.T) {
	s, mux := testMux(t)
	s.GetMetrics().IncMessage() // SWE100821: bump a counter so there's data

	req := httptest.NewRequest(http.MethodGet, "/metricsz", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rec.Code)
	}
	var body map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decoding: %v", err)
	}
	for _, key := range []string{"messages_total", "llm_calls_total", "tool_calls_total", "uptime_seconds"} {
		if _, ok := body[key]; !ok {
			t.Errorf("missing key %q in metricsz response", key)
		}
	}
}

// SWE100821: TestRegisterHandler — register custom handler, verify it responds
func TestRegisterHandler(t *testing.T) {
	s, mux := testMux(t)

	s.RegisterHandler("/custom", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("custom-ok"))
	}))

	req := httptest.NewRequest(http.MethodGet, "/custom", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rec.Code)
	}
	if rec.Body.String() != "custom-ok" {
		t.Errorf("body = %q, want \"custom-ok\"", rec.Body.String())
	}
}
