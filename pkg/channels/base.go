package channels

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/Dawgomatic/Xagent/pkg/bus"
	"github.com/Dawgomatic/Xagent/pkg/logger"
)

// genRequestID generates a short unique request ID for log tracing.
// SWE100821: 8 hex chars = 4 bytes = enough for per-message uniqueness.
func genRequestID() string {
	b := make([]byte, 4)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// SWE100821: Per-user rate limiting to prevent abuse/cost overrun
const (
	rateLimitWindow  = 60 * time.Second // 1-minute sliding window
	rateLimitMaxMsgs = 10               // max messages per user per window
)

type Channel interface {
	Name() string
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	Send(ctx context.Context, msg bus.OutboundMessage) error
	IsRunning() bool
	IsAllowed(senderID string) bool
}

type BaseChannel struct {
	config    interface{}
	bus       *bus.MessageBus
	running   bool
	name      string
	allowList []string
	// SWE100821: Rate limiter - tracks message timestamps per sender
	rateMu  sync.Mutex
	rateMap map[string][]time.Time
}

func NewBaseChannel(name string, config interface{}, bus *bus.MessageBus, allowList []string) *BaseChannel {
	return &BaseChannel{
		config:    config,
		bus:       bus,
		name:      name,
		allowList: allowList,
		running:   false,
		rateMap:   make(map[string][]time.Time),
	}
}

// isRateLimited returns true if the sender has exceeded the rate limit.
// Uses a sliding window of rateLimitWindow with rateLimitMaxMsgs max messages.
// SWE100821: Prevents single user from burning GPU/API credits.
func (c *BaseChannel) isRateLimited(senderID string) bool {
	c.rateMu.Lock()
	defer c.rateMu.Unlock()

	now := time.Now()
	cutoff := now.Add(-rateLimitWindow)

	// Prune old entries
	times := c.rateMap[senderID]
	fresh := times[:0]
	for _, t := range times {
		if t.After(cutoff) {
			fresh = append(fresh, t)
		}
	}

	if len(fresh) >= rateLimitMaxMsgs {
		c.rateMap[senderID] = fresh
		return true
	}

	c.rateMap[senderID] = append(fresh, now)
	return false
}

func (c *BaseChannel) Name() string {
	return c.name
}

func (c *BaseChannel) IsRunning() bool {
	return c.running
}

func (c *BaseChannel) IsAllowed(senderID string) bool {
	if len(c.allowList) == 0 {
		return true
	}

	// Extract parts from compound senderID like "123456|username"
	idPart := senderID
	userPart := ""
	if idx := strings.Index(senderID, "|"); idx > 0 {
		idPart = senderID[:idx]
		userPart = senderID[idx+1:]
	}

	for _, allowed := range c.allowList {
		// Strip leading "@" from allowed value for username matching
		trimmed := strings.TrimPrefix(allowed, "@")
		allowedID := trimmed
		allowedUser := ""
		if idx := strings.Index(trimmed, "|"); idx > 0 {
			allowedID = trimmed[:idx]
			allowedUser = trimmed[idx+1:]
		}

		// Support either side using "id|username" compound form.
		// This keeps backward compatibility with legacy Telegram allowlist entries.
		if senderID == allowed ||
			idPart == allowed ||
			senderID == trimmed ||
			idPart == trimmed ||
			idPart == allowedID ||
			(allowedUser != "" && senderID == allowedUser) ||
			(userPart != "" && (userPart == allowed || userPart == trimmed || userPart == allowedUser)) {
			return true
		}
	}

	return false
}

func (c *BaseChannel) HandleMessage(senderID, chatID, content string, media []string, metadata map[string]string) {
	if !c.IsAllowed(senderID) {
		return
	}

	// SWE100821: Rate limit check — send feedback instead of silent drop.
	// Previously rate-limited messages were silently discarded, making the bot
	// appear unresponsive to the user.
	if c.isRateLimited(senderID) {
		logger.WarnCF(c.name, "Rate limited", map[string]interface{}{
			"sender_id": senderID,
			"chat_id":   chatID,
		})
		if c.bus != nil {
			c.bus.PublishOutbound(bus.OutboundMessage{
				Channel: c.name,
				ChatID:  chatID,
				Content: "I'm processing your previous message. Please wait a moment before sending another.",
			})
		}
		return
	}

	sessionKey := fmt.Sprintf("%s:%s", c.name, chatID)
	c.publishInbound(senderID, chatID, sessionKey, content, media, metadata)
}

// HandleMessageWithSessionKey publishes an inbound message with an explicit session key for history.
// SWE100821: Discord uses this for session_key_mode guild_user (shared history across text channels).
func (c *BaseChannel) HandleMessageWithSessionKey(senderID, chatID, sessionKey, content string, media []string, metadata map[string]string) {
	if !c.IsAllowed(senderID) {
		return
	}

	if c.isRateLimited(senderID) {
		logger.WarnCF(c.name, "Rate limited", map[string]interface{}{
			"sender_id": senderID,
			"chat_id":   chatID,
		})
		if c.bus != nil {
			c.bus.PublishOutbound(bus.OutboundMessage{
				Channel: c.name,
				ChatID:  chatID,
				Content: "I'm processing your previous message. Please wait a moment before sending another.",
			})
		}
		return
	}

	c.publishInbound(senderID, chatID, sessionKey, content, media, metadata)
}

func (c *BaseChannel) publishInbound(senderID, chatID, sessionKey, content string, media []string, metadata map[string]string) {
	msg := bus.InboundMessage{
		RequestID:  genRequestID(), // SWE100821: trace ID for log correlation
		Channel:    c.name,
		SenderID:   senderID,
		ChatID:     chatID,
		Content:    content,
		Media:      media,
		SessionKey: sessionKey,
		Metadata:   metadata,
	}

	c.bus.PublishInbound(msg)
}

func (c *BaseChannel) setRunning(running bool) {
	c.running = running
}
