// task_tool.go exposes the centralized task queue to the LLM as four tools:
//   create_task  — add a new task to the queue
//   list_tasks   — list tasks by status
//   get_task     — fetch a single task by ID
//   update_task  — update status/result (used by taskrunner and agents)
//
// SWE100821: New file — no existing task management found in pkg/tools/.
package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/Dawgomatic/Xagent/pkg/tasks"
)

// ---- create_task ----

// CreateTaskTool lets the LLM enqueue work items for the task pipeline.
type CreateTaskTool struct{ store *tasks.Store }

func NewCreateTaskTool(store *tasks.Store) *CreateTaskTool { return &CreateTaskTool{store: store} }
func (t *CreateTaskTool) Name() string                     { return "create_task" }
func (t *CreateTaskTool) Description() string {
	return "Add a new task to the centralized task queue. Code tasks are automatically implemented, tested, and pushed to GitHub by the task pipeline. Use type='code' for programming tasks, 'research' for investigation, 'general' for everything else."
}
func (t *CreateTaskTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"title":       map[string]interface{}{"type": "string", "description": "Short task title (≤80 chars)"},
			"description": map[string]interface{}{"type": "string", "description": "Full task description — what needs to be done, acceptance criteria, relevant context"},
			"task_type":   map[string]interface{}{"type": "string", "enum": []string{"code", "research", "general"}, "description": "Type of task"},
			"priority":    map[string]interface{}{"type": "integer", "description": "Priority 1 (low) to 5 (critical), default 3"},
		},
		"required": []string{"title", "description", "task_type"},
	}
}
func (t *CreateTaskTool) Execute(_ context.Context, args map[string]interface{}) *ToolResult {
	title, _ := args["title"].(string)
	desc, _ := args["description"].(string)
	tt, _ := args["task_type"].(string)
	if title == "" || desc == "" || tt == "" {
		return ErrorResult("create_task: title, description, and task_type are required")
	}
	priority := 3
	if p, ok := args["priority"].(float64); ok && p >= 1 && p <= 5 {
		priority = int(p)
	}
	task, err := t.store.Create(title, desc, tasks.Type(tt), priority, "agent")
	if err != nil {
		return ErrorResult(fmt.Sprintf("create_task: %v", err))
	}
	return NewToolResult(fmt.Sprintf("Task created:\n  ID:       %s\n  Title:    %s\n  Type:     %s\n  Priority: %d\n  Status:   %s\n\nThe task pipeline will pick it up automatically.", task.ID, task.Title, task.Type, task.Priority, task.Status))
}

// ---- list_tasks ----

// ListTasksTool lets the LLM and runner inspect the queue.
type ListTasksTool struct{ store *tasks.Store }

func NewListTasksTool(store *tasks.Store) *ListTasksTool { return &ListTasksTool{store: store} }
func (t *ListTasksTool) Name() string                    { return "list_tasks" }
func (t *ListTasksTool) Description() string {
	return "List tasks in the task queue, optionally filtered by status (pending, in_progress, testing, done, failed, cancelled). Leave status empty to list all."
}
func (t *ListTasksTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"status": map[string]interface{}{"type": "string", "description": "Filter by status (optional)"},
		},
		"required": []string{},
	}
}
func (t *ListTasksTool) Execute(_ context.Context, args map[string]interface{}) *ToolResult {
	status, _ := args["status"].(string)
	list := t.store.List(tasks.Status(status))
	if len(list) == 0 {
		return NewToolResult("No tasks found.")
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Tasks (%d):\n\n", len(list)))
	for _, task := range list {
		sb.WriteString(fmt.Sprintf("[%s] %s (%s/%s, pri=%d)\n  %s\n", task.ID, task.Title, task.Type, task.Status, task.Priority, truncate(task.Description, 120)))
		if task.Branch != "" {
			sb.WriteString(fmt.Sprintf("  branch: %s\n", task.Branch))
		}
		if task.ErrorMsg != "" {
			sb.WriteString(fmt.Sprintf("  error: %s\n", truncate(task.ErrorMsg, 80)))
		}
		sb.WriteString("\n")
	}
	return NewToolResult(sb.String())
}

// ---- get_task ----

// GetTaskTool returns full detail on a single task.
type GetTaskTool struct{ store *tasks.Store }

func NewGetTaskTool(store *tasks.Store) *GetTaskTool { return &GetTaskTool{store: store} }
func (t *GetTaskTool) Name() string                  { return "get_task" }
func (t *GetTaskTool) Description() string {
	return "Get full details of a task by its ID, including description, status, result, and branch."
}
func (t *GetTaskTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type":       "object",
		"properties": map[string]interface{}{"id": map[string]interface{}{"type": "string", "description": "Task ID"}},
		"required":   []string{"id"},
	}
}
func (t *GetTaskTool) Execute(_ context.Context, args map[string]interface{}) *ToolResult {
	id, _ := args["id"].(string)
	task, err := t.store.Get(id)
	if err != nil {
		return ErrorResult(err.Error())
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Task: %s\nTitle: %s\nType: %s\nStatus: %s\nPriority: %d\nCreated: %s\nUpdated: %s\n",
		task.ID, task.Title, task.Type, task.Status, task.Priority,
		task.CreatedAt.Format("2006-01-02 15:04:05 UTC"),
		task.UpdatedAt.Format("2006-01-02 15:04:05 UTC")))
	if task.Branch != "" {
		sb.WriteString(fmt.Sprintf("Branch: %s\n", task.Branch))
	}
	sb.WriteString(fmt.Sprintf("\nDescription:\n%s\n", task.Description))
	if task.Result != "" {
		sb.WriteString(fmt.Sprintf("\nResult:\n%s\n", task.Result))
	}
	if task.ErrorMsg != "" {
		sb.WriteString(fmt.Sprintf("\nError: %s\n", task.ErrorMsg))
	}
	return NewToolResult(sb.String())
}

// ---- update_task ----

// UpdateTaskTool lets agents update task status and attach results.
type UpdateTaskTool struct{ store *tasks.Store }

func NewUpdateTaskTool(store *tasks.Store) *UpdateTaskTool { return &UpdateTaskTool{store: store} }
func (t *UpdateTaskTool) Name() string                     { return "update_task" }
func (t *UpdateTaskTool) Description() string {
	return "Update the status or result of a task. Used by agents completing pipeline stages."
}
func (t *UpdateTaskTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"id":     map[string]interface{}{"type": "string"},
			"status": map[string]interface{}{"type": "string", "enum": []string{"pending", "in_progress", "testing", "done", "failed", "cancelled"}},
			"result": map[string]interface{}{"type": "string", "description": "Output / summary of what was done"},
			"error":  map[string]interface{}{"type": "string", "description": "Error message if failed"},
			"branch": map[string]interface{}{"type": "string", "description": "Git branch name if code task"},
		},
		"required": []string{"id", "status"},
	}
}
func (t *UpdateTaskTool) Execute(_ context.Context, args map[string]interface{}) *ToolResult {
	id, _ := args["id"].(string)
	status, _ := args["status"].(string)
	result, _ := args["result"].(string)
	errMsg, _ := args["error"].(string)
	branch, _ := args["branch"].(string)
	if err := t.store.UpdateStatus(id, tasks.Status(status), result, errMsg, branch); err != nil {
		return ErrorResult(err.Error())
	}
	return NewToolResult(fmt.Sprintf("Task %s updated to status=%s", id, status))
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "…"
}
