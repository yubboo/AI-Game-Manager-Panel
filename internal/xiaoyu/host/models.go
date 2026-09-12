package host

import (
	"strings"
	"time"
)

const ActiveBrainPluginID = "xiaoyu.model.active"

type ModelProtocol string

const (
	ProtocolOpenAIResponses  ModelProtocol = "openai-responses"
	ProtocolDeepSeek         ModelProtocol = "deepseek"
	ProtocolOpenAICompatible ModelProtocol = "openai-compatible"
	ProtocolAnthropic        ModelProtocol = "anthropic"
	ProtocolGemini           ModelProtocol = "gemini"
	ProtocolCodexAppServer   ModelProtocol = "codex-app-server"
)

type ModelAuthMode string

const (
	ModelAuthAPIKey       ModelAuthMode = "api-key"
	ModelAuthSubscription ModelAuthMode = "subscription"
	ModelAuthLocal        ModelAuthMode = "local"
)

type ModelProviderKind string

const (
	ModelProviderAPI   ModelProviderKind = "model-api"
	ModelProviderLocal ModelProviderKind = "local-runtime"
	ModelProviderAgent ModelProviderKind = "agent-provider"
)

type ModelCapabilities struct {
	Adapter                      string   `json:"adapter"`
	Native                       bool     `json:"native"`
	Reasoning                    bool     `json:"reasoning"`
	ToolCalling                  bool     `json:"toolCalling"`
	ToolChoice                   bool     `json:"toolChoice"`
	ThinkingToolChoiceCompatible bool     `json:"thinkingToolChoiceCompatible"`
	ParallelToolCalls            bool     `json:"parallelToolCalls"`
	Replay                       bool     `json:"replay"`
	ReasoningReplay              bool     `json:"reasoningReplay"`
	Vision                       bool     `json:"vision"`
	Streaming                    bool     `json:"streaming"`
	InputModalities              []string `json:"inputModalities,omitempty"`
	NativeToolKinds              []string `json:"nativeToolKinds,omitempty"`
	Notes                        []string `json:"notes,omitempty"`
}

type ModelPreset struct {
	ID              string            `json:"id"`
	Name            string            `json:"name"`
	Description     string            `json:"description"`
	Protocol        ModelProtocol     `json:"protocol"`
	DefaultBaseURL  string            `json:"defaultBaseUrl"`
	Local           bool              `json:"local"`
	APIKeyOptional  bool              `json:"apiKeyOptional"`
	Kind            ModelProviderKind `json:"kind"`
	AuthModes       []ModelAuthMode   `json:"authModes"`
	DefaultAuthMode ModelAuthMode     `json:"defaultAuthMode"`
	Executable      string            `json:"executable,omitempty"`
	BrainEligible   bool              `json:"brainEligible"`
}

type ModelProfile struct {
	ID              string         `json:"id"`
	Name            string         `json:"name"`
	Provider        string         `json:"provider"`
	Protocol        ModelProtocol  `json:"protocol"`
	AuthMode        ModelAuthMode  `json:"authMode"`
	BaseURL         string         `json:"baseUrl"`
	Model           string         `json:"model"`
	Enabled         bool           `json:"enabled"`
	ContextWindow   int            `json:"contextWindow"`
	MaxOutputTokens int            `json:"maxOutputTokens"`
	ThinkingMode    string         `json:"thinkingMode"`
	ReasoningEffort string         `json:"reasoningEffort,omitempty"`
	Extra           map[string]any `json:"extra,omitempty"`
	SecretRef       string         `json:"-"`
	LastTestAt      time.Time      `json:"lastTestAt,omitempty"`
	LastTestOK      bool           `json:"lastTestOk,omitempty"`
	LastTestMessage string         `json:"lastTestMessage,omitempty"`
	CreatedAt       time.Time      `json:"createdAt"`
	UpdatedAt       time.Time      `json:"updatedAt"`
}

