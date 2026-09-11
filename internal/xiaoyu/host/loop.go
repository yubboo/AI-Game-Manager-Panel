package host

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	xiaoyucontract "github.com/yubboo/AI-Game-Manager-Panel/internal/xiaoyu/contract"
)

type RunStatus string

const (
	RunCreated         RunStatus = "created"
	RunRunning         RunStatus = "running"
	RunWaitingApproval RunStatus = "waiting_approval"
	RunWaitingUser     RunStatus = "waiting_user"
	RunPaused          RunStatus = "paused"
	RunCompleted       RunStatus = "completed"
	RunFailed          RunStatus = "failed"
	RunCancelled       RunStatus = "cancelled"
)

type AgentPhase string

const (
	PhaseUnderstanding   AgentPhase = "understanding"
	PhasePlanning        AgentPhase = "planning"
	PhaseExecuting       AgentPhase = "executing"
	PhaseVerifying       AgentPhase = "verifying"
	PhaseRecovering      AgentPhase = "recovering"
	PhaseWaitingApproval AgentPhase = "waiting_approval"
	PhaseWaitingUser     AgentPhase = "waiting_user"
	PhasePaused          AgentPhase = "paused"
	PhaseCompleted       AgentPhase = "completed"
	PhaseFailed          AgentPhase = "failed"
	PhaseCancelled       AgentPhase = "cancelled"
)

type DecisionKind string

const (
	DecisionTool     DecisionKind = "tool"
	DecisionComplete DecisionKind = "complete"
	DecisionFail     DecisionKind = "fail"
	DecisionWait     DecisionKind = "wait"
)

type Decision struct {
	Kind      DecisionKind   `json:"kind"`
	Tool      string         `json:"tool,omitempty"`
	Arguments map[string]any `json:"arguments,omitempty"`
	Message   string         `json:"message,omitempty"`
}

type Observation struct {
	Step       int       `json:"step"`
	Tool       string    `json:"tool,omitempty"`
	Summary    string    `json:"summary"`
	Data       any       `json:"data,omitempty"`
	Error      string    `json:"error,omitempty"`
	Pending    bool      `json:"pending,omitempty"`
	Denied     bool      `json:"denied,omitempty"`
	ApprovalID string    `json:"approvalId,omitempty"`
	Time       time.Time `json:"time"`
}

// ThreadAction is a bounded, cross-Run receipt used to preserve conversational
// continuity without replaying an unbounded raw transcript into the model.
type ThreadUndoHint struct {
	Tool      string         `json:"tool"`
	Arguments map[string]any `json:"arguments"`
	Summary   string         `json:"summary,omitempty"`
}

type ThreadAction struct {
	Tool    string          `json:"tool,omitempty"`
	Summary string          `json:"summary,omitempty"`
	Data    any             `json:"data,omitempty"`
	Error   string          `json:"error,omitempty"`
	Undo    *ThreadUndoHint `json:"undo,omitempty"`
}

type ThreadTurn struct {
	RunID   string         `json:"runId"`
	Goal    string         `json:"goal"`
	Status  RunStatus      `json:"status"`
	Message string         `json:"message,omitempty"`
	Actions []ThreadAction `json:"actions,omitempty"`
}

type ThreadContext struct {
	RecentTurns []ThreadTurn `json:"recentTurns,omitempty"`
}

type RunContext struct {
	SessionID  string `json:"sessionId,omitempty"`
	TaskID     string `json:"taskId,omitempty"`
	GameID     string `json:"gameId,omitempty"`
	ServerID   string `json:"serverId,omitempty"`
	InstanceID string `json:"instanceId,omitempty"`
	UIRoute    string `json:"uiRoute,omitempty"`
}

type CapabilityContext struct {
	ToolCount  int      `json:"toolCount"`
	Categories []string `json:"categories,omitempty"`
	Read       int      `json:"read"`
	Operate    int      `json:"operate"`
	Modify     int      `json:"modify"`
	HighRisk   int      `json:"highRisk"`
}

