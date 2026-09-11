# AI Game Manager Panel 0.1.88 总架构

> AGMP 是唯一产品；0.1.88 将产品控制模型冻结为 **Human + XiaoYu，两个大脑、同一副身体**。Rust XiaoYu 是唯一智能大脑，用户是最终人为大脑；两者通过同一个 AGMP Host、同一组 Domain 能力和同一个 Shared Runtime 控制游戏服务器。Web/Linux/Docker 与 Windows Desktop 必须保持核心 AI 能力等价。

## 1. 永久核心关系

**Web / Wails / Electron / Rust Native：同一产品，不同运行端。**

```text
                           用户
                            │
                  ┌─────────┴─────────┐
                  │                   │
               XiaoYu AI           Human Control
               智能大脑             人为大脑
                  │                   │
                  └─────────┬─────────┘
                            │
                   Domain Tool / API
                            │
      ┌─────────┬───────────┼───────────┬──────────┐
      │         │           │           │          │
    games     server       ops        deploy     system
      └─────────┴───────────┼───────────┴──────────┘
                            │
                         platform
                            │
                OS / Steam / Game Server
```

- **Rust = 小鱼 Intelligence Core。** Brain / Planner / Workflow / Context / Memory / Tool 选择 / Result Validation 归 Rust；它只思考和编排，不直接碰 OS/业务数据。
- **Go = AGMP 业务身体 + 安全执行边界。** 游戏、服务器、运维、部署、系统能力由 Go 提供稳定领域 API；Host Tool Registry、身份/RBAC/审批/审计和最终 Tool Handler 都在 Go 强制执行。
- **Vue + TypeScript = 统一 UI。** Web/Wails/Electron 复用同一前端。
- **Python = 可选 Sidecar。** 不进入小鱼核心硬依赖。


## 1.1 双大脑、一副身体与多端原则

```text
Human UI / Manual Control        XiaoYu Rust Brain
          \                         /
           \                       /
            └────── AGMP Host ─────┘
             Identity / RBAC / Approval
             Tool / Capability / Audit
                       │
              Domain Services
                       │
              Shared Platform Runtime
                       │
             Windows / Linux / Game
```

硬规则：

- **Human override always wins**：用户暂停、接管或取消时，XiaoYu 当前可取消推理回合立即停止，直到用户明确交还控制权。
- **同一副身体**：人工按钮与 XiaoYu Tool 不允许维护两套业务实现；必须汇入同一 Domain Action，再进入同一 Shared Runtime。
- **Server-owned Run**：自主任务属于 AGMP Server，不属于某个浏览器标签页；Web 断线只取消 Event Subscription，不取消 Run。
- **Web AI parity**：Linux Server Web / Docker Web / Windows Web 与桌面端共享同一 XiaoYu、模型中心、Harness、Tool Host、审批和事件流。桌面仅允许托盘、窗口等真正 Desktop-only 能力例外。
- **多用户隔离**：Run 记录 Initiator；Owner/Admin 可全局监督，普通 Operator 默认只读取/控制自己创建的 Run。全局模型/插件管理事件不向普通用户 Event Stream 广播。
- **Headless first**：删除 `desktop/` 后，AGMP Core + XiaoYu + Web 仍必须能够独立构建和运行。

## 2. 根目录

```text
AI-Game-Manager-Panel/
├─ cmd/              # Go 程序入口
├─ configs/          # 非敏感产品配置
├─ desktop/          # Desktop Shell
├─ distribution/     # Installer / Docker / License 发行资源
├─ docs/             # 集中文档
├─ frontend/         # Vue 3 + TypeScript
├─ internal/         # Go 领域业务与平台能力
├─ runtime/          # 用户可写运行数据
├─ rust/             # XiaoYu Intelligence Core
└─ scripts/          # 开发、Gate、构建、发行工具
```

普通功能不得新增根级源码目录。

## 3. Go 骨架

```text
internal/
├─ app/                      # 应用装配/生命周期
├─ bridge/                   # HTTP / Wails Adapter
├─ config/                   # configs 加载
│
├─ xiaoyu/                   # Go <-> XiaoYu 边界，不是第二个 Brain
│  ├─ contract/              # Tool/Plan/Provider/Registry 契约
│  ├─ control/               # 审批状态/权限策略桥
│  └─ runtime/               # Rust Runtime 进程桥
│
├─ games/                    # 游戏差异
│  ├─ common/
│  ├─ dst/
│  │  ├─ dedicated/
│  │  ├─ logcenter/
│  │  ├─ preferences/
│  │  ├─ runtime/
│  │  ├─ setup/
│  │  └─ workspace/
│  └─ steam/
│     └─ dst/
│
├─ server/                   # 通用服务器域；当前真实代码为 workspace
│  └─ workspace/
│
├─ ops/                      # 运维域
│  ├─ files/                 # 工作区文件安全读取基础
│  └─ logs/
│
├─ deploy/                   # 部署域
│  ├─ environment/
│  ├─ steam/
│  │  └─ maintenance/
│  └─ updater/
│
├─ system/                   # AGMP 系统域
│  ├─ auth/
│  ├─ license/
│  └─ settings/
│
└─ platform/                 # 最底层共享基础设施
   ├─ files/
   ├─ http/
   ├─ net/
   ├─ os/
   ├─ runtime/
   ├─ security/
   ├─ steam/
   └─ telemetry/
```

### 3.1 Application 编排文件