type ModelProfileView struct {
	ID              string            `json:"id"`
	Name            string            `json:"name"`
	Provider        string            `json:"provider"`
	Protocol        ModelProtocol     `json:"protocol"`
	AuthMode        ModelAuthMode     `json:"authMode"`
	ProviderKind    ModelProviderKind `json:"providerKind"`
	BrainEligible   bool              `json:"brainEligible"`
	BaseURL         string            `json:"baseUrl"`
	Model           string            `json:"model"`
	Enabled         bool              `json:"enabled"`
	ContextWindow   int               `json:"contextWindow"`
	MaxOutputTokens int               `json:"maxOutputTokens"`
	ThinkingMode    string            `json:"thinkingMode"`
	ReasoningEffort string            `json:"reasoningEffort,omitempty"`
	Extra           map[string]any    `json:"extra,omitempty"`
	Capabilities    ModelCapabilities `json:"capabilities"`
	HasAPIKey       bool              `json:"hasApiKey"`
	HasCredential   bool              `json:"hasCredential"`
	LastTestAt      time.Time         `json:"lastTestAt,omitempty"`
	LastTestOK      bool              `json:"lastTestOk,omitempty"`
	LastTestMessage string            `json:"lastTestMessage,omitempty"`
	IsDefault       bool              `json:"isDefault"`
	CreatedAt       time.Time         `json:"createdAt"`
	UpdatedAt       time.Time         `json:"updatedAt"`
}

type ModelCatalog struct {
	Presets        []ModelPreset      `json:"presets"`
	Profiles       []ModelProfileView `json:"profiles"`
	DefaultBrainID string             `json:"defaultBrainId,omitempty"`
	BrainReady     bool               `json:"brainReady"`
	Message        string             `json:"message"`
}

type SaveModelRequest struct {
	ID              string         `json:"id,omitempty"`
	Name            string         `json:"name"`
	Provider        string         `json:"provider"`
	Protocol        ModelProtocol  `json:"protocol"`
	AuthMode        ModelAuthMode  `json:"authMode,omitempty"`
	BaseURL         string         `json:"baseUrl"`
	Model           string         `json:"model"`
	APIKey          string         `json:"apiKey,omitempty"`
	Enabled         bool           `json:"enabled"`
	ContextWindow   int            `json:"contextWindow"`
	MaxOutputTokens int            `json:"maxOutputTokens"`
	ThinkingMode    string         `json:"thinkingMode"`
	ReasoningEffort string         `json:"reasoningEffort,omitempty"`
	Extra           map[string]any `json:"extra,omitempty"`
}

type ModelConnectionRequest struct {
	ID              string         `json:"id,omitempty"`
	Provider        string         `json:"provider"`
	Protocol        ModelProtocol  `json:"protocol"`
	AuthMode        ModelAuthMode  `json:"authMode,omitempty"`
	BaseURL         string         `json:"baseUrl"`
	Model           string         `json:"model"`
	APIKey          string         `json:"apiKey,omitempty"`
	ThinkingMode    string         `json:"thinkingMode,omitempty"`
	ReasoningEffort string         `json:"reasoningEffort,omitempty"`
	Extra           map[string]any `json:"extra,omitempty"`
}

type ModelConnectionResult struct {
	OK           bool              `json:"ok"`
	Message      string            `json:"message"`
	Endpoint     string            `json:"endpoint"`
	LatencyMS    int64             `json:"latencyMs"`
	Models       []string          `json:"models,omitempty"`
	Capabilities ModelCapabilities `json:"capabilities"`
}

