// SWE100821: Fallback sandbox for non-Linux platforms.
// Provides the same interface but uses basic process isolation (no namespaces).
//
//go:build !linux

package sandbox

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"time"
)

// NamespaceSandbox is a fallback for non-Linux platforms.
// Uses basic timeout/process isolation without namespace support.
type NamespaceSandbox struct {
	workspace     string
	enableNetwork bool
	enablePID     bool
	maxMemoryMB   int
	maxCPUPercent int
	timeout       time.Duration
	readOnlyPaths []string
}

// NewNamespaceSandbox creates a basic sandbox without namespace support.
func NewNamespaceSandbox(workspace string) *NamespaceSandbox {
	return &NamespaceSandbox{
		workspace: workspace,
		timeout:   60 * time.Second,
	}
}

// SetNetworkEnabled is a no-op on non-Linux.
func (ns *NamespaceSandbox) SetNetworkEnabled(enabled bool) {
	ns.enableNetwork = enabled
}

// SetMemoryLimit is a no-op on non-Linux.
func (ns *NamespaceSandbox) SetMemoryLimit(mb int) {
	ns.maxMemoryMB = mb
}

// SetTimeout sets the execution timeout.
func (ns *NamespaceSandbox) SetTimeout(d time.Duration) {
	ns.timeout = d
}

// Execute runs a command with basic timeout isolation.
func (ns *NamespaceSandbox) Execute(ctx context.Context, command, workDir string) (stdout, stderr string, exitCode int, err error) {
	if workDir == "" {
		workDir = ns.workspace
	}

	cmdCtx, cancel := context.WithTimeout(ctx, ns.timeout)
	defer cancel()

	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.CommandContext(cmdCtx, "powershell", "-NoProfile", "-NonInteractive", "-Command", command)
	} else {
		cmd = exec.CommandContext(cmdCtx, "sh", "-c", command)
	}
	cmd.Dir = workDir

	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf

	runErr := cmd.Run()

	stdout = outBuf.String()
	stderr = errBuf.String()

	if runErr != nil {
		if cmdCtx.Err() == context.DeadlineExceeded {
			return stdout, stderr, -1, fmt.Errorf("command timed out after %v", ns.timeout)
		}
		if exitErr, ok := runErr.(*exec.ExitError); ok {
			return stdout, stderr, exitErr.ExitCode(), nil
		}
		return stdout, stderr, -1, runErr
	}

	return stdout, stderr, 0, nil
}

// ExecuteWithLimits is identical to Execute on non-Linux.
func (ns *NamespaceSandbox) ExecuteWithLimits(ctx context.Context, command, workDir string) (stdout, stderr string, exitCode int, err error) {
	return ns.Execute(ctx, command, workDir)
}

// CreateJail is a no-op on non-Linux. Returns a no-op cleanup function.
func (ns *NamespaceSandbox) CreateJail(jailDir string) (cleanup func(), err error) {
	return func() {}, fmt.Errorf("jail not supported on %s", runtime.GOOS)
}

// IsAvailable returns false on non-Linux platforms.
func IsAvailable() bool {
	return false
}
