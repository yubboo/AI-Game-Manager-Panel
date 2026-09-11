package app

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	authservice "github.com/yubboo/AI-Game-Manager-Panel/internal/system/auth"
	xiaoyuhost "github.com/yubboo/AI-Game-Manager-Panel/internal/xiaoyu/host"
	xiaoyuruntime "github.com/yubboo/AI-Game-Manager-Panel/internal/xiaoyu/runtime"
)

type XiaoYuHarnessSnapshot struct {
	Kernel       string                        `json:"kernel"`
	PluginAPI    string                        `json:"pluginApi"`
	Brain        *xiaoyuhost.BrainInfo         `json:"brain,omitempty"`
	Capabilities xiaoyuhost.CapabilitySnapshot `json:"capabilities"`
	Runs         []xiaoyuhost.RunState         `json:"runs"`
	Limits       xiaoyuhost.LoopConfig         `json:"limits"`
}

type XiaoYuAttachmentRequest struct {
	Name       string `json:"name,omitempty"`
	MediaType  string `json:"mediaType"`
	DataBase64 string `json:"dataBase64"`
}

type XiaoYuRunRequest struct {
	Goal        string                    `json:"goal"`
	Context     xiaoyuhost.RunContext     `json:"context,omitempty"`
	Attachments []XiaoYuAttachmentRequest `json:"attachments,omitempty"`
}

type XiaoYuContinueRunRequest struct {
	Message     string                    `json:"message,omitempty"`
	Attachments []XiaoYuAttachmentRequest `json:"attachments,omitempty"`
}

type XiaoYuControlRequest struct {
	Reason string `json:"reason,omitempty"`
}

type xiaoyuApplicationExecutor struct {
	app   *Application
	token string
}

func (e xiaoyuApplicationExecutor) Execute(ctx context.Context, call xiaoyuhost.ToolCall) (xiaoyuhost.ToolOutcome, error) {
	result, err := e.app.xiaoyuCallToolForRun(ctx, e.token, call.RunID, xiaoyuruntime.ToolCallRequest{Tool: call.Name, Arguments: call.Arguments, ApprovalID: call.ApprovalID})
	outcome := xiaoyuhost.ToolOutcome{
		Summary:    result.Summary,
		Data:       map[string]any{"decision": result.Decision, "risk": result.Risk, "data": result.Data},
		Pending:    result.Pending,
		Denied:     result.Decision == "deny",
		ApprovalID: result.ApprovalID,
	}
	return outcome, err
}

// hostReceiptValidator only checks whether AGMP Host produced a structurally
// usable execution receipt. It does NOT decide whether the user's goal is done,
// how to recover, or which Tool should run next; those semantic decisions stay
// inside the XiaoYu Rust Agent Runtime.
type hostReceiptValidator struct{}

func (hostReceiptValidator) Validate(_ context.Context, _ string, _ xiaoyuhost.Decision, outcome xiaoyuhost.ToolOutcome) (xiaoyuhost.Validation, error) {
	if outcome.Pending {
		return xiaoyuhost.Validation{OK: false, Summary: "Tool 仍在等待批准"}, nil
	}
	if outcome.Denied {
		return xiaoyuhost.Validation{OK: true, Summary: "Tool 被安全边界拒绝，交回 XiaoYu Agent Runtime 重新规划"}, nil
	}
	return xiaoyuhost.Validation{OK: true, Summary: "AGMP Host 已收到结构化执行结果"}, nil
}

