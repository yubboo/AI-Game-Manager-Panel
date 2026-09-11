// Package tools 定义 AI Game Manager Panel AI 可调用工具的执行接口。
package contract

import (
	"context"
)

// Result 是一次 Tool 执行后的结构化结果。
type Result struct {
	Summary string         `json:"summary"`
	Data    map[string]any `json:"data,omitempty"`
}

// Tool 是 AI 与 AI Game Manager Panel 业务之间唯一允许的执行边界。
// 实现必须调用现有 Service，不允许在这里重新复制业务逻辑。
type Tool interface {
	Spec() ToolSpec
	Execute(ctx context.Context, arguments map[string]any) (Result, error)
}
