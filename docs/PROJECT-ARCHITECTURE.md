# AI Game Manager Panel 0.2.14 总架构

> AGMP 是唯一产品。用户与 XiaoYu 是“两个大脑、同一副身体”：用户拥有最终授权与接管权；XiaoYu 负责理解、规划、执行编排、验证、恢复与总结。0.2.9 起语言职责正式冻结为 **Rust-first Agent Runtime / Go Domain Host / Vue UI**。

## 1. 产品模型

```text
                           用户
                            │
                 ┌──────────┴──────────┐
                 │                     │
            XiaoYu Agent            Human UI
                 │                     │
                 └──────────┬──────────┘
                            │
                       AGMP Product
                            │
          ┌─────────────────┴─────────────────┐
          │                                   │
 Rust XiaoYu Agent Runtime              Go Domain Host
          │                                   │
 Agent / Tool / Session                 Game / Steam / Instance
 Native Runtime / Sandbox               Auth / Update / Web API
          │                                   │
          └─────────────────┬─────────────────┘
                            │
                         OS / Game
```

永久规则：

- **同一产品，不同运行端**：Web / Wails / Electron / Linux Server / Docker / 未来 Rust Native 都只是完整 AGMP 的不同交付目标，不拆成独立 XiaoYu 产品。
- **Human override always wins**：暂停、接管、取消优先于 XiaoYu 自动执行。
- **同一副身体**：人工 UI 与 XiaoYu 不能维护两套业务实现。
- **Server-owned Run**：浏览器断线不取消后台 Run。
- **多端 AI parity**：Web / Wails / Electron / Linux Server / Docker 必须共享同一个 XiaoYu 能力模型。
- **Headless first**：删除桌面壳后，Go Core + Rust XiaoYu + Web 仍应工作。

## 2. Language Ownership

详细规范：[`architecture/LANGUAGE-OWNERSHIP.md`](architecture/LANGUAGE-OWNERSHIP.md)。

### 2.1 Rust = XiaoYu Agent Runtime

Rust 长期负责：

```text
Brain Policy
Agent Loop / Goal State
Tool Search / Capability Discovery
Context / Session / Thread
Long-running Jobs
PTY
Generic Shell / Process / Filesystem
Apply Patch
Sandbox / Capability Lease
Reflection / Experience
Subagent / Specialist Dispatch
xiaoyu.v1 Protocol
```

Rust 的目标不是复制游戏业务，而是成为 XiaoYu 可跨 Windows/Linux/Web Host 复用的通用 Agent Runtime 与 Native Security Boundary。

### 2.2 Go = AGMP Product Host / Domain Services

Go 负责：

```text
Minecraft / DST / future games
Steam / SteamCMD
Instance / Workspace
Backup / Logs / Tasks
Environment / Update
Auth / RBAC / License / Settings
Web API
Wails current desktop host
Domain Tool Provider
```

Go 的业务 Service 是 Domain 事实来源。Rust 需要安装 DST、管理 Minecraft 或更新 Steam App 时，应调用 Go Domain Tool，而不是复制这些业务逻辑。

### 2.3 Vue / TypeScript = Presentation / UX

Vue 统一承载 Web/Wails/Electron UI：XiaoYu、Dashboard、设置、实例、日志、游戏工作台、审批和 Trace。前端不能成为权限或业务状态权威。

## 3. 当前迁移状态

0.2.8 已建立全绿 CI，因此 0.2.9 不进行“大爆炸式重写”。当前仍有成熟 Go 实现：

```text
internal/xiaoyu/host       # 当前 Agent Host / Run / provider transport
internal/platform/runtime  # 当前 Go Process/Session runtime
internal/ops/files         # 当前 Go workspace file boundary
```

从 0.2.9 起这些代码进入**兼容迁移期**：

1. 不再默认把新的通用 Agent Runtime 能力加入 Go；
2. 新 Tool Search / Session / Job / PTY / Sandbox / Reflection / Subagent 优先 Rust；
3. 每迁移一块必须先有稳定协议与 Agent Bench；
4. 新旧实现短期可并存，但只能有一个权威路径；
5. 迁移完成并验证后再删除旧路径。

因此，“Rust-first”表示**新能力归属与迁移方向**，不是宣称 0.2.9 已经把所有 Go Runtime 重写完成。

## 4. 根目录