func (a *Application) xiaoyuLoop(token string) (*xiaoyuhost.Loop, xiaoyuhost.BrainInfo, error) {
	user, err := a.requireXiaoYuMember(token)
	if err != nil {
		return nil, xiaoyuhost.BrainInfo{}, err
	}
	if a.xiaoyuHost == nil {
		return nil, xiaoyuhost.BrainInfo{}, errors.New("XiaoYu Host 不可用")
	}
	brain, ok := a.xiaoyuHost.Brain()
	if !ok {
		return nil, xiaoyuhost.BrainInfo{}, xiaoyuhost.ErrBrainUnavailable
	}
	info := brain.Info(a.Context())
	if !info.Ready {
		message := strings.TrimSpace(info.Message)
		if message == "" {
			message = "XiaoYu Model Provider 尚未就绪"
		}
		return nil, info, errors.New(message)
	}
	cfg := a.platformConfig.AI.AgentLoop
	loopConfig := xiaoyuhost.LoopConfig{MaxSteps: cfg.MaxSteps, MaxToolCalls: cfg.MaxToolCalls, MaxFailures: cfg.MaxFailures, RepeatLimit: cfg.RepeatLimit}.Normalized()
	var validator xiaoyuhost.Validator
	if cfg.VerifyAfterAction {
		validator = hostReceiptValidator{}
	}
	loop := xiaoyuhost.NewLoop(
		brain,
		xiaoyuApplicationExecutor{app: a, token: token},
		validator,
		xiaoyuhost.ToolsFromRegistry(a.xiaoyuTools),
		a.xiaoyuHost.Trace(),
		loopConfig,
	)
	if a.xiaoyuIntelligence != nil {
		loop.WithIntelligence(func(state xiaoyuhost.RunState) xiaoyuhost.IntelligenceContext {
			return a.xiaoyuIntelligenceContextForRun(token, user.OrganizationID, state)
		})
	}
	if a.xiaoyuRuns != nil {
		loop.WithThreadContext(func(state xiaoyuhost.RunState) xiaoyuhost.ThreadContext {
			return a.xiaoyuRuns.ThreadContext(state.Context.SessionID, state.ID, 6)
		})
	}
	return loop, info, nil
}

// xiaoyuIntelligenceContextForRun is deliberately fail-closed. Detached Runs
// re-check current membership/core entitlement before every Brain turn, and an
// administrator supervising another member's Run receives organization-shared
// Intelligence only—not that member's Private or Group context.
func (a *Application) xiaoyuIntelligenceContextForRun(token, expectedOrganizationID string, state xiaoyuhost.RunState) xiaoyuhost.IntelligenceContext {
	if a.xiaoyuIntelligence == nil {
		return xiaoyuhost.IntelligenceContext{}
	}
	current, err := a.auth.RequireCoreAccess(token)
	if err != nil || current.OrganizationID == "" || current.OrganizationID != strings.TrimSpace(expectedOrganizationID) {
		return xiaoyuhost.IntelligenceContext{}
	}
	groupID, userID := current.GroupID, current.ID
	if state.InitiatorID != "" && state.InitiatorID != current.ID {
		if !isXiaoYuSupervisor(current) {
			return xiaoyuhost.IntelligenceContext{}
		}
		groupID, userID = "", ""
	}
	ctx := a.xiaoyuIntelligence.ContextForGoal(current.OrganizationID, groupID, userID, state.Context, state.Goal)
	ctx.ModuleCatalog = a.xiaoyuSystemModuleCatalog()
	ctx.Modules = a.xiaoyuSystemModules(state.Goal, state.Context)
	return ctx
}

// XiaoYuHarnessStatus exposes the live Agent Harness graph without leaking
// model credentials or plugin-private configuration.
func (a *Application) XiaoYuHarnessStatus(token string) (XiaoYuHarnessSnapshot, error) {
	user, err := a.requireXiaoYuMember(token)
	if err != nil {
		return XiaoYuHarnessSnapshot{}, err
	}
	if a.xiaoyuHost == nil || a.xiaoyuRuns == nil {
		return XiaoYuHarnessSnapshot{}, errors.New("XiaoYu Harness 不可用")
	}
	cfg := a.platformConfig.AI.AgentLoop
	loopConfig := xiaoyuhost.LoopConfig{MaxSteps: cfg.MaxSteps, MaxToolCalls: cfg.MaxToolCalls, MaxFailures: cfg.MaxFailures, RepeatLimit: cfg.RepeatLimit}.Normalized()
	kernelName := strings.TrimSpace(a.platformConfig.AI.Harness.Kernel)
	if kernelName == "" {
		kernelName = "xiaoyu.host.v1"
	}
	value := XiaoYuHarnessSnapshot{
		Kernel:       kernelName,
		PluginAPI:    xiaoyuhost.PluginAPIVersion,
		Capabilities: a.xiaoyuHost.Capabilities(),
		Runs:         filterXiaoYuRunsForUser(user, a.xiaoyuRuns.List()),
		Limits:       loopConfig,
	}
	if brain, ok := a.xiaoyuHost.Brain(); ok {
		info := brain.Info(a.Context())
		value.Brain = &info
	}
	return value, nil
}

