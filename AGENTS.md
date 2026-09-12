
## AI 开发阅读顺序

开始较大修改前依次阅读：`docs/PROJECT-STATUS.md` → `docs/PROJECT-ARCHITECTURE.md` → `docs/architecture/LANGUAGE-OWNERSHIP.md` → `docs/development/PROJECT-RULES.md` → `docs/DEVELOPMENT-PLAN.md`。历史细节只在需要时读取 `docs/PROJECT-HISTORY.md`。当前状态与长期规范优先于历史版本描述。

# AI Game Manager Panel AI / 人工开发规范

> 本文件是 AI Game Manager Panel 的强制开发规范。人工开发、ChatGPT、Codex 或其他 AI 修改项目时都必须遵守。

## 1. 项目定位与不可破坏的产品架构

AI Game Manager Panel（AI游戏管理器面板）定位为：**现代化、智能化、AI 驱动的一键游戏服务器部署与管理平台**。

以下是项目的最高级架构约束，优先级高于局部实现便利，后续任何 AI、人工开发、重构和多端适配都不得违反：

1. **AI Game Manager Panel 是唯一产品主体。** Web、Wails、Electron、未来 Rust Native 都只是同一产品的不同构建/运行端，不得拆成彼此独立、各自复制业务的产品。
2. **小鱼是 AGMP 内置的核心大脑。** Agent 负责理解用户意图、规划、调用 Tool、请求审批、执行、验证和汇报；它不是外挂产品、不是单独面向用户下载/启动的应用。即使底层为了稳定性使用 Rust 原生模块或内部子进程，也只能视为 AGMP 的内部实现细节。
3. **两个大脑，同一副身体；AI 辅助人。** 用户/管理员是最终人为大脑，小鱼是持续在线的智能大脑。用户可以手动控制，也可以把目标交给小鱼；Human override 永远优先。两者不得各造一套业务逻辑。
4. **AI 与人工必须调用同一能力层。** Steam、DST、实例、备份、日志、文件、网络、RCON、终端、更新等能力必须复用同一 Core/Service/Tool 契约，禁止“AI 一套实现、手动 UI 再写一套实现”。
5. **多端只做适配，不做业务分叉。** Web / Wails / Electron / Rust Native 的差异只允许存在于 Shell、Bridge、平台适配和打包层；Agent 能力、业务规则、权限、审计、数据模型必须尽量共用。
6. **内部组件不得变成用户心智中的独立产品。** `AI-Game-Manager-XiaoYu.exe`、Electron Go Core 等若因进程隔离而作为内部二进制存在，只允许随完整 AGMP 产物一起打包，不创建独立快捷方式、不单独作为普通用户 Release Asset、不要求用户手工启动。

当前阶段优先级固定为：**核心能力 > 稳定可用 > 性能 > 安全 > 省心运维 > 架构美化/目录重排**。在 Agent 核心闭环和主要服务器能力稳定前，禁止为了“看起来更漂亮”大规模移动根目录或重写总体项目布局。

**语言归属先读规则：** 新增或迁移代码前必须阅读 `docs/architecture/LANGUAGE-OWNERSHIP.md`。Agent 通用能力默认 Rust-first；游戏/产品 Domain 默认 Go；UI 默认 Vue/TypeScript。


### 0.2.22 Capability Scope / Web Asset 硬规则

- Capability Lease `scope` 禁止继续使用任意自由字符串；当前模型执行只允许已注册的 `process.exec:workspace-cwd`。
- `process.exec:workspace-cwd` 只授权一次已批准 process execution，并要求 cwd 经过 AGMP workspace resolver；不得把它解释为 filesystem/network/container 隔离。
- Go Host、Go→Rust bridge、Rust Native Terminal 都必须对 `capabilityScope` fail-closed；`capabilityLeaseId`、`HostAuthorized=true` 任何一个都不能单独替代 scope。
- `cmd/aigame-manager-web/web/assets/` 是 `frontend/dist` 经 `scripts/common/sync-web-assets.mjs` 生成的 hash 资产，必须被 `.gitignore` 忽略。
- 一键推送删除白名单只能精确放行 `cmd/aigame-manager-web/web/assets/`；`web/index.html`、`cmd/aigame-manager-web/main.go` 与其他 `cmd/` 源码继续严格保护。
- 禁止为了“通过推送”把旧 hash 构建资产复制回源码包，也禁止放宽整个 `cmd/` 的删除保护。

### 0.2.21 Capability Lease 硬规则

- server-owned XiaoYu `shell.exec` 进入 Native Terminal 前必须持有 Host 签发的短时 Capability Lease。
- 租约必须在 identity / RBAC / sensitive step-up / Approval 之后签发，并绑定 scope、Tool、RunID、组织/用户 principal 与精确请求指纹。
- Native Terminal 租约默认 30 秒、单次消费；过期、重放、参数变化、跨 Run、跨 principal 必须 fail-closed。
- 租约只保存在内存，Host 重启自动撤销；禁止把原始命令、Token、密码、API Key 写入租约状态。
- Go→Rust `terminal/start` 必须携带 `capabilityLeaseId`；`HostAuthorized=true` 不能替代租约。
- 不得以“Sandbox”名义虚报尚未实现的 OS namespace/container 隔离。

## 1.0 GitHub 基线与固定开发版交付流程

这是所有人工开发者与 AI 开发代理的强制工作流，不需要用户重复提醒。

### 主动查看 GitHub

默认仓库固定为 `https://github.com/yubboo/AI-Game-Manager-Panel.git`，默认分支为 `main`。以下时点必须主动查看 GitHub，而不是等待用户要求：

1. 开始新的版本开发前；
2. 用户说“推好了 / 已推送 / 看看 GitHub”后；
3. 修复 CI 失败前；
4. 准备下一版源码包前。

每次至少确认：`main` 最新 commit SHA/消息、该 commit 对应 GitHub Actions、失败 Job/Step 与日志。下一版修复必须以真实 Runner 结果为依据；本地推测不能覆盖 GitHub 实际证据。

