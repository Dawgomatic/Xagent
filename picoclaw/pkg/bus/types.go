package bus

// SWE100821: RequestID enables end-to-end tracing through logs
type InboundMessage struct {
	RequestID  string            `json:"request_id,omitempty"` // SWE100821: Unique trace ID
	Channel    string            `json:"channel"`
	SenderID   string            `json:"sender_id"`
	ChatID     string            `json:"chat_id"`
	Content    string            `json:"content"`
	Media      []string          `json:"media,omitempty"`
	SessionKey string            `json:"session_key"`
	Metadata   map[string]string `json:"metadata,omitempty"`
}

type OutboundMessage struct {
	RequestID string `json:"request_id,omitempty"` // SWE100821: Correlates with inbound
	Channel   string `json:"channel"`
	ChatID    string `json:"chat_id"`
	Content   string `json:"content"`
}

type MessageHandler func(InboundMessage) error
