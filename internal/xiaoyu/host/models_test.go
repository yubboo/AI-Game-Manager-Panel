package host

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestModelManagerKeepsAPIKeyOutOfProfileJSON(t *testing.T) {
	root := t.TempDir()
	manager := NewModelManager(filepath.Join(root, "models.json"), filepath.Join(root, "vault"))
	saved, err := manager.Save(SaveModelRequest{Name: "Main", Provider: "openai", Protocol: ProtocolOpenAICompatible, BaseURL: "https://api.openai.com/v1", Model: "gpt-test", APIKey: "sk-super-secret", Enabled: true})
	if err != nil {
		t.Fatal(err)
	}
	if !saved.HasAPIKey || !saved.IsDefault {
		t.Fatalf("unexpected profile: %+v", saved)
	}
	raw, err := os.ReadFile(filepath.Join(root, "models.json"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "sk-super-secret") {
		t.Fatal("API key leaked into models.json")
	}
	profile, secret, err := manager.Default()
	if err != nil {
		t.Fatal(err)
	}
	if profile.ID != saved.ID || secret != "sk-super-secret" {
		t.Fatalf("unexpected default: %+v secret=%q", profile, secret)
	}
}

func TestModelManagerPreservesStoredSecretOnBlankEdit(t *testing.T) {
	root := t.TempDir()
	manager := NewModelManager(filepath.Join(root, "models.json"), filepath.Join(root, "vault"))
	saved, err := manager.Save(SaveModelRequest{Name: "Main", Provider: "openai", Protocol: ProtocolOpenAICompatible, BaseURL: "https://api.openai.com/v1", Model: "gpt-a", APIKey: "sk-old", Enabled: true})
	if err != nil {
		t.Fatal(err)
	}
	_, err = manager.Save(SaveModelRequest{ID: saved.ID, Name: "Main 2", Provider: "openai", Protocol: ProtocolOpenAICompatible, BaseURL: "https://api.openai.com/v1", Model: "gpt-b", Enabled: true})
	if err != nil {
		t.Fatal(err)
	}
	profile, secret, err := manager.Default()
	if err != nil {
		t.Fatal(err)
	}
	if profile.Model != "gpt-b" || secret != "sk-old" {
		t.Fatalf("secret was not preserved: %+v %q", profile, secret)
	}
}

func TestDiscoverOpenAICompatibleModels(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/models" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer sk-test" {
			t.Fatalf("missing auth: %q", r.Header.Get("Authorization"))
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"data": []map[string]any{{"id": "model-b"}, {"id": "model-a"}}})
	}))
	defer server.Close()
	models, endpoint, err := DiscoverModelNames(context.Background(), ModelProfile{Protocol: ProtocolOpenAICompatible, BaseURL: server.URL + "/v1"}, "sk-test")
	if err != nil {
		t.Fatal(err)
	}
	if endpoint != server.URL+"/v1/models" || len(models) != 2 || models[0] != "model-a" {
		t.Fatalf("unexpected discovery: %s %#v", endpoint, models)
	}
}

func TestConfiguredOpenAIBrainReturnsToolDecision(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		tools, ok := body["tools"].([]any)
		if !ok || len(tools) != 1 {
			t.Fatalf("tool schema missing: %#v", body["tools"])
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"choices": []map[string]any{{"message": map[string]any{"tool_calls": []map[string]any{{"type": "function", "function": map[string]any{"name": "system.info", "arguments": `{}`}}}}}}})
	}))
	defer server.Close()
	turn, err := callOpenAICompatibleModel(context.Background(), ModelProfile{Protocol: ProtocolOpenAICompatible, BaseURL: server.URL + "/v1", Model: "test", MaxOutputTokens: 100}, "sk-test", Frame{Goal: "inspect", Step: 1, Tools: []xiaoyuToolView{{Name: "system.info", Description: "info", Parameters: map[string]any{"type": "object"}}}}, BrainPrompt{System: "system", User: "user"})
	if err != nil {
		t.Fatal(err)
	}
	if turn.Tool == nil || turn.Tool.Name != "system.info" {
		t.Fatalf("unexpected turn: %+v", turn)
	}
}

