# AGMP Language Ownership

> 本文定义 AI Game Manager Panel 的长期语言职责边界。它不是“哪门语言更高级”的比较，而是为了让人类与 AI 开发代理在新增代码时，第一时间知道代码应该放到哪里。


## 0.2.21 Capability Lease Ownership

- 0.2.20 的 Approved Agent → Native Terminal / Linux PTY / Windows ConPTY 稳定基线保持不变。
- 0.2.21 在 Go Host 审批边界之后签发内存态、短时、精确指纹、Run/principal 绑定的单次 Capability Lease。
- Native Terminal 启动前必须消费租约；Go→Rust `terminal/start` 必须携带 `capabilityLeaseId`，Rust 继续 fail-closed。
- Capability Lease 不持久化原始命令或秘密；本阶段不宣称已经实现完整 OS namespace/container Sandbox。

## 1. 三层长期边界

规范化职责声明（供人类、AI 与 CI 共同读取）：

- **Rust = XiaoYu Agent Runtime / Native Execution / Security Boundary**
- **Go = AGMP Product Host / Game Domain Services**
- **Vue / TypeScript = UI / Presentation / UX**

```text
Vue / TypeScript
    ↓
Presentation / UX

Go
    ↓
AGMP Product Host / Game Domain Services

Rust
    ↓
XiaoYu Agent Runtime / Native Execution / Security Boundary
```

### Vue / TypeScript

负责用户交互与展示：

- Dashboard、设置、日志、实例、游戏工作台；
- XiaoYu Chat / Run / Approval / Trace UI；
- Wails / Electron / Web 共用前端；
- 只展示和请求能力，不成为权限、业务规则或 Agent 决策权威。

### Go

负责 AGMP 产品业务与领域服务：

- 游戏服务器：Minecraft / DST / 后续游戏 Adapter；
- Steam / SteamCMD / Environment / Instance / Backup / Update；
- Web API、认证、组织、许可证、配置与业务持久化；
- Domain Tool Provider：把稳定业务能力以结构化 Tool 暴露给 XiaoYu；
- Wails 当前桌面 Shell 与产品生命周期。

Go 的价值是成熟的网络/并发/服务生态、清晰的业务代码和简单部署，不是“AI 主语言”。

### Rust

负责 XiaoYu 的通用 Agent Runtime：

- Brain Policy / Decision Grammar；
- Agent Loop / Goal State / Recovery；
- Tool Search / Capability Discovery；
- Context / Session / Thread；
- Long-running Job / PTY；
- 通用 Shell / Process / Filesystem Runtime；
- Apply Patch；
- Sandbox / Capability Lease / Native Security Boundary；
- Reflection / Experience Pipeline；
- Subagent / Specialist Dispatch；
- XiaoYu Protocol 与跨端可复用 Native Runtime。

Rust 不是为了 GitHub 语言占比而使用。只有真正属于 Agent 通用运行时、Native 执行或安全边界的能力才进入 Rust。

## 2. Domain 与 Agent Runtime 的判定规则

新增功能先问：

> 这是“游戏/产品业务能力”，还是“任何 Agent 都可能需要的通用能力”？

### 应放 Go 的例子

```text
安装 DST Dedicated Server
解析 cluster.ini
管理 Minecraft 实例
Steam App 更新策略
备份游戏存档
AGMP 用户/RBAC/License
Updater 业务规则
```

### 应优先放 Rust 的例子

```text
Agent 下一步决策协议
Tool Search
PTY Session
后台 Job
通用 Shell 执行器
通用文件修改 / Apply Patch
Sandbox
命令取消与超时
Agent Context Compaction
Reflection
Subagent
```

## 3. 调用关系

长期目标：

```text
User
  ↓
Vue / Web / Desktop
  ↓
XiaoYu Rust Agent Runtime
  ├─ Provider-facing Agent semantics
  ├─ Tool Search / Session / Context
  ├─ Generic Native Tools
  └─ Sandbox / Approval Enforcement
         │
         ├──────── Generic capability ───────→ OS
         │
         └──────── Domain capability ────────→ Go AGMP Domain Provider
                                                ↓
                                         Game / Steam / Instance / Update
```

Go Domain Service 仍是游戏业务事实来源；Rust 不复制 DST/Minecraft/Steam 业务。

## 4. 安全模型

Rust 的内存安全不能替代产品授权。XiaoYu 安全必须同时依赖：

```text
Rust memory safety
+ capability scope
+ sandbox
+ path boundary
+ command policy
+ approval
+ RBAC
+ audit
+ execution verification
```

- 用户的三种审批模式继续是最终授权入口；
- Go Product Host 负责用户身份、RBAC、License 与 Domain 权限；
- Rust Native Runtime 负责通用 Agent Tool 的执行边界与 Sandbox；
- 高风险动作必须携带 Host 授权上下文，不能因为进入 Rust 就绕过审批；
- Domain Tool 与通用 Tool 都必须可审计、可取消、可验证。

## 5. 当前迁移现实

0.2.9 是 **Rust-first Agent Runtime 架构冻结点**，不是一次性重写。

当前已有大量稳定 Go Runtime/Host 代码，例如：

```text
internal/xiaoyu/host
internal/platform/runtime
internal/ops/files
```

这些代码暂时继续工作，保证 0.2.8 已经全绿的产品基线不被推倒。

从 0.2.9 起：

