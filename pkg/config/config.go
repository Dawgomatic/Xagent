package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/caarlos0/env/v11"
)

// FlexibleStringSlice is a []string that also accepts JSON numbers,
// so allow_from can contain both "123" and 123.
type FlexibleStringSlice []string

func (f *FlexibleStringSlice) UnmarshalJSON(data []byte) error {
	// Try []string first
	var ss []string
	if err := json.Unmarshal(data, &ss); err == nil {
		*f = ss
		return nil
	}

	// Try []interface{} to handle mixed types
	var raw []interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	result := make([]string, 0, len(raw))
	for _, v := range raw {
		switch val := v.(type) {
		case string:
			result = append(result, val)
		case float64:
			result = append(result, fmt.Sprintf("%.0f", val))
		default:
			result = append(result, fmt.Sprintf("%v", val))
		}
	}
	*f = result
	return nil
}

type Config struct {
	Agents    AgentsConfig    `json:"agents"`
	Channels  ChannelsConfig  `json:"channels"`
	Providers ProvidersConfig `json:"providers"`
	Gateway   GatewayConfig   `json:"gateway"`
	Tools     ToolsConfig     `json:"tools"`
	Heartbeat HeartbeatConfig `json:"heartbeat"`
	Devices   DevicesConfig   `json:"devices"`
	Vault     VaultConfig     `json:"vault"`
	MCP            MCPConfig            `json:"mcp"`             // SWE100821: Config-driven MCP server registration
	Phone          PhoneConfig          `json:"phone"`           // SWE100821: USB-attached phone access via ADB/libimobiledevice
	SemanticMemory SemanticMemoryConfig `json:"semantic_memory"` // SWE100821: Vector memory via Qdrant + Ollama embeddings
	mu             sync.RWMutex
}

// SWE100821: MCPConfig holds config-driven MCP server definitions.
type MCPConfig struct {
	Servers []MCPServerConfig `json:"servers"`
}

// MCPServerConfig defines a single MCP server to connect to at startup.
type MCPServerConfig struct {
	Name    string   `json:"name"`
	Command string   `json:"command"`
	Args    []string `json:"args"`
	Enabled bool     `json:"enabled"`
}

// VaultConfig configures the Obsidian-compatible knowledge vault.
// When enabled, xagent writes markdown notes with [[wikilinks]] for graph view.
type VaultConfig struct {
	Enabled bool   `json:"enabled" env:"XAGENT_VAULT_ENABLED"`
	Path    string `json:"path" env:"XAGENT_VAULT_PATH"` // default: ~/.xagent/vault
}

type AgentsConfig struct {
	Defaults AgentDefaults `json:"defaults"`
}

type AgentDefaults struct {
	Workspace           string  `json:"workspace" env:"XAGENT_AGENTS_DEFAULTS_WORKSPACE"`
	RestrictToWorkspace bool    `json:"restrict_to_workspace" env:"XAGENT_AGENTS_DEFAULTS_RESTRICT_TO_WORKSPACE"`
	Provider            string  `json:"provider" env:"XAGENT_AGENTS_DEFAULTS_PROVIDER"`
	Model               string  `json:"model" env:"XAGENT_AGENTS_DEFAULTS_MODEL"`
	MaxTokens           int     `json:"max_tokens" env:"XAGENT_AGENTS_DEFAULTS_MAX_TOKENS"`
	// SWE100821: ContextWindow is the model's total context size (input+output) in tokens.
	// Used for summarization thresholds. 0 = auto-detect from max_tokens*4 or 8192 default.
	ContextWindow       int     `json:"context_window" env:"XAGENT_AGENTS_DEFAULTS_CONTEXT_WINDOW"`
	Temperature         float64 `json:"temperature" env:"XAGENT_AGENTS_DEFAULTS_TEMPERATURE"`
	MaxToolIterations   int     `json:"max_tool_iterations" env:"XAGENT_AGENTS_DEFAULTS_MAX_TOOL_ITERATIONS"`
	// SWE100821: Per-message timeout in seconds. Covers entire processing (all LLM calls + tool iterations).
	// 0 = default (5min desktop, 15min embedded). Set higher for slow hardware or complex multi-tool tasks.
	MessageTimeoutSecs  int     `json:"message_timeout_secs" env:"XAGENT_AGENTS_DEFAULTS_MESSAGE_TIMEOUT_SECS"`
}

