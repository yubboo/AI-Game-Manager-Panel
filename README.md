# AI游戏管理器面板（AI Game Manager Panel）

AI游戏管理器面板是一个面向 Windows / Linux 的多游戏服务器部署、管理与 AI 辅助平台。全局产品品牌从 0.1.67 起统一为 **AI游戏管理器面板 / AI Game Manager Panel**。

> 旧项目品牌只保留在历史版本记录中；当前产品、模块和用户可见命名统一使用 **AGMP / XiaoYu**。

当前版本：**0.2.21**  
目标仓库：`https://github.com/yubboo/AI-Game-Manager-Panel.git`
当前状态：[`docs/PROJECT-STATUS.md`](docs/PROJECT-STATUS.md)  
架构边界：[`docs/PROJECT-ARCHITECTURE.md`](docs/PROJECT-ARCHITECTURE.md) / [`docs/architecture/LANGUAGE-OWNERSHIP.md`](docs/architecture/LANGUAGE-OWNERSHIP.md)

## Windows 源码开发/发行构建入口

开发者或发布者双击根目录：

```text
AI-Game-Manager-Panel.bat
```

这是源码开发/发行构建助手，不是普通用户生产启动器。普通用户只安装或运行已经打包好的完整 AGMP。开发助手仍提供 1–10 菜单：环境初始化、Wails / Electron / Web 开发、项目检查、Windows Release、Linux Server Release、预览、诊断、清理与一键发布。

## build 目录怎么理解

从 0.1.67 开始只需要记住两个目录：

```text
build/
├─ release/                 # 只放完整用户发行产物：可以分发、上传 GitHub Release
│  ├─ windows/
│  │  ├─ wails/
│  │  │  ├─ portable/      # 完整 Wails Portable ZIP（内置小鱼 Core）
│  │  │  └─ installer/     # 完整 Wails Setup EXE（内置小鱼 Core）
│  │  └─ electron/
│  │     ├─ installer/     # Electron NSIS Setup
│  │     └─ portable/      # Electron Portable EXE
│  └─ linux-server/        # Linux Server 发行包
└─ work/                    # 临时构建区：可清理，不上传 GitHub
   ├─ windows-wails/
   ├─ electron-core/
   ├─ electron-builder/
   ├─ linux-server/
   ├─ dev/
   └─ check/
```

`build/bin/` **不再是正式产物目录**。Wails CLI 可能在构建瞬间创建它，脚本会把有效文件转移到 `build/work/` / `build/release/` 后自动清掉，因此 Electron-only 发布时看到 `build/bin` 为空是正常的。




## 0.2.21 Sandbox / Capability Lease

0.2.20 已在 GitHub Actions 四条主 Job 全绿，Approved Agent → Native Terminal 接线正式冻结。0.2.21 在这条稳定链路上增加 **Host-owned Capability Lease**：server-owned XiaoYu Run 的 `shell.exec` 只有在身份、RBAC、敏感操作 step-up、审批策略与精确审批指纹全部完成后，才会获得一张短时执行租约。

租约默认仅存内存、30 秒过期、一次性使用，并绑定 `scope + tool + RunID + principal + request fingerprint`；Host 重启会自动撤销全部未使用租约，参数变化、跨 Run、跨账号、过期或重放都会 fail-closed。Native Terminal 在启动前必须原子消费租约，Go→Rust `terminal/start` 还必须携带 `capabilityLeaseId`，Rust 侧再次拒绝空租约 ID。租约不保存原始命令、Token、密码或凭据。

本版不把“Sandbox”虚报成完整 OS namespace/container 隔离；当前 Sandbox 边界是工作区 CWD + Tool timeout + 有界输出 + Host Approval + 一次性 Capability Lease。下一阶段再基于这条租约能力逐步增加更细的文件/网络/进程 Capability Scope。

## 0.2.20 Approved Agent Native Terminal Wiring

0.2.19 已在 GitHub Actions 上实现四条主 Job 全绿，Linux PTY 与 Windows ConPTY 的真实 integration/workspace tests 均通过，Native Terminal 因此冻结为跨平台稳定执行原语。

0.2.20 开始把**已经经过 Go Host 身份、RBAC、敏感操作 step-up、审批策略与精确指纹恢复**的 server-owned XiaoYu `shell.exec` 动作接入 Rust Native Terminal。Agent Run 使用一次性 `terminal/start → output/status → close` 路径，CWD 仍由工作区 Service 解析，`HostAuthorized=true` 只由 Host 内部设置，输出继续限制为 512 KiB 并继承 Tool timeout。人工 `shell.exec` 调用与兼容 `process.run` 暂时保留既有 `platform/runtime` 路径；Agent Native Terminal 失败时不静默回退，避免同一批准动作出现双执行权威。

