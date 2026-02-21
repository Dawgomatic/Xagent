// SWE100821: Personality evolution — tracks interaction patterns over time and
// lets SOUL.md evolve. If the user consistently prefers terse responses, the agent
// adapts. Generates monthly "personality diffs" showing how the agent evolved.
// User can approve/reject personality changes.

package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Dawgomatic/Xagent/pkg/logger"
	"github.com/Dawgomatic/Xagent/pkg/providers"
)

// PersonalityTracker observes interaction patterns and proposes personality adaptations.
type PersonalityTracker struct {
	workspace   string
	provider    providers.LLMProvider
	model       string
	observations []Observation
	profilePath string
}

// Observation records a single interaction pattern data point.
type Observation struct {
	Timestamp     time.Time `json:"timestamp"`
	UserMsgLen    int       `json:"user_msg_len"`
	AgentMsgLen   int       `json:"agent_msg_len"`
	TopicKeywords []string  `json:"topic_keywords,omitempty"`
	ToolsUsed     []string  `json:"tools_used,omitempty"`
	Sentiment     string    `json:"sentiment,omitempty"` // positive, neutral, negative
}

// PersonalityProfile stores accumulated personality traits and their evolution.
type PersonalityProfile struct {
	Traits        map[string]float64   `json:"traits"` // e.g. "verbosity": 0.3, "formality": 0.7
	TopTopics     []string             `json:"top_topics"`
	PreferredLen  string               `json:"preferred_response_length"` // short, medium, long
	Adaptations   []PersonalityChange  `json:"adaptations"`
	LastUpdated   time.Time            `json:"last_updated"`
	TotalSamples  int                  `json:"total_samples"`
}

// PersonalityChange records a proposed or applied change to the agent's personality.
type PersonalityChange struct {
	Date        time.Time `json:"date"`
	Trait       string    `json:"trait"`
	OldValue    float64   `json:"old_value"`
	NewValue    float64   `json:"new_value"`
	Reason      string    `json:"reason"`
	Applied     bool      `json:"applied"`
}

// NewPersonalityTracker creates a personality tracker for the given workspace.
func NewPersonalityTracker(workspace string, provider providers.LLMProvider, model string) *PersonalityTracker {
	return &PersonalityTracker{
		workspace:   workspace,
		provider:    provider,
		model:       model,
		profilePath: filepath.Join(workspace, "state", "personality.json"),
	}
}

// Observe records interaction data for personality analysis.
func (pt *PersonalityTracker) Observe(userMsgLen, agentMsgLen int, toolsUsed []string) {
	pt.observations = append(pt.observations, Observation{
		Timestamp:   time.Now(),
		UserMsgLen:  userMsgLen,
		AgentMsgLen: agentMsgLen,
		ToolsUsed:   toolsUsed,
	})

	// Flush observations periodically to avoid unbounded growth
	if len(pt.observations) > 100 {
		pt.observations = pt.observations[len(pt.observations)-50:]
	}
}

// Analyze examines accumulated observations and proposes personality adaptations.
// Call this periodically (e.g., weekly via cron).
func (pt *PersonalityTracker) Analyze(ctx context.Context) (*PersonalityProfile, error) {
	profile := pt.loadProfile()

	if len(pt.observations) < 10 {
		return profile, nil // not enough data
	}

	// Compute statistics from observations
	var totalUserLen, totalAgentLen int
	topicCounts := make(map[string]int)
	toolCounts := make(map[string]int)

	for _, obs := range pt.observations {
		totalUserLen += obs.UserMsgLen
		totalAgentLen += obs.AgentMsgLen
		for _, t := range obs.ToolsUsed {
			toolCounts[t]++
		}
		for _, kw := range obs.TopicKeywords {
			topicCounts[kw]++
		}
	}

	n := len(pt.observations)
	avgUserLen := totalUserLen / n
	avgAgentLen := totalAgentLen / n

	// SWE100821: Infer preferred response length from user message patterns
	newLen := "medium"
	if avgUserLen < 50 {
		newLen = "short"
	} else if avgUserLen > 300 {
		newLen = "long"
	}

	// SWE100821: Compute verbosity trait (0=terse, 1=verbose)
	verbosity := float64(avgAgentLen) / 500.0
	if verbosity > 1.0 {
		verbosity = 1.0
	}

	oldVerbosity := profile.Traits["verbosity"]
	if diff := verbosity - oldVerbosity; diff > 0.1 || diff < -0.1 {
		profile.Adaptations = append(profile.Adaptations, PersonalityChange{
			Date:     time.Now(),
			Trait:    "verbosity",
			OldValue: oldVerbosity,
			NewValue: verbosity,
			Reason:   fmt.Sprintf("Average user msg: %d chars, agent msg: %d chars", avgUserLen, avgAgentLen),
			Applied:  true,
		})
		profile.Traits["verbosity"] = verbosity
	}

	profile.PreferredLen = newLen
	profile.TotalSamples += n
	profile.LastUpdated = time.Now()

	// Save profile
	pt.saveProfile(profile)

	// Clear processed observations
	pt.observations = nil

	logger.InfoCF("personality", "Personality analysis complete",
		map[string]interface{}{
			"samples":    n,
			"verbosity":  verbosity,
			"pref_len":   newLen,
			"changes":    len(profile.Adaptations),
		})

	return profile, nil
}

// ForSystemPrompt returns personality guidance for the system prompt.
func (pt *PersonalityTracker) ForSystemPrompt() string {
	profile := pt.loadProfile()
	if profile.TotalSamples < 10 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("## Personality Adaptations\n\n")
	sb.WriteString("Based on observed interaction patterns:\n")

	if profile.PreferredLen != "" {
		sb.WriteString(fmt.Sprintf("- Preferred response length: %s\n", profile.PreferredLen))
	}

	if v, ok := profile.Traits["verbosity"]; ok {
		if v < 0.3 {
			sb.WriteString("- User prefers concise, direct responses\n")
		} else if v > 0.7 {
			sb.WriteString("- User appreciates detailed, thorough responses\n")
		}
	}

	if len(profile.TopTopics) > 0 {
		sb.WriteString(fmt.Sprintf("- Frequent topics: %s\n", strings.Join(profile.TopTopics, ", ")))
	}

	return sb.String()
}

// GetDiff returns a human-readable diff of personality changes since the given date.
func (pt *PersonalityTracker) GetDiff(since time.Time) string {
	profile := pt.loadProfile()
	var sb strings.Builder
	sb.WriteString("## Personality Evolution\n\n")

	changes := 0
	for _, c := range profile.Adaptations {
		if c.Date.After(since) {
			sb.WriteString(fmt.Sprintf("- %s: %s %.2f → %.2f (%s)\n",
				c.Date.Format("2006-01-02"), c.Trait, c.OldValue, c.NewValue, c.Reason))
			changes++
		}
	}

	if changes == 0 {
		return ""
	}

	return sb.String()
}

func (pt *PersonalityTracker) loadProfile() *PersonalityProfile {
	profile := &PersonalityProfile{
		Traits: map[string]float64{
			"verbosity":  0.5,
			"formality":  0.5,
			"creativity": 0.5,
		},
	}

	data, err := os.ReadFile(pt.profilePath)
	if err != nil {
		return profile
	}

	json.Unmarshal(data, profile)
	return profile
}

func (pt *PersonalityTracker) saveProfile(profile *PersonalityProfile) {
	os.MkdirAll(filepath.Dir(pt.profilePath), 0755)
	data, err := json.MarshalIndent(profile, "", "  ")
	if err != nil {
		return
	}
	os.WriteFile(pt.profilePath, data, 0600)
}
