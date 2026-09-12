# AI Game Manager Panel 开发计划

> **0.1.65 菜单说明：** 本文较早阶段记录中的旧菜单编号只作为历史验收记录保留。当前 Windows Helper 顶层菜单以 `AI Game Manager Panel.ps1` 的 0-10 为准：Wails Release=`4`、Electron Release=`5`、Linux=`6`、双桌面完整发布=`10 -> 3`。

> 项目目标：现代化、智能化、AI 驱动的一键游戏服务器部署与管理平台。每个阶段都必须经过开发、自检、正式构建、Windows 实机验证、问题修复、更新记录、冻结基线。


## 0.2.17：Windows ConPTY Runtime Convergence

- 以 0.2.16 Windows Runner 的真实 integration 日志为修复依据，不增加新的模型执行权限；
- Windows `CreateProcessW` 之前必须把 Rust `canonicalize()` 产生的本地 `\\?\X:\...` cwd 还原为普通 drive path，避免 `cmd.exe` 误判为 UNC 并回退到 `C:\Windows`；
- `terminal/write` 的 `appendNewline` 必须按 backend 发送：Windows ConPTY=`\r\n`，Linux PTY/其他 backend=`\n`；
- Terminal resize 的 process mutex 必须通过 lexical scope 释放，禁止 `drop(&mut TerminalProcess)` 这种无效解锁；
- `TerminalProcess::Pipe` 仅在没有 native PTY/ConPTY 的 fallback 平台编译，减少平台 warning；
- 增加 cwd verbatim-prefix 与 ConPTY CRLF 的静态/单元 Gate，已通过的 Linux PTY、Safety、Headless、Windows Helper 不得回退。

**冻结条件：** `Windows Rust Runtime + ConPTY` 的 `cargo check`、`windows_terminal_` integration、workspace tests 全绿；其余三个主要 Job 继续全绿。

### 下一步

Windows ConPTY integration 全绿后，再进入 Approved Agent → Native Terminal wiring 与 Sandbox / Capability Lease。

## 0.2.15：Native Terminal CI Convergence

- 按 0.2.14 GitHub Runner 精确输出修复 `pty_windows.rs` 两处 rustfmt 差异；
- 不继续叠加 Sandbox/Capability Lease，先让 Linux PTY 与 Windows ConPTY 真正跑到 integration test；
- Rust format 即使失败，未取消的 `cargo check` / platform integration / workspace tests 仍继续执行，避免一次只暴露一个问题；
- Terminal/PTY Gate 必须验证 Linux/Windows integration 命令和“格式失败后继续验证”策略都存在；
- `AGMP-GitHub` 检测到 Cargo 时执行本机 `cargo fmt --check`，缺少 Cargo 时只警告并由 CI 权威验证；
- 只有双平台 Native Terminal CI 全绿，下一阶段才允许把 Approved Agent Terminal wiring 接入模型执行链。

**冻结条件：** Safety、Windows Rust Runtime + ConPTY、Windows Helper、Linux Headless 四个主要 Job 全绿；Rust fmt/check/test、Linux PTY integration、Windows ConPTY integration 全部实际执行并通过。

### 下一步

全绿后进入 Approved Agent Terminal wiring + Sandbox/Capability Lease 基础；仍保持 Go Host 决定授权、Rust Runtime 执行原语。

## 0.2.14：Windows ConPTY / Cross-platform Native Terminal

- 修正 0.2.13 GitHub Runner 剩余的一处 Rust import-order `cargo fmt --check` 差异；
- 保持既有 `terminal/*` contract，不为 Windows 再造第二套 Terminal API；
- Windows backend 使用 ConPTY：`CreatePseudoConsole`、`PROC_THREAD_ATTRIBUTE_PSEUDOCONSOLE`、`CreateProcessW`、`ResizePseudoConsole`；
- Windows Snapshot 目标为 `backend=windows-conpty-v1`，Linux 继续 `linux-pty-v1`；
- ConPTY input/output 继续有界并持久，start/write/resize 均保留 Host authorization；
- 新增独立 `Windows Rust Runtime + ConPTY` CI Job，Windows Runner 必须执行 Rust check/tests 和 ConPTY integration test；
- 未经 Windows Runner 实际验证，不把 ConPTY 宣布为稳定完成态。

**冻结条件：** 既有 Go/Headless/Windows Helper Gate 不回退；Rust fmt/check/test 通过；Linux PTY integration 通过；Windows ConPTY integration 通过；四个主要 CI Job 全绿。

