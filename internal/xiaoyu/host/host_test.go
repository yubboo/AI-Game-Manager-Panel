package host

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	xiaoyucontract "github.com/yubboo/AI-Game-Manager-Panel/internal/xiaoyu/contract"
)

func TestKernelMountsDependenciesAndRevokesTools(t *testing.T) {
	tools := xiaoyucontract.New()
	kernel := New(tools)
	provider := PluginFunc{Meta: Manifest{ID: "test.provider", Provides: []string{"demo"}}, ApplyFn: func(_ context.Context, c *Context) (DisposeFunc, error) {
		return nil, c.RegisterService("demo", "ready")
	}}
	consumer := PluginFunc{Meta: Manifest{ID: "test.consumer", Requires: []string{"demo"}, Kinds: []CapabilityKind{CapabilityTool}}, ApplyFn: func(_ context.Context, c *Context) (DisposeFunc, error) {
		value, ok := c.Service("demo")
		if !ok || value != "ready" {
			return nil, errors.New("service missing")
		}
		return nil, c.RegisterTool(xiaoyucontract.ToolSpec{Name: "demo.read", Risk: xiaoyucontract.RiskRead, XiaoYu: true}, func(context.Context, map[string]any) (xiaoyucontract.ToolExecution, error) {
			return xiaoyucontract.ToolExecution{Summary: "ok"}, nil
		})
	}}
	if err := kernel.Add(consumer); err != nil {
		t.Fatal(err)
	}
	if err := kernel.Add(provider); err != nil {
		t.Fatal(err)
	}
	if err := kernel.MountAll(context.Background()); err != nil {
		t.Fatal(err)
	}
	if _, ok := tools.Get("demo.read"); !ok {
		t.Fatal("tool not mounted")
	}
	if err := kernel.Unmount(context.Background(), "test.provider"); err != nil {
		t.Fatal(err)
	}
	if _, ok := tools.Get("demo.read"); ok {
		t.Fatal("dependent tool should be revoked")
	}
}

type scriptedBrain struct {
	decisions []Decision
	index     int
}

func (b *scriptedBrain) Next(context.Context, Frame) (Decision, error) {
	if b.index >= len(b.decisions) {
		return Decision{Kind: DecisionComplete, Message: "done"}, nil
	}
	value := b.decisions[b.index]
	b.index++
	return value, nil
}

type fakeExecutor struct {
	calls   int
	pending bool
}

func (e *fakeExecutor) Execute(_ context.Context, call ToolCall) (ToolOutcome, error) {
	e.calls++
	if e.pending {
		return ToolOutcome{Summary: "approval", Pending: true, ApprovalID: "A-1"}, nil
	}
	return ToolOutcome{Summary: "ok", Data: call.Name}, nil
}

func TestLoopCompletesAfterObservation(t *testing.T) {
	brain := &scriptedBrain{decisions: []Decision{{Kind: DecisionTool, Tool: "demo.read"}, {Kind: DecisionComplete, Message: "完成"}}}
	exec := &fakeExecutor{}
	loop := NewLoop(brain, exec, nil, nil, nil, LoopConfig{})
	result := loop.Advance(context.Background(), loop.NewRun("R-1", "test"))
	if result.Status != RunCompleted || exec.calls != 1 || len(result.Observations) != 1 {
		t.Fatalf("unexpected: %+v", result)
	}
}

func TestLoopPausesForApproval(t *testing.T) {
	brain := &scriptedBrain{decisions: []Decision{{Kind: DecisionTool, Tool: "demo.write"}}}
	loop := NewLoop(brain, &fakeExecutor{pending: true}, nil, nil, nil, LoopConfig{})
	result := loop.Advance(context.Background(), loop.NewRun("R-2", "test"))
	if result.Status != RunWaitingApproval || result.ApprovalID != "A-1" {
		t.Fatalf("unexpected: %+v", result)
	}
}