const (
	xiaoYuMaxAttachments         = 4
	xiaoYuMaxAttachmentBytes     = 5 * 1024 * 1024
	xiaoYuMaxAttachmentsBytes    = 12 * 1024 * 1024
	xiaoYuMaxRunAttachments      = 8
	xiaoYuMaxRunAttachmentsBytes = 20 * 1024 * 1024
)

func normalizeXiaoYuAttachments(requests []XiaoYuAttachmentRequest) ([]xiaoyuhost.ModelInputAttachment, error) {
	if len(requests) == 0 {
		return nil, nil
	}
	if len(requests) > xiaoYuMaxAttachments {
		return nil, fmt.Errorf("XiaoYu 一次最多允许 %d 张图片", xiaoYuMaxAttachments)
	}
	allowed := map[string]bool{"image/png": true, "image/jpeg": true, "image/webp": true, "image/gif": true}
	result := make([]xiaoyuhost.ModelInputAttachment, 0, len(requests))
	total := 0
	for _, item := range requests {
		mediaType := strings.ToLower(strings.TrimSpace(item.MediaType))
		if !allowed[mediaType] {
			return nil, fmt.Errorf("XiaoYu 暂不支持图片类型 %q", item.MediaType)
		}
		raw, err := base64.StdEncoding.DecodeString(strings.TrimSpace(item.DataBase64))
		if err != nil {
			return nil, errors.New("XiaoYu 图片附件不是有效 Base64")
		}
		if len(raw) == 0 || len(raw) > xiaoYuMaxAttachmentBytes {
			return nil, fmt.Errorf("XiaoYu 单张图片必须在 1 byte 到 %d MiB 之间", xiaoYuMaxAttachmentBytes/(1024*1024))
		}
		total += len(raw)
		if total > xiaoYuMaxAttachmentsBytes {
			return nil, fmt.Errorf("XiaoYu 图片总大小不能超过 %d MiB", xiaoYuMaxAttachmentsBytes/(1024*1024))
		}
		sum := sha256.Sum256(raw)
		result = append(result, xiaoyuhost.ModelInputAttachment{ID: "img_" + hex.EncodeToString(sum[:8]), Name: strings.TrimSpace(item.Name), MediaType: mediaType, Data: raw})
	}
	return result, nil
}

func validateXiaoYuRunAttachmentBudget(existing, incoming []xiaoyuhost.ModelInputAttachment) error {
	seen := map[string]bool{}
	count, total := 0, 0
	for _, group := range [][]xiaoyuhost.ModelInputAttachment{existing, incoming} {
		for _, item := range group {
			if seen[item.ID] {
				continue
			}
			seen[item.ID] = true
			count++
			total += len(item.Data)
		}
	}
	if count > xiaoYuMaxRunAttachments {
		return fmt.Errorf("XiaoYu 单个任务最多保留 %d 张图片", xiaoYuMaxRunAttachments)
	}
	if total > xiaoYuMaxRunAttachmentsBytes {
		return fmt.Errorf("XiaoYu 单个任务图片总大小不能超过 %d MiB", xiaoYuMaxRunAttachmentsBytes/(1024*1024))
	}
	return nil
}

