# AGMP 0.2.14 模块边界

> 0.2.9 冻结语言职责：Rust-first XiaoYu Agent Runtime、Go Domain Host、Vue UI。详细边界见 `docs/architecture/LANGUAGE-OWNERSHIP.md`。模块是否规划存在仍由 `configs/modules.json` 描述；源码目录只为真实实现创建。

## 1. 一级领域

**一个业务模块一个明确归属。** 页面位置不改变业务归属；强关联能力优先在同域维护。

| 领域 | 后端主归属 | 负责内容 | 禁止内容 |
| --- | --- | --- | --- |
| 小鱼 | `rust/crates/xiaoyu-core` + `internal/xiaoyu` | Rust Agent Runtime；Go 侧 Domain Contract / Host Bridge | Go 继续扩张通用 Agent Runtime；Rust 复制游戏 Domain |
| 游戏 | `internal/games` | 每款游戏的规则、解析、Adapter、配置差异 | 重写通用进程、文件、Steam、备份能力 |
| 服务器 | `internal/server` | 实例、服务器控制台、RCON、游戏工作区 | 写具体游戏规则 |
| 运维 | `internal/ops` | Files、Logs、Backup、Tasks、Scheduler | 写安装器/Steam 下载流程 |
| 部署 | `internal/deploy` | Deploy、Steam、SteamCMD、Environment、Updater、Build | 写游戏运行时状态机 |
| 系统 | `internal/system` | Auth、License、Settings、Nodes、Network、Security、Plugins 等 | 写游戏专属逻辑 |
| 平台 | `internal/platform` | OS、文件、Process/PTY、网络、HTTP、Telemetry、Steam 底层 | 依赖任何上层业务域 |

## 2. 目录存在的含义

0.1.83 起取消“用 `doc.go` 或 `module.ts` 预占未来模块”的做法：

- `configs/modules.json`：产品功能矩阵、状态、路线图来源；允许 `skeleton`。
- `configs/games.json`：planned 游戏来源；planned 不要求创建 Go 目录。
- `internal/**`：目录原则上代表已经有真实 Go 代码；一级领域聚合包可仅保留 `doc.go` 说明边界。
- `frontend/src/features/**`：只保留真实页面/组件；planned 页面统一由 `ModulePlaceholderView.vue` 根据 `modules.json` 渲染。

这样避免“目录很多，看起来像完成了，实际只有占位文件”。

## 3. XiaoYu Rust-first Agent Runtime

```text
rust/crates/xiaoyu-core       = XiaoYu Agent Runtime 主实现
rust/crates/xiaoyu-protocol   = 稳定协议
internal/xiaoyu/contract      = Go Domain Tool 契约 / 迁移期 Registry
internal/xiaoyu/control       = Go Host 身份/RBAC/审批桥接
internal/xiaoyu/host          = 当前兼容 Host/Run/Provider 层，冻结新增通用 Agent Runtime
internal/xiaoyu/runtime       = Go <-> Rust Agent Runtime Bridge
```

**AI 不直接接管模块私有数据。** 小鱼只能通过公开 Tool / Service / API 使用领域能力。`internal/xiaoyu` 不得再增加 `brain/`、`planner/`、`memory/` 等第二套智能核心。

固定调用链：

```text
用户
  -> XiaoYu Rust Agent Runtime
  -> Tool Search / Plan / Session
  -> Approval / Capability Policy
  -> Generic Native Tool 或 Go Domain Tool
  -> OS / Steam / Game Server
  -> Observation / Validation
  -> XiaoYu 继续规划或完成
```

`shell.exec` 当前仍是 Go Host 提供的受控通用后备能力；0.2.9 起新的通用 Shell/File/Process/PTY/Job 能力优先迁入 Rust Runtime。小鱼仍应优先 Domain Tool，但不能把“没有专用 Tool”误判为无能力。迁入 Rust 后同样必须经过身份、RBAC、三种审批模式、Sandbox、审计和验证。

## 3.1 Capability / Tool Registry

迁移期间 Go `internal/xiaoyu/contract.Registry` 继续是 Domain Tool 的权威目录，负责唯一名称、风险级别与 Handler 映射。Rust Runtime 负责 Tool Search/Capability 选择，并将在后续承接通用 Native Tool Registry。两边都禁止同名静默覆盖。

