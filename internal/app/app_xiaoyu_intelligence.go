package app

import (
	"errors"
	"fmt"
	"strings"

	authservice "github.com/yubboo/AI-Game-Manager-Panel/internal/system/auth"
	xiaoyuhost "github.com/yubboo/AI-Game-Manager-Panel/internal/xiaoyu/host"
)

func isXiaoYuIntelligenceSupervisor(user authservice.User) bool {
	return user.Role == authservice.RoleOwner || user.Role == authservice.RoleAdministrator
}

func (a *Application) xiaoyuIntelligenceUser(token string) (authservice.User, bool, error) {
	user, err := a.requireXiaoYuMember(token)
	if err != nil {
		return authservice.User{}, false, err
	}
	if a.xiaoyuIntelligence == nil {
		return authservice.User{}, false, errors.New("XiaoYu Intelligence Store 不可用")
	}
	if strings.TrimSpace(user.OrganizationID) == "" {
		return authservice.User{}, false, authservice.ErrOrganizationRequired
	}
	return user, isXiaoYuIntelligenceSupervisor(user), nil
}

func canPublishXiaoYuIntelligence(user authservice.User, visibility xiaoyuhost.IntelligenceVisibility) bool {
	if visibility == "" || visibility == xiaoyuhost.VisibilityPrivate {
		return true
	}
	return isXiaoYuIntelligenceSupervisor(user)
}

// XiaoYuIntelligenceCatalog only returns intelligence visible inside the
// authenticated member's Organization/Group/User boundary. It never performs
// an instance-global query.
func (a *Application) XiaoYuIntelligenceCatalog(token string) (xiaoyuhost.IntelligenceCatalog, error) {
	user, _, err := a.xiaoyuIntelligenceUser(token)
	if err != nil {
		return xiaoyuhost.IntelligenceCatalog{}, err
	}
	return a.xiaoyuIntelligence.CatalogFor(user.OrganizationID, user.GroupID, user.ID), nil
}

func (a *Application) SaveXiaoYuMemory(token string, request xiaoyuhost.MemorySaveRequest) (xiaoyuhost.MemoryRecord, error) {
	user, supervisor, err := a.xiaoyuIntelligenceUser(token)
	if err != nil {
		return xiaoyuhost.MemoryRecord{}, err
	}
	if !canPublishXiaoYuIntelligence(user, request.Visibility) {
		return xiaoyuhost.MemoryRecord{}, authservice.ErrForbidden
	}
	// Personal/session/task memory cannot be published as organization knowledge.
	// Shared server/instance/experience knowledge is an explicit administrator act.
	switch request.Kind {
	case xiaoyuhost.MemoryUser, xiaoyuhost.MemorySession, xiaoyuhost.MemoryTask:
		request.Visibility = xiaoyuhost.VisibilityPrivate
	}
	switch request.Kind {
	case xiaoyuhost.MemoryUser:
		request.Scope.ID = user.ID
	case xiaoyuhost.MemoryExperience:
		if request.Visibility == xiaoyuhost.VisibilityPrivate {
			request.Scope.ID = user.ID
		} else {
			// Shared experience is organization/group knowledge, not a single
			// member's identity-scoped memory. Use the explicit wildcard scope
			// understood by memoryMatchesRun so it can be reused by authorized
			// peers without leaking a private user scope.
			request.Scope.ID = "*"
		}
	case xiaoyuhost.MemoryTask:
		request.Visibility = xiaoyuhost.VisibilityPrivate
		if a.xiaoyuRuns == nil {
			return xiaoyuhost.MemoryRecord{}, errors.New("XiaoYu RunManager 不可用")
		}
		state, ok := a.xiaoyuRuns.Get(strings.TrimSpace(request.Scope.ID))
		if !ok {
			return xiaoyuhost.MemoryRecord{}, xiaoyuhost.ErrRunNotFound
		}
		// A supervisor may supervise the Run, but private task memory belongs to
		// the Run initiator. Do not let supervision become a private-memory write.
		if state.InitiatorID != user.ID {
			return xiaoyuhost.MemoryRecord{}, authservice.ErrForbidden
		}
		request.Scope.ID = state.Context.TaskID
	}
	if request.Visibility == xiaoyuhost.VisibilityOrganization && !supervisor {
		return xiaoyuhost.MemoryRecord{}, authservice.ErrForbidden
	}
	value, err := a.xiaoyuIntelligence.SaveMemory(user.OrganizationID, user.GroupID, user.ID, request)
	if err != nil {
		return value, err
	}
	a.recordOperation("info", "xiaoyu", "save_memory", value.ID, "success", fmt.Sprintf("保存 XiaoYu Memory（kind=%s visibility=%s，内容不写审计日志）", value.Kind, value.Visibility), "", "")
	a.observeXiaoYu("intelligence/memory-saved", "XiaoYu Memory 已更新", map[string]any{"id": value.ID, "kind": value.Kind, "visibility": value.Visibility})
	return value, nil
}