1. 不再把新的通用 Agent Runtime 能力默认堆进 Go；
2. 新的 Tool Search / Session / Job / PTY / Sandbox / Reflection / Subagent 优先进入 Rust；
   - 0.2.9：Tool Search；
   - 0.2.10：Session Registry + Long-running Job primitives；
3. 旧 Go Agent 机制按测试覆盖逐步迁移，不允许“大爆炸式重写”；
4. 每迁移一块，必须先建立协议和 Agent Bench，再删除旧实现；
5. Wails 桌面 Shell 暂时保留，不因为 Rust 路线立刻切换 Tauri。

## 6. Desktop 原则

当前 Windows 主桌面仍是：

```text
Wails + Vue + Go Host
```

Electron 是兼容壳。

Rust 作为 XiaoYu 内部 Runtime 随完整 AGMP 一起发行。未来只有当 Rust Runtime 已经足够完整、Tauri 迁移收益明显且 Wails Bridge 足够薄时，才重新评估 Desktop Shell；禁止为了“Rust 百分比”重写已经稳定的桌面层。

## 7. AI 开发代理硬规则

所有 AI 在新增 XiaoYu 代码前必须：

1. 阅读本文；
2. 判断功能是 Domain 还是 Agent Runtime；
3. Agent 通用能力默认 Rust-first；
4. Domain 业务默认 Go；
5. UI 默认 Vue/TypeScript；
6. 不因迁移目标复制两套长期实现；
7. 不为了 Rust 占比迁移稳定 Go 业务；
8. 不把 Rust 当成绕过 Approval/RBAC 的“更高权限层”。
9. Session / Job / PTY 属于 Rust Runtime；Domain Service 不得复制一套 Agent Job Core。

### 0.2.11 增量迁移

- Rust：Tool Search + Session Registry + Long-running Jobs + persistent stdio Runtime process。
- Go：监督 Rust Worker 生命周期并继续提供 Product/Domain authority。
- 下一步：只把已通过 Host Approval 的长任务切到 Rust Jobs，然后再引入 PTY。

### 0.2.20 Approved Agent Native Terminal wiring

- **Go Host owns authorization and routing**：identity、RBAC、step-up、approval fingerprint、workspace CWD 与 server-owned Run context 在 Go 中完成。
- **Rust owns native execution**：已授权的 Agent `shell.exec` 使用 Rust `terminal/*`，Linux PTY / Windows ConPTY 不复制到 Go。
- **No dual execution**：Agent Native Terminal 失败时不得静默回退到 Go legacy shell；人工/compat 路径仍独立留在 `platform/runtime`。
- `HostAuthorized` 是 Host-internal capability bit，不属于模型、前端或 Tool arguments。

### 0.2.19 Windows ConPTY stdio isolation

Windows `CreateProcessW`/`STARTUPINFOEXW` 的标准句柄隔离继续属于 Rust Native Runtime。Go Host 不复制 Win32 handle 语义，只维持身份、RBAC、审批、审计与 Domain authority。

### 0.2.18 Windows ConPTY Input 收敛

Rust 继续独占 PTY / ConPTY 平台输入语义。Windows ConPTY Enter 使用单个 CR（`\r`）；Linux PTY / fallback 使用 LF。Go Host 不复制 CR/LF 平台判断，只负责身份、RBAC、审批与 Domain Service authority。

### 0.2.17 Windows ConPTY Runtime 收敛

Rust 继续独占 PTY / ConPTY 平台实现。0.2.16 已证明 Windows ConPTY 可以在锁定的 `windows-sys 0.61.2` 下通过真实 `cargo check`；0.2.17 只在 Rust backend 内修正 Win32 cwd 与 CRLF 输入语义，并收紧 native/fallback 编译边界。Go Host 不复制这些平台细节，仍只负责身份、RBAC、审批与 Domain Service。
### 0.2.15 验证收敛

- 语言归属不变：Native Terminal 仍属于 Rust XiaoYu Runtime，Go Host 仍负责授权和 Domain authority。
- 本版不把新 Agent 通用能力塞回 Go，也不新增 Rust Domain 业务。
- 双平台 Native Terminal 必须先通过真实 Linux PTY / Windows ConPTY integration，再继续 Agent wiring / Sandbox；静态结构 Gate 不能替代平台运行证据。
- Rust CI 必须继续后续 check/integration/tests，即使 rustfmt 已经让 Job 标记失败，以减少验证迭代盲区。

### 0.2.14 增量迁移

- Linux Native PTY 继续归 Rust XiaoYu Runtime。
- Windows Terminal backend 从 stdio fallback 升级为 ConPTY，并由独立 Windows Rust CI build/integration test 验证。
- `terminal/*` contract 跨平台共用；平台差异只存在 Rust native backend，禁止复制 Agent 业务逻辑。
- start/write/resize 继续由 Go Host 的身份、RBAC、审批决定，Rust 负责被授权后的 native execution。

### 0.2.13 增量迁移

- Rust：Terminal contract 保持不变，Linux backend 升级为真实 `linux-pty-v1`，并新增 Host-authorized resize。
- Go：继续提供身份、RBAC、审批、Domain authority，只通过 Persistent Worker 调用 Terminal RPC。
- Windows：仍明确使用 `stdio-pipe-v1` fallback；ConPTY 只有在专用 Windows Rust CI 实际验证后才能进入 capability。
- 原则：Native Runtime 因真实能力进入 Rust，而不是为了提高 GitHub Rust 百分比；Go Domain Service 不因 PTY 迁移而重写。
