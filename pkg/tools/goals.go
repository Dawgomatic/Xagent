// Package tools: goals.go implements the goals tool — reads/writes GOALS.md in the workspace
// for lightweight project/goal tracking (list, add, update status, review summary).
// Reused: pkg/tools/base.go L5-L11 (Tool interface), pkg/tools/result.go L35-L45, L87-L100 (NewToolResult, ErrorResult).
package tools

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// GoalsTool reads and updates GOALS.md under the agent workspace (EXA Goals format).
type GoalsTool struct {
	workspace string // SWE100821: root directory containing GOALS.md
}

// NewGoalsTool returns a goals tracker scoped to workspace.
func NewGoalsTool(workspace string) *GoalsTool {
	return &GoalsTool{workspace: workspace}
}

func (t *GoalsTool) Name() string { return "goals" }

func (t *GoalsTool) Description() string {
	return "Track goals in GOALS.md: list (read file), add (new active goal with priority), update (change status by goal title), review (counts and summary)."
}

func (t *GoalsTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"action": map[string]interface{}{
				"type":        "string",
				"enum":        []string{"list", "add", "update", "review"},
				"description": "list | add | update | review",
			},
			"goal": map[string]interface{}{
				"type":        "string",
				"description": "Goal description text (required for add).",
			},
			"priority": map[string]interface{}{
				"type":        "string",
				"enum":        []string{"high", "medium", "low"},
				"description": "Priority for add (default: medium).",
			},
			"status": map[string]interface{}{
				"type":        "string",
				"enum":        []string{"active", "completed", "paused", "abandoned"},
				"description": "New status for update.",
			},
			"goal_id": map[string]interface{}{
				"type":        "string",
				"description": "Goal title to match for update (same as the description text in GOALS.md).",
			},
		},
		"required": []string{"action"},
	}
}

func (t *GoalsTool) Execute(ctx context.Context, args map[string]interface{}) *ToolResult {
	if err := ctx.Err(); err != nil {
		return ErrorResult("cancelled").WithError(err)
	}
	action, _ := args["action"].(string)
	action = strings.TrimSpace(strings.ToLower(action))
	switch action {
	case "list":
		return t.doList()
	case "add":
		return t.doAdd(ctx, args)
	case "update":
		return t.doUpdate(ctx, args)
	case "review":
		return t.doReview()
	default:
		return ErrorResult(`action must be one of: list, add, update, review`)
	}
}

// SWE100821: checkbox char inside brackets: active ' ', completed 'x', paused '~', abandoned '-'
var goalLineRE = regexp.MustCompile(`^-\s+\[([ x~-])\]\s*(.+)$`)

var priorityFromMetaRE = regexp.MustCompile(`(?i)priority:\s*(\w+)`)

var goalsSectionOrder = []string{"Active", "Completed", "Paused", "Abandoned"}

type parsedGoals struct {
	titleBlock []string
	sections   map[string][]string
}

func (t *GoalsTool) goalsPath() string {
	return filepath.Join(t.workspace, "GOALS.md")
}

func defaultGoalsMarkdown() string {
	return strings.TrimSpace(`# EXA Goals

## Active

## Completed

## Paused

## Abandoned`) + "\n"
}

// parseGoalsMD splits GOALS.md into a title block and per-section goal bullet lines.
// Side effects: none. Inputs: raw file content. Outputs: parsedGoals for render/update.
func parseGoalsMD(content string) *parsedGoals {
	d := &parsedGoals{
		sections: map[string][]string{
			"Active": {}, "Completed": {}, "Paused": {}, "Abandoned": {},
		},
	}
	content = strings.ReplaceAll(content, "\r\n", "\n")
	lines := strings.Split(content, "\n")
	phase := "title"
	cur := ""
	for _, line := range lines {
		if strings.HasPrefix(line, "## ") {
			name := strings.TrimSpace(strings.TrimPrefix(line, "## "))
			if _, ok := d.sections[name]; ok {
				phase = "body"
				cur = name
				continue
			}
		}
		switch phase {
		case "title":
			d.titleBlock = append(d.titleBlock, line)
		case "body":
			if cur != "" && strings.HasPrefix(strings.TrimSpace(line), "- ") {
				d.sections[cur] = append(d.sections[cur], line)
			}
		}
	}
	return d
}

