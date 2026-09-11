package host

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"
)

func TestModelConnection(ctx context.Context, profile ModelProfile, apiKey string) (ModelConnectionResult, error) {
	start := time.Now()
	caps := ResolveModelCapabilities(profile)
	models, endpoint, discoverErr := DiscoverModelNames(ctx, profile, apiKey)
	if discoverErr == nil {
		result := ModelConnectionResult{OK: true, Endpoint: endpoint, LatencyMS: time.Since(start).Milliseconds(), Models: models, Message: "连接成功", Capabilities: caps}
		if profile.Model != "" {
			found := false
			for _, m := range models {
				if m == profile.Model {
					found = true
					break
				}
			}
			probeEndpoint, probeErr := probeModel(ctx, profile, apiKey)
			result.Endpoint = probeEndpoint
			result.LatencyMS = time.Since(start).Milliseconds()
			if probeErr != nil {
				result.OK = false
				result.Message = fmt.Sprintf("模型目录可访问，但 XiaoYu Tool 实际调用测试失败：%v", probeErr)
				return result, probeErr
			}
			if found || len(models) == 0 {
				result.Message = "连接成功，模型生成与 XiaoYu Tool Calling 均可用"
			} else {
				result.Message = "模型目录中未发现当前模型代码，但实际 Tool Calling 测试成功；请确认是否使用模型别名。"
			}
		}
		return result, nil
	}
	probeEndpoint, probeErr := probeModel(ctx, profile, apiKey)
	result := ModelConnectionResult{OK: probeErr == nil, Endpoint: probeEndpoint, LatencyMS: time.Since(start).Milliseconds(), Capabilities: caps}
	if probeErr != nil {
		result.Message = fmt.Sprintf("模型目录不可用：%v；实际调用测试也失败：%v", discoverErr, probeErr)
		return result, probeErr
	}
	result.Message = "连接成功（该服务未提供模型目录，已通过最小生成请求与 Tool Schema 请求验证）"
	return result, nil
}

func probeModel(ctx context.Context, p ModelProfile, key string) (string, error) {
	if strings.TrimSpace(p.Model) == "" {
		return p.BaseURL, errors.New("未填写模型代码，无法进行实际调用测试")
	}
	base := strings.TrimRight(p.BaseURL, "/")
	name := "agmp_connection_probe"
	schema := map[string]any{"type": "object", "properties": map[string]any{}, "additionalProperties": false}
	var ep string
	var body map[string]any
	switch p.Protocol {
	case ProtocolOpenAIResponses:
		ep = base + "/responses"
		body = map[string]any{"model": p.Model, "input": []map[string]any{{"role": "user", "content": []map[string]any{{"type": "input_text", "text": "Call the provided AGMP probe tool now."}}}}, "max_output_tokens": 32, "store": false, "tools": []map[string]any{{"type": "function", "name": name, "description": "AGMP XiaoYu tool compatibility probe", "parameters": schema}}, "tool_choice": map[string]any{"type": "function", "name": name}}
		if e := modelReasoningEffort(p); e != "" && e != "off" {
			if e == "max" {
				e = "high"
			}
			body["reasoning"] = map[string]any{"effort": e, "summary": "auto"}
			body["include"] = []string{"reasoning.encrypted_content"}
		}
	case ProtocolDeepSeek:
		ep = base + "/chat/completions"
		body = map[string]any{"model": p.Model, "messages": []map[string]any{{"role": "user", "content": "Call the provided AGMP probe tool now."}}, "max_tokens": 32, "tools": []map[string]any{{"type": "function", "function": map[string]any{"name": name, "description": "AGMP XiaoYu tool compatibility probe", "parameters": schema}}}, "tool_choice": map[string]any{"type": "function", "function": map[string]any{"name": name}}}
		if en := thinkingEnabled(p); en != nil {
			kind := "disabled"
			if *en {
				kind = "enabled"
			}
			body["thinking"] = map[string]any{"type": kind}
		}
	case ProtocolAnthropic:
		ep = base + "/v1/messages"
		body = map[string]any{"model": p.Model, "max_tokens": 32, "messages": []map[string]any{{"role": "user", "content": "Call the provided AGMP probe tool now."}}, "tools": []map[string]any{{"name": name, "description": "AGMP XiaoYu tool compatibility probe", "input_schema": schema}}, "tool_choice": map[string]any{"type": "tool", "name": name}}
		if en := thinkingEnabled(p); en != nil && *en {
			body["thinking"] = map[string]any{"type": "adaptive"}
		}
	case ProtocolGemini:
		ep = base + "/models/" + escapedModelPath(p.Model) + ":generateContent"
		body = map[string]any{"contents": []map[string]any{{"role": "user", "parts": []map[string]any{{"text": "Call the provided AGMP probe tool now."}}}}, "tools": []map[string]any{{"functionDeclarations": []map[string]any{{"name": name, "description": "AGMP XiaoYu tool compatibility probe", "parameters": schema}}}}, "toolConfig": map[string]any{"functionCallingConfig": map[string]any{"mode": "ANY", "allowedFunctionNames": []string{name}}}, "generationConfig": map[string]any{"maxOutputTokens": 32}}
	default:
		ep = base + "/chat/completions"
		body = map[string]any{"model": p.Model, "messages": []map[string]any{{"role": "user", "content": "Call the provided AGMP probe tool now."}}, "max_tokens": 32, "tools": []map[string]any{{"type": "function", "function": map[string]any{"name": name, "description": "AGMP XiaoYu tool compatibility probe", "parameters": schema}}}, "tool_choice": map[string]any{"type": "function", "function": map[string]any{"name": name}}}
	}
	raw, err := postModelJSON(ctx, ep, p.Protocol, key, body)
	if err != nil {
		return ep, err
	}
	if err := verifyModelToolProbe(p.Protocol, raw, name); err != nil {
		return ep, err
	}
	return ep, nil
}