type ChannelsConfig struct {
	WhatsApp WhatsAppConfig `json:"whatsapp"`
	Telegram TelegramConfig `json:"telegram"`
	Feishu   FeishuConfig   `json:"feishu"`
	Discord  DiscordConfig  `json:"discord"`
	MaixCam  MaixCamConfig  `json:"maixcam"`
	QQ       QQConfig       `json:"qq"`
	DingTalk DingTalkConfig `json:"dingtalk"`
	Slack    SlackConfig    `json:"slack"`
	LINE     LINEConfig     `json:"line"`
	OneBot   OneBotConfig   `json:"onebot"`
}

type WhatsAppConfig struct {
	Enabled   bool                `json:"enabled" env:"XAGENT_CHANNELS_WHATSAPP_ENABLED"`
	SessionDB string              `json:"session_db" env:"XAGENT_CHANNELS_WHATSAPP_SESSION_DB"`
	AllowFrom FlexibleStringSlice `json:"allow_from" env:"XAGENT_CHANNELS_WHATSAPP_ALLOW_FROM"`
}

type TelegramConfig struct {
	Enabled   bool                `json:"enabled" env:"XAGENT_CHANNELS_TELEGRAM_ENABLED"`
	Token     string              `json:"token" env:"XAGENT_CHANNELS_TELEGRAM_TOKEN"`
	Proxy     string              `json:"proxy" env:"XAGENT_CHANNELS_TELEGRAM_PROXY"`
	AllowFrom FlexibleStringSlice `json:"allow_from" env:"XAGENT_CHANNELS_TELEGRAM_ALLOW_FROM"`
}

// DISABLED: Chinese service (ByteDance/Feishu). Struct retained for config file compatibility.
type FeishuConfig struct {
	Enabled           bool                `json:"enabled" env:"XAGENT_CHANNELS_FEISHU_ENABLED"`
	AppID             string              `json:"app_id" env:"XAGENT_CHANNELS_FEISHU_APP_ID"`
	AppSecret         string              `json:"app_secret" env:"XAGENT_CHANNELS_FEISHU_APP_SECRET"`
	EncryptKey        string              `json:"encrypt_key" env:"XAGENT_CHANNELS_FEISHU_ENCRYPT_KEY"`
	VerificationToken string              `json:"verification_token" env:"XAGENT_CHANNELS_FEISHU_VERIFICATION_TOKEN"`
	AllowFrom         FlexibleStringSlice `json:"allow_from" env:"XAGENT_CHANNELS_FEISHU_ALLOW_FROM"`
}

type DiscordConfig struct {
	Enabled   bool                `json:"enabled" env:"XAGENT_CHANNELS_DISCORD_ENABLED"`
	Token     string              `json:"token" env:"XAGENT_CHANNELS_DISCORD_TOKEN"`
	AllowFrom FlexibleStringSlice `json:"allow_from" env:"XAGENT_CHANNELS_DISCORD_ALLOW_FROM"`
}

type MaixCamConfig struct {
	Enabled   bool                `json:"enabled" env:"XAGENT_CHANNELS_MAIXCAM_ENABLED"`
	Host      string              `json:"host" env:"XAGENT_CHANNELS_MAIXCAM_HOST"`
	Port      int                 `json:"port" env:"XAGENT_CHANNELS_MAIXCAM_PORT"`
	AllowFrom FlexibleStringSlice `json:"allow_from" env:"XAGENT_CHANNELS_MAIXCAM_ALLOW_FROM"`
}