func TestConfiguredOpenAIBrainUsesProviderSafeToolAlias(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		tools := body["tools"].([]any)
		function := tools[0].(map[string]any)["function"].(map[string]any)
		providerName, _ := function["name"].(string)
		if providerName == "system.info" || strings.Contains(providerName, ".") {
			t.Fatalf("provider received invalid dotted tool name: %q", providerName)
		}
		for _, r := range providerName {
			if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == '-') {
				t.Fatalf("provider alias contains invalid character %q in %q", r, providerName)
			}
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"choices": []map[string]any{{"message": map[string]any{"tool_calls": []map[string]any{{"type": "function", "function": map[string]any{"name": providerName, "arguments": `{}`}}}}}}})
	}))
	defer server.Close()

	turn, err := callOpenAICompatibleModel(context.Background(), ModelProfile{Protocol: ProtocolOpenAICompatible, BaseURL: server.URL, Model: "test", MaxOutputTokens: 100}, "", Frame{Tools: []xiaoyuToolView{{Name: "system.info", Description: "info"}}}, BrainPrompt{System: "system", User: "user"})
	if err != nil {
		t.Fatal(err)
	}
	if turn.Tool == nil || turn.Tool.Name != "system.info" {
		t.Fatalf("provider alias was not restored to Host tool name: %+v", turn)
	}
}

func TestConnectionFallsBackToMinimalGenerationWhenModelsEndpointMissing(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/models":
			http.Error(w, "not supported", http.StatusNotFound)
		case "/v1/chat/completions":
			_ = json.NewEncoder(w).Encode(map[string]any{"choices": []map[string]any{{"message": map[string]any{"content": "OK"}}}})
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()
	result, err := TestModelConnection(context.Background(), ModelProfile{Provider: "custom", Protocol: ProtocolOpenAICompatible, BaseURL: server.URL + "/v1", Model: "custom-model"}, "")
	if err != nil {
		t.Fatal(err)
	}
	if !result.OK || !strings.Contains(result.Message, "最小生成请求") {
		t.Fatalf("unexpected result: %+v", result)
	}
}

func TestConnectionDoesNotTrustModelsEndpointWithoutToolProbe(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/models":
			_ = json.NewEncoder(w).Encode(map[string]any{"data": []map[string]any{{"id": "model-a"}}})
		case "/v1/chat/completions":
			http.Error(w, `{"error":{"message":"tool schema rejected"}}`, http.StatusBadRequest)
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()
	result, err := TestModelConnection(context.Background(), ModelProfile{Provider: "custom", Protocol: ProtocolOpenAICompatible, BaseURL: server.URL + "/v1", Model: "model-a"}, "")
	if err == nil || result.OK || !strings.Contains(result.Message, "Tool 实际调用测试失败") {
		t.Fatalf("models-only success must not hide a broken XiaoYu tool call: result=%+v err=%v", result, err)
	}
}

func TestModelExtraRejectsEmbeddedSecrets(t *testing.T) {
	root := t.TempDir()
	manager := NewModelManager(filepath.Join(root, "models.json"), filepath.Join(root, "vault"))
	_, err := manager.Save(SaveModelRequest{Name: "Unsafe", Provider: "custom", Protocol: ProtocolOpenAICompatible, BaseURL: "http://127.0.0.1:8000/v1", Model: "local", Enabled: true, Extra: map[string]any{"headers": map[string]any{"Authorization": "Bearer hidden"}}})
	if err == nil || !strings.Contains(err.Error(), "专用 API Key") {
		t.Fatalf("expected secret rejection, got %v", err)
	}
}

