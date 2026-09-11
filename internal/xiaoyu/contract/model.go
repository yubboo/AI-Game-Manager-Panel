// Package model 定义 小鱼（XiaoYu） 与 Go Core 之间的稳定结构。
package contract

// RiskLevel 表示 小鱼 Tool 的操作风险等级。
type RiskLevel string

const (
	RiskRead        RiskLevel = "read"
	RiskOperate     RiskLevel = "operate"
	RiskModify      RiskLevel = "modify"
	RiskDestructive RiskLevel = "destructive"
	RiskSystem      RiskLevel = "system"
)

// ToolSpec 描述一个允许小鱼调用的 AI Game Manager Panel 工具。
// 小鱼只能调用注册表中存在的工具，不能直接获得任意 Shell 权限。
type ToolSpec struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Risk        RiskLevel      `json:"risk"`
	Category    string         `json:"category"`
	Manual      bool           `json:"manual"`
	XiaoYu      bool           `json:"xiaoyu"`
	Parameters  map[string]any `json:"parameters,omitempty"`
	Source      string         `json:"source,omitempty"`
}

// PlanStep 是 小鱼计划中的一个受控执行步骤。
type PlanStep struct {
	ID        string         `json:"id"`
	Tool      string         `json:"tool"`
	Summary   string         `json:"summary"`
	Arguments map[string]any `json:"arguments,omitempty"`
}

// Plan 是展示给用户确认的结构化执行计划。
type Plan struct {
	Title string     `json:"title"`
	Steps []PlanStep `json:"steps"`
}