// DISABLED: Chinese service (Tencent/QQ). Struct retained for config file compatibility.
type QQConfig struct {
	Enabled   bool                `json:"enabled" env:"XAGENT_CHANNELS_QQ_ENABLED"`
	AppID     string              `json:"app_id" env:"XAGENT_CHANNELS_QQ_APP_ID"`
	AppSecret string              `json:"app_secret" env:"XAGENT_CHANNELS_QQ_APP_SECRET"`
	AllowFrom FlexibleStringSlice `json:"allow_from" env:"XAGENT_CHANNELS_QQ_ALLOW_FROM"`
}

// DISABLED: Chinese service (Alibaba/DingTalk). Struct retained for config file compatibility.
type DingTalkConfig struct {
	Enabled      bool                `json:"enabled" env:"XAGENT_CHANNELS_DINGTALK_ENABLED"`
	ClientID     string              `json:"client_id" env:"XAGENT_CHANNELS_DINGTALK_CLIENT_ID"`
	ClientSecret string              `json:"client_secret" env:"XAGENT_CHANNELS_DINGTALK_CLIENT_SECRET"`
	AllowFrom    FlexibleStringSlice `json:"allow_from" env:"XAGENT_CHANNELS_DINGTALK_ALLOW_FROM"`
}

type SlackConfig struct {
	Enabled   bool                `json:"enabled" env:"XAGENT_CHANNELS_SLACK_ENABLED"`
	BotToken  string              `json:"bot_token" env:"XAGENT_CHANNELS_SLACK_BOT_TOKEN"`
	AppToken  string              `json:"app_token" env:"XAGENT_CHANNELS_SLACK_APP_TOKEN"`
	AllowFrom FlexibleStringSlice `json:"allow_from" env:"XAGENT_CHANNELS_SLACK_ALLOW_FROM"`
}

type LINEConfig struct {
	Enabled            bool                `json:"enabled" env:"XAGENT_CHANNELS_LINE_ENABLED"`
	ChannelSecret      string              `json:"channel_secret" env:"XAGENT_CHANNELS_LINE_CHANNEL_SECRET"`
	ChannelAccessToken string              `json:"channel_access_token" env:"XAGENT_CHANNELS_LINE_CHANNEL_ACCESS_TOKEN"`
	WebhookHost        string              `json:"webhook_host" env:"XAGENT_CHANNELS_LINE_WEBHOOK_HOST"`
	WebhookPort        int                 `json:"webhook_port" env:"XAGENT_CHANNELS_LINE_WEBHOOK_PORT"`
	WebhookPath        string              `json:"webhook_path" env:"XAGENT_CHANNELS_LINE_WEBHOOK_PATH"`
	AllowFrom          FlexibleStringSlice `json:"allow_from" env:"XAGENT_CHANNELS_LINE_ALLOW_FROM"`
}

// DISABLED: Chinese QQ bot protocol. Struct retained for config file compatibility.
type OneBotConfig struct {
	Enabled            bool                `json:"enabled" env:"XAGENT_CHANNELS_ONEBOT_ENABLED"`
	WSUrl              string              `json:"ws_url" env:"XAGENT_CHANNELS_ONEBOT_WS_URL"`
	AccessToken        string              `json:"access_token" env:"XAGENT_CHANNELS_ONEBOT_ACCESS_TOKEN"`
	ReconnectInterval  int                 `json:"reconnect_interval" env:"XAGENT_CHANNELS_ONEBOT_RECONNECT_INTERVAL"`
	GroupTriggerPrefix []string            `json:"group_trigger_prefix" env:"XAGENT_CHANNELS_ONEBOT_GROUP_TRIGGER_PREFIX"`
	AllowFrom          FlexibleStringSlice `json:"allow_from" env:"XAGENT_CHANNELS_ONEBOT_ALLOW_FROM"`
}

type HeartbeatConfig struct {
	Enabled  bool `json:"enabled" env:"XAGENT_HEARTBEAT_ENABLED"`
	Interval int  `json:"interval" env:"XAGENT_HEARTBEAT_INTERVAL"` // minutes, min 5
}