func verifyModelToolProbe(protocol ModelProtocol, payload []byte, expected string) error {
	var root map[string]any
	if err := json.Unmarshal(payload, &root); err != nil {
		return err
	}
	found := false
	switch protocol {
	case ProtocolOpenAIResponses:
		for _, r := range anySlice(root["output"]) {
			m, _ := r.(map[string]any)
			if m["type"] == "function_call" && m["name"] == expected {
				found = true
				break
			}
		}
	case ProtocolAnthropic:
		for _, r := range anySlice(root["content"]) {
			m, _ := r.(map[string]any)
			if m["type"] == "tool_use" && m["name"] == expected {
				found = true
				break
			}
		}
	case ProtocolGemini:
		for _, cr := range anySlice(root["candidates"]) {
			c, _ := cr.(map[string]any)
			co, _ := c["content"].(map[string]any)
			for _, pr := range anySlice(co["parts"]) {
				p, _ := pr.(map[string]any)
				fc, _ := p["functionCall"].(map[string]any)
				if fc["name"] == expected {
					found = true
					break
				}
			}
		}
	default:
		for _, cr := range anySlice(root["choices"]) {
			c, _ := cr.(map[string]any)
			m, _ := c["message"].(map[string]any)
			for _, tr := range anySlice(m["tool_calls"]) {
				tc, _ := tr.(map[string]any)
				fn, _ := tc["function"].(map[string]any)
				if fn["name"] == expected {
					found = true
					break
				}
			}
		}
	}
	if !found {
		if protocol == ProtocolOpenAICompatible {
			return nil
		}
		return fmt.Errorf("Provider 返回成功，但没有执行强制 XiaoYu Tool Probe %q", expected)
	}
	return nil
}
func anySlice(v any) []any { x, _ := v.([]any); return x }