func TestLoopStopsDoomLoop(t *testing.T) {
	brain := &scriptedBrain{decisions: []Decision{
		{Kind: DecisionTool, Tool: "demo.read", Arguments: map[string]any{"x": 1}},
		{Kind: DecisionTool, Tool: "demo.read", Arguments: map[string]any{"x": 1}},
		{Kind: DecisionTool, Tool: "demo.read", Arguments: map[string]any{"x": 1}},
	}}
	loop := NewLoop(brain, &fakeExecutor{}, nil, nil, nil, LoopConfig{RepeatLimit: 2, MaxSteps: 5})
	result := loop.Advance(context.Background(), loop.NewRun("R-3", "test"))
	if result.Status != RunFailed || result.Message != ErrAgentDoomLoop.Error() {
		t.Fatalf("unexpected: %+v", result)
	}
}

type approvalExecutor struct {
	approved bool
	calls    []ToolCall
}

func (e *approvalExecutor) Execute(_ context.Context, call ToolCall) (ToolOutcome, error) {
	e.calls = append(e.calls, call)
	if !e.approved {
		return ToolOutcome{Summary: "need approval", Pending: true, ApprovalID: "A-9"}, nil
	}
	if call.ApprovalID != "A-9" {
		return ToolOutcome{}, errors.New("approval id not resumed")
	}
	return ToolOutcome{Summary: "executed"}, nil
}

func TestLoopResumesExactPendingCallAfterApproval(t *testing.T) {
	brain := &scriptedBrain{decisions: []Decision{{Kind: DecisionTool, Tool: "demo.write", Arguments: map[string]any{"path": "a"}}, {Kind: DecisionComplete, Message: "done"}}}
	exec := &approvalExecutor{}
	loop := NewLoop(brain, exec, nil, nil, nil, LoopConfig{})
	state := loop.Advance(context.Background(), loop.NewRun("R-4", "write"))
	if state.Status != RunWaitingApproval || state.PendingCall == nil {
		t.Fatalf("expected pending call: %+v", state)
	}
	exec.approved = true
	state = loop.Advance(context.Background(), state)
	if state.Status != RunCompleted || len(exec.calls) != 2 || exec.calls[1].ApprovalID != "A-9" {
		t.Fatalf("unexpected resume: %+v calls=%+v", state, exec.calls)
	}
}

type scriptedProvider struct {
	decisions []Decision
	index     int
}

func (p *scriptedProvider) Info(context.Context) BrainInfo {
	return BrainInfo{ID: "test.brain", Name: "Test Brain", Source: "test", Ready: true}
}

func (p *scriptedProvider) Next(_ context.Context, _ Frame) (Decision, error) {
	if p.index >= len(p.decisions) {
		return Decision{Kind: DecisionComplete, Message: "done"}, nil
	}
	value := p.decisions[p.index]
	p.index++
	return value, nil
}

type registryExecutor struct{ registry *xiaoyucontract.Registry }

func (e registryExecutor) Execute(ctx context.Context, call ToolCall) (ToolOutcome, error) {
	value, err := e.registry.Execute(ctx, call.Name, call.Arguments)
	return ToolOutcome{Summary: value.Summary, Data: value.Data}, err
}

func TestBrainPluginAndRunManagerAutonomousLoop(t *testing.T) {
	registry := xiaoyucontract.New()
	kernel := New(registry)
	provider := &scriptedProvider{decisions: []Decision{
		{Kind: DecisionTool, Tool: "system.echo", Arguments: map[string]any{"value": "hello"}},
		{Kind: DecisionComplete, Message: "goal complete"},
	}}
	if err := kernel.Add(BrainPlugin{Meta: Manifest{ID: "test.brain", Name: "Test Brain", Version: "1.0.0", Source: "test"}, Provider: provider}); err != nil {
		t.Fatal(err)
	}
	if err := kernel.Add(PluginFunc{Meta: Manifest{ID: "test.tools", Name: "Test Tools", Version: "1.0.0", Kinds: []CapabilityKind{CapabilityTool}}, ApplyFn: func(_ context.Context, ctx *Context) (DisposeFunc, error) {
		return nil, ctx.RegisterTool(xiaoyucontract.ToolSpec{Name: "system.echo", Description: "echo", Risk: xiaoyucontract.RiskRead, Category: "system", XiaoYu: true}, func(_ context.Context, args map[string]any) (xiaoyucontract.ToolExecution, error) {
			return xiaoyucontract.ToolExecution{Summary: "echoed", Data: args["value"]}, nil
		})
	}}); err != nil {
		t.Fatal(err)
	}
	if err := kernel.MountAll(context.Background()); err != nil {
		t.Fatal(err)
	}
	brain, ok := kernel.Brain()
	if !ok || !brain.Info(context.Background()).Ready {
		t.Fatal("primary brain was not mounted")
	}
	loop := NewLoop(brain, registryExecutor{registry: registry}, nil, ToolsFromRegistry(registry), kernel.Trace(), LoopConfig{MaxSteps: 5, MaxToolCalls: 3, MaxFailures: 2, RepeatLimit: 2})
	runs := NewRunManager()
	state, err := runs.Create("echo hello and finish")
	if err != nil {
		t.Fatal(err)
	}
	state, err = runs.Advance(context.Background(), state.ID, loop)
	if err != nil {
		t.Fatal(err)
	}
	if state.Status != RunCompleted || state.ToolCalls != 1 || len(state.Observations) != 1 {
		t.Fatalf("unexpected state: %+v", state)
	}
}

