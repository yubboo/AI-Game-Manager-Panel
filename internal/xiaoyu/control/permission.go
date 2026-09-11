// Package permissions 定义 AGMP 内置 小鱼 Tool 的权限与审批策略。
package control

import "github.com/yubboo/AI-Game-Manager-Panel/internal/xiaoyu/contract"

// Decision 表示权限引擎对一次 小鱼 Tool 请求的判定。
type Decision string

const (
	DecisionAllow   Decision = "allow"
	DecisionConfirm Decision = "confirm"
	DecisionDeny    Decision = "deny"
)

// Mode 对应 小鱼里由最高管理员选择的三个固定审批模式。
type Mode string

const (
	ModeAsk  Mode = "ask"  // 请求批准
	ModeRisk Mode = "risk" // 帮我批准
	ModeFull Mode = "full" // 完全访问权限
)

func ValidMode(mode Mode) bool {
	switch mode {
	case ModeAsk, ModeRisk, ModeFull:
		return true
	default:
		return false
	}
}

func ModeLabel(mode Mode) string {
	switch mode {
	case ModeRisk:
		return "帮我批准"
	case ModeFull:
		return "完全访问权限"
	default:
		return "请求批准"
	}
}

// Policy 是与模型 Provider 解耦的权限策略快照。
// 三个模式必须各用自己的策略表，禁止把“请求批准”和“帮我批准”混成同一规则。
type Policy struct {
	Mode Mode
	Ask  map[string]string
	Risk map[string]string
	Full map[string]string
}

// Decide 只决定“已注册 Tool”是否需要批准；Tool 是否存在/是否可供 小鱼调用由 Registry/Module 负责。
func (p Policy) Decide(risk contract.RiskLevel) Decision {
	switch p.Mode {
	case ModeFull:
		return parseDecision(p.Full[string(risk)], defaultFullDecision(risk))
	case ModeRisk:
		return parseDecision(p.Risk[string(risk)], defaultAssistedDecision(risk))
	default:
		return parseDecision(p.Ask[string(risk)], defaultRequestDecision(risk))
	}
}

// 请求批准：只读可以直接完成；会产生副作用的下一步都必须等待用户明确决定。
func defaultRequestDecision(risk contract.RiskLevel) Decision {
	if risk == contract.RiskRead {
		return DecisionAllow
	}
	return DecisionConfirm
}

// 帮我批准：只读、普通操作和可逆修改自动完成；破坏性/系统级动作仍必须由用户确认。
// 这对应 Codex 一类“低风险自动推进，高风险再打断”的 assisted 工作流。
// Tool 自身的 RBAC、参数校验、作用域、敏感操作 step-up 与结果验证仍是独立硬边界。
func defaultAssistedDecision(risk contract.RiskLevel) Decision {
	switch risk {
	case contract.RiskRead, contract.RiskOperate, contract.RiskModify:
		return DecisionAllow
	default:
		return DecisionConfirm
	}
}

// 完全访问权限：审批层对已注册且已启用的 AGMP Tool 自动放行。
// 这不关闭模块自身的参数校验、作用域、备份、幂等、回滚和执行后验证等硬安全规则。
func defaultFullDecision(contract.RiskLevel) Decision { return DecisionAllow }

func parseDecision(value string, fallback Decision) Decision {
	switch Decision(value) {
	case DecisionAllow, DecisionConfirm, DecisionDeny:
		return Decision(value)
	default:
		return fallback
	}
}