type DevicesConfig struct {
	Enabled    bool `json:"enabled" env:"XAGENT_DEVICES_ENABLED"`
	MonitorUSB bool `json:"monitor_usb" env:"XAGENT_DEVICES_MONITOR_USB"`
}

// SWE100821: PhoneConfig controls USB-attached phone access via ADB or libimobiledevice.
type PhoneConfig struct {
	Enabled    bool     `json:"enabled" env:"XAGENT_PHONE_ENABLED"`
	ADBPath    string   `json:"adb_path" env:"XAGENT_PHONE_ADB_PATH"`
	AutoDetect bool     `json:"auto_detect" env:"XAGENT_PHONE_AUTO_DETECT"`
	Serial     string   `json:"serial" env:"XAGENT_PHONE_SERIAL"`
	DenyShell  []string `json:"deny_shell"`
}

// SWE100821: SemanticMemoryConfig controls vector-based memory via Qdrant + Ollama embeddings.
// When qdrant_url is reachable, the agent stores and retrieves memories by semantic similarity.
// Falls back gracefully to file-based memory if Qdrant is unavailable.
type SemanticMemoryConfig struct {
	QdrantURL  string `json:"qdrant_url" env:"XAGENT_SEMANTIC_MEMORY_QDRANT_URL"`
	OllamaURL  string `json:"ollama_url" env:"XAGENT_SEMANTIC_MEMORY_OLLAMA_URL"`
	Collection string `json:"collection" env:"XAGENT_SEMANTIC_MEMORY_COLLECTION"`
	EmbedModel string `json:"embed_model" env:"XAGENT_SEMANTIC_MEMORY_EMBED_MODEL"`
}

type ProvidersConfig struct {
	Anthropic     ProviderConfig `json:"anthropic"`
	OpenAI        ProviderConfig `json:"openai"`
	OpenRouter    ProviderConfig `json:"openrouter"`
	Groq          ProviderConfig `json:"groq"`
	VLLM          ProviderConfig `json:"vllm"`
	Gemini        ProviderConfig `json:"gemini"`
	Nvidia        ProviderConfig `json:"nvidia"`
	GitHubCopilot ProviderConfig `json:"github_copilot"`
	BitNet        BitNetConfig   `json:"bitnet"`
	PicoLM        PicoLMConfig   `json:"picolm"` // SWE100821: Local-first inference for embedded/edge
	RL            RLConfig       `json:"rl"`
}

// BitNetConfig configures the 1.58-bit localized LLM runtime.
type BitNetConfig struct {
	Enabled     bool   `json:"enabled" env:"XAGENT_PROVIDERS_BITNET_ENABLED"`
	Model       string `json:"model" env:"XAGENT_PROVIDERS_BITNET_MODEL"`
	Runtime     string `json:"runtime" env:"XAGENT_PROVIDERS_BITNET_RUNTIME"`       // e.g., "llama.cpp", "python"
	QuantType   string `json:"quant_type" env:"XAGENT_PROVIDERS_BITNET_QUANT_TYPE"` // e.g., "i2_s", "tl1"
	ContextSize int    `json:"context_size" env:"XAGENT_PROVIDERS_BITNET_CONTEXT_SIZE"`
	Threads     int    `json:"threads" env:"XAGENT_PROVIDERS_BITNET_THREADS"`
}