本版不新增模型权限、不绕过审批、不开放任意路径。下一阶段进入 Sandbox / Capability Lease，把 Host 授权进一步绑定到可执行能力与作用域。

## 0.2.19 Windows ConPTY Input Pipe Convergence

0.2.18 的真实 Windows Runner 进一步证明：ConPTY 能创建、`cmd.exe` 能启动、cwd 正确、CR 规则也正确，但 `cmd.exe` 的 banner/prompt 直接出现在 Rust test harness 的父进程输出中，而写入 ConPTY input pipe 的命令没有被 shell 接收。这与微软 Terminal 对“父进程 stdout/stderr 被重定向/捕获”场景的已知行为一致。

0.2.19 在 `CreateProcessW` 边界显式启用 `STARTF_USESTDHANDLES`，并把 `hStdInput / hStdOutput / hStdError` 保持为 `NULL`，阻止 Windows 自动复制父进程标准句柄到子进程，从而强制交互 I/O 走 ConPTY pipes。CR、cwd normalize、resize lock 与 Host authorization 均保持不变；本版不扩大模型可见 Terminal 权限。

## 0.2.18 Windows ConPTY Input Convergence

0.2.17 已经证明 Windows ConPTY 可以在真实 Windows Runner 上通过 `cargo fmt` 与 `cargo check --workspace --locked`，`cmd.exe` 也能在正确仓库 cwd 启动。剩余失败只集中在输入：0.2.17 把 `appendNewline` 设为 CRLF，但真实终端 Enter 在 ConPTY 输入流中应为单个 CR（`\r` / `0x0D`）。

0.2.18 只修正这一输入语义：Windows ConPTY 使用 CR，Linux PTY / fallback 继续使用 LF；对应单元测试和 Terminal/PTY Gate 同步冻结 CR 规则。本版同时把 GitHub 主动检查与完整源码交付流程写入开发规范，后续 AI 必须先查看 `main` 最新提交和对应 Actions，再决定下一版改动。

## 0.2.17 Windows ConPTY Runtime Convergence

0.2.16 已经让 Windows Rust Runtime 真正通过 `cargo fmt` 与 `cargo check --workspace --locked`，说明 ConPTY FFI/ABI 编译边界已经站稳。Windows Runner 随后进入真实 integration test，并暴露两个运行时语义问题：Rust `canonicalize()` 产生的 `\\?\D:\...` verbatim cwd 会让 `cmd.exe` 把当前目录视为 UNC 风格路径并回退到 `C:\Windows`；同时 `terminal/write` 的 `appendNewline` 统一发送 `\n`，不足以稳定表达 Windows Console 的 Enter 语义。

0.2.17 在不扩大任何 Agent 权限的前提下，只收敛 Windows Native Terminal：`CreateProcessW` 边界会把 verbatim 本地盘符路径规范化回普通 Win32 drive path，Windows ConPTY 的 `appendNewline` 使用 CRLF；同时修正 Terminal resize 的 MutexGuard 作用域，并让 stdio `Pipe` variant 只在 fallback 平台编译。Windows ConPTY 仍以 GitHub Windows Runner 的 integration + workspace tests 全绿作为稳定完成证据。

## 0.2.15 Native Terminal CI Convergence

0.2.15 不继续叠加新 Runtime 能力，先收敛 0.2.14 的跨平台 Native Terminal 验证链。GitHub 0.2.14 已确认 Windows Helper、Linux Headless、Go test/vet、Agent Bench 与 Terminal/PTY structure gate 均通过；Safety 与 Windows Rust Runtime 都只在 `cargo fmt --check` 的 `pty_windows.rs` 两处排版差异处停止，因此 ConPTY/Linux PTY integration 尚未真正执行。

本版按 GitHub Runner 输出修正 `pty_windows.rs`，并调整 Actions：Rust format 即使失败，后续 `cargo check`、Linux/Windows PTY integration、workspace tests 仍会在未取消的情况下继续执行。这样一次 CI 就能暴露格式、编译和平台 integration 的全部真实问题，不再因为前序格式失败串行隐藏后续结果。`AGMP-GitHub` 也会在本机存在 Cargo 时先执行 `cargo fmt --check`；没有 Cargo 时明确提示由 GitHub Runner 做权威验证。

## 0.2.14 Windows ConPTY / Cross-platform Native Terminal

