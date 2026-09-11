package app

import (
	"testing"

	authservice "github.com/yubboo/AI-Game-Manager-Panel/internal/system/auth"
	xiaoyuhost "github.com/yubboo/AI-Game-Manager-Panel/internal/xiaoyu/host"
)

func TestXiaoYuRunControlIsScopedToInitiatorOrSupervisor(t *testing.T) {
	run := xiaoyuhost.RunState{InitiatorID: "user-a"}
	if !canControlXiaoYuRun(authservice.User{ID: "user-a", Role: authservice.RoleOperator}, run) {
		t.Fatal("run initiator should control its own XiaoYu run")
	}
	if canControlXiaoYuRun(authservice.User{ID: "user-b", Role: authservice.RoleOperator}, run) {
		t.Fatal("operator must not control another user's XiaoYu run")
	}
	if !canControlXiaoYuRun(authservice.User{ID: "admin", Role: authservice.RoleAdministrator}, run) {
		t.Fatal("administrator should supervise XiaoYu runs")
	}
	if !canControlXiaoYuRun(authservice.User{ID: "owner", Role: authservice.RoleOwner}, run) {
		t.Fatal("owner should supervise XiaoYu runs")
	}
}

func TestXiaoYuOperatorEventFilterOnlyExposesOwnRun(t *testing.T) {
	runs := xiaoyuhost.NewRunManager()
	own, err := runs.CreateFor("own", "user-a", "A")
	if err != nil {
		t.Fatal(err)
	}
	other, err := runs.CreateFor("other", "user-b", "B")
	if err != nil {
		t.Fatal(err)
	}
	a := &Application{xiaoyuRuns: runs}
	filter := a.xiaoyuEventFilter(authservice.User{ID: "user-a", Role: authservice.RoleOperator})
	if !filter(xiaoyuhost.Event{Type: "run", RunID: own.ID}) {
		t.Fatal("operator should receive its own run events")
	}
	if filter(xiaoyuhost.Event{Type: "run", RunID: other.ID}) {
		t.Fatal("operator must not receive another user's run events")
	}
	if filter(xiaoyuhost.Event{Type: "model/default"}) {
		t.Fatal("operator must not receive global administrative XiaoYu events")
	}
}