### 固定开发版交付

除非用户明确要求 patch-only，正式开发版一律交付**完整源码包**：

```text
下载 agmp-<version>.zip + SHA-256
        ↓
解压到 H:\一键部署\agmp-<version>
        ↓
AGMP-Sync.bat
        ↓
H:\一键部署\AI-Game-Manager-Panel
        ↓
AGMP-GitHub.bat
        ↓
1. 一键推送
```

源码 ZIP 必须包含完整项目目录（`.github/cmd/configs/desktop/distribution/docs/frontend/internal/runtime/rust/scripts` 等）以及根目录开发入口。禁止把只有几个脚本的同步包伪装成完整版本源码包。交付回复必须同时给出 ZIP 的 SHA-256。

## 1.1 开发环境与生产环境硬边界

必须严格区分“源码开发/发行构建环境”和“普通用户生产运行环境”：

- **开发环境**：面向开发者，可包含源码、Go、Node.js、pnpm、Wails/Electron 工具链；只有开发/修改 Rust AI Core 或从源码制作正式发行包时，才按需准备 Rust/Cargo/MSVC/Windows SDK。
- **发行构建机/CI**：属于开发/发布基础设施，不等于普通用户生产环境。它负责把 Go/Rust/Frontend/Desktop 预编译成最终可分发产物，并执行测试、签名、哈希、许可证和安全 Gate。
- **生产环境**：面向普通用户，只运行已经编译好的 AGMP。Windows 用户下载完整 Setup/Portable，Web 用户运行完整 Web 发行包；生产环境**不得现场编译**，不得要求安装 Rust、Cargo、MSVC、Go、Node.js、pnpm、Wails CLI 或 Visual Studio Build Tools。
- 根目录 `AI-Game-Manager-Panel.bat` 是**源码开发/发行构建助手**，不是生产启动器。普通用户不应运行它。
- 任何正式发布目标都必须自带该端运行所需的 AGMP 内部 AI Core 组件，用户只选择“使用哪个 AGMP 端”，不选择“要不要另外安装 Agent”。


## 2. 技术栈

- **Go = AGMP Product Host / Domain Services。** 游戏、Steam、实例、部署、更新、Web API、认证、许可证与稳定业务 Service 继续使用 Go；Go 不是 XiaoYu 的长期 Agent Runtime 主语言。
- **Rust = XiaoYu Agent Runtime / Native Execution / Security Boundary。** Brain Policy、Agent Loop、Tool Search、Context/Session、PTY/Jobs、通用 Shell/File/Process、Sandbox、Reflection、Subagent 等 Agent 通用能力优先进入 Rust。
- **Vue + TypeScript = Presentation / UX。** Windows 主桌面当前仍为 Wails + Vue 3 + TypeScript + Vite + Pinia；不为了 Rust 占比立即重写 Wails。
- 语言职责长期规范见 `docs/architecture/LANGUAGE-OWNERSHIP.md`。
- Web 管理平台：复用同一套 Vue 前端，通过 HTTP/WebSocket Bridge 调用同一 Go Core。
- JavaScript/TypeScript 包管理只允许 **pnpm**。
- 正式用户产物按“完整 AGMP 运行端”发布：Web、Wails、Electron、未来 Rust Native。内部 Agent/Core 二进制可以存在于包内，但禁止单独作为普通用户产品发布。
- Linux 定位为一等 Headless Server Edition：完整 AGMP Web/Server + 预编译 XiaoYu Core + Model Center/Harness/Tool/审批/事件流 + `install.sh/systemd`；核心 AI 能力不得少于 Windows Desktop。

## 3. 中文规范

- UI、按钮、错误提示、README、开发文档、配置说明统一中文。
- **新增业务代码必须写必要中文注释**，重点说明边界、原因、并发/性能和安全假设。
- Steam AppID、API 字段、协议名称、Go/TypeScript 标识符可以保留英文。
- 错误信息必须告诉普通用户：发生了什么、为什么、下一步怎么做。

## 4. 版本规范

按连续小版本推进：`0.1.0 -> ... -> 0.1.99 -> 0.1.100 -> 0.2.0`。

每次交付：

- 源码：`agmp-<version>.zip`；
- 更新记录：统一追加到 `docs/PROJECT-HISTORY.md`；
- Go / frontend / Wails / BAT / Installer 版本必须一致。

## 5. 阶段制开发

每个 Phase 必须经过：

`开发 -> 自检 -> 构建 -> 实机验证 -> 记录问题 -> 修复 -> 再验证 -> 冻结基线 -> 下一阶段`。

禁止跨阶段把多个没有验证的大功能同时并入稳定基线。占位功能必须明确显示“规划中/骨架已预留”，禁止伪装完成。

## 6. 0.1.81 冻结后的架构边界

源码业务骨架按领域聚合，禁止恢复 `internal/service` 平铺大仓库：

- `rust/crates/xiaoyu-core`：**XiaoYu Agent Runtime 主实现**。负责 Brain Policy、Agent 状态机、Tool Search、Context/Session，以及后续 PTY/Jobs/Sandbox/Reflection/Subagent。
- `rust/crates/xiaoyu-protocol`：XiaoYu Runtime 与 AGMP Host 的稳定协议边界，当前协议 `xiaoyu.v1`。
- `internal/xiaoyu`：当前 Go 兼容/桥接层，包含 Contract / Control / Host / Runtime Bridge。0.2.9 起冻结“继续往 Go 增加通用 Agent Runtime”的做法；新通用 Agent 能力必须先评估 Rust。
- 现有 `internal/xiaoyu/host` 与 `internal/platform/runtime` 在迁移期间继续工作，禁止为了架构美观一次性重写；每次迁移必须有协议、测试和 Agent Bench 后再删除旧实现。
- `internal/games`：具体游戏特殊规则与 Adapter；公共能力禁止复制进游戏目录。
- `internal/server`：实例、服务器控制台、RCON、通用游戏工作区。
- `internal/ops`：文件、日志、备份、任务、计划任务等强关联运维能力。
- `internal/deploy`：部署、Steam/SteamCMD、环境检测、更新与构建。
- `internal/system`：认证、许可证、设置、节点、网络、安全、插件、通知和系统监控。
- `internal/platform`：最底层 OS/文件/网络/进程/终端/凭据/Telemetry/Steam 基础设施，不得依赖上层业务域。
- `internal/app`：应用装配与生命周期，不承载具体业务实现。
- `internal/bridge`：Wails / HTTP / WebSocket 适配，不写业务状态机。
- `frontend/src/features/xiaoyu`：小鱼用户入口；前端其余 feature 仅负责 UI 组合，业务规则仍归后端领域。

