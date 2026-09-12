# AI-Game-Manager-Panel 项目历史

## AI-Game-Manager-Panel 0.2.15

### Native Terminal CI Convergence

- 0.2.14 已成功推送；Windows Helper 与 Linux Headless 全绿，Go tests/vet、Agent Bench、Terminal/PTY structure gate 全绿。
- Safety 与 `Windows Rust Runtime + ConPTY` 均在 `cargo fmt --check` 停止；GitHub Runner 精确指出 `pty_windows.rs` 两处 rustfmt 差异，后续 Rust check/PTy integration/tests 因前序失败被跳过，并非自身失败。
- 0.2.15 按 Runner 输出修正 import 顺序和长 `assert_eq!` 布局。
- CI 中 Rust check、Linux PTY integration、Windows ConPTY integration、workspace tests 改为在 `!cancelled()` 时继续执行，避免格式失败隐藏真正编译/平台问题。
- `check-xiaoyu-terminal.mjs` 固定上述 CI 收敛规则；`AGMP-GitHub` 在检测到本机 Cargo 时执行 `cargo fmt --check`，无 Cargo 时明确交给 GitHub Runner。
- 本版不新增 Agent 权限或模型可见执行能力；Native Terminal 仍必须在双平台 integration 全绿后才能进入下一阶段。

### 冻结目标

- Safety：rustfmt、cargo check、Linux PTY integration、workspace tests 全绿；
- Windows Rust Runtime + ConPTY：rustfmt、cargo check、ConPTY integration、workspace tests 全绿；
- Windows Helper 与 Linux Headless 不回退。

## AI-Game-Manager-Panel 0.2.14

### Windows ConPTY / Cross-platform Native Terminal

- 修复 0.2.13 GitHub Actions 唯一剩余的 Rust import-order rustfmt 差异。
- Rust 新增 `pty_windows.rs`，实现 Windows ConPTY 创建、进程启动、输入输出、状态、终止和 resize。
- Linux 继续使用 `linux-pty-v1`；Windows 使用 `windows-conpty-v1`；其他平台才保留明确的 stdio fallback。
- ConPTY 继续复用 `terminal/start|get|list|write|output|resize|close`，没有制造平台专用的第二套协议。
- 新增独立 `Windows Rust Runtime + ConPTY` GitHub Actions Job，要求 Windows Runner 真实 build/test ConPTY backend。
- Host 授权仍是 start/write/resize 的硬边界；Rust Native Terminal 不成为绕过三种审批模式的后门。

### 验证目标

- Linux Safety：rustfmt / cargo check / Linux PTY integration / Rust tests 全绿；
- Windows Rust Runtime + ConPTY：cargo fmt/check、Windows ConPTY integration、workspace tests 全绿；
- Windows Helper 与 Linux Headless 不回退。

## AI-Game-Manager-Panel 0.2.13

### 主题
Linux Native PTY Foundation / Terminal resize。

### 主要变化
- 修复 0.2.12 GitHub Actions 唯一红灯：`terminal.rs` 两处 rustfmt 布局差异；
- 保持 0.2.12 的 `terminal/*` RPC contract，不创建第二套终端协议；
- Linux Rust Runtime 新增真实 PTY backend：`posix_openpt` / slave / `setsid` / controlling TTY；
- Linux Snapshot 返回 `backend=linux-pty-v1`；非 Linux 继续显式 `stdio-pipe-v1` fallback；
- 新增 `terminal/resize`、rows/cols protocol 与 Go Host Bridge；resize 继续要求 Host authorization；
- PTY master 使用 close-on-exec；关闭会话时终止 PTY process group；Linux PTY slave 关闭后的 EIO 按正常 EOF 处理；
- Linux Rust tests 验证 `test -t 0` 与 `stty size`，避免只靠结构 Gate 虚报 PTY；
- Terminal Gate 升级为 Linux Native PTY Gate，并明确禁止在 Windows Rust CI 验证前声明 ConPTY 已完成。

### 0.2.12 GitHub 基线
- Windows Helper + Encoding：PASS；
- Linux Headless + Web + XiaoYu：PASS；
- Go tests / Go vet / Agent Bench / Terminal Gate：PASS；
- Safety 唯一失败：`cargo fmt --check` 在 `terminal.rs` 两处排版差异处停止，Rust check/test 因前序失败被跳过。

### 下一阶段
- 0.2.14：增加 Windows Rust Runtime CI，实际编译并集成测试 ConPTY；只有 Windows Runner 通过后再发布 Windows native PTY capability。

---


## AI-Game-Manager-Panel 0.2.12

### 主题
Rust Interactive Terminal Session / PTY-ready protocol foundation。

### 主要变化
- 基于 0.2.11 全绿 Persistent Worker 新增 Rust `TerminalManager`；
- 新增 `terminal/start|get|list|write|output|close` RPC；
- Terminal 输出有界、可 cursor 增量读取；
- 启动与每次输入都要求 Host authorization，Go/Rust 双层拒绝未授权请求；
- Terminal 工作目录复用 Runtime Root / Session scope；
- Snapshot 明确 `backend=stdio-pipe-v1`，本版不虚报 Native PTY/ConPTY；
- 新增 `check-xiaoyu-terminal.mjs` 并接入 CI / Windows Helper / Push Gate。

### GitHub 验证结果
- Windows Helper + Encoding：PASS；
- Linux Headless + Web + XiaoYu：PASS；
- Go test/vet、Agent Bench、Terminal Session Gate：PASS；
- Safety 仅 `cargo fmt --check` 失败：`terminal.rs` 两处布局差异；因此该 run 中 Rust check/test 被跳过；功能路径与其余 Gate 未发现回退。

---

## AI-Game-Manager-Panel 0.2.11

### 主题

Persistent Rust Runtime Worker / Windows 源码同步编码修复。

### 主要变化

- Go Host 新增长期 `xiaoyu rpc` Worker，Rust Session/Job 状态可跨 RPC 保持；
- Startup 预热、Shutdown 关闭，超时/断管自动回收失效 Worker；
- 新增 Host-internal Session/Job Bridge，`jobs/start` 仍要求 Host authorization；
- Worker stderr 有界保留，避免诊断输出无限增长；
- 修复 0.2.10 GitHub rustfmt 差异；
- Robocopy 原生日志改写 Unicode 临时日志，控制台不再把 `H:\一键部署` 显示成乱码；
- 新增 Persistent Worker Gate。

### 验证

- 0.2.10 GitHub：Windows Helper PASS；Linux Headless + Web + XiaoYu PASS；Go test/vet 与 Session/Job Gate PASS；Safety 仅在 rustfmt 差异处停止。
- 0.2.11 本地：Node 架构 Gate 与可运行的 Go 兼容检查；最终 Rust/Go1.25/pnpm 由 GitHub Actions 权威验证。

---

## AI-Game-Manager-Panel 0.2.10

### 主题
Rust Session / Long-running Job Runtime foundation。

### 主要变化
- 修复 0.2.9 GitHub Actions 的 rustfmt 唯一红灯；
- Rust 新增 Session Registry：create/get/list/close；
- Rust 新增 Long-running Job：start/get/list/output/cancel；
- Job 输出有界、支持游标读取、PID/退出状态与取消；
- Job 启动要求 Host authorization，cwd 限制在 Runtime Root；
- 新增 `check-xiaoyu-jobs.mjs` 并接入 CI/Windows Helper/一键推送；
- 当前仅建立 Rust 原语，Go `shell.exec` 尚未切换，下一阶段先做 persistent RPC worker。
- 源码同步助手新增 dirty bundle 防护：明确排除 runtime 用户数据、依赖缓存和构建产物，源码 ZIP 仅交付 Git 应跟踪内容。

---
本文件是 AGMP 唯一版本历史入口。自 0.2.3 起，不再为每个版本新增独立 Release Notes / Validation / Completion / Prompt 历史文件；长期有效的规范继续维护在 `AGENTS.md` 与当前架构/开发文档中。

> 版本顺序采用 `0.2.1 ... 0.2.100 -> 0.3.0`。新版本记录追加到本文件顶部。

## AI-Game-Manager-Panel 0.2.9

### Rust-first Agent Runtime / Dependency Freeze

- 0.2.8 GitHub Actions 首次确认 Safety、Linux Headless + Web + XiaoYu、Windows Helper + Encoding 三个 Job 全绿，Rust fmt/check/test、Go test/vet、Frontend build、Linux headless 集成全部通过。
- 正式冻结 `docs/architecture/LANGUAGE-OWNERSHIP.md`：Rust 负责 XiaoYu Agent Runtime / Native Execution / Security Boundary；Go 负责 AGMP Product Host / Game Domain Services；Vue/TypeScript 负责 UI。
- Rust `xiaoyu-core` 从历史 Brain-only 定位进入 Agent Runtime 渐进迁移期；不一次性重写现有 Go Host/Process Runtime。
- 新增 Rust `tool_search.rs`、`ToolSearchRequest/Hit/Response` 与 `tools/search` JSON-RPC，作为第一块 Rust-first Capability Discovery。Go `agmp.capability.search` 优先使用 Rust ranking，失败时只使用迁移期兼容排序。
- 收回 0.2.8 GitHub Runner 真实生成的 `go.mod/go.sum`、`rust/Cargo.lock`、Frontend/Electron `pnpm-lock.yaml`，正式进入可复现依赖基线。
- GitHub CI 改为 Cargo `--locked`、pnpm `--frozen-lockfile`、Go `mod verify + tidy diff`。新增 Language Ownership Gate 与 Dependency Lock Gate，并纳入本地推送/Windows Helper。
- Windows 主桌面继续 Wails + Vue + Go Host + 内置 Rust XiaoYu Runtime；不为了 Rust 百分比立即迁移 Tauri。

### 下一阶段

- 继续 Rust-first 迁移 Session/Job、PTY、Sandbox，再逐步处理 Reflection / Experience 与 Subagent。
- 每迁移一个权威路径必须先有 protocol tests + Agent Bench + CI，再删除旧 Go 兼容实现。

---
## AI-Game-Manager-Panel 0.2.8

### CI 全绿收口与 XiaoYu Agent Bench

- 0.2.7 GitHub Actions 已确认 Windows Helper + Encoding、Linux Headless + Web + XiaoYu 全绿；Safety Job 的 Go test、Go vet、开发许可证测试也全部通过，唯一剩余红灯为 Rust `cargo fmt --check`。
- 按 GitHub Rustfmt 输出修正 `xiaoyu-core` / `xiaoyu-protocol` 源码格式，目标是不再让格式问题阻断后续 `cargo check` / `cargo test`。
- 新增 `internal/xiaoyu/host/bench_test.go` 与 `scripts/common/check-xiaoyu-agent-bench.mjs`，把“领域路径失败后切换通用 fallback”“mutation 后必须读回验证”“审批后恢复原 Tool Call”变成独立 Agent Bench。
- GitHub Safety Job 新增可见的 `XiaoYu Agent Bench` 测试步骤；Agent Bench 不用 Tool 数量衡量智能，而是验证 Agent Loop 是否保持恢复、验证和审批边界。
- Headless Job 生成 Go/Rust/Frontend/Electron 依赖锁快照并上传 `agmp-dependency-locks` Artifact；下一版从真实联网 Runner 收回 `go.sum`、`Cargo.lock`、两个 `pnpm-lock.yaml`，正式切换到 locked/frozen 构建。
- 清理 Headless CI 中重复执行的 Duplicate Source Gate。

### 下一阶段

- 以 0.2.8 GitHub Actions 首次全绿为冻结条件。
- 下载 `agmp-dependency-locks` Artifact，进入 0.2.9 的依赖冻结与 `--locked` / `--frozen-lockfile`。
- 完成可复现依赖基线后，再继续 Windows Wails/Electron 真编译矩阵与 XiaoYu Tool Search / Jobs / Reflection。

---
## AI-Game-Manager-Panel 0.2.7

### CI 收敛

- 修复 Linux/CI 可复现的 Process Runtime 输出竞态：`Session.waitLoop` 先等待 stdout/stderr reader 完整 drain，再调用 `cmd.Wait()` 回收子进程。`Session.Wait()` 成功后，`History()` 现在保证可读取最终输出。
- `TestManagerConvenienceAPIsAndStopAll` 增加显式 `ready` 同步，移除依赖调度时序的偶发失败；本地对该测试连续运行 100 次通过。
- 新增 `scripts/common/check-duplicates.mjs`：禁止已重命名的旧 Model Center 测试文件回归，并检测同一 Go package 下内容完全相同的 `*_test.go`，防止重复 Test 函数声明。
- `sync-agmp.ps1` 与 `push-agmp.ps1` 都会主动删除 `server_xiaoyu_models_test.go` / `server_xiaoyu_models_release_test.go` 旧路径，解决 Git 工作副本覆盖更新后旧文件残留。
- Go 基线从 1.23 提升至 1.25，与 Wails v2.15.0 的最低 Go 版本对齐。GitHub Actions 在 Go test / Linux Headless build 前执行 `go mod tidy` 与 `go mod download`，补齐旧 `go.sum` 不完整导致的传递依赖校验失败。
- Duplicate Source Gate 已接入 Linux Safety、Linux Headless、Windows Helper 和本地一键推送。

### 验证

- 0.2.6 GitHub Actions 已确认：Windows Helper + Encoding 全绿、Frontend `vue-tsc + vite build` 通过、Rust XiaoYu `cargo build --release` 通过、Linux AGMP Core build 通过。
- 0.2.6 剩余红灯已定位为：旧 Model Center 测试文件残留、Wails 传递依赖 `go.sum` 缺失、Process Runtime history 偶发为空。本版逐项处理。
- 本地 `TestManagerConvenienceAPIsAndStopAll -count=100`：PASS。
- 本版目标：推送后让 GitHub Actions 首次全绿；全绿后再冻结 `Cargo.lock` / `pnpm-lock.yaml` 与完整 Go module graph。

---
## AI-Game-Manager-Panel 0.2.6

### 修复

- 修复 `push-agmp.ps1` 项目完整性检查与 `check-source-tree.mjs` 的 Rust 最小文件数不一致：实际 `rust/crates` 当前为 5 个源码/配置文件，PowerShell 错误要求 6 个，导致 `AGMP-GitHub.bat` 启动后立即红色失败并退出。
- `AGMP-GitHub.bat` 现在捕获 PowerShell 非零退出代码；发生错误时会 `pause` 保留窗口，不再“一闪而过”，便于直接查看并反馈真实错误。
- `AGMP-Sync.bat` 作为一次性同步入口，无论成功或失败都会停留显示结果；`sync-agmp.ps1` 在脚本已经位于目标工作副本时明确提示“无需同步，直接运行 AGMP-GitHub.bat”。
- BAT 继续严格保持 ASCII + CRLF + 无 BOM；中文交互仅由 UTF-8 BOM + CRLF 的 PowerShell 脚本输出。

### 验证

- `scripts/common/check-source-tree.mjs` 与 PowerShell Project Integrity Guard 的 Rust 源码树阈值统一为 5。
- 根 BAT 编码重新验证为 ASCII / CRLF / 无 BOM。
- 重新执行 Source Tree、Naming、GitHub Safety、Windows Helper 等项目 Gate。
- 重新执行非 Wails Go `go test` / `go vet`；联网 Frontend/Rust/Wails/Linux Headless 继续由 GitHub Actions 验证。

### 下一阶段

- 推送 0.2.7 后以 GitHub Actions 的真实结果继续修复，直到 Safety、Linux Headless + Web + XiaoYu、Windows Helper + Encoding 全绿。

---

## AI-Game-Manager-Panel 0.2.5

### 修复与优化

- 强化根 `AI-Game-Manager-Panel.bat`：关键 PowerShell 入口缺失时明确提示源码树不完整并暂停，错误不再一闪而过；继续坚持 BAT ASCII + CRLF + 无 BOM。
- 新增 `AGMP-Sync.bat + sync-agmp.ps1`，使用 Windows `robocopy` 从完整解压目录同步源码到专用 Git 工作副本，保留 `.git` 与未跟踪本机数据，并移除新版已不存在的旧 Git 跟踪源码。用于替代 Explorer 手工覆盖数百文件，降低 `0x80004005`、跳过文件和漏目录风险。
- 新增 `docs/NAMING-CONVENTIONS.md`，正式规定源码文件、测试文件、路径、脚本、文档和版本包命名规则。
- 将 `internal/bridge/httpapi/server_xiaoyu_models_release_test.go` / `server_xiaoyu_models_test.go` 收敛为 `models_release_test.go` / `models_test.go`，示范“目录承担命名空间，文件名只表达职责”的规则。
- 新增 `scripts/common/check-naming.mjs`：普通文件名 >40 字符、测试文件名 >48 字符失败；仓库相对路径 >180 字符告警、>220 字符失败。
- 新增 `scripts/common/check-source-tree.mjs`：关键源码文件/目录缺失时，本地 Gate 与 GitHub Actions 均在构建前失败。
- `push-agmp.ps1` 将 Source Tree Gate / Naming Gate / GitHub Safety Gate 纳入一键推送前检查，并要求同步助手、命名规范与核心源码完整存在。

### 验证

- 本地 Node 项目 Gate：全部非 Publisher Gate 重新执行。
- Windows Helper Gate 覆盖三个根 BAT/PS1 工作流的编码与关键入口。
- 使用全新临时 Git 仓库验证源码包可完整 `git add`，`scripts/`、`rust/`、`runtime/README.md`、Frontend logs/instances 与 internal ops logs 不再丢失。
- Go 非 Wails 包执行 `go test` / `go vet`；联网 Frontend/Rust/Wails/Linux Headless 继续由本版 GitHub Actions 最终验证。

### 下一阶段

- 以 GitHub Actions 全绿为当前第一目标；再推进 Windows Wails/Electron CI 与 XiaoYu Agent Bench。

---

## AI-Game-Manager-Panel 0.2.4

### 修复

- 修复 0.2.3 推送后的 GitHub 工作副本完整性问题：实际 commit 中 `scripts/`、`rust/` 与 `runtime/README.md` 被删除，导致两个 GitHub Actions Job 在第一批 Node Gate 处全部失败。
- 经重新检查，0.2.3 交付 ZIP 本身包含上述目录；问题发生在 Git 工作副本更新/推送链路没有检查“关键源码是否完整”。0.2.4 因此把完整性检查放到 Commit 之前，而不是等 CI 才发现。
- `push-agmp.ps1` 新增 Project Integrity Guard：检查 `.github/workflows/safety.yml`、Rust XiaoYu Runtime、核心 scripts Gate、Frontend/Go/XiaoYu 关键入口及关键源码树最小文件数量。
- 新增 Staged Deletion Guard：关键源码删除默认直接拒绝；超过阈值的非历史大规模删除默认拒绝，避免覆盖源码时漏目录后被 `git add -A` 误提交。历史文档收敛产生的预期删除不计入误删保护。
- GitHub Actions 的 `safety` 与 `Linux Headless + Web + XiaoYu` 两个 Job 都增加 Source tree integrity guard，在运行 Node/Rust/Frontend Gate 前先验证完整源码树。
- 继续保持 BAT 为 ASCII + CRLF + 无 BOM，PowerShell 为 UTF-8 BOM + CRLF。

### 验证

- 0.2.3 GitHub Actions 的全红已定位为缺失 `scripts/common/*.mjs` 等文件导致的 `MODULE_NOT_FOUND`，不是 XiaoYu Agent Runtime 本身测试失败。
- 0.2.4 本地重新执行项目非 Publisher Gates、Go test / Go vet，并验证完整 ZIP 中包含 `scripts/`、`rust/`、`runtime/README.md`、Frontend logs/instances 与 internal ops logs 源码。
- Frontend 完整联网构建、Rust Cargo 全套与 Linux Headless 集成继续由本版推送后的 GitHub Actions 进行最终联网验证。

### 下一阶段

- 以 GitHub Actions 全绿为 0.2.4 验收目标；再继续补 Windows Wails/Electron CI 与 XiaoYu Agent Bench。

---

## AI-Game-Manager-Panel 0.2.3

### 更新

- 修复 Git `.gitignore` 误伤源码目录的问题：根运行数据规则改为 `/logs/`、`/instances/` 等 root-anchored 写法，不再把 `internal/ops/logs/`、`frontend/src/features/logs/`、`frontend/src/features/instances/` 当运行数据忽略。
- 将版本历史收敛为单一 `docs/PROJECT-HISTORY.md`；历史 Release Notes、Validation、Completion 与 Prompt 归并到本文件，不再持续制造每版本独立历史文件。
- 新增/更新 `AGMP-GitHub.bat + push-agmp.ps1` GitHub 工作台。BAT 保持 ASCII + CRLF + 无 BOM，仅负责启动 PowerShell，彻底规避 CMD 对 UTF-8 BOM/中文批处理的乱码与 `锘緻echo` 问题。
- GitHub 推送前增加源码误忽略检查、敏感文件检查、项目 Safety Gate、远端 rebase 同步；脚本拒绝自动 force push。
- 当前源码包继续包含 0.2.2 XiaoYu Agent Runtime / Guided Autonomy 的完整实现。

### 验证

- 已确认 0.2.2 GitHub Actions 能成功安装 Rust 并完成 `xiaoyu-core --release` 构建。
- 0.2.2 GitHub Actions 暴露的 Frontend/Project Layout 失败根因为 `.gitignore` 误忽略源码目录，本版从根因修复，并用全新临时 Git 仓库验证关键 `logs/instances` 源码能够被 `git add` 正常跟踪。
- 0.2.3 本地执行全部 19 个非 Publisher Release-Key Gate：PASS。
- 0.2.3 本地执行 44 个非 Wails `internal` Go package + 2 个 `cmd` package 的 `go test`：PASS；同一组 `go vet`：PASS。
- `AGMP-GitHub.bat` 已通过 ASCII + CRLF + 无 BOM Gate；`push-agmp.ps1` 已通过 UTF-8 BOM + CRLF Gate，专门防止 CMD `锘緻echo` / 中文乱码回归。
- 完整 Frontend `vue-tsc/vite build`、Rust `cargo test`、Wails Windows build 继续由推送后的 GitHub Actions 联网环境验证。

### 下一阶段

- 让 GitHub Actions 全绿，并补 Windows Wails/Electron 构建矩阵。
- 建立 XiaoYu Agent Bench，以真实任务成功率而不是 Tool 数量衡量智能化能力。

---

## AI-Game-Manager-Panel 0.2.2

### Release Notes

<!-- historical source: docs/releases/0.2.2.md -->

> XiaoYu Agent Runtime / Guided Autonomy foundation.

## XiaoYu intelligence

- Reframed XiaoYu as a persistent Agent whose configured provider model supplies the general intelligence. Expert / Skill / Memory / Experience are priority guidance, not hard capability limits.
- Added `tier=domain/general/control/fallback` guidance to model-facing tools. Domain tools stay preferred; general tools and controlled Shell may be used when a specialized path is missing or insufficient.
- Increased the default autonomous run budget to 64 steps / 48 tool calls / 8 recoverable failures, while keeping the existing approval modes as the execution authority.
- Added bounded Observation projection so large command output and old receipts do not consume the whole model context. Durable Run receipts remain intact.
- Memory selection now ranks server/instance/task/session/experience memories by scope, relevance, confidence and recency before prompt assembly.

## General hands

- Added workspace-scoped `fs.stat`, `fs.write`, `fs.replace`, `fs.mkdir`, and `fs.remove` model tools with traversal/symlink boundary checks.
- Added `shell.exec` as XiaoYu's controlled general fallback. `process.run` remains a manual compatibility alias.
- `shell.exec` is still subject to RBAC, step-up, timeout/output bounds and the existing ask/risk/full approval policy. Capability is no longer removed merely to enforce safety.
- Added Runtime Manager tools: `environment.resolve_runtime`, `environment.set_default_runtime`, and `environment.remove_runtime` so install/uninstall/default changes can form a real closed loop.

## Native model preservation

- Expanded model capability metadata for tool choice, native tool kinds, reasoning replay, parallel-call support and thinking/tool-choice compatibility.
- Fixed DeepSeek native thinking requests so `tool_choice` is omitted while thinking is active unless the configured endpoint explicitly declares compatibility. `reasoning_content` replay remains preserved across tool turns.
- Frontend model capability types now expose the same capability matrix.

## Architecture direction

0.2.2 deliberately does not fake multi-tool parallelism, PTY/jobs, hosted web/computer tools or subagents. Those remain next-stage Agent Runtime work. The goal of this release is to stop treating a missing specialized button as XiaoYu's intelligence boundary and establish the general-capability fallback needed for later Tool Search, Jobs, Reflection and XiaoYu Studio.

### Validation

<!-- historical source: AGMP-0.2.2-Validation.md -->

## Scope

This release is the first XiaoYu Agent Runtime / Guided Autonomy pass. It changes capability exposure and context handling, so validation focuses on Tool Registry, approval boundaries, provider-native request compatibility, workspace file safety and Agent-loop regression.

## Required checks

- Go unit tests for `internal/xiaoyu/host`, `internal/ops/files`, `internal/app`.
- All non-Wails internal Go tests and `go vet` where the local toolchain can resolve dependencies.
- Project layout, module, XiaoYu core, model center and updater gates.
- Frontend `vue-tsc --noEmit` / build when pnpm dependencies are available.
- Rust `cargo test` when the Rust toolchain is installed.

## Key regressions covered

- DeepSeek thinking mode must not force incompatible `tool_choice`.
- DeepSeek reasoning content must still be replayed across a Tool turn.
- XiaoYu Tool Registry must expose `shell.exec` and workspace file mutation tools.
- `process.run` remains a manual compatibility alias rather than the model-facing Shell name.
- Workspace file mutation must reject symlink escape and workspace-root deletion.
- Old/large Observations must be compacted for model context without deleting durable Run receipts.
- Version must be consistent at `0.2.2` across Go, Vue, Electron, Wails, Rust workspace and installer metadata.


---

## AI-Game-Manager-Panel 0.2.1

### Release Notes

<!-- historical source: docs/releases/0.2.1.md -->

## Startup / Model Contract Hotfix

- 修复 `XiaoYuSaveModelRequest` 漏掉 `reasoningEffort`，导致模型管理页在 `vue-tsc --noEmit` 阶段失败、Web 一体服务无法继续启动的问题。
- XiaoYu Model Center Gate 现在会检查 SaveModelRequest 与模型表单的 `reasoningEffort` 契约，避免同类前后端类型漂移再次进入源码包。
- 产品版本统一更新为 `0.2.1`，并把 Rust workspace 纳入版本一致性 Gate。
- Updater Gate 不再把当前产品版本写死为某个版本号；版本边界测试继续保留 `0.1.100 -> 0.2.0`。
- 版本节奏继续采用每 100 个小版本升级一个中版本：`0.2.1 ... 0.2.100 -> 0.3.0`。
- 对外交付源码压缩包采用短命名：`agmp-<version>.zip`。

### Validation

<!-- historical source: AGMP-0.2.1-Validation.md -->

## 修复目标

Web 一体服务开发在 `pnpm run typecheck` 阶段失败：`ModelManagementSection.vue` 使用了 `form.reasoningEffort`，但 `XiaoYuSaveModelRequest` 没有声明该字段。

## 根因

Go Core 的 `SaveModelRequest`、`ModelProfile`、`ModelConnectionRequest` 已经支持 `reasoningEffort`；前端共享类型 `frontend/src/shared/types/backend.ts` 的 `XiaoYuSaveModelRequest` 漏同步，形成前后端契约漂移。

## 0.2.1 修复

- `XiaoYuSaveModelRequest` 增加 `reasoningEffort: string`。
- XiaoYu Model Center Gate 增加该字段的契约检查，防止再次漏同步。
- 版本统一为 `0.2.1`：Go、Vue、Electron、Wails、Rust Workspace、Windows Installer、release manifest。
- Updater Gate 去掉当前版本的硬编码断言，继续保留 `0.1.100 -> 0.2.0` 的百小版本边界测试。
- 版本规则：`0.2.1 ... 0.2.100 -> 0.3.0`。

## 本轮已执行

- 18 个非 Release-Key `scripts/common/check-*.mjs`：PASS。
- 44 个非 Wails `internal` Go package：`go test` PASS。
- 44 个非 Wails `internal` Go package：`go vet` PASS。

## 当前环境未执行

- `pnpm run typecheck`
- `pnpm run build`
- Wails 完整编译

原因：执行环境无法访问 npm / Go 外部依赖源，且没有可复用的 pnpm node_modules。Windows 开发机已能安装依赖，因此请重新运行“开发模式 -> Web 一体服务开发”；预期原先 3 个 `reasoningEffort` TypeScript 错误将被消除。


---

## AI-Game-Manager-Panel 0.2.0

### Release Notes

<!-- historical source: AGMP-0.2.0-Release-Notes.md -->

0.2.0 upgrades XiaoYu from a lowest-common-denominator model transport into a provider-aware Harness.

- OpenAI official preset uses Responses API and preserves encrypted reasoning replay.
- DeepSeek uses a dedicated native protocol path and preserves `reasoning_content` / thinking settings.
- Claude preserves thinking/signature blocks across Tool turns.
- Gemini preserves provider thought/function parts, including `thoughtSignature` fields returned by the provider.
- Compatible gateways retain a separate OpenAI-compatible path and multi-step Tool replay.
- Model Center exposes adapter/capability metadata and explicit reasoning effort.
- XiaoYu Workbench supports direct image attachments for Vision-capable adapters; raw bytes are excluded from normal Run JSON serialization.
- Model inference uses bounded retry for transient 408/429/5xx failures; Tool side effects remain behind Host/RBAC/Approval.
- Built-in one-click game-server deployment intelligence adds Preflight, data protection, dependency preparation, install/update, configuration, start, verification, and recovery workflow guidance.

### Release Notes

<!-- historical source: docs/releases/0.2.0.md -->

## XiaoYu Native Model Harness

- OpenAI 官方 Provider 改用 Responses API，保留 reasoning encrypted replay 与 Tool 连续回合。
- DeepSeek 官方 Provider 保留 thinking / reasoning_effort / reasoning_content。
- Claude 保留 thinking / signature / tool_use / tool_result。
- Gemini 保留 thinkingConfig / thoughtSignature / functionCall / functionResponse。
- OpenAI-compatible 网关继续作为兼容 fallback，并支持多步 Tool replay。
- 模型管理显示 Adapter、Reasoning、Tool、Replay、Vision 能力，并新增显式推理强度。
- XiaoYu 工作台支持 PNG/JPEG/WebP/GIF 图片输入；原始图片字节不会进入普通 Run JSON。
- 新增“游戏服务器一键部署” Skill：Preflight → 数据保护 → 依赖 → 安装/配置 → 启动 → 验证 → Recovery。
- 模型请求对 408/429/5xx 做有限退避重试；真实副作用仍只发生在通过 Host Tool/Approval 后。

### Validation

<!-- historical source: AGMP-0.2.0-Validation.md -->

## Candidate

- Baseline: `AI-Game-Manager-Panel-0.1.100-Codex-Turn-Approval-Fix-Source.zip`
- Candidate: `AI-Game-Manager-Panel-0.2.0-Native-Model-Harness-Source.zip`
- Scope: XiaoYu Native Model Harness / Provider Reasoning / Tool Replay / Vision / one-click server deployment intelligence.

## Passed

- All `scripts/common/check-*.mjs` gates passed except `check-release-key.mjs`, which is publisher-only and intentionally skipped in a source candidate environment.
- `go test ./internal/xiaoyu/... ./internal/app ./internal/bridge/httpapi ./internal/config ./internal/system/settings ./internal/deploy/updater` passed.
- `go vet` for the same packages passed.
- TypeScript parser check passed for `AIWorkbenchView.vue`, `ModelManagementSection.vue`, and `shared/types/backend.ts`.
- Native model regression tests cover OpenAI Responses encrypted reasoning replay, DeepSeek reasoning_content replay, native protocol selection, Vision serialization, and explicit rejection of unsupported image input.
- GitHub Safety Gate passed.

## Environment-blocked checks

- `go test ./internal/...` cannot complete in this sandbox because Go cannot download `github.com/wailsapp/wails/v2@v2.15.0` from `proxy.golang.org` (network/DNS blocked). This is not a source compile error in the tested packages.
- `corepack pnpm install` cannot download `pnpm@11.17.0` from npm registry because outbound DNS/network is blocked; therefore `pnpm build` / `vue-tsc` full project build cannot be honestly marked PASS here.
- `cargo` / `rustc` are not installed in this execution environment; Rust XiaoYu Core requires local Windows/CI compilation before release publishing.
- Release Key Gate is intentionally not run for a source candidate; formal publisher release still requires the private release key.

## Required local final verification

Run `AGMP.bat` menu 3 full project check, then Wails/Electron candidate build, and execute Rust/Frontend builds on the normal Windows development machine before menu 10 formal publication.


---

## AI-Game-Manager-Panel 0.1.100

### Release Notes

<!-- historical source: docs/releases/0.1.100.md -->

