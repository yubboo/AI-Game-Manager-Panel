package host

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
)

type BrainPrompt struct {
	System string `json:"system"`
	User   string `json:"user"`
}

type ModelToolCall struct {
	Name      string         `json:"name"`
	Arguments map[string]any `json:"arguments,omitempty"`
}
type ModelUsage struct {
	InputTokens     int64 `json:"inputTokens,omitempty"`
	OutputTokens    int64 `json:"outputTokens,omitempty"`
	ReasoningTokens int64 `json:"reasoningTokens,omitempty"`
	TotalTokens     int64 `json:"totalTokens,omitempty"`
}
type ModelReplayState struct {
	Protocol      ModelProtocol `json:"protocol"`
	Tool          string        `json:"tool"`
	ToolAlias     string        `json:"toolAlias,omitempty"`
	CallID        string        `json:"callId,omitempty"`
	AttachmentIDs []string      `json:"-"`
	Payload       any           `json:"-"`
}
type ModelTurn struct {
	Tool      *ModelToolCall    `json:"tool,omitempty"`
	Text      string            `json:"text,omitempty"`
	Reasoning string            `json:"reasoning,omitempty"`
	Usage     ModelUsage        `json:"usage,omitempty"`
	Replay    *ModelReplayState `json:"-"`
}

type BrainPolicy interface {
	Ready(context.Context) (bool, string)
	Prepare(context.Context, Frame) (BrainPrompt, error)
	Resolve(context.Context, ModelTurn) (Decision, error)
}

type ConfiguredModelBrain struct {
	Models   *ModelManager
	Policy   BrainPolicy
	replayMu sync.Mutex
	replay   map[string]ModelReplayState
}

func (b *ConfiguredModelBrain) Info(ctx context.Context) BrainInfo {
	if b == nil || b.Models == nil {
		return BrainInfo{ID: "xiaoyu.model", Name: "小鱼大脑", Source: "model-center", Ready: false, Message: "模型管理器未初始化"}
	}
	profile, _, err := b.Models.Default()
	if err != nil {
		return BrainInfo{ID: "xiaoyu.model", Name: "小鱼大脑", Source: "model-center", Ready: false, Message: "请先在系统设置 → 模型管理配置默认模型"}
	}
	if b.Policy == nil {
		return BrainInfo{ID: profile.ID, Name: profile.Name, Model: profile.Model, Source: profile.Provider, Ready: false, Message: "XiaoYu Rust Agent Runtime 未连接"}
	}
	if ready, msg := b.Policy.Ready(ctx); !ready {
		if strings.TrimSpace(msg) == "" {
			msg = "XiaoYu Rust Agent Runtime 未就绪"
		}
		return BrainInfo{ID: profile.ID, Name: profile.Name, Model: profile.Model, Source: profile.Provider, Ready: false, Message: msg}
	}
	caps := ResolveModelCapabilities(profile)
	return BrainInfo{ID: profile.ID, Name: profile.Name, Model: profile.Model, Source: profile.Provider, Ready: true, Message: fmt.Sprintf("%s · 模型大脑与 XiaoYu Rust Core 已连接", caps.Adapter), Capabilities: &caps}
}

func (b *ConfiguredModelBrain) Next(ctx context.Context, frame Frame) (Decision, error) {
	if b == nil || b.Models == nil || b.Policy == nil {
		return Decision{}, ErrBrainUnavailable
	}
	profile, key, err := b.Models.Default()
	if err != nil {
		return Decision{}, err
	}
	prompt, err := b.Policy.Prepare(ctx, frame)
	if err != nil {
		return Decision{}, err
	}
	replay := b.replayFor(frame.RunID)
	var turn ModelTurn
	switch profile.Protocol {
	case ProtocolOpenAIResponses:
		turn, err = callOpenAIResponsesModelWithReplay(ctx, profile, key, frame, prompt, replay)
	case ProtocolDeepSeek:
		turn, err = callDeepSeekModelWithReplay(ctx, profile, key, frame, prompt, replay)
	case ProtocolAnthropic:
		turn, err = callAnthropicModelWithReplay(ctx, profile, key, frame, prompt, replay)
	case ProtocolGemini:
		turn, err = callGeminiModelWithReplay(ctx, profile, key, frame, prompt, replay)
	default:
		turn, err = callOpenAICompatibleModelWithReplay(ctx, profile, key, frame, prompt, replay)
	}
	if err != nil {
		return Decision{}, err
	}
	if turn.Replay != nil && turn.Tool != nil {
		b.storeReplay(frame.RunID, *turn.Replay)
	} else {
		b.clearReplay(frame.RunID)
	}
	return b.Policy.Resolve(ctx, turn)
}
func (b *ConfiguredModelBrain) replayFor(id string) *ModelReplayState {
	if b == nil || strings.TrimSpace(id) == "" {
		return nil
	}
	b.replayMu.Lock()
	defer b.replayMu.Unlock()
	st, ok := b.replay[id]
	if !ok {
		return nil
	}
	cp := st
	return &cp
}
func (b *ConfiguredModelBrain) storeReplay(id string, st ModelReplayState) {
	if strings.TrimSpace(id) == "" {
		return
	}
	b.replayMu.Lock()
	defer b.replayMu.Unlock()
	if b.replay == nil {
		b.replay = map[string]ModelReplayState{}
	}
	b.replay[id] = st
}
func (b *ConfiguredModelBrain) clearReplay(id string) {
	if b == nil {
		return
	}
	b.replayMu.Lock()
	delete(b.replay, id)
	b.replayMu.Unlock()
}

