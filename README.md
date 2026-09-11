# AI游戏管理器面板（AI Game Manager Panel）

AI游戏管理器面板是一个面向 Windows / Linux 的多游戏服务器部署、管理与 AI 辅助平台。全局产品品牌从 0.1.67 起统一为 **AI游戏管理器面板 / AI Game Manager Panel**。

> 旧项目品牌只保留在历史版本记录中；当前产品、模块和用户可见命名统一使用 **AGMP / XiaoYu**。

当前版本：**0.2.6**  
目标仓库：`https://github.com/yubboo/AI-Game-Manager-Panel.git`

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



## 0.2.6 Windows 启动 / 源码同步 / 命名规范

0.2.6 在 0.2.4 完整源码保护基础上继续修复 Windows 源码工作流：根 `AI-Game-Manager-Panel.bat` 在关键 PowerShell 入口缺失时会明确报错并暂停，不再双击后一闪而过；`AGMP-GitHub.bat` 继续保持 ASCII + CRLF + 无 BOM。

新增 `AGMP-Sync.bat + sync-agmp.ps1`。推荐把新版源码完整解压到临时目录后运行 `AGMP-Sync.bat`，由 `robocopy` 稳定同步到 `H:\一键部署\AI-Game-Manager-Panel`，保留目标 `.git` 和未跟踪的 runtime/实例/备份/日志等本机数据，同时清理新版已经删除的旧 Git 跟踪源码。这样不再依赖 Windows Explorer 拖拽覆盖数百个文件。

新增 [`docs/NAMING-CONVENTIONS.md`](docs/NAMING-CONVENTIONS.md) 与 Naming Gate：普通源码文件名默认上限 40 字符、测试文件 48 字符；仓库相对路径超过 180 字符告警、超过 220 字符直接失败。目录本身视为命名空间，禁止为了“描述完整”重复父目录语义。源码包继续统一为 `agmp-<version>.zip`。

GitHub/本地检查新增 `check-source-tree.mjs` 与 `check-naming.mjs`。一键推送前会同时检查源码完整性、关键删除、命名路径、安全凭据与 GitHub Safety Gate。

版本历史统一维护在 [`docs/PROJECT-HISTORY.md`](docs/PROJECT-HISTORY.md)。XiaoYu Agent Runtime 当前设计说明见 [`docs/development/XIAOYU-AGENT-RUNTIME.md`](docs/development/XIAOYU-AGENT-RUNTIME.md)。

## Electron 与 Wails 体积

Electron 必须携带 Chromium + Node.js Runtime，因此安装包天然会比 Wails 大。0.1.67 已做安全可维护的瘦身，但不会为了追求极端体积使用 UPX 或删除 Electron 必需资源。

如果优先考虑 **体积小**，使用 Wails Release；如果优先考虑 **Electron 运行一致性与兼容性**，使用 Electron Release。

## GitHub 安全

公开仓库只能提交源码与许可证公钥。禁止提交真实许可证私钥、CDK/BFLC/activation、管理员账户与安全密钥、DST/Steam Token、运行数据、日志、构建产物、`.env` 等敏感内容。

本地 `项目检查` 会运行 GitHub Safety Gate；GitHub Actions 也会再次检查。详细规则见 `docs/development/GITHUB-SAFETY.md`。