### 下一步

Native Terminal 双平台站稳后，进入 Sandbox / Capability Lease 与 Approved Agent Terminal wiring，继续保持 Rust-first Runtime 和 Go Domain Host 的职责边界。

## 0.2.13：Linux Native PTY Foundation

- 修正 0.2.12 GitHub Runner 报出的两处 `cargo fmt --check` 差异；
- 保持既有 `terminal/*` contract，不创建第二套 Terminal API；
- Linux backend 使用真实 PTY master/slave、`setsid`、controlling TTY；
- 新增 `terminal/resize`，rows/cols resize 必须再次经过 Host authorization；
- Linux CI 测试必须证明 `test -t 0` 为真，并用 `stty size` 验证 resize；
- PTY output 继续有界，cwd 继续限制在 Runtime Root / Session scope；
- Windows 当前继续明确使用 `stdio-pipe-v1` fallback，不允许提前声明 ConPTY 已完成。

**冻结条件：** 0.2.12 已通过的 Go/Headless/Windows Gate 不回退；Rust fmt/check/test 通过；Linux PTY TTY/resize 测试通过；Terminal Gate PASS。

### 下一步

0.2.14 增加 Windows Rust Runtime CI，并实现/验证 Windows ConPTY backend；只有 Windows Runner 实际 build + integration test 通过后，才允许发布 `windows-conpty-*` capability。

## 0.2.11：Persistent Rust Runtime Worker

- 修正 0.2.10 GitHub Runner 报出的 Rust rustfmt 差异；
- Go Host 建立受监督的长期 `xiaoyu rpc` stdio Worker；
- Tool Search / Brain / Session / Job RPC 复用同一 Rust 进程，保持 stateful Runtime；
- Application Startup 预热 Worker，Shutdown 负责关闭；
- 超时/断管时终止坏 Worker，禁止无限卡死；
- Worker stderr 采用有界 tail buffer；
- Host-internal Session/Job Go Bridge 保留双层 `hostAuthorized` 防线；
- 修复 `AGMP-Sync` 的 Robocopy 中文路径乱码；
- 新增 Persistent Worker Gate 并接入 GitHub / Windows Helper / 一键推送。

**冻结条件：** Rust fmt/check/test 通过；Go test/vet 通过；Linux Headless / Windows Helper 继续通过；Persistent Worker Gate PASS。

### 下一步

0.2.12 把已经通过 Approval 的长任务逐步切到 Rust Job Runtime，并准备 PTY/interactive session；不绕过现有三种审批模式。

## 0.2.10：Rust Session / Long-running Job Runtime

- 修正 0.2.9 `cargo fmt --check` 唯一红灯；
- Rust `xiaoyu-core` 建立真实 Session Registry：create/get/list/close；
- 建立 Long-running Job：start/get/list/output/cancel；
- Job 输出必须有界，防止模型/Runtime 被无限日志撑爆；
- `jobs/start` 必须显式携带 Host authorization，Rust 不得成为绕过三种审批模式的新入口；
- cwd 必须限制在 Runtime Root；
- 新增 XiaoYu Session/Job Gate，并接入 GitHub / Windows Helper / 一键推送；
- 本阶段不直接替换现有 Go `shell.exec`，先把 Rust 原语、协议、测试和安全边界站稳。

**冻结条件：** Rust fmt/check/test 通过；Session lifecycle 与 Job execution/output/cancel 测试通过；三 Job CI 重新全绿。

### 下一步

0.2.11 建立 Go ↔ Rust persistent RPC worker，让 stateful Session/Job Runtime 真正成为 Host 可长期复用的执行进程；随后再迁移批准后的长任务与 PTY。

## 0.2.9：Rust-first Agent Runtime 与依赖冻结

- 冻结语言职责：Rust = XiaoYu Agent Runtime / Native Execution / Security Boundary；Go = AGMP Domain Host；Vue/TypeScript = UI；
- 不推翻 0.2.8 全绿基线，现有 Go Host/Process Runtime 进入渐进迁移期；
- 第一块 Rust-first 迁移能力为 Tool Search / Capability Discovery，Go 仅保留兼容 fallback；
- 正式提交 0.2.8 GitHub Runner 生成的 Go / Cargo / Frontend / Electron 锁文件；
- CI 切换到 `cargo --locked`、`pnpm --frozen-lockfile`、`go mod verify + tidy diff`；
- 新增 Language Ownership Gate 与 Dependency Lock Gate；
- Wails 继续作为 Windows 主桌面壳，不为了 Rust 占比立即迁移 Tauri。