// XiaoYuStartRun creates server-owned autonomous work and detaches execution
// from the calling browser/WebView. Closing a tab only disconnects the client;
// the Run stays alive under Application.Context().
func (a *Application) XiaoYuStartRun(token string, request XiaoYuRunRequest) (xiaoyuhost.RunState, error) {
	user, err := a.requireXiaoYuMember(token)
	if err != nil {
		return xiaoyuhost.RunState{}, err
	}
	if a.xiaoyuRuns == nil {
		return xiaoyuhost.RunState{}, errors.New("XiaoYu RunManager 不可用")
	}
	goal := strings.TrimSpace(request.Goal)
	if goal == "" {
		return xiaoyuhost.RunState{}, errors.New("XiaoYu 自主任务目标不能为空")
	}
	if len([]byte(goal)) > 16*1024 {
		return xiaoyuhost.RunState{}, errors.New("XiaoYu 自主任务目标超过 16KiB 安全上限")
	}
	maxConcurrent := maxInt(a.platformConfig.AI.MaxConcurrentRuns, 1)
	if a.xiaoyuRuns.ActiveCount() >= maxConcurrent {
		return xiaoyuhost.RunState{}, fmt.Errorf("XiaoYu 当前并发任务已达到上限 %d", maxConcurrent)
	}
	loop, _, err := a.xiaoyuLoop(token)
	if err != nil {
		return xiaoyuhost.RunState{}, err
	}
	runContext, err := normalizeXiaoYuRunContext(request.Context)
	if err != nil {
		return xiaoyuhost.RunState{}, err
	}
	attachments, err := normalizeXiaoYuAttachments(request.Attachments)
	if err != nil {
		return xiaoyuhost.RunState{}, err
	}
	state, err := a.xiaoyuRuns.CreateForContext(goal, user.ID, xiaoyuActorLabel(user.DisplayName, user.Username), runContext)
	if err != nil {
		return xiaoyuhost.RunState{}, err
	}
	if len(attachments) > 0 {
		state, err = a.xiaoyuRuns.SetAttachments(state.ID, attachments)
		if err != nil {
			return xiaoyuhost.RunState{}, err
		}
	}
	a.observeXiaoYuRun("run/created", state.ID, "自主任务已创建", map[string]any{"goal": safeXiaoYuTraceText(state.Goal), "actor": xiaoyuActorLabel(user.DisplayName, user.Username)})
	return a.xiaoyuRuns.AdvanceAsync(a.Context(), state.ID, loop)
}

