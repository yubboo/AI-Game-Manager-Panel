# AGMP 0.1.88 验证记录

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