依赖方向固定为：`小鱼 -> Tool/Domain -> platform -> OS/外部程序`。`platform` 禁止反向依赖 `games/server/ops/deploy/system/xiaoyu`。

## 7. 配置规范（禁止散落硬编码）

项目可调整的平台默认值统一放在根目录 `configs/`：

- `app.json`：产品元信息；
- `ui.json`：导航和 小鱼文案；
- `ai.json`：AI 非敏感配置；
- `permissions.json`：AI 审批和权限策略；
- `paths.json`：运行目录；
- `logging.json`：日志策略；
- `games.json`：游戏模板目录；
- `modules.json`：功能模块和阶段矩阵；
- `release.json`：Windows/Linux Release 命名、架构与运行目录策略。

**API Key、Token、密码、Cookie 不允许写入 `configs/`。** 后续由系统凭据存储模块管理。

新增可调整值时优先扩展现有配置文件；禁止在多个 Vue/Go 文件里复制同一个默认值。

## 8. 复用规则

日志、备份、计划任务、终端、文件、Steam 校验、实例生命周期、权限、通知必须优先公共化。

禁止出现：

- 每个游戏复制一套 Steam 校验器；
- Desktop / Web 各写一套业务；
- 每个游戏复制一套日志/备份/文件/终端页面；
- 一个 `Manager` 或 Vue 页面承担整个系统。

## 8.1 日志系统规范

- 左侧 `日志中心` 是全平台唯一历史日志入口，具体游戏页面只能跳转并带筛选条件。
- 游戏模块负责产生日志和必要的游戏专用诊断，目录索引、分页、搜索、导出、删除统一由 `internal/ops/logs` 管理。
- 日志正文必须按需分页/流式读取，禁止把完整大日志一次发送给 Vue。
- `runtime/log/exports/` 是 0.1.65 新默认导出副本目录，不计入真实日志数量；旧配置的 `log/exports/` 继续兼容。
- 活动日志必须有删除保护；外部手工删除后的真实文件系统状态优先。
- 操作审计只能记录安全摘要，Token、密码、API Key、Cookie、完整控制台命令禁止写入。
- 日志目录名称、分页、自动刷新等可调整默认值统一由 `configs/logging.json` 提供。

## 9. 性能规则

- 禁止无界数组、日志 DOM、goroutine、Channel、缓存。
- 高频状态优先事件推送或单一自适应轮询，禁止多个页面重复轮询同一数据。
- 大日志、大文件必须分页/流式/增量处理。
- 重任务按职责归属：游戏/产品业务由 Go Domain Service 承担；XiaoYu 通用 Native Runtime（Shell/File/Process/PTY/Job/Sandbox）优先 Rust；Vue 只负责交互。
- 长期后台任务必须可取消、可关闭、可观测。
- 单 Vue/Go 文件约 800 行时必须评估拆分，禁止继续形成巨型万能文件。

## 10. AI 权限与最高管理员

标准调用链：

`AI Provider -> XiaoYu Rust Agent Runtime -> Capability / Tool -> Approval -> Generic Native Runtime 或 Go Domain Provider -> OS / Game`

游戏服务器业务优先经 Go Domain API；通用 Agent 能力（Tool Search、Shell、File、Process、PTY、Job、Sandbox 等）长期归 Rust Runtime。迁移期间现有 `shell.exec` / `fs.*` / Process 仍可由 Go Host 执行，但不得成为继续扩张 Go Agent Runtime 的理由。任何 Rust Native 执行也必须携带 Host 身份/RBAC/审批上下文并进入审计，Rust 不是绕过用户 Yes/No 的高权限后门。

审批模式由最高管理员决定：

- 请求批准：高风险操作逐次询问；
- 帮我批准：低风险自动执行，高风险询问；
- 完全访问权限：管理员主动开启后，对**已注册且已启用的 AI Game Manager Panel Tool**自动放行。

管理员可以在 `configs/permissions.json` 中关闭通用 Shell 等系统能力；默认能力存在，但 ask/risk/full 三种审批模式决定具体动作是否自动执行。AI Game Manager Panel 不替最高管理员做最终授权决定，必须清楚展示风险、保留审计、避免秘密泄露。

## 11. 安全边界

- Token、密码、API Key、Cookie 不得写日志、诊断包或普通前端持久化。
- 文件 API 必须校验允许根目录并防路径穿越。
- 压缩包解压必须限制路径、符号链接、文件数和总大小。
- Web 远程管理在开放到非 loopback 前必须先完成认证和权限系统。
- 删除、恢复、端口/防火墙、系统命令等操作必须进入操作审计。

## 12. UI 规范

- 主界面采用现代、克制、低干扰的 小鱼风格。
- 软件默认进入 `小鱼`。
- AI 未配置时，手动一键部署和全部传统管理功能仍可使用。
- 同一业务只保留一个信息源；游戏工作台中的日志/备份等入口应跳转到公共中心并带筛选条件。
- 不为了“炫”增加影响性能的持续动画、模糊和大面积重绘。

## 13. 文档规范

不允许每个目录都创建 README。核心维护文档集中在：

- `docs/PROJECT-ARCHITECTURE.md`：目录、关键文件、模块、配置、依赖和边界总说明；
- `docs/DEVELOPMENT-PLAN.md`：阶段开发计划；
- 具体已实现复杂功能可以保留专项文档，但禁止为了占位创建大量重复 MD。