type AgentContext struct {
	Phase           AgentPhase        `json:"phase"`
	DecisionSummary string            `json:"decisionSummary,omitempty"`
	Capabilities    CapabilityContext `json:"capabilities"`
}

type ModelInputAttachment struct {
	ID        string `json:"id"`
	Name      string `json:"name,omitempty"`
	MediaType string `json:"mediaType"`
	Data      []byte `json:"-"`
}

type Frame struct {
	Attachments  []ModelInputAttachment `json:"-"`
	RunID        string                 `json:"runId"`
	Goal         string                 `json:"goal"`
	Step         int                    `json:"step"`
	Budget       FrameBudget            `json:"budget"`
	Tools        []xiaoyuToolView       `json:"tools"`
	Context      RunContext             `json:"context"`
	Agent        AgentContext           `json:"agent"`
	Thread       ThreadContext          `json:"thread"`
	Intelligence IntelligenceContext    `json:"intelligence"`
	Observations []Observation          `json:"observations"`
}

type FrameBudget struct {
	StepsRemaining     int `json:"stepsRemaining"`
	ToolCallsRemaining int `json:"toolCallsRemaining"`
	FailuresRemaining  int `json:"failuresRemaining"`
}

type xiaoyuToolView struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Risk        string         `json:"risk"`
	Category    string         `json:"category"`
	Tier        string         `json:"tier"`
	Source      string         `json:"source,omitempty"`
	Parameters  map[string]any `json:"parameters,omitempty"`
}

type Brain interface {
	Next(context.Context, Frame) (Decision, error)
}

type ToolCall struct {
	RunID      string         `json:"runId,omitempty"`
	Name       string         `json:"name"`
	Arguments  map[string]any `json:"arguments,omitempty"`
	ApprovalID string         `json:"approvalId,omitempty"`
}

type ToolOutcome struct {
	Summary    string `json:"summary"`
	Data       any    `json:"data,omitempty"`
	Pending    bool   `json:"pending,omitempty"`
	Denied     bool   `json:"denied,omitempty"`
	ApprovalID string `json:"approvalId,omitempty"`
}

type Executor interface {
	Execute(context.Context, ToolCall) (ToolOutcome, error)
}

type Validation struct {
	OK      bool   `json:"ok"`
	Summary string `json:"summary,omitempty"`
}

type Validator interface {
	Validate(context.Context, string, Decision, ToolOutcome) (Validation, error)
}

type LoopConfig struct {
	MaxSteps     int `json:"maxSteps"`
	MaxToolCalls int `json:"maxToolCalls"`
	MaxFailures  int `json:"maxFailures"`
	RepeatLimit  int `json:"repeatLimit"`
}

func (c LoopConfig) Normalized() LoopConfig {
	if c.MaxSteps <= 0 {
		c.MaxSteps = 64
	}
	if c.MaxToolCalls <= 0 {
		c.MaxToolCalls = 48
	}
	if c.MaxFailures <= 0 {
		c.MaxFailures = 8
	}
	if c.RepeatLimit <= 0 {
		c.RepeatLimit = 3
	}
	return c
}

type ControlOwner string

const (
	ControlXiaoYu ControlOwner = "xiaoyu"
	ControlHuman  ControlOwner = "human"
	ControlShared ControlOwner = "shared"
)

