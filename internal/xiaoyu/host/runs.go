package host

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

var (
	ErrRunNotFound   = errors.New("xiaoyu run not found")
	ErrRunBusy       = errors.New("xiaoyu run is already advancing")
	ErrRunHumanOwned = errors.New("xiaoyu run is controlled by a human")
	ErrRunNotPaused  = errors.New("xiaoyu run is not paused")
	ErrRunTerminal   = errors.New("xiaoyu run is terminal and cannot be resumed")
)

type runInterrupt struct {
	status RunStatus
	owner  ControlOwner
	by     string
	reason string
	steer  string
}

type RunManager struct {
	mu         sync.RWMutex
	seq        atomic.Uint64
	runs       map[string]RunState
	active     map[string]bool
	cancels    map[string]context.CancelFunc
	interrupts map[string]runInterrupt
	maxHistory int
}

func NewRunManager(limit ...int) *RunManager {
	maxHistory := 100
	if len(limit) > 0 && limit[0] > 0 {
		maxHistory = limit[0]
	}
	return &RunManager{
		runs: make(map[string]RunState), active: make(map[string]bool), cancels: make(map[string]context.CancelFunc),
		interrupts: make(map[string]runInterrupt), maxHistory: maxHistory,
	}
}

func (m *RunManager) Create(goal string) (RunState, error) {
	return m.CreateFor(goal, "", "")
}

// CreateFor binds a server-owned Run to the authenticated human that created
// it. This becomes the access boundary for multi-user Web deployments; owner
// and administrator roles may still supervise every Run at the Application layer.
func (m *RunManager) CreateFor(goal, initiatorID, initiatorName string) (RunState, error) {
	return m.CreateForContext(goal, initiatorID, initiatorName, RunContext{})
}

func (m *RunManager) CreateForContext(goal, initiatorID, initiatorName string, runContext RunContext) (RunState, error) {
	goal = strings.TrimSpace(goal)
	if goal == "" {
		return RunState{}, errors.New("xiaoyu run goal is required")
	}
	now := time.Now()
	id := fmt.Sprintf("XYRUN-%d-%06d", now.UnixMilli(), m.seq.Add(1))
	state := RunState{
		ID: id, Goal: goal, Status: RunCreated, ControlOwner: ControlXiaoYu,
		InitiatorID: strings.TrimSpace(initiatorID), InitiatorName: strings.TrimSpace(initiatorName),
		// TaskID is always server-owned. Never trust a client-provided task scope,
		// otherwise a caller could point Intelligence retrieval at another Run.
		Context: RunContext{SessionID: strings.TrimSpace(runContext.SessionID), TaskID: id, GameID: strings.TrimSpace(runContext.GameID), ServerID: strings.TrimSpace(runContext.ServerID), InstanceID: strings.TrimSpace(runContext.InstanceID), UIRoute: strings.TrimSpace(runContext.UIRoute)},
		Phase:   PhaseUnderstanding, CreatedAt: now, UpdatedAt: now,
	}
	m.mu.Lock()
	m.runs[id] = cloneRunState(state)
	m.pruneLocked()
	m.mu.Unlock()
	return cloneRunState(state), nil
}

func (m *RunManager) SetAttachments(id string, attachments []ModelInputAttachment) (RunState, error) {
	id = strings.TrimSpace(id)
	m.mu.Lock()
	defer m.mu.Unlock()
	state, ok := m.runs[id]
	if !ok {
		return RunState{}, fmt.Errorf("%w: %s", ErrRunNotFound, id)
	}
	if m.active[id] || state.Status != RunCreated {
		return cloneRunState(state), ErrRunBusy
	}
	state.Attachments = cloneModelInputAttachments(attachments)
	state.UpdatedAt = time.Now()
	m.runs[id] = cloneRunState(state)
	return cloneRunState(state), nil
}