func BuiltinModelPresets() []ModelPreset {
	presets := []ModelPreset{
		{ID: "openai", Name: "OpenAI API", Description: "OpenAI 官方 Responses API", Protocol: ProtocolOpenAIResponses, DefaultBaseURL: "https://api.openai.com/v1"},
		{ID: "openai-codex", Name: "OpenAI Codex 套餐", Description: "使用本机官方 Codex CLI 的 ChatGPT/Codex 订阅登录；0.3.0 继续使用官方状态探测，Brain Adapter 不会绕过 AGMP Tool/Approval 边界。", Protocol: ProtocolCodexAppServer, APIKeyOptional: true, Kind: ModelProviderAgent, AuthModes: []ModelAuthMode{ModelAuthSubscription}, DefaultAuthMode: ModelAuthSubscription, Executable: "codex", BrainEligible: false},
		{ID: "deepseek", Name: "DeepSeek", Description: "DeepSeek 官方原生接口", Protocol: ProtocolDeepSeek, DefaultBaseURL: "https://api.deepseek.com"},
		{ID: "minimax", Name: "MiniMax", Description: "MiniMax OpenAI 兼容接口", Protocol: ProtocolOpenAICompatible, DefaultBaseURL: "https://api.minimax.chat/v1"},
		{ID: "qwen", Name: "通义千问", Description: "阿里云百炼 OpenAI 兼容接口", Protocol: ProtocolOpenAICompatible, DefaultBaseURL: "https://dashscope.aliyuncs.com/compatible-mode/v1"},
		{ID: "volcengine", Name: "火山方舟", Description: "火山方舟 OpenAI 兼容接口", Protocol: ProtocolOpenAICompatible, DefaultBaseURL: "https://ark.cn-beijing.volces.com/api/v3"},
		{ID: "zhipu", Name: "智谱", Description: "智谱开放平台", Protocol: ProtocolOpenAICompatible, DefaultBaseURL: "https://open.bigmodel.cn/api/paas/v4"},
		{ID: "siliconflow", Name: "硅基流动", Description: "多模型聚合接口", Protocol: ProtocolOpenAICompatible, DefaultBaseURL: "https://api.siliconflow.cn/v1"},
		{ID: "openrouter", Name: "OpenRouter", Description: "多模型路由聚合接口", Protocol: ProtocolOpenAICompatible, DefaultBaseURL: "https://openrouter.ai/api/v1"},
		{ID: "gemini", Name: "Gemini", Description: "Google Gemini 官方接口", Protocol: ProtocolGemini, DefaultBaseURL: "https://generativelanguage.googleapis.com/v1beta"},
		{ID: "claude", Name: "Claude", Description: "Anthropic Claude 官方 Messages 接口", Protocol: ProtocolAnthropic, DefaultBaseURL: "https://api.anthropic.com"},
		{ID: "xai", Name: "xAI Grok", Description: "xAI OpenAI 兼容接口", Protocol: ProtocolOpenAICompatible, DefaultBaseURL: "https://api.x.ai/v1"},
		{ID: "ollama", Name: "Ollama 本地部署", Description: "本机 Ollama OpenAI 兼容服务", Protocol: ProtocolOpenAICompatible, DefaultBaseURL: "http://127.0.0.1:11434/v1", Local: true, APIKeyOptional: true},
		{ID: "lmstudio", Name: "LM Studio 本地部署", Description: "本机 LM Studio OpenAI 兼容服务", Protocol: ProtocolOpenAICompatible, DefaultBaseURL: "http://127.0.0.1:1234/v1", Local: true, APIKeyOptional: true},
		{ID: "custom", Name: "自定义 / 第三方中转", Description: "自定义兼容接口与中转站", Protocol: ProtocolOpenAICompatible, APIKeyOptional: true},
	}
	for i := range presets {
		if presets[i].Kind == "" {
			if presets[i].Local {
				presets[i].Kind = ModelProviderLocal
				presets[i].AuthModes = []ModelAuthMode{ModelAuthLocal, ModelAuthAPIKey}
				presets[i].DefaultAuthMode = ModelAuthLocal
			} else {
				presets[i].Kind = ModelProviderAPI
				presets[i].AuthModes = []ModelAuthMode{ModelAuthAPIKey}
				presets[i].DefaultAuthMode = ModelAuthAPIKey
			}
			presets[i].BrainEligible = true
		}
	}
	return presets
}