type modelToolNames struct {
	outbound map[string]string
	inbound  map[string]string
}

func buildModelToolNames(frame Frame) modelToolNames {
	n := modelToolNames{map[string]string{}, map[string]string{}}
	for _, t := range frame.Tools {
		real := strings.TrimSpace(t.Name)
		if real == "" {
			continue
		}
		alias := providerSafeToolName(real)
		if ex, ok := n.inbound[alias]; ok && ex != real {
			sum := sha256.Sum256([]byte(real + "#collision"))
			base := alias
			if len(base) > 54 {
				base = base[:54]
			}
			alias = fmt.Sprintf("%s_%x", base, sum[:4])
		}
		n.outbound[real] = alias
		n.inbound[alias] = real
	}
	return n
}
func providerSafeToolName(name string) string {
	name = strings.TrimSpace(name)
	var b strings.Builder
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == '-' {
			b.WriteRune(r)
		} else {
			b.WriteByte('_')
		}
	}
	a := strings.Trim(b.String(), "_-")
	if a == "" {
		a = "tool"
	}
	if a == name && len(a) <= 64 {
		return a
	}
	if len(a) > 54 {
		a = a[:54]
	}
	sum := sha256.Sum256([]byte(name))
	return fmt.Sprintf("%s_%x", a, sum[:4])
}
func modelToolName(n modelToolNames, real string) string {
	if v := n.outbound[real]; v != "" {
		return v
	}
	return providerSafeToolName(real)
}
func hostToolName(n modelToolNames, p string) string {
	if v := n.inbound[p]; v != "" {
		return v
	}
	return p
}
func modelToolDescription(t xiaoyuToolView) string {
	return fmt.Sprintf("AGMP Tool: %s。%s 风险=%s；类别=%s；层级=%s；来源=%s。", t.Name, t.Description, t.Risk, t.Category, t.Tier, t.Source)
}

func openAITools(frame Frame, n modelToolNames) []map[string]any {
	out := make([]map[string]any, 0, len(frame.Tools))
	for _, t := range frame.Tools {
		p := t.Parameters
		if len(p) == 0 {
			p = map[string]any{"type": "object", "properties": map[string]any{}, "additionalProperties": false}
		}
		out = append(out, map[string]any{"type": "function", "function": map[string]any{"name": modelToolName(n, t.Name), "description": modelToolDescription(t), "parameters": p}})
	}
	return out
}
func openAIResponsesTools(frame Frame, n modelToolNames) []map[string]any {
	out := make([]map[string]any, 0, len(frame.Tools))
	for _, t := range frame.Tools {
		p := t.Parameters
		if len(p) == 0 {
			p = map[string]any{"type": "object", "properties": map[string]any{}, "additionalProperties": false}
		}
		out = append(out, map[string]any{"type": "function", "name": modelToolName(n, t.Name), "description": modelToolDescription(t), "parameters": p, "strict": false})
	}
	return out
}