func (m *RunManager) AppendAttachments(id string, attachments []ModelInputAttachment) (RunState, error) {
	id = strings.TrimSpace(id)
	m.mu.Lock()
	defer m.mu.Unlock()
	state, ok := m.runs[id]
	if !ok {
		return RunState{}, fmt.Errorf("%w: %s", ErrRunNotFound, id)
	}
	switch state.Status {
	case RunCompleted, RunFailed, RunCancelled:
		return cloneRunState(state), ErrRunTerminal
	}
	seen := map[string]struct{}{}
	merged := cloneModelInputAttachments(state.Attachments)
	for _, item := range merged {
		seen[item.ID] = struct{}{}
	}
	for _, item := range attachments {
		if _, ok := seen[item.ID]; ok {
			continue
		}
		seen[item.ID] = struct{}{}
		merged = append(merged, item)
	}
	state.Attachments = cloneModelInputAttachments(merged)
	state.UpdatedAt = time.Now()
	m.runs[id] = cloneRunState(state)
	return cloneRunState(state), nil
}

func (m *RunManager) Get(id string) (RunState, bool) {
	m.mu.RLock()
	state, ok := m.runs[strings.TrimSpace(id)]
	m.mu.RUnlock()
	return cloneRunState(state), ok
}

func (m *RunManager) List() []RunState {
	m.mu.RLock()
	result := make([]RunState, 0, len(m.runs))
	for _, state := range m.runs {
		result = append(result, cloneRunState(state))
	}
	m.mu.RUnlock()
	sort.Slice(result, func(i, j int) bool { return result[i].CreatedAt.After(result[j].CreatedAt) })
	return result
}

// ThreadContext returns a bounded, server-owned conversational history for one
// workbench session. It intentionally keeps only compact execution receipts and
// excludes pending approval arguments, credentials and arbitrary browser text.
func (m *RunManager) ThreadContext(sessionID, currentRunID string, limit int) ThreadContext {
	sessionID = strings.TrimSpace(sessionID)
	currentRunID = strings.TrimSpace(currentRunID)
	if sessionID == "" {
		return ThreadContext{}
	}
	if limit <= 0 || limit > 12 {
		limit = 6
	}
	m.mu.RLock()
	runs := make([]RunState, 0, len(m.runs))
	for _, state := range m.runs {
		if state.ID == currentRunID || strings.TrimSpace(state.Context.SessionID) != sessionID {
			continue
		}
		runs = append(runs, cloneRunState(state))
	}
	m.mu.RUnlock()
	sort.Slice(runs, func(i, j int) bool { return runs[i].CreatedAt.After(runs[j].CreatedAt) })
	if len(runs) > limit {
		runs = runs[:limit]
	}
	turns := make([]ThreadTurn, 0, len(runs))
	for _, run := range runs {
		turn := ThreadTurn{RunID: run.ID, Goal: run.Goal, Status: run.Status, Message: run.Message, Actions: []ThreadAction{}}
		start := len(run.Observations) - 8
		if start < 0 {
			start = 0
		}
		for _, observation := range run.Observations[start:] {
			if observation.Tool == "" {
				continue
			}
			turn.Actions = append(turn.Actions, ThreadAction{Tool: observation.Tool, Summary: observation.Summary, Data: compactThreadData(observation.Data), Error: safeThreadText(observation.Error, 240), Undo: threadUndoHint(observation)})
		}
		turns = append(turns, turn)
	}
	return ThreadContext{RecentTurns: turns}
}

func threadUndoHint(observation Observation) *ThreadUndoHint {
	if observation.Pending || observation.Denied || strings.TrimSpace(observation.Error) != "" {
		return nil
	}
	// Reversible domain operations expose the previous value in their normal
	// structured receipt. ThreadContext converts that receipt into an explicit
	// undo hint so short references such as “改回去” do not depend on guesswork.
	switch observation.Tool {
	case "settings.theme.set":
		root, ok := observation.Data.(map[string]any)
		if !ok {
			return nil
		}
		data, _ := root["data"].(map[string]any)
		previous, _ := data["previousTheme"].(string)
		previous = strings.TrimSpace(previous)
		if previous == "light" || previous == "dark" || previous == "system" {
			return &ThreadUndoHint{Tool: "settings.theme.set", Arguments: map[string]any{"theme": previous}, Summary: "恢复修改前的主题"}
		}
	}
	return nil
}

