package control

import (
	"testing"

	"github.com/yubboo/AI-Game-Manager-Panel/internal/xiaoyu/contract"
)

func TestThreeApprovalModesHaveDifferentSemantics(t *testing.T) {
	tests := []struct {
		mode Mode
		risk contract.RiskLevel
		want Decision
	}{
		{ModeAsk, contract.RiskRead, DecisionAllow},
		{ModeAsk, contract.RiskOperate, DecisionConfirm},
		{ModeAsk, contract.RiskSystem, DecisionConfirm},
		{ModeRisk, contract.RiskRead, DecisionAllow},
		{ModeRisk, contract.RiskOperate, DecisionAllow},
		{ModeRisk, contract.RiskModify, DecisionAllow},
		{ModeRisk, contract.RiskDestructive, DecisionConfirm},
		{ModeRisk, contract.RiskSystem, DecisionConfirm},
		{ModeFull, contract.RiskSystem, DecisionAllow},
	}
	for _, tt := range tests {
		got := (Policy{Mode: tt.mode}).Decide(tt.risk)
		if got != tt.want {
			t.Fatalf("mode=%s risk=%s got=%s want=%s", tt.mode, tt.risk, got, tt.want)
		}
	}
}

func TestPolicyConfigCanTightenButNotRenameModes(t *testing.T) {
	policy := Policy{
		Mode: ModeRisk,
		Risk: map[string]string{"operate": "confirm"},
	}
	if got := policy.Decide(contract.RiskOperate); got != DecisionConfirm {
		t.Fatalf("expected configured confirm, got %s", got)
	}
	if !ValidMode(ModeAsk) || !ValidMode(ModeRisk) || !ValidMode(ModeFull) || ValidMode("agent") {
		t.Fatal("approval mode ids must stay ask/risk/full")
	}
	if ModeLabel(ModeAsk) != "请求批准" || ModeLabel(ModeRisk) != "帮我批准" || ModeLabel(ModeFull) != "完全访问权限" {
		t.Fatal("approval mode labels changed unexpectedly")
	}
}
