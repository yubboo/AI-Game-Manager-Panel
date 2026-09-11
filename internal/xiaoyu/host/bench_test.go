package host

import (
	"context"
	"errors"
	"reflect"
	"testing"
)

type agentBenchExecutor struct {
	calls    []ToolCall
	failTool string
	pending  bool
	approved bool
}

func (e *agentBenchExecutor) Execute(_ context.Context, call ToolCall) (ToolOutcome, error) {
	e.calls = append(e.calls, call)
	if e.pending && !e.approved {
		return ToolOutcome{Summary: "需要用户批准", Pending: true, ApprovalID: "bench-approval"}, nil
	}
	if e.pending && e.approved && call.ApprovalID != "bench-approval" {
		return ToolOutcome{}, errors.New("approved call was not resumed with the original approval id")
	}
	if call.Name == e.failTool {
		return ToolOutcome{Summary: "专业路径失败"}, errors.New("simulated domain failure")
	}
	return ToolOutcome{Summary: "执行成功", Data: map[string]any{"tool": call.Name}}, nil
}

func TestAgentBenchRecoversFromDomainFailureWithFallback(t *testing.T) {
	brain := &scriptedBrain{decisions: []Decision{
		{Kind: DecisionTool, Tool: "environment.remove_runtime", Arguments: map[string]any{"id": "java-21"}},
		{Kind: DecisionTool, Tool: "shell.exec", Arguments: map[string]any{"command": "repair-runtime"}},
		{Kind: DecisionTool, Tool: "environment.catalog"},
		{Kind: DecisionComplete, Message: "恢复完成"},
	}}
	exec := &agentBenchExecutor{failTool: "environment.remove_runtime"}
	loop := NewLoop(brain, exec, nil, func() []xiaoyuToolView {
		return []xiaoyuToolView{
			{Name: "environment.remove_runtime", Category: "environment", Risk: "modify", Tier: "domain"},
			{Name: "environment.catalog", Category: "environment", Risk: "read", Tier: "domain"},
			{Name: "shell.exec", Category: "system", Risk: "system", Tier: "fallback"},
		}
	}, nil, LoopConfig{MaxSteps: 10, MaxToolCalls: 6, MaxFailures: 3, RepeatLimit: 2})

	result := loop.Advance(context.Background(), loop.NewRun("bench-recovery", "修复并验证 Java 运行环境"))
	if result.Status != RunCompleted || result.Message != "恢复完成" {
		t.Fatalf("bench recovery did not complete: %+v", result)
	}
	got := make([]string, 0, len(exec.calls))
	for _, call := range exec.calls {
		got = append(got, call.Name)
	}
	want := []string{"environment.remove_runtime", "shell.exec", "environment.catalog"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected recovery path: got=%v want=%v", got, want)
	}
	if result.Failures != 1 {
		t.Fatalf("expected exactly one recoverable failure, got %d", result.Failures)
	}
}

func TestAgentBenchBlocksPrematureCompletionUntilVerified(t *testing.T) {
	brain := &scriptedBrain{decisions: []Decision{
		{Kind: DecisionTool, Tool: "settings.theme.set", Arguments: map[string]any{"theme": "light"}},
		{Kind: DecisionComplete, Message: "不能现在完成"},
		{Kind: DecisionTool, Tool: "settings.get"},
		{Kind: DecisionComplete, Message: "验证完成"},
	}}
	exec := &agentBenchExecutor{}
	loop := NewLoop(brain, exec, nil, func() []xiaoyuToolView {
		return []xiaoyuToolView{
			{Name: "settings.theme.set", Category: "settings", Risk: "modify", Tier: "domain"},
			{Name: "settings.get", Category: "settings", Risk: "read", Tier: "domain"},
		}
	}, nil, LoopConfig{MaxSteps: 8, MaxToolCalls: 4, MaxFailures: 2, RepeatLimit: 2})

	result := loop.Advance(context.Background(), loop.NewRun("bench-verify", "把主题改成浅色并确认生效"))
	if result.Status != RunCompleted || result.Message != "验证完成" {
		t.Fatalf("bench verification failed: %+v", result)
	}
	if len(exec.calls) != 2 || exec.calls[0].Name != "settings.theme.set" || exec.calls[1].Name != "settings.get" {
		t.Fatalf("mutation was not followed by read-back verification: %+v", exec.calls)
	}
	guarded := false
	for _, observation := range result.Observations {
		if observation.Summary == "完成判定暂缓" {
			guarded = true
			break
		}
	}
	if !guarded {
		t.Fatal("completion evidence guard did not block premature completion")
	}
}

func TestAgentBenchApprovalResumesExactToolCall(t *testing.T) {
	brain := &scriptedBrain{decisions: []Decision{
		{Kind: DecisionTool, Tool: "fs.remove", Arguments: map[string]any{"path": "old-cache"}},
		{Kind: DecisionComplete, Message: "清理完成"},
	}}
	exec := &agentBenchExecutor{pending: true}
	loop := NewLoop(brain, exec, nil, nil, nil, LoopConfig{MaxSteps: 6, MaxToolCalls: 3, MaxFailures: 2, RepeatLimit: 2})

	state := loop.Advance(context.Background(), loop.NewRun("bench-approval", "删除旧缓存"))
	if state.Status != RunWaitingApproval || state.PendingCall == nil || state.ApprovalID != "bench-approval" {
		t.Fatalf("bench did not stop at approval boundary: %+v", state)
	}
	exec.approved = true
	state = loop.Advance(context.Background(), state)
	if state.Status != RunCompleted || len(exec.calls) != 2 {
		t.Fatalf("bench approval resume failed: state=%+v calls=%+v", state, exec.calls)
	}
	if exec.calls[1].Name != "fs.remove" || exec.calls[1].ApprovalID != "bench-approval" {
		t.Fatalf("approval resumed a different call: %+v", exec.calls[1])
	}
}