type RunState struct {
	Attachments     []ModelInputAttachment `json:"-"`
	ID              string                 `json:"id"`
	Goal            string                 `json:"goal"`
	Status          RunStatus              `json:"status"`
	ControlOwner    ControlOwner           `json:"controlOwner"`
	InitiatorID     string                 `json:"initiatorId,omitempty"`
	InitiatorName   string                 `json:"initiatorName,omitempty"`
	Context         RunContext             `json:"context"`
	PausedBy        string                 `json:"pausedBy,omitempty"`
	PauseReason     string                 `json:"pauseReason,omitempty"`
	ResumeStatus    RunStatus              `json:"-"`
	Step            int                    `json:"step"`
	ToolCalls       int                    `json:"toolCalls"`
	Failures        int                    `json:"failures"`
	Message         string                 `json:"message,omitempty"`
	Phase           AgentPhase             `json:"phase,omitempty"`
	DecisionSummary string                 `json:"decisionSummary,omitempty"`
	ApprovalID      string                 `json:"approvalId,omitempty"`
	PendingCall     *ToolCall              `json:"pendingCall,omitempty"`
	Observations    []Observation          `json:"observations"`
	LastAction      string                 `json:"-"`
	RepeatStreak    int                    `json:"-"`
	CreatedAt       time.Time              `json:"createdAt"`
	UpdatedAt       time.Time              `json:"updatedAt"`
}

type Loop struct {
	brain        Brain
	executor     Executor
	validator    Validator
	tools        func() []xiaoyuToolView
	trace        *Trace
	config       LoopConfig
	intelligence func(RunState) IntelligenceContext
	thread       func(RunState) ThreadContext
	stateSink    func(RunState)
}

func NewLoop(brain Brain, executor Executor, validator Validator, tools func() []xiaoyuToolView, trace *Trace, config LoopConfig) *Loop {
	if trace == nil {
		trace = NewTrace()
	}
	if tools == nil {
		tools = func() []xiaoyuToolView { return nil }
	}
	return &Loop{brain: brain, executor: executor, validator: validator, tools: tools, trace: trace, config: config.Normalized()}
}

// WithIntelligence attaches a read-only, host-filtered intelligence resolver.
// The resolver cannot execute tools or elevate permissions; it only supplies
// Memory/Skill/Expert context already filtered for the current user/run.
func (l *Loop) WithIntelligence(resolve func(RunState) IntelligenceContext) *Loop {
	l.intelligence = resolve
	return l
}

// WithThreadContext injects a bounded session/thread history. This mirrors the
// steerable-thread model used by mature agent runtimes: a new Run can understand
// references such as “改回去/继续刚才的任务” without trusting browser-local chat text.
func (l *Loop) WithThreadContext(resolve func(RunState) ThreadContext) *Loop {
	l.thread = resolve
	return l
}

// WithStateSink publishes safe, user-visible execution snapshots while a Run is
// still advancing in the background. It does not make semantic decisions; it
// only prevents the UI from being stuck at "step 0 / tool 0" until completion.
func (l *Loop) WithStateSink(sink func(RunState)) *Loop {
	l.stateSink = sink
	return l
}

func (l *Loop) publish(state RunState) {
	if l.stateSink == nil {
		return
	}
	l.stateSink(state)
}

func (l *Loop) NewRun(id, goal string) RunState {
	now := time.Now()
	return RunState{ID: strings.TrimSpace(id), Goal: strings.TrimSpace(goal), Status: RunCreated, Phase: PhaseUnderstanding, ControlOwner: ControlXiaoYu, CreatedAt: now, UpdatedAt: now}
}

