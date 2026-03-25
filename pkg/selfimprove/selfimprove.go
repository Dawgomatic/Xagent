// Package selfimprove runs periodic autonomous improvement passes: the agent researches,
// edits allowed paths in the Git repo, runs tests, commits, and optionally pushes.
// Reused: agent loop ProcessDirect pattern from cmd/xagent/cmd_gateway.go (proactive loop).
// SWE100821: Disabled by default; requires explicit config and Git credentials on the host.
package selfimprove

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/Dawgomatic/Xagent/pkg/config"
	"github.com/Dawgomatic/Xagent/pkg/logger"
)

// AgentRunner is the subset of the agent used for improvement runs (avoids import cycles).
type AgentRunner interface {
	ProcessDirect(ctx context.Context, content, sessionKey string) (string, error)
}

// Runner serializes self-improve passes so only one runs at a time.
type Runner struct {
	mu sync.Mutex
}

// NewRunner creates a runner with no background state until Start.
func NewRunner() *Runner {
	return &Runner{}
}

// Start launches a goroutine that triggers improvement passes on an interval until ctx is done.
// SWE100821: initialDelay avoids competing with gateway startup; interval from cfg (hours).
func (r *Runner) Start(ctx context.Context, cfg *config.Config, ar AgentRunner) {
	si := cfg.SelfImprove
	if !si.Enabled {
		return
	}
	interval := time.Duration(si.IntervalHours) * time.Hour
	if si.IntervalHours <= 0 {
		interval = 168 * time.Hour // 1 week default
	}
	initial := time.Duration(si.InitialDelayMins) * time.Minute
	if si.InitialDelayMins <= 0 {
		initial = 30 * time.Minute
	}
	ws := cfg.WorkspacePath()
	repo := resolveRepoPath(ws, si.RepoPath)

	go func() {
		select {
		case <-ctx.Done():
			return
		case <-time.After(initial):
		}
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		r.runOnce(ctx, cfg, ar, ws, repo)
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				r.runOnce(ctx, cfg, ar, ws, repo)
			}
		}
	}()
	logger.InfoCF("self_improve", "Background self-improve loop started",
		map[string]interface{}{
			"interval":       interval.String(),
			"initial_delay":  initial.String(),
			"repo":           repo,
			"auto_push":      si.AutoPush,
		})
}

func resolveRepoPath(workspace, configured string) string {
	configured = strings.TrimSpace(configured)
	if configured != "" {
		return filepath.Clean(configured)
	}
	return FindGitRoot(workspace)
}

func (r *Runner) runOnce(ctx context.Context, cfg *config.Config, ar AgentRunner, workspace, repo string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	si := cfg.SelfImprove
	if repo == "" {
		logger.WarnCF("self_improve", "Skipping run: no git repo (set self_improve.repo_path or place workspace inside a clone)", nil)
		return
	}

	logDir := filepath.Join(workspace, "self-improve")
	if err := os.MkdirAll(logDir, 0755); err != nil {
		logger.ErrorCF("self_improve", "mkdir log dir", map[string]interface{}{"error": err.Error()})
		return
	}
	ts := time.Now().UTC().Format("2006-01-02T15-04-05Z")
	runLog := filepath.Join(logDir, "run-"+ts+".md")
	prefix := si.BranchPrefix
	if prefix == "" {
		prefix = "autonomous/self-improve"
	}
	branch := fmt.Sprintf("%s-%s", prefix, time.Now().UTC().Format("20060102-150405"))
	remote := si.RemoteName
	if remote == "" {
		remote = "origin"
	}
	allowed := si.AllowedPathPrefixes
	if len(allowed) == 0 {
		allowed = []string{"pkg/", "cmd/", "docs/", "config/"}
	}

	prompt := buildPrompt(repo, workspace, branch, remote, si.AutoPush, allowed)

	header := fmt.Sprintf("# Self-improve run %s\n\n- repo: `%s`\n- branch: `%s`\n- auto_push: %v\n\n---\n\n## Prompt\n\n%s\n\n---\n\n## Agent output\n\n",
		ts, repo, branch, si.AutoPush, prompt)
	if err := os.WriteFile(runLog, []byte(header), 0644); err != nil {
		logger.ErrorCF("self_improve", "write run log", map[string]interface{}{"error": err.Error()})
		return
	}

	resp, err := ar.ProcessDirect(ctx, prompt, "self-improve:scheduled")
	out := resp
	if err != nil {
		out = fmt.Sprintf("ERROR: %v\n\n%s", err, resp)
	}
	f, err := os.OpenFile(runLog, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		logger.ErrorCF("self_improve", "append log", map[string]interface{}{"error": err.Error()})
		return
	}
	_, _ = f.WriteString(out)
	_ = f.Close()

	// Rolling combined log
	combined := filepath.Join(logDir, "log.md")
	summary := fmt.Sprintf("\n\n---\n## %s\n%s\n", ts, truncateForSummary(out, 8000))
	if cf, err := os.OpenFile(combined, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644); err == nil {
		_, _ = cf.WriteString(summary)
		_ = cf.Close()
	}

	if err != nil {
		logger.WarnCF("self_improve", "Run finished with error", map[string]interface{}{"error": err.Error(), "log": runLog})
	} else {
		logger.InfoCF("self_improve", "Run finished", map[string]interface{}{"log": runLog, "chars": len(out)})
	}
}

func truncateForSummary(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "\n\n…(truncated in log.md; see run file for full output)…"
}

func buildPrompt(repoRoot, agentWorkspace, branch, remote string, autoPush bool, allowed []string) string {
	ap := "**Allowed path prefixes (edit ONLY under these relative to repo root):**\n"
	for _, p := range allowed {
		ap += fmt.Sprintf("- `%s`\n", p)
	}
	push := "Do **not** run `git push`. Commit locally only; the user will push."
	if autoPush {
		push = fmt.Sprintf(`After a successful commit, push with:
cd %q && git push -u %s HEAD
Only if SSH or a credential helper is already configured — never put tokens in files or commit messages.`, repoRoot, remote)
	}
	return fmt.Sprintf(`You are running a **scheduled autonomous self-improvement** pass for the Xagent codebase.

## Repository
- **Git root:** %q
- **Agent workspace** (memory/skills; not necessarily the repo): %q
- **Create and use branch:** %q (checkout from main/master as appropriate)

## Your mission
1. Use **web_search** (and **fetch** if needed) to find 1–3 concrete, small improvements relevant to a Go AI agent (patterns, libraries, UX, tests, docs). Stay high-signal; avoid unrelated stacks.
2. **read_file** / **list_directory** to inspect only files under the allowed prefixes.
3. Implement **one small, testable improvement** (prefer: bugfix, test, doc clarity, minor feature behind existing patterns).
4. Run **exec** from repo root: run "go test ./..." or a narrower path if full tree is too heavy. Fix failures before committing.
5. **git** via **exec**: git status, git checkout -b (branch), git add (only allowed paths), git commit with a conventional message. Never add secrets or config keys.

%s

## Safety (hard rules)
- Do NOT modify: .env, real config.json with secrets, **reference/** subtrees, or huge vendor trees.
- Do NOT force-push, reset --hard to remote, or delete branches you did not create.
- If tests cannot pass in reasonable time, **document** blockers in the commit message or skip commit and explain in text.

%s

End with a short summary of what changed and what remains.`,
		repoRoot, agentWorkspace, branch, ap, push)
}