**冻结条件：** 本地 Go test/vet 与全部 Node Gate 通过；推送后 GitHub Actions 三 Job 继续全绿；Rust `tools/search` cargo tests 通过；锁文件在 CI 不产生 diff。

### 后续 Rust-first 顺序

1. Session / Job Runtime；
2. PTY / Long-running Process；
3. Sandbox / Capability Lease；
4. Apply Patch / Generic Files；
5. Reflection / Experience；
6. Subagent / Specialist Dispatch。

## 0.2.8：CI 全绿与 XiaoYu Agent Bench

- 修复 Rust `cargo fmt --check` 唯一剩余红灯，让 Safety Job 能继续进入 `cargo check` / `cargo test`；
- 建立首批 XiaoYu Agent Bench：fallback recovery、mutation verification、approval resume；
- Agent Bench 作为单独 CI 步骤显示，不再只依赖静态 Gate 或“Tool 数量”；
- Headless 联网 Runner 产出 `go.sum`、`Cargo.lock`、Frontend/Electron `pnpm-lock.yaml` 依赖快照 Artifact，为下一版冻结依赖；
- 保持 0.2.7 已通过的 Windows Helper、Frontend Build、Linux Headless、Go test/vet 不回退。

**冻结条件：** GitHub Actions 三个 Job 全绿；`XiaoYu Agent Bench` 独立 PASS；成功生成 `agmp-dependency-locks` Artifact。

## 0.2.7：可复现源码同步与命名治理

- Windows 源码工作副本不再依赖 Explorer 大批量覆盖，新增 `AGMP-Sync.bat + sync-agmp.ps1`。
- Source Tree Gate / Naming Gate 同时进入本地一键推送与 GitHub Actions。
- 命名长期规范维护在 `docs/NAMING-CONVENTIONS.md`，避免无意义超长文件名与路径。
- 根开发助手在关键源码缺失时必须可诊断、可停留，不允许静默闪退。

**冻结条件：** 本地 Gate 通过；全新 Git 仓库可完整跟踪源码；GitHub Actions 通过源码完整性、Frontend、Rust、Go 与 Headless XiaoYu 验证。

## 0.2.3：GitHub 可复现基线与历史文档收敛

- 修复 `.gitignore` 未锚定根目录导致 `internal/ops/logs`、前端 `logs/instances` 源码被 Git 静默忽略的问题；
- GitHub 推送工作台改为 ASCII BAT 启动 UTF-8 BOM PowerShell，禁止再出现 BOM/代码页导致的 CMD 乱码；
- 版本历史统一到 `docs/PROJECT-HISTORY.md`，不再为每个版本增加 Release/Validation/Completion/Prompt 历史文件；
- 推送前自动检查源码误忽略、敏感凭据、运行数据、构建产物与大文件；同步远端使用 rebase，不允许脚本自动 force push；
- 以 GitHub Actions 作为 Frontend / Rust / Linux / 后续 Windows 的联网可复现验证环境。

**冻结条件：** 本地非 Publisher Gates + 非 Wails Go tests/vet 全部通过；GitHub Actions 重新验证 Frontend、Rust 与 Headless；后续补 Windows Wails/Electron 矩阵。

## 0.2.2：XiaoYu Agent Runtime / Guided Autonomy

- 把“模型智能”正式作为 XiaoYu 的通用智能来源，Expert / Skill / Memory / Experience 只做优先专业指导；
- Domain Tool 优先，但不再把缺少专用 Tool 等同于 XiaoYu 无能力；
- 增加受控通用文件写入/替换/创建/删除与 `shell.exec` 后备能力；
- 继续沿用请求批准 / 帮我批准 / 完全访问三种审批模式作为最终执行授权；
- Runtime Manager 向 XiaoYu 暴露 resolve/default/remove，形成安装→验证→移除闭环；
- 引入 Tool guidance tier、Memory relevance ranking 与 Observation compaction；
- Provider Capability Matrix 开始保证原生 reasoning/tool/replay 能力不被 Harness 无意削弱；
- 修复 DeepSeek thinking 与 `tool_choice` 兼容冲突。

**冻结条件：** Agent Runtime 相关 Go 测试、Module/XiaoYu/Model/Project Gates 通过；前端与 Rust 在具备对应工具链的环境完成 typecheck/build/cargo test。下一阶段优先 PTY/Jobs、Tool Search、Reflection、并行 Tool 和 Subagent，而不是先堆大量角色 Prompt。

