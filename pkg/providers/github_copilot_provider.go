package providers

import (
	"context"
	"fmt"

	json "encoding/json"

	copilot "github.com/github/copilot-sdk/go"
)

// SWE100821: Stores client reference so lifecycle is managed with the provider
type GitHubCopilotProvider struct {
	uri         string
	connectMode string
	client      *copilot.Client
	session     *copilot.Session
}

func NewGitHubCopilotProvider(uri string, connectMode string, model string) (*GitHubCopilotProvider, error) {
	var client *copilot.Client
	var session *copilot.Session
	if connectMode == "" {
		connectMode = "grpc"
	}
	switch connectMode {
	case "stdio":
		// SWE100821: stdio transport not implemented — fail loudly instead of returning nil session
		return nil, fmt.Errorf("github copilot: stdio connect mode is not yet implemented; use 'grpc'")
	case "grpc":
		client = copilot.NewClient(&copilot.ClientOptions{
			CLIUrl: uri,
		})
		if err := client.Start(context.Background()); err != nil {
			return nil, fmt.Errorf("can't connect to Github Copilot: %w", err)
		}
		// SWE100821: removed `defer client.Stop()` — was killing the client before Chat() could use it
		var err error
		session, err = client.CreateSession(context.Background(), &copilot.SessionConfig{
			Model: model,
			Hooks: &copilot.SessionHooks{},
		})
		if err != nil {
			client.Stop()
			return nil, fmt.Errorf("failed to create copilot session: %w", err)
		}
	}

	return &GitHubCopilotProvider{
		uri:         uri,
		connectMode: connectMode,
		client:      client,
		session:     session,
	}, nil
}

// Chat sends a chat request to GitHub Copilot
func (p *GitHubCopilotProvider) Chat(ctx context.Context, messages []Message, tools []ToolDefinition, model string, options map[string]interface{}) (*LLMResponse, error) {
	type tempMessage struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	}
	out := make([]tempMessage, 0, len(messages))

	for _, msg := range messages {
		out = append(out, tempMessage{
			Role:    msg.Role,
			Content: msg.Content,
		})
	}

	fullcontent, err := json.Marshal(out)
	if err != nil {
		return nil, fmt.Errorf("github copilot: marshal messages: %w", err)
	}

	// SWE100821: Guard against nil session — can happen if constructor was called with unknown mode
	if p.session == nil {
		return nil, fmt.Errorf("github copilot: session not initialized (connect_mode=%s)", p.connectMode)
	}

	content, err := p.session.Send(ctx, copilot.MessageOptions{
		Prompt: string(fullcontent),
	})
	if err != nil {
		return nil, fmt.Errorf("github copilot: send failed: %w", err)
	}

	return &LLMResponse{
		FinishReason: "stop",
		Content:      content,
	}, nil

}

func (p *GitHubCopilotProvider) GetDefaultModel() string {

	return "gpt-4.1"
}
