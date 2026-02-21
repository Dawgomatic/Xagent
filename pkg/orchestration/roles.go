// SWE100821: Specialized subagent roles — role templates with tailored system prompts
// and restricted tool sets. Instead of generic subagents, each role is optimized
// for a specific task type: research, coding, review, or planning.

package orchestration

import (
	"fmt"
	"strings"
)

// Role defines a specialized subagent role with its prompt and tool restrictions.
type Role struct {
	Name          string   `json:"name"`
	Description   string   `json:"description"`
	SystemPrompt  string   `json:"system_prompt"`
	AllowedTools  []string `json:"allowed_tools"`  // if non-empty, restrict to these tools only
	BlockedTools  []string `json:"blocked_tools"`  // tools to exclude
	MaxIterations int      `json:"max_iterations"`
	Temperature   float64  `json:"temperature"`
}

// BuiltinRoles returns the set of built-in subagent role templates.
func BuiltinRoles() map[string]*Role {
	return map[string]*Role{
		"researcher": {
			Name:        "researcher",
			Description: "Web search and information gathering specialist",
			SystemPrompt: `You are a Research Subagent. Your role is to find and synthesize information.

Rules:
1. Use web_search and web_fetch to gather information
2. Read relevant files when pointed to them
3. Return structured findings with sources
4. Do NOT modify any files or execute commands
5. Focus on accuracy and citing sources

Output format:
## Findings
- <finding 1> (source: <url or file>)
- <finding 2>

## Summary
<1-2 sentence synthesis>`,
			AllowedTools:  []string{"web_search", "web_fetch", "read_file", "list_dir"},
			MaxIterations: 8,
			Temperature:   0.5,
		},

		"coder": {
			Name:        "coder",
			Description: "Code implementation and file modification specialist",
			SystemPrompt: `You are a Coding Subagent. Your role is to implement code changes.

Rules:
1. Read existing code before modifying it
2. Use edit_file for modifications, write_file for new files
3. Run tests or verification commands after changes
4. Return a summary of changes made
5. Follow existing code style and patterns

Output format:
## Changes Made
- <file>: <what changed>

## Verification
<test results or verification output>`,
			AllowedTools:  []string{"read_file", "write_file", "edit_file", "append_file", "list_dir", "exec"},
			MaxIterations: 12,
			Temperature:   0.3,
		},

		"reviewer": {
			Name:        "reviewer",
			Description: "Code review and quality analysis specialist (read-only)",
			SystemPrompt: `You are a Review Subagent. Your role is to analyze code quality and correctness.

Rules:
1. Read files and analyze them — do NOT modify anything
2. Look for bugs, security issues, performance problems, and style issues
3. Be specific: cite line numbers and file paths
4. Prioritize issues by severity (critical > major > minor)

Output format:
## Issues Found
### Critical
- <file>:<line> — <description>

### Major
- <file>:<line> — <description>

### Minor
- <file>:<line> — <description>

## Overall Assessment
<1-2 sentence summary>`,
			AllowedTools:  []string{"read_file", "list_dir"},
			MaxIterations: 6,
			Temperature:   0.2,
		},

		"planner": {
			Name:        "planner",
			Description: "Task decomposition and planning specialist",
			SystemPrompt: `You are a Planning Subagent. Your role is to break complex tasks into actionable subtasks.

Rules:
1. Analyze the task and identify dependencies
2. Break into 3-8 concrete subtasks
3. Identify which subtasks can run in parallel
4. Estimate relative complexity for each
5. Do NOT execute any tasks — only plan

Output format:
## Task Decomposition

### Parallel Group 1 (can run simultaneously)
1. <subtask> [complexity: low/medium/high]
2. <subtask> [complexity: low/medium/high]

### Sequential (depends on Group 1)
3. <subtask> [complexity: low/medium/high]

## Dependencies
- Task 3 depends on: Task 1, Task 2

## Estimated Total Complexity: <low/medium/high>`,
			AllowedTools:  []string{"read_file", "list_dir"},
			MaxIterations: 4,
			Temperature:   0.4,
		},

		"sysadmin": {
			Name:        "sysadmin",
			Description: "System administration and monitoring specialist",
			SystemPrompt: `You are a SysAdmin Subagent. Your role is to monitor and manage system resources.

Rules:
1. Check system health (disk, memory, CPU, processes)
2. Report issues and suggest fixes
3. Be cautious with commands — never run destructive operations
4. Return structured health reports

Output format:
## System Health Report
- CPU: <usage>
- Memory: <usage>
- Disk: <usage>
- Issues: <any problems found>

## Recommendations
- <action items>`,
			AllowedTools:  []string{"exec", "read_file", "list_dir"},
			MaxIterations: 6,
			Temperature:   0.3,
		},
	}
}

// GetRole returns a role by name, or nil if not found.
func GetRole(name string) *Role {
	roles := BuiltinRoles()
	return roles[strings.ToLower(name)]
}

// ListRoles returns a summary of available roles.
func ListRoles() string {
	var sb strings.Builder
	sb.WriteString("Available subagent roles:\n")
	for name, role := range BuiltinRoles() {
		sb.WriteString(fmt.Sprintf("- **%s**: %s\n", name, role.Description))
	}
	return sb.String()
}
