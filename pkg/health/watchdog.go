// SWE100821: Subsystem watchdog — monitors all xagent subsystems, reports status,
// and attempts auto-recovery for external services (Ollama, Qdrant).
// Exposes status via GetStatus() for the dashboard /api/watchdog endpoint.
package health

import (
	"context"
	"fmt"
	"net/http"
	"os/exec"
	"sync"
	"time"

	"github.com/Dawgomatic/Xagent/pkg/logger"
)

// SubsystemStatus represents the health state of a single subsystem.
type SubsystemStatus struct {
	Name      string    `json:"name"`
	Status    string    `json:"status"` // "up", "down", "degraded", "unknown"
	Message   string    `json:"message,omitempty"`
	LastCheck time.Time `json:"last_check"`
	Since     time.Time `json:"since"`
	Restarts  int       `json:"restarts"`
}

// SubsystemChecker probes a subsystem and returns nil if healthy.
type SubsystemChecker struct {
	Name    string
	Check   func(ctx context.Context) error
	Recover func(ctx context.Context) error // nil = no auto-recovery
}

// Watchdog periodically checks all registered subsystems.
type Watchdog struct {
	mu       sync.RWMutex
	checkers []SubsystemChecker
	statuses map[string]*SubsystemStatus
	interval time.Duration
	cancel   context.CancelFunc
}

// NewWatchdog creates a watchdog with the given check interval.
func NewWatchdog(interval time.Duration) *Watchdog {
	return &Watchdog{
		statuses: make(map[string]*SubsystemStatus),
		interval: interval,
	}
}

// Register adds a subsystem to monitor.
func (w *Watchdog) Register(c SubsystemChecker) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.checkers = append(w.checkers, c)
	w.statuses[c.Name] = &SubsystemStatus{
		Name:   c.Name,
		Status: "unknown",
		Since:  time.Now(),
	}
}

// Start begins the watchdog loop. Call Stop() to terminate.
func (w *Watchdog) Start() {
	ctx, cancel := context.WithCancel(context.Background())
	w.cancel = cancel

	// SWE100821: Run initial check immediately, then tick
	go func() {
		w.runChecks(ctx)
		ticker := time.NewTicker(w.interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				w.runChecks(ctx)
			}
		}
	}()
}

// Stop terminates the watchdog loop.
func (w *Watchdog) Stop() {
	if w.cancel != nil {
		w.cancel()
	}
}

// GetStatus returns a snapshot of all subsystem statuses.
func (w *Watchdog) GetStatus() []SubsystemStatus {
	w.mu.RLock()
	defer w.mu.RUnlock()
	result := make([]SubsystemStatus, 0, len(w.checkers))
	for _, c := range w.checkers {
		if s, ok := w.statuses[c.Name]; ok {
			result = append(result, *s)
		}
	}
	return result
}

// AllHealthy returns true if every subsystem reports "up".
func (w *Watchdog) AllHealthy() bool {
	w.mu.RLock()
	defer w.mu.RUnlock()
	for _, s := range w.statuses {
		if s.Status != "up" {
			return false
		}
	}
	return true
}

func (w *Watchdog) runChecks(ctx context.Context) {
	for _, c := range w.checkers {
		checkCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		err := c.Check(checkCtx)
		cancel()

		w.mu.Lock()
		s := w.statuses[c.Name]
		prevStatus := s.Status
		s.LastCheck = time.Now()

		if err == nil {
			if prevStatus != "up" {
				s.Since = time.Now()
			}
			s.Status = "up"
			s.Message = ""
		} else {
			if prevStatus == "up" || prevStatus == "unknown" {
				s.Since = time.Now()
			}
			s.Status = "down"
			s.Message = err.Error()

			logger.WarnCF("watchdog", fmt.Sprintf("Subsystem %s is down", c.Name),
				map[string]interface{}{"error": err.Error()})

			// SWE100821: Attempt auto-recovery if a Recover func is registered
			if c.Recover != nil {
				s.Restarts++
				w.mu.Unlock()
				recoverCtx, rCancel := context.WithTimeout(ctx, 30*time.Second)
				if rErr := c.Recover(recoverCtx); rErr != nil {
					logger.ErrorCF("watchdog", fmt.Sprintf("Recovery failed for %s", c.Name),
						map[string]interface{}{"error": rErr.Error()})
				} else {
					logger.InfoCF("watchdog", fmt.Sprintf("Recovery succeeded for %s", c.Name), nil)
				}
				rCancel()
				continue
			}
		}
		w.mu.Unlock()
	}
}

// --- Built-in checkers for common subsystems ---

// HTTPChecker pings a URL and expects a 2xx or 3xx status.
func HTTPChecker(name, url string, recover func(ctx context.Context) error) SubsystemChecker {
	return SubsystemChecker{
		Name: name,
		Check: func(ctx context.Context) error {
			req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
			if err != nil {
				return err
			}
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				return fmt.Errorf("unreachable: %w", err)
			}
			resp.Body.Close()
			if resp.StatusCode >= 400 {
				return fmt.Errorf("status %d", resp.StatusCode)
			}
			return nil
		},
		Recover: recover,
	}
}

// SystemdRecover returns a recovery func that restarts a systemd unit.
func SystemdRecover(unit string) func(ctx context.Context) error {
	return func(ctx context.Context) error {
		cmd := exec.CommandContext(ctx, "sudo", "systemctl", "restart", unit)
		return cmd.Run()
	}
}

// ProcessChecker verifies a process name is running via pgrep.
func ProcessChecker(name, processName string) SubsystemChecker {
	return SubsystemChecker{
		Name: name,
		Check: func(ctx context.Context) error {
			cmd := exec.CommandContext(ctx, "pgrep", "-x", processName)
			if err := cmd.Run(); err != nil {
				return fmt.Errorf("process %s not running", processName)
			}
			return nil
		},
	}
}

// CallbackChecker wraps a simple health-check function.
func CallbackChecker(name string, fn func() error) SubsystemChecker {
	return SubsystemChecker{
		Name: name,
		Check: func(ctx context.Context) error {
			return fn()
		},
	}
}
