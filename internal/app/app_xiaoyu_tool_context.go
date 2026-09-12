package app

import (
	"context"
	"strings"

	authservice "github.com/yubboo/AI-Game-Manager-Panel/internal/system/auth"
	xiaoyucontrol "github.com/yubboo/AI-Game-Manager-Panel/internal/xiaoyu/control"
	xiaoyuhost "github.com/yubboo/AI-Game-Manager-Panel/internal/xiaoyu/host"
)

type xiaoyuInvocationContextKey struct{}

type xiaoyuInvocationContext struct {
	User       authservice.User
	RunID      string
	RunContext xiaoyuhost.RunContext
	Lease      *xiaoyucontrol.CapabilityLease
}

func withXiaoYuInvocationContext(ctx context.Context, value xiaoyuInvocationContext) context.Context {
	return context.WithValue(ctx, xiaoyuInvocationContextKey{}, value)
}

func getXiaoYuInvocationContext(ctx context.Context) (xiaoyuInvocationContext, bool) {
	value, ok := ctx.Value(xiaoyuInvocationContextKey{}).(xiaoyuInvocationContext)
	return value, ok
}

func xiaoyuLeasePrincipal(user authservice.User) string {
	organizationID := strings.TrimSpace(user.OrganizationID)
	userID := strings.TrimSpace(user.ID)
	if organizationID == "" || userID == "" {
		return ""
	}
	return organizationID + ":" + userID
}