新增/移动核心模块时必须同步 `PROJECT-ARCHITECTURE.md`。

## 14. 多平台安装与发布

正式 Windows 发布必须支持：

- `AI-Game-Manager-Panel-Setup.exe` 安装向导；
- 用户选择安装目录；
- 开始菜单快捷方式，可选桌面快捷方式；
- 支持卸载和固定 AppId 覆盖升级；
- 升级默认保留运行数据；
- 安装包禁止包含任何用户秘密。

正式 Linux 发布必须支持：

- 无 GUI Server Edition + Web 管理平台；
- `install.sh` / systemd 安装与管理流程；
- 可直接解压运行的 `tar.gz` 便携包；
- Debian/Ubuntu `.deb` 安装包；
- 系统安装后的运行数据必须进入当前用户可写目录，禁止把日志/配置写进只读 `/usr`；
- Linux 原生依赖必须由构建脚本检测，禁止在 Windows 里伪造“已成功交叉编译”。

Windows Wails Release 使用菜单 `4`，Electron Windows Release 使用菜单 `5`，完整双桌面发布使用菜单 `10 -> 3`；只有对应 Installer 与 Portable 都生成后才能报告 Release Success。Linux Server Edition 使用菜单 `6`（WSL）或 Linux 原生脚本。

## 15. 构建与测试

交付前至少执行：

```text
pnpm --dir frontend run build
go test ./...
go vet ./...
Windows 正式 Release
Linux Release（涉及 Linux 代码/发布脚本时）
```

平台相关功能还必须做对应 Windows 实机验收。编译通过不能冒充实机验证通过。

## 16. 更新记录

每版统一在 `docs/PROJECT-HISTORY.md` 顶部追加 `## AI-Game-Manager-Panel <version>`；至少记录新增/修复、验证结果、已知问题和下一阶段。禁止继续为 Release Notes / Validation / Completion 新建每版本历史文件。

## Linux Server Edition 规范（0.1.43 起）

- 云服务器/NAS/VPS 默认使用无界面的 Go Server Edition，不强制安装 Wails、GTK 或 WebKitGTK。
- 正式 Linux Server Release 必须同时覆盖 `linux/amd64` 与 `linux/arm64` 控制面二进制。
- ARM64 仅代表 AI Game Manager Panel 控制面可运行；具体游戏服务端是否支持 ARM64 必须由游戏模板单独声明，禁止笼统承诺“所有游戏全平台”。
- 安装脚本可以使用 root 完成系统用户、目录、systemd 等系统级变更，但 AI Game Manager Panel 服务默认必须以独立低权限用户运行。
- Docker Runtime 必须默认非 root，镜像运行阶段不得包含 Node/pnpm 构建工具。
- Linux Server 默认配置统一放在 `configs/server.json` 与 `configs/release.json`，业务代码不得散落写死安装路径、监听端口、服务用户名或架构列表。
- `bash scripts/build_linux.sh` 是 Linux Server Edition 的正式顶层构建入口；构建失败时不得生成“成功”状态文件。

## 0.1.52 双桌面 Adapter 规则

- Wails 与 Electron 只允许拥有不同桌面 Adapter / Shell，不允许复制业务逻辑。
- SteamCMD、DST、认证、日志、文件、实例、任务、网络等实现必须继续落在 Go Core。
- Electron Renderer 禁止 `nodeIntegration=true`，必须保持 `contextIsolation=true` 与 `sandbox=true`。
- Electron preload 不得暴露通用 Node、任意 Shell 或通用 `ipcRenderer`；新增能力先进入 Go Application/HTTP API。
- Wails 与 Electron 必须拥有独立 Release 入口：菜单 4 只构建 Wails，菜单 5 只构建 Electron；任何一方失败不得修改另一方最近一次成功产物。
- 双桌面完整发布统一由菜单 `10 -> 3` 顺序调用两个已独立验证的 Release，不得维护第三套重复构建逻辑。任一 Adapter 的 bug 修复优先修 Core/公共 Vue，而不是复制补丁。

## 0.1.53 首次启动布局规则

- 首次启动整个舞台必须在可用窗口内保持视觉居中；内容超高时必须允许完整滚动，不能通过强制顶对齐破坏大屏居中效果。
- AI Game Manager Panel 品牌标题必须脱离实际表单卡片，放在顶部左侧独立区域。
- 当前步骤标题必须脱离实际表单卡片，放在顶部右侧独立区域，与品牌区构成左右布局。
- 注册、登录、环境初始化的真正表单内容必须作为独立卡片在标题区下方居中。
- 禁止在内容卡片内再次重复 `auth-step` 标题。
- 小屏允许收窄与滚动，但不允许丢失底部操作按钮或破坏表单可访问性。

## 授权保险规则

- CDK / 机器码 / BFID 属于“系统设置 → 授权与设备安全”的独立保险层，不得放在 Owner 注册或登录之前阻断首次使用。
- Owner 注册/登录继续使用账号 + 密码 + 可选安全密钥；授权保险与身份认证必须保持职责分离。
- BFLC2 离线许可证必须使用 Ed25519 签名，不能使用可逆“算法 CDK”或客户端内置共享密钥；短 CDK 仅作为未来在线 License Server 的兑换码。
- 发行私钥永远不得进入 AI Game Manager Panel 源码、客户端、安装包、日志或配置目录。
- 发行私钥只能由项目所有者在仓库外本机生成并离线备份；正式 Release 必须通过 Release Key Gate，确认源码存在一个合法 `active` 发行公钥。
- 公钥环允许 `active / retired / legacy` 共存；密钥轮换不得默认删除仍用于验证历史 BFLC2 的旧公钥。
- 许可证公钥必须编译进 Go Core，不能从用户可编辑 JSON 读取作为信任根。
- Windows 机器码只使用稳定系统标识派生结果，UI/日志不得输出原始 MachineGuid。

## pnpm Store 与依赖恢复规则

