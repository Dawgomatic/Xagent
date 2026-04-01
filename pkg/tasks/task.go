// Package tasks defines the Task struct and status/type enums for the centralized task queue.
// Tasks are created by the LLM via the task_tool, persisted to disk, and consumed by
// the taskrunner pipeline (code → test → push).
// SWE100821: New package for centralized agent task management.
package tasks

import "time"

// Status represents the lifecycle state of a task.
type Status string

const (
	StatusPending    Status = "pending"
	StatusInProgress Status = "in_progress"
	StatusTesting    Status = "testing"
	StatusDone       Status = "done"
	StatusFailed     Status = "failed"
	StatusCancelled  Status = "cancelled"
)

// Type classifies what kind of work the task requires.
type Type string

const (
	TypeCode     Type = "code"     // write code, test, push to GitHub
	TypeResearch Type = "research" // research a topic, store findings
	TypeGeneral  Type = "general"  // general agent task
)

// Task is the canonical unit of work in the task queue.
type Task struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Type        Type      `json:"type"`
	Status      Status    `json:"status"`
	Priority    int       `json:"priority"` // 1=low … 5=critical
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	StartedAt   *time.Time `json:"started_at,omitempty"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
	Branch      string    `json:"branch,omitempty"`      // git branch used for code tasks
	Result      string    `json:"result,omitempty"`      // agent output / test results
	ErrorMsg    string    `json:"error,omitempty"`       // failure reason
	RequestedBy string    `json:"requested_by,omitempty"` // channel/user that created the task
}