## Phase 1：平台 Shell 与完整骨架（0.1.41）

- Codex 风格 小鱼成为默认首页；
- AI 未配置时保留完整手动一键部署；
- 审批模式 UI：请求批准 / 帮我批准 / 完全访问；
- 根目录 `configs/` 配置中心；
- GSM 功能完整映射到模块矩阵和目录骨架；
- 清理每目录 README，集中架构文档；
- Installer 工程继续保留并同步版本。

**冻结条件：** `pnpm build`、`go test ./...`、`go vet ./...`、Windows 双 EXE 正式构建通过，AI 首页/手动部署入口/现有 DST 工作台无回归。

## Phase 2：全局日志中心（0.1.42）

- 左侧日志中心成为全平台唯一历史日志入口；
- 统一聚合 AI Game Manager Panel Core、操作记录、Steam、游戏、AI、节点日志；
- Go 端真实行数统计与增量缓存；
- 日志目录分页、来源/游戏/实例/Shard/状态/日期筛选；
- 正文从头/从尾/游标分页、全文搜索、级别和分类筛选；
- 单日志导出、筛选 ZIP、单条删除、筛选删除、清空历史；
- 活动日志删除保护；
- DST 工作台日志入口跳转全局 LogHub；
- 操作审计使用有界队列和 Buffered I/O，不记录秘密和完整控制台命令。

**冻结条件：** `go test ./internal/...`、`go vet ./internal/...`、`pnpm --dir frontend run build`、Windows Release（Desktop/Web/Portable/Setup）通过；Linux Release 脚本完成原生/WSL 构建验收；实机确认日志刷新、筛选、删除、导出、手工删除同步和 DST 运行日志保护。

## Phase 2.5：Linux Server Edition 与多架构部署基础（0.1.43）

- 无 GUI 的 Linux Server Edition；
- amd64 / arm64 Go 静态二进制；
- `bash scripts/build_linux.sh` 统一构建入口；
- systemd 安装、启动、停止、重启、日志、开机自启和诊断；
- 安装用 root、运行用独立 `ai-game-manager-panel` 用户；
- Docker 多架构 Server Edition；
- `configs/server.json` 统一 Server/Linux/Docker 默认值；
- Linux Desktop 与 Linux Server 发布链分离；
- 为后续实例、资源监控、RCON、SteamCMD 与 AI 远程部署建立 Server 基座。

**冻结条件：** `go test ./internal/...`、`go vet ./internal/...`、amd64/arm64 Server Edition 交叉编译通过、Shell 语法检查通过、项目骨架检查通过；至少在一个真实 Linux amd64 环境完成 systemd 安装/启停验收，ARM64 在后续真实设备或 CI Runner 补实机验证。

## Release Hotfix：Windows 正式发布链（0.1.44）

- `build/work` 保留最近一次成功编译的 Desktop/Web EXE；
- Portable 与 Setup 使用 staging，两者都成功后才晋升正式 Release；
- Inno Setup 6 提前检查；
- `BUILD-STATUS.txt` 记录失败步骤与原因；
- Portable 与 Installer 分目录；
- 增加“仅构建 Windows 安装程序”入口。

**冻结条件：** Windows 本机菜单 `4` 生成 `build/release/windows/wails/desktop/AI-Game-Manager-Panel.exe`、`build/release/windows/wails/web/AI-Game-Manager-Web.exe`、Portable ZIP、`AI-Game-Manager-Panel-<version>-Windows-x64-Setup.exe`；模拟 Installer 失败时确认旧 `build/work` 与旧完整 Release 不被删除。

## Startup Security Gate：首次管理员、会话与环境初始化（0.1.45）

- 首次数据目录仅开放一次 Owner Bootstrap，创建后永久关闭公开注册；
- `bootstrap.lock` 让账号库丢失/损坏时保持失效关闭，不自动重新开放 Owner 创建；
- PBKDF2-HMAC-SHA256 + 随机盐保存密码派生结果；
- 随机 Bearer Session 与 24 小时会话过期；
- Web `/api/v1/*` 默认受 Session 保护，仅健康检查、Bootstrap 状态、首个 Owner 创建、登录为公开端点；
- 登录管理员在用户中心创建后续账号；
- 首次登录后检测 SteamCMD、DST Dedicated Server 与 AI Game Manager Panel 运行目录；
- Windows 提供官方 SteamCMD 托管安装入口；Steam 身份、SteamCMD 登录与游戏凭据边界分离；
- Desktop/Web 共用 Application 认证与环境服务，环境初始化完成后才进入主 Shell。

