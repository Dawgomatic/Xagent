// SWE100821: mDNS-style peer discovery using UDP broadcast.
// Advertises agent presence and discovers peers on the local network.
// No external deps — uses raw UDP on port 18799.

package agent2agent

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/Dawgomatic/Xagent/pkg/logger"
)

const (
	discoveryPort     = 18799
	beaconInterval    = 30 * time.Second
	peerExpiry        = 120 * time.Second // remove peers not seen for 2 min
	maxBeaconSize     = 1024
)

// PeerInfo describes a discovered peer agent.
type PeerInfo struct {
	AgentID  string    `json:"agent_id"`
	Endpoint string    `json:"endpoint"`
	Tier     string    `json:"tier"`
	LastSeen time.Time `json:"last_seen"`
}

// beacon is the JSON payload broadcast over UDP.
type beacon struct {
	AgentID string `json:"agent_id"`
	Port    int    `json:"port"`
	Tier    string `json:"tier"`
}

// PeerDiscovery advertises this agent and discovers peers via UDP broadcast.
type PeerDiscovery struct {
	agentID string
	port    int
	tier    string
	hub     *A2AHub
	running bool
	peers   map[string]*PeerInfo
	cancel  context.CancelFunc
	mu      sync.Mutex
}

// NewPeerDiscovery creates a new peer discovery instance.
func NewPeerDiscovery(agentID string, port int, tier string, hub *A2AHub) *PeerDiscovery {
	return &PeerDiscovery{
		agentID: agentID,
		port:    port,
		tier:    tier,
		hub:     hub,
		peers:   make(map[string]*PeerInfo),
	}
}

// Start begins advertising and browsing for peers.
func (pd *PeerDiscovery) Start(ctx context.Context) {
	pd.mu.Lock()
	if pd.running {
		pd.mu.Unlock()
		return
	}
	pd.running = true
	childCtx, cancel := context.WithCancel(ctx)
	pd.cancel = cancel
	pd.mu.Unlock()

	// SWE100821: Listener goroutine — receives beacons from peers
	go pd.listen(childCtx)

	// SWE100821: Advertiser goroutine — broadcasts our beacon every 30s
	go pd.advertise(childCtx)

	// SWE100821: Reaper goroutine — removes stale peers
	go pd.reapStale(childCtx)

	logger.InfoCF("discovery", "Peer discovery started",
		map[string]interface{}{"agent_id": pd.agentID, "port": pd.port, "tier": pd.tier})
}

// Stop halts advertising and browsing.
func (pd *PeerDiscovery) Stop() {
	pd.mu.Lock()
	defer pd.mu.Unlock()
	if !pd.running {
		return
	}
	pd.running = false
	if pd.cancel != nil {
		pd.cancel()
	}
	logger.InfoCF("discovery", "Peer discovery stopped", nil)
}

// GetDiscoveredPeers returns a snapshot of currently known peers.
func (pd *PeerDiscovery) GetDiscoveredPeers() []PeerInfo {
	pd.mu.Lock()
	defer pd.mu.Unlock()
	result := make([]PeerInfo, 0, len(pd.peers))
	for _, p := range pd.peers {
		result = append(result, *p)
	}
	return result
}

func (pd *PeerDiscovery) advertise(ctx context.Context) {
	b := beacon{
		AgentID: pd.agentID,
		Port:    pd.port,
		Tier:    pd.tier,
	}
	data, err := json.Marshal(b)
	if err != nil {
		logger.ErrorCF("discovery", "Failed to marshal beacon", map[string]interface{}{"error": err.Error()})
		return
	}

	addr, err := net.ResolveUDPAddr("udp4", fmt.Sprintf("255.255.255.255:%d", discoveryPort))
	if err != nil {
		logger.ErrorCF("discovery", "Failed to resolve broadcast addr", map[string]interface{}{"error": err.Error()})
		return
	}

	ticker := time.NewTicker(beaconInterval)
	defer ticker.Stop()

	// SWE100821: Broadcast immediately, then on each tick
	pd.sendBeacon(addr, data)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			pd.sendBeacon(addr, data)
		}
	}
}

func (pd *PeerDiscovery) sendBeacon(addr *net.UDPAddr, data []byte) {
	conn, err := net.DialUDP("udp4", nil, addr)
	if err != nil {
		logger.DebugCF("discovery", "Beacon send failed", map[string]interface{}{"error": err.Error()})
		return
	}
	defer conn.Close()
	conn.Write(data)
}

func (pd *PeerDiscovery) listen(ctx context.Context) {
	addr := &net.UDPAddr{Port: discoveryPort, IP: net.IPv4zero}
	conn, err := net.ListenUDP("udp4", addr)
	if err != nil {
		logger.ErrorCF("discovery", "Failed to listen for beacons",
			map[string]interface{}{"error": err.Error(), "port": discoveryPort})
		return
	}
	defer conn.Close()

	buf := make([]byte, maxBeaconSize)
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		conn.SetReadDeadline(time.Now().Add(5 * time.Second))
		n, remoteAddr, err := conn.ReadFromUDP(buf)
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				continue
			}
			continue
		}

		var b beacon
		if err := json.Unmarshal(buf[:n], &b); err != nil {
			continue
		}

		// SWE100821: Skip our own beacons
		if b.AgentID == pd.agentID {
			continue
		}

		endpoint := fmt.Sprintf("http://%s:%d", remoteAddr.IP.String(), b.Port)

		pd.mu.Lock()
		existing, known := pd.peers[b.AgentID]
		if !known {
			pd.peers[b.AgentID] = &PeerInfo{
				AgentID:  b.AgentID,
				Endpoint: endpoint,
				Tier:     b.Tier,
				LastSeen: time.Now(),
			}
			// SWE100821: Auto-register in A2A hub
			pd.hub.RegisterPeer(b.AgentID, endpoint)
			logger.InfoCF("discovery", "Discovered new peer",
				map[string]interface{}{"peer": b.AgentID, "tier": b.Tier, "endpoint": endpoint})
		} else {
			existing.LastSeen = time.Now()
			existing.Tier = b.Tier
			existing.Endpoint = endpoint
		}
		pd.mu.Unlock()
	}
}

func (pd *PeerDiscovery) reapStale(ctx context.Context) {
	ticker := time.NewTicker(peerExpiry / 2)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			pd.mu.Lock()
			now := time.Now()
			for id, p := range pd.peers {
				if now.Sub(p.LastSeen) > peerExpiry {
					delete(pd.peers, id)
					pd.hub.RemovePeer(id)
					logger.InfoCF("discovery", "Peer expired", map[string]interface{}{"peer": id})
				}
			}
			pd.mu.Unlock()
		}
	}
}
