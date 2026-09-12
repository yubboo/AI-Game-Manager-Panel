# 小鱼（XiaoYu）Agent Runtime

小鱼是 AI Game Manager Panel 的内置智能核心和主自动化入口。AGMP 是产品；XiaoYu 是产品的智能灵魂。二者不是两个用户产品。

## 1. 0.2.11 定位

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

它们在 0.2.10 继续工作。迁移规则：

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
- Session Registry；
- Job start/status/list/output/cancel；
- 后续 PTY / Sandbox / Native Tool requests。

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



## 9. 0.2.10 Session / Job 边界

`xiaoyu rpc` 现在在单个 Runtime 进程生命周期内维护 Session 与 Job 状态。Job 可以后台运行、查询状态、读取有界输出并取消。当前 Go Host 仍以短 RPC 调用为主，因此这些 stateful primitives 尚未直接替换 `shell.exec`；0.2.11 会先建立持久 RPC Worker，再迁移模型可用的长任务路径。

任何 `jobs/start` 都必须来自已经通过 Host 身份、RBAC 和审批链的调用方；Rust 的 `hostAuthorized` 是内部契约防线，不是独立身份认证系统。

## 10. 0.2.11 Persistent Worker 边界

Go Host 现在长期监督一个 `xiaoyu rpc` 进程，Rust Session/Job 状态因此可以跨多次 RPC 保持。Worker 是 XiaoYu Runtime 的内部基础设施，不是新的用户产品或独立权限层。Host 仍负责身份、RBAC、审批、审计与 Domain Tool；Rust 负责 Agent Runtime state/native primitives。Worker 重启意味着易失 Session/Job state 丢失，Host 必须重新观察真实系统状态。

## 13. 0.2.15 Native Terminal CI 收敛边界

Native Terminal 的完成态必须同时具备：Rust 格式、平台编译、真实 PTY/ConPTY integration、workspace tests。CI 不允许因为 rustfmt 失败就跳过后续编译/集成测试，否则会形成串行盲区。

0.2.15 只修正证据链与 Runner 报出的格式差异，不扩大 Terminal 权限。Go Host 的 identity/RBAC/approval 仍是唯一授权源，Rust 只执行已授权的 native primitive。

## 12. 0.2.14 Windows ConPTY 边界

Windows backend 使用 `CreatePseudoConsole` 创建 ConPTY，通过 `PROC_THREAD_ATTRIBUTE_PSEUDOCONSOLE` + `CreateProcessW` 连接子进程，使用 `ResizePseudoConsole` 更新 rows/cols。它与 Linux PTY 共用 `terminal/start|get|list|write|output|resize|close`，Snapshot 仅通过 `backend` 暴露实现差异。

ConPTY 不能绕过 Host：Terminal start、每次 write、每次 resize 都必须携带已由 Go Host 审批生成的授权状态。Windows 完成态必须由独立 `windows-latest` Rust CI 实际 check/test 后确认。

## 11. 0.2.13 Linux Native PTY 边界

`xiaoyu-core` 继续拥有唯一的长期 Terminal 生命周期：`start/get/list/write/output/resize/close`。Terminal stdin 可多次写入，因此**每个输入 frame 和每次 resize 都必须重新经过 Host authorization**。输出采用有界缓冲与 cursor 读取，工作目录继续受 Runtime Root / Session scope 约束。

Linux backend 现在是真实 `linux-pty-v1`：Rust 创建 PTY master/slave、建立新 session 和 controlling TTY，并允许更新 kernel window size。Linux tests 必须证明 stdin 是 TTY，并验证 `stty size` 随 resize 改变。非 Linux backend 当前仍是明确的 `stdio-pipe-v1` fallback。尤其 Windows ConPTY 尚未在 0.2.13 声明完成；后续实现必须复用同一 Terminal contract。