// Advance runs autonomously until the goal completes, a safety budget is hit,
// user input is required, or an approval pauses execution. Re-enter Advance with
// the returned state after the caller resolves the approval/user dependency.
func (l *Loop) Advance(ctx context.Context, state RunState) RunState {
	if state.Status == RunCompleted || state.Status == RunFailed || state.Status == RunCancelled || state.Status == RunPaused {
		return state
	}
	if state.ControlOwner == ControlHuman {
		state.Status = RunPaused
		state.Phase = PhasePaused
		state.Message = "任务已由人工接管"
		state.DecisionSummary = "等待人工处理"
		state.UpdatedAt = time.Now()
		l.publish(state)
		return state
	}
	if l.brain == nil || l.executor == nil {
		return l.fail(state, "XiaoYu Harness 缺少 Brain 或 Tool Executor")
	}
	previousStatus := state.Status
	state.Status = RunRunning
	if state.Phase == "" {
		state.Phase = PhaseUnderstanding
	}
	state.UpdatedAt = time.Now()
	l.publish(state)
	l.record(state.ID, "run/started", "小鱼开始处理目标")

	// Approval is a suspended Tool call, not a new reasoning turn. After the
	// user resolves it, resume the exact fingerprinted call with its approvalId
	// before asking the Brain for another decision.
	if previousStatus == RunWaitingApproval && state.PendingCall != nil {
		pending := *state.PendingCall
		pending.ApprovalID = state.ApprovalID
		state.Phase = PhaseExecuting
		state.DecisionSummary = "继续执行已批准步骤：" + pending.Name
		state.UpdatedAt = time.Now()
		l.publish(state)
		outcome, execErr := l.executor.Execute(ctx, pending)
		observation := Observation{Step: state.Step, Tool: pending.Name, Summary: outcome.Summary, Data: outcome.Data, Pending: outcome.Pending, Denied: outcome.Denied, ApprovalID: outcome.ApprovalID, Time: time.Now()}
		if execErr != nil {
			observation.Error = execErr.Error()
			state.Failures++
		}
		state.Observations = append(state.Observations, observation)
		if outcome.Pending {
			state.Status = RunWaitingApproval
			state.Phase = PhaseWaitingApproval
			state.DecisionSummary = "等待批准后继续：" + pending.Name
			if outcome.ApprovalID != "" {
				state.ApprovalID = outcome.ApprovalID
			}
			state.UpdatedAt = time.Now()
			l.publish(state)
			return state
		}
		state.PendingCall = nil
		state.ApprovalID = ""
		if execErr != nil || outcome.Denied {
			state.Phase = PhaseRecovering
			state.DecisionSummary = "执行结果异常，正在重新诊断"
		} else {
			state.Phase = PhaseVerifying
			state.DecisionSummary = "步骤已执行，正在核对真实结果"
		}
		state.UpdatedAt = time.Now()
		l.publish(state)
		if state.Failures >= l.config.MaxFailures {
			return l.failWith(state, ErrAgentFailureBudget)
		}
	}

	for {
		if err := ctx.Err(); err != nil {
			state.Status = RunCancelled
			state.Phase = PhaseCancelled
			state.Message = err.Error()
			state.DecisionSummary = "任务已停止"
			state.UpdatedAt = time.Now()
			l.publish(state)
			l.record(state.ID, "run/cancelled", state.Message)
			return state
		}
		if state.Step >= l.config.MaxSteps {
			return l.failWith(state, ErrAgentStepBudget)
		}
		state.Step++
		if lastObservationFailed(state.Observations) {
			state.Phase = PhaseRecovering
			state.DecisionSummary = "发现上一步异常，正在诊断并寻找安全替代方案"
		} else if state.ToolCalls == 0 {
			state.Phase = PhaseUnderstanding
			state.DecisionSummary = "正在理解目标并选择最合适的 AGMP 能力"
		} else {
			state.Phase = PhasePlanning
			state.DecisionSummary = "正在根据最新观察规划下一步"
		}
		state.UpdatedAt = time.Now()
		l.publish(state)

		intelligence := IntelligenceContext{}
		if l.intelligence != nil {
			intelligence = l.intelligence(state)
		}
		thread := ThreadContext{}
		if l.thread != nil {
			thread = l.thread(state)
		}
		tools := l.tools()
		decision, err := l.brain.Next(ctx, Frame{
			Attachments: cloneModelInputAttachments(state.Attachments),
			RunID:       state.ID, Goal: state.Goal, Step: state.Step, Context: state.Context, Thread: thread, Intelligence: intelligence,
			Agent: AgentContext{Phase: state.Phase, DecisionSummary: state.DecisionSummary, Capabilities: buildCapabilityContext(tools)},
			Budget: FrameBudget{
				StepsRemaining:     maxZero(l.config.MaxSteps - state.Step),
				ToolCallsRemaining: maxZero(l.config.MaxToolCalls - state.ToolCalls),
				FailuresRemaining:  maxZero(l.config.MaxFailures - state.Failures),
			},
			Tools: tools, Observations: observationsForModel(state.Observations),
		})
		if err != nil {
			state.Failures++
			state.Phase = PhaseRecovering
			state.DecisionSummary = "大脑调用异常，正在尝试安全恢复"
			state.Observations = append(state.Observations, Observation{Step: state.Step, Summary: "Brain 返回错误", Error: err.Error(), Time: time.Now()})
			state.UpdatedAt = time.Now()
			l.publish(state)
			if state.Failures >= l.config.MaxFailures {
				return l.failWith(state, ErrAgentFailureBudget)
			}
			continue
		}
		switch decision.Kind {
		case DecisionComplete:
			if reason := completionEvidenceGap(state, tools); reason != "" {
				state.Phase = PhaseVerifying
				state.DecisionSummary = reason
				state.Observations = append(state.Observations, Observation{Step: state.Step, Summary: "完成判定暂缓", Data: map[string]any{"reason": reason}, Time: time.Now()})
				state.UpdatedAt = time.Now()
				l.publish(state)
				l.record(state.ID, "run/verification-required", reason)
				continue
			}
			state.Status = RunCompleted
			state.Phase = PhaseCompleted
			state.Message = strings.TrimSpace(decision.Message)
			state.DecisionSummary = state.Message
			state.UpdatedAt = time.Now()
			l.publish(state)
			l.record(state.ID, "run/completed", state.Message)
			return state
		case DecisionFail:
			if strings.TrimSpace(decision.Message) == "" {
				decision.Message = "小鱼判断当前目标无法继续"
			}
			return l.fail(state, decision.Message)
		case DecisionWait:
			state.Status = RunWaitingUser
			state.Phase = PhaseWaitingUser
			state.Message = strings.TrimSpace(decision.Message)
			state.DecisionSummary = state.Message
			state.UpdatedAt = time.Now()
			l.publish(state)
			l.record(state.ID, "run/waiting-user", state.Message)
			return state
		case DecisionTool:
			if state.ToolCalls >= l.config.MaxToolCalls {
				return l.failWith(state, ErrAgentToolBudget)
			}
			decision.Tool = strings.TrimSpace(decision.Tool)
			if decision.Tool == "" {
				return l.failWith(state, ErrAgentInvalidDecision)
			}
			fingerprint, fingerprintErr := actionFingerprint(decision.Tool, decision.Arguments)
			if fingerprintErr != nil {
				return l.fail(state, fingerprintErr.Error())
			}
			if state.LastAction == fingerprint {
				state.RepeatStreak++
			} else {
				state.LastAction = fingerprint
				state.RepeatStreak = 1
			}
			if state.RepeatStreak > l.config.RepeatLimit {
				return l.failWith(state, ErrAgentDoomLoop)
			}
			state.Phase = PhaseExecuting
			state.DecisionSummary = publicDecisionSummary(decision.Message, "准备执行 "+decision.Tool)
			state.ToolCalls++
			state.UpdatedAt = time.Now()
			l.publish(state)
			l.record(state.ID, "run/decision", state.DecisionSummary)
			l.record(state.ID, "tool/request", decision.Tool)
			outcome, execErr := l.executor.Execute(ctx, ToolCall{RunID: state.ID, Name: decision.Tool, Arguments: decision.Arguments})
			observation := Observation{Step: state.Step, Tool: decision.Tool, Summary: outcome.Summary, Data: outcome.Data, Pending: outcome.Pending, Denied: outcome.Denied, ApprovalID: outcome.ApprovalID, Time: time.Now()}
			if execErr != nil {
				observation.Error = execErr.Error()
				state.Failures++
			}
			state.Observations = append(state.Observations, observation)
			if outcome.Pending {
				state.Status = RunWaitingApproval
				state.Phase = PhaseWaitingApproval
				state.ApprovalID = outcome.ApprovalID
				state.PendingCall = &ToolCall{RunID: state.ID, Name: decision.Tool, Arguments: decision.Arguments}
				state.Message = outcome.Summary
				state.DecisionSummary = "需要批准后继续：" + decision.Tool
				state.UpdatedAt = time.Now()
				l.publish(state)
				l.record(state.ID, "run/waiting-approval", decision.Tool)
				return state
			}
			if execErr != nil || outcome.Denied {
				state.Phase = PhaseRecovering
				state.DecisionSummary = "执行结果异常，正在诊断并重新规划"
				state.UpdatedAt = time.Now()
				l.publish(state)
			} else {
				state.Phase = PhaseVerifying
				state.DecisionSummary = "已执行 " + decision.Tool + "，正在验证结果"
				state.UpdatedAt = time.Now()
				l.publish(state)
			}
			if execErr == nil && l.validator != nil {
				validation, validationErr := l.validator.Validate(ctx, state.Goal, decision, outcome)
				if validationErr != nil || !validation.OK {
					state.Failures++
					summary := validation.Summary
					if validationErr != nil {
						summary = validationErr.Error()
					}
					state.Phase = PhaseRecovering
					state.DecisionSummary = "验证未通过，正在根据证据调整方案"
					state.Observations = append(state.Observations, Observation{Step: state.Step, Tool: decision.Tool, Summary: "执行后验证未通过", Error: summary, Time: time.Now()})
					state.UpdatedAt = time.Now()
					l.publish(state)
				} else if strings.TrimSpace(validation.Summary) != "" {
					state.DecisionSummary = publicDecisionSummary(validation.Summary, state.DecisionSummary)
					state.UpdatedAt = time.Now()
					l.publish(state)
				}
			}
			if state.Failures >= l.config.MaxFailures {
				return l.failWith(state, ErrAgentFailureBudget)
			}
		default:
			return l.failWith(state, ErrAgentInvalidDecision)
		}
	}
}