## Codex-style Turn Boundary + Approval Modes

- 修复“请求批准 / 帮我批准 / 完全访问权限”三种模式实际上规则重复的问题。
  - 请求批准：read 自动；operate / modify / destructive / system 均需人工批准。
  - 帮我批准：read / operate / modify 自动；destructive / system 必须人工批准。
  - 完全访问权限：审批层自动放行已注册且启用的 AGMP Tool，但 RBAC、参数校验、作用域、敏感操作 step-up、审计与验证继续生效。
- 审批指纹改为绑定 XiaoYu Run，禁止一个 Run 的待审批授权被另一个 Run 的相同 Tool 调用复用。
- 完成 / 失败 / 取消后的 Run 进入不可恢复终态；continue/resume 不得重新把 terminal Run 标记为 running。
- 聊天输入框聚焦仅是 UI 行为，不再保留“正在继续任务”等陈旧状态提示，也不会触发任何旧 Run。
- 当前审批卡只在 Run 真实处于 waiting_approval 且 approvalId 匹配时展示。
- 权限下拉增加点击组件外任意空白关闭和 Escape 关闭，避免浮层常驻。

## Codex Reference

本轮生命周期与审批边界参考 `openai/codex` 的 Thread / Turn / approval-policy 设计：终态 Turn 与后续新 Turn 明确分离；审批是执行前的独立策略边界，而不是聊天 UI 状态。


---

## AI-Game-Manager-Panel 0.1.99

### Release Notes

<!-- historical source: docs/releases/0.1.99.md -->

## Workbench Sidebar Snap Collapse

- 左侧一级导航最大拖拽宽度从 360px 提高到 520px。
- 左侧栏向左拖到最小边界后继续轻拉会进入吸附收起，不再卡死在最小宽度。
- 收起/展开改为稳定五轨 Grid 的 190ms 过渡，不再通过销毁 Grid track 产生跳变。
- 收起时 Sidebar 同步淡出/轻移；展开时反向恢复。
- 拖拽尚未松手时，继续向右越过回弹阈值可重新展开，避免误收起。
- 左右栏尺寸仍保持独立；中央区域只使用剩余空间。
- 布局状态升级到 `agmp.workbench.layout.v6`。

## Compatibility

- 保持 5% / 75% / 20% 初始目标比例。
- 保持 220px 左栏正常最小宽度与 260px 右栏正常最小宽度。
- 顶栏“显示左侧栏”按钮仍可恢复已收起侧栏。


---

## AI-Game-Manager-Panel 0.1.98

### Release Notes

<!-- historical source: docs/releases/0.1.98.md -->

本版不再继续用“补一个 Tool / 加一句 Prompt”的方式修 Agent，而是按成熟 Agent Harness 的控制主干收口：会话连续性、可插话 steering、审批硬门禁、执行回执和可逆上下文必须由 Host 保证，不能靠模型自觉。

- **审批硬门禁修复**：`帮我批准` 不再把普通副作用操作自动当成已经批准；只有只读自动放行，任何会改变系统状态的 Tool 都必须经过真实用户确认。`settings.theme.set` 从 `operate` 修正为 `modify`，避免设置变更绕过审批。
- **运行中可插话 / Steering**：正在执行、等待批准、等待补充的 Run 都可以接受用户新指令。Host 会取消当前推理 turn，在安全边界收口，作废旧审批，再把“用户实时纠正”写回 Run Observation 后重新规划同一个 Run。
- **跨 Run Thread Context**：同一工作台 Session 的最近 Run 由服务端生成有限历史，不再只依赖浏览器本地聊天文本。新 Run 能看到最近目标、结果和结构化 Tool 回执，用于理解“改回去 / 撤销刚才 / 继续刚才”。
- **可逆设置回执**：`settings.theme.set` 返回 `previousTheme`；Thread Context 会生成显式 Undo Hint，让“把刚才主题改回去”可以直接调用同一设置 Tool 恢复上一个值，而不是再次问用户“改回哪个”。
- **Prompt Core 强化**：Rust Brain 明确要求实时纠正优先于旧计划；审批 pending 只能由 Host 的真实 approvalId 解除；`frame.thread.recentTurns` 是跨 Run 连续性的服务端依据。
- **对话工作台允许干预**：输入框在 Run 执行期间保持可用。用户发送新消息会进入当前 Run，而不是必须等任务结束或先取消任务。

这版对应 Agent Runtime 的第一批硬能力：**Thread / Steering / Approval / Observation**。后续继续把 Session Event Log、Plan/Review、Recovery、Skills/Experts 和 Tool Pipeline 做成更完整的可插拔主干。


---

## AI-Game-Manager-Panel 0.1.97

### Release Notes

<!-- historical source: docs/releases/0.1.97.md -->

- XiaoYu Intelligence Context 新增 AGMP **全量紧凑 Module Catalog + 目标相关详细 Module Map**：先知道整个系统有哪些功能域，再按当前目标加载细节，并明确 implemented/partial/skeleton 状态，避免把路线图能力误当真实 Tool。
- 新增 `agmp.capability.search`：能力未知、下一步不明确或直接 Tool 不明显时，小鱼先检索真实模块与当前可执行 Tool，不再直接停机。
- Memory / Skills / Experts 改为按目标相关性选择；始终注入 AGMP 自主执行基础 Skill/Expert，环境、设置、DST 等领域再按目标动态加载。
- 新增 AGMP 自主执行、设置/UI、运行环境 Doctor 内置 Skills，以及 AGMP 产品专家、运行环境专家。
- Agent Loop 新增完成证据 Gate：明确操作型目标没有成功 Tool Observation 时不能直接 complete；状态变更后如果同类别存在读取 Tool，必须验证真实状态再完成。
- Rust Brain System Core 禁止用 `wait` 询问“是否继续 / 要不要修复”等程序性问题；真正缺少 Token、密码、业务选择时才等待用户。
- 审批卡从右侧上下文移动到聊天输入框上方，只在当前任务存在待审批时出现；批准/拒绝后立即消失，下一条审批出现时再显示。
- 右侧栏只保留运行状态、模型、Tool、设置入口与 Trace，不再承载审批操作。


---

## AI-Game-Manager-Panel 0.1.96

### Release Notes

<!-- historical source: docs/releases/0.1.96.md -->

## 定位

0.1.96 是 XiaoYu Intelligence 2.0 的第一阶段。重点不是继续堆 UI，而是把“小鱼会聊天 + 会调用 Tool”升级为可观察、Goal-first、可恢复的 Agent Kernel。

## 完成

- Rust `xiaoyu-core` 固化 System Core v2：目标优先、直接能力优先、组合 Tool、执行后验证、失败后恢复、禁止假完成。
- 明确禁止把隐藏思维链直接展示给用户；模型只输出简短、可审计的行动摘要。
- Provider 在 Tool Call 同时返回普通文本时会保留该行动摘要；OpenAI-Compatible / Anthropic / Gemini 三条链路统一。
- Run 新增 Agent Phase：`understanding / planning / executing / verifying / recovering / waiting_* / completed / failed / cancelled`。
- Run 新增 `decisionSummary`，前端在对话线程内显示“小鱼正在做什么”，不展示内部 chain-of-thought。
- 后台 Run 增加实时 State Sink，步骤、Tool 次数、阶段与恢复状态在任务执行期间持续更新，不再等任务结束才刷新。
- Frame 新增 Capability Context（Tool 数、类别、风险分布）以及当前 `uiRoute`，帮助 Rust Brain 正确理解“当前页面 / 这里 / 这个”等上下文。
- 新增 `settings.get` 与 `settings.theme.set`。用户说“把主题改成浅色”时，小鱼可以直接调用领域能力，不必只导航到设置页。
- `settings.theme.set` 执行后向当前 XiaoYu 客户端发送 `settings/changed` 事件，主题立即生效。

## 安全边界

- Rust Core 仍然只负责 Brain/Policy，不获得 OS、文件、进程执行权。
- Go Host 仍然是 Tool、RBAC、License、审批、真实业务执行的唯一边界。
- `uiRoute` 仅接受 AGMP 站内路径，拒绝外部 URL。
- `settings.theme.set` 仍受现有 Tool 风险与审批策略约束，本版本没有绕过 Approval Policy。

## 下一阶段

0.1.97 进入 Plan Mode + Smart Approval：计划级审批、风险解释、推荐批准/拒绝、低风险自动执行与高风险强制停下。


---

## AI-Game-Manager-Panel 0.1.95

### Release Notes

<!-- historical source: docs/releases/0.1.95.md -->

## 修复

- 三栏保留 5 / 75 / 20 作为首次目标比例，但左侧增加 190px（大屏）可读最小宽度，禁止菜单文字被压成竖排。
- 左右 Splitter 完全独立：拖右栏不再改变左栏；拖左栏不再改变右栏。
- 拖拽期间不再每个 pointermove 写 localStorage，降低卡顿；pointer capture + pointercancel 保证拖拽不易“卡住”。
- Splitter 命中区从 5px 扩大到 9px，视觉仍保持 1px 分隔线。
- 小鱼主工作区改成真正聊天界面：消息在同一对话流内展示，输入框固定在底部。
- Enter 发送，Shift+Enter 换行；输入框按内容自动增长到 160px。
- XiaoYu Run 状态、Tool 次数和 Observation 收到助手消息下方，不再在页面上方单独堆大卡片。
- 布局持久化键升级为 `agmp.workbench.layout.v5`。

## XiaoYu UI Control

- 新增 Host 白名单 Tool `ui.navigate`。小鱼可以按用户自然语言要求打开 AGMP 内部页面，例如设置中心、模型管理、运行环境与存储、日志、部署等。
- UI Tool 不接受 URL，只接受固定 target ID；Host 通过 `ui/navigate` Trace Event 通知发起该 Run 的 `sessionId`，其他浏览器标签/其他用户不会跟随跳转。
- 前端不让 AI 直接操作 DOM/模拟鼠标；真正导航仍由 Vue Router 执行，因此权限路由与 License Gate 继续生效。


---

## AI-Game-Manager-Panel 0.1.94

### Release Notes

<!-- historical source: docs/releases/0.1.94.md -->

## 目标

继续按实际视觉效果调整三栏工作台初始宽度，不改变三栏职责。

## 变更

- 大屏目标初始比例改为：左侧一级导航约 **5%**、中央主工作区约 **75%**、右侧上下文/二级菜单约 **20%**。
- 左栏桌面最小宽度下调，确保 1920/2048 等常见大屏能够真实接近 5%，而不是被旧的 160px 下限顶回更宽。
- 中央保底比例提升到 75%，页面正文、对话和配置表单获得更多横向空间。
- 布局持久化键升级为 `agmp.workbench.layout.v4`，首次打开 0.1.94 时不会继承 0.1.93 的 10/60/30 已保存宽度。
- 拖拽、收起、恢复以及后续用户自定义宽度保持不变。

### Validation

<!-- historical source: docs/development/VALIDATION-0.1.94-WORKBENCH-RATIO.md -->

## 验证目标

将首次三栏布局从 10% / 60% / 30% 改为用户指定的 5% / 75% / 20%，同时保留拖拽、收起、持久化和响应式保护。

## 静态检查

- `agmp.workbench.layout.v4`：应存在。
- 初始比例常量：0.05 / 0.75 / 0.20。
- Project Layout Gate：应强制验证新比例和新缓存键。
- 版本一致性：0.1.94。

## 目标视觉

- 1920px 宽：左栏约 96px、中央约 1434px、右栏约 382px（另有拖拽分隔线）。
- 2048px 宽：左栏约 102px、中央约 1530px、右栏约 408px。
- 2560px 宽：左栏约 128px、中央约 1914px、右栏约 510px。

窄窗口仍允许响应式最小宽度保护，因此不强制死守百分比。


---

## AI-Game-Manager-Panel 0.1.93

### Release Notes

<!-- historical source: docs/releases/0.1.93.md -->

本版只调整三栏工作台的**首次/重置后的初始宽度**，不改变三栏职责，不改 0.1.92 的首屏与刷新修复。

## 调整

- 大屏目标初始比例：左侧一级导航约 **10%**、中央主工作区约 **60%**、右侧上下文/二级菜单约 **30%**。
- 左栏继续保持最精简，只承载一级导航。
- 右栏获得更多默认空间，用于设置中心二级菜单、小鱼运行上下文、审批、Tool/Harness 等。
- 左右栏仍可拖拽、收起并持久化。
- 保留响应式最小宽度与超宽屏上限，极端分辨率下允许小幅偏离 10/60/30。
- 布局持久化键升级到 `agmp.workbench.layout.v3`，确保升级后能看到新的默认宽度。

## 不变

- 中央区仍禁止再嵌套第二套左右菜单。
- Bottom Panel 仍独立承载终端/日志/任务输出。
- 0.1.92 的无感刷新、主题首帧和 XiaoYu 首屏修复保持不变。

### Validation

<!-- historical source: docs/development/VALIDATION-0.1.93-WORKBENCH-RATIO.md -->

## 目标

把首次三栏布局从上一版窄右栏方案调整为用户指定的约 10% / 60% / 30%，同时保持拖拽、收起、持久化和中央工作区保底。

## 自动检查

- `agmp.workbench.layout.v3`：PASS。
- 初始比例常量 `0.10 / 0.60 / 0.30`：由 Project Layout Gate 固定。
- 版本一致性：0.1.93。
- 最终源码包再次解压后运行 Project Layout / Windows Helper Gate。

## 实机视觉验收建议

- 1600×900：左栏约 160px，右栏接近 30%，中央约 60%。
- 1920×1080：目标约 192 / 1152 / 568px（另有分隔条占位）。
- 2560×1440：左栏约 256px；右栏接近 30%，受上限保护；中央不少于约 60%。
- 检查左右拖拽、左右收起/恢复、刷新后保持用户手动宽度。
- 980px 以下继续优先收起右栏。


---

## AI-Game-Manager-Panel 0.1.92

### Release Notes

<!-- historical source: docs/releases/0.1.92.md -->

本版只处理 0.1.91 实机发现的两个前端回归，不扩展功能范围。

## 修复

- 修复小鱼首页“未配置大模型大脑”提示被历史 `ai-workspace` 三行 Grid 的 `1fr` 拉伸成超大空白矩形的问题。
- 小鱼工作台改为纵向 Flex：标题、配置提示、主对话工作区均按正确职责布局，配置提示只占内容高度。
- Web/Electron 刷新优先使用现有 HttpOnly Cookie 恢复 `currentUser`，不再先请求 Bootstrap 并展示认证 Loading UI。
- 会话恢复成功后先恢复三栏 Shell，再异步加载 Info / Config / Settings / License，降低整页切换感。
- 主题选择写入 `agmp.ui.theme-choice`，`index.html` 在 Vue/CSS 启动前预先恢复主题，避免浏览器默认背景造成白闪/亮暗闪。
- 首帧使用 `agmp-preload` 暂停动画和 transition，第二个 animation frame 后恢复。

## 边界

- 不改变三栏职责。
- 不改变认证安全模型：Cookie 仍为 HttpOnly，会话失效后仍进入登录页。
- 不跳过权限、License 或后端 Session Gate。

### Validation

<!-- historical source: docs/development/VALIDATION-0.1.92-FIRST-SCREEN.md -->

## 回归目标

1. XiaoYu 未配置 Brain 时，黄色/中性配置提示只按内容高度显示，不得撑满中央工作区。
2. 已登录 Web 页面按 F5/Ctrl+R 后，不得短暂显示登录卡或 Bootstrap 页面。
3. Dark/Light 模式刷新时，首帧背景必须与刷新前主题一致，不得出现白闪/黑闪。
4. Session 过期时必须仍然进入登录页，不能因为“无感刷新”绕过认证。

## 自动检查

- XiaoYu Harness Gate 新增工作台布局防回归标记检查。
- Web Session Security Gate 新增 Silent Session Restore、Shell-first hydration、Prepaint Theme Restore 检查。


---

## AI-Game-Manager-Panel 0.1.91

### Release Notes

<!-- historical source: docs/releases/0.1.91.md -->

## 修复原因

0.1.90 源码交付包错误遗漏了三个仍被现役代码与 Gate 依赖的核心目录：

- `internal/platform/runtime/`
- `internal/xiaoyu/runtime/`
- `internal/games/dst/runtime/`

同时遗漏 `runtime/README.md`。这会导致 Go 把本应存在于当前 module 内的 import 当成外部依赖查找，进而出现 `no matching versions for query "latest"`。

## 修复内容

- 从最后一个确认完整且通过 Shared Runtime 验证的 0.1.88 源码恢复上述 Runtime 源文件。
- 保留 0.1.90 的 UI Recovery、Web Session、模型 Tool Name 兼容与 Environment Manager 改动。
- 修复 `scripts/tools/license/acceptance-test.ps1` 的 UTF-8 BOM + CRLF 格式回归。
- 不新增第二套 Process Runtime，不改变既有架构边界。

## 验证

- 全部 `scripts/common/check-*.mjs` Gate 通过；Release Key Gate 以开发模式 `--allow-unconfigured` 执行。
- `go test` 通过：`internal/platform/runtime`、`internal/games/dst/runtime`、`internal/xiaoyu/runtime`、`internal/app`、`internal/bridge/httpapi`、`internal/deploy/environment`、`internal/xiaoyu/host`。

### Validation

<!-- historical source: docs/development/VALIDATION-0.1.91-RUNTIME-RESTORE.md -->

0.1.90 交付包不是网络代理故障，而是源码完整性故障：源码包缺少本 module 内三个 Runtime package，`go mod tidy` 才会错误尝试从 GitHub/Go Proxy 查找它们。

本版本恢复：

- `internal/platform/runtime`：统一 Process / stdio / Session / Run / Shell 边界；
- `internal/xiaoyu/runtime`：Go Host 到 XiaoYu Rust Runtime 的桥接；
- `internal/games/dst/runtime`：DST 运行态与进程管理；
- `runtime/README.md`：运行数据目录契约。

已执行：

```text
node scripts/common/check-*.mjs

go test ./internal/platform/runtime \
  ./internal/games/dst/runtime \
  ./internal/xiaoyu/runtime \
  ./internal/app \
  ./internal/bridge/httpapi \
  ./internal/deploy/environment \
  ./internal/xiaoyu/host
```

结果：PASS。


---

## AI-Game-Manager-Panel 0.1.90

### Release Notes

<!-- historical source: docs/releases/0.1.90.md -->

## 本版目标

0.1.90 是针对 0.1.89 实机界面回归的修复版：恢复三栏信息架构、收窄大屏侧栏、把页面二级菜单归位到右侧上下文栏，同时修复刷新掉登录、Intelligence / Runtime 空白以及“模型已配置但小鱼 Tool 调用失败”的链路问题。

## 修复

- Web / Electron HTTP 会话刷新不再因为前端读不到 HttpOnly Cookie token 而强制回登录页；刷新时会先使用已有安全 Cookie 调用 `currentUser` 恢复会话。Wails 仍使用自身 Bearer Session。
- 模型 Provider 边界新增 Tool 名别名映射：`system.info` 等 AGMP 命名空间 Tool 会转换为只含字母、数字、`_`、`-` 的供应商安全名称，模型返回后再恢复真实 Host Tool 名称。解决 DeepSeek / OpenAI-Compatible `tools[].function.name` HTTP 400。
- 模型“测试连接”不再只检查 `/models`；填写模型代码后会追加一次带安全 Tool Schema 的最小实际调用，避免“连接测试成功、真正 XiaoYu Agent 却失败”。
- `记忆、技能与专家` 的 Go Catalog 改为始终输出空数组而不是 JSON `null`，前端同时做防御性归一化，避免异步刷新后 `.length` 触发渲染崩溃。
- `运行环境与存储` 的 Runtime Catalog 空 Registry 改为输出 `[]`，前端对 `runtimes/javaMajors/warnings` 同时做空集合归一化，修复空 Runtime Registry 时页面渲染异常。

## UI / 信息架构

- 三栏保留，但大屏不再按 20% / 60% / 20% 无限放大。左栏默认约 220–260px、最大 280px；右栏默认约 250–300px、最大 320px。中央工作区优先吃满剩余空间。
- 布局键升级为 `agmp.workbench.layout.v2`，自动摆脱旧版本保存的超宽侧栏值。
- 设置中心的分类导航从中央左侧移出，统一进入工作台右侧上下文栏。
- 小鱼中央区移除内部第二套右栏，恢复单一、居中的对话 / Autonomous Run / Composer 主画布。
- XiaoYu Runtime、Brain、待审批、Tool、Harness Plugin 与 Trace 统一进入右侧上下文栏。
- 小鱼的“完整终端”改为打开 Bottom Panel，不再在中央/内部侧栏复制受控终端。

## 测试与验证

- 新增 OpenAI-Compatible Tool Alias 单元测试，验证 Provider 收到的工具名不含点号并能映射回 `system.info`。
- 本环境已执行源码静态检查与针对性代码审查。
- Go 测试受当前执行环境无法访问 `proxy.golang.org` 影响，依赖 `wails/v2` 无法下载，因此不能在此环境冒充 `go test ./...` 已通过。
- Frontend 正式 `pnpm build` 受当前执行环境没有预装 pnpm 且无法访问 npm registry 影响，需在你的 Windows 开发机执行完整 Gate。

## 本地验收重点

1. 登录一次后按 F5 / Ctrl+R，多次刷新仍保持登录。
2. 1920px 及更宽窗口：左右栏明显收窄，中央区为主；拖拽与收起仍正常。
3. 设置中心：右栏切换“模型管理 / 记忆技能专家 / 运行环境与存储”，中央不再出现第二列菜单。
4. 空 Intelligence / 空 Runtime Registry 也必须显示完整页面，不允许白屏。
5. DeepSeek 保存为默认模型后执行“小鱼，检查本机开服环境”，不得再出现 `tools[0].function.name` pattern 400。
6. 模型连接测试必须同时反馈模型目录与 XiaoYu Tool Schema 实际调用结果。

### Validation

<!-- historical source: docs/development/VALIDATION-0.1.90-UI-RECOVERY.md -->

## 修复基线

- 输入基线：`AI-Game-Manager-Panel-0.1.89-Developer-Helper-Fixed-Source.zip`
- 输出候选：`AI-Game-Manager-Panel-0.1.90-source.zip`
- 本轮范围：三栏信息架构、设置二级导航、小鱼中央工作区、刷新登录恢复、Intelligence / Runtime 空白、模型实际 Tool 调用兼容。

## 已确认根因

### 1. 刷新后强制重新登录

HTTP/Web/Electron 使用 HttpOnly `agmp_session` Cookie；浏览器 JavaScript 本来就不能读取该 Token。旧 `AuthGate` 却先以 `getSessionToken()` 是否为空决定是否展示登录页，因此刷新时会在调用 `currentUser()` 前错误跳回登录。

0.1.90 规则：只有 Wails Bridge 依赖显式 Bearer Token；HTTP/Electron 刷新时直接使用 Cookie 调用 `currentUser()` 恢复会话，真正收到 401 才回登录页。

### 2. “记忆、技能与专家”空白

空 Memory Catalog 在 Go 中可能把 `nil` slice 序列化成 `null`；前端异步加载后直接调用 `catalog.memories.length`，会触发渲染异常。

0.1.90 双层修复：Go API 永远返回数组；Vue 再把 `memories/skills/experts` 防御性归一化为 `[]`。

### 3. “运行环境与存储”空白

空 Runtime Registry 的 `runtimes` 曾可能返回 `null`，前端 `catalog?.runtimes.find(...)` 只保护了 `catalog` 本身，没有保护 `runtimes`，异步响应到达后可触发 `.find` 异常。

0.1.90 双层修复：Go Runtime Catalog 空集合返回 `[]`；Vue 对 `runtimes/javaMajors/warnings` 归一化，并使用 `runtimes?.find`。

### 4. 模型已配置但 XiaoYu 实际无法调用

实机 Observation 已显示 Provider 返回：`HTTP 400: Invalid 'tools[0].function.name'`。AGMP Host Tool 使用 `system.info` 等点号命名空间，而 DeepSeek / OpenAI-Compatible function name 只允许字母、数字、下划线和短横线。

0.1.90 在 **Provider 边界**建立稳定别名：真实 Host Tool 名不变，只在发给模型时转换为合法名称；模型返回 Tool Call 后再映射回 Host Registry 原名。Anthropic / Gemini 走同一别名规则。

连接测试也不再把 `/models` 成功当成“AI 可用”；当填写模型代码后，会继续发送一条带 XiaoYu Tool Schema 的最小生成请求。

## UI 冻结结果

- 左侧只承载一级导航；大屏默认约 `220–260px`，最大 `280px`。
- 中央只承载当前页面主内容；小鱼恢复为单一居中的对话 / Run / Composer 工作区。
- 右侧承载上下文与二级导航；大屏默认约 `250–300px`，最大 `320px`。
- 设置中心的 `模型管理 / 记忆、技能与专家 / 运行环境与存储 / ...` 已从中央左列迁移到真正的右侧栏。
- XiaoYu Runtime / Brain / 审批 / Tool / Harness / Trace 已迁移到真正的右侧上下文栏。
- 终端只进入 Bottom Panel，不再在中央或 XiaoYu 内部复制第二套侧栏/终端。
- 布局持久化键升级为 `agmp.workbench.layout.v2`，旧 0.1.89 超宽侧栏状态不会继续污染 0.1.90。

## 已执行静态验证

通过：

- 修改过的 9 个 TypeScript/Vue Script 块使用 TypeScript 5.8.3 `transpileModule` 语法检查。
- 7 个修改过的 Vue 外层 Template 进行 HTML parser 结构检查。
- `base.css`、`AIWorkbenchView.vue`、`XiaoYuRightContext.vue` 花括号数量一致。
- 修改过的 Go 文件全部 `gofmt`，`gofmt -d` 无差异。
- `frontend/package.json`、`desktop/electron/package.json`、`configs/release.json` JSON 解析通过。
- `check-updater.mjs`、`check-windows-helper.mjs` Node 语法检查通过。
- 新增模型 Tool Alias 单元测试；新增 `/models` 成功但实际 Tool Probe 失败不得误报成功的测试；补充空 Catalog 必须输出数组的断言。

## 当前环境未能完成的 Gate

### Go test

`GOPROXY=off go test ./internal/xiaoyu/host` 在依赖解析阶段停止：当前容器没有缓存 `github.com/wailsapp/wails/v2@v2.15.0`，且联网依赖下载不可用。因此本记录 **不声明 Go Test 已通过**。

### Frontend build

当前容器没有项目 `node_modules`；Corepack 获取 pnpm 时因 `registry.npmjs.org` DNS/网络不可用失败。因此本记录 **不声明 `pnpm build` 已通过**。

### Project Layout Gate

基线源码自身的 `scripts/common/check-project-layout.mjs` 仍引用当前压缩包中不存在的 `internal/xiaoyu/runtime/*`、`internal/platform/runtime/*`、`internal/games/dst/runtime/*` 等路径，因此 Gate 在本轮修改前的源码结构上即不满足。本轮没有用伪文件掩盖该问题；它应作为独立的基线/Checker 一致性任务处理。

## Windows 实机验收顺序

1. `pnpm install`
2. `pnpm --dir frontend run build`
3. `go test ./...`
4. `go vet ./...`
5. 通过开发入口启动 Web/Wails。
6. 登录一次后连续 F5 / Ctrl+R 3 次，确认会话保持。
7. 1920px、2560px 大屏分别检查三栏：中央明显为主，左右栏不再按 20% 放大；拖拽、收起、恢复正常。
8. 进入设置中心，确认设置分类只出现在外层右侧栏；中央不再有第二列导航。
9. 分别打开“记忆、技能与专家”“运行环境与存储”，在空数据情况下仍能完整渲染并可操作。
10. 保存 DeepSeek 为默认模型，先点“测试连接”，再回小鱼执行“检查本机开服环境”。不得再出现 `tools[0].function.name` Pattern HTTP 400。
11. 小鱼页中央只保留对话/任务主体；Runtime、审批、Tool、Harness、Trace 在外层右栏；完整终端从 Bottom Panel 打开。


---

## AI-Game-Manager-Panel 0.1.89

### Release Notes

<!-- historical source: docs/releases/0.1.89.md -->

## 定位

0.1.89 在 0.1.88 已冻结的 Human + XiaoYu 双大脑、Rust Brain-only、Go Host/Domain/Shared Runtime 和多端 AI Parity 基线上，补齐两类产品基础设施：**组织安全与 XiaoYu Intelligence**，以及 **Environment Manager v2**。本版不允许把 AI、运行环境或组织数据做成第二套桌面专属实现。

## 本版核心

- 组织邀请安全：首次 Owner 后永久关闭公开注册；新成员只能通过当前 AGMP 实例 Ed25519 签名的一次性邀请加入组织。
- 成员核心授权：加入组织不等于获得核心能力；正式产品 License 与 Owner 签发的成员 Core Authorization 是独立 Gate。
- Web 会话安全：HttpOnly Session Cookie、SameSite、CSRF、Origin/Sec-Fetch-Site、CSP 等公网 Web 防护；Wails 保持显式本地会话桥接。
- 邮箱为可选安全增强：用于找回密码、安全通知和风险二验；SMTP 由系统设置中的管理员配置。未配置 SMTP 不阻断 AGMP、邀请、成员授权或 XiaoYu 的正常基础授权链。
- XiaoYu Intelligence：Memory / Skills / Experts 只存在当前 AGMP 实例内，支持 Private / Group / Organization 可见性；敏感 Memory 不自动发送模型；所有上下文先经过 Host ACL。
- Server-owned Task Scope：TaskID 由 RunManager 服务端生成，禁止客户端伪造 Task Memory 范围。
- Environment Manager v2：统一管理 Java 与 SteamCMD Runtime；Java 8/17/21/25 可并存，可登记外部 Runtime，也可由 AGMP 安装受管 Runtime。
- 跨平台 SteamCMD：支持 Windows 官方 ZIP 与 Linux 官方 tar.gz；Linux managed 入口识别 `steamcmd.sh`。
- Game Runtime Profile：Minecraft 仅声明所需 Java；DST 才声明 SteamCMD，并在 Linux amd64 上诊断 SteamCMD 的 32 位系统运行库。
- Linux 系统依赖：仅固定 Profile 白名单可安装，经过 Shared Runtime，不允许 XiaoYu 任意 Shell；XiaoYu Tool 标记为 RiskSystem，继续经过统一高风险审批/二验链。
- Runtime 安全：Java 下载使用 Adoptium 发布 SHA256 校验；ZIP/TAR 安全解压拒绝 traversal、symlink 和特殊文件；外部 Runtime 永不被 AGMP 删除文件。
- 唯一 Process Core：Environment Manager 不直接使用 `os/exec`，Java 验证与 Linux 包管理器执行统一经过 `internal/platform/runtime`。

## 发行原则

0.1.89 的用户成品仍然是预编译 AGMP Core + XiaoYu Runtime + Web/Desktop 入口。Java、SteamCMD 等属于**游戏服务器运行环境**，由 AGMP 启动后的 Runtime Manager 管理；Go、Node、pnpm、Rust、Cargo、MSVC、Wails 等仍然只属于开发/发行环境，不得成为最终用户运行依赖。

## 当前验证边界

Environment Manager v2 的 Go 环境包、Application、HTTP、Auth、XiaoYu Host 测试和相关 `go vet` 已在当前开发环境通过；Environment 包 Race Detector 已通过。前端真实 pnpm/Vite、Rust cargo 和 Wails 完整依赖构建仍需要在具备对应工具链/网络缓存的正式开发或 CI 环境完成，不能由静态 Gate 替代。

## Windows Developer / Publisher Helper 修复

- 重做菜单语义：4/5/6 为 Candidate Build，只有菜单 10 为严格 Official Publish。
- Candidate Build 不再因本机没有 ReleaseKeys/key-metadata.json 被 Release Key Gate 阻断；正式发布仍 fail-closed。
- 修复 Wails Bridge 的 `dstruntime` 未定义别名和 DST `token` 包名遮蔽，Wails dev 启动前增加 Bridge Go 编译预检。
- 修正发行公钥同步目标到 `internal/system/license/vendor_public_keys.json`。
- Rust/MSVC/Wails/Inno 安装与日常构建分离：构建菜单只检查，显式修复集中到菜单 1/8。
- Rust bootstrap 使用官方 rustup-init + SHA256，不使用 winget 安装 Rust。
- `-SelfTest` Dry-Run 现在覆盖菜单 1–10 且不下载、不构建、不创建构建目录、不依赖真实 XiaoYu Runtime 或 Release Key。
- 根 `AI-Game-Manager-Panel.bat` 支持转发 `-SelfTest` / `-Task` 快捷参数。

### Validation

<!-- historical source: docs/development/VALIDATION-0.1.89-DEVELOPER-HELPER.md -->

## 目标

本阶段针对 Windows 源码开发助手 `AI-Game-Manager-Panel.bat` 及其 PowerShell Task Runner 做一次完整的菜单边界和可用性收口。重点解决真实 Windows 环境暴露的两个问题：

