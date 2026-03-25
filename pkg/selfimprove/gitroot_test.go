package selfimprove

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFindGitRoot(t *testing.T) {
	tmp := t.TempDir()
	repo := filepath.Join(tmp, "repo")
	if err := os.MkdirAll(filepath.Join(repo, ".git"), 0755); err != nil {
		t.Fatal(err)
	}
	nested := filepath.Join(repo, "a", "b", "workspace")
	if err := os.MkdirAll(nested, 0755); err != nil {
		t.Fatal(err)
	}
	if g := FindGitRoot(nested); g != repo {
		t.Fatalf("FindGitRoot = %q want %q", g, repo)
	}
	if g := FindGitRoot(tmp); g != "" {
		t.Fatalf("FindGitRoot(no .git) = %q want empty", g)
	}
}