const (
	maxModelObservations       = 18
	maxModelObservationBytes   = 32 * 1024
	maxEarlierObservationHints = 10
)

// observationsForModel is XiaoYu's first context-compaction seam. Durable Run
// state keeps full receipts, while each model turn receives a bounded projection
// of older observations plus the most recent concrete evidence. Mature agent
// runtimes such as Codex/DSH separate durable event history from the model
// projection for exactly this reason: a noisy command must not consume the
// whole reasoning window forever.
func observationsForModel(items []Observation) []Observation {
	if len(items) == 0 {
		return nil
	}
	start := 0
	out := make([]Observation, 0, minIntLoop(len(items), maxModelObservations+1))
	if len(items) > maxModelObservations {
		start = len(items) - maxModelObservations
		earlier := items[:start]
		hints := make([]map[string]any, 0, minIntLoop(len(earlier), maxEarlierObservationHints))
		hintStart := 0
		if len(earlier) > maxEarlierObservationHints {
			hintStart = len(earlier) - maxEarlierObservationHints
		}
		for _, item := range earlier[hintStart:] {
			hint := map[string]any{"step": item.Step, "summary": item.Summary}
			if item.Tool != "" {
				hint["tool"] = item.Tool
			}
			if item.Error != "" {
				hint["error"] = item.Error
			}
			if item.Denied {
				hint["denied"] = true
			}
			hints = append(hints, hint)
		}
		out = append(out, Observation{
			Step:    earlier[len(earlier)-1].Step,
			Summary: fmt.Sprintf("已压缩 %d 条较早 Observation；以下保留最近摘要，完整回执仍由 Host 持有", len(earlier)),
			Data:    map[string]any{"count": len(earlier), "recent": hints},
			Time:    earlier[len(earlier)-1].Time,
		})
	}
	for _, item := range items[start:] {
		clone := item
		clone.Data = compactObservationData(item.Data)
		out = append(out, clone)
	}
	return out
}