1. Wails 开发在 bindings 阶段因 DST Runtime/token import alias 错误无法编译。
2. Wails/Electron/Linux 候选构建错误地被正式 Release Key Gate 阻塞。

本 Helper 只服务源码开发/发布人员；普通用户仍只安装正式 AGMP 成品，不需要 Go/Node/pnpm/Rust/MSVC/Wails。

## 已修复

### Wails 开发编译

- `internal/bridge/wails/app.go` 的 DST Runtime 类型统一使用 `dstruntimecore`。
- DST token 包统一别名为 `dsttoken`，避免方法参数 `token string` 遮蔽包名。
- Wails 开发启动前新增 `go test -tags agmp_dev_license ./internal/bridge/wails` 预检，让 Go Bridge 编译错误在进入 Wails bindings 前直接显示。

### Candidate / Official Publish 分离

- 菜单 4：Wails Windows **候选构建**，不要求 active Release Key，写入 `build/candidate/windows/wails`。
- 菜单 5：Electron Windows **候选构建**，不要求 active Release Key，写入 `build/candidate/windows/electron`。
- 菜单 6：Linux Server **候选构建**，不要求 active Release Key，写入 `build/candidate/linux-server`。
- 菜单 10：唯一的**正式发布**入口，严格要求 active Release Key，写入 `build/release`。
- `Invoke-CoreCandidateGate` 与 `Invoke-CoreReleaseGate` 已分离。
- 发行公钥同步目标修正为 `internal/system/license/vendor_public_keys.json`。
- 没有仓库外 `ReleaseKeys/key-metadata.json` 时，候选构建不再尝试同步或失败；只阻止菜单 10。

### 开发工具链边界

- 菜单 1 负责基础环境：Go / Node / pnpm / Frontend / Wails CLI，不安装 Rust/MSVC。
- Wails/Electron 开发和候选构建只 `Assert` Rust/MSVC/Wails/Inno，不在构建中偷偷安装。
- Rust 只允许在菜单 8 显式准备，使用 Rust 官方 `rustup-init` + SHA256 校验，不调用 winget 安装 Rust，也不启动第二个 rustup 控制台。
- 已有 Visual Studio Build Tools / MSVC / Windows SDK 优先复用。
- Inno Setup 只允许菜单 8 显式准备；候选构建只检查。

### 菜单与便携性

主菜单现在明确分为：

1. 初始化 / 修复基础开发环境
2. 开发模式
3. 项目检查
4. Wails Windows 候选构建
5. Electron Windows 候选构建
6. Linux Server 候选构建
7. 预览 / 启动构建产物
8. 状态、诊断与修复
9. 清理与重置
10. 正式发布

根 BAT 会原样转发参数，因此支持 `AI-Game-Manager-Panel.bat -SelfTest` 和 `AI-Game-Manager-Panel.bat -Task <0-10>`。

菜单 3 的“开发助手菜单自检”使用 Dry-Run。Dry-Run 现在：

- 不下载依赖；
- 不真实构建；
- 不创建 build/cache 目录；
- 不要求已有 `xiaoyu.exe`；
- 不要求 Rust/MSVC/Inno/Release Key；
- 覆盖菜单 1–10，并包含 Wails、Electron、Linux OfficialPublish 路由。

## 自动 Gate

`check-windows-helper.mjs` 现在额外防止：

- Candidate 构建重新绑到严格 Release Key；
- 菜单 10 漏掉 Linux OfficialPublish；
- Wails Bridge 回归到未定义 `dstruntime`；
- `token` 参数再次遮蔽 DST token 包；
- Wails dev 删除 Bridge Go 预编译检查；
- `Build-AGMPXiaoYuCore` 在 Dry-Run 下依赖真实 exe；
- 构建过程恢复自动安装 Rust/MSVC；
- Rust 恢复使用 winget 而不是官方 rustup-init；
- PS1 丢失 UTF-8 BOM / CRLF；
- 根 BAT 不再转发快捷参数。

## 本阶段实际验证

### Common Gates

所有 `scripts/common/check-*.mjs` 已逐项执行；Release Key Gate 使用开发/候选模式 `--allow-unconfigured`。19 个 Common Gate 全部通过。

### Go

真实核心包：

```text
go test ./internal/app ./internal/system/auth ./internal/bridge/httpapi ./internal/deploy/environment ./internal/xiaoyu/host
```

通过。

当前容器无法联网拉取 Wails v2.15.0，因此使用**测试专用、本地最小 Wails runtime/options stub**只做 Go 类型编译，未进入源码包。先验证 `internal/...`，随后临时创建只供 `go:embed` 编译使用的 `frontend/dist/index.html` 占位文件（测试结束立即删除），再验证整个 module：

```text
GOWORK=off go test -modfile=/mnt/data/agmp-test.mod ./internal/...
GOWORK=off go vet  -modfile=/mnt/data/agmp-test.mod ./internal/...
GOWORK=off go test -modfile=/mnt/data/agmp-test.mod ./...
GOWORK=off go vet  -modfile=/mnt/data/agmp-test.mod ./...
```

全部通过，其中包括根 Wails main、`internal/bridge/wails`、全部 cmd/internal package 的 Go 类型编译。测试 stub、临时 modfile 和 embed 占位文件均不进入源码包。

### PowerShell / Windows 边界

当前验证容器没有 Windows PowerShell / pwsh，因此不能声称菜单 1–10 已在本机逐项真实执行。已完成：

- 10 个 Windows PS1 的 UTF-8 BOM + CRLF 检查；
- Helper 函数调用图检查，无未定义的 AGMP Helper 调用；
- Wails DST alias/shadowing 回归扫描；
- Windows Helper Common Gate；
- Dry-Run 路由结构 Gate。

用户提供的真实 Windows 日志已经证明 Rust/MSVC/Windows SDK/Wails CLI 能被识别，并且 XiaoYu Rust Runtime 能实际编译；原 Wails 失败点位于本次已修复的 Go Bridge alias。

## 仍需真实 Windows 复验

更新源码后，应优先运行：

```text
AI-Game-Manager-Panel.bat -SelfTest
```

随后验证：

```text
菜单 2 -> 1  Wails Desktop Development
菜单 4       Wails Windows Candidate Build
```

预期：

- 菜单 2 -> 1 不再出现 `undefined: dstruntime` / `token.Status is not a type`。
- 菜单 4 在没有 ReleaseKeys/key-metadata.json 时继续构建 Candidate，而不是进入正式发行密钥同步。
- 只有菜单 10 正式发布才会因缺少 active Release Key 而拒绝继续。

## 阶段判断

从源码结构、Go 类型编译、全部 Common Gate 和候选/正式发布边界看，本次 Developer Helper 修复达到源码交付条件。由于当前验证环境不是 Windows，真实 Windows PowerShell/Wails runtime 启动仍需在用户机器执行上述复验；不能用静态 Gate 冒充真实 Windows GUI 运行结果。

### Validation

<!-- historical source: docs/development/VALIDATION-0.1.89-ENVIRONMENT-MANAGER-V2.md -->

## 阶段目标

在不破坏 0.1.89 已冻结的组织安全、成员核心授权和 XiaoYu Intelligence 边界前提下，将游戏运行环境从“SteamCMD 全局前置”重构为可按游戏声明依赖的统一 Runtime Manager。

## 已实现

### Runtime Registry

- Runtime 类型：Java / SteamCMD。
- Java 支持 8 / 17 / 21 / 25 多版本共存。
- AGMP-managed Runtime 默认位于 `<AGMP_ROOT>/runtime/environments`。
- 可登记用户已有的外部 Runtime；Java 会实际执行 `java -version` 并验证主版本。
- Registry 持久化到 AGMP DataDir；默认 Runtime 选择重启后保持。
- 修复/重新登记 Runtime 时，如未显式切换默认项，不会意外清除原 Default 状态。
- 删除外部 Runtime 只删除登记记录，不删除用户文件。
- 自定义 managed 目录如果位于 AGMP RuntimeRoot 之外，也只取消登记，防止用户提供路径演化为任意递归删除原语。

### Java

- Eclipse Adoptium 最新 JRE 元数据。
- 支持 Windows / Linux / macOS 的 amd64/arm64（以 Adoptium 支持能力为准）。
- 下载后使用 Adoptium 提供的 SHA256 做 fail-closed 校验。
- 安装后通过 Shared Process Runtime 执行 `java -version`，验证实际主版本。
- Java 8 legacy `1.8.x` 与 Java 17/21/25 modern version parser 均有回归测试。

### SteamCMD

- Windows：官方 SteamCMD ZIP。
- Linux：官方 `steamcmd_linux.tar.gz`，managed executable 为 `steamcmd.sh`。
- 基础 AGMP Environment 初始化不再强制要求 SteamCMD。
- SteamCMD 只由需要它的 Game Runtime Profile 声明。
- SteamCMD 下载会记录实际 SHA256；当前实现不声称存在与 Java 相同的“上游公开预期 SHA256 比对”。

### Game Runtime Profile

- Minecraft：默认 Java 21，可指定需要的 Java 主版本；不要求 SteamCMD。
- DST：要求 SteamCMD。
- Linux amd64 DST：额外诊断 32-bit glibc loader 与 32-bit libstdc++。
- Linux 系统依赖自动安装只支持固定 ID 和固定包白名单，使用 apt-get/dnf/yum/pacman 的参数数组，不经过 shell。
- i386 multiarch 等全局系统配置不自动偷偷修改；缺失时由包管理器返回明确失败信息。

### Human / XiaoYu 共用 Domain

Web、Wails、TypeScript Backend、系统设置 UI 和 XiaoYu Tools 都调用同一 Application / Environment Service。

XiaoYu Environment Tools：

- `environment.status`
- `environment.catalog`
- `environment.game_profile`
- `environment.install_java`
- `environment.install_steamcmd`
- `environment.install_system_prerequisite`（RiskSystem）

最后一项属于系统级修改，不能成为任意 Shell；仍需走 XiaoYu 已冻结的 Role / Core Access / Risk Step-up / Approval 链。

## 安全回归

当前真实通过：

```text
go test ./internal/deploy/environment ./internal/system/auth ./internal/xiaoyu/host ./internal/app ./internal/bridge/httpapi
go vet  ./internal/deploy/environment ./internal/system/auth ./internal/xiaoyu/host ./internal/app ./internal/bridge/httpapi
go test -race ./internal/deploy/environment
```

Environment Manager 关键测试包括：

- 基础初始化不要求 SteamCMD。
- Minecraft / DST 只声明各自相关依赖。
- RuntimeRoot 位于 AGMP Root。
- Registry 默认项持久化。
- 修复安装保留默认项。
- Java 8/17/21/25 版本解析。
- archive traversal 拒绝。
- ZIP symlink 拒绝。
- tar.gz symlink 拒绝。
- 外部 Runtime 删除不删除外部文件。
- RuntimeRoot 外 custom managed 路径只取消登记。
- Java 安装关键区由 install mutex 串行化。
- Runtime Catalog 使用 AGMP 服务端平台，而不是访问 Web UI 的浏览器平台。

## 自动 Gate

`scripts/common/check-environment-manager.mjs` 已接入 Windows Checks 和 GitHub Safety Workflow，并验证：

- Environment 包没有 `os/exec` 第二套 Process Core。
- Java / Linux prerequisite 都使用 `internal/platform/runtime`。
- Java 多版本、Windows/Linux SteamCMD、Game Profile、Linux prerequisite、HTTP/Wails/TS/UI/XiaoYu Tool 接线存在。
- 系统 prerequisite XiaoYu Tool 必须为 `RiskSystem`。
- 关键安全测试存在。

本阶段检查时，19 个 common Gate 中除正式 Release Key Gate 外均通过；开发阶段使用 `check-release-key.mjs --allow-unconfigured` 时全部通过。正式发布仍必须配置 active Release Key。

## 当前环境阻塞

- App/HTTP 的本轮 Race Detector 在当前容器长编译/传输时超时，因此不能把这一轮写成通过；此前 Intelligence 阶段曾通过对应 Race，但本记录不借用旧结果冒充本阶段验证。
- Wails 完整编译仍需要下载/缓存 `github.com/wailsapp/wails/v2`；当前环境访问 `proxy.golang.org` DNS/网络失败。
- Frontend 当前没有完整 pnpm/node_modules，因此不能声明 `vue-tsc` / Vite production build 已完成。
- Rust 当前环境缺少 cargo/rustc，因此不能声明 XiaoYu Runtime 的 cargo fmt/check/test/clippy 已完成。

## 阶段判断

Environment Manager v2 的 Go Domain / Registry / Profile / HTTP / Application / XiaoYu Tool / Settings 接线与安全测试已达到阶段冻结条件；最终 0.1.89 正式 Release 仍必须补齐 Frontend、Rust、Wails 和正式 Release Key 的真实构建验收。

### Validation

<!-- historical source: docs/development/VALIDATION-0.1.89-INTELLIGENCE.md -->

## 阶段目标

在已经冻结的 **Human + XiaoYu 两个大脑、同一副身体**、Rust Brain-only、Go Host/Tool 执行边界上，完成 XiaoYu 的实例私有 Intelligence 层，并保持组织协作与数据隔离：

- Memory、Skills、Experts 都属于当前 AGMP 实例；不存在公网 Global Memory。
- Intelligence 可见性只允许 `private / group / organization`。
- 同一组织成员按 ACL 协作；不属于组织的账号无法读取组织 Intelligence。
- 进入模型之前由 Host 先过滤 Organization / Group / User / Run Context；禁止依赖模型“看到了但不说”。
- Sensitive Memory 只保存在本地管理面，不自动发送给模型 Provider。
- Rust XiaoYu 仍是唯一语义 Brain；Memory / Skill / Expert 是受限上下文，不能扩大 Tool/权限/License/Approval。

## 已完成实现

### 1. Intelligence Store

`internal/xiaoyu/host/intelligence.go`

- Memory：Session / Task / User / Server / Instance / Experience。
- 可见性：Private / Group / Organization。
- 敏感级别：Normal / Sensitive。
- 自定义 Skill：Prompt、Tags、Game IDs、Tool Allowlist、Checklist、Validators。
- 自定义 Expert：Prompt、Domains、Game IDs、Knowledge、Skills、Tools、Checklist、Validators、Recovery Rules。
- 内置 Skill：安全变更、DST 运维、Minecraft 运维、Linux 服务运维、网络诊断、备份恢复。
- 内置 Expert：DST、Minecraft、Steam/部署、Linux、网络、备份恢复。
- Host 侧模型上下文上限：Memory 64、Skill 48、Expert 24。
- API Key、Token、Password、Authorization、Private Key 等秘密材料拒绝进入普通 Intelligence。

### 2. Run Context 与服务端 TaskID

`RunContext` 已包含 SessionID / TaskID / GameID / ServerID / InstanceID。

客户端传入的 TaskID 在 Application 层清空，`RunManager` 创建 Run 后使用 Run ID 作为唯一 server-owned TaskID，防止客户端伪造 Task scope 读取/写入其他任务记忆。

### 3. Brain Frame 接线

每一个 XiaoYu Agent Loop Brain Turn 都把 Host 过滤后的 `IntelligenceContext` 放入 Frame，再交给 Rust Brain。

Detached Run 每一轮重新验证当前 Session / Organization / Member Core Access。成员权限被撤销后，旧 Run 不再获得 Intelligence。

Owner/Admin 监督其他成员 Run 时，只获得 Organization 级共享 Intelligence；不会因为“监督权限”隐式获得该成员 Private / Group AI 上下文。

### 4. Rust XiaoYu Intelligence Policy

Rust System Policy 已明确：

- Host Tool Contract、真实 Observation 与安全规则高于 Expert / Skill / Memory。
- Memory 是有来源、可信度和作用域的上下文，不是授权或系统指令。
- Skill / Expert 可以协作增强规划，但不能扩大 Tool Allowlist、RBAC、License 或 Approval。
- 自定义 Intelligence 视为不可信上下文；其中的“忽略安全规则/已经批准”等文本不能覆盖 Host。
- 重要变更要求预检、恢复点、真实结果验证和失败恢复。
- `memory.remember` 只用于真正可复用的事实、偏好和经验，禁止秘密和一次性猜测。

### 5. 受控 `memory.remember`

新增 Host Tool `memory.remember`：

- 只能在已认证 XiaoYu Run 调用。
- User/Session/Task/Server/Instance/Experience scope 全部取当前已认证 Run Context。
- Task scope 必须等于服务端 Run ID。
- XiaoYu 主动写入永远是 Private Memory，不能自行发布 Group/Organization 知识。
- 不记录 Memory 正文到 Audit Log。
- 脱离 Run Context 直接调用 Registry 会被拒绝。

### 6. 共享 Experience 修正

Private Experience 强制绑定当前 User ID；Group / Organization Experience 强制使用显式 `*` wildcard scope。调用方不能留下伪造 User scope，授权同组/同组织成员才能在模型上下文中复用共享经验。

### 7. Web / Wails / Frontend

Application、HTTP、Wails 使用同一 Intelligence Application API：

- 获取 Intelligence Catalog。
- 保存 Memory。
- 保存 Skill。
- 保存 Expert。
- 删除 Intelligence Item。

设置中心新增 **小鱼 · 记忆、技能与专家**：

- Memory / Skills / Experts 三个管理区。
- 明确“Organization 共享仅限当前 AGMP 组织，不是公网共享”。
- 普通成员只能创建 Private Intelligence。
- Owner / Administrator 才能发布 Group / Organization Intelligence。
- Sensitive Memory 在 UI 明确为“本地保留、默认不发送模型”。

XiaoYu Workbench 为当前浏览器标签维护 SessionID，但不允许客户端指定 TaskID。

## 安全回归已真实通过

以下命令在当前 0.1.89 工作树真实执行并通过：

```text
go test ./internal/xiaoyu/host ./internal/system/auth ./internal/app ./internal/bridge/httpapi
go test -tags agmp_dev_license ./internal/xiaoyu/host ./internal/system/auth ./internal/app ./internal/bridge/httpapi
go vet ./internal/xiaoyu/host ./internal/system/auth ./internal/app ./internal/bridge/httpapi
```

除 Wails 外的全部 `internal` Go 包也分别在正式模式与 `agmp_dev_license` 模式完成回归，并通过 `go vet`。

Race Detector 已真实通过：

```text
go test -race ./internal/xiaoyu/host
go test -race -tags agmp_dev_license ./internal/app
go test -race -tags agmp_dev_license ./internal/bridge/httpapi
```

新增的关键回归包括：

- Private / Group / Organization 可见性。
- Sensitive / Expired / 低可信 Memory 不进入模型。
- Memory / Skill / Expert 秘密材料拒绝。
- Server-owned TaskID，忽略客户端伪造值。
- 成员不能写入另一个成员 Task Memory。
- Supervisor 监督 Run 不泄漏发起者 Private / Group Intelligence。
- 成员 Core Access 被撤销后 Detached Run Intelligence fail-closed。
- Organization Experience wildcard scope。
- `memory.remember` 只能使用认证后的 server-owned Run scope。
- Release build 无官方 License 时 Intelligence API fail-closed。

## 自动 Gate

新增 `scripts/common/check-xiaoyu-intelligence.mjs`，并已接入：

- Windows `Checks.ps1` XiaoYu Gate。
- `.github/workflows/safety.yml` 常规 Safety Job。
- Linux Headless AI parity Job。

当前全部 18 个 `scripts/common/check-*.mjs` Gate 均 `rc=0`；Release Key Gate 使用开发阶段 `--allow-unconfigured`，正式发行仍必须配置 active 发行公钥。

`Checks.ps1` 已复核为 UTF-8 BOM + CRLF。

## 前端静态语法验证

当前容器没有 pnpm/node_modules，无法声称 `vue-tsc` / Vite build 已完成。但使用当前环境 TypeScript 5.8.3 parser 对以下脚本做了语法级解析，全部通过：

- `frontend/src/shared/types/backend.ts`
- `frontend/src/shared/api/backend.ts`
- `XiaoYuIntelligenceSection.vue` 的 `<script setup lang="ts">`
- `SettingsView.vue` 的 `<script setup lang="ts">`
- `AIWorkbenchView.vue` 的 `<script setup lang="ts">`

该检查只证明 TypeScript 语法有效，不替代最终 `pnpm typecheck/build`。

## 当前环境阻塞项

### Wails

根级 `go test ./internal/...` 会在 `internal/bridge/wails` 停止：

```text
missing go.sum entry for module providing package github.com/wailsapp/wails/v2/pkg/runtime
```

因此本环境不能声称 Wails 完整编译通过。其他 internal Go 包已通过。

### Frontend

当前没有 pnpm / frontend node_modules，且环境不能访问 npm registry，因此不能完成真实 `pnpm install / typecheck / build`。

### Rust

当前容器没有 cargo/rustc，因此不能完成真实 `cargo fmt / check / test / clippy`。Rust Intelligence Policy 已由静态 Gate 检查，但正式 Release 前必须在具备 Rust 工具链的开发/CI 环境执行完整验证。

## 阶段冻结结论

**XiaoYu Intelligence 功能链在当前能够真实执行的 Go/HTTP/Race/Gate 范围内已完成并可冻结。**

冻结边界：

1. Rust 继续是唯一 Brain / Policy / Semantic Decision 边界。
2. Go Host 只负责身份、组织 ACL、License/Core Gate、Tool、持久化、作用域过滤与执行。
3. Intelligence 永远先经过 Host ACL，再进入模型。
4. Private / Group / Organization 是唯一用户 Intelligence 共享层级；禁止跨 AGMP 实例公网 Global Memory。
5. Sensitive Memory 默认永不自动进入 Provider。
6. Supervisor 不因为监督 Run 自动获得成员私人 AI 上下文。
7. TaskID 永远由服务器生成。
8. `memory.remember` 永远不能自行发布组织共享知识。

下一阶段可以在不改动上述冻结边界的前提下进入 Environment Manager v2（Windows/Linux Java 多版本、SteamCMD 与游戏运行依赖）。


---

## AI-Game-Manager-Panel 0.1.88

### Release Notes

<!-- historical source: docs/releases/0.1.88.md -->

## 定位

0.1.88 不重做 0.1.83/0.1.85 已冻结的领域骨架与 Shared Runtime，而是把 AGMP 的产品控制模型进一步定型为：**Human + XiaoYu，两个大脑、同一副身体**。用户负责目标、边界、审批与最终接管；XiaoYu 负责理解、规划、观察、验证和自动执行。两者必须使用同一 AGMP Host、Domain Action 与 Shared Runtime。

## 本版核心

- XiaoYu Harness：Plugin Kernel、Capability Discovery、Agent Loop、Step/Tool/Failure Budget、Doom-loop 防护、Observation、等待审批/等待用户、暂停/接管/恢复、服务器端 RunManager 与有界 Trace。
- Model Center：系统设置中集中管理大模型，支持主流云 Provider、本地 Ollama/LM Studio、自定义 OpenAI-Compatible 与第三方中转；默认模型作为 XiaoYu 大脑来源，API Key 与普通 Profile 分离存储。
- Rust `xiaoyu-core` 保持唯一 Brain/Policy/Decision 边界；Go 仅承担模型 HTTP 传输、Host/Run 生命周期、安全策略与领域执行，不复制 Planner/Memory/语义恢复。
- Web/Linux/Docker AI Feature Parity：Linux Server、Docker 和 Web 都是完整产品端，正式发行必须携带预编译 XiaoYu Runtime；浏览器关闭只断开事件订阅，不取消服务器上的 XiaoYu Run。
- 双大脑控制权：新增 Pause / Takeover / Resume / Cancel；Human override 永远优先，人工接管后 XiaoYu 不得自行继续。
- 多用户 Web Run 隔离：Run 记录 Initiator；Owner/Admin 可全局监督，普通 Operator 只能读取/控制自己创建的 Run 和对应 Event Stream。
- XiaoYu Event Stream：服务端有界 Sequence Trace，支持断线后按 sequence 续接；最终写入边界统一脱敏，慢客户端不会阻塞 Agent。
- DSH Tool Bridge：保持 `xiaoyu.plugin.v1` 为 AGMP 自己的稳定插件边界；DeepSeek Harness Tool 插件经兼容 Adapter 接入。外部 JS 默认在 Node Permission 受限子进程中运行，只读插件自身目录、不继承 AGMP 秘密环境、不允许 child_process/文件写入；插件 entry/patch/root 做真实 symlink 边界校验。
- Shared Domain Action：XiaoYu DST Tool 与人工入口开始统一复用相同领域动作，禁止 AI 直接绕过 Domain 调用游戏 Runtime。
- Docker 多架构 XiaoYu 修复：Rust builder 使用目标平台构建，避免 arm64 镜像意外携带 amd64 XiaoYu 二进制。

## 发行原则

完整 AGMP 的核心能力组合固定为：`AGMP Core + XiaoYu Runtime + Web UI`。Wails/Electron 只是桌面入口；删除桌面层后 Linux/Headless/Web 仍必须拥有完整模型中心、Harness、Tool、审批、Run 和事件能力。

## 仍属后续增强

- 真正 PTY / Windows ConPTY、Job Object 和跨 AGMP 重启的外部 Session Reattach。
- DSH 完整 Cordis/Profile 兼容与更强 OS 级插件沙箱；0.1.88 只承诺受限 Tool Bridge，不承诺任意 DSH 插件零修改兼容。
- XiaoYu Memory/长期 Context、更多结构化 Domain Tools、更多游戏 Adapter 与更深的执行后语义验证。
- Windows/Linux 真机完整 Rust + Vue + Wails/Electron/Installer 长时间压力验收。

### Validation

<!-- historical source: docs/development/VALIDATION-0.1.88.md -->

## 验证目标

验证 0.1.88 在不推翻 0.1.83/0.1.85 已冻结骨架与 Shared Runtime 的前提下，真正建立 **Human + XiaoYu 两个大脑、同一副身体**，并防止 Web/Linux/Docker 出现“能打开面板但没有 XiaoYu”的阉割发行。

## 当前环境已真实通过

- **14 个结构/安全 Gate**：GitHub Safety、Project Layout、Product Architecture、Module、XiaoYu Core、XiaoYu Harness、XiaoYu Model Center、Multi-Client AI、Headless XiaoYu、Distribution Boundary、Windows Helper、Windows Installer、Updater、Release Key（开发模式允许未配置 active key）。
- **44 个可用 `internal` Go package**（排除因外部 Wails checksum 缺失无法载入的 `internal/bridge/wails`）：`go test` 全部通过。
- 同一批 44 个 package：`go vet` 全部通过。
- `agmp_dev_license`：`internal/system/license` 与 `internal/bridge/httpapi` 回归通过。
- 关键并发/安全路径 `go test -race` 通过：`platform/runtime`、DST Runtime、XiaoYu Contract、XiaoYu Control、XiaoYu Host、`ops/files`、`app`、HTTP API。
- 当前 Linux 环境实际构建通过：`cmd/aigame-manager-web`、`cmd/aigame-manager-license-admin`。
- Windows amd64 / CGO disabled 编译级回归通过：Shared Runtime、DST Runtime、XiaoYu Runtime、XiaoYu Host、Updater、Workspace Files。
- DSH Tool Bridge 的真实 Node 测试通过，包括受限 Node Permission 进程、Tool list/call、配置传递、卸载撤销、symlink 根目录逃逸拒绝与宿主秘密环境隔离。
- 多用户 XiaoYu Run 访问规则单测通过：发起者可控制自己的 Run，Operator 不能控制/订阅其他人的 Run，Owner/Admin 可以全局监督。
- Trace 最终写入边界秘密字段脱敏、过滤后 backlog/live stream 隔离、慢消费者有界行为测试通过。

## 新增 CI 强制验证

`.github/workflows/safety.yml` 已加入：

- XiaoYu Harness Gate
- XiaoYu Model Center Gate
- Multi-Client AI Parity Gate
- Linux Headless XiaoYu Gate
- 独立 `Linux Headless + Web + XiaoYu` 作业：构建 Rust XiaoYu、Vue Web、Headless Go Core，并通过真实 HTTP `/api/v1/xiaoyu/runtime` 集成测试验证 **不启动任何 Desktop Shell 也能连接 XiaoYu Runtime**。

这些 CI 步骤已写入并通过本地静态 Gate；由于当前执行环境缺少 Cargo/pnpm，独立 GitHub Actions 作业本轮无法在本机实际执行，必须由在线 CI 再做最终确认。

## 当前无法在本环境完成

当前容器没有 `cargo`、`rustc`、`pnpm`、`wails`，因此本轮不能在这里声称以下内容已通过：

- Rust `cargo fmt/check/test/clippy/build` 的真实 0.1.88 完整执行；
- Vue `pnpm install/build/typecheck`；
- Wails/Electron 完整桌面构建和启动；
- Docker amd64/arm64 真实 BuildKit 多架构镜像构建；
- Windows/Linux 真机长时间 XiaoYu + 游戏服务器压力测试。

## `go test ./...` 仍被源码依赖完整性阻塞

根级 `go test ./...` 会在 Wails/根桌面入口前停止，当前原因：

1. 源码仍没有 `go.sum`，Wails v2.15.0 依赖缺 checksum；
2. `main.go` 的 `frontend/dist` embed 需要先完成 Vue 构建；
3. 当前隔离环境尝试 `go mod download` 时 DNS 无法访问 `proxy.golang.org`，因此没有伪造 `go.sum`。

这属于**正式发行前必须解决的构建完整性项**，不是本轮 Go 业务测试失败。

## 0.1.88 冻结判断

- 主骨架：继续冻结，不再大搬家。
- Shared Process Runtime：继续冻结，只允许扩展 PTY/ConPTY/Job Object/Reattach。
- Rust XiaoYu：继续冻结为唯一 Brain/Policy/Decision 边界。
- Go XiaoYu Host：允许继续扩展 Harness/Plugin/Model transport/Run lifecycle，但不得出现第二套 Planner/Memory/语义恢复大脑。
- Web/Linux/Docker XiaoYu Feature Parity：从本版起视为硬规则，禁止回退。
- Human override：从本版起视为硬规则，禁止 XiaoYu 与用户争夺同一资源控制权。

## 交付 ZIP 二次验收

最终源码 ZIP 生成后重新解压到独立目录，并从解压副本再次通过：Project Layout、Product Architecture、Module、XiaoYu Core、XiaoYu Harness、Model Center、Multi-Client AI、Headless XiaoYu、GitHub Safety Gate，以及 Shared Runtime、DST Runtime、XiaoYu Host/Runtime、Application、HTTP API 核心 Go 测试。交付文件 SHA256 随后重新生成并全部校验通过。

### Completion

<!-- historical source: docs/development/COMPLETION-0.1.88.md -->

## 总结

**架构方向：正确且已进入长期稳定阶段。底层：可冻结。XiaoYu Harness/模型中心：已有真实第一版，但智能能力仍在成长。产品总体功能：仍处于早中期。**

0.1.88 最大变化不是增加一个 AI 页面，而是把产品控制模型真正改造成：

`Human Brain + XiaoYu Brain -> 同一 AGMP Host -> 同一 Domain -> 同一 Shared Runtime`

人工与 XiaoYu 可以选择不同的控制方式，但不能拥有两套相互竞争的业务实现。

## 已完成的关键能力

### XiaoYu 大脑与 Harness

- Rust `xiaoyu-core` 保持 Brain-only；真实 OS/文件/游戏执行仍全部回到 Go Host/Domain。
- `xiaoyu.plugin.v1` Plugin Kernel、Capability Discovery、可逆 Tool 注册。
- Agent Loop / RunManager、Step/Tool/Failure Budget、Doom-loop 防护。
- Observation、等待审批、等待用户、取消、Pause/Takeover/Resume。
- Server-owned Run：浏览器断开不会取消 XiaoYu。
- 有界 Sequence Trace/Event Stream、断线续接基础、慢消费者保护。
- 多用户 Run Initiator 隔离与 Owner/Admin 全局监督。

### XiaoYu 大脑配置

- 系统设置 -> 模型管理成为唯一用户级大脑配置中心。
- 支持 OpenAI、DeepSeek、Claude、Gemini、OpenRouter、国内兼容 Provider、本地 Ollama/LM Studio、自定义 OpenAI-Compatible/中转站。
- 默认模型选择、模型发现、连接测试、上下文/输出 Token/思考模式/额外 JSON。
- API Key 与普通 Profile 分离；Windows DPAPI / 非 Windows 本地加密 Vault；前端不读取完整 Key。
- 没有可用默认模型或 Rust Brain 时 XiaoYu Fail Closed，不使用 Shell 冒充 AI。

### 手脚与底层

- Shared Runtime 仍是唯一 Process/stdin/stdout/stderr 实现。
- DST 已复用 Shared Runtime。
- 首批结构化 Domain Tools 已覆盖系统、文件、游戏发现、Steam/环境状态、DST Cluster/命令/日志。
- `process.run` 仍是人工/管理员兜底，默认不作为 XiaoYu 日常手脚。
- DST XiaoYu Tool 与人工启动入口开始统一复用相同 Domain Action。

