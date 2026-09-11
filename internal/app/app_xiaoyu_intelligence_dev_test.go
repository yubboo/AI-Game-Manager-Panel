//go:build agmp_dev_license

package app

import (
	"context"
	"errors"
	"testing"

	authservice "github.com/yubboo/AI-Game-Manager-Panel/internal/system/auth"
	xiaoyuhost "github.com/yubboo/AI-Game-Manager-Panel/internal/xiaoyu/host"
)

func intelligenceAccessFixture(t *testing.T) (*Application, authservice.Session, authservice.Session, authservice.Session) {
	t.Helper()
	a := NewWithOptions(Options{Root: t.TempDir(), DataDir: "data"})
	owner, err := a.CreateInitialAdministrator(authservice.CreateOwnerRequest{Username: "owner", Password: "owner-password-123"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := a.auth.ConfirmCredentialStepUp(owner.Token, authservice.CredentialStepUpRequest{Password: "owner-password-123"}); err != nil {
		t.Fatal(err)
	}
	makeMember := func(username string) authservice.Session {
		invite, err := a.CreateUserInvitation(owner.Token, authservice.CreateInvitationRequest{Role: authservice.RoleOperator, TargetUsername: username})
		if err != nil {
			t.Fatal(err)
		}
		member, err := a.RegisterInvitedUser(authservice.RegisterInvitationRequest{InvitationToken: invite.Token, Username: username, Password: "member-password-123"})
		if err != nil {
			t.Fatal(err)
		}
		grant, err := a.CreateMemberCoreAuthorization(owner.Token, authservice.CreateMemberAuthorizationRequest{UserID: member.User.ID})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := a.RedeemMyCoreAuthorization(member.Token, authservice.RedeemMemberAuthorizationRequest{Token: grant.Token}); err != nil {
			t.Fatal(err)
		}
		member.User, err = a.auth.Validate(member.Token)
		if err != nil {
			t.Fatal(err)
		}
		return member
	}
	return a, owner, makeMember("operator-a"), makeMember("operator-b")
}

func TestXiaoYuIntelligencePrivateGroupOrganizationBoundaries(t *testing.T) {
	a, owner, opA, opB := intelligenceAccessFixture(t)
	privateSkill, err := a.SaveXiaoYuSkill(opA.Token, xiaoyuhost.SkillSaveRequest{Name: "A 的私人 Skill", Prompt: "只给 A 使用。", Visibility: xiaoyuhost.VisibilityPrivate})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := a.SaveXiaoYuSkill(opA.Token, xiaoyuhost.SkillSaveRequest{Name: "非法共享", Prompt: "operator 不应发布。", Visibility: xiaoyuhost.VisibilityOrganization}); !errors.Is(err, authservice.ErrForbidden) {
		t.Fatalf("operator organization publish must be denied: %v", err)
	}
	shared, err := a.SaveXiaoYuMemory(owner.Token, xiaoyuhost.MemorySaveRequest{Kind: xiaoyuhost.MemoryServer, Scope: xiaoyuhost.MemoryScope{ID: "server-1"}, Content: "公司规则：维护前先建立恢复点。", Visibility: xiaoyuhost.VisibilityOrganization})
	if err != nil {
		t.Fatal(err)
	}
	catalogB, err := a.XiaoYuIntelligenceCatalog(opB.Token)
	if err != nil {
		t.Fatal(err)
	}
	for _, skill := range catalogB.Skills {
		if skill.ID == privateSkill.ID {
			t.Fatal("private Skill leaked to another organization member")
		}
	}
	seenShared := false
	for _, memory := range catalogB.Memories {
		seenShared = seenShared || memory.ID == shared.ID
	}
	if !seenShared {
		t.Fatal("organization-published Memory was not visible to same organization member")
	}
}

func TestXiaoYuSensitiveMemoryVisibleLocallyButNeverInjected(t *testing.T) {
	a, _, opA, _ := intelligenceAccessFixture(t)
	item, err := a.SaveXiaoYuMemory(opA.Token, xiaoyuhost.MemorySaveRequest{Kind: xiaoyuhost.MemoryServer, Scope: xiaoyuhost.MemoryScope{ID: "server-1"}, Content: "本地敏感备注，不应发送模型", Sensitivity: xiaoyuhost.MemorySensitive, Visibility: xiaoyuhost.VisibilityPrivate})
	if err != nil {
		t.Fatal(err)
	}
	catalog, err := a.XiaoYuIntelligenceCatalog(opA.Token)
	if err != nil {
		t.Fatal(err)
	}
	seen := false
	for _, memory := range catalog.Memories {
		seen = seen || memory.ID == item.ID
	}
	if !seen {
		t.Fatal("sensitive Memory should remain manageable locally")
	}
	ctx := a.xiaoyuIntelligence.ContextFor(opA.User.OrganizationID, opA.User.GroupID, opA.User.ID, xiaoyuhost.RunContext{ServerID: "server-1"})
	for _, memory := range ctx.Memories {
		if memory.ID == item.ID {
			t.Fatal("sensitive Memory entered model context")
		}
	}
}

func TestXiaoYuTaskMemoryCannotTargetAnotherMembersRun(t *testing.T) {
	a, _, opA, opB := intelligenceAccessFixture(t)
	runA, err := a.xiaoyuRuns.CreateForContext("A run", opA.User.ID, opA.User.Username, xiaoyuhost.RunContext{TaskID: "forged"})
	if err != nil {
		t.Fatal(err)
	}
	if runA.Context.TaskID != runA.ID {
		t.Fatal("RunManager did not replace caller task scope")
	}
	_, err = a.SaveXiaoYuMemory(opB.Token, xiaoyuhost.MemorySaveRequest{Kind: xiaoyuhost.MemoryTask, Scope: xiaoyuhost.MemoryScope{ID: runA.ID}, Content: "attempt to write into A task"})
	if !errors.Is(err, authservice.ErrForbidden) {
		t.Fatalf("other member task Memory write should be forbidden: %v", err)
	}
}

func TestMemoryRememberUsesAuthenticatedServerOwnedRunScope(t *testing.T) {
	a, _, opA, _ := intelligenceAccessFixture(t)
	run, err := a.xiaoyuRuns.CreateForContext("remember", opA.User.ID, opA.User.Username, xiaoyuhost.RunContext{SessionID: "session-a", ServerID: "server-a", InstanceID: "instance-a"})
	if err != nil {
		t.Fatal(err)
	}
	ctx := withXiaoYuInvocationContext(context.Background(), xiaoyuInvocationContext{User: opA.User, RunID: run.ID, RunContext: run.Context})
	result, err := a.xiaoyuTools.Execute(ctx, "memory.remember", map[string]any{"scope": "task", "content": "这个任务已经确认 Java Runtime。", "confidence": 0.9})
	if err != nil {
		t.Fatal(err)
	}
	if result.Summary == "" {
		t.Fatal("memory.remember returned no receipt")
	}
	catalog := a.xiaoyuIntelligence.CatalogFor(opA.User.OrganizationID, opA.User.GroupID, opA.User.ID)
	if len(catalog.Memories) != 1 || catalog.Memories[0].Scope.ID != run.ID || catalog.Memories[0].Visibility != xiaoyuhost.VisibilityPrivate {
		t.Fatalf("memory.remember escaped private server-owned task scope: %#v", catalog.Memories)
	}
	if _, err := a.xiaoyuTools.Execute(context.Background(), "memory.remember", map[string]any{"scope": "user", "content": "no run context"}); err == nil {
		t.Fatal("memory.remember accepted execution without authenticated Run invocation context")
	}
}

func TestXiaoYuIntelligenceRunResolverRechecksAccessAndLimitsSupervision(t *testing.T) {
	a, owner, opA, _ := intelligenceAccessFixture(t)
	privateMemory, err := a.SaveXiaoYuMemory(opA.Token, xiaoyuhost.MemorySaveRequest{
		Kind: xiaoyuhost.MemoryServer, Scope: xiaoyuhost.MemoryScope{ID: "server-shared-test"},
		Content: "A 的私人服务器备注。", Visibility: xiaoyuhost.VisibilityPrivate,
	})
	if err != nil {
		t.Fatal(err)
	}
	groupMemory, err := a.SaveXiaoYuMemory(owner.Token, xiaoyuhost.MemorySaveRequest{
		Kind: xiaoyuhost.MemoryServer, Scope: xiaoyuhost.MemoryScope{ID: "server-shared-test"},
		Content: "管理组共享服务器备注。", Visibility: xiaoyuhost.VisibilityGroup,
	})
	if err != nil {
		t.Fatal(err)
	}
	orgMemory, err := a.SaveXiaoYuMemory(owner.Token, xiaoyuhost.MemorySaveRequest{
		Kind: xiaoyuhost.MemoryServer, Scope: xiaoyuhost.MemoryScope{ID: "server-shared-test"},
		Content: "组织共享服务器备注。", Visibility: xiaoyuhost.VisibilityOrganization,
	})
	if err != nil {
		t.Fatal(err)
	}

	run, err := a.xiaoyuRuns.CreateForContext("resolver ACL", opA.User.ID, opA.User.Username, xiaoyuhost.RunContext{ServerID: "server-shared-test"})
	if err != nil {
		t.Fatal(err)
	}
	contains := func(ctx xiaoyuhost.IntelligenceContext, id string) bool {
		for _, item := range ctx.Memories {
			if item.ID == id {
				return true
			}
		}
		return false
	}

	own := a.xiaoyuIntelligenceContextForRun(opA.Token, opA.User.OrganizationID, run)
	if !contains(own, privateMemory.ID) || !contains(own, groupMemory.ID) || !contains(own, orgMemory.ID) {
		t.Fatalf("run initiator must receive private+group+organization context: %#v", own.Memories)
	}

	// A supervisor may control the Run, but supervision must not become a way to
	// read the initiator's private or group-scoped AI context.
	supervised := a.xiaoyuIntelligenceContextForRun(owner.Token, owner.User.OrganizationID, run)
	if contains(supervised, privateMemory.ID) || contains(supervised, groupMemory.ID) || !contains(supervised, orgMemory.ID) {
		t.Fatalf("supervisor context boundary violated: %#v", supervised.Memories)
	}

	// Detached autonomous runs re-check current member entitlement every turn.
	// Revoking the member invalidates the token and the resolver must fail closed.
	if _, err := a.RevokeMemberCoreAccess(owner.Token, opA.User.ID); err != nil {
		t.Fatal(err)
	}
	afterRevoke := a.xiaoyuIntelligenceContextForRun(opA.Token, opA.User.OrganizationID, run)
	if len(afterRevoke.Memories) != 0 || len(afterRevoke.Skills) != 0 || len(afterRevoke.Experts) != 0 {
		t.Fatalf("revoked member still received detached-run intelligence: %#v", afterRevoke)
	}
}

func TestSharedExperienceUsesExplicitWildcardScope(t *testing.T) {
	a, owner, _, opB := intelligenceAccessFixture(t)
	item, err := a.SaveXiaoYuMemory(owner.Token, xiaoyuhost.MemorySaveRequest{
		Kind:       xiaoyuhost.MemoryExperience,
		Scope:      xiaoyuhost.MemoryScope{ID: "caller-controlled-id-must-not-survive"},
		Content:    "组织经验：重要升级完成后必须验证服务 Ready。",
		Visibility: xiaoyuhost.VisibilityOrganization,
	})
	if err != nil {
		t.Fatal(err)
	}
	if item.Scope.ID != "*" {
		t.Fatalf("shared experience scope=%q, want wildcard", item.Scope.ID)
	}
	ctx := a.xiaoyuIntelligence.ContextFor(opB.User.OrganizationID, opB.User.GroupID, opB.User.ID, xiaoyuhost.RunContext{})
	seen := false
	for _, memory := range ctx.Memories {
		seen = seen || memory.ID == item.ID
	}
	if !seen {
		t.Fatal("organization experience wildcard was not reusable by authorized peer")
	}
}