func modelReasoningEffort(p ModelProfile) string {
	e := strings.ToLower(strings.TrimSpace(p.ReasoningEffort))
	switch e {
	case "off", "minimal", "low", "medium", "high", "xhigh", "max":
		return e
	}
	if raw, ok := p.Extra["reasoning_effort"].(string); ok {
		e = strings.ToLower(strings.TrimSpace(raw))
		switch e {
		case "minimal", "low", "medium", "high", "xhigh", "max":
			return e
		}
	}
	switch strings.ToLower(strings.TrimSpace(p.ThinkingMode)) {
	case "on", "high", "max":
		return "high"
	case "low":
		return "low"
	case "medium":
		return "medium"
	case "off":
		return "off"
	}
	return ""
}
func thinkingEnabled(p ModelProfile) *bool {
	switch strings.ToLower(strings.TrimSpace(p.ThinkingMode)) {
	case "on", "low", "medium", "high", "max":
		v := true
		return &v
	case "off":
		v := false
		return &v
	}
	return nil
}
func replayObservation(frame Frame, r *ModelReplayState) *Observation {
	if r == nil {
		return nil
	}
	for i := len(frame.Observations) - 1; i >= 0; i-- {
		o := frame.Observations[i]
		if o.Tool == r.Tool && !o.Pending {
			cp := o
			return &cp
		}
	}
	return nil
}
func replayObservationText(frame Frame, r *ModelReplayState) string {
	o := replayObservation(frame, r)
	if o == nil {
		return "AGMP Host has not produced a tool result yet."
	}
	m := map[string]any{"summary": o.Summary}
	if o.Data != nil {
		m["data"] = o.Data
	}
	if o.Error != "" {
		m["error"] = o.Error
	}
	if o.Denied {
		m["denied"] = true
	}
	raw, _ := json.Marshal(m)
	if len(raw) > 128*1024 {
		return string(raw[:128*1024]) + "…"
	}
	return string(raw)
}
func ensureAttachmentSupport(p ModelProfile, f Frame) error {
	if len(f.Attachments) == 0 {
		return nil
	}
	c := ResolveModelCapabilities(p)
	if !c.Vision {
		return fmt.Errorf("当前模型适配器 %s 未声明 Vision 能力，不能静默丢弃 %d 张图片", c.Adapter, len(f.Attachments))
	}
	return nil
}
func attachmentDataURL(a ModelInputAttachment) string {
	return "data:" + a.MediaType + ";base64," + base64.StdEncoding.EncodeToString(a.Data)
}
func modelAttachmentsForTurn(items []ModelInputAttachment, r *ModelReplayState) []ModelInputAttachment {
	seen := map[string]bool{}
	if r != nil {
		for _, id := range r.AttachmentIDs {
			seen[id] = true
		}
	}
	out := []ModelInputAttachment{}
	for _, x := range items {
		if !seen[x.ID] {
			out = append(out, x)
		}
	}
	return out
}
func replayAttachmentIDs(r *ModelReplayState, items []ModelInputAttachment) []string {
	seen := map[string]bool{}
	out := []string{}
	if r != nil {
		for _, id := range r.AttachmentIDs {
			if id != "" && !seen[id] {
				seen[id] = true
				out = append(out, id)
			}
		}
	}
	for _, x := range items {
		if x.ID != "" && !seen[x.ID] {
			seen[x.ID] = true
			out = append(out, x.ID)
		}
	}
	return out
}
func openAIUserContent(text string, imgs []ModelInputAttachment) any {
	if len(imgs) == 0 {
		return text
	}
	p := []map[string]any{{"type": "text", "text": text}}
	for _, x := range imgs {
		p = append(p, map[string]any{"type": "image_url", "image_url": map[string]any{"url": attachmentDataURL(x)}})
	}
	return p
}
func openAIResponsesUserContent(text string, imgs []ModelInputAttachment) []map[string]any {
	p := []map[string]any{{"type": "input_text", "text": text}}
	for _, x := range imgs {
		p = append(p, map[string]any{"type": "input_image", "image_url": attachmentDataURL(x)})
	}
	return p
}
func anthropicUserContent(text string, imgs []ModelInputAttachment) any {
	if len(imgs) == 0 {
		return text
	}
	p := []map[string]any{{"type": "text", "text": text}}
	for _, x := range imgs {
		p = append(p, map[string]any{"type": "image", "source": map[string]any{"type": "base64", "media_type": x.MediaType, "data": base64.StdEncoding.EncodeToString(x.Data)}})
	}
	return p
}
func geminiUserParts(text string, imgs []ModelInputAttachment) []any {
	p := []any{map[string]any{"text": text}}
	for _, x := range imgs {
		p = append(p, map[string]any{"inlineData": map[string]any{"mimeType": x.MediaType, "data": base64.StdEncoding.EncodeToString(x.Data)}})
	}
	return p
}

