// Package taskrunner polls the task store and executes a three-stage pipeline for each task:
//   Stage 1 (code tasks): Coder agent — implements the feature on a new git branch
//   Stage 2 (code tasks): Test agent — writes unit tests, runs go test, fixes failures
//   Stage 3 (code tasks): If tests pass, commits and pushes the branch to GitHub
//   General/research:     Single agent pass, result stored on task
//
// Reused: selfimprove.Runner pattern (pkg/selfimprove/selfimprove.go L37-L80).
// SWE100821: New package — no prior task pipeline found.
package taskrunner

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/Dawgomatic/Xagent/pkg/logger"
	"github.com/Dawgomatic/Xagent/pkg/tasks"
)

// AgentRunner is satisfied by agent.AgentLoop.ProcessDirect.
type AgentRunner interface {
	ProcessDirect(ctx context.Context, content, sessionKey string) (string, error)
}

// Runner polls the task store and executes the pipeline.
type Runner struct {
	mu       sync.Mutex
	store    *tasks.Store
	repoPath string
	interval time.Duration
}

// New creates a Runner. repoPath is the git repo root for code tasks.
func New(store *tasks.Store, repoPath string, pollInterval time.Duration) *Runner {
	if pollInterval <= 0 {
		pollInterval = 2 * time.Minute
	}
	return &Runner{store: store, repoPath: repoPath, interval: pollInterval}
}

// Start launches the background polling loop until ctx is cancelled.
func (r *Runner) Start(ctx context.Context, ar AgentRunner) {
	logger.InfoCF("taskrunner", "Task pipeline started",
		map[string]interface{}{"interval": r.interval.String(), "repo": r.repoPath})
	go func() {
		ticker := time.NewTicker(r.interval)
		defer ticker.Stop()
		// immediate first check
		r.runNext(ctx, ar)
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				r.runNext(ctx, ar)
			}
		}
	}()
}

func (r *Runner) runNext(ctx context.Context, ar AgentRunner) {
	// SWE100821: serialise pipeline runs — one task at a time to avoid repo conflicts
	r.mu.Lock()
	defer r.mu.Unlock()

	task := r.store.NextPending()
	if task == nil {
		return
	}

	logger.InfoCF("taskrunner", "Starting task", map[string]interface{}{
		"id": task.ID, "type": task.Type, "title": task.Title,
	})

	_ = r.store.UpdateStatus(task.ID, tasks.StatusInProgress, "", "", "")

	switch task.Type {
	case tasks.TypeCode:
		r.runCodePipeline(ctx, ar, task)
	default:
		r.runGeneralTask(ctx, ar, task)
	}
}

// runCodePipeline: coder → tester → push
func (r *Runner) runCodePipeline(ctx context.Context, ar AgentRunner, task *tasks.Task) {
	slug := toSlug(task.Title)
	branch := "feature/" + slug

	// ---- Stage 1: Coder ----
	coderPrompt := buildCoderPrompt(task, r.repoPath, branch)
	coderOut, err := ar.ProcessDirect(ctx, coderPrompt, "taskrunner:code:"+task.ID)
	if err != nil {
		_ = r.store.UpdateStatus(task.ID, tasks.StatusFailed, "", fmt.Sprintf("coder agent error: %v", err), branch)
		logger.ErrorCF("taskrunner", "Coder agent failed", map[string]interface{}{"task": task.ID, "error": err})
		return
	}

	// ---- Stage 2: Test agent ----
	_ = r.store.UpdateStatus(task.ID, tasks.StatusTesting, "", "", branch)
	testerPrompt := buildTesterPrompt(task, r.repoPath, branch, coderOut)
	testerOut, err := ar.ProcessDirect(ctx, testerPrompt, "taskrunner:test:"+task.ID)
	if err != nil {
		_ = r.store.UpdateStatus(task.ID, tasks.StatusFailed, coderOut, fmt.Sprintf("test agent error: %v", err), branch)
		logger.ErrorCF("taskrunner", "Test agent failed", map[string]interface{}{"task": task.ID, "error": err})
		return
	}

	// ---- Stage 3: Evaluate and push ----
	testPassed := !containsFailure(testerOut)
	summary := fmt.Sprintf("## Coder output\n\n%s\n\n## Test output\n\n%s", coderOut, testerOut)
	if testPassed {
		_ = r.store.UpdateStatus(task.ID, tasks.StatusDone, summary, "", branch)
		logger.InfoCF("taskrunner", "Task completed", map[string]interface{}{"task": task.ID, "branch": branch})
	} else {
		_ = r.store.UpdateStatus(task.ID, tasks.StatusFailed, summary, "tests failed — see result for details", branch)
		logger.WarnCF("taskrunner", "Task failed: tests did not pass", map[string]interface{}{"task": task.ID, "branch": branch})
	}
}