// renderGoalsMD writes canonical EXA Goals markdown from parsedGoals.
func (d *parsedGoals) renderGoalsMD() string {
	var b strings.Builder
	if len(d.titleBlock) == 0 {
		b.WriteString("# EXA Goals\n\n")
	} else {
		for i, l := range d.titleBlock {
			if i > 0 {
				b.WriteByte('\n')
			}
			b.WriteString(l)
		}
		b.WriteString("\n\n")
	}
	for _, name := range goalsSectionOrder {
		b.WriteString("## ")
		b.WriteString(name)
		b.WriteByte('\n')
		for _, gl := range d.sections[name] {
			b.WriteString(gl)
			b.WriteByte('\n')
		}
		b.WriteByte('\n')
	}
	out := strings.TrimRight(b.String(), "\n")
	if out == "" {
		return defaultGoalsMarkdown()
	}
	return out + "\n"
}

func goalDescFromLine(line string) (desc string, ok bool) {
	m := goalLineRE.FindStringSubmatch(strings.TrimSpace(line))
	if m == nil {
		return "", false
	}
	rest := strings.TrimSpace(m[2])
	if idx := strings.LastIndex(rest, " ("); idx >= 0 && strings.HasSuffix(rest, ")") {
		rest = strings.TrimSpace(rest[:idx])
	}
	return rest, true
}

func extractPriorityFromTail(line string) string {
	m := priorityFromMetaRE.FindStringSubmatch(line)
	if len(m) < 2 {
		return ""
	}
	return strings.ToLower(strings.TrimSpace(m[1]))
}

func goalMatchesID(line, goalID string) bool {
	desc, ok := goalDescFromLine(line)
	if !ok {
		return false
	}
	gid := strings.TrimSpace(goalID)
	if gid == "" {
		return false
	}
	if strings.EqualFold(desc, gid) {
		return true
	}
	return strings.Contains(strings.ToLower(desc), strings.ToLower(gid))
}

func statusToMark(status string) (mark rune, section string, ok bool) {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "active":
		return ' ', "Active", true
	case "completed":
		return 'x', "Completed", true
	case "paused":
		return '~', "Paused", true
	case "abandoned":
		return '-', "Abandoned", true
	default:
		return 0, "", false
	}
}

func metaForLine(mark rune, desc, today string, priority string, oldLine string) string {
	switch mark {
	case ' ':
		p := strings.TrimSpace(priority)
		if p == "" {
			p = extractPriorityFromTail(oldLine)
		}
		if p == "" {
			p = "medium"
		}
		return fmt.Sprintf("priority: %s, added: %s", p, today)
	case 'x':
		return fmt.Sprintf("completed: %s", today)
	case '~':
		return fmt.Sprintf("paused: %s", today)
	case '-':
		return fmt.Sprintf("abandoned: %s", today)
	default:
		return fmt.Sprintf("updated: %s", today)
	}
}

func formatGoalLine(mark rune, desc, meta string) string {
	return fmt.Sprintf("- [%c] %s (%s)", mark, desc, meta)
}

func (t *GoalsTool) readGoalsFile() ([]byte, error) {
	return os.ReadFile(t.goalsPath())
}

func (t *GoalsTool) writeGoalsFile(data string) error {
	dir := filepath.Dir(t.goalsPath())
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	return os.WriteFile(t.goalsPath(), []byte(data), 0644)
}

func (t *GoalsTool) doList() *ToolResult {
	b, err := t.readGoalsFile()
	if err != nil {
		if os.IsNotExist(err) {
			return NewToolResult("GOALS.md does not exist yet. Use action \"add\" to create it with a first goal.")
		}
		return ErrorResult(fmt.Sprintf("read GOALS.md: %v", err))
	}
	return &ToolResult{ForLLM: string(b)}
}