func TestModelManagerRecordsConnectionState(t *testing.T) {
	root := t.TempDir()
	manager := NewModelManager(filepath.Join(root, "models.json"), filepath.Join(root, "vault"))
	saved, err := manager.Save(SaveModelRequest{Name: "Local", Provider: "ollama", Protocol: ProtocolOpenAICompatible, BaseURL: "http://127.0.0.1:11434/v1", Model: "qwen", Enabled: true})
	if err != nil {
		t.Fatal(err)
	}
	if err := manager.RecordTest(saved.ID, ModelConnectionResult{OK: true, Message: "连接正常"}); err != nil {
		t.Fatal(err)
	}
	catalog, err := manager.Catalog()
	if err != nil {
		t.Fatal(err)
	}
	if len(catalog.Profiles) != 1 || catalog.Profiles[0].LastTestAt.IsZero() || !catalog.Profiles[0].LastTestOK || catalog.Profiles[0].LastTestMessage != "连接正常" {
		t.Fatalf("unexpected test state: %+v", catalog.Profiles)
	}
}

func TestBuiltinProvidersUseNativeProtocols(t *testing.T) {
	byID := map[string]ModelPreset{}
	for _, preset := range BuiltinModelPresets() {
		byID[preset.ID] = preset
	}
	if byID["openai"].Protocol != ProtocolOpenAIResponses {
		t.Fatalf("OpenAI must use Responses: %+v", byID["openai"])
	}
	if byID["deepseek"].Protocol != ProtocolDeepSeek {
		t.Fatalf("DeepSeek must use native protocol: %+v", byID["deepseek"])
	}
	for _, id := range []string{"openai", "deepseek", "claude", "gemini"} {
		caps := ResolveModelCapabilities(ModelProfile{Provider: id, Protocol: byID[id].Protocol})
		if !caps.Native || !caps.Reasoning || !caps.ToolCalling || !caps.Replay {
			t.Fatalf("%s capability contract incomplete: %+v", id, caps)
		}
	}
}

func TestDeepSeekCapabilityNegotiatesThinkingToolChoice(t *testing.T) {
	base := ModelProfile{Protocol: ProtocolDeepSeek}
	caps := ResolveModelCapabilities(base)
	if caps.ThinkingToolChoiceCompatible {
		t.Fatal("DeepSeek thinking/tool_choice compatibility must be fail-closed by default")
	}
	base.Extra = map[string]any{"capabilities": map[string]any{"thinking_tool_choice": true}}
	if !ResolveModelCapabilities(base).ThinkingToolChoiceCompatible {
		t.Fatal("explicit compatible endpoint must be able to opt in")
	}
}

