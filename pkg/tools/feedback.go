// Xagent - Ultra-lightweight personal AI agent
// OpenClaw-RL integration: User feedback tool for RL reward signals
// License: MIT
//
// Copyright (c) 2026 Xagent contributors

package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// FeedbackEntry represents a single piece of user feedback.
type FeedbackEntry struct {
	Timestamp time.Time `json:"timestamp"`
	SessionID string    `json:"session_id"`
	Channel   string    `json:"channel"`
	Rating    int       `json:"rating"`     // +1 (good), -1 (bad), 0 (neutral)
	Comment   string    `json:"comment"`    // Optional text feedback
	TurnIndex int       `json:"turn_index"` // Which turn this feedback applies to (-1 = last)
}

// FeedbackTool allows users to provide explicit feedback on agent responses.
// When connected to an OpenClaw-RL server, this feedback improves the RL reward signal.
type FeedbackTool struct {
	workspace string
	mu        sync.Mutex
	channel   string
	chatID    string
	sessionID string

	// In-memory buffer of recent feedback for context injection
	recentFeedback []FeedbackEntry
}

func NewFeedbackTool(workspace string) *FeedbackTool {
	return &FeedbackTool{
		workspace:      workspace,
		recentFeedback: make([]FeedbackEntry, 0),
	}
}

func (t *FeedbackTool) Name() string {
	return "feedback"
}

func (t *FeedbackTool) Description() string {
	return "Record user feedback on the agent's responses. Accepts thumbs up/down or text feedback to improve future responses."
}

func (t *FeedbackTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"rating": map[string]interface{}{
				"type":        "string",
				"description": "User's rating: 'good' (👍), 'bad' (👎), or 'neutral'",
				"enum":        []string{"good", "bad", "neutral"},
			},
			"comment": map[string]interface{}{
				"type":        "string",
				"description": "Optional text feedback explaining what was good or bad",
			},
			"turn_index": map[string]interface{}{
				"type":        "integer",
				"description": "Which turn to rate (0-indexed). Defaults to -1 (most recent turn)",
			},
		},
		"required": []string{"rating"},
	}
}

func (t *FeedbackTool) SetContext(channel, chatID string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.channel = channel
	t.chatID = chatID
}

// SetSessionID sets the current session ID for feedback tracking.
func (t *FeedbackTool) SetSessionID(sessionID string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.sessionID = sessionID
}

func (t *FeedbackTool) Execute(ctx context.Context, args map[string]interface{}) *ToolResult {
	ratingStr, _ := args["rating"].(string)
	comment, _ := args["comment"].(string)
	turnIndex := -1
	if ti, ok := args["turn_index"].(float64); ok {
		turnIndex = int(ti)
	}

	// Parse rating
	var rating int
	switch strings.ToLower(ratingStr) {
	case "good", "thumbs_up", "👍", "positive", "+1":
		rating = 1
	case "bad", "thumbs_down", "👎", "negative", "-1":
		rating = -1
	default:
		rating = 0
	}

	t.mu.Lock()
	entry := FeedbackEntry{
		Timestamp: time.Now(),
		SessionID: t.sessionID,
		Channel:   t.channel,
		Rating:    rating,
		Comment:   comment,
		TurnIndex: turnIndex,
	}

	// Keep last 50 feedback entries in memory
	t.recentFeedback = append(t.recentFeedback, entry)
	if len(t.recentFeedback) > 50 {
		t.recentFeedback = t.recentFeedback[len(t.recentFeedback)-50:]
	}
	t.mu.Unlock()

	// Persist to JSONL file
	if err := t.appendToLog(entry); err != nil {
		return &ToolResult{
			ForLLM:  fmt.Sprintf("Feedback recorded in memory but failed to save to disk: %v", err),
			IsError: true,
			Err:     err,
		}
	}

	ratingEmoji := "😐"
	if rating == 1 {
		ratingEmoji = "👍"
	} else if rating == -1 {
		ratingEmoji = "👎"
	}

	response := fmt.Sprintf("Feedback recorded: %s", ratingEmoji)
	if comment != "" {
		response += fmt.Sprintf(" — \"%s\"", comment)
	}

	return &ToolResult{
		ForLLM:  response + "\nThis feedback will be used to improve future responses via reinforcement learning.",
		ForUser: response,
	}
}

func (t *FeedbackTool) appendToLog(entry FeedbackEntry) error {
	logDir := filepath.Join(t.workspace, "feedback")
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return err
	}

	logFile := filepath.Join(logDir, "feedback.jsonl")
	f, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	data, err := json.Marshal(entry)
	if err != nil {
		return err
	}

	_, err = f.Write(append(data, '\n'))
	return err
}

// GetRecentFeedback returns recent feedback entries for context injection.
func (t *FeedbackTool) GetRecentFeedback() []FeedbackEntry {
	t.mu.Lock()
	defer t.mu.Unlock()

	result := make([]FeedbackEntry, len(t.recentFeedback))
	copy(result, t.recentFeedback)
	return result
}

// GetFeedbackSummary returns a text summary of recent feedback for system prompt injection.
func (t *FeedbackTool) GetFeedbackSummary() string {
	t.mu.Lock()
	defer t.mu.Unlock()

	if len(t.recentFeedback) == 0 {
		return ""
	}

	// Count recent sentiment
	positive, negative, neutral := 0, 0, 0
	var recentComments []string
	for _, f := range t.recentFeedback {
		switch f.Rating {
		case 1:
			positive++
		case -1:
			negative++
		default:
			neutral++
		}
		if f.Comment != "" {
			recentComments = append(recentComments, f.Comment)
		}
	}

	summary := fmt.Sprintf("## User Feedback Summary\nRecent ratings: %d positive, %d negative, %d neutral\n", positive, negative, neutral)

	// Include last 3 comments
	if len(recentComments) > 0 {
		summary += "Recent feedback comments:\n"
		start := len(recentComments) - 3
		if start < 0 {
			start = 0
		}
		for _, c := range recentComments[start:] {
			summary += fmt.Sprintf("- %s\n", c)
		}
	}

	return summary
}