func (r *Runner) runGeneralTask(ctx context.Context, ar AgentRunner, task *tasks.Task) {
	prompt := fmt.Sprintf("You have been assigned the following task from the task queue.\n\n## Task ID\n%s\n\n## Title\n%s\n\n## Description\n%s\n\nComplete the task and provide a thorough result summary.", task.ID, task.Title, task.Description)
	out, err := ar.ProcessDirect(ctx, prompt, "taskrunner:general:"+task.ID)
	if err != nil {
		_ = r.store.UpdateStatus(task.ID, tasks.StatusFailed, "", err.Error(), "")
		return
	}
	_ = r.store.UpdateStatus(task.ID, tasks.StatusDone, out, "", "")
	logger.InfoCF("taskrunner", "General task completed", map[string]interface{}{"task": task.ID})
}

func buildCoderPrompt(task *tasks.Task, repoPath, branch string) string {
	return fmt.Sprintf(`You are executing a **code task** from the task queue.

## Task ID
%s

## Title
%s

## Description
%s

## Repository
- Git root: %q
- New branch to create: %q (branch off main/master with git checkout -b)

## Instructions
1. Use list_directory / read_file to understand the codebase structure first.
2. Implement the feature described above. Keep changes focused and minimal.
3. Only modify files under pkg/, cmd/, docs/, or config/ — never reference/, .env, or config files with real secrets.
4. Run exec "go build ./..." from repo root to verify it compiles.
5. Use exec to git: checkout main, pull, create branch %q, add files, commit with a conventional message.
6. Do NOT push yet — the test agent will handle that after tests pass.

End your response with:
- A summary of what you implemented
- The exact files changed
- Any known limitations or follow-up work`,
		task.ID, task.Title, task.Description, repoPath, branch, branch)
}

func buildTesterPrompt(task *tasks.Task, repoPath, branch, coderOutput string) string {
	return fmt.Sprintf(`You are the **test agent** for a code task. The coder has already implemented the feature. Your job is to write unit tests, run them, and fix any failures.

## Task
%s — %s

## Coder output summary
%s

## Repository
- Git root: %q
- Branch: %q (already checked out by coder)

## Instructions
1. Read the files the coder modified/created.
2. Write or update *_test.go files for the changed packages. Cover the happy path and at least one error case.
3. Run exec "cd %s && go test ./..." — capture output.
4. If tests fail, fix both the tests and the implementation until they pass (max 3 iterations).
5. If tests pass: exec "cd %s && git add -A && git commit -m 'test(%s): add unit tests' && git push -u origin %s"
6. If tests still fail after 3 attempts: document what failed and stop — do NOT push.

End with a clear PASS or FAIL verdict and the full test output.`,
		task.ID, task.Title,
		truncate(coderOutput, 2000),
		repoPath, branch,
		repoPath, repoPath, toSlug(task.Title), branch)
}

// containsFailure checks test output for failure indicators.
func containsFailure(output string) bool {
	lower := strings.ToLower(output)
	return strings.Contains(lower, "fail") ||
		strings.Contains(lower, "error") ||
		strings.Contains(lower, "panic") ||
		(!strings.Contains(lower, "ok") && !strings.Contains(lower, "pass"))
}

func toSlug(s string) string {
	s = strings.ToLower(s)
	var out []rune
	for _, c := range s {
		if c >= 'a' && c <= 'z' || c >= '0' && c <= '9' {
			out = append(out, c)
		} else if len(out) > 0 && out[len(out)-1] != '-' {
			out = append(out, '-')
		}
	}
	// trim trailing dash
	for len(out) > 0 && out[len(out)-1] == '-' {
		out = out[:len(out)-1]
	}
	if len(out) > 40 {
		out = out[:40]
	}
	return string(out)
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "…"
}