```text
AI-Game-Manager-Panel/
├─ cmd/              # Go 程序入口
├─ configs/          # 非敏感产品配置
├─ desktop/          # Desktop Shell / compatibility shell
├─ distribution/     # Installer / Docker / License 发行资源
├─ docs/             # 当前规范与唯一项目历史
├─ frontend/         # Vue 3 + TypeScript
├─ internal/         # Go Product Host / Domain Services
├─ runtime/          # 用户可写运行数据（不进 Git）
├─ rust/             # XiaoYu Agent Runtime
└─ scripts/          # 开发、Gate、构建、发行工具
```

普通功能不得新增根级源码目录。

## 5. Go Domain 骨架

```text
internal/
├─ app/                  # 应用装配/生命周期
├─ bridge/               # HTTP / Wails Adapter
├─ config/               # configs 加载
├─ xiaoyu/               # 迁移期 Go Host / Bridge，不是长期第二套 Runtime
│  ├─ contract/          # Domain Tool 契约
│  ├─ control/           # Host 身份/审批桥
│  ├─ host/              # 当前兼容 Agent Host；新通用语义不再扩张
│  └─ runtime/           # Go <-> Rust Runtime Bridge
├─ games/                # 游戏差异
├─ server/               # 通用服务器域
├─ ops/                  # Files / Logs / Backup / Tasks
├─ deploy/               # Environment / Steam / Updater
├─ system/               # Auth / License / Settings / Security
└─ platform/             # Go Domain 底层基础设施
```

依赖规则：

1. `platform` 不反向依赖上层业务域。
2. `games` 不复制通用文件/网络/进程/Steam 基础设施。
3. `app` 只装配公开 Service。
4. `bridge` 只做协议适配/鉴权入口。
5. `internal/xiaoyu/host` 不新增长期 Planner/Tool Search/Reflection/Subagent 等能力。
6. Domain Tool Handler 必须调用所属 Go Service，不把业务实现写进 XiaoYu Bridge。

## 6. Rust Runtime 骨架

0.2.9 继续保持两个 workspace crate，避免过早拆成十几个包：

```text
rust/
├─ Cargo.toml
├─ Cargo.lock
└─ crates/
   ├─ xiaoyu-core/
   │  └─ src/
   │     ├─ lib.rs
   │     ├─ tool_search.rs      # Tool Search
   │     ├─ session.rs          # 0.2.10 Session Registry
   │     └─ jobs.rs             # 0.2.10 Long-running Jobs
   │     └─ bin/xiaoyu.rs
   └─ xiaoyu-protocol/
      └─ src/lib.rs
```

逻辑可以继续按模块拆文件；只有真正需要独立发布、权限边界或生命周期时才拆 crate。

### 6.1 Rust Native Tool 安全边界

未来 Rust Shell/File/Process/PTY 不等于“裸 OS 权限”。执行链必须保持：

```text
Model Decision
   ↓
Rust Agent Runtime
   ↓
Capability / Policy
   ↓
Host identity + RBAC + approval context
   ↓
Sandbox / path / argument boundary
   ↓
Execution
   ↓
Audit + Observation + Verification
```

三种审批模式（请求批准 / 帮我批准 / 完全访问权限）继续由用户决定最终自动化级别。Full 只减少交互审批，不取消硬安全检查。

## 7. Domain Tool 与 Generic Tool

两种能力长期并存：

### Domain Tool

例如：

```text
environment.install_java
environment.remove_runtime
dst.cluster.stop
backup.create
instance.start
```

由 Go Domain Service 提供，结构化、可验证、优先使用。

### Generic Tool

例如未来 Rust Runtime 的：

```text
shell.exec
fs.read / fs.write / fs.replace
process.run
pty.open
job.start
patch.apply
```

用于未知程序、第三方 CLI、异常诊断和 Domain Tool 覆盖不到的现实情况。

原则：**Domain Tool 优先，不是能力上限。**

## 8. Desktop

当前 Windows 主桌面保持：

```text
Wails + Vue + Go Host + embedded Rust XiaoYu Runtime
```

Electron 保持兼容桌面方案。

不为了 GitHub Rust 百分比立即切 Tauri。未来当 Rust Runtime 足够完整、桌面 Bridge 足够薄且迁移收益明确时，再单独评估 Tauri / Rust Native Shell。

## 9. 前端

```text
frontend/src/
├─ app/
├─ features/
│  ├─ auth/
│  ├─ dashboard/
│  ├─ deployment/
│  ├─ games/
│  ├─ instances/
│  ├─ logs/
│  ├─ nodes/
│  ├─ settings/
│  ├─ terminal/
│  ├─ users/
│  └─ xiaoyu/
└─ shared/
```

