package config

import (
	"os"
	"path/filepath"
	"testing"
)

// TestDefaultConfig_HeartbeatEnabled verifies heartbeat is enabled by default
func TestDefaultConfig_HeartbeatEnabled(t *testing.T) {
	cfg := DefaultConfig()

	if !cfg.Heartbeat.Enabled {
		t.Error("Heartbeat should be enabled by default")
	}
}

// TestDefaultConfig_WorkspacePath verifies workspace path is correctly set
func TestDefaultConfig_WorkspacePath(t *testing.T) {
	cfg := DefaultConfig()

	// Just verify the workspace is set, don't compare exact paths
	// since expandHome behavior may differ based on environment
	if cfg.Agents.Defaults.Workspace == "" {
		t.Error("Workspace should not be empty")
	}
}

// TestDefaultConfig_Model verifies model is set
func TestDefaultConfig_Model(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Agents.Defaults.Model == "" {
		t.Error("Model should not be empty")
	}
}

// TestDefaultConfig_MaxTokens verifies max tokens has default value
func TestDefaultConfig_MaxTokens(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Agents.Defaults.MaxTokens == 0 {
		t.Error("MaxTokens should not be zero")
	}
}

// TestDefaultConfig_MaxToolIterations verifies max tool iterations has default value
func TestDefaultConfig_MaxToolIterations(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Agents.Defaults.MaxToolIterations == 0 {
		t.Error("MaxToolIterations should not be zero")
	}
}

// TestDefaultConfig_Temperature verifies temperature has default value
func TestDefaultConfig_Temperature(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Agents.Defaults.Temperature == 0 {
		t.Error("Temperature should not be zero")
	}
}

// TestDefaultConfig_Gateway verifies gateway defaults
func TestDefaultConfig_Gateway(t *testing.T) {
	cfg := DefaultConfig()

	// SWE100821: Default is loopback; use gateway.remote_access or host 0.0.0.0 for VPN/Tailscale
	if cfg.Gateway.Host != "127.0.0.1" {
		t.Error("Gateway host should default to loopback")
	}
	if cfg.Gateway.RemoteAccess {
		t.Error("Gateway remote_access should be off by default")
	}
	if cfg.Gateway.Port == 0 {
		t.Error("Gateway port should have default value")
	}
}

// TestDefaultConfig_Providers verifies provider structure
func TestDefaultConfig_Providers(t *testing.T) {
	cfg := DefaultConfig()

	// Verify all providers are empty by default
	if cfg.Providers.Anthropic.APIKey != "" {
		t.Error("Anthropic API key should be empty by default")
	}
	if cfg.Providers.OpenAI.APIKey != "" {
		t.Error("OpenAI API key should be empty by default")
	}
	if cfg.Providers.OpenRouter.APIKey != "" {
		t.Error("OpenRouter API key should be empty by default")
	}
	if cfg.Providers.Groq.APIKey != "" {
		t.Error("Groq API key should be empty by default")
	}
	// SWE100821: Zhipu provider removed (Chinese service)
	if cfg.Providers.Groq.APIKey != "" {
		t.Error("Groq API key should be empty by default")
	}
	if cfg.Providers.VLLM.APIKey != "" {
		t.Error("VLLM API key should be empty by default")
	}
	if cfg.Providers.Gemini.APIKey != "" {
		t.Error("Gemini API key should be empty by default")
	}
}

// TestDefaultConfig_Channels verifies channels are disabled by default
func TestDefaultConfig_Channels(t *testing.T) {
	cfg := DefaultConfig()

	// Verify all channels are disabled by default
	if cfg.Channels.WhatsApp.Enabled {
		t.Error("WhatsApp should be disabled by default")
	}
	if cfg.Channels.Telegram.Enabled {
		t.Error("Telegram should be disabled by default")
	}
	if cfg.Channels.Feishu.Enabled {
		t.Error("Feishu should be disabled by default")
	}
	if cfg.Channels.Discord.Enabled {
		t.Error("Discord should be disabled by default")
	}
	if cfg.Channels.MaixCam.Enabled {
		t.Error("MaixCam should be disabled by default")
	}
	if cfg.Channels.QQ.Enabled {
		t.Error("QQ should be disabled by default")
	}
	if cfg.Channels.DingTalk.Enabled {
		t.Error("DingTalk should be disabled by default")
	}
	if cfg.Channels.Slack.Enabled {
		t.Error("Slack should be disabled by default")
	}
}