`internal/app` 保持一个 Go package，不增加新的中间层；原 1397 行 `app.go` 按统一前缀收为 `app.go / app_core.go / app_games.go / app_license.go / app_xiaoyu.go / app_auth.go / app_environment.go`。领域 Service 仍在各自目录，`app_*` 只负责装配和对外编排。

### 3.2 为什么没有大量 skeleton 目录

规划能力由 `configs/modules.json` 与 `configs/games.json` 记录。0.1.83 起，`internal` 子目录尽量代表“已经有真实源码”，不再为了 roadmap 创建只有 `doc.go` 的空包。一级领域根可保留 `doc.go` 作为依赖边界声明。

## 4. XiaoYu Rust Core

```text
rust/
└─ crates/
   ├─ xiaoyu-core/
   │  ├─ src/lib.rs
   │  └─ src/bin/xiaoyu.rs
   └─ xiaoyu-protocol/
      └─ src/lib.rs
```

当前坚持两个 crate，不把 Brain/Planner/Context/Memory/Workflow 拆成十几个项目。只有独立发布或独立生命周期真正成立时才允许拆 crate。Rust Core 是 Brain-only；可执行 Tool 由 Go Host 提供。

Go 侧只保留：

```text
internal/xiaoyu/contract
internal/xiaoyu/control
internal/xiaoyu/runtime
```

禁止新增 Go `brain/`、`planner/`、`memory/` 来形成第二个大脑。

## 5. 服务器终端

终端是公共控制入口，不属于某一游戏，也不是 XiaoYu 的裸 Shell 后门。

最终目标依赖链：

```text
小鱼 / 人工 UI
      ↓
Server Domain
      ↓
Platform Runtime
      ↓
Process / PTY / stdin / stdout
      ↓
Game Server
```

0.1.83 已先解决 UI 重复：真正可执行受控命令的 `TerminalPanel.vue` 已嵌入 Bottom Panel，旧的独立死页面删除；`/terminal` 只负责打开 Bottom Panel。

底层公共 Process/Terminal Session 已由 DST 实际复用。0.1.85 进一步加入 stdout/stderr 来源分离、串行 stdin、环境变量 Overlay、状态快照、命名 Session Manager、受限一次性 Run、超时取消、异步回收、Unix 进程组终止、非阻塞输出订阅、单调 Sequence 与固定容量环形历史。PTY/ConPTY 与跨应用重连恢复仍属于高级增强，但不会再改变共享 Runtime 的职责边界。

### 5.1 通用 Process / Terminal Runtime

`internal/platform/runtime` 是唯一进程创建边界：`Session` 负责长生命周期控制台，`Run` 负责有输出上限/超时取消的一次性内部命令，`RunShell` 是显式受控 Shell，`StartDetached` 负责无需交互但必须正确回收的外部启动；`Manager` 管理命名 Session。环境变量默认继承父进程并按调用方 Overlay，stdout/stderr 保留来源；Session 提供非阻塞订阅和固定容量历史。XiaoYu Runtime、Updater 与系统打开器都复用这一底座；Gate 同时禁止 Go 业务模块绕过它以及 Rust XiaoYu Brain 自建 Process/File 执行。


## 5.2 XiaoYu Host Tool 边界

```text
Rust XiaoYu Brain
        ↓ Tool request
internal/xiaoyu/contract.Registry
        ↓
Go identity / permission / approval / audit
        ↓
Domain Handler (ops / server / deploy / system / games)
        ↓
platform/runtime / files / net / ...
```

Rust Core 不直接包含 `std::process` / `std::fs` 的 Host 执行实现。系统、文件、Shell 等真实能力都由 AGMP Host Registry 提供；0.2.2 起 `shell.exec` 是 XiaoYu 可见的通用后备 Tool，`process.run` 仅保留为人工/兼容别名。小鱼优先使用结构化 Domain Tool，Domain Tool 不足时允许转向通用能力；是否执行由 RBAC、Step-up 与三种审批模式决定。Tool Registry 拒绝重复名称，防止后注册模块覆盖/劫持现有能力。

## 6. 前端

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
   └─ components/ModulePlaceholderView.vue
```

planned 功能不创建空 Feature 目录。Router 直接使用统一 `ModulePlaceholderView` + `configs/modules.json` 展示规划能力。

## 7. 依赖硬规则

1. `platform` 不得依赖 `xiaoyu/games/server/ops/deploy/system`。
2. `games` 只描述游戏差异，不复制通用 Process/File/HTTP/Steam。
3. `server` 不硬编码某款游戏分支；游戏通过 Adapter/Contract 提供差异。
4. `ops/deploy/system` 跨域必须通过公开能力，不访问其他域私有状态。
5. `app` 只装配和协调公开 Service，不发展成业务实现仓库。
6. `bridge` 只做协议转换/鉴权入口，不写业务状态机。
7. XiaoYu 只通过注册 Tool / Domain API 操控 AGMP，不能用 `process.run` 绕过已存在的系统能力。
8. 单个未来规划点不能成为新目录理由；禁止恢复 `internal/service`、`internal/core`、`internal/games/dst/service`。

## 8. 数据与发行边界

所有用户可写数据继续进入 `runtime/`。正式用户只获得预编译完整 AGMP；不会被要求安装 Rust、Cargo、MSVC、Go、Node 或 pnpm。XiaoYu Runtime 可以作为内部子进程隔离，但只属于完整 AGMP，不作为单独用户产品。