- Windows 开发/打包依赖默认使用 AI Game Manager Panel 专用 Store：`%LOCALAPPDATA%\AI Game Manager Panel\DevTools\pnpm-store`，不得依赖用户全局 pnpm Store。
- 正常安装只允许使用经过项目审核的最小参数集：`pnpm install --store-dir <AI Game Manager PanelStore>`。禁止添加未经当前 pnpm 官方文档/帮助确认的参数。
- 首次安装失败时，只允许重置当前 AI Game Manager Panel 项目的 `node_modules` 与 AI Game Manager Panel 专用 Store；不得删除或修改用户其他项目的全局 pnpm Store。
- Electron Runtime 使用 `install-electron`，缓存位于 `%LOCALAPPDATA%\AI Game Manager Panel\DevTools\electron-cache`。

## 0.1.62 Windows 开发助手与编码硬规则

- 根目录只允许一个用户入口 `AI-Game-Manager-Panel.bat`；该文件必须 ASCII-safe + CRLF，只负责切换 UTF-8 code page、定位项目目录并启动 `scripts/windows/AIGameManagerPanel.ps1`。
- `scripts/windows/` 下禁止再出现 `.bat`；Windows 菜单、任务编排、依赖、检查与 Release 统一使用 PowerShell。
- 所有 `.ps1` 必须保存为 **UTF-8 with BOM + CRLF**，兼容 Windows PowerShell 5.1 与 PowerShell 7。
- PowerShell 启动后必须显式设置 Console Input/OutputEncoding 为 UTF-8；任何 `�`、非法字节、中文残片被 CMD 当作命令、菜单行乱码，都视为 **Gate FAIL**，禁止进入 Release。
- 禁止 `Invoke-Expression`、禁止字符串拼接后执行外部命令、禁止 Node `shell:true`。外部程序统一使用 `& $exe @args` / 参数数组调用，必须支持中文路径和包含空格的路径。
- 初始化（菜单 1）只负责工具链、Go Modules 与 Frontend 依赖；Electron 依赖/Runtime 在菜单 2 的 Electron 开发或菜单 5 的 Electron Release 时按需准备。初始化不得把 TypeScript/业务代码错误混进“环境初始化失败”；代码质量统一由菜单 3 检查。
- 主菜单保持精简，当前固定 0-10；新增菜单前必须先证明不能通过现有子菜单合并。
- Wails、Electron、Web 必须可独立开发/检查/发布；一键发布只能调度已存在的独立 Release，不允许维护重复构建逻辑。
- `scripts/common/check-windows-helper.mjs` 与 `scripts/windows/Test-WindowsHelper.ps1` 是强制门禁：检查编码、路径、禁用模式、Frontend 相对导入与菜单映射。
- 每次交付 Windows Helper 变更，必须执行菜单 1-10 Dry-Run 自检；在 Windows 实机上还必须至少验证菜单 1、3（完整检查）、4、5、8、10。
- 任何 Windows Helper 交付包若出现 `�`、乱码菜单、CMD 把中文残片当命令、路径被空格截断或 native command 输出编码错误，必须立即判定该版本不可交付；禁止以“用户环境差异”绕过编码 Gate。

## 0.1.63+ 历史记录收敛规则（0.2.3 起）

- 历史 Release Notes、Validation、Completion、开发 Prompt 已统一归并到 `docs/PROJECT-HISTORY.md`。
- 新版本不再创建 `docs/releases/<version>.md`、`docs/prompts/<version>-*.md`、`VALIDATION-*` 或 `COMPLETION-*`。
- 用户确认的长期开发约束应更新到 `AGENTS.md` / 当前架构与开发规范；仅用于追溯的版本背景摘要追加到 `docs/PROJECT-HISTORY.md`。
- `docs/PROJECT-HISTORY.md` 是唯一历史入口，历史条目不得改写成当前规范；当前规范也不得只存在于历史记录中。

## 0.1.64 工作台、PowerShell 与 Electron 发布回归硬规则

- PowerShell 变量名大小写不敏感。`scripts/windows/` 内禁止把 `$home`、`$host`、`$pid` 等名称用作可写局部变量；新增脚本必须通过自动变量/只读变量冲突 Gate。
- 只负责显示信息的 PowerShell 函数不得把 Hashtable / PSObject 意外写入成功输出管道；如需内部返回对象，调用端必须显式接收或 `$null = ...`。
- Electron 原生 Application Menu 必须显式中文化；禁止回退 Electron 默认英文菜单，也禁止“帮助”跳转无关占位网址。
- Electron Windows Release 在 AI Game Manager Panel 已准备本地 Electron Runtime 后，必须优先使用经验证的本地 `electronDist`；Release 前先验证 `electron.exe` 与 builder 配置，不允许重新把 Electron distribution 下载链当作默认兜底。
- AI Game Manager Panel 主界面采用可拖拽工作台能力：左右侧栏可独立调整宽度并完全收起；底部面板可调整高度并完全收起；布局状态持久化。终端入口归属 Bottom Panel，不得重新放回左侧一级导航。
- Theme 必须覆盖 `html/body/#app`、左右侧栏、Topbar、Bottom Panel 和全部主要 Surface。任何“浅色卡片 + 深色外围壳”的混合状态都判定 Theme Gate FAIL。
- 设置/功能页不得通过窄 `max-width` 人为浪费工作区；大屏中央区域默认只保留约 24~32px 安全边距，并使用统一 Typography/Spacing Token。
- 0.1.64 十项统一修复的历史需求已归并到 `docs/PROJECT-HISTORY.md`；长期有效规则以当前 `AGENTS.md` 与 Gate 为准。

## 0.1.74 Agent-first / Rust Runtime 规则