func callOpenAICompatibleModel(ctx context.Context, p ModelProfile, key string, f Frame, pr BrainPrompt) (ModelTurn, error) {
	return callOpenAICompatibleModelWithReplay(ctx, p, key, f, pr, nil)
}
func callOpenAICompatibleModelWithReplay(ctx context.Context, p ModelProfile, key string, f Frame, pr BrainPrompt, r *ModelReplayState) (ModelTurn, error) {
	if err := ensureAttachmentSupport(p, f); err != nil {
		return ModelTurn{}, err
	}
	ep := strings.TrimRight(p.BaseURL, "/") + "/chat/completions"
	names := buildModelToolNames(f)
	hist := []map[string]any{}
	imgs := modelAttachmentsForTurn(f.Attachments, r)
	if r != nil && r.Protocol == ProtocolOpenAICompatible {
		if prev, ok := r.Payload.([]map[string]any); ok {
			for _, x := range prev {
				hist = append(hist, cloneAnyMap(x))
			}
		}
		if replayObservation(f, r) != nil {
			hist = append(hist, map[string]any{"role": "tool", "tool_call_id": r.CallID, "content": replayObservationText(f, r)})
		}
	}
	hist = append(hist, map[string]any{"role": "user", "content": openAIUserContent(pr.User, imgs)})
	msgs := append([]map[string]any{{"role": "system", "content": pr.System}}, hist...)
	body := map[string]any{"model": p.Model, "messages": msgs, "tools": openAITools(f, names), "tool_choice": "auto", "max_tokens": p.MaxOutputTokens, "stream": false}
	if ResolveModelCapabilities(p).Reasoning {
		if e := modelReasoningEffort(p); e != "" && e != "off" {
			body["reasoning_effort"] = e
		}
	}
	mergeExtra(body, p.Extra, "model", "messages", "tools", "tool_choice", "max_tokens", "stream", "reasoning_effort", "capabilities")
	raw, err := postModelJSON(ctx, ep, p.Protocol, key, body)
	if err != nil {
		return ModelTurn{}, err
	}
	var resp struct {
		Choices []struct {
			Message struct {
				Role             string `json:"role"`
				Content          any    `json:"content"`
				ReasoningContent string `json:"reasoning_content"`
				ToolCalls        []struct {
					ID       string `json:"id"`
					Function struct {
						Name      string `json:"name"`
						Arguments string `json:"arguments"`
					} `json:"function"`
				} `json:"tool_calls"`
			} `json:"message"`
		} `json:"choices"`
		Usage struct {
			PromptTokens     int64 `json:"prompt_tokens"`
			CompletionTokens int64 `json:"completion_tokens"`
			TotalTokens      int64 `json:"total_tokens"`
		} `json:"usage"`
	}
	if json.Unmarshal(raw, &resp) != nil || len(resp.Choices) == 0 {
		return ModelTurn{}, errors.New("模型没有返回可用响应")
	}
	m := resp.Choices[0].Message
	t := ModelTurn{Text: contentString(m.Content), Reasoning: m.ReasoningContent, Usage: ModelUsage{InputTokens: resp.Usage.PromptTokens, OutputTokens: resp.Usage.CompletionTokens, TotalTokens: resp.Usage.TotalTokens}}
	if len(m.ToolCalls) == 0 {
		return t, nil
	}
	c := m.ToolCalls[0]
	args, err := decodeToolArguments(c.Function.Arguments)
	if err != nil {
		return ModelTurn{}, err
	}
	real := hostToolName(names, c.Function.Name)
	id := c.ID
	if id == "" {
		id = "call_" + providerSafeToolName(real)
	}
	assistant := map[string]any{"role": "assistant", "content": t.Text, "tool_calls": []map[string]any{{"id": id, "type": "function", "function": map[string]any{"name": c.Function.Name, "arguments": c.Function.Arguments}}}}
	if m.ReasoningContent != "" {
		assistant["reasoning_content"] = m.ReasoningContent
	}
	hist = append(hist, assistant)
	t.Tool = &ModelToolCall{Name: real, Arguments: args}
	t.Replay = &ModelReplayState{Protocol: ProtocolOpenAICompatible, Tool: real, ToolAlias: c.Function.Name, CallID: id, AttachmentIDs: replayAttachmentIDs(r, imgs), Payload: hist}
	return t, nil
}

