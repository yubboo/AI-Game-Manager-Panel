# 小鱼（XiaoYu）Agent Runtime

小鱼是 AI Game Manager Panel 的内置智能核心和主自动化入口。AGMP 是产品；XiaoYu 是产品的智能灵魂。二者不是两个用户产品。

## 1. 0.2.9 定位

0.2.9 起 Rust `xiaoyu-core` 的长期定位从“Brain-only”升级为：

> **XiaoYu Agent Runtime / Native Execution / Security Boundary**

这不是一次性把现有 Go 全部重写，而是明确从现在开始新的通用 Agent 能力优先 Rust。

长期职责：

```text
Goal / Brain Policy
Agent Loop / Recovery
Tool Search / Capability Discovery
Context / Session / Thread
Job / PTY
Generic Shell / Process / Filesystem
Apply Patch
Sandbox / Capability Scope
Reflection / Experience
Subagent
Protocol
```

## 2. 不属于 Rust 的内容

Rust 不复制 AGMP Domain：

```text
DST / Minecraft 业务规则
Steam / SteamCMD 业务流程
实例模型
备份业务
许可证
Updater 业务规则
用户/组织业务
```

这些继续由 Go Service 维护，并通过结构化 Domain Tool 提供给 XiaoYu。

## 3. 当前现实与迁移方式

当前已有稳定 Go Agent/Runtime 代码：

```text
internal/xiaoyu/host
internal/platform/runtime
internal/ops/files
```

它们在 0.2.9 继续工作。迁移规则：

1. 新 Agent 通用能力不再默认加到 Go；
2. Rust 先建立等价协议、测试与 Agent Bench；
3. Host 切换权威路径；
4. GitHub CI 全绿；
5. 再删除对应旧 Go 实现。

禁止为了架构整齐一次性重写已经工作的 Go 代码。

## 4. Provider 原则

模型 Provider 不决定 XiaoYu 的产品身份。用户配置的 GPT / DeepSeek / Claude / Gemini / 本地模型提供通用智能；XiaoYu Runtime 提供持续身份、Context、Tool、Memory、Skill、Expert、审批、安全与执行环境。

Provider Adapter 应尽可能保留厂商原生能力：

- reasoning / thinking；
- tool calling semantics；
- reasoning replay；
- multimodal；
- streaming；
- context/tool capability differences。

不能为了统一接口把高端模型压成最低公分母。

## 5. Domain-first, not Domain-only

能力选择：

```text
Expert / Skill / Memory
        ↓
Domain Tool（优先）
        ↓
Generic Tool
        ↓
Shell / PTY fallback
        ↓
Observation / Verify / Recover
```

专业知识是优先调教，不是能力限制。缺少专用 Domain Tool 时，只要通用能力能合法完成任务，XiaoYu 不应直接认输。

## 6. 安全原则

Rust 更适合 Native Agent Runtime，并不意味着 Rust 可以拥有无限权限。

安全由多层共同完成：

```text
memory safety
+ Host identity / RBAC
+ capability scope
+ approval
+ sandbox
+ path/argument policy
+ secret protection
+ audit
+ verification
```

三种审批模式继续固定为：

- 请求批准；
- 帮我批准；
- 完全访问权限。

用户拥有最终 Yes/No。Full Access 仍不能绕过路径、参数、秘密、审计和 Sandbox 硬边界。

## 7. 协议

稳定协议当前为 `xiaoyu.v1`。

Rust Runtime 与 Go Host 通过协议交换：

- Runtime Status；
- Brain Prompt / Decision；
- Tool/Capability metadata；
- Session；
- 后续 Job / PTY / Sandbox / Native Tool requests。

协议字段修改必须同步 Go/Rust 测试与 Gate。

## 8. 当前 crate 策略

当前保持两个主要 crate：

```text
xiaoyu-core
xiaoyu-protocol
```

优先在 crate 内拆短模块文件，例如：

```text
tool_search.rs
session.rs
job.rs
sandbox.rs
```

只有独立生命周期、权限边界或发布需求真实出现后，才拆成更多 crate。

## 9. Desktop 关系

XiaoYu Rust Runtime 是内部组件，不等于桌面壳。

当前 Windows 主桌面仍是 Wails；未来是否迁移 Tauri 单独评估。无论桌面壳是什么，XiaoYu Runtime 都不应与某个 UI 框架强绑定。