- 小鱼是默认主入口；传统可视化页面和手动终端是辅助/兜底，不得反过来让 AI 变成装饰聊天框。
- 小鱼 Runtime 与 Go Domain Core 必须保持职责分离：Rust 不复制 Steam/DST/实例/授权等领域业务，Go 不继续扩张 Brain/Planner/Tool Search/Session/Reflection 等通用 Agent Runtime。0.2.9 起 Process/stdio/PTY 进入渐进迁移期：现有 `internal/platform/runtime` 保持兼容，新 XiaoYu 通用执行能力优先 Rust。
- Rust wire protocol 当前固定为 `xiaoyu.v1`，UI/Go/Rust 三层字段变更必须同步并有 Gate。
- XiaoYu 智能能力不能只靠静态 Prompt/Gate 验证；`internal/xiaoyu/host/bench_test.go` 是首批 Agent Bench 基线，至少覆盖 fallback recovery、mutation 后验证、approval resume。新增 Agent Runtime 行为必须优先增加可重复 Bench。
- Agent Bench 测量“目标是否被正确完成/恢复/验证并尊重审批边界”，禁止以 Tool 数量、Prompt 长度或角色数量冒充智能提升。
- Tool 必须声明风险级别；Agent 不得直接执行未注册能力。
- `ask/risk/full` 是审批策略，不等于“模型想做什么都能做”；`full` 仍只对明确注册并允许的 Tool 生效。
- 0.1.74 的 `process.run` 是一次性受控命令，不得在文档中冒充持久 PTY。
- 后续可参考 Codex / OpenCode / Claude Code / ChatGPT 的公开交互与架构思想，但禁止直接复制不兼容授权或非开源实现。

## 17. 0.1.83 核心架构定型与 XiaoYu 能力接口硬规则

- 一个业务模块一个明确归属。文件、设置、节点、授权、备份、日志、终端、Steam、实例等各管各的，禁止把 A 模块业务逻辑塞进 B 模块目录。
- 页面归属不等于业务归属。例如授权 UI 可以显示在设置中心，但授权核心仍归 `internal/system/license`。
- AI 只能调用 Host 注册的 Tool / Service / API；`internal/xiaoyu` 不得复制业务实现。已有可靠 Domain Tool 时优先使用它；没有合适领域能力或现实异常超出预设时，允许转向受控通用 Tool / `shell.exec`，但不得绕过 Host 执行边界。
- 小功能不要为了“模块化”拆得过碎；强关联代码可同文件。需要拆时使用相同短前缀，例如 `node_scan.go`、`node_scan_win.go`，让人工和 AI 一眼看出是一组。
- 未实现功能不得创建只有 `doc.go`、`module.ts` 或空 README 的占位包；规划状态统一由 `configs/modules.json` 管理，出现真实代码时再创建目录。
- `internal/app` 允许按 `app_core.go / app_games.go / app_xiaoyu.go / app_auth.go` 等同包文件聚合，禁止重新长成一个 1000+ 行全能文件。
- 终端只维护一套用户入口：Bottom Panel `TerminalPanel`；系统能力统一走 XiaoYu/Domain 安全边界，不得出现死代码式第二套 Terminal UI。
- `internal/platform/runtime` 继续作为 Go Domain 的共享 Process/Terminal 基座；游戏域不得各自复制进程实现。XiaoYu 的新通用 PTY/Job/Shell Runtime 优先进入 Rust，并通过稳定协议调用 Go Domain，而不是继续扩张 Go Host Agent 机制。
- 新增文件/目录/接口名称必须短、明确、可搜索。优先 `files`、`nodes`、`license`、`settings`、`backup` 等稳定业务词，避免冗长重复命名。
- 同一功能拆成多个文件时必须使用统一短前缀，例如 `file_scan.go`、`file_scan_rule.go`、`file_scan_win.go`；有明确顺序时才使用 `xxx_one.go`、`xxx_two.go`。不得把同一功能拆成看不出关联的一堆文件。
- 新增源码文件名优先控制在 32 个字符以内；超过 48 个字符必须有明确理由并由 Module Gate 拦截。
- 当前阶段不做大规模项目搬家；只做不破坏功能的轻量归属纠偏。核心能力、稳定、性能、安全和省心优先。

AI 权限名称固定为 **请求批准 / 帮我批准 / 完全访问权限**。完全访问权限只减少审批，不取消任何硬安全检查。即使 Full Access，AI 仍必须通过模块能力接口，并执行必要的校验、保护和结果验证。

### 0.1.88 双大脑 / 多端 XiaoYu 硬规则

- AGMP 产品模型固定为 **Human + XiaoYu / two-brains-one-body**。用户拥有最终控制权；XiaoYu 负责智能规划与自动执行，两者共享 Host/Domain/Runtime，禁止两套业务身体。
- XiaoYu 属于 AGMP Core，不属于桌面端。Web/Linux/Docker 必须拥有与 Windows Desktop 等价的核心 AI：Model Center、Harness、Tool、审批、Run、事件流。
- Headless 是强制架构能力：`internal/xiaoyu` 与 Rust XiaoYu Runtime 禁止依赖 Wails/Electron；Linux/Docker 正式包缺预编译 XiaoYu Runtime 直接失败。
- Autonomous Run 归 Server 所有，客户端断线不取消；Run 绑定 Initiator，多用户 Operator 只能查看/控制自己的 Run，Owner/Admin 可监督全部。
- Human takeover/pause/cancel 永远高于 XiaoYu；人工接管后自动循环停止，直到明确交还。
- Go Host/RunManager 当前负责产品生命周期、模型通信、身份/RBAC 与 Domain Tool Bridge；新的 Planner/Tool Search/Session/Recovery/Reflection 等语义能力优先迁入 Rust，Go 不得继续形成第二套长期 Agent Runtime。
- 第三方 DSH Tool 插件默认以 Node Permission 受限子进程运行，只读插件自身目录，不继承 AGMP 秘密环境，不允许直接 child_process/文件写；插件入口与 patch 必须通过 symlink 真实路径边界检查。

### 0.2.9 Rust-first Agent Runtime 硬规则