// SWE100821: PicoLMConfig configures the local PicoLM inference engine.
// Pure C, 45MB RAM, 80KB binary, zero dependencies. Designed for
// Jetson Xavier, Raspberry Pi, and other ARM/embedded platforms.
type PicoLMConfig struct {
	Enabled     bool    `json:"enabled" env:"XAGENT_PROVIDERS_PICOLM_ENABLED"`
	Binary      string  `json:"binary" env:"XAGENT_PROVIDERS_PICOLM_BINARY"`             // path to picolm binary
	ModelPath   string  `json:"model_path" env:"XAGENT_PROVIDERS_PICOLM_MODEL_PATH"`     // path to .gguf model file
	MaxTokens   int     `json:"max_tokens" env:"XAGENT_PROVIDERS_PICOLM_MAX_TOKENS"`     // generation limit per call
	Threads     int     `json:"threads" env:"XAGENT_PROVIDERS_PICOLM_THREADS"`            // CPU threads for matmul
	ContextSize int     `json:"context_size" env:"XAGENT_PROVIDERS_PICOLM_CONTEXT_SIZE"` // context window override
	Temperature float64 `json:"temperature" env:"XAGENT_PROVIDERS_PICOLM_TEMPERATURE"`
	CachePath   string  `json:"cache_path" env:"XAGENT_PROVIDERS_PICOLM_CACHE_PATH"` // KV cache file (skip prefill on reuse)
}

// RLConfig configures the OpenClaw-RL reinforcement learning server.
// When enabled, xagent routes LLM requests through the RL proxy server
// which collects training data from live conversations.
type RLConfig struct {
	Enabled      bool   `json:"enabled" env:"XAGENT_RL_ENABLED"`
	ServerURL    string `json:"server_url" env:"XAGENT_RL_SERVER_URL"` // e.g. "http://gpu-box:30000/v1"
	APIKey       string `json:"api_key" env:"XAGENT_RL_API_KEY"`
	Model        string `json:"model" env:"XAGENT_RL_MODEL"`                 // e.g. "qwen3-4b"
	FeedbackMode string `json:"feedback_mode" env:"XAGENT_RL_FEEDBACK_MODE"` // "implicit" or "explicit"
}

type ProviderConfig struct {
	APIKey      string `json:"api_key" env:"XAGENT_PROVIDERS_{{.Name}}_API_KEY"`
	APIBase     string `json:"api_base" env:"XAGENT_PROVIDERS_{{.Name}}_API_BASE"`
	Proxy       string `json:"proxy,omitempty" env:"XAGENT_PROVIDERS_{{.Name}}_PROXY"`
	AuthMethod  string `json:"auth_method,omitempty" env:"XAGENT_PROVIDERS_{{.Name}}_AUTH_METHOD"`
	ConnectMode string `json:"connect_mode,omitempty" env:"XAGENT_PROVIDERS_{{.Name}}_CONNECT_MODE"` //only for Github Copilot, `stdio` or `grpc`
}

type GatewayConfig struct {
	Host string `json:"host" env:"XAGENT_GATEWAY_HOST"`
	Port int    `json:"port" env:"XAGENT_GATEWAY_PORT"`
}

type BraveConfig struct {
	Enabled    bool   `json:"enabled" env:"XAGENT_TOOLS_WEB_BRAVE_ENABLED"`
	APIKey     string `json:"api_key" env:"XAGENT_TOOLS_WEB_BRAVE_API_KEY"`
	MaxResults int    `json:"max_results" env:"XAGENT_TOOLS_WEB_BRAVE_MAX_RESULTS"`
}

type DuckDuckGoConfig struct {
	Enabled    bool `json:"enabled" env:"XAGENT_TOOLS_WEB_DUCKDUCKGO_ENABLED"`
	MaxResults int  `json:"max_results" env:"XAGENT_TOOLS_WEB_DUCKDUCKGO_MAX_RESULTS"`
}

type WebToolsConfig struct {
	Brave      BraveConfig      `json:"brave"`
	DuckDuckGo DuckDuckGoConfig `json:"duckduckgo"`
}

// SWE100821: Exec tool config — controls which shell commands are allowed/denied
type ExecConfig struct {
	AllowNetwork bool     `json:"allow_network" env:"XAGENT_TOOLS_EXEC_ALLOW_NETWORK"` // Allow curl, wget, ssh, etc.
	AllowScripts bool     `json:"allow_scripts" env:"XAGENT_TOOLS_EXEC_ALLOW_SCRIPTS"` // Allow python3, node, etc.
	DenyCommands []string `json:"deny_commands"`                                        // Additional deny regex patterns
	TimeoutSecs  int      `json:"timeout_secs" env:"XAGENT_TOOLS_EXEC_TIMEOUT_SECS"`   // Command timeout (default 60)
}