func callOpenAIResponsesModelWithReplay(ctx context.Context, p ModelProfile, key string, f Frame, pr BrainPrompt, r *ModelReplayState) (ModelTurn, error) {
	if err := ensureAttachmentSupport(p, f); err != nil {
		return ModelTurn{}, err
	}
	ep := strings.TrimRight(p.BaseURL, "/") + "/responses"
	names := buildModelToolNames(f)
	input := []any{}
	imgs := modelAttachmentsForTurn(f.Attachments, r)
	if r != nil && r.Protocol == ProtocolOpenAIResponses {
		if prev, ok := r.Payload.([]any); ok {
			input = append(input, cloneAnySlice(prev)...)
		}
		if replayObservation(f, r) != nil {
			input = append(input, map[string]any{"type": "function_call_output", "call_id": r.CallID, "output": replayObservationText(f, r)})
		}
	}
	input = append(input, map[string]any{"role": "user", "content": openAIResponsesUserContent(pr.User, imgs)})
	body := map[string]any{"model": p.Model, "instructions": pr.System, "input": input, "tools": openAIResponsesTools(f, names), "tool_choice": "auto", "max_output_tokens": p.MaxOutputTokens, "store": false, "include": []string{"reasoning.encrypted_content"}, "stream": false}
	if e := modelReasoningEffort(p); e != "" && e != "off" {
		if e == "max" {
			e = "high"
		}
		body["reasoning"] = map[string]any{"effort": e, "summary": "auto"}
	}
	mergeExtra(body, p.Extra, "model", "instructions", "input", "tools", "tool_choice", "max_output_tokens", "store", "include", "stream", "reasoning", "capabilities")
	raw, err := postModelJSON(ctx, ep, p.Protocol, key, body)
	if err != nil {
		return ModelTurn{}, err
	}
	var resp struct {
		Output []map[string]any `json:"output"`
		Usage  map[string]any   `json:"usage"`
	}
	if err := json.Unmarshal(raw, &resp); err != nil {
		return ModelTurn{}, err
	}
	t := ModelTurn{Usage: parseOpenAIResponsesUsage(resp.Usage)}
	selected := []any{}
	var call map[string]any
	for _, it := range resp.Output {
		typ, _ := it["type"].(string)
		switch typ {
		case "reasoning":
			selected = append(selected, cloneAnyMap(it))
			t.Reasoning += responseReasoningSummary(it)
		case "message":
			selected = append(selected, cloneAnyMap(it))
			t.Text += responseMessageText(it)
		case "function_call":
			if call == nil {
				call = it
				selected = append(selected, cloneAnyMap(it))
			}
		}
	}
	if call == nil {
		return t, nil
	}
	pn, _ := call["name"].(string)
	argstr, _ := call["arguments"].(string)
	args, err := decodeToolArguments(argstr)
	if err != nil {
		return ModelTurn{}, err
	}
	id, _ := call["call_id"].(string)
	if id == "" {
		id, _ = call["id"].(string)
	}
	real := hostToolName(names, pn)
	input = append(input, selected...)
	t.Tool = &ModelToolCall{Name: real, Arguments: args}
	t.Replay = &ModelReplayState{Protocol: ProtocolOpenAIResponses, Tool: real, ToolAlias: pn, CallID: id, AttachmentIDs: replayAttachmentIDs(r, imgs), Payload: input}
	return t, nil
}

func callDeepSeekModelWithReplay(ctx context.Context, p ModelProfile, key string, f Frame, pr BrainPrompt, r *ModelReplayState) (ModelTurn, error) {
	if err := ensureAttachmentSupport(p, f); err != nil {
		return ModelTurn{}, err
	}
	ep := strings.TrimRight(p.BaseURL, "/") + "/chat/completions"
	names := buildModelToolNames(f)
	hist := []map[string]any{}
	imgs := modelAttachmentsForTurn(f.Attachments, r)
	if r != nil && r.Protocol == ProtocolDeepSeek {
		if prev, ok := r.Payload.([]map[string]any); ok {
			for _, x := range prev {
				hist = append(hist, cloneAnyMap(x))
			}
		}
		if replayObservation(f, r) != nil {
			hist = append(hist, map[string]any{"role": "tool", "tool_call_id": r.CallID, "content": replayObservationText(f, r)})
		}
	}
	hist = append(hist, map[string]any{"role": "user", "content": openAIUserContent(pr.User, imgs)})
	msgs := append([]map[string]any{{"role": "system", "content": pr.System}}, hist...)
	body := map[string]any{"model": p.Model, "messages": msgs, "tools": openAITools(f, names), "max_tokens": p.MaxOutputTokens, "stream": false}
	thinkingActive := false
	if en := thinkingEnabled(p); en != nil {
		kind := "disabled"
		if *en {
			kind = "enabled"
			thinkingActive = true
		}
		body["thinking"] = map[string]any{"type": kind}
	}
	// DeepSeek thinking + tool_choice is not universally compatible. Preserve
	// native thinking instead of forcing a lowest-common-denominator request.
	// Profiles may explicitly opt in when a compatible gateway/model supports it.
	if !thinkingActive || ResolveModelCapabilities(p).ThinkingToolChoiceCompatible {
		body["tool_choice"] = "auto"
	}
	if e := modelReasoningEffort(p); e != "" && e != "off" {
		if e == "medium" {
			e = "high"
		}
		if e == "xhigh" {
			e = "max"
		}
		body["reasoning_effort"] = e
	}
	mergeExtra(body, p.Extra, "model", "messages", "tools", "tool_choice", "max_tokens", "stream", "thinking", "reasoning_effort", "capabilities")
	raw, err := postModelJSON(ctx, ep, p.Protocol, key, body)
	if err != nil {
		return ModelTurn{}, err
	}
	var resp struct {
		Choices []struct {
			Message struct {
				Content          string `json:"content"`
				ReasoningContent string `json:"reasoning_content"`
				ToolCalls        []struct {
					ID       string `json:"id"`
					Function struct {
						Name      string `json:"name"`
						Arguments string `json:"arguments"`
					} `json:"function"`
				} `json:"tool_calls"`
			} `json:"message"`
		} `json:"choices"`
		Usage struct {
			PromptTokens     int64 `json:"prompt_tokens"`
			CompletionTokens int64 `json:"completion_tokens"`
			TotalTokens      int64 `json:"total_tokens"`
			Details          struct {
				ReasoningTokens int64 `json:"reasoning_tokens"`
			} `json:"completion_tokens_details"`
		} `json:"usage"`
	}
	if err := json.Unmarshal(raw, &resp); err != nil || len(resp.Choices) == 0 {
		return ModelTurn{}, errors.New("DeepSeek 没有返回可用响应")
	}
	m := resp.Choices[0].Message
	t := ModelTurn{Text: m.Content, Reasoning: m.ReasoningContent, Usage: ModelUsage{InputTokens: resp.Usage.PromptTokens, OutputTokens: resp.Usage.CompletionTokens, ReasoningTokens: resp.Usage.Details.ReasoningTokens, TotalTokens: resp.Usage.TotalTokens}}
	if len(m.ToolCalls) == 0 {
		return t, nil
	}
	c := m.ToolCalls[0]
	args, err := decodeToolArguments(c.Function.Arguments)
	if err != nil {
		return ModelTurn{}, err
	}
	real := hostToolName(names, c.Function.Name)
	id := c.ID
	if id == "" {
		id = "call_" + providerSafeToolName(real)
	}
	assistant := map[string]any{"role": "assistant", "content": m.Content, "reasoning_content": m.ReasoningContent, "tool_calls": []map[string]any{{"id": id, "type": "function", "function": map[string]any{"name": c.Function.Name, "arguments": c.Function.Arguments}}}}
	hist = append(hist, assistant)
	t.Tool = &ModelToolCall{Name: real, Arguments: args}
	t.Replay = &ModelReplayState{Protocol: ProtocolDeepSeek, Tool: real, ToolAlias: c.Function.Name, CallID: id, AttachmentIDs: replayAttachmentIDs(r, imgs), Payload: hist}
	return t, nil
}