- Rust `xiaoyu-core` 不再定义为 Brain-only，而是 **XiaoYu Agent Runtime**。新增 Tool Search、Context/Session、PTY/Jobs、通用 Shell/File/Process、Sandbox、Reflection、Subagent 等能力优先 Rust。
- Rust 不复制 Steam/DST/Minecraft/Instance/License/Updater 等 Go Domain 业务；Domain Tool 仍由对应 Go Service 提供。
- `internal/xiaoyu/contract.Registry` 在迁移期间继续作为 Go Domain Tool Catalog/Dispatch 边界；同名 Tool 禁止覆盖，名称和风险级别必须通过校验。
- Rust Native Tool 不能绕过身份/RBAC/三种审批模式。Host 授权上下文、路径/参数边界、审计和执行后验证都是硬安全要求。
- `internal/platform/runtime` 继续服务 Go Domain；现有业务域不得直接 `exec.Command`、`exec.CommandContext` 或自行创建 stdin/stdout/stderr pipe。XiaoYu 新的通用 Native Runtime 不再默认堆到此 Go 包。
- 长生命周期游戏/终端会话统一使用 `Session`；一次性内部命令统一使用有输出上限和 Context 取消的 `Run`；无需交互的外部启动使用 `StartDetached` 并异步回收。
- Session 必须保留 stdout/stderr 来源、串行化 stdin、PID/状态/退出码快照；Unix 终止以进程组为单位，Windows 保持隐藏控制台并允许显式 `ShowWindow` 的 GUI 安装器例外。
- Linux Native PTY 已进入 Rust XiaoYu Runtime；Windows ConPTY 和跨 AGMP 重启后的会话重连仍是独立增强项。实现时必须扩展当前 Terminal/Runtime contract，不得再创建另一套 Process Core。

## 18. 0.1.81 架构冻结

0.1.81 完成核心目录聚合后冻结骨架。后续新增能力优先进入既有领域，禁止为了单一小功能新增一级业务域。

## Naming and path convention

All contributors and AI agents must follow `docs/NAMING-CONVENTIONS.md` before creating or renaming files. Directory context is part of the namespace: do not repeat parent-directory meaning in filenames. Prefer short, conventional names; repository-relative paths over 180 characters require review, and paths over 220 characters are forbidden by CI. Version source bundles use `agmp-<version>.zip`.


### 0.2.10 Rust Session / Job Runtime 硬规则

- `rust/crates/xiaoyu-core/src/session.rs` 是 XiaoYu Session Registry；`jobs.rs` 是长任务 Runtime。
- Session / Job / PTY 属于 Rust Agent Runtime，不得为了方便再在 Go Host 新建第二套 XiaoYu Job Core。
- `jobs/start` 必须要求 Host 授权上下文；Rust 不得被当成绕过三种审批模式、RBAC 或 Sandbox 的高权限后门。
- Job stdout/stderr 必须有界，必须支持 status/output/cancel，禁止无限内存日志。
- 0.2.10 的 Job RPC 先作为内部 Runtime primitive；在 persistent Go↔Rust RPC worker 完成前，不得宣称模型 `shell.exec` 已迁入 Rust。
- 新增/修改 Rust Runtime 代码必须通过 `cargo fmt --check`、`cargo check --locked`、`cargo test --locked` 和 `check-xiaoyu-jobs.mjs`。

### 0.2.11 Persistent Rust Runtime Worker 硬规则

- `xiaoyu rpc` 是 Rust Agent Runtime 的长期进程边界；Go Host 不得重新退回“每次 RPC 都启动一个 Rust 子进程”的短进程模式。
- `internal/xiaoyu/runtime/rpc_worker.go` 负责 Worker 生命周期、stdio JSON-RPC、超时回收和有界 stderr；不得复制第二套 Worker。
- Session / Job 状态只在同一 Rust Worker 生命周期内可靠存在；Worker 崩溃/重启后 Host 必须重新观察状态，禁止假装旧 Session 仍存在。
- Go Bridge 可以暴露 Host-internal Session/Job API，但 `jobs/start` 必须在 Go 和 Rust 两侧都要求 Host authorization；模型不可直接伪造授权。
- Persistent Worker RPC 当前串行化，先保证状态一致与可审计；后续若引入并发 multiplexer，必须先增加 request-id correlation、取消语义和并发测试。
- Windows 源码同步禁止直接显示 Robocopy OEM 报表；中文路径诊断必须使用 Unicode 日志或 PowerShell 自身输出。
- 新增/修改 Worker 必须通过 `check-xiaoyu-worker.mjs`，并继续通过 Rust fmt/check/test、Go test/vet 与 Agent Bench。

### 0.2.20 Approved Agent Native Terminal 硬规则

- 0.2.19 GitHub 四条主 Job 已全绿，Native Terminal backend 冻结；后续 AI 不得无证据继续改 ConPTY/PTTY。
- `shell.exec` 只有在 server-owned XiaoYu Run 且 Go Host 已完成 identity/RBAC/step-up/approval 后，才可进入 Rust Native Terminal。
- `HostAuthorized=true` 只能由 Go Host bridge 写入，绝不接受模型/前端参数。
- Agent Native Terminal 必须使用 workspace-resolved CWD、Tool timeout、有界输出，并保证 Terminal lifecycle 收口。
- Native Terminal 失败必须 fail-closed；禁止对同一已批准动作静默 fallback 到 legacy shell。
- 人工 Shell / `process.run` compatibility 本版继续留在 `platform/runtime`；不要借迁移扩大模型权限或重写 Domain Tool。
- 下一阶段才做 Sandbox / Capability Lease；在 Lease 落地前不向模型开放长期 Terminal ID、任意 input/resize 或跨 workspace scope。

### 0.2.19 Windows ConPTY stdio isolation 硬规则