// SWE100821: VisionConfig controls which Ollama vision model to use for image analysis
type VisionConfig struct {
	Model     string `json:"model" env:"XAGENT_TOOLS_VISION_MODEL"`          // Vision model name (default: moondream)
	OllamaURL string `json:"ollama_url" env:"XAGENT_TOOLS_VISION_OLLAMA_URL"` // Ollama endpoint (default: http://localhost:11434)
}

type ToolsConfig struct {
	Web    WebToolsConfig `json:"web"`
	Exec   ExecConfig     `json:"exec"`
	Vision VisionConfig   `json:"vision"` // SWE100821: Configurable vision model
}

func DefaultConfig() *Config {
	return &Config{
		Agents: AgentsConfig{
			Defaults: AgentDefaults{
				Workspace:           "~/.xagent/workspace",
				RestrictToWorkspace: true,
				Provider:            "",
				Model:               "glm-4.7",
				MaxTokens:           8192,
				Temperature:         0.7,
				MaxToolIterations:   20,
			},
		},
		Channels: ChannelsConfig{
			WhatsApp: WhatsAppConfig{
				Enabled:   false,
				SessionDB: "",
				AllowFrom: FlexibleStringSlice{},
			},
			Telegram: TelegramConfig{
				Enabled:   false,
				Token:     "",
				AllowFrom: FlexibleStringSlice{},
			},
			Feishu: FeishuConfig{
				Enabled:           false,
				AppID:             "",
				AppSecret:         "",
				EncryptKey:        "",
				VerificationToken: "",
				AllowFrom:         FlexibleStringSlice{},
			},
			Discord: DiscordConfig{
				Enabled:   false,
				Token:     "",
				AllowFrom: FlexibleStringSlice{},
			},
			MaixCam: MaixCamConfig{
				Enabled:   false,
				Host:      "0.0.0.0",
				Port:      18790,
				AllowFrom: FlexibleStringSlice{},
			},
			QQ: QQConfig{
				Enabled:   false,
				AppID:     "",
				AppSecret: "",
				AllowFrom: FlexibleStringSlice{},
			},
			DingTalk: DingTalkConfig{
				Enabled:      false,
				ClientID:     "",
				ClientSecret: "",
				AllowFrom:    FlexibleStringSlice{},
			},
			Slack: SlackConfig{
				Enabled:   false,
				BotToken:  "",
				AppToken:  "",
				AllowFrom: FlexibleStringSlice{},
			},
			LINE: LINEConfig{
				Enabled:            false,
				ChannelSecret:      "",
				ChannelAccessToken: "",
				WebhookHost:        "0.0.0.0",
				WebhookPort:        18791,
				WebhookPath:        "/webhook/line",
				AllowFrom:          FlexibleStringSlice{},
			},
			OneBot: OneBotConfig{
				Enabled:            false,
				WSUrl:              "ws://127.0.0.1:3001",
				AccessToken:        "",
				ReconnectInterval:  5,
				GroupTriggerPrefix: []string{},
				AllowFrom:          FlexibleStringSlice{},
			},
		},
		// SWE100821: Removed Zhipu, Moonshot, ShengSuanYun (Chinese providers stripped)
		Providers: ProvidersConfig{
			Anthropic:  ProviderConfig{},
			OpenAI:     ProviderConfig{},
			OpenRouter: ProviderConfig{},
			Groq:       ProviderConfig{},
			VLLM:       ProviderConfig{},
			Gemini:     ProviderConfig{},
			Nvidia:     ProviderConfig{},
		BitNet: BitNetConfig{
			Enabled:     false,
			Model:       "bitnet_b1_58-3B",
			Runtime:     "python",
			QuantType:   "i2_s",
			ContextSize: 2048,
			Threads:     4,
		},
		// SWE100821: PicoLM defaults — auto-enabled on Tegra/RPi by hwprofile
		PicoLM: PicoLMConfig{
			Enabled:     false,
			Binary:      "picolm",
			ModelPath:   "/opt/picolm/models/tinyllama-1.1b-chat-v1.0.Q4_K_M.gguf",
			MaxTokens:   256,
			Threads:     4,
			ContextSize: 2048,
			Temperature: 0.7,
			CachePath:   "",
		},
			RL: RLConfig{
				Enabled:      false,
				ServerURL:    "",
				APIKey:       "",
				Model:        "qwen3-4b",
				FeedbackMode: "implicit",
			},
		},
		// SWE100821: Default to localhost — dashboard has no auth, so binding to
		// 0.0.0.0 exposes config/memory/chat to anyone on the LAN.
		// Set to "0.0.0.0" explicitly in config.json if LAN access is intended.
		Gateway: GatewayConfig{
			Host: "127.0.0.1",
			Port: 18790,
		},
		Tools: ToolsConfig{
			Web: WebToolsConfig{
				Brave: BraveConfig{
					Enabled:    false,
					APIKey:     "",
					MaxResults: 5,
				},
				DuckDuckGo: DuckDuckGoConfig{
					Enabled:    true,
					MaxResults: 5,
				},
			},
		},
		Heartbeat: HeartbeatConfig{
			Enabled:  true,
			Interval: 30, // default 30 minutes
		},
		Devices: DevicesConfig{
			Enabled:    false,
			MonitorUSB: true,
		},
		Vault: VaultConfig{
			Enabled: true,
			Path:    "~/.xagent/vault",
		},
		// SWE100821: Explicit empty MCP defaults for clarity
		MCP: MCPConfig{
			Servers: []MCPServerConfig{},
		},
		// SWE100821: Phone access defaults — disabled, auto-detect first device
		Phone: PhoneConfig{
			Enabled:    false,
			ADBPath:    "adb",
			AutoDetect: true,
			Serial:     "",
			DenyShell:  []string{},
		},
		// SWE100821: Semantic memory defaults — empty strings use hardcoded defaults in pkg/memory/semantic.go
		SemanticMemory: SemanticMemoryConfig{
			QdrantURL:  "http://localhost:6333",
			OllamaURL:  "http://localhost:11434",
			Collection: "xagent_memory",
			EmbedModel: "nomic-embed-text",
		},
	}
}

