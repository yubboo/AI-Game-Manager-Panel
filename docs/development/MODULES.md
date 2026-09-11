# AGMP 0.1.88 模块边界

> 0.1.83 冻结领域骨架、0.1.85 冻结 Shared Runtime；0.1.88 在不搬主骨架的前提下增加 XiaoYu Host/Harness、模型中心、双大脑控制权和 Web/Linux AI Parity。模块是否规划存在仍由 `configs/modules.json` 描述；源码目录只为真实实现创建。

## 1. 一级领域

**一个业务模块一个明确归属。** 页面位置不改变业务归属；强关联能力优先在同域维护。

| 领域 | 后端主归属 | 负责内容 | 禁止内容 |
| --- | --- | --- | --- |
| 小鱼 | `rust/crates/xiaoyu-core` + `internal/xiaoyu` | Rust Brain；Go 侧 Contract / Control / Runtime Bridge | Go 侧重建 Planner/Brain；复制业务实现 |
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

## 3. XiaoYu 唯一大脑

```text
rust/crates/xiaoyu-core       = 唯一 Intelligence Core / Brain
rust/crates/xiaoyu-protocol   = 稳定协议
internal/xiaoyu/contract      = Go Tool 契约与唯一可执行 Registry
internal/xiaoyu/control       = Go 审批状态与权限桥接
internal/xiaoyu/host          = Plugin/Capability Host、Run 生命周期、模型通信与安全驱动（不是第二个 Brain）
internal/xiaoyu/runtime       = Go <-> Rust Brain Runtime Bridge
```

**AI 不直接接管模块私有数据。** 小鱼只能通过公开 Tool / Service / API 使用领域能力。`internal/xiaoyu` 不得再增加 `brain/`、`planner/`、`memory/` 等第二套智能核心。

固定调用链：

```text
用户
  -> 小鱼 Rust Core
  -> Tool 选择 / 计划
  -> AGMP Host Tool Registry
  -> Go Permission / Approval / Audit
  -> Domain Tool / Service
  -> Platform
  -> OS / Steam / Game Server
  -> Result Validator
  -> 小鱼继续规划或完成
```

`shell.exec` 是 0.2.2 起提供给 XiaoYu 的受控通用后备 Host Tool；`process.run` 只保留为人工/兼容别名并维持 `XiaoYu=false`。小鱼应优先使用已有 Domain Tool，但不能把“没有专用 Tool”误判为无能力；Shell 的实际执行仍经过身份、RBAC、Step-up、三种审批模式和审计。Rust Brain 不自行实现 Shell。

## 3.1 Host Tool Registry

可执行 Tool 的权威目录是 `internal/xiaoyu/contract.Registry`，不是 Rust 本地 Tool Registry。它负责唯一名称、风险级别、可调用标记与 Handler 映射；重复名称直接失败，禁止静默覆盖。

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

`internal/platform/runtime` 统一管理所有真正的进程创建与 stdio：长生命周期使用 `Session`，一次性内部命令使用 `Run`，受控 Shell 使用 `RunShell`，非交互外部启动使用 `StartDetached`，多会话由 `Manager` 管理。它负责 PID、状态快照、stdout/stderr 来源、串行 stdin、环境变量 Overlay、超时取消、退出回收、非阻塞订阅、Sequence、固定容量历史以及平台终止策略。DST 和 XiaoYu Runtime 已复用；业务域禁止直接 `exec.Command` / `StdinPipe` / `StdoutPipe` / `StderrPipe`，Rust Brain 也禁止 `std::process/std::fs`。PTY/ConPTY 与跨应用重连属于后续高级能力，但不得另造第二套 Process Core。