func ResolveModelCapabilities(profile ModelProfile) ModelCapabilities {
	// These flags describe what XiaoYu's current adapter can preserve, not every
	// theoretical feature of the vendor model. This prevents the Harness from
	// claiming capabilities that are then silently discarded.
	caps := ModelCapabilities{
		ToolCalling:                  true,
		ToolChoice:                   true,
		ThinkingToolChoiceCompatible: true,
		Replay:                       true,
		InputModalities:              []string{"text"},
		NativeToolKinds:              []string{"function"},
	}
	switch profile.Protocol {
	case ProtocolOpenAIResponses:
		caps.Adapter, caps.Native, caps.Reasoning, caps.Vision = "openai-responses-native", true, true, true
		caps.ReasoningReplay = true
		caps.InputModalities = []string{"text", "image"}
		caps.Notes = []string{"Responses reasoning items and encrypted replay are preserved", "function tools use the native Responses API"}
	case ProtocolDeepSeek:
		caps.Adapter, caps.Native, caps.Reasoning = "deepseek-native", true, true
		caps.ReasoningReplay = true
		// DeepSeek thinking mode accepts tools but some native endpoints reject
		// tool_choice. The adapter therefore omits tool_choice while thinking is
		// active unless the profile explicitly declares compatibility.
		caps.ThinkingToolChoiceCompatible = capabilityFlag(profile.Extra, "thinking_tool_choice")
		caps.Notes = []string{"DeepSeek reasoning_content is replayed across tool turns", "tool_choice is omitted in thinking mode unless explicitly declared compatible"}
	case ProtocolAnthropic:
		caps.Adapter, caps.Native, caps.Reasoning, caps.Vision = "anthropic-messages-native", true, true, true
		caps.ReasoningReplay = true
		caps.InputModalities = []string{"text", "image"}
		caps.Notes = []string{"Claude thinking/signature content is preserved"}
	case ProtocolCodexAppServer:
		caps.Adapter, caps.Native = "codex-app-server", true
		caps.ToolCalling, caps.ToolChoice, caps.Replay, caps.Streaming = false, false, false, false
		caps.Notes = []string{"subscription login status is probed through the official Codex CLI", "XiaoYu Brain Adapter is not enabled until Codex app-server tool mediation is wired through AGMP Host"}
	case ProtocolGemini:
		caps.Adapter, caps.Native, caps.Reasoning, caps.Vision = "gemini-native", true, true, true
		caps.ReasoningReplay = true
		caps.InputModalities = []string{"text", "image"}
		caps.Notes = []string{"Gemini thought signatures are preserved"}
	default:
		caps.Adapter = "openai-compatible"
		caps.ReasoningReplay = capabilityFlag(profile.Extra, "reasoning_replay")
		caps.Notes = []string{"Gateway feature support depends on the configured compatible endpoint"}
	}
	if profile.Protocol == ProtocolDeepSeek && capabilityFlag(profile.Extra, "vision") {
		caps.Vision = true
		caps.InputModalities = []string{"text", "image"}
	}
	if profile.Protocol == ProtocolOpenAICompatible {
		caps.Reasoning = capabilityFlag(profile.Extra, "reasoning")
		caps.Vision = capabilityFlag(profile.Extra, "vision")
		caps.Streaming = capabilityFlag(profile.Extra, "streaming")
		caps.ParallelToolCalls = capabilityFlag(profile.Extra, "parallel_tool_calls")
		if caps.Vision {
			caps.InputModalities = []string{"text", "image"}
		}
	}
	// 0.2.2's loop intentionally executes one provider tool call per Agent step.
	// Do not advertise parallel calls until replay/execution supports every call.
	if profile.Protocol != ProtocolOpenAICompatible {
		caps.ParallelToolCalls = false
	}
	return caps
}

func capabilityFlag(extra map[string]any, name string) bool {
	value, ok := extra["capabilities"]
	if !ok {
		return false
	}
	m, ok := value.(map[string]any)
	if !ok {
		return false
	}
	flag, _ := m[strings.TrimSpace(name)].(bool)
	return flag
}