### 多端

- Windows Wails / Windows Electron / Web / Linux Server Web / Docker Web 都被定义为完整 AGMP 产品目标。
- Linux Server、Linux 发行链、Docker 强制携带预编译 XiaoYu Runtime，缺少 XiaoYu 禁止出包。
- Docker Rust builder 修正为目标平台构建，避免多架构镜像混入错误架构 XiaoYu。
- Web API 已具备模型中心、Harness、Tools、Capabilities、Run Control、Approval、Trace/Event Stream。
- CI 已加入真实 Linux Headless Web -> XiaoYu Runtime HTTP 集成测试。

### 插件安全

- DSH 兼容层仍通过 Adapter，不让 AGMP 核心依赖 DSH 内部版本。
- DSH Tool JS 在独立 Node 子进程运行。
- 默认 Node Permission：只读插件自身目录，不开放文件写入和 child_process。
- 不继承 AGMP API Key/Token 等宿主秘密环境。
- 插件 root/entry/patch 使用真实 symlink 路径边界检查。
- 仍需显式 Owner 信任后才能挂载外部 DSH 插件。

## 模块状态

`configs/modules.json` 当前共 33 个模块：

- `implemented`：1
- `active`：1
- `partial`：7
- `skeleton`：24

这说明**架构成熟度仍明显高于产品功能成熟度**。0.1.88 不应被理解为“产品已经接近完成”。

## 仍需继续开发

优先级最高：

- XiaoYu Memory / 长期 Context / 任务上下文压缩；
- 更成熟的 Planner/Recovery/结果语义验证；
- Capability Router，避免未来数百 Tool Schema 每轮全部进入模型上下文；
- Backup、Instance、Minecraft、Steam Update、Network 等更多结构化 Domain Tools；
- Minecraft Adapter 与更多游戏；
- Run 持久化与 AGMP 重启后的恢复；
- 插件更强 OS 级沙箱及 DSH 更完整兼容。

底层增强但不改架构：PTY/ConPTY、Windows Job Object、外部进程 Reattach、长期压力/故障注入测试。

## 当前最重要的发行阻塞项

- 补正式 `go.sum`；
- 在线 CI 跑通 Rust + Vue + Headless XiaoYu 作业；
- Windows Wails/Electron/Installer 实机完整构建；
- Linux/Docker amd64 + arm64 实际构建；
- 至少一次真实模型 Provider + Linux Web + XiaoYu Autonomous Run 端到端验收。


---

## AI-Game-Manager-Panel 0.1.85

### Release Notes

<!-- historical source: docs/releases/0.1.85.md -->

0.1.85 是 0.1.84 之后的底层定型版本。本版本不改变已经冻结的主业务骨架，重点把 **XiaoYu Brain、Host Tool 与 OS/Process Runtime 的执行边界彻底收口**，并补齐高日志量、多会话和长期运行所需的基础能力。

## 本版完成

- `internal/platform/runtime` 冻结为 AGMP **唯一 Process / stdio 底层边界**。
- Session 支持进程生命周期、串行 stdin、stdout/stderr 来源分离、Sequence、PID/状态/起止时间/退出码快照、Terminate/Kill。
- 输出历史改为固定容量环形缓冲，避免高日志量场景每行移动完整数组。
- 新增非阻塞实时输出订阅；慢消费者不会阻塞游戏服务器 stdout/stderr 管道。
- Manager 增加 Send/SendLine/History/Subscribe/StopAll 等统一会话操作。
- 一次性命令继续统一使用有界 `Run`；受控 Shell 使用 `RunShell`；非交互外部程序使用 `StartDetached`。
- 环境变量采用父环境继承 + Overlay，避免自定义变量时丢失 PATH 等基础环境。
- Linux/macOS 终止使用进程组；Windows 使用新进程组并按场景控制控制台窗口。
- DST、XiaoYu Runtime、Updater 等进程入口继续复用 Shared Runtime；业务域禁止自行创建第二套 `os/exec` / stdio 管线。
- Rust `xiaoyu-core` 正式冻结为 **Brain-only**：不直接 `std::process`、不直接 `std::fs`、不执行 `fs.read/fs.list/process.run`。
- 可执行 Tool 统一归 AGMP Go Host Registry；小鱼负责理解、规划、策略提示与 Tool 编排，真实执行回到所属业务模块。
- Host Tool Registry 拒绝同名覆盖并校验 Tool 名称/风险，避免后注册 Tool 劫持已有能力。
- `internal/ops/files` 增加工作区文件沙箱，使用 canonical/symlink 校验阻止路径逃逸。

## 底层冻结规则

`platform/runtime` 是唯一 Process Core。PTY/ConPTY、跨 AGMP 重启的会话重连等后续能力只能扩展这一 Runtime，禁止在游戏、XiaoYu、Updater 或其他业务域重新实现 Process Core。

XiaoYu Rust Core 是 AGMP 的超级大脑，不是系统 Shell。标准调用链固定为：

`XiaoYu Brain → AGMP Host Tool Registry → Permission/Approval → Domain Handler → platform/runtime / domain platform → OS/Game`

## 仍属后续增强

- 真正 PTY / Windows ConPTY 交互传输。
- AGMP 主程序重启后的外部进程重新发现与会话重连。
- 更高级的 Windows Job Object / 完整进程树约束（需要 Windows 实机专项验证）。
- XiaoYu Provider、Planner/Executor 循环、Context/Memory、Workflow 与完整领域 Tool 覆盖。

这些增强均不要求重新设计当前 Process/Tool/Brain 边界。

### Validation

<!-- historical source: docs/development/VALIDATION-0.1.85.md -->

## 验证目标

验证 0.1.85 Runtime Foundation Freeze 没有产生第二套 Process Core，并确认 Shared Runtime、XiaoYu Host Tool 边界与现有 Go 业务在本环境可稳定回归。

## 已通过

- **10 个 Gate 全部通过**：Project Layout、Module、XiaoYu Core、Product Architecture、Distribution Boundary、Windows Helper、Windows Installer、Updater、GitHub Safety、Release Key（开发模式允许未配置 active issuer key）。
- **43 个可用 `internal` Go package**：`go test` 全部通过。
- 同一批 43 个 package：`go vet` 全部通过。
- 关键底层 `go test -race` 通过：`platform/runtime`、DST Runtime、XiaoYu Contract、XiaoYu Control、`ops/files`、`app`。
- 本机实际构建通过：`cmd/aigame-manager-web`、`cmd/aigame-manager-license-admin`。
- Windows amd64 / CGO disabled 交叉编译通过：`platform/runtime`、DST Runtime、XiaoYu Runtime、Updater、`ops/files`。
- 扫描确认：`internal/platform/runtime` 之外的 Go 业务域没有直接 `exec.Command/CommandContext/StdinPipe/StdoutPipe/StderrPipe`；Rust `xiaoyu-core` 没有 `std::process/std::fs/Command::new` 或本地 `fs.read/fs.list/process.run` 执行实现。

## `go test ./...` 当前为何仍不通过

完整根级测试会在 Wails/桌面入口前停止，原因不是本轮 Go 业务代码测试失败，而是源码当前仍没有可用 `go.sum`，环境也无法联网拉取 Wails v2.15.0；同时本环境没有生成 `frontend/dist`。错误集中在 Wails module checksum 与 `embed all:frontend/dist` 缺失。

## 当前环境限制

当前执行环境没有 `cargo`、`rustc`、`pnpm`、`wails`。因此以下项目仍需 Windows 开发机完整验收：

- Rust `xiaoyu-core + xiaoyu-protocol` 真正 `cargo test/clippy/build`。
- Vue/TypeScript `pnpm build`。
- Wails/Electron/Windows 桌面程序与 Setup 实机运行。
- 真实 DST/SteamCMD 长时间控制台压力与 Windows GUI 行为。

## 底层冻结判断

标准 Process/stdin/stdout/stderr、会话生命周期、输出背压/历史、Host Tool 执行边界已经达到可冻结级别。以后新增 Minecraft、Terraria 等符合“可执行程序 + 参数 + stdio 控制”的服务器，应复用该 Shared Runtime。

真正 PTY/Windows ConPTY、AGMP 主程序重启后的外部进程重新发现/会话重连、Windows Job Object 进程树管理属于后续高级能力；只能扩展现有 Shared Runtime，不得另建第二套 Process Core，因此不会要求推翻当前底层架构。

### Completion

<!-- historical source: docs/development/COMPLETION-0.1.85.md -->

## 结论

**底层核心：可冻结。产品功能：仍处于早中期。XiaoYu 超级大脑：架构边界已定，智能闭环尚未完成。**

0.1.85 已把最容易导致未来“大换心”的部分收口：Rust `xiaoyu-core` 唯一负责 Brain/Policy/Protocol；AGMP Go Host 唯一负责 Tool 注册、最终权限/审批与领域分发；`internal/platform/runtime` 是唯一 Process/stdio 实现。

## 本轮底层完成

- 统一长生命周期 Session 与 Manager。
- stdout/stderr 分源、全局 Sequence。
- 固定容量环形历史，避免高日志量 O(n) 移动。
- 非阻塞实时订阅，慢 UI/WebSocket 不阻塞游戏进程输出。
- 串行 stdin / Send / SendLine / CloseInput。
- PID、状态、开始/退出时间、退出码与错误快照。
- Environment parent inheritance + Overlay / 显式 Replace。
- bounded Run、Shell、超时/取消、输出上限、Detached launch。
- Unix process-group TERM/KILL；Windows process group + hidden-window foundation。
- DST、XiaoYu Runtime、Updater 等现役进程入口统一复用 Shared Runtime。
- Gate 强制禁止业务域恢复 `os/exec`/stdio 私有实现。
- XiaoYu Rust Brain 清除直接 OS/file/process 执行；Host Tool Registry 拒绝重复名/非法名/非法风险等级。
- 工作区文件 Tool 使用 canonical + symlink 边界检查。

## 当前模块总体状态

`configs/modules.json` 目前共 33 个模块：

- implemented：1
- active：1
- partial：6
- skeleton：25

因此不能把“底层架构完成”误解为“AGMP 产品已经接近完成”。

## XiaoYu 下一阶段重点

- Model Provider 抽象与真实模型接入。
- Planner / Executor Agent Loop。
- Context Builder 与系统状态感知。
- Memory / Session / Task Context。
- 多步骤 Workflow、失败恢复、重试与结果验证。
- 将文件、备份、实例、服务器、Steam、环境、网络、日志等真实 Domain Tool 逐步注册到 AGMP Host Registry。

## 后续底层增强（不改架构）

- PTY / Windows ConPTY。
- 跨 AGMP 重启的外部进程发现和 Session Reattach。
- Windows Job Object / 进程树约束。
- 大规模长期运行压力测试与 WebSocket 终端恢复策略。

这些功能都在现有 `platform/runtime` 和 Host Tool 边界上扩展，不再允许重写第二套底层。


---

## AI-Game-Manager-Panel 0.1.84

### Release Notes

<!-- historical source: docs/releases/0.1.84.md -->

0.1.84 不再调整 0.1.83 已冻结的主要业务骨架。本版只继续完成底层运行时，让“进程、终端、子进程执行”真正成为 AGMP 的单一平台能力。

## 完成

- `internal/platform/runtime` 从基础 Session 扩展为统一 Process Runtime：
  - 长生命周期 `Session`；
  - 有界一次性 `Run`；
  - 异步回收的 `StartDetached`；
  - 命名 `Manager`。
- stdout / stderr 保留来源，不再强制合流。
- stdin 写入串行化，同时支持 raw `Send` 与 `SendLine`。
- Session 提供 created / starting / running / stopping / exited / failed 状态、PID、开始/退出时间、退出码快照。
- 环境变量默认继承父进程，再由调用方 Overlay；只有显式 `ReplaceEnvironment` 才完整替换。
- 一次性命令增加输出大小上限、Context 超时/取消和进程终止。
- Linux/macOS 使用独立进程组，Terminate/Kill 作用到整个进程组；Windows 继续使用独立进程组和隐藏控制台，GUI 安装器可显式 `ShowWindow`。
- XiaoYu Runtime 不再维护自己的 Windows/Other 进程启动文件，统一调用 `platform/runtime.Run`。
- Updater、非 Windows 文件夹/浏览器打开器统一使用 `StartDetached`。
- Module Gate 新增硬规则：业务域不得直接 `exec.Command/CommandContext`，也不得在共享 Runtime 之外持有 stdio Pipe。

## 仍然不是本版目标

- PTY / ConPTY 真交互终端；
- AGMP 重启后的跨进程会话重连；
- Minecraft Adapter；
- XiaoYu Provider、Planner/Executor、Memory/Workflow 完整智能闭环。

这些功能以后只能扩展现有边界，不能再另造第二套 Process Core。

### Validation

<!-- historical source: docs/development/VALIDATION-0.1.84.md -->

## Result

**Shared Runtime hardening: PASS in the available environment.**

The architecture remains frozen from 0.1.83; 0.1.84 validates the common process/terminal foundation and removes remaining duplicate process launch paths.

## Go tests and vet

42 internal Go packages were enumerated directly, excluding only `internal/bridge/wails` because this source snapshot still has no `go.sum` and the isolated environment cannot fetch the external Wails module.

- `go test`: PASS for all 42 available internal packages.
- `go vet`: PASS for all 42 available internal packages.
- `internal/platform/runtime`: PASS, including stream source separation, environment overlay, timeout/output bounds and Manager lifecycle tests.
- `internal/games/dst/runtime`: PASS after continued shared Runtime use.
- `internal/xiaoyu/runtime`: PASS after migrating one-shot process execution to `platformruntime.Run`.
- `internal/deploy/updater`: PASS after migrating installer launch to `StartDetached`.

## Build validation

- `go build ./cmd/aigame-manager-web`: PASS.
- `go build ./cmd/aigame-manager-license-admin`: PASS.
- `GOOS=windows GOARCH=amd64 go test -c ./internal/platform/runtime`: PASS.
- `GOOS=windows GOARCH=amd64 go test -c ./internal/games/dst/runtime`: PASS.
- `GOOS=windows GOARCH=amd64 go test -c ./internal/xiaoyu/runtime`: PASS.
- `GOOS=windows GOARCH=amd64 go test -c ./internal/deploy/updater`: PASS.

## Architecture / safety gates

PASS:

- Distribution Boundary Gate
- GitHub Safety Gate
- Module Gate
- Product Architecture Gate
- Project Layout / Version Gate
- Release Key Gate in development mode (`--allow-unconfigured`)
- Updater Gate
- Windows Helper / Encoding / Frontend Import Gate
- Windows Installer Gate
- XiaoYu Core Gate

The Release Key Gate correctly warns that no active publisher key is configured; this is allowed for source development and must be configured before a formal Release build.

## Process ownership invariant

Repository scan found no active direct process creation outside `internal/platform/runtime`:

- no business `exec.Command(...)`;
- no business `exec.CommandContext(...)`;
- no business `StdinPipe/StdoutPipe/StderrPipe` ownership.

`exec.LookPath` may still be used only to discover executable paths.

The Module Gate now enforces this invariant.

## Environment-limited validation not claimed

The current execution environment does not provide:

- Cargo / Rust toolchain;
- pnpm;
- Wails CLI;
- `go.sum` for the Wails external dependency.

Therefore the following are **not claimed as passed** here:

- `cargo check/test --workspace`;
- Vue production build;
- Wails full desktop build;
- Electron full desktop build;
- Windows installer real execution;
- real Windows PTY/ConPTY behavior.

These remain Windows/local CI acceptance items. No checksum or dependency file was fabricated.

### Completion

<!-- historical source: docs/development/COMPLETION-0.1.84.md -->

## 结论

0.1.84 不再做第三次大骨架搬迁，而是在 0.1.83 已冻结架构上把 **共享 Process / Terminal Runtime** 做到可以作为后续游戏控制台的稳定地基。

对“可执行程序 + 参数 + stdin/stdout/stderr”这一类游戏服务器，底层公共控制链已经成立：

```text
Game Adapter / XiaoYu / AGMP Module
                ↓
       platform/runtime
       ├─ Session
       ├─ Manager
       ├─ Run
       └─ StartDetached
                ↓
        Windows / Linux / macOS
```

DST 已实际复用该 Runtime；XiaoYu Runtime 的内部子进程调用、Updater 外部安装器启动、非 Windows 系统打开器也已收口进同一底层。业务域中已无直接 `exec.Command/CommandContext` 或自行 stdio pipe 的现役实现。

## 共享 Runtime 已完成

### 长生命周期 Session

- 进程启动与参数传递；
- Working Directory；
- 环境变量继承 + Overlay；
- 显式 ReplaceEnvironment；
- stdin raw `Send` / `SendLine`；
- 并发 stdin 串行保护；
- stdout / stderr 独立读取并保留来源；
- 4 MiB 单行 Scanner 防护；
- PID；
- created / starting / running / stopping / exited / failed 状态；
- StartedAt / ExitedAt / ExitCode / Error 快照；
- Done / Wait / WaitContext；
- CloseInput；
- Terminate / Kill；
- 真实 `cmd.Wait()` 回收后才发布退出状态。

### 命名 Session Manager

- ID 唯一性；
- Start / Get / Stop / Delete；
- 活跃 Session 禁止误删；
- Terminate 超时后 Kill；
- 完成 Session 可保留快照，显式删除；
- 多 Session 快照稳定排序。

### 一次性 Run

用于 XiaoYu Core 等内部受控 CLI 调用：

- 不经过 Shell；
- stdout / stderr 独立捕获；
- 默认单流 8 MiB 上限；
- 可配置输出上限；
- Context 超时/取消；
- 超时主动 Kill；
- 输出超限显式返回错误，防止无界内存增长或截断 JSON 被误当成功。

### StartDetached

用于安装器、系统打开器等无需交互的外部进程：

- 统一平台进程参数；
- stdout/stderr 丢弃；
- 后台 `Wait()` 回收，避免僵尸/句柄泄露；
- Windows GUI 安装器可显式 `ShowWindow`，不会被共享 Runtime 的默认隐藏窗口策略误伤。

### 平台终止策略

- Windows：新进程组 + 默认隐藏控制台；GUI 可显式 ShowWindow；Terminate/Kill 使用进程句柄终止。
- Linux/macOS：新进程组；Terminate 使用 SIGTERM，Kill 使用 SIGKILL，作用于整个组，避免只杀父进程留下子进程。

## 已经消除的重复实现

- DST 私有 Windows/Other Process 实现：已在 0.1.83 删除。
- XiaoYu Runtime 私有 `process_windows.go / process_other.go`：0.1.84 删除。
- Updater `exec.Command`：改为 `StartDetached`。
- 非 Windows 文件夹/浏览器打开器：改为 `StartDetached`。

Module Gate 已阻止这些实现重新长回来。

## 还没有完成，但不会要求重构底层

### PTY / ConPTY

当前 `Session` 是可靠的 pipe-backed console，已经足够服务 DST、Minecraft Paper 等标准 stdin/stdout 游戏服务器。真正 PTY/ConPTY 仍需后续实现，主要用于：

- 完整 ANSI/光标控制；
- 交互式 Shell；
- 依赖 TTY 检测的程序；
- 更接近原生 PowerShell/cmd/bash 体验。

PTY 会作为现有 Runtime 的 Transport 扩展，**不会再创建第二套 Terminal Core**。

### 跨 AGMP 重启的会话恢复

当前 Session 在一个 AGMP 进程生命周期内可稳定管理。AGMP 自身重启后要重新接回旧控制台，需要进一步设计守护进程/IPC/Named Pipe 或持久 PTY；仅靠 PID 无法安全恢复 stdin/stdout。

这也是后续增强，不需要改变现有 Game Adapter -> platform/runtime 边界。

## XiaoYu 超级大脑状态

0.1.84 继续保证：

```text
Rust xiaoyu-core = 唯一 Brain
Go internal/xiaoyu = contract / control / runtime
```

小鱼当前已有 Rust Core/Protocol、Tool Registry、审批/权限基础和 Go Bridge，但真正智能闭环仍需继续：Provider -> Planner/Executor -> Context/Memory -> Workflow -> Result Validation -> Domain Tools。

因此“底层运行时已经成熟”和“整个 AGMP 已经完成”不能混为一谈。

## 产品模块矩阵

`configs/modules.json` 当前共 33 个模块：

- implemented: 1
- active: 1
- partial: 6
- skeleton: 25

骨架/Runtime 成熟度明显高于业务功能完成度。

## 下一阶段建议

0.1.85 起不再继续搬架构。优先：

1. XiaoYu Provider 与真正 Agent Loop；
2. Planner / Executor / Result Validator；
3. Context / Memory / Workflow；
4. 把现有 AGMP Domain Service 逐步注册为正式 XiaoYu Tools；
5. Minecraft Adapter 直接复用 0.1.84 Shared Runtime；
6. 之后再做 PTY/ConPTY 与跨重启 Session Agent。


---

## AI-Game-Manager-Panel 0.1.83

### Release Notes

<!-- historical source: docs/releases/0.1.83.md -->

0.1.83 是 0.1.82 之后的第二轮结构收敛版本，目标是不改变现有用户能力的前提下，把 AGMP 与 XiaoYu 的长期边界真正落实到物理代码结构。

## 本版完成

- XiaoYu Go 侧收敛为 `contract / control / runtime` 三组职责，Rust `xiaoyu-core` 保持唯一超级大脑实现。
- `internal/service` / `internal/core` / `internal/games/dst/service` 泛化中间层不再允许恢复。
- DST 的 dedicated、runtime、setup、workspace、logcenter 归回游戏域自身，减少跨目录追踪。
- 平台底层将一函数一目录的微包收拢为 `files / runtime / os / net / http / security / telemetry` 等稳定分组。
- `internal/app` 从 1397 行单文件拆为同包 `app_*.go` 业务分组，不增加额外层级。
- `internal/platform/runtime` 落地共享 Process/Terminal Session；DST 删除自己的平台进程实现并改为复用公共 Session。
- 前端删除仅含 `module.ts` 的规划占位目录；规划状态统一由 `configs/modules.json` 管理。
- Dashboard / Instances 命名与领域一致；通用规划页面使用统一 `ModulePlaceholderView`。
- 受控终端合并进 Bottom Panel，删除死代码式第二套终端页面；人工终端与 XiaoYu 共用身份、审批与审计边界。
- 当前产品标识统一使用 AGMP，内置超级大脑统一使用 XiaoYu / 小鱼；历史 Release 记录不改写。
- 新许可证 KeyID 使用 AGMP 命名，同时继续验证历史 KeyID；机器码派生协议保持冻结。

## 不属于本版“完成”的能力

本版是架构定型，不把 skeleton 功能伪装为完成。XiaoYu 的完整 Provider、Planner/Executor、Context、Memory、Workflow、恢复/重试，以及持久 PTY/会话恢复、多游戏完整适配等仍属于后续功能阶段。基础 Process/Terminal Session 已在本版落地。

## 冻结原则

从 0.1.83 起，新增代码优先落入现有 `xiaoyu / games / server / ops / deploy / system / platform` 边界；没有真实实现不得创建空包占位。需要拆文件时在同包使用统一短前缀，只有真正独立生命周期或复用边界才新增子包。

### Validation

<!-- historical source: docs/development/VALIDATION-0.1.83.md -->

## Result

0.1.83 structural/core optimization validation: **PASS with environment-limited items explicitly excluded**.

## Passed Gates

- Project Layout Gate
- Module Boundary Gate
- XiaoYu Core Gate
- Product Architecture Gate
- Distribution Boundary Gate
- Windows Helper / UTF-8 / CRLF / Frontend Import Gate
- Windows Installer Gate
- Updater Gate
- GitHub Safety Gate
- Release Key Gate (`--allow-unconfigured`, development-only)

## Go validation

- `go test`: 42 available `internal/...` packages passed; `internal/bridge/wails` excluded only because the source has no `go.sum` for the external Wails dependency.
- `go vet`: same package set passed.
- `go test -tags agmp_dev_license ./internal/system/license ./internal/app`: passed.
- `go build ./cmd/aigame-manager-web`: passed.
- `go build ./cmd/aigame-manager-license-admin`: passed.
- `GOOS=windows GOARCH=amd64 go test -c ./internal/platform/runtime`: passed.
- `GOOS=windows GOARCH=amd64 go test -c ./internal/games/dst/runtime`: passed.

## Shared terminal runtime regression

`internal/platform/runtime` now owns process launch, stdin/stdout/stderr streaming, PID, exit observation and terminate/kill behavior. DST uses this Session and its existing process lifecycle tests continue to pass.

## Environment-limited checks

- Cargo/Rust toolchain is not installed in this execution environment, so Rust workspace build/test/clippy were not run.
- pnpm is not installed, so the Vue production build was not run.
- Wails CLI is not installed.
- The repository still has no `go.sum`. An actual `go mod download github.com/wailsapp/wails/v2@v2.15.0` was attempted, but this environment cannot resolve `proxy.golang.org`; no checksum was fabricated.
- Full Windows Wails/Electron/Installer runtime acceptance must be run on the Windows development machine.

## Conclusion

The 0.1.83 architecture cleanup is internally consistent and Go-regression-safe in the available environment. It should be treated as a **candidate architecture baseline** until the missing Rust/frontend/Wails/Windows real-machine acceptance passes.

### Completion

<!-- historical source: docs/development/COMPLETION-0.1.83.md -->

> 本报告严格区分“架构优化完成度”和“产品功能完成度”。目录、接口或 Gate 存在不代表功能已经完成。

## 总体判断

**0.1.83 已完成本轮未完成架构优化，适合作为新的架构候选基线。** 这一版重点不是堆新功能，而是把 AGMP / XiaoYu 长期边界、公共 Process/Terminal Runtime、DST 归属、Application 编排和前端真实目录继续收干净。

架构已经接近冻结，但产品功能仍处于早中期：`configs/modules.json` 目前声明 33 个模块，其中 `implemented` 1 个、`active` 1 个、`partial` 6 个、`skeleton` 25 个。

## 1. XiaoYu 超级大脑边界

状态：**架构边界已定型，完整智能循环仍未完成。**

已完成：

- AGMP 是产品；小鱼 / XiaoYu / xiaoyu 是 AGMP 内置超级大脑和核心伙伴。
- Rust `xiaoyu-core` 是唯一 Brain；`xiaoyu-protocol` 提供 `xiaoyu.v1` 稳定边界。
- Go `internal/xiaoyu` 已从多散包收敛为 `contract / control / runtime` 三组职责。
- Go 侧禁止新增第二套 Brain / Planner / Memory；AI 必须通过 Tool / Domain API 调用系统能力。
- 受控人工终端与 XiaoYu 共用身份、审批和审计边界。

仍未完成：

- 真正可替换的模型 Provider 层。
- Planner / Executor 自动循环。
- Context Builder、会话/长期 Memory。
- 多步骤 Workflow、恢复、重试、取消和执行后验证闭环。
- 完整领域 Tool 覆盖和流式事件体系。

因此当前 XiaoYu 的准确定位仍是 **安全核心基础 + Tool/审批 Runtime**，还不是已经完全自主运行的超级大脑。

## 2. 项目结构优化

状态：**本轮目标已完成。**

- 禁止恢复 `internal/service`、`internal/core` 和 `internal/games/dst/service` 泛化中间层。
- 删除大量只有 `doc.go` 的后端规划空包；未实现能力改由 `configs/modules.json` 管理。
- 删除大量只有 `module.ts` / 包装占位页的前端目录；真实 `features` 现在主要是 `auth / dashboard / deployment / games / instances / logs / nodes / settings / terminal / users / xiaoyu`。
- `home -> dashboard`、`servers -> instances`，前后端术语更一致。
- 通用规划页面统一由 `ModulePlaceholderView` 渲染，不再维护第二套 FeaturePlaceholder。
- DST `service/*` 被收回 `dedicated / logcenter / runtime / setup / workspace` 自身，减少“游戏域里再套 Service 层”。
- 平台微包收拢为 `files / http / net / os / runtime / security / steam / telemetry` 等稳定组。
- 原 1397 行 `internal/app/app.go` 拆成同一 package 的 `app_*.go` 业务文件；最大业务文件目前约 526 行，没有新增额外架构层。

## 3. 通用 Process / Terminal Runtime

状态：**基础公共 Runtime 已真实落地，持久 PTY/会话体系待后续。**

0.1.83 新增真实 `internal/platform/runtime`：

- `Session` 通用进程会话；
- executable / arguments / working directory；
- stdin 命令写入；
- stdout + stderr 合流读取；
- PID；
- 真实进程退出等待；
- Terminate / Kill；
- Windows `CREATE_NO_WINDOW`；
- Line / ReadError / Exit Hook。

DST 已经实际改为复用该 Session，并删除 `internal/games/dst/runtime/process_windows.go` 与 `process_other.go`。DST 只保留 `c_shutdown()`、Ready 判定、Shard 状态和日志语义。

这意味着 Minecraft、Terraria 等未来只要满足“可执行程序 + 参数 + stdin/stdout”模型，就可以共用这一底座，而不是复制进程实现。

后续仍需：PTY、持久会话、会话恢复、WebSocket 实时 Session、多游戏 Console Adapter。

## 4. 前端终端

0.1.82 存在“真正能执行受控命令的 TerminalView 是死代码，而 Bottom Panel 是 disabled 占位”的结构/行为错位。

0.1.83 已改为唯一 `TerminalPanel.vue`，直接嵌入 Bottom Panel：

- 查询 XiaoYu Runtime 状态；
- 显示当前审批模式；
- 执行受控命令；
- Approve / Reject；
- `/terminal` 只负责打开 Bottom Panel。

以后禁止维护第二套独立 Terminal UI。

## 5. 真实模块状态

当前 33 个模块：

- `implemented`：1（logs）
- `active`：1（updater）
- `partial`：6（ai、deployment、terminal、environment、network、settings）
- `skeleton`：25

这说明**架构成熟度明显高于产品功能成熟度**。0.1.83 不是 RC，也不能把 skeleton 数量用目录占位“做少”。

## 6. 本轮验证结果

已真实通过：

- Project Layout Gate
- Module Boundary Gate
- XiaoYu Core Gate
- Product Architecture Gate
- Distribution Boundary Gate
- Windows Helper / UTF-8 / CRLF / Frontend Import Gate
- Windows Installer Gate
- Updater Gate
- GitHub Safety Gate
- Release Key Gate（development-only / allow-unconfigured）
- 42 个可用 `internal/...` Go package 的 `go test`
- 同范围 `go vet`
- `agmp_dev_license` 开发授权模式回归
- `cmd/aigame-manager-web` 构建
- `cmd/aigame-manager-license-admin` 构建
- `platform/runtime` Linux 实际测试
- `platform/runtime` 与 DST Runtime 的 Windows amd64 交叉编译测试
- DST Process 生命周期、Ready、命令、优雅停止、日志游标等原有测试继续通过

当前环境无法完成：

- Rust `cargo check/test/clippy`：执行环境没有 Cargo/Rust 工具链。
- Vue/pnpm build：执行环境没有 pnpm。
- Wails 根项目完整构建：源码仍没有 `go.sum`；本环境尝试下载 `github.com/wailsapp/wails/v2@v2.15.0` 时网络 DNS 被隔离，不能伪造校验文件。
- Windows Wails/Electron/Setup 真机启动验收：需 Windows 开发机执行。

## 7. 下一阶段应该做什么

架构不建议再做第三次大整理。0.1.83 通过 Windows 本机完整构建后，应优先进入 XiaoYu 功能阶段：Provider -> Planner/Executor -> Context/Memory -> Workflow -> Domain Tools；同时继续完成 PTY/持久 Console Session 和 Minecraft Adapter。


---

## AI-Game-Manager-Panel 0.1.82

### Release Notes

<!-- historical source: docs/releases/0.1.82.md -->

> AGMP / XiaoYu Naming & Architecture Polish

## 本版目标

0.1.82 不新增游戏玩法功能，继续完成 0.1.81 架构定型后的最后一轮命名和边界收口，让“产品”和“超级大脑”在代码层彻底分离。

## 已完成

- 产品缩写统一为 **AGMP**（AI Game Manager Panel），当前源码不再新增旧缩写命名。
- 内置超级大脑正式统一为 **小鱼 / XiaoYu / xiaoyu**。
- Rust Workspace 核心收敛为 `xiaoyu-core` 与 `xiaoyu-protocol` 两个主要 crate。
- 小鱼 wire protocol 统一为 `xiaoyu.v1`。
- Rust 内部 CLI/Runtime 主身份统一为 `xiaoyu`，不再把产品缩写混入小鱼核心 crate 名称。
- Go 小鱼边界继续固定在 `internal/xiaoyu`；Rust `xiaoyu-core` 是唯一 Brain，Go 不允许形成第二套 Planner/Brain。
- 产品侧 Go/TypeScript/PowerShell/配置中的主缩写统一迁为 `AGMP/agmp`。
- 许可证新 KeyID 使用 `AGMP-KID-*`；验证端保留历史 KeyID 的只读兼容，MachineCode 派生协议保持冻结，避免命名调整破坏已有设备授权。
- 后端继续冻结 `xiaoyu / games / server / ops / deploy / system / platform` 七个主要业务与基础领域，禁止恢复 `internal/service` 大平铺。
- 通用游戏终端继续固定为 `server/terminal -> platform/runtime/terminal -> platform/runtime/process`，游戏只保留自身差异。