0.2.14 在 0.2.13 Linux PTY 基础上补齐 Windows 原生终端 backend：Rust XiaoYu Runtime 在 Windows 使用 `CreatePseudoConsole` / `ResizePseudoConsole` / `PROC_THREAD_ATTRIBUTE_PSEUDOCONSOLE` 创建 ConPTY，并继续复用既有 `terminal/start|get|list|write|output|resize|close` 协议。Linux 继续使用 `linux-pty-v1`，Windows Snapshot 目标为 `backend=windows-conpty-v1`，其他未验证平台才保留 `stdio-pipe-v1` fallback。

本版新增独立 **Windows Rust Runtime + ConPTY** GitHub Actions Job，真实执行 Windows Rust `cargo check/test` 与 ConPTY integration test；Terminal start、每次输入、resize 仍必须先经过 Go Host RBAC/审批边界。0.2.13 GitHub Runner 唯一的 Rust import-order `cargo fmt` 差异也一并修复。ConPTY 是否完成以 Windows Runner 实际全绿为准。

## 0.2.13 Linux Native PTY Foundation

0.2.13 在 0.2.12 已经稳定的 Terminal protocol 上把 Linux backend 从普通 stdio pipe 升级为**真实 PTY**：Rust 直接创建 PTY master/slave、建立独立 session/controlling terminal，并继续通过 Persistent `xiaoyu rpc` Worker 保持长期交互状态。`terminal/start|get|list|write|output|resize|close` 仍复用同一套协议，不再另建第二套终端核心。

Linux Snapshot 明确返回 `backend=linux-pty-v1`，并支持 Host-authorized rows/cols resize；CI 会真实验证 `test -t 0` 与 `stty size`。Windows 当前仍明确返回 `stdio-pipe-v1` fallback，**本版不虚报 ConPTY 已完成**。Terminal Start、每次输入和 resize 都必须先经过 Go Host 授权，不能利用长期 shell 绕过三种审批模式。下一步将建立 Windows Rust Runtime CI，再实现并验证 ConPTY。

## 0.2.12 Rust Interactive Terminal Session

0.2.12 在 0.2.11 已全绿的 Persistent Rust Runtime Worker 上新增 **Interactive Terminal Session v1**：Rust Runtime 可以启动长期交互进程、持续读取有界 stdout/stderr、向同一会话发送输入、查询状态并关闭会话。`terminal/start|get|list|write|output|close` 通过同一个 `xiaoyu rpc` Worker 保存状态。

该阶段明确使用 `backend=stdio-pipe-v1`，只建立 PTY-ready 的协议和生命周期，不把普通 pipe 冒充 Native PTY/ConPTY。

## 0.2.11 Persistent Rust Runtime Worker

0.2.11 把 0.2.10 的 Rust Session / Job 原语真正接到长期存活的 Go ↔ Rust stdio Worker 上：AGMP 启动时会预热 `xiaoyu rpc`，后续 `Tool Search / Brain / Session / Job` RPC 复用同一 Rust 进程，因此 Session/Job 状态不会再因为“每个 RPC 都新启一个进程”而丢失。应用关闭时会监督退出 Worker；RPC 超时或管道损坏会终止失效 Worker，下一次调用再重新建立。

Go Host 仍然是身份、RBAC 与三种审批模式的最终权威。Rust Job Bridge 在 Go 和 Rust 两侧都要求 Host authorization，当前不会把 `jobs/start` 直接暴露成绕过审批的模型 Tool。0.2.11 同时修复 Windows 源码同步时 Robocopy 的中文路径乱码：原生 Robocopy 报表写入临时 Unicode 日志，交互控制台只由 PowerShell 输出。

## 0.2.10 Rust Session / Long-running Job Runtime

0.2.9 已经冻结 Rust/Go/Vue 的长期职责边界并提交真实依赖锁。0.2.10 开始继续把 XiaoYu 的通用“手脚”迁入 Rust：新增真实 Session Registry 与 Long-running Job Runtime，提供持久 RPC 进程内的 `session/get|list|close`、`jobs/start|get|list|output|cancel`。Job 输出采用有界缓冲，支持状态、PID、退出码、输出游标和取消。

Rust Job 不是新的高权限后门：`jobs/start` 默认拒绝未携带 `hostAuthorized=true` 的请求，工作目录必须限制在 Runtime Root 内。当前这些 RPC 是 Native Runtime 基础设施，**尚未直接暴露给模型**；现有 `shell.exec` 仍经过 Go Host 的 RBAC / 三种审批模式。下一步会建立 Go ↔ Rust 的持久 RPC Worker，再把批准后的长任务逐步切换到 Rust Job Runtime。