func (a *Application) SaveXiaoYuSkill(token string, request xiaoyuhost.SkillSaveRequest) (xiaoyuhost.SkillDefinition, error) {
	user, _, err := a.xiaoyuIntelligenceUser(token)
	if err != nil {
		return xiaoyuhost.SkillDefinition{}, err
	}
	if !canPublishXiaoYuIntelligence(user, request.Visibility) {
		return xiaoyuhost.SkillDefinition{}, authservice.ErrForbidden
	}
	value, err := a.xiaoyuIntelligence.SaveSkill(user.OrganizationID, user.GroupID, user.ID, request)
	if err != nil {
		return value, err
	}
	a.recordOperation("info", "xiaoyu", "save_skill", value.ID, "success", fmt.Sprintf("保存 XiaoYu Skill：%s（visibility=%s，Prompt 不写审计日志）", value.Name, value.Visibility), "", "")
	a.observeXiaoYu("intelligence/skill-saved", "XiaoYu Skill 已更新", map[string]any{"id": value.ID, "name": value.Name, "visibility": value.Visibility})
	return value, nil
}

func (a *Application) SaveXiaoYuExpert(token string, request xiaoyuhost.ExpertSaveRequest) (xiaoyuhost.ExpertDefinition, error) {
	user, _, err := a.xiaoyuIntelligenceUser(token)
	if err != nil {
		return xiaoyuhost.ExpertDefinition{}, err
	}
	if !canPublishXiaoYuIntelligence(user, request.Visibility) {
		return xiaoyuhost.ExpertDefinition{}, authservice.ErrForbidden
	}
	value, err := a.xiaoyuIntelligence.SaveExpert(user.OrganizationID, user.GroupID, user.ID, request)
	if err != nil {
		return value, err
	}
	a.recordOperation("info", "xiaoyu", "save_expert", value.ID, "success", fmt.Sprintf("保存 XiaoYu Expert：%s（visibility=%s，Prompt/Knowledge 不写审计日志）", value.Name, value.Visibility), "", "")
	a.observeXiaoYu("intelligence/expert-saved", "XiaoYu Expert 已更新", map[string]any{"id": value.ID, "name": value.Name, "visibility": value.Visibility})
	return value, nil
}

func (a *Application) DeleteXiaoYuIntelligence(token, kind, id string) error {
	user, supervisor, err := a.xiaoyuIntelligenceUser(token)
	if err != nil {
		return err
	}
	kind, id = strings.ToLower(strings.TrimSpace(kind)), strings.TrimSpace(id)
	if kind != "memory" && kind != "skill" && kind != "expert" {
		return errors.New("Intelligence kind 只允许 memory、skill 或 expert")
	}
	if err := a.xiaoyuIntelligence.Delete(user.OrganizationID, user.ID, kind, id, supervisor); err != nil {
		return err
	}
	a.recordOperation("warn", "xiaoyu", "delete_intelligence", id, "success", "删除 XiaoYu Intelligence 项（内容不写审计日志）", "", "")
	a.observeXiaoYu("intelligence/deleted", "XiaoYu Intelligence 项已删除", map[string]any{"id": id, "kind": kind})
	return nil
}
