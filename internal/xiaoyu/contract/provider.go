// Package provider 定义 AI 模型 Provider 的抽象接口。
//
// 该接口不绑定 OpenAI 或其他模型厂商，真实 Provider 会在 AI Phase 中单独实现。
package contract

import (
	"context"
)

// Request 是用户交给 小鱼的目标描述。
type Request struct {
	Prompt string `json:"prompt"`
}

// Provider 负责把自然语言目标转换为结构化计划，不直接执行系统操作。
type Provider interface {
	Name() string
	BuildPlan(ctx context.Context, request Request, tools []ToolSpec) (Plan, error)
}