planned 功能使用 `configs/modules.json` + `ModulePlaceholderView.vue`，不创建空源码目录。

## 10. 数据与发行

- 用户可写数据进入根 `runtime/`，不得提交 GitHub。
- 普通用户只获得预编译完整 AGMP；不要求安装 Rust/Cargo/MSVC/Go/Node/pnpm。
- XiaoYu Runtime 可以是内部子进程，但不作为独立用户产品发布。
- Release Key、API Key、Token、真实服务器数据永远不进入公开仓库。

## 11. 可复现构建

0.2.9 正式提交并冻结：

```text
go.mod / go.sum
rust/Cargo.lock
frontend/pnpm-lock.yaml
desktop/electron/pnpm-lock.yaml
```

CI 必须使用 `cargo --locked` 与 `pnpm --frozen-lockfile`；Go module graph 若被 `go mod tidy` 改动，CI 必须失败。

## 12. AI 开发代理读取顺序

处理架构/AI Runtime 任务时优先读取：

1. `AGENTS.md`
2. `docs/architecture/LANGUAGE-OWNERSHIP.md`
3. `docs/PROJECT-ARCHITECTURE.md`
4. `docs/architecture/XIAOYU-CORE.md`
5. `docs/development/XIAOYU-AGENT-RUNTIME.md`
6. `docs/DEVELOPMENT-PLAN.md`
7. `docs/PROJECT-HISTORY.md`（只用于历史）



## 0.2.10 Rust Session / Job Runtime

0.2.10 adds the first stateful Rust Native Runtime layer. A persistent `xiaoyu rpc` process can own Session and Job state, start Host-authorized native jobs, expose bounded output and cancel them. This is deliberately staged before model-visible migration: Go remains the authority for user identity, RBAC and the three approval modes, and `shell.exec` is not switched to Rust until the Host has a supervised persistent RPC worker.

## 0.2.11 Persistent Rust Runtime Worker

```text
Vue / Web / Wails
       ↓
Go AGMP Host
  ├─ Identity / RBAC / Approval / Domain Services
  └─ supervised stdio worker
             ↓
      Rust `xiaoyu rpc`
       ├─ Tool Search
       ├─ Brain policy
       ├─ Session Registry
       └─ Long-running Jobs
```

0.2.11 后 Go 不再为每个 XiaoYu RPC 启动一次 Rust 进程。长期 Worker 保持 stateful Runtime，同时在超时/断管时被回收。模型可见 `shell.exec` 仍未直接迁移；下一步只迁移已审批的长任务。

## 0.2.14 Windows ConPTY / Cross-platform Native Terminal

0.2.14 不改变上层 Terminal contract，而是在 Rust Runtime 后端补齐 Windows Native Terminal：Linux 使用 `linux-pty-v1`，Windows 使用 `windows-conpty-v1`。Windows backend 通过 ConPTY 与扩展进程启动属性连接子进程，并支持同一 `terminal/resize` API。

平台 Native Terminal 仍位于 Rust XiaoYu Runtime / Security Boundary 内；Go Host 继续负责身份、RBAC 与三种审批模式。任何 start、input、resize 都必须在 Host 侧授权后才能进入 Rust。独立 Windows Rust CI 必须实际编译并运行 ConPTY integration，避免“接口存在但 Windows 根本不能工作”的假完成。

## 0.2.13 Linux Native PTY Foundation

```text
Approved Host action
       ↓
Go XiaoYu Runtime Bridge
       ↓
Persistent `xiaoyu rpc` Worker
       ↓
Rust TerminalManager
  ├─ start / get / list
  ├─ write       (每次输入重新要求 Host authorization)
  ├─ output      (bounded + cursor)
  ├─ resize      (Host authorization)
  └─ close
       ↓
Linux: native PTY master/slave + controlling TTY
Other: explicit stdio-pipe fallback
```

0.2.13 不改变 Terminal 上层协议，而是把 Linux backend 升级成真实 `linux-pty-v1`。TTY detection 与 window resize 由 Rust integration tests 验证。Windows 本版仍明确为 `stdio-pipe-v1`，不把未验证的接口包装成 ConPTY 完成态；下一阶段先建立 Windows Rust Runtime CI，再在相同 contract 后面实现 ConPTY。