当前真实 Host Tools 已覆盖系统、工作区文件读写、游戏发现、Steam/环境状态、Runtime 管理、DST Cluster/命令/日志以及受控 `shell.exec`。Tool Schema、风险、来源与 guidance tier 统一由 Host Registry 暴露给 Brain；人工入口与 XiaoYu Tool 优先复用同一领域动作，领域能力不足时再使用通用后备能力。

## 4. DST 归属

0.1.83 删除 `internal/games/dst/service/*` 泛化中间层，强关联代码归回 DST 自己：

```text
internal/games/dst/
├─ dedicated/       # Dedicated Server 发现、配置、Launch Spec 与管理服务
├─ logcenter/       # DST 日志 Store + Service
├─ preferences/     # DST 偏好
├─ runtime/         # 当前 DST 进程/控制台/Cluster Runtime
├─ setup/           # Token/Klei/Preflight/导入编排
└─ workspace/       # DST 工作区聚合
```

DST `runtime/` 当前只保留 DST 专属生命周期、命令、Ready 判定与日志状态；通用进程/stdin/stdout 已迁到 `internal/platform/runtime`，后续游戏必须复用该底座。

## 5. 平台层收口

0.1.83 将一函数一目录的微包收回同一基础设施组：

```text
internal/platform/
├─ files/       # Root、AtomicReplace、OpenFolder
├─ http/        # 外部 URL / 后续 HTTP 原语
├─ net/         # 端口与网络原语
├─ os/          # 系统目录/OS 信息
├─ runtime/     # 已落地通用 Process/Terminal Session；PTY 按后续真实需求扩展
├─ security/    # OS 凭据安全原语
├─ steam/       # Steam 平台底层
└─ telemetry/   # Logger / Metrics 基础
```

`platform` 禁止依赖 `xiaoyu/games/server/ops/deploy/system`。

## 6. 前端收口

真实 Feature 目录当前只保留：

```text
frontend/src/features/
├─ auth/
├─ dashboard/
├─ deployment/
├─ games/
├─ instances/
├─ logs/
├─ nodes/
├─ settings/
├─ terminal/
├─ users/
└─ xiaoyu/
```

- `home/` 已归为 `dashboard/`。
- `servers/` 已归为 `instances/`。
- 未实现模块不再创建无引用 `module.ts`。
- 占位页面统一由 `frontend/src/shared/components/ModulePlaceholderView.vue` 渲染。
- 真正受控终端统一嵌入 Bottom Panel，只有一套 Terminal UI。

## 7. 命名

- 产品：**AI Game Manager Panel / AGMP**。
- 内置 AI：**小鱼 / XiaoYu / xiaoyu**。
- Core：`xiaoyu-core`。
- Protocol：`xiaoyu.v1`。
- 内部二进制：`AI-Game-Manager-XiaoYu(.exe)`。
- 旧开发期产品名只允许出现在历史 release/prompt 文档和必要兼容代码，不允许进入现役 UI/业务代码。

## 8. 拆分原则

强关联代码优先同域维护。新建子目录必须满足至少一个条件：代码规模需要、独立生命周期、独立平台实现、清晰复用边界。单个 `doc.go`、单个 `module.ts` 或“以后可能会做”不能成为新目录的理由。

## 9. 通用进程终端

`internal/platform/runtime` 继续作为 **Go Domain** 的统一进程/stdin/stdout 基座；业务域禁止直接 `exec.Command` / stdio pipe。XiaoYu 的新通用 Native Runtime 从 0.2.9 起优先 Rust，未来 PTY/Jobs/Sandbox 不再默认扩张 Go `platform/runtime`。迁移时必须保持单一权威执行路径，禁止永久维护两套 Process Core。


## 0.2.10 Rust Session / Job Runtime

Rust `xiaoyu-core` 已新增 Session Registry 与 Long-running Job primitives。它们属于通用 Agent Runtime，不属于任何游戏 Domain。当前模型可见 `shell.exec` 仍走 Go Host；在 persistent RPC worker 完成前禁止双写/双执行。未来切换时必须保证同一个已批准动作只有一个权威 Process Runtime。

## 0.2.11 Persistent Runtime Worker

XiaoYu Runtime 新增 Go-supervised Rust Worker：一个 `xiaoyu rpc` 进程承载 Tool Search、Brain、Session、Job state。该 Worker 属于 AI Runtime 基础设施，不创建新产品模块；游戏/部署/实例等 Domain 模块继续由 Go Service 提供。