func TestOpenAIResponsesReplaysEncryptedReasoningAndToolResult(t *testing.T) {
	step := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		step++
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if step == 1 {
			if body["store"] != false {
				t.Fatal("Responses must use store=false")
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"output": []any{
				map[string]any{"type": "reasoning", "id": "rs_1", "encrypted_content": "opaque", "summary": []any{map[string]any{"text": "check"}}},
				map[string]any{"type": "function_call", "id": "fc_1", "call_id": "call_1", "name": "system_info", "arguments": "{}"},
			}, "usage": map[string]any{"input_tokens": 2, "output_tokens": 3, "total_tokens": 5, "output_tokens_details": map[string]any{"reasoning_tokens": 1}}})
			return
		}
		raw, _ := json.Marshal(body["input"])
		text := string(raw)
		if !strings.Contains(text, "opaque") || !strings.Contains(text, "function_call_output") || !strings.Contains(text, "call_1") {
			t.Fatalf("native replay missing: %s", text)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"output": []any{map[string]any{"type": "message", "content": []any{map[string]any{"type": "output_text", "text": "done"}}}}})
	}))
	defer server.Close()
	profile := ModelProfile{Protocol: ProtocolOpenAIResponses, BaseURL: server.URL, Model: "gpt-test", MaxOutputTokens: 128, ThinkingMode: "on", ReasoningEffort: "high"}
	frame := Frame{RunID: "r1", Tools: []xiaoyuToolView{{Name: "system_info"}}}
	turn, err := callOpenAIResponsesModelWithReplay(context.Background(), profile, "", frame, BrainPrompt{System: "s", User: "u1"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if turn.Tool == nil || turn.Replay == nil || turn.Usage.ReasoningTokens != 1 {
		t.Fatalf("unexpected first turn: %+v", turn)
	}
	frame.Observations = []Observation{{Step: 1, Tool: "system_info", Summary: "ok"}}
	if _, err := callOpenAIResponsesModelWithReplay(context.Background(), profile, "", frame, BrainPrompt{System: "s", User: "u2"}, turn.Replay); err != nil {
		t.Fatal(err)
	}
}

func TestDeepSeekReplaysReasoningContent(t *testing.T) {
	step := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		step++
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		if step == 1 {
			if _, ok := body["thinking"]; !ok {
				t.Fatal("DeepSeek thinking request must preserve native thinking")
			}
			if _, ok := body["tool_choice"]; ok {
				t.Fatalf("DeepSeek thinking request must omit incompatible tool_choice: %+v", body["tool_choice"])
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"choices": []any{map[string]any{"message": map[string]any{"content": "checking", "reasoning_content": "private-state", "tool_calls": []any{map[string]any{"id": "ds1", "function": map[string]any{"name": "system_info", "arguments": "{}"}}}}}}})
			return
		}
		raw, _ := json.Marshal(body["messages"])
		if !strings.Contains(string(raw), "private-state") || !strings.Contains(string(raw), "ds1") {
			t.Fatalf("reasoning replay missing: %s", raw)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"choices": []any{map[string]any{"message": map[string]any{"content": "done"}}}})
	}))
	defer server.Close()
	p := ModelProfile{Protocol: ProtocolDeepSeek, BaseURL: server.URL, Model: "deepseek-test", MaxOutputTokens: 128, ThinkingMode: "on", ReasoningEffort: "high"}
	f := Frame{RunID: "r2", Tools: []xiaoyuToolView{{Name: "system_info"}}}
	turn, err := callDeepSeekModelWithReplay(context.Background(), p, "", f, BrainPrompt{System: "s", User: "u1"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	f.Observations = []Observation{{Tool: "system_info", Summary: "ok"}}
	if _, err := callDeepSeekModelWithReplay(context.Background(), p, "", f, BrainPrompt{System: "s", User: "u2"}, turn.Replay); err != nil {
		t.Fatal(err)
	}
}

func TestDeepSeekNonThinkingKeepsToolChoice(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		if got, ok := body["tool_choice"]; !ok || got != "auto" {
			t.Fatalf("non-thinking DeepSeek should keep tool_choice=auto, got %#v", got)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"choices": []any{map[string]any{"message": map[string]any{"content": "done"}}}})
	}))
	defer server.Close()
	p := ModelProfile{Protocol: ProtocolDeepSeek, BaseURL: server.URL, Model: "deepseek-test", MaxOutputTokens: 128, ThinkingMode: "off"}
	f := Frame{RunID: "r3", Tools: []xiaoyuToolView{{Name: "system_info"}}}
	if _, err := callDeepSeekModelWithReplay(context.Background(), p, "", f, BrainPrompt{System: "s", User: "u"}, nil); err != nil {
		t.Fatal(err)
	}
}

func TestNativeVisionSerializersAndUnsupportedProvider(t *testing.T) {
	img := ModelInputAttachment{ID: "img", MediaType: "image/png", Data: []byte{1, 2, 3}}
	if len(openAIResponsesUserContent("look", []ModelInputAttachment{img})) != 2 {
		t.Fatal("OpenAI vision block missing")
	}
	if blocks, ok := anthropicUserContent("look", []ModelInputAttachment{img}).([]map[string]any); !ok || len(blocks) != 2 {
		t.Fatal("Anthropic vision block missing")
	}
	if len(geminiUserParts("look", []ModelInputAttachment{img})) != 2 {
		t.Fatal("Gemini vision block missing")
	}
	if err := ensureAttachmentSupport(ModelProfile{Protocol: ProtocolDeepSeek}, Frame{Attachments: []ModelInputAttachment{img}}); err == nil {
		t.Fatal("text-only provider must not silently drop image")
	}
}
