// store.go — JSON file-backed task store.
// Persists to workspace/tasks/tasks.json; safe for concurrent reads/writes via mutex.
// SWE100821: Linear scan acceptable — task queues rarely exceed hundreds of entries.
package tasks

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Store is a thread-safe, file-backed task queue.
type Store struct {
	mu   sync.RWMutex
	path string
	data []*Task
}

// NewStore opens or creates a task store at workspace/tasks/tasks.json.
func NewStore(workspace string) (*Store, error) {
	dir := filepath.Join(workspace, "tasks")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("tasks: mkdir %s: %w", dir, err)
	}
	s := &Store{path: filepath.Join(dir, "tasks.json")}
	if err := s.load(); err != nil {
		return nil, err
	}
	return s, nil
}

// Create adds a new task and persists it. Returns the created Task.
func (s *Store) Create(title, description string, taskType Type, priority int, requestedBy string) (*Task, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	t := &Task{
		ID:          fmt.Sprintf("task-%d", time.Now().UnixNano()),
		Title:       title,
		Description: description,
		Type:        taskType,
		Status:      StatusPending,
		Priority:    priority,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
		RequestedBy: requestedBy,
	}
	s.data = append(s.data, t)
	return t, s.save()
}

// Get returns a task by ID, or an error if not found.
func (s *Store) Get(id string) (*Task, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, t := range s.data {
		if t.ID == id {
			cp := *t
			return &cp, nil
		}
	}
	return nil, fmt.Errorf("task %q not found", id)
}

// List returns tasks optionally filtered by status. Empty status = all.
func (s *Store) List(status Status) []*Task {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []*Task
	for _, t := range s.data {
		if status == "" || t.Status == status {
			cp := *t
			out = append(out, &cp)
		}
	}
	return out
}

// UpdateStatus atomically updates a task's status and optional fields.
func (s *Store) UpdateStatus(id string, status Status, result, errMsg, branch string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, t := range s.data {
		if t.ID != id {
			continue
		}
		t.Status = status
		t.UpdatedAt = time.Now().UTC()
		if result != "" {
			t.Result = result
		}
		if errMsg != "" {
			t.ErrorMsg = errMsg
		}
		if branch != "" {
			t.Branch = branch
		}
		now := time.Now().UTC()
		switch status {
		case StatusInProgress, StatusTesting:
			if t.StartedAt == nil {
				t.StartedAt = &now
			}
		case StatusDone, StatusFailed, StatusCancelled:
			t.CompletedAt = &now
		}
		return s.save()
	}
	return fmt.Errorf("task %q not found", id)
}

// NextPending returns the highest-priority pending task, or nil if none.
func (s *Store) NextPending() *Task {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var best *Task
	for _, t := range s.data {
		if t.Status != StatusPending {
			continue
		}
		if best == nil || t.Priority > best.Priority || (t.Priority == best.Priority && t.CreatedAt.Before(best.CreatedAt)) {
			cp := *t
			best = &cp
		}
	}
	return best
}

func (s *Store) load() error {
	data, err := os.ReadFile(s.path)
	if os.IsNotExist(err) {
		s.data = []*Task{}
		return nil
	}
	if err != nil {
		return fmt.Errorf("tasks: read %s: %w", s.path, err)
	}
	return json.Unmarshal(data, &s.data)
}

func (s *Store) save() error {
	data, err := json.MarshalIndent(s.data, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, data, 0644)
}