func callAnthropicModel(ctx context.Context, p ModelProfile, key string, f Frame, pr BrainPrompt) (ModelTurn, error) {
	return callAnthropicModelWithReplay(ctx, p, key, f, pr, nil)
}
func callAnthropicModelWithReplay(ctx context.Context, p ModelProfile, key string, f Frame, pr BrainPrompt, r *ModelReplayState) (ModelTurn, error) {
	if err := ensureAttachmentSupport(p, f); err != nil {
		return ModelTurn{}, err
	}
	ep := strings.TrimRight(p.BaseURL, "/") + "/v1/messages"
	names := buildModelToolNames(f)
	tools := []map[string]any{}
	for _, x := range f.Tools {
		sc := x.Parameters
		if len(sc) == 0 {
			sc = map[string]any{"type": "object", "properties": map[string]any{}}
		}
		tools = append(tools, map[string]any{"name": modelToolName(names, x.Name), "description": modelToolDescription(x), "input_schema": sc})
	}
	msgs := []map[string]any{}
	imgs := modelAttachmentsForTurn(f.Attachments, r)
	if r != nil && r.Protocol == ProtocolAnthropic {
		if prev, ok := r.Payload.([]map[string]any); ok {
			for _, x := range prev {
				msgs = append(msgs, cloneAnyMap(x))
			}
		}
		if replayObservation(f, r) != nil {
			uc := []any{map[string]any{"type": "tool_result", "tool_use_id": r.CallID, "content": replayObservationText(f, r)}, map[string]any{"type": "text", "text": pr.User}}
			for _, x := range imgs {
				uc = append(uc, map[string]any{"type": "image", "source": map[string]any{"type": "base64", "media_type": x.MediaType, "data": base64.StdEncoding.EncodeToString(x.Data)}})
			}
			msgs = append(msgs, map[string]any{"role": "user", "content": uc})
		}
	}
	if len(msgs) == 0 {
		msgs = append(msgs, map[string]any{"role": "user", "content": anthropicUserContent(pr.User, imgs)})
	}
	body := map[string]any{"model": p.Model, "max_tokens": p.MaxOutputTokens, "system": pr.System, "messages": msgs, "tools": tools}
	if en := thinkingEnabled(p); en != nil && *en {
		body["thinking"] = map[string]any{"type": "adaptive"}
		if e := modelReasoningEffort(p); e != "" && e != "off" {
			if e == "max" || e == "xhigh" {
				e = "high"
			}
			body["output_config"] = map[string]any{"effort": e}
		}
	}
	mergeExtra(body, p.Extra, "model", "messages", "tools", "system", "max_tokens", "stream", "thinking", "output_config", "capabilities")
	raw, err := postModelJSON(ctx, ep, p.Protocol, key, body)
	if err != nil {
		return ModelTurn{}, err
	}
	var resp struct {
		Content []map[string]any `json:"content"`
		Usage   struct {
			InputTokens  int64 `json:"input_tokens"`
			OutputTokens int64 `json:"output_tokens"`
		} `json:"usage"`
	}
	if err := json.Unmarshal(raw, &resp); err != nil {
		return ModelTurn{}, err
	}
	t := ModelTurn{Usage: ModelUsage{InputTokens: resp.Usage.InputTokens, OutputTokens: resp.Usage.OutputTokens, TotalTokens: resp.Usage.InputTokens + resp.Usage.OutputTokens}}
	selected := []any{}
	var tb map[string]any
	for _, part := range resp.Content {
		typ, _ := part["type"].(string)
		switch typ {
		case "thinking", "redacted_thinking":
			selected = append(selected, cloneAnyMap(part))
			if v, _ := part["thinking"].(string); v != "" {
				t.Reasoning += v
			}
		case "text":
			selected = append(selected, cloneAnyMap(part))
			if v, _ := part["text"].(string); v != "" {
				t.Text += v
			}
		case "tool_use":
			if tb == nil {
				tb = part
				selected = append(selected, cloneAnyMap(part))
			}
		}
	}
	if tb == nil {
		return t, nil
	}
	pn, _ := tb["name"].(string)
	id, _ := tb["id"].(string)
	args, _ := tb["input"].(map[string]any)
	real := hostToolName(names, pn)
	msgs = append(msgs, map[string]any{"role": "assistant", "content": selected})
	t.Tool = &ModelToolCall{Name: real, Arguments: cloneAnyMap(args)}
	t.Replay = &ModelReplayState{Protocol: ProtocolAnthropic, Tool: real, ToolAlias: pn, CallID: id, AttachmentIDs: replayAttachmentIDs(r, imgs), Payload: msgs}
	return t, nil
}