func TestRunManagerResumeUser(t *testing.T) {
	runs := NewRunManager()
	state, err := runs.Create("need user input")
	if err != nil {
		t.Fatal(err)
	}
	state.Status = RunWaitingUser
	runs.mu.Lock()
	runs.runs[state.ID] = state
	runs.mu.Unlock()
	state, err = runs.ResumeUser(state.ID, "use Cluster_1")
	if err != nil {
		t.Fatal(err)
	}
	if len(state.Observations) != 1 || state.Observations[0].Summary != "用户补充信息" {
		t.Fatalf("unexpected resumed state: %+v", state)
	}
}

func TestManifestRejectsUnsupportedAPIAndCapabilityKind(t *testing.T) {
	kernel := New(nil)
	if err := kernel.Add(PluginFunc{Meta: Manifest{ID: "test.api", APIVersion: "xiaoyu.plugin.v999"}}); !errors.Is(err, ErrPluginAPIUnsupported) {
		t.Fatalf("unsupported api error = %v", err)
	}
	if err := kernel.Add(PluginFunc{Meta: Manifest{ID: "test.kind", Kinds: []CapabilityKind{"unknown"}}}); !errors.Is(err, ErrCapabilityInvalid) {
		t.Fatalf("invalid capability error = %v", err)
	}
}

func TestKernelForgetAllowsCleanReload(t *testing.T) {
	registry := xiaoyucontract.New()
	kernel := New(registry)
	makePlugin := func(description string) Plugin {
		return PluginFunc{Meta: Manifest{ID: "external.reload", Kinds: []CapabilityKind{CapabilityTool}}, ApplyFn: func(_ context.Context, host *Context) (DisposeFunc, error) {
			err := host.RegisterTool(xiaoyucontract.ToolSpec{Name: "external.reload.ping", Description: description, Risk: xiaoyucontract.RiskRead, XiaoYu: true}, func(context.Context, map[string]any) (xiaoyucontract.ToolExecution, error) {
				return xiaoyucontract.ToolExecution{Summary: description}, nil
			})
			return nil, err
		}}
	}
	if err := kernel.Add(makePlugin("v1")); err != nil {
		t.Fatal(err)
	}
	if err := kernel.Mount(context.Background(), "external.reload"); err != nil {
		t.Fatal(err)
	}
	if err := kernel.Unmount(context.Background(), "external.reload"); err != nil {
		t.Fatal(err)
	}
	if err := kernel.Forget("external.reload"); err != nil {
		t.Fatal(err)
	}
	if _, ok := registry.Get("external.reload.ping"); ok {
		t.Fatal("tool survived forget")
	}
	if err := kernel.Add(makePlugin("v2")); err != nil {
		t.Fatal(err)
	}
	if err := kernel.Mount(context.Background(), "external.reload"); err != nil {
		t.Fatal(err)
	}
	spec, ok := registry.Get("external.reload.ping")
	if !ok || spec.Description != "v2" {
		t.Fatalf("reloaded spec = %+v ok=%v", spec, ok)
	}
}

