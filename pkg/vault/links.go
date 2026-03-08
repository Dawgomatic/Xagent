// Xagent - Ultra-lightweight personal AI agent
// Obsidian Knowledge Vault — wikilink extraction and topic detection
//
// Copyright (c) 2026 Xagent contributors
// License: MIT

package vault

import (
	"strings"
	"sync"
	"unicode"
)

// TopicRegistry tracks known topics for consistent linking across notes.
type TopicRegistry struct {
	mu     sync.RWMutex
	topics map[string]int // topic → mention count
}

func NewTopicRegistry() *TopicRegistry {
	return &TopicRegistry{
		topics: make(map[string]int),
	}
}

// Record increments the mention count for a topic.
func (tr *TopicRegistry) Record(topic string) {
	tr.mu.Lock()
	defer tr.mu.Unlock()
	tr.topics[strings.ToLower(topic)]++
}

// GetTopTopics returns the N most-mentioned topics.
func (tr *TopicRegistry) GetTopTopics(n int) []string {
	tr.mu.RLock()
	defer tr.mu.RUnlock()

	type kv struct {
		key   string
		count int
	}
	var sorted []kv
	for k, v := range tr.topics {
		sorted = append(sorted, kv{k, v})
	}
	// Simple selection sort for small N
	for i := 0; i < len(sorted) && i < n; i++ {
		maxIdx := i
		for j := i + 1; j < len(sorted); j++ {
			if sorted[j].count > sorted[maxIdx].count {
				maxIdx = j
			}
		}
		sorted[i], sorted[maxIdx] = sorted[maxIdx], sorted[i]
	}

	result := make([]string, 0, n)
	for i := 0; i < len(sorted) && i < n; i++ {
		result = append(result, sorted[i].key)
	}
	return result
}

// ExtractTopics pulls topic keywords from a user message using simple heuristics.
// No LLM call — fast keyword extraction based on known tech terms and patterns.
func ExtractTopics(message string) []string {
	if message == "" {
		return nil
	}

	lower := strings.ToLower(message)
	var found []string
	seen := make(map[string]bool)

	// Check against known topics
	for _, topic := range knownTopics {
		if strings.Contains(lower, topic) && !seen[topic] {
			found = append(found, topic)
			seen[topic] = true
		}
	}

	// Extract capitalized multi-word proper nouns (e.g. "OpenClaw", "Obsidian")
	words := strings.Fields(message)
	for _, word := range words {
		clean := strings.TrimFunc(word, func(r rune) bool {
			return !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '-'
		})
		if len(clean) < 3 {
			continue
		}
		lc := strings.ToLower(clean)

		// Skip common stop words
		if stopWords[lc] {
			continue
		}

		// Capitalized words that aren't at sentence start could be proper nouns/topics
		if unicode.IsUpper(rune(clean[0])) && !seen[lc] {
			// Simple heuristic: if it looks like a tech term or proper noun
			if isTechTerm(clean) {
				found = append(found, lc)
				seen[lc] = true
			}
		}
	}

	// Cap at 5 topics per message to avoid noise
	if len(found) > 5 {
		found = found[:5]
	}

	return found
}

// isTechTerm uses simple heuristics to identify tech-related proper nouns.
func isTechTerm(word string) bool {
	// Contains mixed case inside word (e.g. "GitHub", "OpenAI", "FastAPI")
	if len(word) > 2 {
		for i := 1; i < len(word)-1; i++ {
			if unicode.IsUpper(rune(word[i])) {
				return true
			}
		}
	}

	// Ends with common tech suffixes
	lower := strings.ToLower(word)
	for _, suffix := range []string{"api", "db", "cli", "sdk", "bot", "rl", "ai", "ml", "js", "go"} {
		if strings.HasSuffix(lower, suffix) {
			return true
		}
	}

	return false
}

// BuildWikilinks formats a slice of names as [[wikilinks]].
func BuildWikilinks(names []string) string {
	if len(names) == 0 {
		return ""
	}
	links := make([]string, len(names))
	for i, name := range names {
		links[i] = "[[" + name + "]]"
	}
	return strings.Join(links, " ")
}

// BuildWikilink formats a single name as a [[wikilink]].
func BuildWikilink(name string) string {
	return "[[" + name + "]]"
}

// Known tech topics for fast matching
var knownTopics = []string{
	// Infrastructure
	"docker", "kubernetes", "k8s", "terraform", "ansible", "nginx", "apache",
	"linux", "ubuntu", "debian", "centos", "macos", "windows",

	// Languages & runtimes
	"python", "golang", "javascript", "typescript", "rust", "java", "c++",
	"node.js", "nodejs", "deno", "bun",

	// AI/ML
	"reinforcement learning", "machine learning", "deep learning", "neural network",
	"transformer", "llm", "fine-tuning", "training", "inference",
	"openai", "anthropic", "gemini", "claude", "gpt", "llama",
	"sglang", "vllm", "pytorch", "tensorflow",

	// Frameworks
	"react", "vue", "next.js", "fastapi", "flask", "django", "express",
	"gin", "fiber",

	// Databases
	"postgresql", "postgres", "mysql", "sqlite", "redis", "mongodb",
	"elasticsearch",

	// DevOps
	"git", "github", "gitlab", "ci/cd", "deployment", "monitoring",

	// Protocols
	"mavlink", "ros", "mqtt", "grpc", "websocket", "http",

	// Xagent-specific
	"xagent", "openclaw", "picoclaw", "nanobot", "obsidian",
	"agent loop", "provider", "session", "tool", "channel",
	"dream mode", "personality", "provenance", "memory",

	// Hardware
	"gpu", "cuda", "rocm", "drone", "ardupilot", "airsim",
	"raspberry pi", "jetson", "maixcam",
}

var stopWords = map[string]bool{
	"the": true, "and": true, "for": true, "are": true, "but": true,
	"not": true, "you": true, "all": true, "can": true, "had": true,
	"her": true, "was": true, "one": true, "our": true, "out": true,
	"has": true, "have": true, "this": true, "that": true, "with": true,
	"from": true, "been": true, "some": true, "what": true, "when": true,
	"will": true, "more": true, "make": true, "like": true, "just": true,
	"over": true, "also": true, "then": true, "them": true, "than": true,
	"into": true, "would": true, "could": true, "should": true,
	"about": true, "there": true, "their": true, "which": true,
	"these": true, "after": true, "other": true,
	"want": true, "need": true, "know": true, "think": true,
	"please": true, "thanks": true, "hello": true, "sure": true,
	"yes": true, "yeah": true, "okay": true,
}