func callGeminiModel(ctx context.Context, p ModelProfile, key string, f Frame, pr BrainPrompt) (ModelTurn, error) {
	return callGeminiModelWithReplay(ctx, p, key, f, pr, nil)
}

// Gemini model content is replayed as opaque parts so provider fields such as
// thoughtSignature remain attached to the exact functionCall/thought block.
func callGeminiModelWithReplay(ctx context.Context, p ModelProfile, key string, f Frame, pr BrainPrompt, r *ModelReplayState) (ModelTurn, error) {
	if err := ensureAttachmentSupport(p, f); err != nil {
		return ModelTurn{}, err
	}
	ep := strings.TrimRight(p.BaseURL, "/") + "/models/" + escapedModelPath(p.Model) + ":generateContent"
	names := buildModelToolNames(f)
	decl := []map[string]any{}
	for _, x := range f.Tools {
		sc := x.Parameters
		if len(sc) == 0 {
			sc = map[string]any{"type": "object", "properties": map[string]any{}}
		}
		decl = append(decl, map[string]any{"name": modelToolName(names, x.Name), "description": modelToolDescription(x), "parameters": sc})
	}
	contents := []map[string]any{}
	imgs := modelAttachmentsForTurn(f.Attachments, r)
	if r != nil && r.Protocol == ProtocolGemini {
		if prev, ok := r.Payload.([]map[string]any); ok {
			for _, x := range prev {
				contents = append(contents, cloneAnyMap(x))
			}
		}
		if replayObservation(f, r) != nil {
			parts := []any{map[string]any{"functionResponse": map[string]any{"name": r.ToolAlias, "response": map[string]any{"result": replayObservationText(f, r)}}}, map[string]any{"text": pr.User}}
			for _, x := range imgs {
				parts = append(parts, map[string]any{"inlineData": map[string]any{"mimeType": x.MediaType, "data": base64.StdEncoding.EncodeToString(x.Data)}})
			}
			contents = append(contents, map[string]any{"role": "user", "parts": parts})
		}
	}
	if len(contents) == 0 {
		contents = append(contents, map[string]any{"role": "user", "parts": geminiUserParts(pr.User, imgs)})
	}
	gc := map[string]any{"maxOutputTokens": p.MaxOutputTokens}
	if en := thinkingEnabled(p); en != nil {
		if *en {
			gc["thinkingConfig"] = map[string]any{"includeThoughts": true}
		} else {
			gc["thinkingConfig"] = map[string]any{"thinkingBudget": 0, "includeThoughts": false}
		}
	}
	body := map[string]any{"system_instruction": map[string]any{"parts": []map[string]any{{"text": pr.System}}}, "contents": contents, "tools": []map[string]any{{"functionDeclarations": decl}}, "generationConfig": gc}
	mergeExtra(body, p.Extra, "contents", "tools", "system_instruction", "generationConfig", "capabilities")
	raw, err := postModelJSON(ctx, ep, p.Protocol, key, body)
	if err != nil {
		return ModelTurn{}, err
	}
	var resp struct {
		Candidates []struct {
			Content map[string]any `json:"content"`
		} `json:"candidates"`
		Usage struct {
			Prompt     int64 `json:"promptTokenCount"`
			Candidates int64 `json:"candidatesTokenCount"`
			Thoughts   int64 `json:"thoughtsTokenCount"`
			Total      int64 `json:"totalTokenCount"`
		} `json:"usageMetadata"`
	}
	if err := json.Unmarshal(raw, &resp); err != nil || len(resp.Candidates) == 0 {
		return ModelTurn{}, errors.New("Gemini 没有返回候选结果")
	}
	t := ModelTurn{Usage: ModelUsage{InputTokens: resp.Usage.Prompt, OutputTokens: resp.Usage.Candidates, ReasoningTokens: resp.Usage.Thoughts, TotalTokens: resp.Usage.Total}}
	content := resp.Candidates[0].Content
	parts, _ := content["parts"].([]any)
	selected := []any{}
	var fc map[string]any
	for _, rawp := range parts {
		part, _ := rawp.(map[string]any)
		if txt, _ := part["text"].(string); txt != "" {
			selected = append(selected, cloneAnyMap(part))
			if thought, _ := part["thought"].(bool); thought {
				t.Reasoning += txt
			} else {
				t.Text += txt
			}
			continue
		}
		if c, ok := part["functionCall"].(map[string]any); ok && fc == nil {
			fc = c
			selected = append(selected, cloneAnyMap(part))
		}
	}
	if fc == nil {
		return t, nil
	}
	pn, _ := fc["name"].(string)
	args, _ := fc["args"].(map[string]any)
	real := hostToolName(names, pn)
	contents = append(contents, map[string]any{"role": "model", "parts": selected})
	t.Tool = &ModelToolCall{Name: real, Arguments: cloneAnyMap(args)}
	t.Replay = &ModelReplayState{Protocol: ProtocolGemini, Tool: real, ToolAlias: pn, CallID: pn, AttachmentIDs: replayAttachmentIDs(r, imgs), Payload: contents}
	return t, nil
}

