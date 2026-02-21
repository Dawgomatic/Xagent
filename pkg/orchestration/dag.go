// SWE100821: Task DAG execution engine — models complex tasks as directed acyclic graphs
// with dependencies. Executes independent tasks in parallel, dependent tasks sequentially.
// Most agent frameworks are purely sequential; this is a differentiator.

package orchestration

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/Dawgomatic/Xagent/pkg/logger"
)

// TaskNode represents a single node in the task DAG.
type TaskNode struct {
	ID          string   `json:"id"`
	Description string   `json:"description"`
	Role        string   `json:"role,omitempty"` // subagent role to use
	DependsOn   []string `json:"depends_on"`     // IDs of prerequisite tasks
	Status      string   `json:"status"`         // pending, running, completed, failed
	Result      string   `json:"result,omitempty"`
	StartedAt   *time.Time `json:"started_at,omitempty"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
	Error       string   `json:"error,omitempty"`
}

// TaskDAG is a directed acyclic graph of tasks.
type TaskDAG struct {
	ID    string               `json:"id"`
	Goal  string               `json:"goal"`
	Nodes map[string]*TaskNode `json:"nodes"`
	mu    sync.RWMutex
}

// TaskExecutor is the function signature for executing a single task node.
type TaskExecutor func(ctx context.Context, node *TaskNode) (result string, err error)

// NewTaskDAG creates an empty DAG.
func NewTaskDAG(id, goal string) *TaskDAG {
	return &TaskDAG{
		ID:    id,
		Goal:  goal,
		Nodes: make(map[string]*TaskNode),
	}
}

// AddNode adds a task to the DAG.
func (dag *TaskDAG) AddNode(node *TaskNode) error {
	dag.mu.Lock()
	defer dag.mu.Unlock()

	// Validate dependencies exist
	for _, dep := range node.DependsOn {
		if _, ok := dag.Nodes[dep]; !ok {
			return fmt.Errorf("dependency %q not found for task %q", dep, node.ID)
		}
	}

	node.Status = "pending"
	dag.Nodes[node.ID] = node
	return nil
}

// Validate checks the DAG for cycles and missing dependencies.
func (dag *TaskDAG) Validate() error {
	dag.mu.RLock()
	defer dag.mu.RUnlock()

	// Check for missing dependencies
	for id, node := range dag.Nodes {
		for _, dep := range node.DependsOn {
			if _, ok := dag.Nodes[dep]; !ok {
				return fmt.Errorf("task %q depends on missing task %q", id, dep)
			}
		}
	}

	// Check for cycles using DFS
	visited := make(map[string]bool)
	stack := make(map[string]bool)

	var hasCycle func(id string) bool
	hasCycle = func(id string) bool {
		visited[id] = true
		stack[id] = true

		node := dag.Nodes[id]
		for _, dep := range node.DependsOn {
			if !visited[dep] {
				if hasCycle(dep) {
					return true
				}
			} else if stack[dep] {
				return true
			}
		}

		stack[id] = false
		return false
	}

	for id := range dag.Nodes {
		if !visited[id] {
			if hasCycle(id) {
				return fmt.Errorf("cycle detected in task DAG")
			}
		}
	}

	return nil
}

// Execute runs the DAG, executing independent tasks in parallel.
func (dag *TaskDAG) Execute(ctx context.Context, executor TaskExecutor, maxParallel int) error {
	if err := dag.Validate(); err != nil {
		return err
	}

	if maxParallel <= 0 {
		maxParallel = 3
	}

	sem := make(chan struct{}, maxParallel)

	for {
		// Find ready tasks (all dependencies completed)
		ready := dag.getReadyTasks()
		if len(ready) == 0 {
			// Check if we're done or stuck
			if dag.isComplete() {
				break
			}
			if dag.hasFailed() {
				return fmt.Errorf("DAG execution halted: failed tasks block remaining work")
			}
			// Wait for running tasks
			time.Sleep(100 * time.Millisecond)
			continue
		}

		var wg sync.WaitGroup
		for _, node := range ready {
			wg.Add(1)
			n := node

			sem <- struct{}{} // acquire slot
			go func() {
				defer wg.Done()
				defer func() { <-sem }() // release slot

				dag.executeNode(ctx, n, executor)
			}()
		}
		wg.Wait()

		// Check context cancellation
		if ctx.Err() != nil {
			return ctx.Err()
		}
	}

	logger.InfoCF("dag", "DAG execution complete",
		map[string]interface{}{
			"dag_id":     dag.ID,
			"total_tasks": len(dag.Nodes),
			"completed":  dag.countByStatus("completed"),
			"failed":     dag.countByStatus("failed"),
		})

	return nil
}

// Summary returns a human-readable summary of the DAG execution.
func (dag *TaskDAG) Summary() string {
	dag.mu.RLock()
	defer dag.mu.RUnlock()

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("## Task DAG: %s\n\n", dag.Goal))

	for _, node := range dag.Nodes {
		marker := "⬜"
		switch node.Status {
		case "completed":
			marker = "✅"
		case "running":
			marker = "🔄"
		case "failed":
			marker = "❌"
		}

		deps := ""
		if len(node.DependsOn) > 0 {
			deps = fmt.Sprintf(" (depends on: %s)", strings.Join(node.DependsOn, ", "))
		}

		sb.WriteString(fmt.Sprintf("%s %s: %s%s\n", marker, node.ID, node.Description, deps))

		if node.Result != "" {
			result := node.Result
			if len(result) > 200 {
				result = result[:200] + "..."
			}
			sb.WriteString(fmt.Sprintf("   Result: %s\n", result))
		}
		if node.Error != "" {
			sb.WriteString(fmt.Sprintf("   Error: %s\n", node.Error))
		}
	}

	return sb.String()
}

func (dag *TaskDAG) executeNode(ctx context.Context, node *TaskNode, executor TaskExecutor) {
	dag.mu.Lock()
	now := time.Now()
	node.Status = "running"
	node.StartedAt = &now
	dag.mu.Unlock()

	result, err := executor(ctx, node)

	dag.mu.Lock()
	defer dag.mu.Unlock()
	completed := time.Now()
	node.CompletedAt = &completed

	if err != nil {
		node.Status = "failed"
		node.Error = err.Error()
		logger.ErrorCF("dag", "Task failed",
			map[string]interface{}{"task": node.ID, "error": err.Error()})
	} else {
		node.Status = "completed"
		node.Result = result
		logger.InfoCF("dag", "Task completed",
			map[string]interface{}{"task": node.ID, "result_len": len(result)})
	}
}

func (dag *TaskDAG) getReadyTasks() []*TaskNode {
	dag.mu.RLock()
	defer dag.mu.RUnlock()

	var ready []*TaskNode
	for _, node := range dag.Nodes {
		if node.Status != "pending" {
			continue
		}
		allDepsDone := true
		for _, dep := range node.DependsOn {
			if depNode, ok := dag.Nodes[dep]; ok {
				if depNode.Status != "completed" {
					allDepsDone = false
					break
				}
			}
		}
		if allDepsDone {
			ready = append(ready, node)
		}
	}
	return ready
}

func (dag *TaskDAG) isComplete() bool {
	dag.mu.RLock()
	defer dag.mu.RUnlock()
	for _, node := range dag.Nodes {
		if node.Status == "pending" || node.Status == "running" {
			return false
		}
	}
	return true
}

func (dag *TaskDAG) hasFailed() bool {
	dag.mu.RLock()
	defer dag.mu.RUnlock()
	for _, node := range dag.Nodes {
		if node.Status == "failed" {
			return true
		}
	}
	return false
}

func (dag *TaskDAG) countByStatus(status string) int {
	dag.mu.RLock()
	defer dag.mu.RUnlock()
	count := 0
	for _, node := range dag.Nodes {
		if node.Status == status {
			count++
		}
	}
	return count
}