// XiaoYuContinueRun resumes a waiting Run and again detaches its execution from
// the transport. Approval itself remains a separate one-shot fingerprinted
// action; this method never fabricates approval state.
func (a *Application) XiaoYuContinueRun(token, id string, request XiaoYuContinueRunRequest) (xiaoyuhost.RunState, error) {
	user, err := a.requireXiaoYuMember(token)
	if err != nil {
		return xiaoyuhost.RunState{}, err
	}
	if a.xiaoyuRuns == nil {
		return xiaoyuhost.RunState{}, errors.New("XiaoYu RunManager 不可用")
	}
	state, ok := a.xiaoyuRuns.Get(id)
	if !ok {
		return xiaoyuhost.RunState{}, fmt.Errorf("%w: %s", xiaoyuhost.ErrRunNotFound, id)
	}
	if !canControlXiaoYuRun(user, state) {
		return xiaoyuhost.RunState{}, authservice.ErrForbidden
	}
	if state.Status == xiaoyuhost.RunPaused || state.ControlOwner == xiaoyuhost.ControlHuman {
		return state, xiaoyuhost.ErrRunHumanOwned
	}
	switch state.Status {
	case xiaoyuhost.RunCompleted, xiaoyuhost.RunFailed, xiaoyuhost.RunCancelled:
		return state, fmt.Errorf("%w: %s", xiaoyuhost.ErrRunTerminal, state.Status)
	}
	attachments, err := normalizeXiaoYuAttachments(request.Attachments)
	if err != nil {
		return state, err
	}
	if err := validateXiaoYuRunAttachmentBudget(state.Attachments, attachments); err != nil {
		return state, err
	}
	if len(attachments) > 0 {
		state, err = a.xiaoyuRuns.AppendAttachments(id, attachments)
		if err != nil {
			return state, err
		}
	}
	message := strings.TrimSpace(request.Message)
	if message == "" && len(attachments) > 0 {
		message = "请查看我刚附加的图片，并结合当前任务继续处理。"
	}
	steeredActiveTurn := false
	if message != "" {
		if len([]byte(message)) > 16*1024 {
			return state, errors.New("XiaoYu 用户补充信息超过 16KiB 安全上限")
		}
		actor := xiaoyuActorLabel(user.DisplayName, user.Username)
		// A new human instruction supersedes an unresolved approval. Reject the
		// old request server-side so it can never be replayed after replanning.
		if state.Status == xiaoyuhost.RunWaitingApproval && strings.TrimSpace(state.ApprovalID) != "" {
			_, _ = a.xiaoyuApprovals.Resolve(state.ApprovalID, false, actor)
		}
		if state.Status == xiaoyuhost.RunWaitingUser && !a.xiaoyuRuns.IsActive(id) {
			state, err = a.xiaoyuRuns.ResumeUser(id, message)
		} else {
			steeredActiveTurn = a.xiaoyuRuns.IsActive(id)
			state, err = a.xiaoyuRuns.Steer(id, message, actor)
		}
		if err != nil {
			return state, err
		}
	}
	loop, _, err := a.xiaoyuLoop(token)
	if err != nil {
		return state, err
	}
	if steeredActiveTurn && a.xiaoyuRuns.IsActive(id) {
		// Codex-style steering: return immediately to the UI, then restart the
		// same Run after the cancelled turn reaches the safe Host boundary.
		go a.xiaoyuAdvanceAfterSteer(id, loop)
		return state, nil
	}
	if a.xiaoyuRuns.IsActive(id) {
		return state, xiaoyuhost.ErrRunBusy
	}
	return a.xiaoyuRuns.AdvanceAsync(a.Context(), id, loop)
}

func (a *Application) xiaoyuAdvanceAfterSteer(id string, loop *xiaoyuhost.Loop) {
	deadline := time.Now().Add(3 * time.Second)
	for a.xiaoyuRuns != nil && a.xiaoyuRuns.IsActive(id) && time.Now().Before(deadline) {
		select {
		case <-a.Context().Done():
			return
		case <-time.After(20 * time.Millisecond):
		}
	}
	if a.xiaoyuRuns == nil || a.xiaoyuRuns.IsActive(id) {
		return
	}
	state, ok := a.xiaoyuRuns.Get(id)
	if !ok || state.Status == xiaoyuhost.RunCompleted || state.Status == xiaoyuhost.RunFailed || state.Status == xiaoyuhost.RunCancelled || state.Status == xiaoyuhost.RunPaused {
		return
	}
	_, _ = a.xiaoyuRuns.AdvanceAsync(a.Context(), id, loop)
}

func (a *Application) XiaoYuRunState(token, id string) (xiaoyuhost.RunState, error) {
	user, err := a.requireXiaoYuMember(token)
	if err != nil {
		return xiaoyuhost.RunState{}, err
	}
	state, ok := a.xiaoyuRuns.Get(id)
	if !ok {
		return xiaoyuhost.RunState{}, fmt.Errorf("%w: %s", xiaoyuhost.ErrRunNotFound, id)
	}
	if !canControlXiaoYuRun(user, state) {
		return xiaoyuhost.RunState{}, authservice.ErrForbidden
	}
	return state, nil
}