func compactObservationData(value any) any {
	if value == nil {
		return nil
	}
	raw, err := json.Marshal(value)
	if err != nil || len(raw) <= maxModelObservationBytes {
		return value
	}
	preview := raw[:maxModelObservationBytes]
	for len(preview) > 0 && !utf8.Valid(preview) {
		preview = preview[:len(preview)-1]
	}
	text := string(preview)
	return map[string]any{
		"truncated":     true,
		"originalBytes": len(raw),
		"preview":       text,
	}
}

func minIntLoop(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func completionEvidenceGap(state RunState, tools []xiaoyuToolView) string {
	if !goalLikelyOperational(state.Goal) {
		return ""
	}
	success := false
	for _, item := range state.Observations {
		if item.Tool != "" && !item.Pending && !item.Denied && strings.TrimSpace(item.Error) == "" {
			success = true
			break
		}
	}
	if !success {
		return "目标属于操作/检查任务，但还没有成功的 Tool Observation；继续发现并执行 AGMP 能力后再完成"
	}

	byName := make(map[string]xiaoyuToolView, len(tools))
	readByCategory := map[string]bool{}
	for _, tool := range tools {
		byName[tool.Name] = tool
		if strings.EqualFold(strings.TrimSpace(tool.Risk), "read") {
			readByCategory[strings.TrimSpace(tool.Category)] = true
		}
	}
	lastActionIndex := -1
	lastActionCategory := ""
	lastActionTool := ""
	for i, item := range state.Observations {
		if item.Tool == "" || item.Pending || item.Denied || strings.TrimSpace(item.Error) != "" {
			continue
		}
		tool, ok := byName[item.Tool]
		if !ok || strings.EqualFold(strings.TrimSpace(tool.Risk), "read") {
			continue
		}
		lastActionIndex, lastActionCategory, lastActionTool = i, strings.TrimSpace(tool.Category), item.Tool
	}
	if lastActionIndex < 0 || !readByCategory[lastActionCategory] {
		return ""
	}
	for i := lastActionIndex + 1; i < len(state.Observations); i++ {
		item := state.Observations[i]
		tool, ok := byName[item.Tool]
		if !ok || item.Pending || item.Denied || strings.TrimSpace(item.Error) != "" {
			continue
		}
		if strings.EqualFold(strings.TrimSpace(tool.Risk), "read") && strings.TrimSpace(tool.Category) == lastActionCategory {
			return ""
		}
	}
	return fmt.Sprintf("%s 已产生状态变化；同类别存在读取/状态 Tool，请先验证真实结果再完成", lastActionTool)
}

func goalLikelyOperational(goal string) bool {
	value := strings.ToLower(strings.TrimSpace(goal))
	if value == "" {
		return false
	}
	markers := []string{
		"打开", "进入", "切换", "改", "修改", "设置", "配置", "安装", "卸载", "启动", "停止", "重启", "更新", "升级", "部署", "创建", "删除", "恢复", "备份", "迁移", "修复", "处理", "检查", "诊断", "查看", "读取", "搜索", "测试", "验证", "弄好", "配好", "帮我", "manage", "open", "set", "configure", "install", "start", "stop", "restart", "update", "deploy", "create", "delete", "restore", "backup", "repair", "check", "diagnose", "verify",
	}
	for _, marker := range markers {
		if strings.Contains(value, marker) {
			return true
		}
	}
	return false
}

func publicDecisionSummary(value, fallback string) string {
	value = strings.Join(strings.Fields(strings.TrimSpace(value)), " ")
	if value == "" {
		value = strings.TrimSpace(fallback)
	}
	const max = 240
	if len([]rune(value)) > max {
		value = string([]rune(value)[:max]) + "…"
	}
	return value
}

func lastObservationFailed(items []Observation) bool {
	if len(items) == 0 {
		return false
	}
	last := items[len(items)-1]
	return strings.TrimSpace(last.Error) != "" || last.Denied
}

func buildCapabilityContext(tools []xiaoyuToolView) CapabilityContext {
	result := CapabilityContext{ToolCount: len(tools)}
	categories := map[string]struct{}{}
	for _, tool := range tools {
		if value := strings.TrimSpace(tool.Category); value != "" {
			categories[value] = struct{}{}
		}
		switch strings.ToLower(strings.TrimSpace(tool.Risk)) {
		case "read":
			result.Read++
		case "operate":
			result.Operate++
		case "modify":
			result.Modify++
		default:
			result.HighRisk++
		}
	}
	result.Categories = make([]string, 0, len(categories))
	for category := range categories {
		result.Categories = append(result.Categories, category)
	}
	sort.Strings(result.Categories)
	return result
}

func actionFingerprint(tool string, args map[string]any) (string, error) {
	payload, err := json.Marshal(struct {
		Tool      string         `json:"tool"`
		Arguments map[string]any `json:"arguments,omitempty"`
	}{Tool: tool, Arguments: args})
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:]), nil
}