本版同时修复 0.2.9 GitHub Actions 唯一红灯：新增 Rust Tool Search 代码按 `cargo fmt` 标准整理。

## 0.2.9 Rust-first XiaoYu Runtime / 可复现依赖基线

0.2.8 已在 GitHub Actions 首次实现 Safety、Linux Headless + Web + XiaoYu、Windows Helper + Encoding 三个 Job 全绿。0.2.9 以这条稳定基线为起点，不做大爆炸式重写，而是正式冻结语言职责：**Rust = XiaoYu Agent Runtime / Native Execution / Security Boundary，Go = AGMP Product Host / Game Domain Services，Vue/TypeScript = UI**。

本版新增 `docs/architecture/LANGUAGE-OWNERSHIP.md`，并把该规则写入 `AGENTS.md` 与架构 Gate。Rust `xiaoyu-core` 不再被定义为 Brain-only；第一块真实迁移能力是 **Tool Search / Capability Discovery**：Rust 新增 `tools/search` RPC 和确定性 Tool ranking，Go `agmp.capability.search` 优先委托 Rust，Rust 不可用时只保留迁移期兼容排序。

同时正式提交 GitHub Runner 在 0.2.8 真实生成的 `go.mod/go.sum`、`rust/Cargo.lock`、Frontend/Electron `pnpm-lock.yaml`。CI 切换到 Cargo `--locked`、pnpm `--frozen-lockfile`、Go module verify + tidy diff，防止依赖图在不同机器悄悄漂移。

当前 Windows 主桌面仍保持 Wails + Vue + Go Host，并内置 Rust XiaoYu Runtime。不会为了 GitHub Rust 百分比立即切换 Tauri；后续 Tool Search、Session/Job、PTY、Sandbox、Reflection、Subagent 会逐步 Rust-first。

语言职责见 [`docs/architecture/LANGUAGE-OWNERSHIP.md`](docs/architecture/LANGUAGE-OWNERSHIP.md)，版本历史统一见 [`docs/PROJECT-HISTORY.md`](docs/PROJECT-HISTORY.md)。

## 0.2.8 CI 收敛 / Runtime 输出完整性

0.2.8 聚焦 GitHub Actions 的真实失败，不扩展新业务功能。修复 Process Runtime 在 `StdoutPipe/StderrPipe` 尚未完全读取时提前 `cmd.Wait()` 的竞态：现在先等待输出读取 goroutine 完成，再回收进程，确保 `Session.Wait()` 返回后 `History()` 已包含最终输出。对应 Runtime Manager 测试增加显式 ready 同步并进行重复压力验证。

源码同步与一键推送新增 **Duplicate Source Gate**。已重命名的 `server_xiaoyu_models_test.go` / `server_xiaoyu_models_release_test.go` 会在同步、提交前自动清理；同一 Go package 若出现内容完全相同的 `*_test.go` 文件会直接拒绝推送，避免旧文件残留导致测试函数 redeclared。

Go 基线提升到 **1.25.0**，与 Wails v2.15.0 的最低要求一致；GitHub CI 在 Go 测试与 Linux Headless 构建前执行 `go mod tidy + go mod download`，消除旧 `go.sum` 只含 Wails 主模块两行导致的传递依赖校验失败。依赖锁文件的完全冻结将在 CI 首次全绿后单独收敛，避免在未验证依赖图时提交伪锁文件。

版本历史统一维护在 [`docs/PROJECT-HISTORY.md`](docs/PROJECT-HISTORY.md)。XiaoYu Agent Runtime 当前设计说明见 [`docs/development/XIAOYU-AGENT-RUNTIME.md`](docs/development/XIAOYU-AGENT-RUNTIME.md)。

## Electron 与 Wails 体积

Electron 必须携带 Chromium + Node.js Runtime，因此安装包天然会比 Wails 大。0.1.67 已做安全可维护的瘦身，但不会为了追求极端体积使用 UPX 或删除 Electron 必需资源。

如果优先考虑 **体积小**，使用 Wails Release；如果优先考虑 **Electron 运行一致性与兼容性**，使用 Electron Release。

## GitHub 安全

公开仓库只能提交源码与许可证公钥。禁止提交真实许可证私钥、CDK/BFLC/activation、管理员账户与安全密钥、DST/Steam Token、运行数据、日志、构建产物、`.env` 等敏感内容。

本地 `项目检查` 会运行 GitHub Safety Gate；GitHub Actions 也会再次检查。详细规则见 `docs/development/GITHUB-SAFETY.md`。
