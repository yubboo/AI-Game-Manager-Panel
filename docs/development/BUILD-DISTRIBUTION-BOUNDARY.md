# AGMP 开发、发行构建与生产环境边界

## 产品原则

AI Game Manager Panel 是唯一产品。AI Agent 是内置核心大脑，内部 Rust/Go 二进制只是实现细节，不是用户独立产品。

## 三类环境

```text
源码开发环境
  ├─ Go / Node / pnpm
  └─ Rust / Cargo / MSVC：仅开发 AI Core Rust 实现时按需准备

发行构建机 / CI
  ├─ 完整编译工具链
  ├─ Test / Vet / Frontend / Rust / Installer / Safety Gates
  └─ 生成完整 AGMP 发行端
          ↓
普通用户生产环境
  ├─ Windows：Setup / Portable
  └─ Web：完整 Web 发行包
```

生产环境禁止现场编译，正式用户产物不得要求、下载或安装 Rust、Cargo、Rustup、MSVC/Visual Studio Build Tools、Go、Node.js、pnpm、Wails CLI。

## 内部 AI Core

发布构建阶段可以生成 `AI-Game-Manager-XiaoYu.exe` 作为内部 Runtime，但它只能被完整 AGMP 包携带：

- 不创建桌面/开始菜单快捷方式；
- 不要求用户手工启动；
- 不作为普通用户独立 Release Asset；
- Wails 安装包放入 `internal/xiaoyu` 内部目录；
- Electron 放入应用 `resources` 内部目录；
- `build/release` 只发布完整产品包，原始内部二进制留在 `build/work`。

## 基础开发初始化

菜单 1 只准备 Go/Frontend 基础环境，不自动安装 Rust/MSVC。开发 AI Core 或本机从源码生成 Release 时才进入对应工具链准备流程。