## 兼容策略

0.1.82 是源码架构整理版本。对尚未正式发行的开发期内部标识优先直接统一命名；运行数据目录和正式用户数据边界不因本次命名整理迁移或删除。

## 验收重点

- Project Layout / Module Boundary / XiaoYu Core / Product Architecture Gate。
- Go test / go vet。
- Rust workspace build/test（需要本机 Cargo/Rust/MSVC 环境）。
- Frontend / Wails / Electron / Web / Installer 最终实机构建。

### Completion

<!-- historical source: docs/development/COMPLETION-0.1.82.md -->

> 本报告区分“架构完成度”和“真实功能完成度”。目录存在或页面存在不等于功能已经完成。

## 总体判断

0.1.82 已经完成一轮核心架构定型与命名收口，适合作为后续开发候选基线；但**还不是功能完成版，也不是最终 Release Candidate**。

## 1. 架构与工程边界

状态：**高完成度**。

已完成：

- 产品与超级大脑身份分离：AGMP = 产品，XiaoYu = 内置超级大脑。
- Rust `xiaoyu-core` 明确为唯一 Brain，Go `internal/xiaoyu` 定位为桥接/Tool/审批持久化等过渡能力，不允许发展第二套 Planner/Brain。
- Go 后端从平铺 Service/Core 收敛到 `xiaoyu / games / server / ops / deploy / system / platform` 主要领域。
- Module Gate 强制检查一级领域、禁止恢复 `internal/service` / `internal/core`、禁止旧产品缩写重新进入源码。
- Windows Helper / Installer / Updater / GitHub Safety Gate 均通过。

仍需继续：

- `internal/xiaoyu` 中部分 Go 审批/权限/Registry 仍属于过渡实现，最终权威决策应逐步收回 Rust XiaoYu Core。
- `internal/platform` 内部仍有若干小型子包，但已限制在明确基础设施组内；后续只在真实维护痛点出现时继续合并，避免为了目录漂亮再次大迁移。

## 2. XiaoYu 超级大脑

状态：**核心基础已落地，完整智能循环尚未完成**。

已落地：

- Rust Workspace：`xiaoyu-core` + `xiaoyu-protocol`。
- `xiaoyu.v1` JSON-RPC stdio 协议。
- Runtime status / session foundation / Tool Registry。
- Allow / Confirm / Deny 审批基础。
- 首批 `system.info`、`fs.list`、`fs.read`、受控 `process.run`。
- Go ↔ XiaoYu Runtime bridge。
- 用户审批状态、单次批准消费、审计边界基础。

关键未完成：

- 真正模型 Provider 接入与可替换 Provider 管理。
- Planner / Executor 自动循环。
- Context Builder、长期/会话 Memory。
- 多步骤 Workflow 恢复、重试、取消与结果验证闭环。
- Tool 并行、流式 reasoning/event、故障恢复。
- XiaoYu 对全部领域 Tool 的完整覆盖。

因此当前 XiaoYu 更准确的定位是：**安全可控的 Intelligence Core Foundation，而不是已经完成的自主超级大脑。**

## 3. 通用终端与服务器控制

状态：**Partial**。

架构边界已经固定：

```text
XiaoYu / 人工 UI
    -> server/terminal
    -> platform/runtime/terminal
    -> platform/runtime/process
    -> Game Server
```

但 DST 当前仍保留 `internal/games/dst/runtime` 的既有进程/stdin/stdout 实现。后续需要在不破坏 DST 行为的前提下逐步抽到公共 Runtime，再让 Minecraft/其他游戏复用。

## 4. 模块状态

`configs/modules.json` 当前共声明 **33 个模块**：

- `implemented`：1 个（日志中心）
- `active`：1 个（更新中心）
- `partial`：6 个（包括 XiaoYu、部署、终端、环境、网络、设置）
- `skeleton`：25 个

这说明目前项目的**架构成熟度明显高于功能成熟度**。如果仅按模块状态观察，产品仍处于早中期开发阶段，不能把目录齐全误认为功能接近完成。

## 5. 本轮真实验证

已通过：

- Project Layout Gate
- Module Boundary Gate
- XiaoYu Core Gate
- Product Architecture Gate
- Windows Helper / UTF-8 / CRLF / Frontend Import Gate
- Windows Installer Gate
- Updater Gate
- GitHub Safety Gate
- 85 个 Go package 的 `go test`（仅排除依赖 Wails 外部模块的 `internal/bridge/wails`）
- 同范围 `go vet`
- `agmp_dev_license` 开发许可证模式测试
- `cmd/aigame-manager-web` Go 构建
- `cmd/aigame-manager-license-admin` Go 构建
- 许可证命名迁移兼容回归测试

当前环境无法完成：

- Rust `cargo build/test/clippy`：当前执行环境没有 Cargo/Rust 工具链。
- Vue/pnpm build：当前执行环境没有 pnpm 与前端依赖。
- Wails 根项目完整构建：源码当前未带 `go.sum`，同时 `frontend/dist` 需要先由前端构建生成；当前隔离环境无法联网下载 Wails 模块。
- Windows 真机 Setup/Wails/Electron 启动验收：需要在 Windows 开发机完成。

## 6. 下一阶段优先级

1. Windows 本机完成 0.1.82 全构建验收并补齐 `go.sum`。
2. 把 XiaoYu 的 Provider + Planner/Executor Loop 做成真正的超级大脑主循环。
3. 完成公共 Terminal Runtime，先迁移 DST，再接 Minecraft。
4. 按领域 Tool Contract 给 XiaoYu 接入实例、备份、文件、Steam、环境、日志等真实能力。
5. 功能阶段继续遵守现有七领域边界，不再做总体骨架大手术。


---

## AI-Game-Manager-Panel 0.1.81

### Release Notes

<!-- historical source: docs/releases/0.1.81.md -->

主题：**XiaoYu Core Architecture Freeze / 小鱼核心架构定型**。

## 已完成

- AGMP 内置智能伙伴正式命名为 **小鱼（XiaoYu）**；AGMP 保持唯一产品身份，小鱼成为主智能控制入口和超级大脑身份。
- Rust Workspace 从 `agmp-agent-*` 命名迁为 `xiaoyu-*`，并将 Runtime + CLI 收敛进 `xiaoyu-core`，协议独立保留 `xiaoyu-protocol`。
- 协议升级为 `xiaoyu.v1`；生产内部二进制命名为 `AI-Game-Manager-XiaoYu.exe`，默认安装到 `internal/xiaoyu/`。
- 保留旧 `AGMP_AGENT_RUNTIME` 环境变量和旧二进制名称兼容读取，避免开发环境直接失效。
- Go 的 AI 相关代码从 `internal/agent` 归入 `internal/xiaoyu`，删除只有占位文档的 Planner/Audit/Conversation/Executor 空目录，避免假模块化。
- 取消 `internal/service` 大量服务平铺，将服务按业务域归入 `server / ops / deploy / system / games/dst`。
- `internal/core/game` 归入 `internal/games/common`；旧的纯占位 core 子目录移除。
- `internal/platform` 按 files/runtime/os/net/http/security/telemetry 聚合；Steam 底层作为较大独立平台能力保留。
- 前端 `features/ai` 更名为 `features/xiaoyu`，导航和工作台文案统一以“小鱼”为用户身份。
- 项目架构、模块边界、发行脚本与 Gate 开始同步到 0.1.81 新结构。

## 核心规则

- Rust `xiaoyu-core` 是唯一 Brain；Go `internal/xiaoyu` 只是桥接与领域 Tool 契约，禁止形成第二套大脑。
- 后端一级领域冻结为 `xiaoyu / games / server / ops / deploy / system / platform`。
- 小鱼和人工 UI 共用领域能力；小鱼不得用裸 Shell 绕过已有 Tool / Service / API。
- 强关联功能同域维护，禁止为了模块化不断制造一级目录和只有 `doc.go` 的空模块。

## 验证说明

本版属于架构整理候选，需要 Gate、Rust Test、Frontend Build、Go Test/Vet 和 Windows 实机发行构建全部通过后才能作为稳定基线。Go 源码包若缺 `go.sum` 或前端依赖缓存，需要在完整开发环境中先恢复依赖/生成 `frontend/dist` 再执行 Wails 全量测试。


---

## AI-Game-Manager-Panel 0.1.80

### Release Notes

<!-- historical source: docs/releases/0.1.80.md -->

## 主题

**Module Ownership + AI Approval Foundation**：把 小鱼已有的三种权限模式从界面占位补成后端可信审批基础，同时冻结“一个模块一个职责、AI 只通过模块能力接口调用系统”的开发规则。

## 新增

- 固定三种 AI 权限名称：`请求批准`、`帮我批准`、`完全访问权限`。
- 新增持久审批状态：模式、待审批队列、批准/拒绝、请求指纹绑定、单次消费。
- 新增 Web/Wails 审批 API：读取状态、切换模式、读取待批准、批准/取消。
- 小鱼显示真实后端权限模式和待批准请求；用户未决定时请求保持等待。
- 新增 `docs/development/MODULES.md` 与 `check-modules.mjs` Module Gate。

## 安全

- 权限模式以后端持久状态为唯一可信来源；客户端不能通过请求字段伪造 Full Access。
- `process.run` 默认 `agent:false`，不能成为 AI 绕过文件、节点、设置等模块 API 的万能 Shell。
- 审批文件只保存安全摘要和不可逆请求指纹，不保存完整命令、Token、API Key 或密码。
- 完全访问权限只减少人工审批，不关闭身份、RBAC、参数、路径、作用域、幂等、备份/回滚、审计和结果验证。

## 架构规则

- 一个业务模块一个明确归属；文件、设置、授权、节点、备份、日志、终端等各管各的。
- UI 可以组合模块，但页面位置不改变业务归属。例如许可证 UI 可以在设置中心，许可证业务仍归 `internal/system/license`。
- AI 只能通过 Tool -> Service/API -> Core/Platform 调用系统能力。
- 小功能不强制过度拆分；多文件使用短而一致的前缀，便于人工和 AI 搜索。
- 新增源码命名优先短业务词；同一功能拆文件时保持统一短前缀，推荐 `xxx.go` / `xxx_rule.go` / `xxx_win.go` / `xxx_test.go`，有明确顺序时才用 `xxx_one.go` / `xxx_two.go`。
- 当前版本只做轻量骨架纠偏，不进行大规模目录迁移。

## 已验证

- Approval store 单测：pending 等待、批准单次消费、拒绝阻断、请求指纹绑定、原始命令不落盘。
- Permission policy 单测：三种模式语义独立。
- Config loader 单测：三个固定名称与三套策略表校验。
- 小鱼核心 Go bridge 单测。
- Module / Product Architecture / Project Layout / GitHub Safety 等静态 Gate。

## 尚未完成

- 模型 Provider、Planner、完整 Agent Loop、自动恢复/继续执行仍属于下一阶段。
- 当前环境没有 Windows Rust/MSVC 与前端 pnpm 依赖，完整 Windows Release、Cargo、Vue Production Build 仍需 Windows/CI 实机验证。

### Historical Development Prompt

<!-- historical source: docs/prompts/0.1.80-module-approval.md -->

本版需求由项目所有者确定：

1. AGMP 内置 AI 是系统工具最高优先级的大脑，主要服务于开服和运维目标，聊天为辅助能力。
2. AI 权限名称固定为“请求批准 / 帮我批准 / 完全访问权限”，不得自行改名。
3. 请求批准：副作用操作先问用户；未回答持续等待。
4. 帮我批准：普通低风险操作自动完成；敏感/高风险操作等待用户确认或取消。
5. 完全访问权限：AI 获得最高 AGMP Tool 权限并可自主完成，但任何硬安全校验都不能关闭。
6. 一个业务模块一个明确目录归属，各模块通过 API / Tool / Service 连接，禁止跨模块乱放业务逻辑。
7. 模块化不能过度拆碎；强关联多文件允许同前缀或 one/two 形式，让人工和 AI 一眼知道是一组。
8. 命名短、清楚、可搜索；当前只轻量优化骨架，不大规模重排总体布局。
9. 优先级：核心能力 -> 稳定 -> 性能 -> 安全 -> 省心 -> 后续目录美化。


---

## AI-Game-Manager-Panel 0.1.79

### Release Notes

<!-- historical source: docs/releases/0.1.79.md -->

主题：**Unified AI-First Architecture / 开发与生产边界纠偏**。

## 架构冻结

- 新增强制规则文档 `docs/development/PROJECT-RULES.md`，与根目录 `AGENTS.md` 共同冻结产品架构和开发/生产边界。

- AI Game Manager Panel 确认为唯一产品主体。
- AI Agent 确认为 AGMP 内置核心大脑，AI 为主控制入口，手动页面/终端为辅助、确认、覆盖和兜底。
- Web / Wails / Electron / Rust Native 定义为同一产品的不同运行端；Rust Native 当前仍是规划目标，不伪装为已完成。
- AI 与人工必须调用同一能力层，禁止为不同端或不同交互方式复制服务器业务逻辑。
- 当前阶段不大规模调整源码根目录，优先完成稳定、可用、高性能、安全、省心的核心能力。

## 开发 / 生产边界

- `AI-Game-Manager-Panel.bat` 仅用于源码开发和发行构建。
- 基础开发环境不强制安装 Rust/MSVC；只有开发 AI Core Rust 实现或源码 Release 时按需准备。
- 普通用户生产环境只使用已编译好的完整 AGMP，不现场编译、不安装开发工具链。

## 打包修复

- Wails Setup/Portable 将 Rust AI Runtime 放到 AGMP 内部组件目录，不创建独立快捷方式。
- Wails `build/release` 不再发布 standalone Agent 二进制；Agent 原始构建物保留在 `build/work`。
- Go AI Runtime 发现逻辑优先查找完整产品内部路径，并保留旧版路径兼容。
- Electron 继续把 Go/AI Runtime 作为 `resources` 内部实现，不作为独立用户产品。

## Gate

新增 `scripts/common/check-product-architecture.mjs`：强制检查单一产品、内置 AI Core、AI-first/Manual-assist、多端同产品、开发/生产隔离和禁止 standalone Agent 发布。

## 后续优先级

继续核心能力：Model Provider → Planner/Agent Loop → Tool 执行/验证 → PTY/会话 → 游戏能力接入与安全加固。

### Historical Development Prompt

<!-- historical source: docs/prompts/0.1.79-unified-ai-first-architecture.md -->

## 用户确认的产品定义

- AI Game Manager Panel 是唯一工具本体。
- AI Agent 内置在 AGMP 中，是系统的核心大脑和默认主控制入口。
- 人工操作是辅助控制：用于确认、精细控制、覆盖、故障排查与兜底。
- Web、Wails、Electron、Rust Native 是同一 AGMP 的不同构建/运行端，不是拆开的多个产品。
- 所有端应共享 Agent、业务能力、权限、审计与数据契约；差异只允许存在于平台/桌面 Adapter 与打包层。
- 内部实现可以使用 Go、Rust、子进程、IPC，但不得因此让普通用户感知为独立 Agent/Core 产品。

## 开发环境 / 生产环境

- 开发环境用于源码开发，可安装 Go、Node、pnpm；Rust/Cargo/MSVC 只在开发 AI Core Rust 实现或从源码制作发行包时按需准备。
- 发布构建机/CI 负责提前编译所有内部组件并打包完整 AGMP。
- 生产环境只运行预编译好的完整产品；普通用户不得被要求安装 Rust、Cargo、MSVC、Go、Node、pnpm、Wails CLI 或 Visual Studio Build Tools。
- `AI-Game-Manager-Panel.bat` 仅属于源码开发/发行构建，不是普通用户生产启动入口。

## 当前阶段约束

- 暂不大规模调整总体项目布局。
- 当前优先完成核心能力，顺序强调：稳定、能用、性能好、能跑、安全、省心。
- 先完成 Agent Loop、Provider、Tool 能力、审批、安全、终端/PTY、游戏管理能力接入，再逐步优化目录和架构外观。

## 0.1.79 本轮修复目标

1. 把上述架构写入 `AGENTS.md`、`docs/development/PROJECT-RULES.md` 和 `docs/PROJECT-ARCHITECTURE.md`，作为强制项目规则。
2. 明确开发环境、发行构建环境和普通用户生产环境边界。
3. 普通用户发行物只能是完整 AGMP 运行端，禁止 standalone Agent 产品。
4. Wails Setup/Portable 把 小鱼核心 Runtime 作为 AGMP 内部 AI Core 组件，不创建用户入口。
5. `build/release` 不再单独发布 Agent 二进制；原始内部二进制留在 `build/work`。
6. 增加产品架构 Gate，防止后续开发再次把 Agent、Core、桌面端拆成独立产品。
7. 不在本轮大规模迁移源码目录。


---

## AI-Game-Manager-Panel 0.1.78

### Release Notes

<!-- historical source: docs/releases/0.1.78.md -->

## Build Architecture Correction

0.1.78 修正 Rust Agent 引入后的开发/发行边界。Rust 仍作为 AI Agent Runtime 技术栈，但 Rust 编译环境不属于普通用户依赖。

### 开发者 / 发布者

- 基础开发初始化：Go、Node.js、pnpm、Frontend、Go Modules。
- Rust Agent 开发：按需准备 Rust/Cargo + Windows MSVC/SDK。
- 本机源码 Release：需要完整 Rust/Cargo/MSVC，因为发布过程会从源码编译 `AI-Game-Manager-XiaoYu.exe`。

### 普通用户

- 只下载并运行 `AI-Game-Manager-Panel-0.1.78-Windows-x64-Setup.exe`。
- Setup 内已经包含预编译 `AI-Game-Manager-XiaoYu.exe`。
- 不需要 Rust、Cargo、Rustup、MSVC、Visual Studio Build Tools、Go、Node.js 或 pnpm。
- 安装过程不会下载或现场编译 Agent Runtime。

### Gate

新增 Distribution Boundary Gate：

- 要求用户发行配置 `runtimeCompilation=false`。
- 要求 Setup 携带预编译 Agent。
- 禁止 Rust 源码/Cargo/Rustup/MSVC Build Tools/开发脚本进入用户安装包。
- 要求基础开发初始化不调用 Rust/MSVC 安装函数。


---

## AI-Game-Manager-Panel 0.1.77

### Release Notes

<!-- historical source: docs/releases/0.1.77.md -->

## Windows Rust/MSVC 精简安装与存储控制

0.1.77 根据 Windows 实机验收继续收紧 Rust Agent 工具链准备流程。

- 缺少 MSVC 时先解释安装内容、官方磁盘范围与 AGMP 空间建议。
- 新装按 Rust 官方最小前置只选择 `Microsoft.VisualStudio.Component.VC.Tools.x86.x64` 与 `Microsoft.VisualStudio.Component.Windows11SDK.22621`。
- 不安装完整 `Microsoft.VisualStudio.Workload.VCTools`，并移除 `--includeRecommended`，避免 CMake、vcpkg、ASAN、ATL/MFC、测试适配器、ARM/ARM64 等非必要组件。
- 用户可选择默认位置或自定义根目录；自定义时分离 BuildTools、PackageCache、Shared、Installer 缓存。
- 安装前显示目标盘剩余空间；不足 8 GB 会再次确认。
- 已存在 Visual Studio / Build Tools 时只补必要组件，遵守其既有全局路径配置，不重复安装。
- 继续校验微软 Authenticode 签名并复用已缓存 bootstrapper。

> Microsoft 对 Build Tools 2022 的官方磁盘需求给的是 2.3 GB 到 60 GB 的范围，实际大小取决于组件和版本；AGMP 不伪造一个固定下载数值，而是只选择最小组件集合并在安装前清楚显示空间策略。


---

## AI-Game-Manager-Panel 0.1.76

### Release Notes

<!-- historical source: docs/releases/0.1.76.md -->

## Windows Rust Bootstrap Hotfix

0.1.75 Windows 实机验证发现两个初始化问题：源码打包遗漏 `runtime/README.md`，以及缺少 MSVC `link.exe` 时自动准备流程错误地依赖 winget。

本版本修复：

- 恢复并强制保留 `runtime/README.md` 项目骨架。
- 缺少 MSVC C++ 工具链时始终先询问用户 Y/N，不再因为 winget 缺失直接失败。
- 使用微软官方 Visual Studio 2022 Build Tools bootstrapper 作为无 winget 安装路径。
- 安装器缓存在 `%LOCALAPPDATA%\AI-Game-Manager-Panel\DevTools\Installers\vs_BuildTools-2022.exe`。
- 缓存文件存在且通过 Microsoft Authenticode 签名检查时直接复用，不重复下载。
- 若 Build Tools/Visual Studio 已安装，优先自动加载开发者环境；缺少 C++ workload 时只补齐所需组件。
- Wails/Electron 正式 Release 继续在耗时构建之前执行 Rust/MSVC 预检。

小鱼核心 Runtime 功能本身仍沿用 0.1.74 Foundation，本版本只修复 Windows 工具链闭环。


---

## AI-Game-Manager-Panel 0.1.75

### Release Notes

<!-- historical source: docs/releases/0.1.75.md -->

## Windows Rust Toolchain Hotfix

0.1.74 首次在 Windows 正式 Release 中编译 小鱼核心 Runtime。实机验收发现 Rustup/Cargo 已安装时，若系统缺少 MSVC `link.exe`，Cargo 会在链接阶段失败。

0.1.75 将 Microsoft Visual C++ Build Tools 纳入 Windows 初始化和 Release 预检：

- 检测 `cargo` / `rustc`。
- 检测 `link.exe`。
- 通过 `vswhere.exe` 寻找现有 Visual Studio / Build Tools。
- 自动载入 `VsDevCmd.bat` 或 `vcvars64.bat` 到当前 PowerShell 进程。
- 未安装时，经用户确认后使用 winget 安装 Visual Studio 2022 Build Tools + `Microsoft.VisualStudio.Workload.VCTools`。
- Wails / Electron Release 在 Frontend 和正式打包前先做 Rust/MSVC 预检，避免最后阶段才失败。

小鱼核心 Runtime 功能本身仍沿用 0.1.74 Foundation。


---

## AI-Game-Manager-Panel 0.1.74

### Release Notes

<!-- historical source: docs/releases/0.1.74.md -->

主题：**小鱼核心 Runtime Foundation / Agent-first**。

## 已实现

- 新增 Rust workspace：protocol / runtime / CLI 三层。
- 新增 `xiaoyu.v1` JSON-RPC stdio 协议。
- 新增 Tool Registry 与 read / operate / modify / destructive / system 风险模型。
- 新增 ask / risk / full 审批模式与 allow / confirm / deny 决策。
- 首批工具：system.info、fs.list、fs.read、process.run。
- 新增 Rust CLI：doctor、tools、tool、exec、rpc。
- Go Core 新增 `internal/xiaoyu/runtime` bridge与通用 `AgentCallTool`，并接入许可证 `ai.workbench` Feature Gate 与操作审计。
- Wails / Web / Electron 接入同一 小鱼核心 Runtime；Wails Setup/Portable 携带 `AI-Game-Manager-XiaoYu.exe`。
- 小鱼调整为 Agent-first，手动页面作为 fallback。
- 手动终端改为通过 小鱼核心 Runtime 执行受控一次性命令。
- Windows 开发助手增加 Rustup/Cargo 检测、安装提示、构建和 CLI 调试入口。

## 本阶段不宣称完成

- Model Provider 尚未接入自动 Agent Loop。
- 暂无持续流式 Planner/Executor。
- 终端当前为一次性 command execution，不是持久 PTY。
- Tauri/Rust Desktop Shell 尚未替代 Wails；Wails 仍是 Windows 官方主桌面端。

## 设计参考

参考 Codex、OpenCode、Claude Code、ChatGPT/Codex App 的 Agent-first 交互与执行分层思想；没有复制第三方源码实现。


---

## AI-Game-Manager-Panel 0.1.73

### Release Notes

<!-- historical source: docs/releases/0.1.73.md -->

## Windows 自动更新与升级安装

- 修复 Inno Setup `Folder Exists` / `Folder Does Not Exist` 英文提示，统一为简体中文。
- 固定 AppId 继续作为 Windows 同应用升级标识；识别已有 EXE 版本并在欢迎页显示“从旧版本升级到 0.1.73”。
- 新增 `configs/update.json`，稳定通道默认检查 `yubboo/AI-Game-Manager-Panel` GitHub Releases。
- 新增 Go `internal/deploy/updater`：检查 latest release、数字版本比较、下载 Windows Setup、SHA256 完整性校验、启动升级安装器。
- 设置中心新增“更新与升级”，支持手动检查、查看 Release Notes、安装包大小和 SHA256 状态。
- 正式 Wails Release 生成 `Setup.exe.sha256` companion asset，供客户端在 GitHub Release digest 不可用时校验。
- 启动后低频检查稳定更新；发现新版本时 Topbar 显示提示，不自动下载、不静默安装，必须由用户点击。
- 升级只覆盖程序文件与静态配置；`runtime` 中的账号、许可证、实例、备份、日志和用户设置不主动删除。

## 首次迁移说明

0.1.72 本身没有自动更新客户端，所以第一次从 0.1.72 升级到 0.1.73 仍需手动运行 0.1.73 Setup。安装 0.1.73 后，后续 0.1.74+ 才进入应用内更新闭环。


---

## AI-Game-Manager-Panel 0.1.72

### Release Notes

<!-- historical source: docs/releases/0.1.72.md -->

## Windows Installer UX

本版本以 0.1.71 正式 Release / BFLC2 验收通过为基础，重点优化 Windows 官方 Wails 安装器。

### 完成

- Windows 官方主发行明确为 Wails + Inno Setup；Electron 标记为兼容发行。
- 安装器改为项目内置简体中文文案，不再依赖额外 `ChineseSimplified.isl`。
- 新增 `EULA-zh-CN.txt` 软件许可及服务协议。
- 使用 Inno `LicenseFile` 强制协议页：未选择“我接受本协议”不能继续安装。
- 恢复欢迎页与开始菜单选择页，保留安装目录和桌面快捷方式选择。
- 增加安装日志、运行中旧版本关闭处理、升级时复用原安装目录和任务设置。
- 安装完成后可直接启动应用。
- 卸载增加中文提示，明确运行数据/日志/备份/实例信息默认保留，降低误删风险。
- Wails Setup 文件改为带版本号的正式发行命名，并在构建完成后输出安装器大小。

### 安装范围

当前继续使用 `%LOCALAPPDATA%\Programs\AI-Game-Manager-Panel` 当前用户安装，避免在运行数据仍位于应用根目录时引入 Program Files 写权限问题。


---

## AI-Game-Manager-Panel 0.1.71

### Release Notes

<!-- historical source: docs/releases/0.1.71.md -->

## Release / Feature Gate Hardening

本版本用于在 0.1.70 BFLC2 激活闭环实机验证通过后，补齐正式发行前的授权边界与重复验收能力。

### 完成

- Windows 正式 Release：若新源码目录尚未配置 Active 发行公钥，自动尝试从 `%LOCALAPPDATA%\AI-Game-Manager-Panel\ReleaseKeys` 同步公开密钥，然后再执行严格 Release Key Gate。
- 新增 `license.Service.RequireFeature()` 与 `Application.RequireLicenseFeature()`，后端高级能力以后必须使用这一入口，而不能只依赖 Vue 路由隐藏。
- 新增 Standard / Pro Feature Entitlement 矩阵测试。
- 新增未授权、过期、解绑后的 Feature Gate 拒绝测试。
- 保留错误机器码 / 错误 BFID / 篡改签名 / 伪造 IssuerKeyID / 重启持久化回归测试。
- 许可证发行控制台新增“一键许可证完整验收”。

### 安全边界

- Active 公钥可进入 GitHub；发行私钥仍只允许保存在源码仓库外。
- Development Build Tag `agmp_dev_license` 只允许开发入口使用，正式 Release 不包含。
- 前端 Feature Gate 只负责用户体验；真正的授权安全边界必须位于 Go Core。


---

## AI-Game-Manager-Panel 0.1.70

### Release Notes

<!-- historical source: docs/releases/0.1.70.md -->

## BFLC2 Activation Closure

这一版本把 0.1.69 的“发行密钥生命周期”继续推进到完整离线授权闭环：**签发 → 回验 → 导入 → 原子持久化 → Core 重启后继续有效**。

### 签发后自动回验

- `issue-bflc2.ps1` 在生成 `.bflc` 后，立即调用公开密钥环重新验证证书。
- 回验同时检查 Ed25519 签名、机器码、安装 ID、有效期和 IssuerKeyID。
- 回验失败时明确停止交付，避免把错误证书发给用户。

### IssuerKeyID 不再只是显示字段

- BFLC2 声明的 `issuerKeyId` 必须与真正完成 Ed25519 验签的公钥 KeyID 一致。
- 历史 BFLC2 若没有 `issuerKeyId`，验证成功后会补齐实际签发 KeyID，保持兼容。
- 新增伪造 IssuerKeyID 回归测试。

### 激活记录安全持久化

- `activation.json` 改为同目录临时文件写入、`fsync`、原子替换。
- 避免程序在写盘过程中异常退出造成半截 JSON。
- 新增 Core Service 重建后的许可证持久化回归测试。

### 授权页支持直接导入 .bflc

- 可以选择发行工具生成的 `.bflc` 文件，也可以继续手动粘贴完整 BFLC2。
- 前端只读取证书文本，不接触发行私钥。
- 导入成功后提示完全退出并重启进行持久化验证。

### 新源码目录恢复 Active 发行公钥

发行私钥一直保存在仓库外，因此解压新的源码版本时不会自动出现在 `vendor_public_keys.json`。

0.1.70 新增：

```text
AGMP-License-Admin.bat
  4. 同步本机发行公钥
```

工具会扫描：

```text
%LOCALAPPDATA%\AI-Game-Manager-Panel\ReleaseKeys\
```

读取 `key-metadata.json` / 公钥信息，把最新有效发行公钥恢复到当前源码的公开密钥环。**不会把私钥复制进源码。**

### 独立 BFLC2 验证

新增发行控制台第 5 项，可以针对 BFM/BFID 和 `.bflc` 文件独立验证许可证，方便正式交付前复核。

### BFID 绑定口径明确

BFID 表示具体安装实例，不是整台电脑的永久 ID。源码开发目录、Portable、正式安装目录如果使用不同运行根目录，可能拥有不同 BFID。**签发 BFLC2 时必须从真正准备激活的那一份 Release 程序中复制 BFM/BFID。**


---

## AI-Game-Manager-Panel 0.1.69

### Release Notes

<!-- historical source: docs/releases/0.1.69.md -->

## Offline License Issuer Key Lifecycle

### 发行密钥不再依赖丢失的旧私钥

- 0.1.68 及更早版本的旧发行公钥保留为 `legacy`，仅用于验证历史许可证。
- 新增公开发行公钥环 `internal/system/license/vendor_public_keys.json`。
- 正式 0.1.69+ Release 必须存在一个 `active` 发行公钥；否则 Release Key Gate 会拒绝打包。
- 源码 Development 模式不受影响，继续使用 `agmp_dev_license`。

### 私钥只在项目所有者本机生成

- 新增 `scripts/tools/license/AGMP-License-Admin.bat`。
- 默认私钥目录：`%LOCALAPPDATA%\AI-Game-Manager-Panel\ReleaseKeys\...`。
- `keygen` 强制要求 `--output-dir`，并阻止把发行私钥生成进源码仓库。
- Windows 工具会尝试进一步收紧私钥 NTFS ACL。

### BFLC2 签发安全

- BFLC2 Payload 新增 `issuerKeyId`。
- 签发前必须确认所用私钥与源码中 `activeKeyId` 一致。
- 证书默认保存到仓库外的 `IssuedLicenses/`。
- 客户端设置页显示 Active 发行 KeyID、公钥 SHA-256 指纹、可信公钥数量和当前许可证签发 KeyID。

### 平滑密钥轮换

- 客户端验证 BFLC2 时会尝试整个可信公钥环。
- 新 key 成为 `active` 后，旧 active 自动变成 `retired`。
- `legacy/retired` 公钥仍可验证历史 BFLC2，避免正常轮换时让旧用户授权立即失效。

### Release Gate

- Windows Wails / Electron 正式 Release 新增 Release Key Gate。
- Linux Desktop / Server 正式 Release 同样强制检查 active 发行 key。
- GitHub Actions 校验公开密钥环结构，但允许尚未持有私钥的源码贡献者进行普通 CI。


