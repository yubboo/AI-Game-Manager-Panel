# AGMP 0.1.85 验证记录

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
