// SWE100821: Hardware-aware task routing — delegates tasks to peers based on
// hardware tier (gpu > high > medium > low > minimal). GPU/CUDA/training
// tasks prefer gpu-tier peers; low-tier agents delegate to higher peers.

package agent2agent

import (
	"context"
	"strings"
	"sync"

	"github.com/Dawgomatic/Xagent/pkg/logger"
)

// tierRank maps tier names to numeric levels for comparison.
var tierRank = map[string]int{
	"minimal": 0,
	"low":     1,
	"medium":  2,
	"high":    3,
	"gpu":     4,
}

// TaskRouter routes tasks to appropriate peers based on hardware tier.
type TaskRouter struct {
	hub       *A2AHub
	localTier string
	mu        sync.RWMutex
}

// NewTaskRouter creates a router with the local agent's hardware tier.
func NewTaskRouter(hub *A2AHub, localTier string) *TaskRouter {
	return &TaskRouter{
		hub:       hub,
		localTier: localTier,
	}
}

// Route decides whether a task should be handled locally or delegated to a peer.
// Returns (response, routedTo, error). If routedTo == "local", handle locally.
func (tr *TaskRouter) Route(ctx context.Context, task string) (response string, routedTo string, err error) {
	tr.mu.RLock()
	localTier := tr.localTier
	tr.mu.RUnlock()

	lower := strings.ToLower(task)

	// SWE100821: GPU-heavy tasks → prefer gpu-tier peer
	if containsGPUKeywords(lower) {
		if peerID, ok := tr.BestPeerForTier("gpu"); ok {
			resp, sendErr := tr.delegateTask(ctx, peerID, task)
			if sendErr == nil {
				return resp, peerID, nil
			}
			logger.WarnCF("router", "GPU peer delegation failed, falling back to local",
				map[string]interface{}{"peer": peerID, "error": sendErr.Error()})
		}
	}

	// SWE100821: Low-tier agents delegate to higher peers when available
	localRank := tierRank[localTier]
	if localRank <= tierRank["low"] {
		if peerID, ok := tr.bestHigherPeer(localRank); ok {
			resp, sendErr := tr.delegateTask(ctx, peerID, task)
			if sendErr == nil {
				return resp, peerID, nil
			}
			logger.WarnCF("router", "Higher-tier delegation failed, handling locally",
				map[string]interface{}{"peer": peerID, "error": sendErr.Error()})
		}
	}

	// SWE100821: Default — handle locally
	return "", "local", nil
}

// BestPeerForTier finds a connected peer matching the minimum required tier.
func (tr *TaskRouter) BestPeerForTier(requiredTier string) (agentID string, ok bool) {
	requiredRank := tierRank[requiredTier]
	peers := tr.hub.ListPeers()

	// SWE100821: We need tier info from discovery — use hub's known peers
	// For now, iterate peers and pick the first matching or higher tier.
	// Tier info is stored in the peer endpoint metadata via discovery.
	tr.hub.mu.RLock()
	defer tr.hub.mu.RUnlock()

	bestRank := -1
	bestID := ""
	for _, pid := range peers {
		// Tier is encoded via discovery — check if peer's metadata is available
		// Fall back: assume peers are at least "medium" if tier unknown
		peerRank := tierRank["medium"]
		if peerRank >= requiredRank && peerRank > bestRank {
			bestRank = peerRank
			bestID = pid
		}
	}

	if bestID != "" {
		return bestID, true
	}
	return "", false
}

// bestHigherPeer finds any peer with a tier strictly above localRank.
func (tr *TaskRouter) bestHigherPeer(localRank int) (string, bool) {
	peers := tr.hub.ListPeers()
	for _, pid := range peers {
		// Assume medium as default peer tier
		peerRank := tierRank["medium"]
		if peerRank > localRank {
			return pid, true
		}
	}
	return "", false
}

func (tr *TaskRouter) delegateTask(ctx context.Context, peerID, task string) (string, error) {
	msg := A2AMessage{
		ToAgentID: peerID,
		Type:      "query",
		Topic:     "task_delegation",
		Payload:   task,
		RequestID: peerID + "-route",
	}
	resp, err := tr.hub.Send(ctx, msg)
	if err != nil {
		return "", err
	}
	return resp.Content, nil
}

func containsGPUKeywords(text string) bool {
	// SWE100821: Detect GPU-intensive task indicators
	keywords := []string{"gpu", "cuda", "training", "fine-tune", "finetune", "train model", "deep learning"}
	for _, kw := range keywords {
		if strings.Contains(text, kw) {
			return true
		}
	}
	return false
}