**冻结条件：** `go test ./internal/...`、`go vet ./internal/...`、项目骨架/版本门禁通过；认证 Bootstrap 关闭、Session、账号持久化、bootstrap.lock 失效关闭与环境初始化均有自动测试；Windows 本机完成 `pnpm --dir frontend run build`、首次 Owner → 登录 → 环境初始化 → 主界面及 Web Bearer API 实机验收。

## Release Toolchain Hotfix：Windows BAT 编码与 Inno 自动补齐（0.1.46）

- Windows 可执行 BAT 全部 ASCII-safe；
- 中文菜单/安装器提示由 Unicode-safe PowerShell 输出；
- Inno Setup 6 支持 PATH / 安装目录 / 注册表探测；
- 优先 winget，缺失 winget 时回退官方签名安装包；
- 下载后执行 SHA-256 与 Authenticode 双校验；
- 构建失败继续保留最近一次 `build/work` 与当前已发布 Release。

**冻结条件：** Windows 本机菜单 `4` 在“未安装 Inno Setup 且无 winget”环境中能完成工具链补齐并继续执行 Release；控制台不得再出现中文乱码或中文残片被当作命令执行。

## Release Test Isolation Hotfix：Windows 环境单测隔离（0.1.47）

- 修复 `internal/deploy/environment/service_test.go` 在 Windows Release 中硬编码 `steamcmd` 的跨平台错误；
- 初始化持久化测试使用显式临时 SteamCMD 路径，不依赖系统安装状态；
- 项目路径自动探测测试根据 `runtime.GOOS` 创建 `steamcmd.exe` / `steamcmd` 临时夹具；
- 增加“没有 SteamCMD 时初始化必须 fail-closed”的回归测试；
- Windows/amd64 对环境测试执行交叉编译门禁。

**冻结条件：** Linux/Windows 平台规则下环境测试均不读取真实机器 SteamCMD；`go test ./...` 在 Windows Release 机器上不因未安装 SteamCMD 而失败。

## Release Installer Language Fallback Hotfix：Inno 语言容错（0.1.49，已由 0.1.72 取代）

0.1.49 曾采用“存在外部简中语言包则启用，否则回退英文”的兼容方案。

从 **0.1.72** 起正式策略调整为：

- 始终使用 Inno Setup 自带 `Default.isl` 作为基础；
- 中文向导文本直接由项目 `AIGameManagerPanel.iss` 覆盖；
- 不再探测或依赖 `Languages\ChineseSimplified.isl`；
- 正式安装器必须显示简体中文；
- 必须启用 `LicenseFile=EULA-zh-CN.txt`，用户不同意协议不得继续安装。

**当前冻结条件：** `check-windows-installer.mjs` 必须 PASS，Windows 实机 Setup 必须完成“欢迎页 → 强制协议 → 安装目录 → 开始菜单 → 附加任务 → 安装 → 完成”的中文流程。

## Release Test Data Isolation Hotfix：HTTP 认证测试隔离（0.1.48）

- `Application` 新增显式 `Options{Root, DataDir}` 与 `NewWithOptions`，生产入口 `New()` 保持兼容；
- HTTP 集成测试必须使用 `t.TempDir()` 注入独立运行根目录和 `dataDir`；
- 禁止测试通过 `XDG_CONFIG_HOME` 等平台特定环境变量猜测用户配置目录；
- 增加 Application 回归测试，验证账号库与 `bootstrap.lock` 必须写入注入的临时数据目录；
- Windows/amd64 对 Application 与 HTTP API 测试执行交叉编译门禁。

**冻结条件：** Windows 本机 `go test ./...` 不受 `%APPDATA%\AI Game Manager Panel` 中历史 Owner/Bootstrap 状态影响；重复运行 Release 测试结果一致，HTTP Bearer 集成测试每次从空临时账号库开始。

## Startup Security UX：登录安全密钥与可跳过环境初始化（0.1.51）

- 首次 Owner Bootstrap 继续作为唯一强制步骤，不允许跳过；
- Owner 注册页可生成 256-bit `BFK1-` 登录安全密钥，并提供复制/下载与“已安全保存”确认；
- 账号库只保存安全密钥 SHA-256 校验值，不保存明文；
- 默认仍支持账号 + 密码登录；每个账号可在用户中心独立启用安全密钥二次验证；
- 启用后登录必须同时通过账号、密码和安全密钥；
- 已登录用户重新确认当前密码后可轮换安全密钥，旧密钥立即失效；
- 首次 Step 02 环境初始化允许持久化跳过，不再阻断主 Shell；
- “运行环境”页提供后续 SteamCMD 安装、路径检测与补初始化入口。

