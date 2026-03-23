// SWE100821: Integrated semantic memory — Go-native Qdrant client for vector search.
// Replaces the disconnected Python memory_bridge.py with a built-in Go implementation
// that auto-embeds conversation summaries and epoch journals, and retrieves relevant
// memories by similarity during context building.
//
// Uses Ollama's /api/embeddings endpoint for local embedding generation.
// Falls back gracefully to file-based memory if Qdrant is unavailable.

package memory

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/Dawgomatic/Xagent/pkg/logger"
)

const (
	defaultQdrantURL      = "http://localhost:6333"
	defaultCollectionName = "xagent_memory"
	defaultEmbedModel     = "nomic-embed-text"
	defaultOllamaURL      = "http://localhost:11434"
	embeddingDim          = 768
)

// SemanticMemory provides vector-based memory retrieval via Qdrant + Ollama embeddings.
type SemanticMemory struct {
	qdrantURL      string
	ollamaURL      string
	collection     string
	embedModel     string
	client         *http.Client
	available      bool
	mu             sync.RWMutex
	nextID         uint64
}

// MemoryPoint represents a stored memory vector with metadata.
type MemoryPoint struct {
	ID       uint64            `json:"id"`
	Text     string            `json:"text"`
	Source   string            `json:"source"` // "conversation", "epoch", "memory_md", "daily_note"
	Score    float64           `json:"score,omitempty"`
	Created  time.Time         `json:"created"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

// NewSemanticMemory creates a semantic memory client.
// Probes Qdrant availability on creation; falls back gracefully if unavailable.
func NewSemanticMemory(qdrantURL, ollamaURL, collection, embedModel string) *SemanticMemory {
	if qdrantURL == "" {
		qdrantURL = defaultQdrantURL
	}
	if ollamaURL == "" {
		ollamaURL = defaultOllamaURL
	}
	if collection == "" {
		collection = defaultCollectionName
	}
	if embedModel == "" {
		embedModel = defaultEmbedModel
	}

	sm := &SemanticMemory{
		qdrantURL:  qdrantURL,
		ollamaURL:  ollamaURL,
		collection: collection,
		embedModel: embedModel,
		client:     &http.Client{Timeout: 10 * time.Second},
		nextID:     uint64(time.Now().UnixNano()),
	}

	// SWE100821: Probe Qdrant availability in background
	go sm.probe()

	return sm
}

// IsAvailable returns whether Qdrant is reachable and the collection exists.
func (sm *SemanticMemory) IsAvailable() bool {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.available
}

// Store embeds and stores a text chunk in Qdrant.
func (sm *SemanticMemory) Store(ctx context.Context, text, source string, metadata map[string]string) error {
	if !sm.IsAvailable() {
		return fmt.Errorf("semantic memory unavailable")
	}

	embedding, err := sm.embed(ctx, text)
	if err != nil {
		return fmt.Errorf("embedding failed: %w", err)
	}

	sm.mu.Lock()
	id := sm.nextID
	sm.nextID++
	sm.mu.Unlock()

	payload := map[string]interface{}{
		"text":    text,
		"source":  source,
		"created": time.Now().Format(time.RFC3339),
	}
	for k, v := range metadata {
		payload[k] = v
	}

	body := map[string]interface{}{
		"points": []map[string]interface{}{
			{
				"id":      id,
				"vector":  embedding,
				"payload": payload,
			},
		},
	}

	return sm.qdrantPut(ctx, fmt.Sprintf("/collections/%s/points", sm.collection), body)
}

// Search finds the top-k most similar memories to the query.
func (sm *SemanticMemory) Search(ctx context.Context, query string, topK int) ([]MemoryPoint, error) {
	if !sm.IsAvailable() {
		return nil, fmt.Errorf("semantic memory unavailable")
	}

	embedding, err := sm.embed(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query embedding failed: %w", err)
	}

	body := map[string]interface{}{
		"vector":     embedding,
		"limit":      topK,
		"with_payload": true,
	}

	respBody, err := sm.qdrantPost(ctx, fmt.Sprintf("/collections/%s/points/search", sm.collection), body)
	if err != nil {
		return nil, err
	}

	var resp struct {
		Result []struct {
			ID      uint64                 `json:"id"`
			Score   float64                `json:"score"`
			Payload map[string]interface{} `json:"payload"`
		} `json:"result"`
	}
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, fmt.Errorf("search response parse failed: %w", err)
	}

	results := make([]MemoryPoint, 0, len(resp.Result))
	for _, r := range resp.Result {
		text, _ := r.Payload["text"].(string)
		source, _ := r.Payload["source"].(string)
		// SWE100821: Parse created timestamp from payload for recency scoring
		var created time.Time
		if createdStr, ok := r.Payload["created"].(string); ok {
			if t, err := time.Parse(time.RFC3339, createdStr); err == nil {
				created = t
			}
		}
		results = append(results, MemoryPoint{
			ID:      r.ID,
			Text:    text,
			Source:  source,
			Score:   r.Score,
			Created: created,
		})
	}

	return results, nil
}

// ForSystemPrompt searches for relevant memories, ranks by importance, and formats for the system prompt.
// SWE100821: Now uses MemoryScorer for multi-factor ranking instead of raw vector similarity.
func (sm *SemanticMemory) ForSystemPrompt(ctx context.Context, userMessage string, maxResults int) string {
	if !sm.IsAvailable() {
		return ""
	}

	results, err := sm.Search(ctx, userMessage, maxResults*2) // fetch extra for re-ranking
	if err != nil {
		logger.DebugCF("memory", "Semantic search failed", map[string]interface{}{"error": err.Error()})
		return ""
	}

	if len(results) == 0 {
		return ""
	}

	// SWE100821: Apply multi-factor scoring (recency, salience, novelty, semantic)
	scorer := DefaultScorer()
	// Add user message keywords to salience detector for context-aware ranking
	for _, word := range strings.Fields(strings.ToLower(userMessage)) {
		if len(word) > 3 {
			scorer.SalienceKeywords = append(scorer.SalienceKeywords, word)
		}
	}

	ranked := scorer.RankMemories(results, nil)

	var sb strings.Builder
	sb.WriteString("## Relevant Memories (semantic recall)\n\n")
	shown := 0
	for _, sm := range ranked {
		if sm.ImportanceScore < 0.2 {
			continue
		}
		shown++
		if shown > maxResults {
			break
		}
		sb.WriteString(fmt.Sprintf("%d. [%s, importance=%.2f] %s\n", shown, sm.Point.Source, sm.ImportanceScore, truncateStr(sm.Point.Text, 300)))
	}

	if shown == 0 {
		return ""
	}

	return sb.String()
}

// StoreConversationSummary stores a session summary for later recall.
func (sm *SemanticMemory) StoreConversationSummary(ctx context.Context, sessionKey, summary string) error {
	return sm.Store(ctx, summary, "conversation", map[string]string{"session": sessionKey})
}

// StoreEpochJournal stores an epoch journal for later recall.
func (sm *SemanticMemory) StoreEpochJournal(ctx context.Context, sessionID, journal string) error {
	return sm.Store(ctx, journal, "epoch", map[string]string{"session_id": sessionID})
}

func (sm *SemanticMemory) embed(ctx context.Context, text string) ([]float64, error) {
	body := map[string]interface{}{
		"model":  sm.embedModel,
		"prompt": text,
	}
	data, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", sm.ollamaURL+"/api/embeddings", bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := sm.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ollama embeddings request failed: %w", err)
	}
	defer resp.Body.Close()

	respData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result struct {
		Embedding []float64 `json:"embedding"`
	}
	if err := json.Unmarshal(respData, &result); err != nil {
		return nil, fmt.Errorf("embedding parse failed: %w", err)
	}

	if len(result.Embedding) == 0 {
		return nil, fmt.Errorf("empty embedding returned")
	}

	// SWE100821: Validate dimension matches Qdrant collection — mismatched dim causes
	// silent upsert/search failures. Catch early with a clear error message.
	if len(result.Embedding) != embeddingDim {
		return nil, fmt.Errorf("embedding dimension mismatch: model returned %d, collection expects %d (model=%s)",
			len(result.Embedding), embeddingDim, sm.embedModel)
	}

	// Normalize
	norm := 0.0
	for _, v := range result.Embedding {
		norm += v * v
	}
	norm = math.Sqrt(norm)
	if norm > 0 {
		for i := range result.Embedding {
			result.Embedding[i] /= norm
		}
	}

	return result.Embedding, nil
}

func (sm *SemanticMemory) probe() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Check if Qdrant is reachable
	req, err := http.NewRequestWithContext(ctx, "GET", sm.qdrantURL+"/collections/"+sm.collection, nil)
	if err != nil {
		return
	}

	resp, err := sm.client.Do(req)
	if err != nil {
		logger.InfoCF("memory", "Qdrant not available (semantic memory disabled)",
			map[string]interface{}{"url": sm.qdrantURL})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == 200 {
		sm.mu.Lock()
		sm.available = true
		sm.mu.Unlock()
		logger.InfoCF("memory", "Semantic memory connected",
			map[string]interface{}{"collection": sm.collection})
		return
	}

	// Collection doesn't exist — create it
	if resp.StatusCode == 404 {
		if err := sm.createCollection(ctx); err != nil {
			logger.WarnCF("memory", "Failed to create Qdrant collection",
				map[string]interface{}{"error": err.Error()})
			return
		}
		sm.mu.Lock()
		sm.available = true
		sm.mu.Unlock()
		logger.InfoCF("memory", "Semantic memory initialized (new collection)",
			map[string]interface{}{"collection": sm.collection})
	}
}

func (sm *SemanticMemory) createCollection(ctx context.Context) error {
	body := map[string]interface{}{
		"vectors": map[string]interface{}{
			"size":     embeddingDim,
			"distance": "Cosine",
		},
	}
	return sm.qdrantPut(ctx, "/collections/"+sm.collection, body)
}

func (sm *SemanticMemory) qdrantPut(ctx context.Context, path string, body interface{}) error {
	data, err := json.Marshal(body)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, "PUT", sm.qdrantURL+path, bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := sm.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("qdrant PUT %s returned %d: %s", path, resp.StatusCode, string(respBody))
	}
	return nil
}

// SWE100821: Check HTTP status — failed searches were returning garbage silently
func (sm *SemanticMemory) qdrantPost(ctx context.Context, path string, body interface{}) ([]byte, error) {
	data, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", sm.qdrantURL+path, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := sm.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("qdrant POST %s returned %d: %s", path, resp.StatusCode, truncateStr(string(respBody), 200))
	}
	return respBody, nil
}

// SWE100821: Rune-safe truncation — byte slicing can split multibyte chars (CJK, emoji)
func truncateStr(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen]) + "..."
}
