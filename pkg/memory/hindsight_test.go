// SWE100821: Tests for biomimetic hindsight memory — Retain, Recall, searchVaultFiles.
// Uses temp directories to avoid vault/LLM dependencies.

package memory

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

// SWE100821: TestSearchVaultFiles — keyword search across vault markdown files
func TestSearchVaultFiles(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "world-facts"), 0755)
	os.MkdirAll(filepath.Join(dir, "experiences"), 0755)
	os.MkdirAll(filepath.Join(dir, "mental-models"), 0755)

	// SWE100821: Write test files with searchable content
	os.WriteFile(filepath.Join(dir, "world-facts", "test.md"),
		[]byte("---\n---\nThe server runs on port 8080"), 0644)
	os.WriteFile(filepath.Join(dir, "experiences", "exp.md"),
		[]byte("---\n---\nI deployed the server yesterday"), 0644)

	hm := &HindsightMemory{vaultRoot: dir}
	results := hm.searchVaultFiles("server port", 5)
	if len(results) == 0 {
		t.Error("expected to find results for 'server port'")
	}

	// SWE100821: Verify result content
	found := false
	for _, r := range results {
		if r == "The server runs on port 8080" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected result containing port info, got: %v", results)
	}
}

// SWE100821: TestSearchVaultFiles_NoMatch — query with no hits returns empty
func TestSearchVaultFiles_NoMatch(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "world-facts"), 0755)
	os.WriteFile(filepath.Join(dir, "world-facts", "test.md"),
		[]byte("---\n---\nThe sky is blue"), 0644)

	hm := &HindsightMemory{vaultRoot: dir}
	results := hm.searchVaultFiles("quantum entanglement", 5)
	if len(results) != 0 {
		t.Errorf("expected no results for unrelated query, got: %v", results)
	}
}

// SWE100821: TestRetain_NilVault — Retain with nil vaultWriter returns nil
func TestRetain_NilVault(t *testing.T) {
	hm := &HindsightMemory{vaultWriter: nil}
	err := hm.Retain(context.Background(), "test memory", "test")
	if err != nil {
		t.Errorf("expected nil error with nil vaultWriter, got: %v", err)
	}
}

// SWE100821: TestRecall_NoBackends — nil semantic + empty vaultRoot returns empty
func TestRecall_NoBackends(t *testing.T) {
	hm := &HindsightMemory{
		semanticMemory: nil,
		vaultRoot:      "",
	}
	results, err := hm.Recall(context.Background(), "anything")
	if err != nil {
		t.Errorf("expected nil error, got: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected empty results with no backends, got: %v", results)
	}
}

// SWE100821: TestRecall_VaultOnly — recall with only vault root set
func TestRecall_VaultOnly(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "world-facts"), 0755)
	os.MkdirAll(filepath.Join(dir, "experiences"), 0755)
	os.MkdirAll(filepath.Join(dir, "mental-models"), 0755)

	os.WriteFile(filepath.Join(dir, "world-facts", "network.md"),
		[]byte("---\n---\nThe API runs on port 3000"), 0644)

	hm := &HindsightMemory{vaultRoot: dir}
	results, err := hm.Recall(context.Background(), "API port")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if len(results) == 0 {
		t.Error("expected results from vault search")
	}
}
