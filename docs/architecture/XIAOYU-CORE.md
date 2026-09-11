# 小鱼（XiaoYu）Intelligence Core

小鱼是 AI Game Manager Panel 的内置超级大脑、主智能控制入口和长期伙伴身份。AGMP 是产品；小鱼是产品的智能灵魂。二者不拆成两个产品。

## 核心职责

小鱼负责：目标理解、上下文收集、规划、工作流状态、Tool 选择、权限/审批、执行结果理解、验证、恢复与汇报。小鱼不复制各业务域代码。

```text
User
 -> XiaoYu Context
 -> Planner / Workflow
 -> Tool Router
 -> Permission / Approval
 -> Domain Service
 -> Result Validator
 -> Continue / Complete
```

## 实现边界

- 核心语言：Rust。
- 核心 crate：`xiaoyu-core`。
- 稳定协议：`xiaoyu-protocol` / `xiaoyu.v1`。
- Go bridge：`internal/xiaoyu/runtime`。
- 用户入口：`frontend/src/features/xiaoyu`。
- 内部二进制：`AI-Game-Manager-XiaoYu.exe`，只随完整 AGMP 发行。

Rust Core 可以使用内部子进程隔离崩溃，但产品层面仍是同一个 AGMP。普通用户不需要 Rust/Cargo/MSVC，也不直接启动小鱼二进制。

## Provider 原则

Brain 不绑定 OpenAI、Claude、Gemini、DeepSeek、Ollama 或其他模型。用户配置的模型提供 XiaoYu 的通用智能；Provider Adapter 必须尽量保留厂商原生 reasoning、tool semantics、replay、multimodal 等公开能力，而不是压成最低公分母。换 Provider 不得要求重写 XiaoYu 身份、Memory、Skill、Workflow 或 Permission。

## 安全原则

小鱼的能力面不再等同于“已注册了多少个领域按钮”。Expert / Skill / Memory 提供优先指导，结构化 Domain Tool 始终优先；当 Domain Tool 缺失或不足时，小鱼可以继续使用通用文件能力与 `shell.exec` 自主解决。

能力与授权严格分离：`shell.exec` 是 Host 注册的受控通用后备能力，不是绕过 Host 的裸 Shell。真实执行仍必须经过身份、RBAC、路径、参数、敏感操作 Step-up、三种审批模式、审计和执行后验证。`process.run` 只保留为人工/兼容别名。

## 状态机方向

核心状态至少能够表达：Planning、WaitingApproval、Executing、Validating、Recovering、WaitingUser、Completed、Failed、Cancelled。后续实现优先使用 Rust 强类型状态，不用散落字符串模拟状态机。

## 目录策略

0.1.81 将 Rust Workspace 收敛为两个主要 crate，避免为 Brain/Planner/Context/Approval/Workflow 各拆一个 crate。逻辑可以分模块，物理项目保持集中；只有独立生命周期出现后再拆。
