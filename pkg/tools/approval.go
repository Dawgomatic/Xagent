// SWE100821: Tool call approval gate — requires user confirmation for destructive operations.
// When enabled, destructive tools (exec, write_file, edit_file) send a confirmation
// prompt to the user before execution. Useful for autonomous channels (Telegram, Discord).

package tools

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/Dawgomatic/Xagent/pkg/logger"
)

// ApprovalCallback sends a confirmation prompt to the user and waits for response.
// Returns true if approved, false if denied or timed out.
type ApprovalCallback func(ctx context.Context, channel, chatID, prompt string) bool

// ApprovalGate manages tool call approval for destructive operations.
type ApprovalGate struct {
	enabled        bool
	destructive    map[string]bool
	callback       ApprovalCallback
	autoApprove    map[string]time.Time // recently-approved patterns get auto-approved
	mu             sync.RWMutex
	approvalWindow time.Duration // how long an approval pattern stays cached
}

// NewApprovalGate creates a new approval gate with default destructive tool list.
func NewApprovalGate() *ApprovalGate {
	return &ApprovalGate{
		enabled: false,
		destructive: map[string]bool{
			"exec":       true,
			"write_file": true,
			"edit_file":  true,
			"append_file": true,
		},
		autoApprove:    make(map[string]time.Time),
		approvalWindow: 5 * time.Minute,
	}
}

// Enable turns on the approval gate.
func (ag *ApprovalGate) Enable() {
	ag.mu.Lock()
	defer ag.mu.Unlock()
	ag.enabled = true
}

// Disable turns off the approval gate.
func (ag *ApprovalGate) Disable() {
	ag.mu.Lock()
	defer ag.mu.Unlock()
	ag.enabled = false
}

// SetCallback sets the function that prompts the user for approval.
func (ag *ApprovalGate) SetCallback(cb ApprovalCallback) {
	ag.mu.Lock()
	defer ag.mu.Unlock()
	ag.callback = cb
}

// SetDestructiveTools overrides which tools require approval.
func (ag *ApprovalGate) SetDestructiveTools(tools []string) {
	ag.mu.Lock()
	defer ag.mu.Unlock()
	ag.destructive = make(map[string]bool, len(tools))
	for _, t := range tools {
		ag.destructive[t] = true
	}
}

// CheckApproval returns a MiddlewareHook that gates destructive tools.
func (ag *ApprovalGate) CheckApproval(channel, chatID string) MiddlewareHook {
	return func(toolName string, args map[string]interface{}, _ *ToolResult) *ToolResult {
		ag.mu.RLock()
		enabled := ag.enabled
		isDestructive := ag.destructive[toolName]
		cb := ag.callback
		ag.mu.RUnlock()

		if !enabled || !isDestructive || cb == nil {
			return nil // allow execution
		}

		// SWE100821: Check if this tool was recently approved (avoid spamming user)
		approvalKey := fmt.Sprintf("%s:%s:%s", channel, chatID, toolName)
		ag.mu.RLock()
		if t, ok := ag.autoApprove[approvalKey]; ok && time.Now().Before(t) {
			ag.mu.RUnlock()
			return nil // auto-approved
		}
		ag.mu.RUnlock()

		prompt := ag.formatPrompt(toolName, args)

		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		approved := cb(ctx, channel, chatID, prompt)
		if !approved {
			logger.InfoCF("approval", "Tool call denied by user",
				map[string]interface{}{"tool": toolName})
			return ErrorResult(fmt.Sprintf("Tool '%s' was denied by user approval gate.", toolName))
		}

		// SWE100821: Cache approval for this tool for the window duration
		ag.mu.Lock()
		ag.autoApprove[approvalKey] = time.Now().Add(ag.approvalWindow)
		ag.mu.Unlock()

		return nil // approved, allow execution
	}
}

func (ag *ApprovalGate) formatPrompt(toolName string, args map[string]interface{}) string {
	switch toolName {
	case "exec":
		cmd, _ := args["command"].(string)
		return fmt.Sprintf(" Approval needed: execute command\n`%s`\nApprove? (yes/no)", cmd)
	case "write_file":
		path, _ := args["path"].(string)
		return fmt.Sprintf(" Approval needed: write file\n`%s`\nApprove? (yes/no)", path)
	case "edit_file":
		path, _ := args["path"].(string)
		return fmt.Sprintf(" Approval needed: edit file\n`%s`\nApprove? (yes/no)", path)
	default:
		return fmt.Sprintf(" Approval needed: %s\nApprove? (yes/no)", toolName)
	}
}