func LoadConfig(path string) (*Config, error) {
	cfg := DefaultConfig()

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return nil, err
	}

	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, err
	}

	if err := env.Parse(cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

func SaveConfig(path string, cfg *Config) error {
	cfg.mu.RLock()
	defer cfg.mu.RUnlock()

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	// SWE100821: Config may contain API keys; use 0600 (owner-only) permissions
	return os.WriteFile(path, data, 0600)
}

func (c *Config) WorkspacePath() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return expandHome(c.Agents.Defaults.Workspace)
}

// SWE100821: Cleaned up references to removed Chinese providers (Zhipu, ShengSuanYun)
func (c *Config) GetAPIKey() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.Providers.OpenRouter.APIKey != "" {
		return c.Providers.OpenRouter.APIKey
	}
	if c.Providers.Anthropic.APIKey != "" {
		return c.Providers.Anthropic.APIKey
	}
	if c.Providers.OpenAI.APIKey != "" {
		return c.Providers.OpenAI.APIKey
	}
	if c.Providers.Gemini.APIKey != "" {
		return c.Providers.Gemini.APIKey
	}
	if c.Providers.Groq.APIKey != "" {
		return c.Providers.Groq.APIKey
	}
	if c.Providers.VLLM.APIKey != "" {
		return c.Providers.VLLM.APIKey
	}
	if c.Providers.Nvidia.APIKey != "" {
		return c.Providers.Nvidia.APIKey
	}
	return ""
}

