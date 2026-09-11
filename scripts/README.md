# AI Game Manager Panel Scripts 0.1.88

Windows 开发助手自 0.1.64 起固定为：**一个 ASCII-safe BAT 启动器 + PowerShell Task Runner**。

```text
AI-Game-Manager-Panel.bat
└─ scripts/windows/AIGameManagerPanel.ps1        # 彩色中文菜单 + 任务路由
   ├─ lib/Common.ps1                  # UTF-8、进程、路径、日志
   ├─ lib/Toolchain.ps1               # Go / Node / pnpm / Wails / Inno
   ├─ lib/Rust.ps1                    # Rustup / Cargo / Agent Runtime
   ├─ lib/Dependencies.ps1            # Frontend / Electron / Runtime
   ├─ lib/Checks.ps1                  # Quick / Full Gate
   ├─ lib/Wails.ps1                   # Wails Dev / Release
   ├─ lib/Electron.ps1                # Electron Dev / Release
   └─ tasks/Tasks.ps1                 # 初始化、Web、Linux、预览、维护、一键发布
```

## Windows 硬规则

- `AI-Game-Manager-Panel.bat`：ASCII + CRLF，不写中文，只负责 `chcp 65001` 和启动 PowerShell。
- `scripts/windows/**/*.ps1`：UTF-8 with BOM + CRLF。
- `scripts/windows/` 下禁止出现其他 `.bat`。
- 禁止 `Invoke-Expression`、命令字符串拼接、Node `shell:true`。
- 外部程序使用 PowerShell 参数数组直接调用，必须支持中文路径与空格路径。
- 任何 `�`、非法 UTF-8、菜单乱码都属于 Gate FAIL。
- `scripts/common/check-windows-helper.mjs` 会检查编码、菜单 1-10、Frontend 相对导入和禁用模式。
- `scripts/windows/AIGameManagerPanel.ps1 -SelfTest -DryRun` 会遍历菜单 1-10 的全部顶层入口。

## 当前主菜单

1. 初始化基础开发环境
2. 开发模式（Wails / Electron / Web / 小鱼核心 Rust Runtime）
3. 项目检查（快速 / 完整）
4. Wails Windows Release
5. Electron Windows Release
6. Linux Server Release
7. 预览已构建版本
8. 状态与环境诊断
9. 清理与重置
10. 一键发布（Wails / Electron / 全部）

Linux 服务器正式构建入口仍为 `bash scripts/build_linux.sh`。

## GitHub Safety Gate

`node scripts/common/check-github-safety.mjs` 会校验 `.gitignore` 并扫描潜在 Git 提交文件。Windows 菜单 3 的快速/完整检查和 Linux Release Gate 都会自动执行。

## 源码开发许可证模式

Wails/Web/Electron 的开发入口会自动使用 `agmp_dev_license` Go Build Tag，便于在没有真实 CDK 的情况下调试全部功能；正式 Release 不使用该 Tag。

## 0.1.70 发行密钥门禁

正式 Release 额外执行 `node scripts/common/check-release-key.mjs`。第一次正式发布前使用 `scripts/tools/license/AGMP-License-Admin.bat` 在仓库外生成发行私钥并把公开公钥加入密钥环。快速开发检查使用 `--allow-unconfigured`，不会阻断纯源码开发。

## 0.1.72 Windows Installer

Wails 正式 Release 使用内置简体中文 Inno Setup 脚本与强制 EULA；Electron 仅作为兼容发行。

## 0.1.73 Updater

Wails Release 额外生成 Setup 的 `.sha256` companion asset。GitHub Release 必须同时上传 Setup.exe 与同名 `.sha256`，供 Windows 客户端自动更新校验。

## 0.1.83 Module Gate / AI Approval

`Invoke-ProjectLayoutGate` 现在同时执行 `check-modules.mjs`。该 Gate 固定模块归属、短命名、AI 只能通过 Tool/Service/API 调用业务能力，并要求 `shell.exec` 作为 XiaoYu 的受控通用后备能力存在；结构化 Domain Tool 仍优先，实际执行由后端 RBAC、Sandbox 与三种审批模式决定。`process.run` 仅保留为人工/兼容别名。审批模式由后端可信状态决定，前端只负责请求切换。

## 0.1.79 统一产品 / 开发生产边界

开发助手只服务源码开发和发行构建；生产用户不运行 BAT/PowerShell Helper。小鱼是 AGMP 内置核心大脑，Rust Runtime 只是内部实现。Wails/Electron/Web/未来 Rust Native 都是完整 AGMP 的不同运行端。基础初始化不安装 Rust/MSVC；只有开发 AI Core Rust 实现或从源码打 Release 时按需准备。`build/release` 禁止单独发布 Agent Runtime。

## 0.1.77 MSVC 精简安装与存储位置选择

Rust/MSVC 首次准备会先显示组件与空间说明，再让用户选择微软默认路径或自定义磁盘。新安装按 Rust 官方最小前置只添加 `Microsoft.VisualStudio.Component.VC.Tools.x86.x64` 与 `Microsoft.VisualStudio.Component.Windows11SDK.22621`；不安装完整 VCTools workload，也不使用 `--includeRecommended`。自定义模式同时设置 Build Tools、安装包缓存与 Shared 路径，并在安装前显示目标盘剩余空间。

## 0.1.76 MSVC Bootstrapper 缓存与无 winget 安装

Windows Rust/MSVC 预检现在独立于 winget：缺少 C++ Build Tools 时仍会询问 Y/N；确认后优先复用本地已签名的 `vs_BuildTools.exe`，否则从微软官方 `aka.ms/vs/17/release/vs_buildtools.exe` 下载到 AGMP 专用安装器缓存。Build Tools 已存在时只补齐/加载 C++ 工作负载，不重复下载安装。

## 0.1.75 Windows Rust/MSVC 预检

Windows Rust target 使用 MSVC ABI。开发助手现在不仅检查 Rustup/Cargo，也检查 `link.exe`、Visual Studio Build Tools C++ workload，并能载入 VsDevCmd/vcvars 环境；未安装时可经用户确认后使用 winget 准备。

## 0.1.74 小鱼核心 Runtime

小鱼从 0.1.74 起采用 Agent-first 架构：小鱼核心 Runtime 负责 JSON-RPC、Tool Registry、审批策略和受控命令执行；Go Core 继续负责游戏服务器业务；Vue 负责桌面/Web 交互。开发菜单保留 AI Core Rust Runtime / CLI 调试入口，仅供源码开发。


## 0.1.88 XiaoYu Harness / Multi-Client Gates

项目检查新增并强制执行：`check-xiaoyu-harness.mjs`、`check-xiaoyu-model-center.mjs`、`check-xiaoyu-intelligence.mjs`、`check-xiaoyu-agent-runtime.mjs`、`check-multi-client-ai.mjs`、`check-headless-xiaoyu.mjs`。GitHub Actions 还包含独立 **Linux Headless + Web + XiaoYu** 作业，真实构建 Rust XiaoYu、Vue Web 和 Headless Go Core。Linux/Docker 发行缺少 XiaoYu Runtime 时禁止出包。

## 0.1.85 Shared Runtime Gate

0.1.85 起 `internal/platform/runtime` 是唯一子进程/stdio 边界。Module Gate 会拒绝业务域新增 `exec.Command/CommandContext` 或直接创建 stdin/stdout/stderr pipe。长生命周期使用 Session，一次性内部调用使用 Run，非交互外部启动使用 StartDetached。