func (a *Application) XiaoYuCancelRun(token, id string) (xiaoyuhost.RunState, error) {
	user, err := a.requireXiaoYuMember(token)
	if err != nil {
		return xiaoyuhost.RunState{}, err
	}
	state, err := requireXiaoYuRunControl(a.xiaoyuRuns, user, id)
	if err != nil {
		return state, err
	}
	state, err = a.xiaoyuRuns.Cancel(id)
	if err == nil {
		a.observeXiaoYuRun("run/cancel-requested", id, "用户请求取消任务", map[string]any{"actor": xiaoyuActorLabel(user.DisplayName, user.Username)})
	}
	return state, err
}

func (a *Application) XiaoYuPauseRun(token, id string, request XiaoYuControlRequest) (xiaoyuhost.RunState, error) {
	user, err := a.requireXiaoYuMember(token)
	if err != nil {
		return xiaoyuhost.RunState{}, err
	}
	actor := xiaoyuActorLabel(user.DisplayName, user.Username)
	state, err := requireXiaoYuRunControl(a.xiaoyuRuns, user, id)
	if err != nil {
		return state, err
	}
	state, err = a.xiaoyuRuns.Pause(id, actor, strings.TrimSpace(request.Reason))
	if err == nil {
		a.observeXiaoYuRun("run/paused", id, "用户暂停任务", map[string]any{"actor": actor, "reason": safeXiaoYuTraceText(request.Reason)})
	}
	return state, err
}

func (a *Application) XiaoYuTakeoverRun(token, id string, request XiaoYuControlRequest) (xiaoyuhost.RunState, error) {
	user, err := a.requireXiaoYuMember(token)
	if err != nil {
		return xiaoyuhost.RunState{}, err
	}
	actor := xiaoyuActorLabel(user.DisplayName, user.Username)
	state, err := requireXiaoYuRunControl(a.xiaoyuRuns, user, id)
	if err != nil {
		return state, err
	}
	state, err = a.xiaoyuRuns.Takeover(id, actor, strings.TrimSpace(request.Reason))
	if err == nil {
		a.observeXiaoYuRun("run/takeover", id, "用户接管任务", map[string]any{"actor": actor, "reason": safeXiaoYuTraceText(request.Reason)})
	}
	return state, err
}

func (a *Application) XiaoYuResumeRun(token, id string) (xiaoyuhost.RunState, error) {
	user, err := a.requireXiaoYuMember(token)
	if err != nil {
		return xiaoyuhost.RunState{}, err
	}
	state, err := requireXiaoYuRunControl(a.xiaoyuRuns, user, id)
	if err != nil {
		return state, err
	}
	state, err = a.xiaoyuRuns.ReleaseToXiaoYu(id)
	if err != nil {
		return state, err
	}
	actor := xiaoyuActorLabel(user.DisplayName, user.Username)
	a.observeXiaoYuRun("run/resumed", id, "控制权已交还小鱼", map[string]any{"actor": actor, "controlOwner": string(xiaoyuhost.ControlXiaoYu)})
	// Releasing human ownership must not bypass an unresolved approval or a
	// question that explicitly requires user input. Those states remain
	// suspended until the normal approval/continue API resolves them.
	if state.Status == xiaoyuhost.RunWaitingApproval || state.Status == xiaoyuhost.RunWaitingUser {
		return state, nil
	}
	loop, _, err := a.xiaoyuLoop(token)
	if err != nil {
		return state, err
	}
	return a.xiaoyuRuns.AdvanceAsync(a.Context(), id, loop)
}

// XiaoYuEventStream is a transport-neutral server-owned stream. HTTP uses it
// for SSE; desktop adapters may use the same sequence source. Cancelling the
// returned subscription never cancels the underlying Run.
func (a *Application) XiaoYuEventStream(token string, after uint64) (<-chan xiaoyuhost.Event, func(), error) {
	user, err := a.requireXiaoYuMember(token)
	if err != nil {
		return nil, nil, err
	}
	if a.xiaoyuHost == nil {
		return nil, nil, errors.New("XiaoYu Host 不可用")
	}
	filter := a.xiaoyuEventFilter(user)
	stream, cancel := a.xiaoyuHost.Trace().SubscribeFiltered(after, 512, filter)
	return stream, cancel, nil
}