---

## AI-Game-Manager-Panel 0.1.68

### Release Notes

<!-- historical source: docs/releases/0.1.68.md -->

## Windows Console Flash + License Snapshot Stabilization

### 黑色终端闪窗修复

- 根因：许可证服务每次 `Status()` 都通过 `reg.exe query HKLM\SOFTWARE\Microsoft\Cryptography /v MachineGuid` 获取机器标识。Wails GUI 进程启动 `reg.exe` 时，Windows 可能短暂创建控制台窗口。
- 修复：Windows 改为直接调用 Registry API，不再启动 `reg.exe`、`cmd.exe` 或 PowerShell。
- Electron Core sidecar 继续保持 `windowsHide: true`；DST Server 进程继续使用既有隐藏窗口策略。

### 机器码 / 安装 ID 稳定性

- MachineCode 在 Go Core 生命周期内只计算一次。
- InstallID 首次读取/创建后缓存；页面刷新不会重新生成。
- BFID 仍持久化在 `runtime/data/license/install.id`，升级继续兼容旧 `device.id`。

### 前端不再“读取中…”重绘

- `LicenseSection` 与 `SecurityView` 统一读取 Pinia `app.license`。
- 进入设置页时，如果全局已有许可证快照，立即显示已有 MachineCode / InstallID，不再重复请求并清空本地状态。
- 手动刷新保留旧值，后台更新成功后原位替换。

### CDK / BFLC2 术语纠正

- 短 CDK：未来在线 License Server 兑换码，当前未开放。
- BFLC2：当前正式 Release 可用的离线许可证证书。
- 源码开发模式：使用 `agmp_dev_license`，无需真实 CDK/BFLC2。

### 0.1.67 回归修复

- 修复品牌迁移误生成的非法 TypeScript 标识符 `AI Game Manager PanelSettings`，统一为 `AGMPSettings`。


---

## AI-Game-Manager-Panel 0.1.67

### Release Notes

<!-- historical source: docs/releases/0.1.67.md -->

## Project Rename + Release Pipeline Cleanup

### 全局品牌迁移

- 全局产品名改为 **AI游戏管理器面板 / AI Game Manager Panel**。
- GitHub 目标仓库改为 `yubboo/AI-Game-Manager-Panel`。
- “篝火 / Bonfire”只保留为 Don't Starve Together 专属模块/工作台子品牌；许可证协议中的历史 `BF*` 标识暂时保留兼容性，不再作为 UI 产品品牌。

### Windows 发布修复

- 修复 Electron Builder 已完成 NSIS + Portable 后，最后 Release 晋升阶段因旧目标文件被其他进程占用而失败。
- Wails 与 Electron 都改为逐文件、带重试的安全发布。
- 如果旧目标持续锁定，新构建会保留为 `-Rebuild-YYYYMMDD-HHmmss`，不会让整次 Release 失败。

### Build 目录收口

- `build/release/`：唯一最终发行区。
- `build/work/`：唯一中间工作区。
- `build/bin/` 不再作为正式产物目录；仅允许 Wails CLI 临时创建，之后自动清理。

### Electron 包体积优化

- ASAR 开启；
- electron-builder `maximum` 压缩；
- Electron Runtime 语言包仅保留 `zh-CN` / `en-US`；
- Go Core 使用 `-trimpath -ldflags "-s -w"`；
- 不使用 UPX 等容易引发杀软误报或签名问题的激进压缩。

### 授权 / GitHub Safety

- 延续 0.1.66 的开发许可证模式，Build Tag 统一为 `agmp_dev_license`。
- 正式 Release 默认关闭开发许可证模式。
- GitHub Safety Gate 与敏感文件忽略规则继续生效。


---

## AI-Game-Manager-Panel 0.1.66

### Release Notes

<!-- historical source: docs/releases/0.1.66.md -->

## GitHub Safety + Development License Mode

### 完成

- 面向公开仓库 `yubboo/ai-server-helper` 加固根 `.gitignore`。
- 明确禁止提交 License 私钥、真实 CDK/BFLC、activation/accounts/install/device 状态、管理员安全密钥导出、DST Token、运行数据、日志、依赖缓存和构建产物。
- 新增 `scripts/common/check-github-safety.mjs` GitHub Safety Gate。
- Windows 菜单 3 快速/完整检查和 Core Release Gate 自动执行 GitHub Safety Gate。
- Linux Build/Server Build 也在结构检查前执行 GitHub Safety Gate。
- 新增 `docs/development/GITHUB-SAFETY.md` 作为长期提交规范。
- 新增 `.github/workflows/safety.yml`，Push/PR 自动执行 GitHub Safety Gate、结构门禁、Go Test/Vet 与开发许可证 Build Tag 测试。
- 新增 `bonfire_dev_license` Go Build Tag：源码开发状态返回 `DEVELOPMENT`，放行全部 Feature Entitlement，避免开发阶段必须持有真实离线 CDK。
- Wails Dev、Web Dev、Electron Dev 自动启用开发许可证 Build Tag。
- 正式 Wails/Electron/Web Release 不带开发 Tag，仍执行真实许可证校验。
- 设置中心授权页新增“源码开发模式”状态展示，并在开发模式下隐藏无必要的激活输入区。
- 修正 0.1.65 骨架收口遗留：Linux `.deb` 打包第三方许可证路径改为 `distribution/licenses`。

### 安全边界

- 许可证**公钥**可以进入公开源码；发行**私钥**绝不能进入 Git、源码包、安装包或日志。
- 不提供/提交“万能离线 CDK”。开发调试通过 Build Tag 解决，正式用户许可证仍由发行端私钥签发。
- 如果发行私钥曾被上传到公开 Git 历史，应视为已泄露并轮换，而不是只删除最新 Commit 中的文件。

### 待本机验证

- Windows `AI-Game-Manager-Panel.bat -> 3 -> 快速检查 / 完整检查`。
- Wails Dev：确认设置中心显示“源码开发模式”，AI/计划任务/插件/远程节点可进入。
- Electron Dev：确认 Go Core 使用开发 Build Tag。
- Web Dev：确认开发构建可正常进入授权 Feature。
- Wails/Electron 正式 Release：确认不出现 DEVELOPMENT 状态，仍按正式许可证执行 Feature Gate。


---

## AI-Game-Manager-Panel 0.1.65

### Release Notes

<!-- historical source: docs/releases/0.1.65.md -->

## 新增

- 新增统一 `runtime/` 运行数据根目录说明；新项目默认将 data/log/backups/instances/temp/exports/plugins/cache 收口到该目录。
- 新增 `distribution/` 分发资源边界，集中 Windows/Linux 安装器、Docker 与第三方许可证。
- 新增 0.1.65 项目骨架 Gate，防止已移除的旧源码目录重新回到根目录。

## 优化

- 根目录从约 20 个一级目录收口为 9 个稳定一级目录：`cmd/configs/desktop/distribution/docs/frontend/internal/runtime/scripts`。
- 开发者许可证工具从根 `tools/` 迁到 `scripts/tools/license/`。
- 历史更新记录从根 `update-log/` 迁到 `docs/releases/`。
- Windows Installer、Linux Server 打包、Linux `.deb`、Docker 与 Wails Portable 的源码路径同步调整。
- Windows 新安装默认创建 `runtime/*`，不再创建八个并列运行目录。
- Docker 新容器默认创建 `runtime/*`。

## 兼容

- 不改变 `AI-Game-Manager-Panel.bat -> scripts/windows/AIGameManagerPanel.ps1`。
- 不改变 Wails / Electron / Web 的业务调用链。
- 已有安装保留旧版 `configs/paths.json` 时，仍继续读取根目录 `data/`、`log/` 等旧路径；0.1.65 不自动搬移用户数据。
- 安装后的第三方许可仍输出为 `<InstallRoot>/licenses/`，仅源码位置迁入 `distribution/licenses/`。

## 架构调整

```text
源码根目录
├─ cmd/
├─ configs/
├─ desktop/
├─ distribution/
├─ docs/
├─ frontend/
├─ internal/
├─ runtime/
└─ scripts/
```

## 测试

- 项目骨架 Gate：通过。
- Windows Helper / UTF-8 / 菜单 / Frontend Import Gate：通过。
- `scripts/common/*.mjs` Node 语法检查、JSON 解析、Linux Shell `bash -n`：通过。
- 92 个 `internal/...` 与 `cmd/...` Go 包（排除依赖外部 Wails 模块的 `internal/bridge/wails`）执行 Test / Vet：通过。
- 完整 `go test ./...`、Frontend / Electron 完整构建受当前离线验证环境限制，详见外部验证报告。

## 已知问题

- Windows Inno/Wails/Electron 正式安装包仍需要 Windows 实机完成最终 Release 验收。
- 已有旧路径数据不会自动迁入 `runtime/`；这是为了避免静默搬动服务器数据。

## 下一阶段

- 在 0.1.65 骨架冻结后，继续管理员首次注册可跳过、安全密钥生成/保存/下载与登录密钥验证流程的后续完善。

### Historical Development Prompt

<!-- historical source: docs/prompts/0.1.65-project-structure-cleanup.md -->

> 基线：AI Game Manager Panel 0.1.64。目标版本：AI Game Manager Panel 0.1.65。

## 目标

只整理项目结构、运行目录、分发资源与开发工具路径，不新增业务功能，不改变 Wails / Electron / Web 的业务能力与入口行为。

## 硬约束

1. 根目录只保留稳定工程边界；禁止继续新增临时一级目录。
2. 运行数据默认统一进入 `runtime/`，不再在源码根目录并列维护 `data/`、`log/`、`backups/`、`instances/`、`temp/`、`exports/`、`plugins/`、`cache/` 八个占位目录。
3. 已有旧版 `configs/paths.json` 继续有效，不强制迁移现有用户数据。
4. Windows / Linux 安装器、Docker 与第三方许可证源码统一进入 `distribution/`。
5. 开发者许可证工具进入 `scripts/tools/`。
6. 历史更新记录统一进入 `docs/releases/`。
7. `AI Game Manager Panel.bat -> scripts/windows/AI Game Manager Panel.ps1` 入口保持不变。
8. Wails / Electron / Web 继续共用一套 Go Core 和 Vue UI。
9. Windows Release、Linux Server Release、Docker 路径必须同步调整，禁止只移动文件不改构建脚本。
10. 项目骨架 Gate 必须能阻止旧源码目录重新回到根目录。

## 目标根目录

```text
AI Game Manager Panel/
├─ cmd/
├─ configs/
├─ desktop/
├─ distribution/
├─ docs/
├─ frontend/
├─ internal/
├─ runtime/
├─ scripts/
├─ AI Game Manager Panel.bat
├─ README.md
├─ AGENTS.md
├─ go.mod
├─ main.go
└─ wails.json
```

## 验证

- `node scripts/common/check-project-layout.mjs`
- `node scripts/common/check-windows-helper.mjs`
- `go test ./...`
- `go vet ./...`
- Frontend typecheck/build（依赖可用时）
- Electron typecheck/build（依赖可用时）
- Windows 实机 Release 仍需在 Windows 上最终验证。


---

## AI-Game-Manager-Panel 0.1.64

### Release Notes

<!-- historical source: docs/releases/0.1.64.md -->

## 十项统一修复

- 首次启动页拆分为左上品牌、右上步骤、中央表单三块独立区域。
- 全局 Light / Dark / System 主题收口到统一 Theme Source 与 Design Token。
- 工作台升级为可拖拽三栏布局，左右侧栏可完全收起，终端迁入可伸缩 Bottom Panel。
- 设置页提升大屏空间利用率与 Typography 信息层级。
- 许可证域明确 LicenseID / ActivationID / MachineCode / InstallID / Feature Entitlement，并保持登录与授权解耦。
- 运行环境迁入设置中心，新增 SteamCMD 与 GameLibraryRoot 等统一路径模型。
- 修复 PowerShell `$home` 与只读 `$HOME` 冲突，并禁止工具链 Hashtable 泄漏到控制台。
- Electron 原生菜单中文化。
- Electron Builder 复用本地 electronDist，规避重复下载链与缓存模式异常。
- Windows Helper / Prompt 归档 / Release Gate 同步升级。

## 验证说明

本版本最终验证结果以 `Bonfire-0.1.64-validation.md` 为准；未在 Windows 实机完成的项目必须明确标记为 `NOT VERIFIED ON WINDOWS`。

### Historical Development Prompt

<!-- historical source: docs/prompts/0.1.64-ten-issue-consolidated-fix.md -->

> 基线：AI Game Manager Panel 0.1.63。目标版本：AI Game Manager Panel 0.1.64。
> 本任务必须一次性收口旧五项基础体系问题与 Windows/Wails/Electron/App Shell 新发现问题；禁止拆成互相覆盖的临时补丁。

## 永久架构约束

- Go Core 是唯一业务核心；Wails / Electron / localhost Web 共用 Vue UI 与 Go Service。
- 平台差异只允许放在 Bridge / Adapter；禁止复制 License、Environment、Settings 或游戏业务逻辑。
- Windows 开发助手继续保持 `AI Game Manager Panel.bat -> scripts/windows/AI Game Manager Panel.ps1`，BAT 只负责 UTF-8 启动。
- PowerShell 文件必须 UTF-8 BOM + CRLF；禁止 `Invoke-Expression`、字符串拼接命令执行、Node `shell:true`、未知 CLI 参数。
- 正式开发提示词必须归档在 `docs/prompts/`，Validation 必须逐项引用本提示词验收。

## 问题 1：首次启动三块区域必须真正独立

- `BONFIRE FIRST START / AI游戏管理器面板` 是页面左上独立品牌区。
- `01 / 创建最高管理员` 是页面右上独立步骤区。
- 注册/登录表单是第三块，独立视觉居中。
- 左右浮动区不得参与中央 Form 宽度计算；1920/2K/4K 不能一起缩在中间。
- 窄窗口允许响应式和滚动，但不得把品牌/步骤重新塞进 Form 内部。

## 问题 2：Light / Dark / System 必须全局生效

建立唯一 Theme Source：`light | dark | system`。

必须覆盖：`html/body/#app`、App Shell、左/右侧栏、Topbar、Bottom Panel、页面背景、Card、Input、Select、Dialog、Dropdown、Button、Table、Tooltip、Toast、状态栏、First Start 和所有主要功能页面。

- `system` 保存的仍是 `system`，并实时监听 `prefers-color-scheme`。
- 禁止再出现“中间白卡片 + 外围黑色壳”的混合主题。
- 历史硬编码颜色必须迁移到 Design Token，只有品牌色/语义状态色允许保留。

## 问题 3：大屏利用率与 Typography

- 主工作区不能再是屏幕中央的一张窄网页卡片；中央内容仅保留约 24~32px 安全边距。
- Feature Title 14~16px/600~700；Description 13~14px；Value 14~16px；Page/Section 标题按层级放大。
- 大屏可用双列/四列信息卡，小屏自动单列。
- “机器码/安装 ID/授权状态/路径”等标题、说明、值必须有明确视觉层级。

## 问题 4：License Domain 完整定义

统一定义：`UserID / LicenseID / MachineCode / InstallID / ActivationID`。

- MachineCode 代表设备；Windows 使用稳定系统标识的 SHA-256 派生值，不暴露原始 MachineGuid；Linux 有对应策略。
- InstallID（BFID）代表当前 AI Game Manager Panel 安装实例；旧 `device.id` 必须兼容迁移。
- CDK 定义为未来在线兑换用短激活码 `BF-XXXX-...`。
- 真正本地签名凭证定义为 `BFLC2.<payload>.<signature>` License Certificate，Ed25519；客户端只含公钥，私钥永不进入仓库/客户端。
- License 不能阻塞 Owner 创建与登录；高级功能通过 Go Core `HasFeature`/Entitlements Gate。
- Wails / Electron / 本机 Web 必须读取同一个 Go Core License 状态。
- 支持本地解绑；云端远程解绑/席位管理只预留架构，不伪装已上线。
- UI 必须解释 MachineCode、InstallID、LicenseID、ActivationID、CDK 与离线证书各自用途。

## 问题 5：Environment / Storage 路径体系

- 左侧一级“运行环境”移入 `设置中心 -> 运行环境与存储`。
- 建立唯一 StoragePaths：`ProgramDir / DataDir / SteamCMDRoot / SteamCMDPath / GameLibraryRoot / InstanceConfigRoot / GameSaveRoot / CacheRoot`。
- Windows 默认 SteamCMD 和 GameLibraryRoot 跟随 AI Game Manager Panel 所在盘符，例如 `H:\SteamCMD`、`H:\GameServers`；不能无脑写死 C 盘或源码目录。
- Linux 使用合理 Unix 路径且全部可配置。
- SteamCMD 支持检测、手动选择、自动安装、迁移；迁移失败旧目录保持可用。
- GameLibraryRoot 是以后安装/更新/校验/启动的唯一默认游戏程序根目录；不得继续把 `instances/games` 当新默认。
- 游戏程序、实例配置、游戏存档、缓存必须分离。

## 问题 6：PowerShell `$HOME` 冲突导致 Wails 开发/Release 失败

- 审计 `scripts/windows` 中 PowerShell 自动变量/只读变量冲突。
- 修复 `Get-AI Game Manager PanelWailsCli` 中 `$home` 与 `$HOME` 的大小写不敏感冲突。
- 门禁扫描至少禁止脚本局部变量赋值到 `HOME/Host/PID/PSVersionTable/PSScriptRoot/PSCommandPath` 等自动/只读变量。
- 必须回归菜单 `2 -> 1` 与菜单 `4` 的执行链。

## 问题 7：PowerShell 工具链对象泄漏

- `Show-AI Game Manager PanelToolchain` 只输出 `[环境] Go/Node/pnpm` 等可读文本。
- 不得把 Hashtable/PSObject 无意写入成功输出管道形成 `Name / Value` 表格。
- 所有以 Show/Write 命名的 UI 函数都检查是否有对象泄漏。

## 问题 8：Electron 原生菜单必须中文化

- Electron Application Menu 显式使用中文：文件、编辑、视图、窗口、帮助。
- Undo/Redo/Cut/Copy/Paste/Delete/Select All、Reload、Zoom、Fullscreen 等都使用中文 label。
- 不得使用默认英文 Application Menu。
- 帮助菜单不得跳转到无关/占位 GitHub 地址；优先使用本地“关于 AI Game Manager Panel”对话框。

## 问题 9：Electron Windows Release `ReadWrite` 崩溃

用户实机日志：`electron-builder 26.16.1` 在 `app-builder-lib/src/util/electronGet.ts -> resolveCacheMode` 报 `Cannot read properties of undefined (reading 'ReadWrite')`。

修复原则：

- 不猜依赖版本、不随意升级 Electron/electron-builder。
- AI Game Manager Panel 已通过 `install-electron` 准备 `desktop/electron/node_modules/electron/dist`，electron-builder 应显式复用本地 unpacked Electron Runtime (`electronDist`) 而不是再次走 `@electron/get` Electron distribution 下载链。
- Release 前验证 `electron.exe` / `electronDist` 实际存在；配置缺失时提前失败并给出中文诊断。
- Electron Release 与 Wails Release 保持独立。

## 问题 10：App Shell 改成 Codex 类三栏可拖拽工作台

默认大屏设计基准：左 20% / 中 60% / 右 20%。

- 左栏、中栏、右栏是 App Shell 层，而不是 Settings 页面自己的假三栏。
- 左/中和中/右之间可拖拽调整宽度，宽度持久化。
- 左右栏均可完全收起到 0；Topbar 提供恢复按钮；恢复时恢复上次宽度。
- 中央工作区随左右栏收起自动扩展，页面只留约 30px 内边距。
- 右栏是上下文辅助面板，可按页面隐藏；窄屏优先隐藏右栏。
- 左侧导航移除终端入口。
- 终端迁到底部 Bottom Panel；Bottom Panel 可打开/关闭、完全隐藏、拖拽高度并持久化；未来承载终端/日志/任务输出标签。
- 不允许底部面板关闭后残留空白。
- Light/Dark/System 必须同时覆盖三栏和 Bottom Panel。

## 强制测试 / Gate

至少执行：

