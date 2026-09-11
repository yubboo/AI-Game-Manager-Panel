# AGMP 0.1.89 · Environment Manager v2 阶段验证记录

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
