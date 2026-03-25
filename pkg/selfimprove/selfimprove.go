// Package selfimprove runs periodic autonomous improvement passes: the agent researches,
// edits allowed paths in the Git repo, runs tests, and pushes a new feature-named branch when configured.
// Reused: agent loop ProcessDirect pattern from cmd/xagent/cmd_gateway.go (proactive loop).
// SWE100821: Enabled by default; auto_push requires Git credentials on the host.
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
		prefix = "feature"
	}
	remote := si.RemoteName
	if remote == "" {
		remote = "origin"
	}
	allowed := si.AllowedPathPrefixes
	if len(allowed) == 0 {
		allowed = []string{"pkg/", "cmd/", "docs/", "config/"}
	}

	prompt := buildPrompt(repo, workspace, prefix, remote, si.AutoPush, allowed)

	header := fmt.Sprintf("# Self-improve run %s\n\n- repo: `%s`\n- branch namespace: `%s/<feature-slug>` (you name the slug)\n- auto_push: %v\n\n---\n\n## Prompt\n\n%s\n\n---\n\n## Agent output\n\n",
		ts, repo, prefix, si.AutoPush, prompt)
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

func buildPrompt(repoRoot, agentWorkspace, branchPrefix, remote string, autoPush bool, allowed []string) string {
	ap := "**Allowed path prefixes (edit ONLY under these relative to repo root):**\n"
	for _, p := range allowed {
		ap += fmt.Sprintf("- `%s`\n", p)
	}
	push := "Do **not** run git push. Commit locally only."
	if autoPush {
		push = fmt.Sprintf(`After a successful commit on your new branch, push it:
cd %q && git push -u %s HEAD
Use the exact branch name you created (prefix + slug). SSH or credential helper must already be configured — never put tokens in files or commit messages.`, repoRoot, remote)
	}
	ex := fmt.Sprintf("%s/add-dashboard-export, %s/self-improve-runner-tests", branchPrefix, branchPrefix)
	return fmt.Sprintf(`You are running a **scheduled autonomous self-improvement** pass for the Xagent codebase.

## Repository
- **Git root:** %q
- **Agent workspace** (memory/skills; not necessarily the repo): %q
- **Branch namespace (prefix):** %q — you must create a **brand-new branch** for this run only.

## Branch naming (required)
1. Choose a **short kebab-case slug** that names the feature or improvement (e.g. add-metrics-export, refactor-cron-validation, dashboard-cache-headers).
2. Full branch name: PREFIX + "/" + slug where PREFIX is %q. Examples: %s
3. If the work is a **large feature** (multiple packages, new APIs, substantial behavior), the slug should still be a **single descriptive phrase** (not a timestamp). One branch per run; name the branch after that feature.
4. Do not reuse an existing branch name; if unsure, append a short suffix like -v2 only after checking git branch -a.

## Your mission
1. Use **web_search** (and **fetch** if needed) to find concrete improvements relevant to a Go AI agent (patterns, libraries, UX, tests, docs). Prefer ideas that can become a **feature-sized** change when appropriate; otherwise a focused fix is fine.
2. **read_file** / **list_directory** only under allowed prefixes.
3. Implement the improvement: for a **large feature**, cover tests and wiring; for a small fix, keep scope tight.
4. Run **exec** from repo root: "go test" on affected packages (or ./... if reasonable). Fix failures before committing.
5. **git** via **exec**: fetch latest base if needed, git checkout main or master (or default branch), git pull, then **git checkout -b %s/<your-slug>** (replace <your-slug> with the kebab name you chose), git add (only allowed paths), git commit with a conventional message. Never add secrets or config keys.

%s

## Safety (hard rules)
- Do NOT modify: .env, real config.json with secrets, **reference/** subtrees, or huge vendor trees.
- Do NOT force-push, reset --hard to remote, or delete branches you did not create.
- If tests cannot pass in reasonable time, **document** blockers in the commit message or skip commit and explain in text.

%s

End with a short summary: branch name, what changed, and what remains.`,
		repoRoot, agentWorkspace, branchPrefix, branchPrefix, ex, branchPrefix, ap, push)
}