1. Frontend relative import integrity。
2. `vue-tsc --noEmit`（环境具备 pnpm 时）。
3. Vite production build（环境具备 pnpm 时）。
4. `go test ./...`。
5. `go vet ./...`。
6. License Service tests：签名错误、MachineCode、InstallID、过期、Feature、解绑。
7. Environment Service tests：Windows/Linux 默认路径模型、GameLibraryRoot 不使用 `instances/games`、迁移失败保护。
8. Windows Helper 编码 Gate。
9. PowerShell 自动变量冲突 Gate。
10. PowerShell 对象泄漏 Gate。
11. Electron 中文菜单 Gate。
12. Electron builder local `electronDist` Gate。
13. Terminal 不得残留左侧导航 Gate。
14. Theme Source / workbench token Gate。
15. 硬编码 `C:\` 与 `instances/games` 扫描（测试样例/兼容 fallback 可说明）。
16. 版本号一致性。
17. ZIP 完整性。

视觉验收至少覆盖：1280x720、1366x768、1600x900、1920x1080、2560x1440；Dark/Light/System 均检查。

## 交付

输出：

- `AI Game Manager Panel-0.1.64-source.zip`
- `AI Game Manager Panel-0.1.63-to-0.1.64-changes.diff`
- `AI Game Manager Panel-0.1.64-validation.md`

Validation 必须逐项列出 10 个问题的实现与测试结果。当前环境没有真实 Windows/Wails/electron-builder 时，必须明确标记 `NOT VERIFIED ON WINDOWS`，不得伪造 PASS。


---

## AI-Game-Manager-Panel 0.1.63

### Historical Development Prompt

<!-- historical source: docs/prompts/0.1.63-five-foundation-fixes.md -->

> 来源：用户确认的 0.1.63 修复要求。此文件作为本版本实施依据归档，后续不得删除或改写历史内容。

你现在负责继续开发 AI Game Manager Panel 项目。

============================================================
【唯一源码基线】
============================================================

当前唯一源码基线：

AI Game Manager Panel-0.1.62-source.zip

除非我明确指定新的源码基线，否则：

- 禁止基于更旧版本修改；
- 禁止把历史版本代码重新覆盖回来；
- 禁止重新实现已经存在且正常工作的功能；
- 禁止擅自删除现有功能；
- 所有修改必须基于 0.1.62；
- 下一版本使用 0.1.63。

版本规范：

0.1.62
→ 0.1.63
→ ...
→ 0.1.99
→ 0.1.100
→ 下一大阶段 0.2.0

============================================================
【开发原则】
============================================================

AI Game Manager Panel 当前架构：

Go Core
+
Vue 3
+
TypeScript
+
Vite
+
Pinia
+
pnpm
+
Wails Desktop
+
Electron Desktop
+
Web

必须继续保持：

                    AI Game Manager Panel Go Core
                           │
              ┌────────────┼────────────┐
              │            │            │
            Wails       Electron       Web
              │            │            │
              └──────── Vue 3 ──────────┘

强制要求：

1. Go Core 是唯一业务核心。
2. Wails / Electron / Web 不允许分别复制业务逻辑。
3. Vue UI 尽量共用。
4. 平台差异必须通过 Adapter / Bridge 层解决。
5. 禁止因为修 Web 而破坏 Wails。
6. 禁止因为修 Electron 而破坏 Web。
7. 禁止新增第二套重复 License / Settings / Environment Service。
8. 必须先阅读现有源码，再决定修改方式。
9. 禁止根据猜测创建不存在的 API。
10. 禁止为了“快速修复”写死路径。
11. 禁止把 Windows 路径假设成只有 C 盘。
12. Windows/Linux 路径必须分别设计。
13. 所有中文 UI、提示、README、开发文档、代码注释继续使用中文。
14. JS/TS 包管理统一 pnpm。
15. 不允许使用 npm / yarn。
16. 不得擅自升级 Vue、Vite、Electron、pnpm 等主要依赖。
17. 不得把实验性代码直接覆盖稳定功能。

============================================================
【Windows 脚本硬性开发规范】
============================================================

当前 Windows 开发助手：

AI Game Manager Panel.bat
→ scripts/windows/AI Game Manager Panel.ps1

继续保持此方向。

强制规范：

AI Game Manager Panel.bat：
- 只负责入口；
- ASCII-safe；
- CRLF；
- 禁止承载复杂业务；
- 禁止直接输出大量中文菜单；
- 禁止负责 pnpm / Electron / Release 核心编排。

PowerShell：
- *.ps1 统一 UTF-8 with BOM；
- CRLF；
- Console InputEncoding UTF-8；
- Console OutputEncoding UTF-8；
- $OutputEncoding UTF-8。

严格禁止：

Invoke-Expression
shell:true
字符串拼接命令执行
--prefer-online
未验证的 CLI 参数
中文脚本乱码
� 替换字符
CMD 将中文残片识别成命令

外部程序调用必须优先：

& $exe @args

而不是拼接：

"$exe $arg1 $arg2"

出现任何：

'��' is not recognized
�
中文乱码
路径因为空格被截断
C:\Program 被当作命令

均直接判定：

WINDOWS SCRIPT GATE FAILED

禁止进入 Release。

============================================================
【开发流程】
============================================================

严格执行：

源码审计
↓
制定修改方案
↓
实现
↓
静态检查
↓
单元测试
↓
Frontend TypeScript
↓
Go Test
↓
Go Vet
↓
Web Build
↓
Wails 检查
↓
Electron 检查
↓
脚本编码检查
↓
Windows 路径检查
↓
最终回归
↓
生成 Validation Report
↓
才允许交付

禁止：

“理论上应该可以”
“应该没问题”
“先发给用户测试看看”

必须尽可能在开发环境提前发现问题。

============================================================
【本次版本目标】
============================================================

版本：

AI Game Manager Panel 0.1.63

主题：

UI / Theme / License / Environment & Storage Architecture Fix

一次性修复下面 5 个问题。

不要拆成 5 个互相独立、互相覆盖的临时补丁。

============================================================
问题一：首次启动页面布局错误
============================================================

当前问题：

首次启动页面：

BONFIRE FIRST START
AI游戏管理器面板

以及：

01
创建最高管理员

虽然已经左右分开，

但是仍然属于中央表单布局体系的一部分。

这不是最终需求。

正确设计：

页面应该存在三个完全独立的区域：

A. 页面左上品牌区

B. 页面右上步骤区

C. 页面中央表单区

结构：

┌────────────────────────────────────────────────────┐
│                                                    │
│ [左上品牌]                           [右上步骤]      │
│ BONFIRE FIRST START                  01             │
│ AI游戏管理器面板                         创建最高管理员  │
│ 首次启动安全与环境初始化              创建账号...     │
│                                                    │
│                                                    │
│                 ┌──────────────┐                   │
│                 │              │                   │
│                 │   注册表单    │                   │
│                 │              │                   │
│                 └──────────────┘                   │
│                                                    │
└────────────────────────────────────────────────────┘

强制要求：

1. 左上品牌区域脱离中央容器。
2. 右上步骤区域脱离中央容器。
3. 两者不参与中央 Form 宽度计算。
4. 中央 Form 永远独立视觉居中。
5. 大屏、1080P、2K、4K 都必须合理。
6. 普通窗口也必须合理。
7. 小窗口内容过高时允许滚动。
8. 小窗口不得把左右 Header 再塞回 Form。
9. 不得因为响应式导致 Header 相互重叠。
10. 左右安全距离需要统一设计。

建议桌面大屏：

top: 48~64px
left/right: 48~64px

但不要机械写死，应根据响应式设计。

验收：

1920×1080：
左上品牌明显属于页面；
右上步骤明显属于页面；
中央卡片严格居中。

2560×1440：
不能三个区域一起缩在屏幕中间。

窗口缩小：
不得出现裁切、覆盖、无法滚动。

============================================================
问题二：主题切换必须真正全局
============================================================

当前问题：

设置中心切换：

浅色
深色
跟随系统

只影响部分 Settings Card。

其他区域仍然保持深色，例如：

Sidebar
Topbar
Page Background
Breadcrumb
Status Bar
部分 Card
部分 Button
Input
Dropdown
Border
文字

造成：

深色背景 + 白色设置卡片

这种错误的混合主题。

必须重新审计整个 Theme System。

最终只允许存在统一 Theme Source。

建议：

theme = light | dark | system

system：

window.matchMedia('(prefers-color-scheme: dark)')

并且系统运行中切换 Windows 主题时：

AI Game Manager Panel 必须即时响应。

禁止：

刷新页面以后才改变；
只 SettingsView 改变；
页面自己写死 #111/#fff；
不同页面自己实现 theme。

必须统一：

CSS Variables / Design Tokens

例如：

--bg-app
--bg-sidebar
--bg-topbar
--bg-surface
--bg-elevated
--text-primary
--text-secondary
--border-default
--input-bg
--hover-bg
--accent
--success
--warning
--danger

所有页面使用变量。

需要审计：

App Shell
Sidebar
Topbar
Breadcrumb
Page
Card
Table
Dialog
Dropdown
Select
Input
Checkbox
Button
Tooltip
Toast
Status Bar
First Start
Settings
Environment
AI Workbench
Instance
Backup
User
License

必须保证：

浅色模式：
整个 UI 统一浅色。

深色模式：
整个 UI 统一深色。

跟随系统：
系统浅 → AI Game Manager Panel 浅；
系统深 → AI Game Manager Panel 深；
系统运行时变化 → AI Game Manager Panel 实时变化。

Wails / Electron / Web 三端必须一致。

主题选择保存后：

重新启动仍保持。

system 模式保存的是：

system

不是保存当时解析出来的：

dark

============================================================
问题三：大屏利用率 + Typography 信息层级
============================================================

当前问题：

在 1920×1080 / 2K / 4K：

中间内容虽然居中，

但是：

左右留白过多；
主内容区过窄；
字体过小；
功能标题和功能描述没有层级；
核心状态不突出；
用户看不清重点。

当前类似：

机器码
一行非常小的说明
BFM-XXXX

全部挤在一起。

必须重新建立 AI Game Manager Panel Typography / Layout Scale。

============================================================
【页面宽度】
============================================================

不要继续固定很窄的 max-width。

建议采用响应式：

小窗口：
100% - safe padding

普通桌面：
960~1200px

1080P：
1200~1360px

2K/4K：
可扩展到 1360~1520px

具体值根据现有 Sidebar 宽度计算。

不要盲目占满全屏。

目标：

充分利用空间
+
保持阅读集中度。

============================================================
【Typography 层级】
============================================================

建立全局信息层级：

Page Eyebrow
例如：
BONFIRE PREFERENCES

Page Title
例如：
设置

Section Title
例如：
授权与设备安全

Feature Title
例如：
机器码
设备识别码
授权状态
CDK
调试模式

Description
功能说明

Value / Status
数据和状态

建议：

Page Title：
22~28px
600~700

Section Title：
16~18px
600~700

Feature Title：
14~16px
600~700

Description：
13~14px
400

Value：
14~16px

不要所有文字都 10~12px。

============================================================
【信息布局】
============================================================

大屏允许使用两列。

例如：

外观与语言：

主题                 语言
[跟随系统]           [简体中文]

授权设备：

机器码               安装 ID
BFM-...               BFID-...
[复制]                [复制]

不要所有内容永远单列堆叠。

但是：

移动端 / 小窗口自动退回单列。

============================================================
【Feature Row】
============================================================

类似“机器码”应该：

机器码
用于识别当前计算机，并参与许可证设备绑定。

BFM-XXXXXXXX...
[复制]

标题明显；
描述次一级；
Value 再形成一个视觉区域。

不要：

标题 + 描述 + Value 全部挤成一行。

============================================================
问题四：许可证 / CDK / 机器码 / InstallID 架构重构
============================================================

当前问题：

现有 Settings 中出现：

机器码
设备识别码
CDK
授权状态

但产品层没有定义清楚：

CDK 是什么？
谁生成？
机器码是什么？
设备识别码为什么存在？
唯一 ID 是什么？
如何绑定？
如何解绑？
换电脑怎么办？
Web 和 Desktop 谁拥有许可证？
离线怎么办？
云端怎么办？

因此这次不能只修改 UI。

必须正式定义 AI Game Manager Panel License Domain Model。

============================================================
【身份模型】
============================================================

至少明确以下实体：

UserID
LicenseID
MachineCode
InstallID
ActivationID

意义：

UserID
谁在使用。

LicenseID
一张许可证唯一 ID。

MachineCode
一台物理/系统设备的稳定指纹。

InstallID
当前 AI Game Manager Panel 安装实例 ID。

ActivationID
License 与设备之间的一次绑定记录。

推荐：

LicenseID：
UUIDv7 / ULID
数据库唯一。

ActivationID：
UUIDv7 / ULID
数据库唯一。

============================================================
【MachineCode】
============================================================

Windows：

读取稳定系统标识，
经过 AI Game Manager Panel 专用 SHA-256 派生。

不得直接显示原始 MachineGuid。

格式例如：

BFM-XXXXXXXX-XXXXXXXX-XXXXXXXX-XXXXXXXX

Linux：

实现对应稳定设备标识策略。

不要写死 Windows MachineGuid。

MachineCode：

代表“设备”。

============================================================
【InstallID】
============================================================

当前 BFID 不应该继续在 UI 里叫：

设备识别码

因为和 MachineCode 容易混淆。

统一定义为：

安装 ID

例如：

BFID-XXXXXXXX-XXXXXXXX-XXXXXXXX

代表：

当前 AI Game Manager Panel 安装实例。

第一次运行生成。

============================================================
【CDK】
============================================================

重新定义 CDK：

CDK 是用户输入的短激活码。

例如：

BF-7K2P-M8QX-4N6R-W9TY

CDK 不应该直接等于 Ed25519 长签名许可证。

CDK 用于：

兑换 License。

============================================================
【License Certificate】
============================================================

真正 Ed25519 签名的数据：

改称：

License Certificate
许可证证书

例如：

BFLC2.<payload>.<signature>

客户端：

只内置 Public Key。

私钥：

只存在 License Server / Developer Offline Signer。

绝对不能放入：

AI Game Manager Panel 客户端源码
Wails
Electron
Web
安装包
Git 仓库

============================================================
【License Payload】
============================================================

至少包含：

LicenseID
Product
Edition
Features
IssuedAt
ExpiresAt
SeatLimit
MachineCode
InstallID
ActivationID

数字签名：

Ed25519。

============================================================
【授权状态】
============================================================

不能只有：

有效
无效

定义：

UNLICENSED
ACTIVE
EXPIRING
EXPIRED
DEVICE_MISMATCH
REVOKED
SEAT_LIMIT
OFFLINE_GRACE
SERVER_UNREACHABLE

UI 显示中文。

============================================================
【不要阻塞登录】
============================================================

重要：

许可证不得重新回到注册/登录前面。

流程必须：

创建 Owner
↓
登录
↓
进入 AI Game Manager Panel
↓
许可证状态检查

账号系统：

判断“你是谁”。

许可证系统：

判断“你能使用哪些授权功能”。

不得混用。

============================================================
【未授权功能】
============================================================

未授权用户仍允许：

登录
设置
许可证页面
环境配置
基础诊断
帮助

需要许可证的高级功能使用 Feature Gate。

例如：

点击高级功能：

此功能需要有效 AI Game Manager Panel 授权。

当前状态：
未激活

[前往激活]

不要把整个 AI Game Manager Panel 卡死。

============================================================
【Feature Entitlements】
============================================================

许可证不要只判断：

valid = true

增加：

Features / Entitlements

例如：

server.basic
server.multi
backup.advanced
remote.agent
web.remote
automation
ai.workbench
plugin.extensions

UI/Go Core 通过统一：

licenseService.HasFeature(...)

判断。

不得在 Vue 里写死 Edition 判断。

============================================================
【设备绑定】
============================================================

License：

Seats = 1 / 3 / N

Activation：

LicenseID
+
MachineCode
+
InstallID
+
ActivationID

支持：

绑定当前设备
解绑当前设备
远程解绑
设备列表

============================================================
【换电脑】
============================================================

旧电脑正常：

设置
→ 授权
→ 解绑当前设备

新电脑：

输入 CDK
或
登录账号重新激活。

旧电脑损坏：

未来云端账号中心支持：

设备管理
→ 远程解绑。

可以设置：

解绑频率限制。

============================================================
【Desktop / Web 架构】
============================================================

同一台机器上的：

Wails
Electron
localhost Web

不得各自拥有 License。

License 属于：

AI Game Manager Panel Go Core / Node。

架构：

License Service
     │
   Go Core
     │
 ┌───┼────┐
Wails Electron Web

三端显示同一个许可证状态。

============================================================
【远程 Web】
============================================================

未来真正 Cloud Web：

Browser
↓
AI Game Manager Panel Cloud
↓
Remote Node / Agent

Browser 不绑定 MachineCode。

MachineCode 属于：

运行 AI Game Manager Panel Core / Agent 的服务器节点。

============================================================
【本地 + 云端】
============================================================

设计成 Hybrid。

一期：

本地 Ed25519 Certificate
+
离线校验。

架构预留：

Online License Server。

未来：

在线激活
周期验证
短期 Lease
离线 Grace Period
吊销
Seat 管理
设备解绑

同时支持：

Offline Activation Request
+
Offline License Certificate Import

============================================================
【License UI】
============================================================

重做设置页：

授权与许可证

授权状态
未激活

许可证版本
—

设备使用量
0 / 1

[输入激活码]
[激活 AI Game Manager Panel]

设备信息

机器码
BFM-...

安装 ID
BFID-...

高级

[离线激活]
[导出离线激活请求]
[导入许可证]
[解绑当前设备]

普通用户不用直接看到：

Raw JSON
Public Key
Signature
Payload

============================================================
问题五：运行环境 / SteamCMD / 游戏服务器路径体系
============================================================

当前问题：

“运行环境”独占左侧一级菜单。

但本质属于设置。

而且当前：

SteamCMD 路径配置不灵活；
自动安装目录逻辑不合理；
不能检测已有 SteamCMD；
不能选择目录；
不能迁移；
缺少统一游戏服务器安装目录；
业务代码仍可能依赖源码目录 / instances/games；
没有明确 GameLibraryRoot。

必须重构。

============================================================
【菜单结构】
============================================================

移除左侧一级：

运行环境

移动到：

设置中心
→ 运行环境与存储

不要删除功能。

只是重新归类。

============================================================
【路径模型】
============================================================

建立统一 Path Service。

明确至少：

AI Game Manager PanelProgramDir
AI Game Manager PanelDataDir
SteamCmdRoot
GameLibraryRoot
InstanceConfigRoot
GameSaveRoot
CacheRoot

禁止以后代码自己随便：

filepath.Join(currentDir, "games")

所有路径统一通过：

PathService / EnvironmentService

============================================================
【SteamCMD 默认目录】
============================================================

Windows：

默认跟 AI Game Manager Panel 所在驱动器。

例如：

AI Game Manager Panel：

H:\一键部署\AI Game Manager Panel\

推荐：

H:\SteamCMD

Game Library：

H:\GameServers

如果 AI Game Manager Panel：

D:\Tools\AI Game Manager Panel

推荐：

D:\SteamCMD
D:\GameServers

如果只有 C：

C:\SteamCMD
C:\GameServers

不得无脑写：

C:\SteamCMD

也不得放到：

AI Game Manager Panel程序目录\SteamCMD

避免升级/删除程序时误删数据。

============================================================
【Linux】
============================================================

Linux 没有 C 盘。

例如：

/opt/ai-game-manager-panel/steamcmd
/srv/ai-game-manager-panel/games

或者：

/mnt/game/steamcmd
/mnt/game/servers

路径全部可配置。

============================================================
【SteamCMD 设置 UI】
============================================================

显示：

SteamCMD

状态
已安装 / 未安装

当前路径
H:\SteamCMD\steamcmd.exe

功能：

[自动检测]
[浏览选择]
[安装 SteamCMD]
[重新检测]
[移动 SteamCMD]
[打开目录]

============================================================
【自动检测 SteamCMD】
============================================================

检查：

当前配置路径
AI Game Manager Panel 同盘默认路径
C:\SteamCMD
D:\SteamCMD
E:\SteamCMD
PATH
常用目录

不要默认扫描整个硬盘。

提供额外：

深度扫描

让用户主动触发。

============================================================
【移动 SteamCMD】
============================================================

不是修改字符串。

流程：

检查没有运行中的 SteamCMD 任务
↓
复制到目标
↓
验证
↓
更新配置
↓
重新检测
↓
成功
↓
询问是否删除旧目录

出现失败：

保持旧目录可用。

============================================================
【游戏服务器安装根目录】
============================================================

新增核心设置：

游戏服务器库

例如：

D:\Games
C:\GameServers
E:\Steam\GameServers
H:\DedicatedServers

所有 AI Game Manager Panel 安装的游戏：

GameLibraryRoot
│
├─ DontStarveTogether
├─ Palworld
├─ ProjectZomboid
└─ ...

后续：

安装
更新
校验
开服
备份
实例
命令生成

统一读取 GameLibraryRoot。

禁止再自动安装进：

AI Game Manager Panel 源码目录
C:\ 默认目录
instances\games

除非用户明确配置。

============================================================
【程序目录和存档目录分离】
============================================================

必须区分：

游戏程序：

D:\GameServers\DontStarveTogetherDedicatedServer

游戏存档：

Klei\DoNotStarveTogether\Cluster_1

不要混淆。

由不同 Game Adapter 提供默认 Save Root。

============================================================
【修改 GameLibraryRoot】
============================================================

用户从：

D:\GameServers

改到：

E:\GameServers

如果已有游戏：

提示：

检测到当前目录存在 86.4GB 游戏服务器文件。

选择：

1. 仅更改以后新游戏默认安装目录
2. 迁移现有游戏到新目录
3. 取消

迁移流程：

停止相关服务器
↓
检查磁盘空间
↓
复制
↓
验证
↓
更新实例
↓
更新启动路径
↓
验证
↓
成功
↓
询问删除旧目录

不得直接移动后祈祷成功。

============================================================
【设置页面 UI 一起重构】
============================================================

“运行环境与存储”至少分：

运行状态
SteamCMD
游戏服务器库
AI Game Manager Panel 数据目录
实例配置
游戏存档
缓存

每个功能：

标题明显
描述清晰
当前值明确
操作按钮明确

不要再使用大量 10px 小字体。

============================================================
【额外要求：视觉统一】
============================================================

这 5 个问题修复过程中：

不要创建 5 种不同设计语言。

必须继续沿用 AI Game Manager Panel：

深色高级管理面板
橙色品牌 Accent
清晰 Card
轻量边框
状态色

并建立：

Spacing Scale
Typography Scale
Card Scale
Responsive Breakpoints

避免：

某页面字号 10px
另一页面 16px
另一页面 20px

所有页面统一。

============================================================
【测试要求】
============================================================

修改完成后必须执行：

1. Frontend import integrity
2. vue-tsc --noEmit
3. Vite production build
4. go test ./...
5. go vet ./...
6. License Service tests
7. Theme Store tests
8. Path Service tests
9. Environment Service tests
10. Web API tests
11. Wails bridge compile validation
12. Electron bridge TypeScript validation
13. Windows script encoding gate
14. Relative path scan
15. hard-coded C:\ scan
16. hard-coded instances/games scan
17. duplicated License Service scan
18. duplicated theme logic scan

============================================================
【视觉验收尺寸】
============================================================

至少检查：

1280×720
1280×800
1366×768
1600×900
1920×1080
2560×1440

重点：

First Start
Settings
License
Environment & Storage
Sidebar
Topbar

============================================================
【主题视觉验收】
============================================================

每个尺寸至少检查：

Dark
Light

另外检查：

System → Dark
System → Light

主题切换后：

不得存在大面积上一主题残留。

============================================================
【首次启动验收】
============================================================

左上：
品牌

右上：
步骤

中央：
Form

三者必须真正独立。

============================================================
【License 验收】
============================================================

至少测试：

无许可证
有效许可证
错误签名
错误 MachineCode
错误 InstallID
已过期
Feature 无权限
Feature 有权限
本地 Wails
本地 Electron
本地 Web

三端状态一致。

============================================================
【Path 验收】
============================================================

至少测试：

C:\AI Game Manager Panel
D:\AI Game Manager Panel
H:\一键部署\AI Game Manager Panel
带空格路径
中文路径

SteamCMD：

不存在
自动检测
手动选择
自动安装
迁移

GameLibrary：

D:\GameServers
E:\Steam\GameServers

确保：

没有写死 C 盘。

============================================================
【禁止事项】
============================================================

禁止：

只改截图对应 CSS
只修当前页面
新增重复 License Service
复制 Wails/Electron 业务代码
为了测试把授权始终设为 true
把 CDK 私钥放客户端
重新把 License 放登录前
无授权直接整个程序不能登录
将游戏默认安装到源码目录
把 SteamCMD 自动塞进源码目录
把所有游戏无脑装 C 盘
重新出现 instances\games 作为唯一默认逻辑
UI 用大量极小字体
浅色只改 Card
使用硬编码 theme color
提交没有测试的源码

============================================================
【交付要求】
============================================================

最终必须输出：

AI Game Manager Panel-0.1.63-source.zip

AI Game Manager Panel-0.1.62-to-0.1.63-changes.diff

AI Game Manager Panel-0.1.63-validation.md

并在 validation 中逐项写清：

问题 1
改了什么
测试什么
结果

问题 2
改了什么
测试什么
结果

...

问题 5
改了什么
测试什么
结果

以及：

仍未验证项

禁止把没有实机验证的项目写成：

PASS

必须标记：

NOT VERIFIED ON WINDOWS

============================================================
【最重要要求】
============================================================

先审计源码。

先列出：

现状
根因
计划修改文件
会影响的模块
回归风险

确认架构后再修改。

不要边猜边写。

不要看到报错就只补一个 if。

这次目标不是“让截图看起来正常”，而是把：

First Start Layout
Global Theme
Responsive UI
License Domain
Environment / Path Domain

五个基础体系真正建立稳定。

完成后再进入下一阶段开发。


---

## AI-Game-Manager-Panel 0.1.62

### Release Notes

<!-- historical source: docs/releases/0.1.62.md -->

## 修复

- 修复 `frontend/src/app/router.ts` 引用了不存在的 `features/backups/BackupsView.vue` 导致 `vue-tsc` 失败；备份恢复当前回到统一模块占位路由，直到真实备份 UI 落地。
- 修复 Windows 开发助手中文乱码根因：不再让 UTF-8 中文直接由 BAT/CMD 解析。
- 移除 0.1.61 大量 BAT 菜单/Task/Lib 编排，避免中文行被 CMD 错误拆分成命令。

## 架构调整

- 根 `AI-Game-Manager-Panel.bat` 仅保留 ASCII-safe 启动职责。
- Windows 菜单、依赖、检查、Wails/Electron Release 全部迁移到 PowerShell。
- 主菜单从 18 项合并为 10 项，删除重复的 Installer-only、双桌面对比、Electron 工具箱等顶层入口；相关能力合并到子菜单或独立 Release。
- 初始化与代码检查解耦：菜单 1 只准备环境和依赖，菜单 3 才负责 TypeScript/Go/Release Gate。

## 强制门禁

- 新增 `scripts/common/check-windows-helper.mjs`：检查根 BAT ASCII/CRLF、PS1 UTF-8 BOM/CRLF、乱码 replacement char、禁用 `Invoke-Expression`/`shell:true`/`--prefer-online`、菜单 1-10 映射、Frontend 相对导入完整性。
- 新增 `scripts/windows/Test-WindowsHelper.ps1`。
- `AIGameManagerPanel.ps1 -SelfTest -DryRun` 会遍历菜单 1-10 顶层入口。

## 测试

- 项目结构门禁。
- Windows Helper/编码/Frontend import 门禁。
- Go 全量测试与 `go vet`。
- Node 脚本语法检查。
- Windows amd64 Go 目标交叉编译。

## 已知限制

- 当前交付环境不是 Windows，不能冒充完成 Wails WebView2、Inno Setup、Electron NSIS 的 Windows GUI 实机执行；Windows 专属流程已通过结构/Dry-Run 契约门禁，最终 Release 仍需在 Windows 上运行菜单 4/5/10 验收。


---

## AI-Game-Manager-Panel 0.1.61

### Release Notes

<!-- historical source: docs/releases/0.1.61.md -->

## Windows BAT / Scripts Clean Rebuild

- 删除 `scripts/dev-helper/` Node 编排层。
- 根 `AI-Game-Manager-Panel.bat` 仅切换 UTF-8 并进入 `scripts/windows/menu.bat`。
- 新增 `scripts/windows/tasks/` 用户级任务层与 `lib/` 依赖/Runtime 层。
- Wails 与 Electron 可完全独立开发、检查与发布；菜单 15 提供 Electron 独立工具箱。
- 菜单 18 改为 Wails / Electron / 全部三选一，不再强制双桌面绑定。
- pnpm 通过 `pnpm.cmd` 直接调用，避免 `C:\Program Files` 路径被 shell 截断。
- 正常 pnpm 输出直接连接 CMD，不再被 Node/PowerShell 捕获导致逐行 Progress。
- Bonfire 继续使用独立 pnpm Store；首次失败自动切换干净恢复 Store。
- Electron Runtime 使用官方 `install-electron --no`，镜像优先、官方回退，并使用独立 cache。
- 网络层默认保留用户代理，仅对 localhost 设置 NO_PROXY。


---

## AI-Game-Manager-Panel 0.1.60

### Release Notes

<!-- historical source: docs/releases/0.1.60.md -->

## Windows Dev Helper Runner Refactor

- `AI-Game-Manager-Panel.bat` 重构为极薄 Node 启动器，不再启动 PowerShell 菜单。
- 新增 `scripts/dev-helper/`：Node.js 负责中文菜单、依赖、自愈、检查与一键打包编排。
- 删除旧 `scripts/windows/actions-ui/*.ps1`、`bonfire-dev.ps1`、`bonfire-common.ps1`，减少 PowerShell 参数绑定/TTY/CLI 参数拼接故障面。
- PowerShell 仅保留 Inno Setup 等 Windows 专属辅助。
- pnpm 安装参数在运行前通过 `pnpm install --help` 校验；移除不存在的 `--prefer-online`。
- 正常安装使用 TTY `default` reporter；只有失败复现才使用 `append-only` 诊断。
- 保留 Bonfire 独立 pnpm Store、ENOENT Store 自愈、依赖指纹。
- Electron workspace 增加 `nodeLinker: hoisted`，符合 Electron 对 pnpm 打包物理 node_modules 的要求。
- Electron Runtime 与 npm 包健康检查分离，继续使用 Electron 42+ lazy binary 机制。
- Wails / Electron / Web 底层 BAT Action 保留；开发 BAT 新增 `BONFIRE_DEPS_READY` 复用，避免重复安装依赖。


---

## AI-Game-Manager-Panel 0.1.59

### Release Notes

<!-- historical source: docs/releases/0.1.59.md -->

## Electron 44 Runtime / pnpm 输出与下载修复

- 修复 Electron 42+ Runtime 下载机制判断：Electron 42 起不再通过 postinstall 自动下载二进制。
- Electron npm 包安装与 Runtime 准备拆成两个阶段；使用官方 `install-electron` 命令按需准备 `electron.exe`。
- Electron Runtime 使用独立缓存目录，避免重复下载。
- 中文 Windows 默认为 Electron Runtime 尝试镜像加速，失败自动回退官方 GitHub Release；不修改系统全局 npm/pnpm registry。
- 正常 `pnpm install` 改为直接继承控制台，不再通过 PowerShell 管道捕获 stdout，恢复动态 TTY 进度显示；失败时才切换 append-only 详细诊断。
- Frontend / Electron 的 packageManager 与开发助手统一为 `pnpm@11.17.0`。
- 独立 pnpm Store、自愈、严格 build-script policy、Wails/Electron 双桌面 Release 继续保留。


---

## AI-Game-Manager-Panel 0.1.58

### Release Notes

<!-- historical source: docs/releases/0.1.58.md -->

## PowerShell Runtime Binding Hotfix

- 修复 Electron pnpm policy 写回时 `Set-Content -Value` 表达式未加括号，导致 Windows PowerShell 把 `+` 解析成额外位置参数的问题。
- 新增 `Test-BonfirePowerShellRuntimeBindings` 真实运行时自检：使用临时 Electron workspace 强制走 policy 修复写回分支。
- 菜单 5（快速检查）、6（完整检查）、18（一键打包）以及首次初始化都会提前执行该自检。
- 保留 0.1.57 Bonfire 专用 pnpm Store 与 ENOENT 自愈逻辑。
- 不修改稳定的 Wails / Electron 底层 Release BAT 构建实现。


---

## AI-Game-Manager-Panel 0.1.57

### Release Notes

<!-- historical source: docs/releases/0.1.57.md -->

## pnpm Store 自愈与智能依赖恢复

- Windows 开发助手默认使用 `%LOCALAPPDATA%\Bonfire\DevTools\pnpm-store`，不再依赖用户全局 pnpm Store。
- Frontend / Electron 安装全部显式使用 Bonfire 专用 `--store-dir`。
- 新增 pnpm 失败分类：Store 损坏、依赖脚本策略、网络异常、未知异常。
- 遇到 `ERR_PNPM_ENOENT`、`importPackage`、Store copyfile ENOENT 时自动执行：
  1. 清理当前项目 `node_modules` 与依赖指纹；
  2. `pnpm store prune`；
  3. `pnpm install --force --prefer-online`；
  4. 若 Store 仍损坏，隔离旧 Store 并创建全新 Store 最终重试。
- Store 隔离只操作 Bonfire 自己的 Store，不会删除用户其他项目的全局 pnpm 缓存。
- 新增 `build/DEPENDENCY-RECOVERY.log`，记录自动恢复动作。
- `build/ONE-CLICK-STATUS.txt` 新增实际 pnpm Store 与恢复日志路径。
- 菜单 `1/6/7/8/9/17/18` 统一复用智能依赖层。
- 0.1.56 的 pnpm build-script 安全策略继续保留：`electron` 明确允许，`electron-winstaller` 明确拒绝。
- CDK / 机器码 / BFID 继续只作为“设置 → 授权与设备安全”的保险配置，不参与首次注册/登录门禁。


---

## AI-Game-Manager-Panel 0.1.56

### Release Notes

<!-- historical source: docs/releases/0.1.56.md -->

## 一键打包智能化与 pnpm 11 构建策略修复

- 修复 Electron 依赖安装因 `ERR_PNPM_IGNORED_BUILDS` 在 `electron-winstaller@5.4.0` 处失败。
- Electron workspace 使用 pnpm 11 `allowBuilds`：`electron: true`，`electron-winstaller: false`。Bonfire 使用 electron-builder 的 NSIS/Portable，不执行 electron-winstaller 生命周期脚本。
- 固定 `electron@44.3.0` 的 minimumReleaseAge 例外，避免 `pnpm install` 自动修改 workspace。
- 新增 Frontend/Electron 依赖健康检查、依赖指纹缓存和 Electron runtime 自动重建。
- 菜单 18 升级为 8 阶段智能流水线，完整门禁只执行一次，Wails/Electron Release 复用已验证依赖、Frontend Build、Go Test、Go Vet 和 Electron TypeScript Gate。
- 新增 `build/ONE-CLICK-STATUS.txt`，记录失败阶段、总耗时、工具版本以及成功产物 SHA-256。
- 未知依赖脚本不会被自动批准；自动修复只针对 Bonfire 已审核的 Electron 依赖策略。


---

## AI-Game-Manager-Panel 0.1.55

### Release Notes

<!-- historical source: docs/releases/0.1.55.md -->

## Windows 开发助手重构
- `AI-Game-Manager-Panel.bat` 变为 ASCII-safe 薄启动器，仅调用 `scripts/windows/bonfire-dev.ps1`。
- 新增彩色中文分组菜单：首次使用 / 日常开发 / 构建与发布 / 项目维护 / 快捷操作。
- 新增统一 `[1/N]`、`[环境]`、`[依赖]`、`[完成]`、`[失败]` 输出。
- 新增初始化、快速检查、完整检查、项目状态、依赖重置、一键双桌面打包等 PowerShell Actions。
- 底层已经实机验证的 Wails / Electron Release BAT 继续保留，由 PowerShell Action 调用，降低构建链回归风险。

## 授权保险调整
- 删除首次启动 Step 00 CDK 强制门禁。
- CDK/机器码配置移动到 `设置 -> 授权与设备安全`，未激活不会阻断注册、登录和正常使用。
- 新增本地随机 `BFID-...` 设备识别码。
- BFCDK2 同时绑定 `BFM 机器码 + BFID 识别码`，并继续使用 Ed25519 签名。
- Web 授权接口不再公开，必须登录后使用。
- 清理旧 `BFC1/BFM1` 授权实现与旧签发入口，只保留 `internal/system/license` + `cmd/aigame-manager-license-admin` 一套 BFCDK2 逻辑。


---

## AI-Game-Manager-Panel 0.1.54

### Release Notes

<!-- historical source: docs/releases/0.1.54.md -->

## License Activation Foundation

- 新增 CDK + 机器码许可证激活层，位于 Owner Bootstrap 之前。
- Windows 机器码优先使用 MachineGuid，经 Bonfire 专用 SHA-256 派生后显示；不保存硬件原始序列号。
- CDK 使用 Ed25519 签名，应用仅内置公钥，私钥只保留在发行方。
- CDK 可绑定单台机器、版本/Edition 与可选过期时间。
- 新增 `cmd/aigame-manager-license-admin` 发行工具，用于 keygen / issue。
- Wails、Electron、Web 共用同一 Go License Service。
- 本地 activation.json 保存签名许可证，不保存私钥。


---

## AI-Game-Manager-Panel 0.1.53

### Release Notes

<!-- historical source: docs/releases/0.1.53.md -->

## First-Start Layout Alignment Hotfix

本版本只调整首次启动视觉布局与响应式行为，不修改已经跑通的 Wails / Electron / Web 核心业务桥接与 Windows Release 主链。

### 首次启动舞台

- 整个首次启动区域继续保持水平、垂直视觉居中；
- 使用可滚动容器 + `margin:auto`，内容高度超过窗口时自动从可访问区域开始滚动，不再依赖小高度窗口强制顶对齐；
- 品牌标题从表单内容卡片中完全移出；
- 当前步骤从表单内容卡片中完全移出；
- 顶部采用左右布局：左侧显示 Bonfire 品牌，右侧显示当前步骤；
- 真正的注册、登录、环境初始化表单作为独立卡片居中显示在标题区下方；
- 大屏、全屏保持居中视觉，小屏保留完整滚动与窄屏适配。

### 保留的安全与交互规则

- Owner 首次注册仍然必须完成；
- 首次注册不要求显示名称，Owner 默认昵称为“超级管理员”；
- 安全密钥保存确认仍然是创建按钮解锁条件；
- “注册后立即启用安全密钥验证”仍为可选且默认不勾选；
- 复制密钥继续使用局部 DOM 反馈，禁止恢复成会触发整页响应式重绘的实现。

### 回归门禁

项目检查新增 0.1.53 布局门禁：

- 必须存在独立 `auth-shell` 与 `auth-stage-header`；
- 当前步骤必须使用独立 `auth-stage-summary`；
- 内容卡片中禁止再次出现旧 `auth-step` 标题；
- 首次启动整体必须使用自动边距视觉居中；
- 顶部必须保持左右布局；
- 实际表单卡片必须独立居中。


---

## AI-Game-Manager-Panel 0.1.52

### Release Notes

<!-- historical source: docs/releases/0.1.52.md -->

## Dual Desktop Foundation

Bonfire 从本版本开始支持双 Windows Desktop Adapter：

- Wails Desktop：继续作为已验证稳定桌面入口；
- Electron Desktop：新增实验桌面入口；
- Web：继续保留浏览器管理入口。

三种入口共用同一个 Vue 3 前端与 Go Core，不复制 SteamCMD、DST、实例、认证、日志、文件和进程业务逻辑。

## Electron 安全基线

- `nodeIntegration: false`；
- `contextIsolation: true`；
- `sandbox: true`；
- Renderer 权限请求默认拒绝；
- Preload 只暴露只读桌面框架元信息，不暴露通用 Node/IPC 能力；
- Electron 通过随机 loopback 端口访问 Bonfire Go Core；
- Production Core 使用用户可写 runtimeRoot，默认配置首次复制后保留用户修改。

## Windows 菜单

新增：

- 12：Electron Desktop 开发模式；
- 13：Electron Windows Release（NSIS + Portable）；
- 14：启动 Electron Portable；
- 15：Wails + Electron 双桌面对比构建。

原菜单 4 Wails Windows Release 保持独立，不因 Electron 实验链变化而被替换。


---

## AI-Game-Manager-Panel 0.1.51

### Release Notes

<!-- historical source: docs/releases/0.1.51.md -->

## First-Start UX Hotfix

- 首次启动窗口改为左侧品牌栏 + 右侧表单，减少纵向占用。
- 小高度窗口改为顶部对齐并完整滚动，避免居中布局导致底部内容难以访问。
- 默认桌面窗口调整为 1280×840；全屏布局保持居中视觉。
- 首次创建 Owner 移除“显示名称”字段，默认显示名称为“超级管理员”，登录后可在“用户与权限 → 个人资料”修改。
- 首次启动自动生成登录安全密钥；提交按钮只有在用户名、密码、确认密码有效且用户确认已保存密钥后才可点击。
- “立即启用账号 + 密码 + 安全密钥验证”保持可选，默认关闭。
- 两个复选框默认均为未选中状态。
- 修复复制安全密钥时因响应式状态更新造成的大面积重绘/闪烁：复制反馈改为按钮局部更新，不触发 AuthGate 整体响应式重绘。
- 登录安全密钥复制逻辑在“用户与权限”页面同步修复。
- 新增当前账号显示名称修改 API，并持久化到账号库。


---

## AI-Game-Manager-Panel 0.1.50

### Release Notes

<!-- historical source: docs/releases/0.1.50.md -->

## 首次启动与登录安全

- Owner 首次注册仍是唯一强制步骤，不能跳过。
- 注册页新增 256-bit `BFK1-` 登录安全密钥生成。
- 新密钥支持复制和下载到本地，并要求用户确认已经安全保存后再提交带密钥的 Owner 注册。
- 账号库只保存安全密钥 SHA-256 校验值，不保存密钥明文。
- 默认登录策略仍为“账号 + 密码”。
- 用户可在“用户与权限 → 登录安全”开启“账号 + 密码 + 安全密钥”三项验证。
- 已启用安全密钥验证时，缺少或错误密钥都会拒绝登录。
- 已登录用户重新验证当前密码后可以重新生成安全密钥，旧密钥立即失效。
- 新生成/轮换后的密钥只在当前界面显示一次，可复制或下载保存。

## 首次环境初始化

- Step 02 SteamCMD / 默认游戏服务器目录初始化改为可选。
- 新增“暂时跳过，直接进入 Bonfire”。
- 跳过状态持久化保存，下次启动不再强制回到环境初始化页。
- “运行环境”页增加 SteamCMD 安装、重新检测和补初始化入口。
- 完成初始化会自动清除此前的 skipped 状态。

## 安全与回归

- 新增安全密钥格式、强制登录、开启/关闭策略、轮换失效测试。
- 新增环境初始化跳过持久化与后续补初始化测试。
- HTTP 集成测试覆盖公开密钥生成、受保护安全状态、环境跳过及密钥登录。
- 项目骨架检查加入 0.1.50 回归门禁，防止安全密钥明文存储或移除可跳过环境初始化。

## 保留修复

0.1.49 Installer 语言回退、0.1.48 HTTP 测试数据隔离、0.1.47 SteamCMD 跨平台测试隔离、0.1.46 Windows BAT ASCII-safe / Inno Setup 自动补齐全部继续保留。


---

## AI-Game-Manager-Panel 0.1.49

### Release Notes

<!-- historical source: docs/releases/0.1.49.md -->

## Windows Installer Language Fallback Hotfix

- 修复 Windows Release 在 12/13 因 `Languages\ChineseSimplified.isl` 不存在而编译失败。
- `AIGameManagerPanel.iss` 始终使用 Inno Setup 自带 `Default.isl`，保证标准安装即可构建。
- 若 `ISCC.exe` 同目录存在 `Languages\ChineseSimplified.isl`，自动通过 `BONFIRE_USE_INNO_CHINESE` 启用简体中文。
- 若不存在，输出明确 WARN 后使用 Inno 默认英文继续构建，不再中断 Release。
- 新增项目门禁，防止未来再次把第三方语言翻译文件变成强制依赖。
- 保留 0.1.46～0.1.48 的 BAT 编码、Inno 自动安装、SteamCMD 测试隔离与认证数据隔离修复。

## 验收重点

Windows `AI-Game-Manager-Panel.bat -> 4` 应从 12/13 正常生成 `build\staging\windows-release\installer\Bonfire-Setup.exe`，随后进入 13/13 发布晋升。


---

## AI-Game-Manager-Panel 0.1.48

### Release Notes

<!-- historical source: docs/releases/0.1.48.md -->

## Release Test Data Isolation Hotfix

- 修复 `internal/bridge/httpapi/server_test.go` 在 Windows 上读取真实 `%APPDATA%\Bonfire` 账号状态的问题。
- 根因：测试只设置 `BONFIRE_ROOT`，临时根目录没有 `configs/` 时 `Application.New()` 的 dataDir 安全回退会使用 `os.UserConfigDir()/Bonfire`；Windows 不使用 `XDG_CONFIG_HOME`，因此测试可能命中真实用户目录。
- `Application` 新增 `Options{Root, DataDir}` 与 `NewWithOptions()`，集成测试使用 `t.TempDir()` 明确注入临时 dataDir。
- 新增 Application 级回归测试，验证 `accounts.json` 和 `bootstrap.lock` 均写在隔离目录。
- 项目结构检查新增门禁：HTTP 集成测试禁止恢复 `application.New()`，禁止依赖 `XDG_CONFIG_HOME` / `APPDATA` / `LOCALAPPDATA` 猜测测试目录。
- 保留 0.1.47 SteamCMD 测试隔离、0.1.46 Inno Setup/Windows BAT 修复和 0.1.45 Startup Security Gate。

## 验证

- `go test ./internal/...`：通过。
- `go vet ./internal/...`：通过。
- `go test ./internal/app ./internal/bridge/httpapi ./internal/system/auth ./internal/deploy/environment`：通过。
- `GOOS=windows GOARCH=amd64 go test -c ./internal/app`：通过。
- `GOOS=windows GOARCH=amd64 go test -c ./internal/bridge/httpapi`：通过。
- Windows 本机完整菜单 4（1/13～13/13）：待用户实机验收。


---

## AI-Game-Manager-Panel 0.1.47

### Release Notes

<!-- historical source: docs/releases/0.1.47.md -->

## Release Test Isolation Hotfix

- 修复 Windows 正式构建 `go test ./...` 在 `internal/deploy/environment` 因 `steamcmd` / `steamcmd.exe` 平台文件名不一致而失败。
- `TestInitializationWithExplicitSteamCMDPersists` 使用 `t.TempDir()` 内的显式临时 SteamCMD 路径，完全不读取真实机器安装状态。
- `TestDetectProjectSteamCMDUsesPlatformExecutableName` 根据运行平台创建 `steamcmd.exe` 或 `steamcmd` 测试夹具。
- 新增 `TestInitializationWithoutSteamCMDFailsClosed`，确认环境缺少 SteamCMD 时不会错误标记初始化完成。
- 项目结构检查新增 0.1.47 单测隔离回归门禁。
- 保留 0.1.46 的 ASCII-safe BAT、PowerShell Unicode 输出和 Inno Setup 自动补齐。

## Windows 实机验证重点

运行 `AI-Game-Manager-Panel.bat` -> `4`。在机器未安装 SteamCMD 的情况下，步骤 `6/13 Running Go tests` 应通过；SteamCMD 的真实安装只属于首次启动环境初始化流程，不应成为源码单测的外部依赖。


---

## AI-Game-Manager-Panel 0.1.46

### Release Notes

<!-- historical source: docs/releases/0.1.46.md -->

## Windows Release Toolchain Hotfix

- 修复 Windows `cmd.exe` 对 UTF-8 中文 BAT 的乱码与误解析：`scripts/windows/**/*.bat` 统一保持 ASCII-safe。
- 中文 Installer 工具链提示迁移到带 UTF-8 BOM 的 PowerShell helper，避免中文残片被 `cmd.exe` 当成命令执行。
- `AI-Game-Manager-Panel.bat` 中文菜单改为 PowerShell Unicode 输出，不再直接 `type` UTF-8 文件。
- Inno Setup 6 检测增加 PATH、常见安装目录与注册表探测。
- 有 `winget` 时优先安装 `JRSoftware.InnoSetup`。
- 没有 `winget` 或 winget 安装失败时，自动下载 Inno Setup 官方不可变 GitHub Release `6.7.3`。
- 安装包执行前进行 SHA-256 与 Authenticode 发布者校验。
- Release staging / promote / 失败保护上一份完整 Release 的事务语义保持不变。


---

## AI-Game-Manager-Panel 0.1.45

### Release Notes

<!-- historical source: docs/releases/0.1.45.md -->

## 版本定位

首次启动安全门禁。本版建立 Bonfire 的首个最高管理员、登录会话、Web API 身份门禁和环境初始化启动链，为后续 AI Workbench 与远程能力提供可信控制面基础。

## 新增

- 首次 Owner Bootstrap：全新数据目录只允许创建第一个最高管理员。
- `bootstrap.lock` 失效关闭：Owner 创建后公开 Bootstrap 永久关闭；账号库丢失或损坏也不会自动重开。
- 账号服务：Owner / Administrator / Operator 角色、管理员创建后续账号、账号列表。
- 密码保护：随机盐 + PBKDF2-HMAC-SHA256，不保存明文密码。
- 会话服务：安全随机 Bearer Token、24 小时过期、校验与登出。
- Web API 门禁：除健康检查、Bootstrap 状态、首个 Owner 创建和登录外，`/api/v1/*` 默认要求 Bearer Session。
- 首次启动 AuthGate：Owner 创建 → 登录 → 环境初始化 → Bonfire 主 Shell。
- 环境服务：SteamCMD 探测、DST Dedicated Server 默认路径探测、运行目录初始化与状态持久化。
- Windows SteamCMD 托管安装入口，下载官方 SteamCMD 压缩包并安全解压到 Bonfire 数据目录。
- “用户与权限”页面替换占位页，支持登录管理员查看和创建账号、主动退出。

## 安全边界

- Steam OpenID 只用于未来 SteamID 身份绑定，不与 Bonfire 密码会话混用。
- SteamCMD 默认匿名流程和具体游戏需要的凭据独立处理，本版不保存 Steam 密码。
- Web 即使已有 Bearer Session，0.1.45 仍保持 loopback-only；远程监听等待传输安全、显式绑定与更细 RBAC。
- Desktop/Wails 与 Web 共用 Application 服务，但 Web 额外执行 HTTP Bearer 门禁。

## 自动验证

- 首个 Owner 创建后 Bootstrap 关闭。
- 账号持久化、登录、Session 校验、登出和管理员创建后续账号。
- 删除账号库但保留 `bootstrap.lock` 时不会重新开放 Owner Bootstrap。
- 环境初始化与持久化。
- Go Core 单元测试、go vet 与项目骨架/版本一致性检查。

## Windows 实机待验收

- `pnpm --dir frontend run build` / Wails Windows Desktop 正式构建。
- 首次 Owner → 登录 → SteamCMD 探测/安装 → 环境初始化 → 主界面。
- 浏览器登录后的 Bearer API、登出与 401 会话失效回到 AuthGate。
- `AI-Game-Manager-Panel.bat -> 4` 产出 Desktop/Web/Portable/Setup。

## 下一版本

0.1.51：AI Workbench Framework（0.1.46～0.1.50 连续插入 Windows Release/Test/Installer 与 Startup Security UX 版本后顺延）。


---

## AI-Game-Manager-Panel 0.1.44

### Release Notes

<!-- historical source: docs/releases/0.1.44.md -->

## 版本定位

Windows Release Pipeline Hotfix。本版不扩展游戏业务和 AI 能力，专门修复 0.1.43 正式发布链的产物清理、安装器和目录语义问题。

## 修复

- 修复 Windows Installer 构建失败后主动删除 `build/bin/Bonfire.exe` 与 `Bonfire-Web.exe` 的问题。
- 修复 Portable 已生成但 Setup 失败时，半成品仍残留在正式 Release 根目录的问题。
- 修复 `BUILD-STATUS.txt` 只写笼统 FAILED、无法判断具体失败步骤的问题。
- 修复安装器路径、README、构建文档和诊断脚本之间不一致的问题。

## 优化

- `build/bin/` 固定表示最近一次成功编译的 Windows Desktop/Web 二进制。
- Windows Release 改成 staging 完整生成后再晋升，失败不会覆盖上一份完整 Release。
- Portable 改到 `build/release/windows/portable/`。
- Setup 改到 `build/release/windows/installer/`。
- Inno Setup 6 提前到耗时构建之前检查，缺失时可由用户选择 winget 安装。
- 菜单新增“仅构建 Windows 安装程序”，可复用现有 `build/bin`。
- `BUILD-STATUS.txt` 增加 `FailedStep`、`Reason` 和现有产物状态。

## 架构调整

Windows 正式发布采用：

```text
build/bin/                         最近一次成功编译 EXE
build/staging/windows-release/     本次发布临时产物
build/release/windows/             最近一次完整正式 Release
├─ portable/
└─ installer/
```

## 测试

- 项目骨架与版本门禁。
- Go 单元测试与 go vet。
- Batch 脚本静态一致性检查。
- Inno Setup 路径与正式产物路径检查。

## 已知限制

当前执行环境不是 Windows，无法在此直接运行 Inno Setup/Wails 的 Windows 实机 Release；最终验收需要在 Windows 上执行 `AI-Game-Manager-Panel.bat -> 4`。

## 下一阶段

进入 小鱼完整框架设计与实现。


---

## AI-Game-Manager-Panel 0.1.43

### Release Notes

<!-- historical source: docs/releases/0.1.43.md -->

## 本版定位

Phase 2.5：Linux Server Edition 与多架构部署基础。

## 新增

- 新增 `scripts/build_linux.sh` 顶层构建入口；
- 新增 linux/amd64 与 linux/arm64 静态 Server Edition；
- 新增 Linux 交互式安装/管理脚本；
- 新增 systemd 服务安装策略；
- 新增独立 `bonfire` 系统用户运行策略；
- 新增多架构 Docker Server Edition；
- 新增 `configs/server.json`；
- `configs/release.json` 增加 Server Edition 架构与产物命名；
- 中央架构文档、开发计划、功能状态矩阵同步更新。

## 安全与性能

- Linux 云服不再要求运行 Wails/GTK，控制面使用单一 Go Web 二进制；
- Server Edition 使用 `CGO_ENABLED=0` 静态构建，减少运行依赖；
- 安装过程可使用 root，但服务默认降权到 `bonfire` 用户；
- Docker 运行阶段使用非 root 用户；
- ARM64 控制面支持与具体游戏 ARM64 支持分离，避免错误承诺。

## 尚未完成

RCON、GIF 壁纸、通用 SteamCMD 管理、Docker 环境可视化、详细资源监控、全局端口/进程、通用配置编辑、Mod、全局备份、玩家管理等当前仍按模块骨架推进，不在本版冒充已完成。

- 远程 Web RBAC 尚未完成前，Linux 安装默认仅监听 `127.0.0.1`，避免无认证管理面板直接暴露公网。


---

## AI-Game-Manager-Panel 0.1.42

### Release Notes

<!-- historical source: docs/releases/0.1.42.md -->

## 版本主题

**Phase 2：全局日志中心**。把原先分散的 Bonfire Core 日志、DST 历史日志入口收敛为全平台唯一 LogHub，为后续实例、任务、SteamCMD、AI、节点和多游戏管理提供统一日志基础设施。

## 新增

- 新增 Go `internal/ops/logs/` 全局日志中心：
  - 日志目录 Catalog；
  - 真实文件数、真实行数、磁盘占用、活动日志统计；
  - size + mtime 行数缓存；
  - 活动日志尾部增长增量计数；
  - 从头、从尾、游标继续读取；
  - 全文、级别、分类筛选；
  - 单日志导出；
  - 筛选结果 ZIP 导出并生成 `manifest.json`；
  - 单条删除、筛选删除、全部历史清理；
  - 活动日志删除保护；
  - 手工删除文件后刷新同步真实状态。
- 新增操作审计 Recorder：有界 Channel + 后台 Buffered I/O + 定时 Flush。
- 新增 Desktop/Web 共用全局 LogHub API。
- 新增“打开项目 log/ 文件夹”。
- `configs/logging.json` 增加目录名称、Catalog 页大小、正文页大小、自动刷新周期和导出子目录配置。

## UI 重构

- 左侧“日志中心”升级为唯一全局日志管理页。
- 顶部展示日志文件、真实记录、磁盘占用、运行中日志统计。
- 支持来源、类型、游戏、实例、Shard、状态、日期和文件搜索。
- 支持正文从头/从尾查看、INFO/WARNING/ERROR/DEBUG、分类筛选和全文搜索。
- 支持分页、自动刷新、打开日志目录、下载当前日志、导出筛选日志包。
- 删除单个/筛选结果/清空历史使用 Bonfire 自己的确认对话框，不使用浏览器原生确认框。
- DST 工作台不再维护第二套历史日志中心；点击“日志中心”会跳转全局 `/logs` 并携带 DST、当前 Cluster、Master/Caves 筛选。

## 日志目录调整

新日志优先写入：

```text
log/
├─ bonfire/
├─ operations/
├─ audit/
├─ ai/
├─ steam/
├─ games/
│  └─ steam.dst/
├─ nodes/
└─ exports/
```

兼容旧版 `log/bonfire.log` 和 `log/dst/`，全局 Catalog 会继续识别旧历史日志。

## 操作审计

已接入部分现有关键操作：应用启停、设置保存、DST Token/Klei 配置导入、世界导入、Master/Caves/Cluster 启停、端口配置与清理、Steam 安装/校验、日志导出/删除等。

安全规则：

- 不记录 Klei Token 正文；
- 不记录 API Key、密码、Cookie；
- 发送 DST 控制台命令只记录目标和命令长度，不记录命令正文；
- 审计 Detail 再做敏感关键词脱敏保护。

## 性能优化

- Vue 不读取完整日志；Go 单次默认返回有限行数。
- Tail 无正文筛选时使用倒序块读取，不扫描整个文件。
- 全文筛选只在用户明确搜索时流式扫描。
- Catalog 的真实行数统计使用缓存；活动日志增长时优先从旧文件尾部增量计数。
- 自动刷新使用无重叠 `setTimeout`，页面不可见时不执行刷新。
- 前端 Catalog 数据无变化时避免替换整棵状态，降低 WebView2 重绘。

## 修复 / 架构调整

- 修复“游戏库内日志中心”和“左侧日志”重复设计问题。
- DST 专用 `DstLogCenter.vue` 已移除；DST 专用 Store 保留为运行日志采集/诊断底层能力。
- Core 日志从旧 `log/bonfire.log` 调整为 `log/bonfire/bonfire.log`。
- 新 DST Session 从旧 `log/dst/` 调整为 `log/games/steam.dst/`。
- 用户导出统一保存到 `log/exports/`，该目录不参与真实日志统计。

## 发布流水线修复

- 修复菜单 `4` 只生成免安装 EXE 却仍显示正式构建成功的问题；现在 `Bonfire-Setup.exe` 缺失时 Windows Release 直接失败。
- Windows Release 新增 `Bonfire-Windows-x64-Portable.zip`，并统一输出到 `build/release/windows/`。
- Inno Setup 工程正式接入 Windows Release；覆盖升级时 `configs/` 使用 `onlyifdoesntexist`，避免覆盖用户已经修改的非敏感配置。
- 新增 Linux 原生构建脚本和 Windows WSL 入口。
- Linux Release 输出 `Bonfire-Linux-x64.tar.gz` 与 `Bonfire_<version>_amd64.deb`。
- Linux `.deb` 使用用户 XDG 数据目录作为 `BONFIRE_ROOT`，解决 `/usr/lib` 只读导致日志/configs/实例无法写入的问题。
- `internal/platform/files/apppath` 新增 `BONFIRE_ROOT` 显式覆盖能力，并增加单元测试。
- 新增 `configs/release.json`，集中记录多平台 Release 命名、架构和运行目录策略。
- Windows 菜单扩展为 Windows Release、Linux Release、发布产物目录、诊断和清理独立入口。

## 测试

- `internal/ops/logs`：多来源聚合与真实行数测试。
- 从头/从尾/游标分页与结构化筛选测试。
- 活动日志删除保护测试。
- 外部手工删除后 Catalog 同步测试。
- 筛选 ZIP 导出与导出目录排除测试。
- 操作 Recorder 写入测试。
- 日志增长增量计数与无换行尾部正确性测试。
- `go test ./internal/...`。
- `go vet ./internal/...`。

## 已知限制

- 当前开发环境没有 pnpm/Vue 与 Linux GTK/WebKitGTK 开发依赖，因此无法在这里生成真实 Windows `Setup.exe` 或 Linux Wails 二进制；Windows 请执行 `AI-Game-Manager-Panel.bat -> 4`，Linux/WSL 请执行菜单 `5` 或 `bash scripts/linux/actions/build.sh` 做最终原生验收。
- `retentionDays / maxFileSizeMB / compressRotated` 已保留在统一配置中心，但自动轮转/过期归档策略将在任务/长期运行治理阶段继续完善；当前不会暗中自动删除用户日志。

## 下一阶段

Phase 3：统一实例管理。建立跨游戏 Instance 模型和实例生命周期，并把实例状态、资源指标、日志、备份、终端全部接入公共平台能力。


---

## AI-Game-Manager-Panel 0.1.41

### Release Notes

<!-- historical source: docs/releases/0.1.41.md -->

## 版本定位

Phase 1：现代化 AI 平台 Shell、统一配置中心与 GSM 全功能项目骨架。

## 新增

- 软件默认入口调整为 **小鱼**，布局参考现代 AI/Codex 工作台：左侧导航、顶部面包屑、中央自然语言入口、底部 Composer。
- AI Composer 新增三种审批模式 UI：
  - 请求批准；
  - 帮我批准；
  - 完全访问权限。
- AI 未配置时明确保留“手动一键部署”入口，不让 AI 成为核心功能的强制依赖。
- 新增根目录 `configs/` 配置中心：
  - `app.json`；
  - `ui.json`；
  - `ai.json`；
  - `permissions.json`；
  - `paths.json`；
  - `logging.json`；
  - `games.json`；
  - `modules.json`。
- 新增 Go `internal/config` 配置加载与校验层，并通过 Wails/HTTP 暴露同一平台配置快照。
- `configs/modules.json` 完整登记 GSM 对照能力，包括部署、实例、文件、终端、备份、任务、环境、插件、SteamCMD、RCON、EasyTier/网络、外部 API、认证、安全、系统监控、分块上传、云端构建等。
- 新增缺失的后端服务/平台/Agent/游戏适配目录占位，防止后续功能无归属。
- 新增集中式 `docs/PROJECT-ARCHITECTURE.md` 与 `docs/DEVELOPMENT-PLAN.md`。

## 修复

- 修正 0.1.40 文档组织过度分散的问题，不再要求每个源码目录都创建 README。
- 游戏列表开始读取 `configs/games.json`，不再由前端复制维护完整游戏模板清单。
- Windows Installer 同步 0.1.41，并把非敏感 `configs/` 一并安装，避免安装后找不到平台配置。

## 优化

- 导航菜单改为读取 `configs/ui.json`，避免侧栏信息架构散落在 Vue 源码里。
- AI 权限模式、工作台默认文案、模块矩阵全部配置化。
- `configs/paths.json` 开始控制 data/log 等运行目录；配置损坏时只保留安全回退用于启动诊断。
- 占位页面统一使用一个 `ModulePlaceholderView`，避免为几十个“尚未实现的模块”复制大量 Vue 页面。
- 项目骨架检查器改为验证：版本一致性、JSON 配置、核心目录、GSM 基线模块，不再维护逐文件 Markdown 索引。

## 架构调整

- AI 调用链冻结为：`Provider -> Planner -> Tool Registry -> Permission -> Approval -> Executor -> Bonfire Core`。
- 最高管理员可以决定 AI 审批模式；“完全访问”只针对已注册并启用的 Bonfire Tool，是否开放任意 Shell 由管理员配置。
- 项目架构说明集中到一个总文档，目录职责变化只更新 `docs/PROJECT-ARCHITECTURE.md`。

## 测试

交付前执行：

- `go test ./internal/...`；
- `go vet ./internal/...`；
- `node scripts/common/check-project-layout.mjs`；
- 前端 TypeScript/Vite 构建（环境可用时）；
- Windows 正式双 EXE 构建由本机环境继续实机验收。

## 已知问题

- Phase 1 只完成 小鱼、权限 UI 和 Tool/Provider 架构，尚未接入真实 AI Provider。
- 多数 GSM 对照模块当前仍是“骨架已预留”，不能当作真实功能已经完成。
- `Bonfire-Setup.exe` 工程已存在，但正式 Installer 仍在后续阶段纳入菜单 4 的强制发布门禁。

## 下一阶段

- 按 `docs/DEVELOPMENT-PLAN.md` 进入全局日志中心 / 统一实例管理等公共能力开发；
- 所有新功能优先填充现有目录和 `configs/modules.json`，禁止重新散乱创建平行系统。


---

## AI-Game-Manager-Panel 0.1.40

### Release Notes

<!-- historical source: docs/releases/0.1.40.md -->

## 版本定位

本版是**架构与源码地图基线**，不新增游戏服务器业务能力。目标是让人工开发者、ChatGPT、Codex 和后续贡献者能明确知道“每个目录负责什么、每个文件负责什么、一个新功能应该放哪里”。

## 新增

- 新增 `docs/architecture/` 架构文档体系：
  - `README.md`：阅读索引；
  - `DIRECTORY-MAP.md`：逐目录职责；
  - `FILE-MAP.md`：逐文件功能说明；
  - `LAYER-BOUNDARIES.md`：分层边界；
  - `MODULE-DEPENDENCIES.md`：模块依赖方向；
  - `NAMING-AND-PLACEMENT.md`：文件命名/归档规则；
  - `NEW-FEATURE-CHECKLIST.md`：新功能架构自检。
- 主要源码层新增就地 README：
  - `internal/`、`core/`、`platform/`、`service/`、`games/`、`agent/`、`bridge/`；
  - DST Dedicated/Runtime/LogCenter、Steam 公共层和 Steam Maintenance；
  - `frontend/`、`frontend/src/`、`features/`、`games/`、`shared/`；
  - `cmd/`、`build/`、`docs/`、`licenses/`、`scripts/windows/`。

## 架构调整

- 明确 `core -> platform/service/games/agent/bridge` 的职责边界和依赖方向。
- 明确公共能力与游戏专属能力的归档决策流程。
- 明确 Go/Vue 文件命名规则，限制万能 `Manager`、大杂烩 `utils/helpers` 继续膨胀。
- `AGENTS.md` 新增“架构文档同步”硬性规则。

## 项目检查

`check-project-layout.mjs` 增加架构文档检查，并检查：

- 当前源码文件是否在 `FILE-MAP.md` 中有记录；
- 当前源码目录是否在 `DIRECTORY-MAP.md` 中有职责说明；
- 构建产物、node_modules、运行日志等生成目录不参与源码地图门禁。

这样以后新增文件/目录却忘记更新架构文档时，项目检查会直接失败。

## 优化

- `docs/PROJECT-STRUCTURE.md` 增加详细源码导航入口。
- 前端 `features/README.md` 明确每个平台功能目录的职责和当前状态。
- 复杂 DST/Steam 模块增加模块就地说明，减少开发时跨目录猜测。

## 测试

交付前执行：

- 项目骨架/版本一致性检查；
- `go test ./internal/...`；
- `go vet ./internal/...`；
- 文件/目录地图覆盖检查。

## 已知问题

- 0.1.40 主要是文档和维护性基线，不代表各占位模块已经完成真实业务。
- 完整 Windows `pnpm + Vite + Wails` 正式构建仍需在 Windows 开发机实机验收。

## 下一阶段

继续按照 Roadmap 逐项把平台占位能力替换为真实实现，并保持每次新增模块同步架构地图。


---

## AI-Game-Manager-Panel 0.1.39

### Release Notes

<!-- historical source: docs/releases/0.1.39.md -->

## 版本定位

平台架构骨架版本。该版本重点不是一次完成 GSM 的所有功能，而是确保未来每个功能都有正确目录、路由、职责和开发计划。

## 新增

- 根目录 `AGENTS.md` 中文强制开发规范。
- GSM 源码学习记录与功能映射。
- 平台级功能占位：文件、终端、任务、备份、网络、插件、用户权限、小鱼、开发者工具。
- Go Core 公共 Service / Platform / Agent 目录占位。
- AI Agent 的 Tool / Permission / Planner / Executor 基础模型。
- `docs/roadmap/` 完整 Phase 占位。
- Windows `Bonfire-Setup.exe` Inno Setup 工程骨架。

## 架构优化

- 明确平台能力与游戏适配器边界。
- 明确游戏库与服务器实例的不同职责。
- 明确日志、文件、终端、备份、任务必须公共化。
- 禁止继续扩大巨型 Workspace/Manager 文件。

## AI

0.1.39 只建立 小鱼和受控工具架构，不调用任何真实模型 API。

## 已知事项

- 新增占位页面暂不包含真实业务。
- 安装器工程已建立，但尚未并入正式双构建流水线。
- 各 Phase 必须逐个完成后再移除“规划中”状态。

## 下一阶段

优先完成全局日志中心，再进入统一实例管理。


---

## AI-Game-Manager-Panel 0.1.38

### Release Notes

<!-- historical source: docs/releases/0.1.38.md -->

## 本次目标

对“游戏库 → 饥荒联机版”工作区做一次稳定性专项整理，重点解决：

1. Master/Caves 实际已经成功启动，但日志中心仍显示防火墙警告、让用户误认为开服失败；
2. 日志/控制台区域周期性闪烁；
3. 日志中心、Runtime、Preflight、Token 等逻辑集中在单个巨大 Vue 文件，耦合过高；
4. 后台轮询和日志读取存在无变化也重复请求的问题；
5. 日志导出写入 Windows 用户 Downloads，而不是 Bonfire 项目目录；
6. 结构化日志存在字符串误分类。

## 已修复

### 1. 修复“成功开服却显示故障”的诊断冲突

DST 会输出：

```text
[Warning] Could not confirm port 10999 is open in the firewall.
```

该提示只表示 DST 无法通过自身探测确认 Windows 防火墙规则，**并不等价于端口绑定失败**。

当同一会话后续已经出现：

```text
Online Server Started on port: ...
[Shard] Shard server started on port: ...
Server registered via geo DNS ...
Secondary Caves ... ready!
World ... is now connected
Sim paused / Sim unpaused
```

Bonfire 现在会把前面的防火墙提示标记为“已被后续成功证据覆盖”，不再保留为当前 WARNING/故障。

### 2. Master 与 Caves 使用不同健康判定

Master 健康链路：

```text
Steam → Token → 在线端口监听 → Shard → Klei/Geo DNS 注册 → World Ready
```

Caves 健康链路：

```text
Steam → Token → 在线端口监听 → 连接 Master → Secondary Ready → World Ready
```

Caves 不再因为没有单独出现 Master 的 Geo DNS 注册标记而显示“未确认”。

### 3. 启动健康与运行期错误分离

服务器已经 World Ready 后，如果管理员输入了错误控制台命令，或 Mod 后续出现普通运行时错误，不再把整次“启动链路”反向改成失败。

真正会阻止启动健康的仍包括：Token 冲突/无效、运行库缺失、端口绑定失败、Server failed to start、权限错误、worldgen 启动错误等。

### 4. 修复结构化控制台误分类

旧逻辑只要一行包含 `disconnected` 就归类为“玩家”，导致：

```text
keep_disconnected_tiles
no_wormholes_to_disconnected_tiles
```

这种世界设置也显示成玩家事件。

现在只匹配明确的玩家连接/断开语句，例如 `client disconnected`、`player disconnected`、玩家认证、加入、Spawn 等。

## 稳定性与性能优化

### 5. 日志中心从 DstWorkspace 拆分

新增：

```text
frontend/src/games/steam/dst/components/DstLogCenter.vue
```

日志会话、分页、搜索、诊断、导出全部由独立组件维护；`DstWorkspace.vue` 只负责工作区导航与 Runtime 编排。

这样 Token/Network/Preflight/Runtime 的刷新不会再无意义地让日志中心整棵响应式状态重算。

### 6. Runtime 轮询改为自适应、无重叠

移除固定 `setInterval(1200)`。

现在：

- 仅“概览与控制台”页面轮询；
- 离开概览后完全停止 Runtime 轮询；
- 运行中约 1.2 秒一次；
- 停止状态约 3 秒一次；
- 应用窗口不可见时降到约 5 秒；
- 上一次请求完成后才调度下一次，不产生重叠请求。

### 7. 相同 Runtime Snapshot 不再触发大区域刷新

Go 后端每次返回新 JSON 对象，但内容经常完全相同。0.1.38 对 Snapshot 做稳定签名，内容没有变化时不重新赋值 Master/Caves/Cluster 响应式对象，减少 WebView2 重绘。

### 8. 实时日志按 logCursor 增量读取

只有后端 `logCursor` 真正前进时才请求新的日志批次；无新日志时不再每轮询一次就调用日志 API。

检测到新的 `logSessionId` 会自动清空旧窗口，避免服务器重启后新旧会话日志混在一起。

### 9. 修复 WebView2 日志区域闪烁风险

清理了日志中心重复的 `.dst-log-diagnostics` CSS 定义，并移除：

```css
content-visibility: auto;
contain: content;
contain-intrinsic-size: ...;
```

这些优化在 Chromium 普通网页常见，但在 Wails WebView2 的大型黑色滚动日志区可能导致重新栅格化/闪屏。

当前日志窗口本身已有 1500/2000 行上限，因此改用有界 DOM、稳定 scrollbar gutter 和固定诊断区域，优先保证桌面端稳定性。

## 日志目录调整

### 10. 所有 Bonfire 日志统一进入根目录 `log/`

开发/源码目录运行：

```text
Bonfire-0.1.38/
└─ log/
   ├─ bonfire.log
   ├─ dst/
   │  └─ <Cluster>/<Master|Caves>/...
   ├─ 手动导出的单次日志.log
   └─ Bonfire_日志_<Cluster>_<时间>.zip
```

正式便携 EXE 如果脱离源码树运行，则使用 `Bonfire.exe` 所在目录下的 `log/`。

Desktop 与本机 Web 管理端的“导出日志/诊断包”现在都调用 Go 后端保存到该目录，不再默认写入 `C:\Users\...\Downloads`。

`cluster_token.txt`、adminlist、blocklist、whitelist 仍禁止进入诊断包。

## 代码结构整理

新增：

```text
internal/platform/files/apppath/
```

统一解析 Bonfire 运行根目录；日志模块不再自己推断 Downloads 路径。

游戏库/DST 当前职责：

```text
DstWorkspace.vue              工作区导航 + Runtime 编排
components/DstLogCenter.vue   历史日志 / 诊断 / 导出
components/NetworkConfig.vue  网络与 Shard 端口
components/TokenManager.vue   Klei Token
components/PreflightPanel.vue 启动前检查
internal/games/dst/logcenter  持久化、诊断、导出
internal/games/dst/runtime    单 Shard 进程状态
internal/games/dst/service/runtime   Cluster 业务编排
```

## 已验证

代码侧要求：

- `go test ./internal/...`
- `go vet ./internal/...`
- logcenter 健康判定回归测试；
- Master 防火墙提示被后续成功证据覆盖测试；
- Caves 不依赖 Geo DNS 标记的健康测试；
- 日志导出路径固定到配置的项目 `log/` 测试；
- Windows amd64 关键包交叉编译；
- Vue/TypeScript 正式构建（可执行环境具备依赖时）。

## 从用户实机日志确认的事实

用户提供的 0.1.37 Master/Caves 日志已经出现：

- Master `Online Server Started on port: 10999`；
- Master Shard `10888` 启动；
- Geo DNS 注册成功；
- Caves `Online Server Started on port: 11000`；
- Caves 连接 Master；
- `secondary shard is now ready!`；
- Master/Caves `World ... is now connected`。

因此该次会话的主要问题是 Bonfire 的诊断/UI 表达不真实，而不是 Dedicated Server 没有建立成功。

## 待用户实机验证

1. 0.1.38 Windows 正式构建 10/10；
2. Wails 桌面日志中心长时间停留是否彻底消除闪屏；
3. 一键启动 Master + Caves 后两张卡都保持 World Ready；
4. 玩家进入地面 → 下洞穴 → 返回地面；
5. “导出日志”和“生成诊断包”实际路径是否位于项目根目录 `log/`。


---

## AI-Game-Manager-Panel 0.1.37

### Hotfix

<!-- historical source: docs/BUILD-HOTFIX-0.1.37.md -->

## 问题

0.1.36 新增 `network` 工作区后，`WorkspaceSection` 已包含 `network`，但 `DstWorkspace.vue` 中的 `placeholderCopy` 仍使用“排除若干真实页面后，其余页面必须全部提供占位文案”的 `Record<Exclude<...>>` 类型。

因此正式构建执行：

```text
vue-tsc --noEmit && vite build
```

会在 `DstWorkspace.vue` 报 TS2741：`network` 缺失，导致双端构建在前端类型检查阶段终止。

## 修复

`placeholderCopy` 改为：

```ts
Partial<Record<WorkspaceSection, { title: string; description: string }>>
```

它只描述真正尚未实现的占位页面。`network`、`tokens`、`saves`、`install` 等真实页面不再需要进入占位映射，也不再依赖手工维护排除列表。

`activePlaceholder` 同时改为安全可空查找：

```ts
const activePlaceholder = computed(() => placeholderCopy[section.value] ?? null)
```

## 影响

- 修复 0.1.36 的 TS2741 正式构建失败。
- 不改变 0.1.36 的 DST 网络与 Shard 配置业务行为。
- 后续新增真实 Workspace 页面时，不会因为遗漏占位映射再次触发同类构建错误。
