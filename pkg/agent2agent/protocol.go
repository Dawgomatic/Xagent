// SWE100821: Agent-to-agent communication protocol.
// Defines a simple protocol so multiple Xagent instances can exchange messages.
// Agent A (e.g., on a Pi monitoring sensors) can send structured messages to
// Agent B (e.g., on a desktop with GPU) for deeper analysis.
// Uses HTTP POST over the existing health/gateway port.

package agent2agent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/Dawgomatic/Xagent/pkg/logger"
)

// A2AMessage is the inter-agent message format.
type A2AMessage struct {
	FromAgentID string                 `json:"from_agent_id"`
	ToAgentID   string                 `json:"to_agent_id,omitempty"` // empty = broadcast
	Type        string                 `json:"type"`                 // "query", "notify", "response"
	Topic       string                 `json:"topic"`
	Payload     string                 `json:"payload"`
	RequestID   string                 `json:"request_id"`
	Timestamp   time.Time              `json:"timestamp"`
	Metadata    map[string]string      `json:"metadata,omitempty"`
}

// A2AResponse is the response to an inter-agent query.
type A2AResponse struct {
	RequestID string `json:"request_id"`
	AgentID   string `json:"agent_id"`
	Content   string `json:"content"`
	Success   bool   `json:"success"`
}

// MessageHandler processes incoming A2A messages.
type MessageHandler func(ctx context.Context, msg A2AMessage) (response string, err error)

// A2AHub manages agent-to-agent communication.
type A2AHub struct {
	agentID     string
	knownPeers  map[string]string // agentID -> endpoint URL
	handler     MessageHandler
	client      *http.Client
	mu          sync.RWMutex
	inbox       chan A2AMessage
}

// NewA2AHub creates a new agent-to-agent communication hub.
func NewA2AHub(agentID string) *A2AHub {
	return &A2AHub{
		agentID:    agentID,
		knownPeers: make(map[string]string),
		client:     &http.Client{Timeout: 30 * time.Second},
		inbox:      make(chan A2AMessage, 100),
	}
}

// SetHandler registers the function that processes incoming messages.
func (h *A2AHub) SetHandler(handler MessageHandler) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.handler = handler
}

// RegisterPeer adds a known peer agent with its endpoint URL.
func (h *A2AHub) RegisterPeer(agentID, endpoint string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.knownPeers[agentID] = endpoint
	logger.InfoCF("a2a", "Peer registered",
		map[string]interface{}{"peer_id": agentID, "endpoint": endpoint})
}

// RemovePeer removes a known peer.
func (h *A2AHub) RemovePeer(agentID string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.knownPeers, agentID)
}

// Send sends a message to a specific peer agent.
func (h *A2AHub) Send(ctx context.Context, msg A2AMessage) (*A2AResponse, error) {
	msg.FromAgentID = h.agentID
	msg.Timestamp = time.Now()

	h.mu.RLock()
	endpoint, ok := h.knownPeers[msg.ToAgentID]
	h.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("unknown peer agent: %s", msg.ToAgentID)
	}

	return h.sendToEndpoint(ctx, endpoint, msg)
}

// Broadcast sends a message to all known peers.
func (h *A2AHub) Broadcast(ctx context.Context, msg A2AMessage) ([]*A2AResponse, error) {
	msg.FromAgentID = h.agentID
	msg.Timestamp = time.Now()

	h.mu.RLock()
	peers := make(map[string]string, len(h.knownPeers))
	for k, v := range h.knownPeers {
		peers[k] = v
	}
	h.mu.RUnlock()

	var responses []*A2AResponse
	for peerID, endpoint := range peers {
		msg.ToAgentID = peerID
		resp, err := h.sendToEndpoint(ctx, endpoint, msg)
		if err != nil {
			logger.WarnCF("a2a", "Broadcast to peer failed",
				map[string]interface{}{"peer": peerID, "error": err.Error()})
			continue
		}
		responses = append(responses, resp)
	}

	return responses, nil
}

// HandleIncoming processes an incoming A2A message (called by HTTP handler).
func (h *A2AHub) HandleIncoming(ctx context.Context, msg A2AMessage) (*A2AResponse, error) {
	h.mu.RLock()
	handler := h.handler
	h.mu.RUnlock()

	if handler == nil {
		return &A2AResponse{
			RequestID: msg.RequestID,
			AgentID:   h.agentID,
			Content:   "No handler registered",
			Success:   false,
		}, nil
	}

	content, err := handler(ctx, msg)
	if err != nil {
		return &A2AResponse{
			RequestID: msg.RequestID,
			AgentID:   h.agentID,
			Content:   fmt.Sprintf("Error: %v", err),
			Success:   false,
		}, nil
	}

	return &A2AResponse{
		RequestID: msg.RequestID,
		AgentID:   h.agentID,
		Content:   content,
		Success:   true,
	}, nil
}

// HTTPHandler returns an http.HandlerFunc for the A2A endpoint.
// Mount this at /a2a on the gateway.
func (h *A2AHub) HTTPHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "failed to read body", http.StatusBadRequest)
			return
		}

		var msg A2AMessage
		if err := json.Unmarshal(body, &msg); err != nil {
			http.Error(w, "invalid message format", http.StatusBadRequest)
			return
		}

		resp, err := h.HandleIncoming(r.Context(), msg)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}
}

// ListPeers returns the IDs of all known peers.
func (h *A2AHub) ListPeers() []string {
	h.mu.RLock()
	defer h.mu.RUnlock()
	peers := make([]string, 0, len(h.knownPeers))
	for k := range h.knownPeers {
		peers = append(peers, k)
	}
	return peers
}

func (h *A2AHub) sendToEndpoint(ctx context.Context, endpoint string, msg A2AMessage) (*A2AResponse, error) {
	data, err := json.Marshal(msg)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", endpoint+"/a2a", bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := h.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("a2a request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var a2aResp A2AResponse
	if err := json.Unmarshal(respBody, &a2aResp); err != nil {
		return nil, fmt.Errorf("a2a response parse failed: %w", err)
	}

	return &a2aResp, nil
}
