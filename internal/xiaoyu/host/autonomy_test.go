package host

import (
	"context"
	"testing"
)

func TestOperationalMutationMustBeVerifiedWhenReadCapabilityExists(t *testing.T) {
	brain := &scriptedBrain{decisions: []Decision{
		{Kind: DecisionTool, Tool: "settings.theme.set", Arguments: map[string]any{"theme": "light"}},
		{Kind: DecisionComplete, Message: "done too early"},
		{Kind: DecisionTool, Tool: "settings.get"},
		{Kind: DecisionComplete, Message: "verified"},
	}}
	exec := &fakeExecutor{}
	loop := NewLoop(brain, exec, nil, func() []xiaoyuToolView {
		return []xiaoyuToolView{
			{Name: "settings.theme.set", Category: "settings", Risk: "operate"},
			{Name: "settings.get", Category: "settings", Risk: "read"},
		}
	}, nil, LoopConfig{MaxSteps: 8, MaxToolCalls: 4, MaxFailures: 2, RepeatLimit: 2})
	result := loop.Advance(context.Background(), loop.NewRun("R-verify", "把主题改成浅色"))
	if result.Status != RunCompleted || result.Message != "verified" {
		t.Fatalf("mutation completed without verification: %+v", result)
	}
	if exec.calls != 2 {
		t.Fatalf("expected mutation + verification tool, calls=%d", exec.calls)
	}
	foundGuard := false
	for _, item := range result.Observations {
		if item.Summary == "完成判定暂缓" {
			foundGuard = true
			break
		}
	}
	if !foundGuard {
		t.Fatal("completion evidence guard did not activate")
	}
}

func TestGoalRelevantIntelligenceAlwaysIncludesAGMPBaseline(t *testing.T) {
	store := NewIntelligenceStore(t.TempDir() + "/intelligence.json")
	ctx := store.ContextForGoal("org", "group", "user", RunContext{}, "帮我检查并修复 SteamCMD 环境")
	if !containsSkill(ctx.Skills, "builtin.skill.agmp-autonomy") {
		t.Fatalf("missing AGMP autonomy baseline: %#v", ctx.Skills)
	}
	if !containsSkill(ctx.Skills, "builtin.skill.environment-doctor") {
		t.Fatalf("missing environment skill: %#v", ctx.Skills)
	}
	if !containsExpert(ctx.Experts, "builtin.expert.agmp") || !containsExpert(ctx.Experts, "builtin.expert.environment") {
		t.Fatalf("missing relevant experts: %#v", ctx.Experts)
	}
	if containsExpert(ctx.Experts, "builtin.expert.minecraft") {
		t.Fatalf("irrelevant minecraft expert polluted context: %#v", ctx.Experts)
	}
}

func containsSkill(items []SkillDefinition, id string) bool {
	for _, item := range items {
		if item.ID == id {
			return true
		}
	}
	return false
}

func containsExpert(items []ExpertDefinition, id string) bool {
	for _, item := range items {
		if item.ID == id {
			return true
		}
	}
	return false
}

func TestToolGuidanceTierKeepsDomainPreferredAndShellAsFallback(t *testing.T) {
	cases := map[string]string{
		"environment.remove_runtime": "domain",
		"fs.write":                   "general",
		"memory.remember":            "control",
		"shell.exec":                 "fallback",
	}
	for name, want := range cases {
		if got := toolGuidanceTier(name, ""); got != want {
			t.Fatalf("tool %s tier=%s, want %s", name, got, want)
		}
	}
}
