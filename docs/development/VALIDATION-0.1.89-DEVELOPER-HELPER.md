# AGMP 0.1.89 · Windows Developer / Publisher Helper 修复验证

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