func (c *Config) GetAPIBase() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.Providers.OpenRouter.APIKey != "" {
		if c.Providers.OpenRouter.APIBase != "" {
			return c.Providers.OpenRouter.APIBase
		}
		return "https://openrouter.ai/api/v1"
	}
	// SWE100821: Removed Zhipu reference
	if c.Providers.VLLM.APIKey != "" && c.Providers.VLLM.APIBase != "" {
		return c.Providers.VLLM.APIBase
	}
	return ""
}

// Validate checks the config for common misconfigurations.
// Returns a list of warnings (non-fatal) and an error if the config is unusable.
// SWE100821: Prevents silent failures from empty/invalid configs.
func (c *Config) Validate() (warnings []string, err error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// Check that at least one provider has credentials or a local API base
	// SWE100821: Cleaned up references to removed Chinese providers
	hasProvider := c.Providers.OpenRouter.APIKey != "" ||
		c.Providers.Anthropic.APIKey != "" || c.Providers.Anthropic.AuthMethod != "" ||
		c.Providers.OpenAI.APIKey != "" || c.Providers.OpenAI.AuthMethod != "" ||
		c.Providers.Groq.APIKey != "" ||
		c.Providers.Gemini.APIKey != "" ||
		c.Providers.VLLM.APIBase != "" ||
		c.Providers.Nvidia.APIKey != "" ||
		c.Providers.GitHubCopilot.APIBase != "" ||
		(c.Providers.RL.Enabled && c.Providers.RL.ServerURL != "") ||
		c.Providers.BitNet.Enabled ||
		c.Providers.PicoLM.Enabled // SWE100821: PicoLM is fully local, no API key needed

	if !hasProvider {
		// For CLI providers that don't need keys
		provName := strings.ToLower(c.Agents.Defaults.Provider)
		if provName != "claude-cli" && provName != "claudecode" && provName != "claude-code" {
			err = fmt.Errorf("no LLM provider configured: add an API key or set vllm.api_base in config")
			return
		}
	}

	// Warn if model is empty
	if c.Agents.Defaults.Model == "" {
		warnings = append(warnings, "agents.defaults.model is empty; provider default will be used")
	}

	// Check gateway port range
	if c.Gateway.Port < 1 || c.Gateway.Port > 65535 {
		err = fmt.Errorf("gateway.port %d is invalid (must be 1-65535)", c.Gateway.Port)
		return
	}

	// Warn if workspace path is empty
	if c.Agents.Defaults.Workspace == "" {
		warnings = append(warnings, "agents.defaults.workspace is empty; defaulting to current directory")
	}

	// SWE100821: Validate numeric bounds — 0 iterations means the LLM loop never runs,
	// negative tokens/timeout cause undefined behavior.
	if c.Agents.Defaults.MaxToolIterations < 0 {
		err = fmt.Errorf("agents.defaults.max_tool_iterations cannot be negative (%d)", c.Agents.Defaults.MaxToolIterations)
		return
	}
	if c.Agents.Defaults.MaxToolIterations == 0 {
		warnings = append(warnings, "agents.defaults.max_tool_iterations is 0; agent will not use tools")
	}
	if c.Agents.Defaults.MaxTokens < 0 {
		err = fmt.Errorf("agents.defaults.max_tokens cannot be negative (%d)", c.Agents.Defaults.MaxTokens)
		return
	}
	if c.Agents.Defaults.MessageTimeoutSecs < 0 {
		err = fmt.Errorf("agents.defaults.message_timeout_secs cannot be negative (%d)", c.Agents.Defaults.MessageTimeoutSecs)
		return
	}

	return
}

func expandHome(path string) string {
	if path == "" {
		return path
	}
	if path[0] == '~' {
		home, _ := os.UserHomeDir()
		if len(path) > 1 && path[1] == '/' {
			return home + path[1:]
		}
		return home
	}
	return path
}