func compactThreadData(value any) any {
	// Cross-run context is deliberately conservative. Only tiny scalar maps are
	// retained; large/nested receipts stay in the originating Run and Trace.
	mapValue, ok := value.(map[string]any)
	if !ok {
		return nil
	}
	out := map[string]any{}
	for key, raw := range mapValue {
		if len(out) >= 12 {
			break
		}
		key = safeThreadText(key, 80)
		if key == "" {
			continue
		}
		switch v := raw.(type) {
		case string:
			out[key] = safeThreadText(v, 240)
		case bool, float64, float32, int, int32, int64, uint, uint32, uint64:
			out[key] = v
		case map[string]any:
			inner := map[string]any{}
			for innerKey, innerRaw := range v {
				if len(inner) >= 8 {
					break
				}
				switch innerValue := innerRaw.(type) {
				case string:
					inner[innerKey] = safeThreadText(innerValue, 160)
				case bool, float64, float32, int, int32, int64, uint, uint32, uint64:
					inner[innerKey] = innerValue
				}
			}
			if len(inner) > 0 {
				out[key] = inner
			}
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func safeThreadText(value string, max int) string {
	value = strings.TrimSpace(value)
	if max <= 0 || len(value) <= max {
		return value
	}
	return value[:max]
}

// Advance serializes execution per Run. Run lifetime belongs to AGMP Host, not
// to a browser/WebView request. Callers should pass Application.Context() for
// autonomous work so client disconnects only detach the UI, never the Run.
func (m *RunManager) Advance(parent context.Context, id string, loop *Loop) (RunState, error) {
	state, ctx, cancel, err := m.reserve(parent, id, loop)
	if err != nil {
		return state, err
	}
	next := loop.Advance(ctx, cloneRunState(state))
	return m.finish(id, next, cancel), nil
}

// AdvanceAsync detaches autonomous execution from the client transport. The
// returned snapshot is already marked running; completion is observed through
// Trace/Event Stream or Get(id).
func (m *RunManager) AdvanceAsync(parent context.Context, id string, loop *Loop) (RunState, error) {
	state, ctx, cancel, err := m.reserve(parent, id, loop)
	if err != nil {
		return state, err
	}
	// Keep the original suspended status for the actual Loop. The public
	// snapshot may say running, but waiting_approval/waiting_user semantics must
	// not be erased before the brain/executor resumes the exact pending step.
	initial := cloneRunState(state)
	display := cloneRunState(state)
	if display.Status != RunCreated && display.Status != RunRunning {
		display.ResumeStatus = display.Status
	}
	display.Status = RunRunning
	display.UpdatedAt = time.Now()
	m.mu.Lock()
	m.runs[id] = cloneRunState(display)
	m.mu.Unlock()
	loop.WithStateSink(func(next RunState) {
		m.publishActive(id, next)
	})
	go func(initial RunState) {
		next := loop.Advance(ctx, cloneRunState(initial))
		m.finish(id, next, cancel)
	}(initial)
	return cloneRunState(display), nil
}

func (m *RunManager) publishActive(id string, next RunState) {
	id = strings.TrimSpace(id)
	if id == "" {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.active[id] {
		return
	}
	if _, interrupted := m.interrupts[id]; interrupted {
		return
	}
	current, ok := m.runs[id]
	if !ok {
		return
	}
	// The advancing Loop owns semantic progress. Access-control ownership and
	// initiator identity remain server-owned from the RunManager snapshot.
	next.ID = current.ID
	next.InitiatorID = current.InitiatorID
	next.InitiatorName = current.InitiatorName
	next.ControlOwner = current.ControlOwner
	m.runs[id] = cloneRunState(next)
}

func (m *RunManager) reserve(parent context.Context, id string, loop *Loop) (RunState, context.Context, context.CancelFunc, error) {
	if loop == nil {
		return RunState{}, nil, nil, errors.New("xiaoyu loop is unavailable")
	}
	id = strings.TrimSpace(id)
	m.mu.Lock()
	state, ok := m.runs[id]
	if !ok {
		m.mu.Unlock()
		return RunState{}, nil, nil, fmt.Errorf("%w: %s", ErrRunNotFound, id)
	}
	if m.active[id] {
		m.mu.Unlock()
		return cloneRunState(state), nil, nil, ErrRunBusy
	}
	switch state.Status {
	case RunCompleted, RunFailed, RunCancelled:
		m.mu.Unlock()
		return cloneRunState(state), nil, nil, fmt.Errorf("%w: %s", ErrRunTerminal, state.Status)
	}
	if state.Status == RunPaused || state.ControlOwner == ControlHuman {
		m.mu.Unlock()
		return cloneRunState(state), nil, nil, ErrRunHumanOwned
	}
	if parent == nil {
		parent = context.Background()
	}
	ctx, cancel := context.WithCancel(parent)
	m.active[id] = true
	m.cancels[id] = cancel
	m.mu.Unlock()
	return cloneRunState(state), ctx, cancel, nil
}

func (m *RunManager) finish(id string, next RunState, cancel context.CancelFunc) RunState {
	if cancel != nil {
		cancel()
	}
	m.mu.Lock()
	if interrupt, exists := m.interrupts[id]; exists {
		current := m.runs[id]
		mergeRunProgress(&current, next)
		switch interrupt.status {
		case RunCreated:
			current.Status = RunCreated
			current.Phase = PhasePlanning
			current.ControlOwner = ControlXiaoYu
			current.Message = ""
			current.ApprovalID = ""
			current.PendingCall = nil
			current.ResumeStatus = ""
			current.PausedBy = ""
			current.PauseReason = ""
			if strings.TrimSpace(interrupt.steer) != "" {
				current.Observations = append(current.Observations, Observation{Step: current.Step, Summary: "用户实时纠正", Data: map[string]any{"message": interrupt.steer}, Time: time.Now()})
			}
			current.DecisionSummary = "收到用户实时纠正，正在基于最新指令重新规划"
			current.UpdatedAt = time.Now()
			next = current
		case RunPaused:
			resume := current.ResumeStatus
			if resume == "" || resume == RunPaused || resume == RunCancelled || resume == RunCompleted || resume == RunFailed || resume == RunRunning {
				resume = RunCreated
			}
			current.ResumeStatus = resume
			current.Status = RunPaused
			current.ControlOwner = interrupt.owner
			current.PausedBy = interrupt.by
			current.PauseReason = interrupt.reason
			current.Message = interrupt.reason
			current.UpdatedAt = time.Now()
			next = current
		case RunCancelled:
			current.Status = RunCancelled
			current.Message = interrupt.reason
			current.UpdatedAt = time.Now()
			next = current
		}
		delete(m.interrupts, id)
	}
	m.runs[id] = cloneRunState(next)
	delete(m.active, id)
	delete(m.cancels, id)
	m.pruneLocked()
	m.mu.Unlock()
	return cloneRunState(next)
}

func (m *RunManager) ResumeUser(id, message string) (RunState, error) {
	id = strings.TrimSpace(id)
	message = strings.TrimSpace(message)
	if message == "" {
		return RunState{}, errors.New("xiaoyu user continuation is empty")
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	state, ok := m.runs[id]
	if !ok {
		return RunState{}, fmt.Errorf("%w: %s", ErrRunNotFound, id)
	}
	if state.Status != RunWaitingUser {
		return cloneRunState(state), fmt.Errorf("xiaoyu run is not waiting for user input: %s", state.Status)
	}
	state.Observations = append(state.Observations, Observation{Step: state.Step, Summary: "用户补充信息", Data: map[string]any{"message": message}, Time: time.Now()})
	state.Message = ""
	state.Phase = PhasePlanning
	state.DecisionSummary = "已收到补充信息，正在继续规划"
	state.UpdatedAt = time.Now()
	m.runs[id] = state
	return cloneRunState(state), nil
}

// Steer injects a new human instruction into an existing non-terminal Run. If
// a Brain turn is currently active, it is cancelled at the Host boundary and
// restarted from the preserved observations with the new instruction appended.
func (m *RunManager) Steer(id, message, by string) (RunState, error) {
	id = strings.TrimSpace(id)
	message = strings.TrimSpace(message)
	by = strings.TrimSpace(by)
	if message == "" {
		return RunState{}, errors.New("xiaoyu steering message is empty")
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	state, ok := m.runs[id]
	if !ok {
		return RunState{}, fmt.Errorf("%w: %s", ErrRunNotFound, id)
	}
	switch state.Status {
	case RunCompleted, RunFailed, RunCancelled:
		return cloneRunState(state), fmt.Errorf("xiaoyu terminal run cannot be steered: %s", state.Status)
	}
	if state.ControlOwner == ControlHuman {
		return cloneRunState(state), ErrRunHumanOwned
	}
	if m.active[id] {
		m.interrupts[id] = runInterrupt{status: RunCreated, owner: ControlXiaoYu, by: by, reason: "用户实时纠正", steer: message}
		if cancel := m.cancels[id]; cancel != nil {
			cancel()
		}
		state.DecisionSummary = "收到用户实时纠正，正在停止当前推理并重新规划"
		state.UpdatedAt = time.Now()
		m.runs[id] = state
		return cloneRunState(state), nil
	}
	state.Observations = append(state.Observations, Observation{Step: state.Step, Summary: "用户实时纠正", Data: map[string]any{"message": message}, Time: time.Now()})
	state.Status = RunCreated
	state.Phase = PhasePlanning
	state.Message = ""
	state.ApprovalID = ""
	state.PendingCall = nil
	state.ControlOwner = ControlXiaoYu
	state.DecisionSummary = "收到用户实时纠正，正在基于最新指令重新规划"
	state.UpdatedAt = time.Now()
	m.runs[id] = state
	return cloneRunState(state), nil
}

func (m *RunManager) Pause(id, by, reason string) (RunState, error) {
	return m.pause(id, ControlShared, by, reason)
}

// Takeover is the hard human override. Human ownership always wins over XiaoYu
// and cancels the currently advancing turn before the Run can continue.
func (m *RunManager) Takeover(id, by, reason string) (RunState, error) {
	if strings.TrimSpace(reason) == "" {
		reason = "用户已接管当前任务"
	}
	return m.pause(id, ControlHuman, by, reason)
}

func (m *RunManager) pause(id string, owner ControlOwner, by, reason string) (RunState, error) {
	id = strings.TrimSpace(id)
	by = strings.TrimSpace(by)
	reason = strings.TrimSpace(reason)
	if reason == "" {
		reason = "用户暂停当前任务"
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	state, ok := m.runs[id]
	if !ok {
		return RunState{}, fmt.Errorf("%w: %s", ErrRunNotFound, id)
	}
	if state.Status == RunCompleted || state.Status == RunFailed || state.Status == RunCancelled {
		return cloneRunState(state), nil
	}
	if m.active[id] {
		m.interrupts[id] = runInterrupt{status: RunPaused, owner: owner, by: by, reason: reason}
		if cancel := m.cancels[id]; cancel != nil {
			cancel()
		}
	}
	resume := state.Status
	if resume == RunPaused {
		resume = state.ResumeStatus
	}
	if resume == "" || resume == RunRunning {
		resume = RunCreated
	}
	state.ResumeStatus = resume
	state.Status = RunPaused
	state.Phase = PhasePaused
	state.DecisionSummary = reason
	state.ControlOwner = owner
	state.PausedBy = by
	state.PauseReason = reason
	state.Message = reason
	state.UpdatedAt = time.Now()
	m.runs[id] = state
	return cloneRunState(state), nil
}

// ReleaseToXiaoYu returns control to the AI brain. It never advances the Run by
// itself; the caller explicitly starts the next server-owned Advance.
func (m *RunManager) ReleaseToXiaoYu(id string) (RunState, error) {
	id = strings.TrimSpace(id)
	m.mu.Lock()
	defer m.mu.Unlock()
	state, ok := m.runs[id]
	if !ok {
		return RunState{}, fmt.Errorf("%w: %s", ErrRunNotFound, id)
	}
	if state.Status != RunPaused {
		return cloneRunState(state), fmt.Errorf("%w: %s", ErrRunNotPaused, state.Status)
	}
	status := state.ResumeStatus
	if status == "" || status == RunRunning || status == RunPaused {
		status = RunCreated
	}
	state.Status = status
	state.Phase = PhasePlanning
	state.DecisionSummary = "控制权已交还小鱼，准备继续"
	state.ControlOwner = ControlXiaoYu
	state.PausedBy = ""
	state.PauseReason = ""
	state.ResumeStatus = ""
	state.Message = ""
	state.UpdatedAt = time.Now()
	m.runs[id] = state
	return cloneRunState(state), nil
}

func (m *RunManager) Cancel(id string) (RunState, error) {
	id = strings.TrimSpace(id)
	m.mu.Lock()
	defer m.mu.Unlock()
	state, ok := m.runs[id]
	if !ok {
		return RunState{}, fmt.Errorf("%w: %s", ErrRunNotFound, id)
	}
	if state.Status == RunCompleted || state.Status == RunFailed || state.Status == RunCancelled {
		return cloneRunState(state), nil
	}
	if m.active[id] {
		m.interrupts[id] = runInterrupt{status: RunCancelled, owner: state.ControlOwner, reason: "用户取消"}
		if cancel := m.cancels[id]; cancel != nil {
			cancel()
		}
	}
	state.Status = RunCancelled
	state.Phase = PhaseCancelled
	state.Message = "用户取消"
	state.DecisionSummary = "任务已取消"
	state.UpdatedAt = time.Now()
	m.runs[id] = state
	return cloneRunState(state), nil
}

func cloneRunState(state RunState) RunState {
	out := state
	out.Attachments = cloneModelInputAttachments(state.Attachments)
	out.Observations = append([]Observation(nil), state.Observations...)
	if state.PendingCall != nil {
		call := *state.PendingCall
		if state.PendingCall.Arguments != nil {
			call.Arguments = make(map[string]any, len(state.PendingCall.Arguments))
			for key, value := range state.PendingCall.Arguments {
				call.Arguments[key] = value
			}
		}
		out.PendingCall = &call
	}
	return out
}

func mergeRunProgress(current *RunState, next RunState) {
	if current == nil {
		return
	}
	if next.Step > current.Step {
		current.Step = next.Step
	}
	if next.ToolCalls > current.ToolCalls {
		current.ToolCalls = next.ToolCalls
	}
	if next.Failures > current.Failures {
		current.Failures = next.Failures
	}
	if next.Phase != "" {
		current.Phase = next.Phase
	}
	if strings.TrimSpace(next.DecisionSummary) != "" {
		current.DecisionSummary = next.DecisionSummary
	}
	if len(next.Observations) > len(current.Observations) {
		current.Observations = append([]Observation(nil), next.Observations...)
	}
	if next.PendingCall != nil {
		current.PendingCall = cloneRunState(next).PendingCall
		current.ApprovalID = next.ApprovalID
	}
}

func (m *RunManager) pruneLocked() {
	if m.maxHistory <= 0 || len(m.runs) <= m.maxHistory {
		return
	}
	type terminalRun struct {
		id string
		at time.Time
	}
	candidates := make([]terminalRun, 0, len(m.runs))
	for id, state := range m.runs {
		if m.active[id] {
			continue
		}
		switch state.Status {
		case RunCompleted, RunFailed, RunCancelled:
			candidates = append(candidates, terminalRun{id: id, at: state.UpdatedAt})
		}
	}
	sort.Slice(candidates, func(i, j int) bool { return candidates[i].at.Before(candidates[j].at) })
	for _, item := range candidates {
		if len(m.runs) <= m.maxHistory {
			break
		}
		delete(m.runs, item.id)
		delete(m.cancels, item.id)
		delete(m.interrupts, item.id)
	}
}

func (m *RunManager) ActiveCount() int {
	m.mu.RLock()
	count := len(m.active)
	m.mu.RUnlock()
	return count
}

func (m *RunManager) IsActive(id string) bool {
	m.mu.RLock()
	active := m.active[strings.TrimSpace(id)]
	m.mu.RUnlock()
	return active
}