// TestDefaultConfig_WebTools verifies web tools config
func TestDefaultConfig_WebTools(t *testing.T) {
	cfg := DefaultConfig()

	// Verify web tools defaults
	if cfg.Tools.Web.Brave.MaxResults != 5 {
		t.Error("Expected Brave MaxResults 5, got ", cfg.Tools.Web.Brave.MaxResults)
	}
	if cfg.Tools.Web.Brave.APIKey != "" {
		t.Error("Brave API key should be empty by default")
	}
	if cfg.Tools.Web.DuckDuckGo.MaxResults != 5 {
		t.Error("Expected DuckDuckGo MaxResults 5, got ", cfg.Tools.Web.DuckDuckGo.MaxResults)
	}
}

// TestDefaultConfig_SelfImproveDefaults verifies autonomous self-improve defaults (on, push, feature prefix)
func TestDefaultConfig_SelfImproveDefaults(t *testing.T) {
	cfg := DefaultConfig()
	if !cfg.SelfImprove.Enabled {
		t.Error("self_improve should be enabled by default")
	}
	if cfg.SelfImprove.IntervalHours != 168 {
		t.Errorf("expected interval 168h, got %d", cfg.SelfImprove.IntervalHours)
	}
	if !cfg.SelfImprove.AutoPush {
		t.Error("auto_push should be true by default")
	}
	if cfg.SelfImprove.BranchPrefix != "feature" {
		t.Errorf("expected branch_prefix feature, got %q", cfg.SelfImprove.BranchPrefix)
	}
}

// TestConfig_Complete verifies all config fields are set
func TestConfig_Complete(t *testing.T) {
	cfg := DefaultConfig()

	// Verify complete config structure
	if cfg.Agents.Defaults.Workspace == "" {
		t.Error("Workspace should not be empty")
	}
	if cfg.Agents.Defaults.Model == "" {
		t.Error("Model should not be empty")
	}
	if cfg.Agents.Defaults.Temperature == 0 {
		t.Error("Temperature should have default value")
	}
	if cfg.Agents.Defaults.MaxTokens == 0 {
		t.Error("MaxTokens should not be zero")
	}
	if cfg.Agents.Defaults.MaxToolIterations == 0 {
		t.Error("MaxToolIterations should not be zero")
	}
	if cfg.Gateway.Host != "127.0.0.1" {
		t.Error("Gateway host should default to loopback")
	}
	if cfg.Gateway.Port == 0 {
		t.Error("Gateway port should have default value")
	}
	if !cfg.Heartbeat.Enabled {
		t.Error("Heartbeat should be enabled by default")
	}
}

// TestLoadConfig_GatewayRemoteAccess verifies SWE100821: remote_access expands loopback to 0.0.0.0
func TestLoadConfig_GatewayRemoteAccess(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	raw := `{"agents":{"defaults":{"model":"m","max_tokens":1,"max_tool_iterations":1,"temperature":0.1}},"providers":{"vllm":{"api_base":"http://localhost:11434/v1"}},"gateway":{"host":"127.0.0.1","port":18790,"remote_access":true}}`
	if err := os.WriteFile(path, []byte(raw), 0600); err != nil {
		t.Fatal(err)
	}
	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Gateway.Host != "0.0.0.0" {
		t.Errorf("expected 0.0.0.0 with remote_access, got %q", cfg.Gateway.Host)
	}
	warns, valErr := cfg.Validate()
	if valErr != nil {
		t.Fatal(valErr)
	}
	if len(warns) == 0 {
		t.Error("expected remote_access validation warning")
	}
}
