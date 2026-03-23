package bus

import (
	"context"
	"log"
	"sync"
	"sync/atomic"
)

type MessageBus struct {
	inbound  chan InboundMessage
	outbound chan OutboundMessage
	handlers map[string]MessageHandler
	mu       sync.RWMutex
	closed   atomic.Bool // SWE100821: guards against send-on-closed-channel panic
	// SWE100821: Track dropped messages for observability
	DroppedInbound  atomic.Int64
	DroppedOutbound atomic.Int64
}

// SWE100821: Increased buffer from 100→500 — under burst load (multiple Discord channels,
// sensor events, proactive messages) the old 100-slot buffer dropped user messages silently.
func NewMessageBus() *MessageBus {
	return &MessageBus{
		inbound:  make(chan InboundMessage, 500),
		outbound: make(chan OutboundMessage, 500),
		handlers: make(map[string]MessageHandler),
	}
}

// PublishInbound sends a message to the agent. Non-blocking: drops if buffer full.
func (mb *MessageBus) PublishInbound(msg InboundMessage) {
	// SWE100821: Guard against send-on-closed-channel panic during shutdown
	if mb.closed.Load() {
		return
	}
	select {
	case mb.inbound <- msg:
	default:
		mb.DroppedInbound.Add(1)
		log.Printf("[WARN] bus: inbound message dropped (buffer full, channel=%s, total_dropped=%d)",
			msg.Channel, mb.DroppedInbound.Load())
	}
}

func (mb *MessageBus) ConsumeInbound(ctx context.Context) (InboundMessage, bool) {
	select {
	case msg := <-mb.inbound:
		return msg, true
	case <-ctx.Done():
		return InboundMessage{}, false
	}
}

// PublishOutbound sends a response to a channel. Non-blocking: drops if buffer full.
func (mb *MessageBus) PublishOutbound(msg OutboundMessage) {
	if mb.closed.Load() {
		return
	}
	select {
	case mb.outbound <- msg:
	default:
		mb.DroppedOutbound.Add(1)
		log.Printf("[WARN] bus: outbound message dropped (buffer full, channel=%s, total_dropped=%d)",
			msg.Channel, mb.DroppedOutbound.Load())
	}
}

func (mb *MessageBus) SubscribeOutbound(ctx context.Context) (OutboundMessage, bool) {
	select {
	case msg := <-mb.outbound:
		return msg, true
	case <-ctx.Done():
		return OutboundMessage{}, false
	}
}

func (mb *MessageBus) RegisterHandler(channel string, handler MessageHandler) {
	mb.mu.Lock()
	defer mb.mu.Unlock()
	mb.handlers[channel] = handler
}

func (mb *MessageBus) GetHandler(channel string) (MessageHandler, bool) {
	mb.mu.RLock()
	defer mb.mu.RUnlock()
	handler, ok := mb.handlers[channel]
	return handler, ok
}

// SWE100821: Close marks the bus as closed before closing channels.
// The closed flag prevents PublishInbound/PublishOutbound from panicking
// with "send on closed channel" during concurrent shutdown.
func (mb *MessageBus) Close() {
	mb.closed.Store(true)
	close(mb.inbound)
	close(mb.outbound)
}