func (l *Loop) failWith(state RunState, err error) RunState { return l.fail(state, err.Error()) }
func (l *Loop) fail(state RunState, message string) RunState {
	state.Status = RunFailed
	state.Phase = PhaseFailed
	state.Message = strings.TrimSpace(message)
	state.DecisionSummary = state.Message
	state.UpdatedAt = time.Now()
	l.publish(state)
	l.record(state.ID, "run/failed", state.Message)
	return state
}
func (l *Loop) record(runID, kind, summary string) {
	l.trace.Append(Event{Type: kind, RunID: runID, Summary: summary})
}

func ToolsFromRegistry(registry *xiaoyucontract.Registry) func() []xiaoyuToolView {
	return func() []xiaoyuToolView {
		if registry == nil {
			return nil
		}
		specs := registry.List()
		result := make([]xiaoyuToolView, 0, len(specs))
		for _, spec := range specs {
			if !spec.XiaoYu {
				continue
			}
			result = append(result, xiaoyuToolView{Name: spec.Name, Description: spec.Description, Risk: string(spec.Risk), Category: spec.Category, Tier: toolGuidanceTier(spec.Name, spec.Category), Source: spec.Source, Parameters: cloneSchema(spec.Parameters)})
		}
		return result
	}
}

func toolGuidanceTier(name, category string) string {
	name = strings.ToLower(strings.TrimSpace(name))
	category = strings.ToLower(strings.TrimSpace(category))
	switch {
	case name == "shell.exec" || name == "process.run":
		return "fallback"
	case strings.HasPrefix(name, "fs.") || category == "filesystem" || category == "network" || category == "download" || category == "archive":
		return "general"
	case strings.HasPrefix(name, "agmp.") || strings.HasPrefix(name, "memory.") || strings.HasPrefix(name, "ui.") || category == "system" || category == "intelligence":
		return "control"
	default:
		return "domain"
	}
}

func cloneModelInputAttachments(items []ModelInputAttachment) []ModelInputAttachment {
	if len(items) == 0 {
		return nil
	}
	out := make([]ModelInputAttachment, len(items))
	for i := range items {
		out[i] = items[i]
		out[i].Data = append([]byte(nil), items[i].Data...)
	}
	return out
}

func maxZero(value int) int {
	if value < 0 {
		return 0
	}
	return value
}

func cloneSchema(value map[string]any) map[string]any {
	if value == nil {
		return nil
	}
	raw, err := json.Marshal(value)
	if err != nil {
		return nil
	}
	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil
	}
	return out
}

func (s RunState) String() string {
	return fmt.Sprintf("%s:%s step=%d tools=%d failures=%d", s.ID, s.Status, s.Step, s.ToolCalls, s.Failures)
}