func (t *GoalsTool) doAdd(ctx context.Context, args map[string]interface{}) *ToolResult {
	if err := ctx.Err(); err != nil {
		return ErrorResult("cancelled").WithError(err)
	}
	goal := strings.TrimSpace(argString(args, "goal"))
	if goal == "" {
		return ErrorResult("goal is required for add")
	}
	priority := strings.ToLower(strings.TrimSpace(argString(args, "priority")))
	if priority == "" {
		priority = "medium"
	}
	if priority != "high" && priority != "medium" && priority != "low" {
		return ErrorResult("priority must be high, medium, or low")
	}
	today := time.Now().UTC().Format("2006-01-02")
	path := t.goalsPath()
	var doc *parsedGoals
	if b, err := os.ReadFile(path); err != nil {
		if !os.IsNotExist(err) {
			return ErrorResult(fmt.Sprintf("read GOALS.md: %v", err))
		}
		doc = parseGoalsMD(defaultGoalsMarkdown())
	} else {
		doc = parseGoalsMD(string(b))
	}
	line := formatGoalLine(' ', goal, fmt.Sprintf("priority: %s, added: %s", priority, today))
	doc.sections["Active"] = append(doc.sections["Active"], line)
	out := doc.renderGoalsMD()
	if err := t.writeGoalsFile(out); err != nil {
		return ErrorResult(fmt.Sprintf("write GOALS.md: %v", err))
	}
	return &ToolResult{
		ForLLM:  fmt.Sprintf("Added active goal:\n%s\n\nFull GOALS.md:\n%s", line, out),
		ForUser: fmt.Sprintf("Goal added: %s", goal),
	}
}

func (t *GoalsTool) doUpdate(ctx context.Context, args map[string]interface{}) *ToolResult {
	if err := ctx.Err(); err != nil {
		return ErrorResult("cancelled").WithError(err)
	}
	gid := strings.TrimSpace(argString(args, "goal_id"))
	status := strings.TrimSpace(argString(args, "status"))
	if gid == "" {
		return ErrorResult("goal_id is required for update")
	}
	if status == "" {
		return ErrorResult("status is required for update")
	}
	mark, targetSec, ok := statusToMark(status)
	if !ok {
		return ErrorResult("status must be active, completed, paused, or abandoned")
	}
	b, err := t.readGoalsFile()
	if err != nil {
		if os.IsNotExist(err) {
			return ErrorResult("GOALS.md does not exist; cannot update")
		}
		return ErrorResult(fmt.Sprintf("read GOALS.md: %v", err))
	}
	doc := parseGoalsMD(string(b))
	var foundLine string
	var foundDesc string
	foundFrom := ""
	for _, sec := range goalsSectionOrder {
		lines := doc.sections[sec]
		for i, line := range lines {
			if goalMatchesID(line, gid) {
				foundLine = line
				foundDesc, _ = goalDescFromLine(line)
				foundFrom = sec
				doc.sections[sec] = append(lines[:i], lines[i+1:]...)
				goto found
			}
		}
	}
found:
	if foundLine == "" {
		return ErrorResult(fmt.Sprintf("no goal matched goal_id %q", gid))
	}
	today := time.Now().UTC().Format("2006-01-02")
	meta := metaForLine(mark, foundDesc, today, "", foundLine)
	newLine := formatGoalLine(mark, foundDesc, meta)
	doc.sections[targetSec] = append(doc.sections[targetSec], newLine)
	out := doc.renderGoalsMD()
	if err := t.writeGoalsFile(out); err != nil {
		return ErrorResult(fmt.Sprintf("write GOALS.md: %v", err))
	}
	return &ToolResult{
		ForLLM: fmt.Sprintf("Moved goal from %s to %s as:\n%s\n\nFull GOALS.md:\n%s", foundFrom, targetSec, newLine, out),
	}
}

func (t *GoalsTool) doReview() *ToolResult {
	b, err := t.readGoalsFile()
	if err != nil {
		if os.IsNotExist(err) {
			return NewToolResult("No GOALS.md yet — zero goals across all sections.")
		}
		return ErrorResult(fmt.Sprintf("read GOALS.md: %v", err))
	}
	doc := parseGoalsMD(string(b))
	var sb strings.Builder
	sb.WriteString("Goals summary:\n")
	total := 0
	for _, sec := range goalsSectionOrder {
		n := len(doc.sections[sec])
		total += n
		sb.WriteString(fmt.Sprintf("- %s: %d\n", sec, n))
	}
	sb.WriteString(fmt.Sprintf("Total: %d\n", total))
	if total == 0 {
		return NewToolResult(sb.String())
	}
	sb.WriteString("\nActive items:\n")
	for _, line := range doc.sections["Active"] {
		sb.WriteString(line)
		sb.WriteByte('\n')
	}
	return NewToolResult(sb.String())
}

// argString coerces common JSON argument types to string (Reused pattern: pkg/tools/filesystem.go-style arg reads).
func argString(args map[string]interface{}, key string) string {
	v, ok := args[key]
	if !ok || v == nil {
		return ""
	}
	switch t := v.(type) {
	case string:
		return t
	default:
		return fmt.Sprintf("%v", t)
	}
}
