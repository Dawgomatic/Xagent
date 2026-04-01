// time_tool.go provides the "get_time" agent tool — returns current system time,
// date, UTC offset, Unix timestamp, and uptime so the agent can reason about
// scheduling, deadlines, and temporal context without shelling out.
package tools

import (
	"context"
	"fmt"
	"time"
)

// TimeTool returns current system time in multiple formats.
// SWE100821: Added for agent temporal awareness and self-improve scheduling.
type TimeTool struct {
	startTime time.Time
}

// NewTimeTool creates a TimeTool anchored to now (process start approximation).
// Pass a fixed startTime for accurate uptime; zero value uses tool-creation time.
func NewTimeTool(startTime time.Time) *TimeTool {
	if startTime.IsZero() {
		startTime = time.Now()
	}
	return &TimeTool{startTime: startTime}
}

func (t *TimeTool) Name() string        { return "get_time" }
func (t *TimeTool) Description() string {
	return "Returns the current system time (UTC + local), date, Unix timestamp, timezone, and process uptime. Use this whenever you need to know what time or date it is."
}

func (t *TimeTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type":       "object",
		"properties": map[string]interface{}{},
		"required":   []string{},
	}
}

// Execute returns a structured time snapshot — no args required.
func (t *TimeTool) Execute(_ context.Context, _ map[string]interface{}) *ToolResult {
	now := time.Now()
	utc := now.UTC()
	local := now.Local()
	zone, offsetSecs := local.Zone()
	offsetHours := float64(offsetSecs) / 3600.0

	uptime := time.Since(t.startTime).Truncate(time.Second)

	out := fmt.Sprintf(
		"Current time:\n"+
			"  UTC:       %s\n"+
			"  Local:     %s\n"+
			"  Timezone:  %s (UTC%+.1f)\n"+
			"  Unix:      %d\n"+
			"  Date:      %s\n"+
			"  Weekday:   %s\n"+
			"  Uptime:    %s",
		utc.Format("2006-01-02 15:04:05 UTC"),
		local.Format("2006-01-02 15:04:05 MST"),
		zone, offsetHours,
		utc.Unix(),
		utc.Format("2006-01-02"),
		utc.Weekday().String(),
		uptime.String(),
	)
	return NewToolResult(out)
}