func TestDoomLoopOnlyCountsConsecutiveIdenticalActions(t *testing.T) {
	brain := &scriptedBrain{decisions: []Decision{
		{Kind: DecisionTool, Tool: "demo.read", Arguments: map[string]any{"x": 1}},
		{Kind: DecisionTool, Tool: "demo.other", Arguments: map[string]any{"x": 2}},
		{Kind: DecisionTool, Tool: "demo.read", Arguments: map[string]any{"x": 1}},
		{Kind: DecisionTool, Tool: "demo.other", Arguments: map[string]any{"x": 2}},
		{Kind: DecisionComplete, Message: "polling made progress"},
	}}
	loop := NewLoop(brain, &fakeExecutor{}, nil, nil, nil, LoopConfig{RepeatLimit: 1, MaxSteps: 8, MaxToolCalls: 8})
	result := loop.Advance(context.Background(), loop.NewRun("R-poll", "alternate polling"))
	if result.Status != RunCompleted {
		t.Fatalf("legitimate repeated polling was mistaken for doom loop: %+v", result)
	}
}

type frameCaptureBrain struct{ frame Frame }

func (b *frameCaptureBrain) Next(_ context.Context, frame Frame) (Decision, error) {
	b.frame = frame
	return Decision{Kind: DecisionComplete, Message: "captured"}, nil
}

func TestBrainFrameIncludesToolSchemaAndRemainingBudget(t *testing.T) {
	registry := xiaoyucontract.New()
	if err := registry.RegisterHandler(xiaoyucontract.ToolSpec{Name: "demo.schema", Description: "schema", Risk: xiaoyucontract.RiskRead, Category: "demo", XiaoYu: true, Source: "test", Parameters: map[string]any{"type": "object", "properties": map[string]any{"name": map[string]any{"type": "string"}}, "required": []string{"name"}}}, func(context.Context, map[string]any) (xiaoyucontract.ToolExecution, error) {
		return xiaoyucontract.ToolExecution{Summary: "ok"}, nil
	}); err != nil {
		t.Fatal(err)
	}
	brain := &frameCaptureBrain{}
	loop := NewLoop(brain, registryExecutor{registry: registry}, nil, ToolsFromRegistry(registry), nil, LoopConfig{MaxSteps: 4, MaxToolCalls: 3, MaxFailures: 2, RepeatLimit: 2})
	state := loop.Advance(context.Background(), loop.NewRun("R-frame", "inspect"))
	if state.Status != RunCompleted {
		t.Fatalf("unexpected state: %+v", state)
	}
	if len(brain.frame.Tools) != 1 || brain.frame.Tools[0].Parameters["type"] != "object" || brain.frame.Tools[0].Source != "test" {
		t.Fatalf("tool contract missing from frame: %+v", brain.frame.Tools)
	}
	if brain.frame.Budget.StepsRemaining != 3 || brain.frame.Budget.ToolCallsRemaining != 3 || brain.frame.Budget.FailuresRemaining != 2 {
		t.Fatalf("unexpected budget: %+v", brain.frame.Budget)
	}
}

func TestRunManagerPrunesOldTerminalHistory(t *testing.T) {
	runs := NewRunManager(2)
	loop := NewLoop(&scriptedBrain{}, &fakeExecutor{}, nil, nil, nil, LoopConfig{})
	for i := 0; i < 3; i++ {
		state, err := runs.Create("goal")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := runs.Advance(context.Background(), state.ID, loop); err != nil {
			t.Fatal(err)
		}
	}
	if got := len(runs.List()); got != 2 {
		t.Fatalf("run history=%d want=2", got)
	}
}

type deniedThenCompleteExecutor struct{ calls int }

func (e *deniedThenCompleteExecutor) Execute(_ context.Context, _ ToolCall) (ToolOutcome, error) {
	e.calls++
	return ToolOutcome{Summary: "denied by user", Denied: true}, nil
}

func TestDeniedToolIsObservationNotFailureBudget(t *testing.T) {
	brain := &scriptedBrain{decisions: []Decision{{Kind: DecisionTool, Tool: "demo.write"}, {Kind: DecisionComplete, Message: "used another plan"}}}
	loop := NewLoop(brain, &deniedThenCompleteExecutor{}, xiaoyuOutcomeValidatorForTest{}, nil, nil, LoopConfig{MaxFailures: 1, MaxSteps: 4})
	result := loop.Advance(context.Background(), loop.NewRun("R-denied", "adapt"))
	if result.Status != RunCompleted || result.Failures != 0 || len(result.Observations) != 1 || !result.Observations[0].Denied {
		t.Fatalf("denial should be replannable observation: %+v", result)
	}
}

