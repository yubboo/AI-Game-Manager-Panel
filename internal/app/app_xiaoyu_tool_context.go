package app

import (
	"context"

	authservice "github.com/yubboo/AI-Game-Manager-Panel/internal/system/auth"
	xiaoyuhost "github.com/yubboo/AI-Game-Manager-Panel/internal/xiaoyu/host"
)

type xiaoyuInvocationContextKey struct{}

type xiaoyuInvocationContext struct {
	User       authservice.User
	RunID      string
	RunContext xiaoyuhost.RunContext
}

func withXiaoYuInvocationContext(ctx context.Context, value xiaoyuInvocationContext) context.Context {
	return context.WithValue(ctx, xiaoyuInvocationContextKey{}, value)
}

func getXiaoYuInvocationContext(ctx context.Context) (xiaoyuInvocationContext, bool) {
	value, ok := ctx.Value(xiaoyuInvocationContextKey{}).(xiaoyuInvocationContext)
	return value, ok
}