- 真实 Windows Runner 中，若父进程 stdout/stderr 被 test harness/CI 捕获，Windows console child 可能在 `bInheritHandles=false` 时仍获得父标准句柄；这会绕过 ConPTY pipes。
- Windows ConPTY `STARTUPINFOEXW` 必须设置 `STARTF_USESTDHANDLES`，且 `hStdInput / hStdOutput / hStdError` 必须为 NULL 后再调用 `CreateProcessW`。
- `bInheritHandles` 继续为 false；不得用继承父标准句柄来“修复”测试。
- Windows ConPTY Enter 继续使用单个 CR；cwd normalize、resize lock scope、fallback cfg 与 per-input Host authorization 不得回退。
- `check-xiaoyu-terminal.mjs` 与 Windows 单元测试必须冻结 stdio isolation；Windows integration/workspace tests 全绿前不扩大模型可见 Terminal 权限。

### 0.2.18 Windows ConPTY Input 硬规则

- 0.2.17 的 CRLF 输入尝试已被真实 Windows Runner 证伪；Windows ConPTY 的 Enter / `appendNewline` 必须发送单个 CR（`\r` / `0x0D`）。
- Linux PTY 与 fallback backend 继续使用 LF（`\n`）；平台输入差异只留在 Rust Terminal Runtime，不复制到 Go Host。
- `check-xiaoyu-terminal.mjs` 与 Windows 单元测试必须冻结 CR 语义并拒绝 CRLF 回归。
- 0.2.17 的 cwd normalize、resize lexical MutexGuard 与 fallback `Pipe` cfg 修复继续有效。
- 本版仍不扩大模型可见 Terminal 权限；Windows integration/workspace tests 全绿前不进入下一阶段。

### 0.2.17 Windows ConPTY Runtime 硬规则

- Runtime Root / Session scope 内部仍使用 canonical path 做边界比较；只有在 Windows `CreateProcessW` current-directory 边界才规范化本地 `\\?\X:\...` verbatim 前缀。
- 0.2.17 曾尝试让 Windows ConPTY `appendNewline` 发送 CRLF；真实 Runner 已证明该输入语义不成立，现由 0.2.18 的单 CR 规则取代。Linux PTY 与 fallback backend 始终使用 LF。
- Terminal process mutex 必须依赖 lexical scope 释放；禁止 `drop(&mut TerminalProcess)` 这种无效解锁写法。
- `TerminalProcess::Pipe` 只属于无 native PTY/ConPTY 的 fallback 平台；Windows/Linux native build 不应保留该 dead-code variant。
- 0.2.17 仍不扩大模型可见 shell 权限。Windows ConPTY integration/workspace tests 全绿前，不进入 Agent Terminal wiring 或 Sandbox/Capability Lease。

### 0.2.16 Windows ConPTY ABI 硬规则

- `windows-sys 0.61.2` 的生成签名是 Rust 侧唯一事实源；Windows FFI 类型不得按 C 头文件印象猜测。
- `UpdateProcThreadAttribute` 的 pseudo-console attribute 必须传 `usize`。
- `HPCON` 当前按 `isize` 处理，空值为 `0`。
- Windows ConPTY 未通过 Windows Runner 的 check/integration/tests 前，不得宣称跨平台 Native Terminal 已完全冻结。
- 本机有 Cargo 时，一键推送前必须运行 Rust fmt + workspace check；没有 Cargo 才交给 CI。

### 0.2.15 Native Terminal CI Convergence 硬规则

- `cargo fmt`、`cargo check`、Linux PTY integration、Windows ConPTY integration、workspace tests 都是独立证据；不得用“前序失败所以后续没跑”当作功能已验证。
- Rust check/integration/test 在 GitHub Actions 中必须使用未取消继续执行策略，使一次 CI 尽量暴露所有真实错误；任何一步失败仍应让 Job 最终失败。
- 本机有 Cargo 时，一键推送必须执行 rustfmt preflight；本机无 Cargo 时必须明确提示由 CI 验证，禁止写成“Rust 已通过”。
- 双平台 Native Terminal 没有实际 integration 全绿前，不进入 Sandbox/Capability Lease 或模型可见 Terminal wiring 的下一阶段。

### 0.2.14 Windows ConPTY / Cross-platform Native Terminal 硬规则

- Windows Native Terminal 必须落在 Rust XiaoYu Runtime，不得回流成 Go Agent Runtime。
- Windows 必须复用既有 `terminal/*` contract；平台差异只允许在 `pty_windows.rs` / `pty_linux.rs` backend。
- `windows-conpty-v1` 必须由独立 Windows Rust CI 实际 build + integration test 后才算稳定能力；禁止只靠源码 token 宣称完成。
- Terminal start、write、resize 仍受 Go Host RBAC / 三种审批模式控制，ConPTY 不得成为长期 shell 权限绕过通道。
- Linux 使用 `linux-pty-v1`，Windows 使用 `windows-conpty-v1`；普通 stdio fallback 必须明确标记，禁止冒充 PTY/ConPTY。

### 0.2.13 Linux Native PTY 硬规则

- `rust/crates/xiaoyu-core/src/terminal.rs` 是唯一 XiaoYu Terminal 生命周期；`pty_linux.rs` 只实现平台 backend，不得复制一套 Session/Approval Core。
- Terminal start、**每次输入**与**每次 resize**都必须通过 Host identity/RBAC/approval 后再携带 `hostAuthorized=true`；“会话之前已经批准”不能自动授权后续任意命令。
- Terminal output 和单次输入必须有界；cwd 必须继续限制在 Runtime Root / Session scope。
- Linux 只有真实 PTY、controlling TTY、TTY detection 与 resize integration tests 都存在时才能声明 `linux-native-pty-v1`。
- Windows 0.2.13 必须明确保持 `stdio-pipe-v1` fallback；没有 Windows Rust CI 实际 build/test 前不得声明 ConPTY 已完成。
- Native PTY/ConPTY 必须复用同一 `terminal/*` protocol；禁止为某个平台创建第二套 Terminal RPC 或绕过 Host Approval。
- 模型可见 `shell.exec` 暂不直接切换到 Terminal RPC；迁移前必须有 Agent Bench 覆盖 approval、输入、输出、取消与恢复。
- 新增/修改 Terminal Runtime 必须通过 `check-xiaoyu-terminal.mjs`、Rust fmt/check/test、Go test/vet 和现有 Agent Bench。