func decodeToolArguments(v string) (map[string]any, error) {
	m := map[string]any{}
	if strings.TrimSpace(v) == "" {
		return m, nil
	}
	if err := json.Unmarshal([]byte(v), &m); err != nil {
		return nil, fmt.Errorf("模型 Tool 参数不是有效 JSON: %w", err)
	}
	return m, nil
}
func responseMessageText(it map[string]any) string {
	c, _ := it["content"].([]any)
	out := []string{}
	for _, r := range c {
		b, _ := r.(map[string]any)
		if b["type"] == "output_text" {
			if t, _ := b["text"].(string); t != "" {
				out = append(out, t)
			}
		}
	}
	return strings.Join(out, "\n")
}
func responseReasoningSummary(it map[string]any) string {
	s, _ := it["summary"].([]any)
	out := []string{}
	for _, r := range s {
		b, _ := r.(map[string]any)
		if t, _ := b["text"].(string); t != "" {
			out = append(out, t)
		}
	}
	return strings.Join(out, "\n")
}
func parseOpenAIResponsesUsage(v map[string]any) ModelUsage {
	u := ModelUsage{InputTokens: int64FromAny(v["input_tokens"]), OutputTokens: int64FromAny(v["output_tokens"]), TotalTokens: int64FromAny(v["total_tokens"])}
	if d, ok := v["output_tokens_details"].(map[string]any); ok {
		u.ReasoningTokens = int64FromAny(d["reasoning_tokens"])
	}
	return u
}
func int64FromAny(v any) int64 {
	switch x := v.(type) {
	case float64:
		return int64(x)
	case int64:
		return x
	case int:
		return int64(x)
	case json.Number:
		n, _ := x.Int64()
		return n
	}
	return 0
}
func contentString(v any) string {
	switch x := v.(type) {
	case string:
		return x
	case []any:
		out := []string{}
		for _, r := range x {
			if m, ok := r.(map[string]any); ok {
				if t, ok := m["text"].(string); ok {
					out = append(out, t)
				}
			}
		}
		return strings.Join(out, "\n")
	}
	return ""
}
func cloneAnyMap(v map[string]any) map[string]any {
	if v == nil {
		return map[string]any{}
	}
	raw, _ := json.Marshal(v)
	out := map[string]any{}
	_ = json.Unmarshal(raw, &out)
	return out
}
func cloneAnySlice(v []any) []any {
	raw, _ := json.Marshal(v)
	var out []any
	_ = json.Unmarshal(raw, &out)
	return out
}
func mergeExtra(target map[string]any, extra map[string]any, reserved ...string) {
	deny := map[string]bool{}
	for _, k := range reserved {
		deny[k] = true
	}
	for k, v := range extra {
		k = strings.TrimSpace(k)
		if k != "" && !deny[k] {
			target[k] = v
		}
	}
}