func DiscoverModelNames(ctx context.Context, p ModelProfile, key string) ([]string, string, error) {
	client := &http.Client{Timeout: 20 * time.Second}
	ep := strings.TrimRight(p.BaseURL, "/") + "/models"
	if p.Protocol == ProtocolAnthropic {
		ep = strings.TrimRight(p.BaseURL, "/") + "/v1/models"
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, ep, nil)
	if err != nil {
		return nil, ep, err
	}
	applyModelAuth(req, p.Protocol, key)
	resp, err := client.Do(req)
	if err != nil {
		return nil, ep, fmt.Errorf("连接模型服务失败: %w", err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if err != nil {
		return nil, ep, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, ep, fmt.Errorf("模型服务返回 HTTP %d: %s", resp.StatusCode, compactError(raw))
	}
	models, err := parseModelList(p.Protocol, raw)
	if err != nil {
		return nil, ep, err
	}
	sort.Strings(models)
	return models, ep, nil
}
func applyModelAuth(req *http.Request, p ModelProtocol, key string) {
	key = strings.TrimSpace(key)
	if key == "" {
		return
	}
	switch p {
	case ProtocolAnthropic:
		req.Header.Set("x-api-key", key)
		req.Header.Set("anthropic-version", "2023-06-01")
	case ProtocolGemini:
		req.Header.Set("x-goog-api-key", key)
	default:
		req.Header.Set("Authorization", "Bearer "+key)
	}
}
func parseModelList(p ModelProtocol, raw []byte) ([]string, error) {
	if p == ProtocolGemini {
		var b struct {
			Models []struct {
				Name string `json:"name"`
			} `json:"models"`
		}
		if err := json.Unmarshal(raw, &b); err != nil {
			return nil, err
		}
		out := []string{}
		for _, x := range b.Models {
			n := strings.TrimPrefix(strings.TrimSpace(x.Name), "models/")
			if n != "" {
				out = append(out, n)
			}
		}
		return uniqueStrings(out), nil
	}
	var b struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(raw, &b); err != nil {
		return nil, err
	}
	out := []string{}
	for _, x := range b.Data {
		if n := strings.TrimSpace(x.ID); n != "" {
			out = append(out, n)
		}
	}
	return uniqueStrings(out), nil
}
func compactError(raw []byte) string {
	var p map[string]any
	if json.Unmarshal(raw, &p) == nil {
		if e, ok := p["error"].(map[string]any); ok {
			if m, ok := e["message"].(string); ok && strings.TrimSpace(m) != "" {
				return strings.TrimSpace(m)
			}
		}
	}
	t := strings.TrimSpace(string(raw))
	if len(t) > 280 {
		t = t[:280] + "…"
	}
	return t
}
func uniqueStrings(v []string) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, x := range v {
		if !seen[x] {
			seen[x] = true
			out = append(out, x)
		}
	}
	return out
}

func postModelJSON(ctx context.Context, ep string, p ModelProtocol, key string, body map[string]any) ([]byte, error) {
	raw, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	client := &http.Client{Timeout: 5 * time.Minute}
	delays := []time.Duration{0, 250 * time.Millisecond, 750 * time.Millisecond, 1500 * time.Millisecond}
	var last error
	for i, d := range delays {
		if i > 0 {
			if err := waitModelRetry(ctx, d); err != nil {
				return nil, err
			}
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, ep, bytes.NewReader(raw))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json")
		applyModelAuth(req, p, key)
		resp, err := client.Do(req)
		if err != nil {
			if ctx.Err() != nil {
				return nil, ctx.Err()
			}
			last = err
			continue
		}
		payload, rerr := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
		_ = resp.Body.Close()
		if rerr != nil {
			last = rerr
			continue
		}
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return payload, nil
		}
		last = fmt.Errorf("模型服务返回 HTTP %d: %s", resp.StatusCode, compactError(payload))
		if !retryableModelStatus(resp.StatusCode) {
			return nil, last
		}
	}
	if last == nil {
		last = errors.New("模型服务请求失败")
	}
	return nil, last
}
func retryableModelStatus(s int) bool {
	switch s {
	case 408, 429, 500, 502, 503, 504:
		return true
	}
	return false
}
func waitModelRetry(ctx context.Context, d time.Duration) error {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}
func escapedModelPath(model string) string { return url.PathEscape(strings.TrimSpace(model)) }
