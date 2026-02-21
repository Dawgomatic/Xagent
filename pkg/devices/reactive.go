// SWE100821: Hardware-reactive behavior — triggers agent actions when hardware
// events occur. USB camera plugged in → offer to analyze photos, USB drive →
// offer to index contents, temperature spike → alert user.
// Builds on the existing device monitoring in pkg/devices/.

package devices

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

// DeviceReaction defines an automatic action for a device event.
type DeviceReaction struct {
	DeviceClass string // USB class or device type pattern
	EventType   string // "add", "remove"
	Action      string // message template for the agent
	Cooldown    time.Duration
	lastFired   time.Time
}

// ReactiveHandler manages hardware-reactive behaviors.
type ReactiveHandler struct {
	reactions []DeviceReaction
	onAction  func(action string) // callback to inject agent action
	mu        sync.RWMutex
}

// NewReactiveHandler creates a handler with default device reactions.
func NewReactiveHandler() *ReactiveHandler {
	return &ReactiveHandler{
		reactions: defaultReactions(),
	}
}

// SetActionCallback registers the function called when a reaction fires.
func (rh *ReactiveHandler) SetActionCallback(fn func(action string)) {
	rh.mu.Lock()
	defer rh.mu.Unlock()
	rh.onAction = fn
}

// AddReaction adds a custom device reaction.
func (rh *ReactiveHandler) AddReaction(r DeviceReaction) {
	rh.mu.Lock()
	defer rh.mu.Unlock()
	rh.reactions = append(rh.reactions, r)
}

// OnDeviceEvent processes a device event and fires matching reactions.
func (rh *ReactiveHandler) OnDeviceEvent(eventType, deviceClass, deviceName string) {
	rh.mu.Lock()
	defer rh.mu.Unlock()

	now := time.Now()
	for i, r := range rh.reactions {
		if r.EventType != eventType {
			continue
		}
		if !matchesClass(deviceClass, r.DeviceClass) {
			continue
		}
		if now.Sub(r.lastFired) < r.Cooldown {
			continue
		}

		rh.reactions[i].lastFired = now

		action := strings.ReplaceAll(r.Action, "{device}", deviceName)
		action = strings.ReplaceAll(action, "{class}", deviceClass)

		if rh.onAction != nil {
			go rh.onAction(action)
		}
	}
}

// ForSystemPrompt returns device-reactive capabilities for the system prompt.
func (rh *ReactiveHandler) ForSystemPrompt() string {
	rh.mu.RLock()
	defer rh.mu.RUnlock()

	if len(rh.reactions) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("## Hardware-Reactive Behaviors\n\n")
	sb.WriteString("The following automatic behaviors are configured for device events:\n")
	for _, r := range rh.reactions {
		sb.WriteString(fmt.Sprintf("- On %s %s: %s\n", r.EventType, r.DeviceClass, r.Action))
	}
	return sb.String()
}

func defaultReactions() []DeviceReaction {
	return []DeviceReaction{
		{
			DeviceClass: "video",
			EventType:   "add",
			Action:      "📷 A camera device ({device}) was connected. I can help you take photos or analyze images if you'd like.",
			Cooldown:    5 * time.Minute,
		},
		{
			DeviceClass: "mass_storage",
			EventType:   "add",
			Action:      "💾 A USB storage device ({device}) was connected. I can help you list, index, or back up its contents.",
			Cooldown:    5 * time.Minute,
		},
		{
			DeviceClass: "mass_storage",
			EventType:   "remove",
			Action:      "⏏️ USB storage device ({device}) was disconnected.",
			Cooldown:    1 * time.Minute,
		},
		{
			DeviceClass: "audio",
			EventType:   "add",
			Action:      "🎤 An audio device ({device}) was connected. Voice mode is available if you'd like to use it.",
			Cooldown:    5 * time.Minute,
		},
		{
			DeviceClass: "printer",
			EventType:   "add",
			Action:      "🖨️ A printer ({device}) was connected. I can help you manage print jobs.",
			Cooldown:    5 * time.Minute,
		},
		{
			DeviceClass: "network",
			EventType:   "add",
			Action:      "🌐 A network adapter ({device}) was connected.",
			Cooldown:    5 * time.Minute,
		},
	}
}

func matchesClass(deviceClass, pattern string) bool {
	return strings.EqualFold(deviceClass, pattern) ||
		strings.Contains(strings.ToLower(deviceClass), strings.ToLower(pattern))
}