func isXiaoYuSupervisor(user authservice.User) bool {
	return user.Role == authservice.RoleOwner || user.Role == authservice.RoleAdministrator
}

func canControlXiaoYuRun(user authservice.User, state xiaoyuhost.RunState) bool {
	if isXiaoYuSupervisor(user) {
		return true
	}
	return strings.TrimSpace(state.InitiatorID) != "" && state.InitiatorID == user.ID
}

func requireXiaoYuRunControl(runs *xiaoyuhost.RunManager, user authservice.User, id string) (xiaoyuhost.RunState, error) {
	if runs == nil {
		return xiaoyuhost.RunState{}, errors.New("XiaoYu RunManager 不可用")
	}
	state, ok := runs.Get(id)
	if !ok {
		return xiaoyuhost.RunState{}, fmt.Errorf("%w: %s", xiaoyuhost.ErrRunNotFound, id)
	}
	if !canControlXiaoYuRun(user, state) {
		return xiaoyuhost.RunState{}, authservice.ErrForbidden
	}
	return state, nil
}

func filterXiaoYuRunsForUser(user authservice.User, runs []xiaoyuhost.RunState) []xiaoyuhost.RunState {
	if isXiaoYuSupervisor(user) {
		return runs
	}
	result := make([]xiaoyuhost.RunState, 0, len(runs))
	for _, state := range runs {
		if canControlXiaoYuRun(user, state) {
			result = append(result, state)
		}
	}
	return result
}

func (a *Application) xiaoyuEventFilter(user authservice.User) xiaoyuhost.EventFilter {
	if isXiaoYuSupervisor(user) {
		return nil
	}
	return func(event xiaoyuhost.Event) bool {
		if strings.TrimSpace(event.RunID) == "" {
			return false
		}
		if a.xiaoyuRuns == nil {
			return false
		}
		state, ok := a.xiaoyuRuns.Get(event.RunID)
		return ok && canControlXiaoYuRun(user, state)
	}
}

func normalizeXiaoYuRunContext(value xiaoyuhost.RunContext) (xiaoyuhost.RunContext, error) {
	clean := func(name, raw string, max int) (string, error) {
		raw = strings.TrimSpace(raw)
		if len([]byte(raw)) > max {
			return "", fmt.Errorf("XiaoYu %s 超过 %d 字节安全上限", name, max)
		}
		return raw, nil
	}
	var err error
	value.SessionID, err = clean("sessionId", value.SessionID, 160)
	if err != nil {
		return xiaoyuhost.RunContext{}, err
	}
	// TaskID is server-owned and replaced by RunManager with the created Run ID.
	value.TaskID = ""
	value.GameID, err = clean("gameId", value.GameID, 128)
	if err != nil {
		return xiaoyuhost.RunContext{}, err
	}
	value.ServerID, err = clean("serverId", value.ServerID, 256)
	if err != nil {
		return xiaoyuhost.RunContext{}, err
	}
	value.InstanceID, err = clean("instanceId", value.InstanceID, 256)
	if err != nil {
		return xiaoyuhost.RunContext{}, err
	}
	value.UIRoute, err = clean("uiRoute", value.UIRoute, 512)
	if err != nil {
		return xiaoyuhost.RunContext{}, err
	}
	if value.UIRoute != "" && (!strings.HasPrefix(value.UIRoute, "/") || strings.HasPrefix(value.UIRoute, "//") || strings.Contains(value.UIRoute, "://")) {
		return xiaoyuhost.RunContext{}, errors.New("XiaoYu uiRoute 只允许 AGMP 站内路径")
	}
	return value, nil
}

func xiaoyuActorLabel(displayName, username string) string {
	if value := strings.TrimSpace(displayName); value != "" {
		return value
	}
	return strings.TrimSpace(username)
}