type xiaoyuOutcomeValidatorForTest struct{}

func (xiaoyuOutcomeValidatorForTest) Validate(_ context.Context, _ string, _ Decision, outcome ToolOutcome) (Validation, error) {
	if outcome.Denied {
		return Validation{OK: true, Summary: "replan"}, nil
	}
	return Validation{OK: true}, nil
}

func TestLoopPublishesAgentKernelPhases(t *testing.T) {
	brain := &scriptedBrain{decisions: []Decision{
		{Kind: DecisionTool, Tool: "demo.read", Message: "先读取当前状态，再判断是否需要下一步。"},
		{Kind: DecisionComplete, Message: "目标已验证完成。"},
	}}
	exec := &fakeExecutor{}
	loop := NewLoop(brain, exec, nil, func() []xiaoyuToolView {
		return []xiaoyuToolView{{Name: "demo.read", Category: "demo", Risk: "read"}}
	}, nil, LoopConfig{})
	var seen []RunState
	loop.WithStateSink(func(state RunState) { seen = append(seen, state) })
	result := loop.Advance(context.Background(), loop.NewRun("R-kernel", "inspect"))
	if result.Status != RunCompleted || result.Phase != PhaseCompleted {
		t.Fatalf("unexpected final state: %+v", result)
	}
	foundExecuting := false
	for _, state := range seen {
		if state.Phase == PhaseExecuting && state.DecisionSummary == "先读取当前状态，再判断是否需要下一步。" {
			foundExecuting = true
			break
		}
	}
	if !foundExecuting {
		t.Fatalf("expected public execution summary in live states: %#v", seen)
	}
}

func TestCapabilityContextIsDeterministicAndRiskAware(t *testing.T) {
	got := buildCapabilityContext([]xiaoyuToolView{
		{Name: "a", Category: "settings", Risk: "operate"},
		{Name: "b", Category: "system", Risk: "read"},
		{Name: "c", Category: "settings", Risk: "modify"},
		{Name: "d", Category: "dst", Risk: "system"},
	})
	if got.ToolCount != 4 || got.Read != 1 || got.Operate != 1 || got.Modify != 1 || got.HighRisk != 1 {
		t.Fatalf("unexpected capability context: %+v", got)
	}
	want := []string{"dst", "settings", "system"}
	if len(got.Categories) != len(want) {
		t.Fatalf("unexpected categories: %#v", got.Categories)
	}
	for i := range want {
		if got.Categories[i] != want[i] {
			t.Fatalf("unexpected category order: %#v", got.Categories)
		}
	}
}

func TestObservationsForModelCompactsOldHistory(t *testing.T) {
	items := make([]Observation, 0, 30)
	for i := 0; i < 30; i++ {
		items = append(items, Observation{Step: i + 1, Tool: "demo.read", Summary: fmt.Sprintf("observation-%d", i+1), Data: map[string]any{"i": i}})
	}
	got := observationsForModel(items)
	if len(got) != maxModelObservations+1 {
		t.Fatalf("expected one compact summary plus %d recent observations, got %d", maxModelObservations, len(got))
	}
	if !strings.Contains(got[0].Summary, "已压缩") {
		t.Fatalf("expected compacted history marker, got %+v", got[0])
	}
	if got[len(got)-1].Summary != "observation-30" {
		t.Fatalf("most recent evidence must be preserved, got %+v", got[len(got)-1])
	}
}

func TestCompactObservationDataBoundsLargePayload(t *testing.T) {
	large := map[string]any{"output": strings.Repeat("x", maxModelObservationBytes*2)}
	got, ok := compactObservationData(large).(map[string]any)
	if !ok || got["truncated"] != true {
		t.Fatalf("expected bounded projection, got %#v", got)
	}
	preview, _ := got["preview"].(string)
	if len(preview) > maxModelObservationBytes {
		t.Fatalf("preview exceeded model observation budget: %d", len(preview))
	}
}
