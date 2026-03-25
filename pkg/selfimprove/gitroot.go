package selfimprove

import (
	"os"
	"path/filepath"
)

// FindGitRoot walks upward from dir until a .git directory is found, or returns "".
// SWE100821: Workspace is often a subdirectory of the clone (e.g. workspace/ vs repo root).
func FindGitRoot(dir string) string {
	dir = filepath.Clean(dir)
	for i := 0; i < 12; i++ {
		gitPath := filepath.Join(dir, ".git")
		if st, err := os.Stat(gitPath); err == nil && (st.IsDir() || st.Mode()&os.ModeType != 0) {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return ""
}