**冻结条件：** Auth/HTTP/Environment 自动测试覆盖安全密钥生成、启用/关闭、错误密钥拒绝、轮换失效与环境跳过持久化；`go test ./...`、`go vet ./...`、前端类型检查/构建通过；Windows 实机确认 Owner 注册密钥下载、两种登录策略、跳过环境后可进入主界面并能在运行环境页补初始化。

## Dual Desktop Foundation：Wails + Electron 并行（0.1.52）

- 保留 Wails Desktop，不重写现有 Go Core；
- 新增 Electron 44.3.0 Desktop Shell；
- Electron 通过随机 loopback HTTP 连接同一 `cmd/ai-game-manager-panel-web` / Application Core；
- Vue UI 通过 Backend Adapter 自动识别 Wails / Electron / Web；
- Electron Renderer 默认开启 context isolation + sandbox，关闭 nodeIntegration；
- Electron NSIS / Portable 使用独立 staging 与 Release，不污染菜单 4 Wails 发布链；
- Windows 菜单提供 Wails、Electron 独立开发/构建以及双桌面对比构建；
- 项目检查同时加入 Electron TypeScript gate。

**冻结条件：** Wails 菜单 4 继续 1/13～13/13 通过；Electron 菜单 13 完成 Go Core sidecar、Electron TypeScript、NSIS、Portable 全链构建；两套桌面运行同一账号/环境业务契约且没有复制 Core 业务代码；实机比较窗口/DPI/中文路径/内存/启动速度后再决定默认桌面框架。

## Phase 3：统一实例管理

建立跨游戏 Instance 模型：创建、启动、停止、重启、更新、复制、删除、状态、资源指标和崩溃恢复。

## Phase 4：文件中心

文件浏览、上传下载、分块上传、压缩解压、复制移动、在线编辑、文件监听、安全根目录。

## Phase 5：终端与 RCON

统一 PTY、游戏控制台、SteamCMD、RCON，会话/命令历史和 WebSocket 实时流。

## Phase 6：任务中心与计划任务

后台任务、取消/重试、Cron、定时备份、重启、更新、命令、失败策略。

## Phase 7：备份恢复

跨游戏备份 Provider、压缩、校验、恢复、下载、删除、保留策略和自动备份。

## Phase 8：网络与组网

端口、防火墙、NAT、EasyTier/其他 Provider、内网穿透、网络诊断。

## Phase 9：运行环境

SteamCMD、Java、VC++、DirectX、Python、Docker 等环境检测和安装 Provider。

## Phase 10：插件扩展

AI Game Manager Panel 插件/Provider 生命周期、权限、版本、依赖、升级、禁用和故障隔离。

## Phase 11：节点 Agent

Windows/Linux 远程 Agent、心跳、资源上报、安全握手、任务分发和 Web 集中管理。

## Phase 12：多游戏模板

Palworld、PZ、Terraria、Minecraft、Factorio、tModLoader、通用 Steam Dedicated Server，复用公共实例/日志/备份/终端。

## Phase 13：用户与权限

最高管理员、角色、RBAC、会话、API Token、Web 登录、审计。远程 Web 在该阶段通过之前仍默认仅本机访问。

## Phase 14：Installer / Updater 深化

Windows `AI-Game-Manager-Panel-<version>-Windows-x64-Setup.exe` 与 Linux `.deb` 的基础发布流水线已提前在 0.1.42 接入。Phase 14 继续完成自动更新、签名、版本检查、差分升级、回滚策略，以及更多 Linux 包格式（如 RPM/AppImage）的发布治理。

## Phase 15：AI Tool Registry

把已完成的部署、实例、日志、文件、备份、网络、任务等能力注册为稳定 Tool，完成风险分类和审计。

## Phase 16：真实 AI Provider 与一句话部署

接入可配置 Provider。自然语言 -> Plan -> Approval -> Tool Execution -> 结果回执。手动 UI 与 AI 始终共用同一 Core。

## Phase 17：AI 智能诊断

日志/状态/配置联合诊断、修复建议、可审批自动修复；禁止 AI 绕过 Tool Registry。

## Phase 18：稳定性与 1.0 候选

性能基准、长时间运行、故障恢复、升级迁移、权限安全、安装器、Desktop/Web/Agent 全链路实机验收。
