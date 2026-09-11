# XiaoYu Agent Spine

> 目标：小鱼不是“LLM + 一堆 Tool”，而是 AGMP 内置、可审计、可插话、可恢复的 Agent Runtime。

## 参考的成熟开源控制主干

设计参考 OpenAI Codex CLI 与 DeepSeek Harness 的公开 Agent Runtime 思路，但不复制其产品代码：

- Codex：Thread / Turn 生命周期、执行审批事件、运行中 steering、取消与回滚语义。
- DeepSeek Harness：Session 作为事实来源、System Prompt 组装、Tool Registry，以及 `pre-execute -> execute -> post-execute -> result` 的受保护执行流水线。

AGMP 保持自己的边界：Rust `xiaoyu-core` 是唯一 Brain；Go Host 是唯一真实执行、权限、审批、Domain Tool 边界。

## 0.1.98 已落地

### 1. Thread continuity

同一 `RunContext.SessionID` 下，Host 构建有限的 `ThreadContext`：

- 最近目标；
- Run 状态；
- 结构化 Tool 回执；
- 明确的可逆操作提示。

浏览器本地聊天记录不是权限或执行依据。

### 2. Steering

用户在 Run 执行中追加消息时：

1. Host 记录最新用户指令；
2. 取消当前 Brain turn；
3. 等待当前 turn 在 Host 边界停稳；
4. 将指令写入 Observation；
5. 重新进入 Planning；
6. 使用同一 Run 继续。

Human steering 永远优先于旧计划。

### 3. Approval hard gate

副作用操作必须经过统一 Host 审批入口：

`model tool call -> Host policy -> pending approval -> user decision -> exact fingerprint consume -> domain execute`

模型推荐、Skill、Memory、Expert、历史批准都不能代替真实批准。

### 4. Execution receipts

Tool 执行结果必须成为下一步模型可见事实。对于可逆操作，Domain Tool 应返回足以构建 undo hint 的结构化回执。

## 0.2.2 Guided Autonomy 增量

- Domain Tool 从“能力边界”改为“优先路径”；
- 新增 workspace file mutation 与受控 `shell.exec` 通用后备能力；
- Expert / Skill / Memory 只做专业 Guidance，不构成 Tool allowlist；
- Model-visible Observation 采用有界投影，完整执行回执仍保留在 Host；
- Memory 进入 Prompt 前按 scope / relevance / confidence / recency 排序；
- Provider Capability Matrix 开始描述 native reasoning / replay / tool-choice 兼容性；
- DeepSeek thinking 模式不再无条件发送不兼容的 `tool_choice`。

详见 `docs/development/XIAOYU-AGENT-RUNTIME.md`。

## 下一阶段

- 持久化 append-only Session Event Log；
- Plan/Review/Approval 卡片与一次性计划审批；
- Tool pre/post hooks、timeout/retry/metrics；
- PTY / long-running Jobs / stdin / cancel；
- Tool Search 与安全的并行 Tool 调度；
- Recovery policy 与 checkpoint/rollback；
- Skills / Experts / Memory 的选择器和验证器；
- Eval Suite：审批绕过率、假完成率、steering 响应、恢复成功率。
