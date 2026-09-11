# AI游戏管理器面板（AI Game Manager Panel）

AI游戏管理器面板是一个面向 Windows / Linux 的多游戏服务器部署、管理与 AI 辅助平台。全局产品品牌从 0.1.67 起统一为 **AI游戏管理器面板 / AI Game Manager Panel**。

> 旧项目品牌只保留在历史版本记录中；当前产品、模块和用户可见命名统一使用 **AGMP / XiaoYu**。

当前版本：**0.2.2**  
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



## 0.2.2 XiaoYu Agent Runtime / Guided Autonomy

0.2.2 开始把 XiaoYu 从“只能调用少量领域按钮的模型前端”重构成能力更完整的持续 Agent：用户配置的模型提供通用智能，Expert / Skill / Memory / Experience 负责优先专业调教；结构化 Domain Tool 仍优先，但缺少专用 Tool 时可以继续使用工作区文件能力与受控 `shell.exec`。实际副作用仍由 Host 的 RBAC、Step-up、Sandbox 与“请求批准 / 帮我批准 / 完全访问”三种审批模式决定。

本版同时加入 Runtime resolve/default/remove 闭环、Memory relevance ranking、Observation context compaction、Tool guidance tier，并修复 DeepSeek thinking 与 `tool_choice` 的原生兼容问题。架构说明见 [`docs/development/XIAOYU-AGENT-RUNTIME.md`](docs/development/XIAOYU-AGENT-RUNTIME.md)。

## 0.2.0 XiaoYu Native Model Harness

0.2.0 将 XiaoYu 模型主干升级为 Provider-native Harness：OpenAI 官方使用 Responses，DeepSeek 保留 thinking/reasoning_content，Claude 保留 thinking signature，Gemini 保留 thoughtSignature；兼容网关保持 OpenAI-compatible fallback。Provider 原生 replay 按 Run 隔离，Vision 图片不会进入普通 Run JSON/Trace；AGMP Skill / Expert / Memory / Tool / Approval 只做专业增强与执行边界，不替代厂商模型本身的推理能力。

## 0.1.100 Workbench Sidebar Snap Collapse

左侧一级导航拖拽体验改为 Codex 风格的“可扩宽 + 临界点吸附收起”：左栏最大宽度提高到 520px；向左拖到最小宽度后继续轻拉会自动吸附折叠，并使用稳定五轨 Grid 做平滑过渡；拖拽过程中继续向右可越过回弹阈值重新展开。左右栏仍保持独立，不互相改写宽度。

## 0.1.98 XiaoYu Steerable Thread + Approval Hard Gate

0.1.98 按成熟 Agent Harness 的控制主干重做三件基础能力：审批必须由 Host 硬门禁，Run 执行中允许用户实时插话并触发同一 Run 重新规划，同一工作台 Session 由服务端提供有限 Thread Context，让“改回去 / 继续刚才”能理解刚刚发生的 Tool 行为。`settings.theme.set` 被修正为真正的 `modify` 风险，并返回修改前主题供回滚语义使用。详见 `docs/development/XIAOYU-AGENT-SPINE.md`。

## 0.1.96 XiaoYu Agent Kernel v2

0.1.96 开始把小鱼从“会调用 Tool 的聊天 AI”升级为 Goal-first Agent：Rust Brain 固化 System Core v2，Host Frame 提供 Capability/当前 UI 上下文，Run 增加 Understanding / Planning / Executing / Verifying / Recovering 阶段和可见 Decision Summary；后台 Run 会实时发布步骤进度，不再长期停在 step 0。新增 `settings.get` / `settings.theme.set`，小鱼可以直接修改主题而不是只会打开设置页。

## 0.1.95 Split Pane + 小鱼对话工作台

0.1.95 修复三栏拖拽耦合、左栏过窄变形和拖拽卡顿，并将小鱼中央区从“仪表盘卡片”收敛为真正的聊天工作台。左右栏拖拽互不影响；小鱼回复进入同一对话流，Enter 发送、Shift+Enter 换行。


## 0.1.94 三栏初始比例调整

- 桌面大屏首次布局调整为约 **10% / 60% / 30%**：左侧一级导航更窄，中央主工作区保持主体，右侧上下文/二级菜单获得更充足空间。
- 左右栏继续支持拖拽与收起；窄窗口仍优先隐藏右侧栏。
- 布局持久化键升级为 `agmp.workbench.layout.v3`，升级后会应用新的初始比例，不继续沿用 0.1.92 的旧侧栏宽度。

## 0.1.88 双大脑 + 多端 XiaoYu

0.1.88 在已冻结的 Shared Runtime 上完成 XiaoYu Harness、模型中心和多端 AI 能力收口。产品模型固定为 **Human + XiaoYu，两个大脑、同一副 AGMP 身体**：人工按钮与 XiaoYu Tool 必须复用同一领域能力，人工接管永远优先。Web/Linux/Docker 是完整产品端，不是桌面阉割版；关闭浏览器只断开事件订阅，不取消服务器上的 XiaoYu Run。

- 系统设置新增“模型管理”，支持云模型、本地 Ollama/LM Studio、自定义 OpenAI-Compatible/第三方中转，并使用独立 Secret Vault 保存 API Key。
- XiaoYu Agent Harness 已具备 Plugin Kernel、Capability Discovery、Agent Loop、预算/死循环保护、Observation、等待审批/用户、暂停/接管/恢复和服务器端 Trace/Event Stream。
- Linux Server、Docker、Linux 发行链强制携带预编译 XiaoYu Runtime；缺少 XiaoYu 时禁止出包。
- 多用户 Web 的 XiaoYu Run 按发起人隔离；Owner/Admin 可全局监督，普通 Operator 只能查看和控制自己的 Run。
- DSH Tool Bridge 默认在受限 Node Permission 子进程运行，只能读取插件自身目录，禁止文件写入和 child_process；插件真实路径会做 symlink 越界检查。
- Rust `xiaoyu-core` 仍是唯一 Brain；Go Host 只负责模型通信、Run 驱动、权限/审批、Tool 分发和业务执行，不允许复制 Planner/Memory/语义恢复策略。

## 0.1.85 Shared Runtime Hardening

0.1.85 在 0.1.83 已冻结的领域骨架上继续完成底层运行时：统一 Session/Run/StartDetached 进程入口，区分 stdout/stderr，串行 stdin，增加命名 Session Manager、状态快照、环境变量 overlay、输出上限、超时取消和 Unix 进程组终止。XiaoYu Runtime、更新安装器及系统打开器不再各自直接创建进程。

## 0.1.83 AGMP / XiaoYu 架构收口

- 产品缩写统一为 **AGMP**；内置超级大脑统一为 **小鱼 / XiaoYu / xiaoyu**。
- Rust 核心固定为 `xiaoyu-core + xiaoyu-protocol`，协议固定为 `xiaoyu.v1`；Rust 是小鱼唯一 Brain。
- Go 后端冻结为 `xiaoyu / games / server / ops / deploy / system / platform` 七个主要领域，禁止恢复 `internal/service` / `internal/core` 大平铺。
- 小鱼只能通过领域 Tool / Service / API 调用系统能力；手动终端与小鱼共享安全边界，裸 Shell 不能绕过模块能力。
- 许可证 KeyID 使用 AGMP 新命名，同时保留历史证书验证兼容；MachineCode 派生协议不随品牌命名变化。
- Module Gate 新增一级领域、旧平铺目录和旧产品缩写残留检查，避免后续开发重新把骨架弄散。
- 通用 `platform/runtime` 已落地 Process/Terminal Session，DST 已实际复用；持久 PTY、会话恢复和更多游戏适配仍按后续阶段开发。
- 本版主要完成架构、命名和安全边界，不把尚未实现的 Planner、Memory/Workflow、多游戏完整控制等能力伪装为完成。

## 0.1.79 统一 AI-First 产品架构

项目最高级产品/工程规则已冻结在 [`docs/development/PROJECT-RULES.md`](docs/development/PROJECT-RULES.md) 和根目录 `AGENTS.md`。后续人工或 AI 开发必须先遵守这些边界；当前不做总体目录重排，优先把核心能力做稳、做能用、做安全、做省心。

- **AI Game Manager Panel 是唯一产品主体**；小鱼（XiaoYu）是 AGMP 内置的核心大脑与智能伙伴，不是外挂产品。
- **AI 为主、手动为辅**：小鱼负责理解/规划/执行/验证，传统页面和终端负责人工确认、精细控制与故障兜底。
- Web / Wails / Electron / Rust Native 定义为同一 AGMP 的不同运行端；当前 Rust Native 仍是规划目标，不伪装为已完成。
- 开发环境与生产环境彻底分离：开发/发行构建机可按需使用 Rust/MSVC；普通用户只运行预编译好的完整 AGMP，不安装任何编译工具链。
- Wails Setup/Portable 中的 Rust Runtime 只作为 AGMP 内部 AI Core 组件存在，不创建独立入口；`build/release` 不再单独发布 Agent 二进制。
- 当前阶段不重排总体源码布局，先完成 Agent Loop、Provider、Tool/审批、PTY/会话和游戏核心能力。

## 0.1.77 Windows Rust/MSVC 精简安装

- 首次缺少 MSVC 时先显示官方容量范围、AGMP 精简组件与空间建议，不再只写“体积较大”。
- 按 Rust 官方最小前置只安装 **MSVC x64/x86 Build Tools + Windows 11 SDK 22621**；不安装完整 `Microsoft.VisualStudio.Workload.VCTools`，也不使用 `--includeRecommended`。
- 用户可选择微软默认安装位置，或输入自定义根目录/磁盘；自定义模式分别设置 BuildTools、PackageCache、Shared 与 bootstrapper 缓存目录。
- 安装前显示目标盘剩余空间；目标盘低于 8 GB 时再次确认。即使安装到其他盘，仍提示系统盘为 Windows SDK/系统共享组件保留空间。
- 已安装 Build Tools 时只补缺少的必需组件，不迁移已有 Visual Studio 全局缓存，也不重复安装。

## 0.1.76 Windows Rust Bootstrap Hotfix

- 修复源码包遗漏 `runtime/README.md` 导致初始化项目结构 Gate 直接失败。
- MSVC Build Tools 安装不再依赖 `winget`：无 winget 也会先询问 Y/N，再从微软官方 bootstrapper 下载。
- Build Tools 安装器缓存到 `%LOCALAPPDATA%\AI-Game-Manager-Panel\DevTools\Installers`；缓存有效时直接复用，不重复下载。
- 已安装 Visual Studio/Build Tools 时优先载入 `VsDevCmd.bat` / `vcvars64.bat`，不会重复安装。
- 对缓存/下载的 `vs_BuildTools.exe` 做 Microsoft Authenticode 签名检查后才允许执行。

## 0.1.75 Windows Rust Toolchain Hotfix

- 修复 Rust 已安装但缺少 `link.exe` 时，Wails/Electron Release 到最后才失败的问题。
- 初始化开发环境现在同时检查 Rust/Cargo 与 Microsoft Visual C++ Build Tools。
- 若已安装 Build Tools，会自动通过 `vswhere` / `VsDevCmd.bat` 恢复 MSVC 环境。
- 若未安装，会在用户确认后通过 winget 安装 Visual Studio 2022 Build Tools + C++ workload。
- Wails/Electron Release 在耗时构建前先执行 Rust/MSVC 预检。

## 0.1.74 关键改动

- AI 交互正式调整为 **Agent-first**：自然语言是主入口，传统页面和手动终端作为辅助与兜底。
- 新增 Rust workspace 与 AI Core Runtime/CLI；底层进程隔离属于内部实现，不代表独立用户产品。
- Rust Runtime 建立 `xiaoyu.v1` JSON-RPC 协议、Tool Registry、会话基础和 Allow/Confirm/Deny 审批策略。
- 首批工具：`system.info`、`fs.list`、`fs.read`、`process.run`。
- Go Core 提供小鱼 Runtime bridge，Wails / Web / Electron 共用同一内置小鱼 Core 执行能力，并继续由 Go License Feature Gate 和操作审计约束。
- 小鱼和终端开始使用真实 Rust Runtime；0.1.74 暂不宣称已具备完整 Codex 式自动规划循环，模型 Provider、流式 Planner/Executor 和持久 PTY 将继续开发。
- Windows Wails 正式安装包开始携带 小鱼核心 Runtime；Electron 保持兼容发行。

## 0.1.73 关键改动

- Windows 安装器补齐已有目录/不存在目录中文提示，并识别旧版本升级。
- 设置中心新增“更新与升级”，从官方 GitHub Releases 检查 stable 新版本。
- 用户点击后下载 Setup + SHA256，校验通过才启动升级安装器。
- 固定 AppId、复用安装目录；更新程序文件时保留 runtime 用户数据。
- 0.1.72 -> 0.1.73 为首次手动迁移；0.1.73 起支持后续应用内升级。

## 0.1.72 关键改动

- Windows 官方桌面发行：Wails + Inno Setup。Electron 作为兼容发行保留。
- Setup.exe 安装向导由项目内置简体中文文案，不再依赖额外 Inno 中文语言包。
- 新增强制软件许可协议页；用户必须选择“我接受本协议”才能继续安装。
- 优化安装目录、开始菜单、桌面快捷方式、升级复用设置、安装日志和卸载数据保留提示。
- 正式 Setup 文件使用带版本号命名，并在 Release 完成后输出文件大小。

## 0.1.70 关键改动

- 完成 **BFLC2 首次签发 → 回验 → 导入 → 持久化 → Core 重启恢复** 的离线授权闭环。
- BFLC2 签发完成后自动使用公开密钥环回验；签名、BFM、BFID、有效期或 IssuerKeyID 任一不一致都会停止交付。
- `issuerKeyId` 从展示字段升级为安全校验字段：必须与真正完成 Ed25519 验签的公钥 KeyID 一致。
- `activation.json` 改为临时文件写入 + fsync + 原子替换，避免程序异常退出留下半截激活记录。
- 设置中心可以直接选择发行工具生成的 `.bflc` 文件，也继续支持粘贴完整 BFLC2。
- 许可证发行控制台新增 `4. 同步本机发行公钥`，解压新源码后可从仓库外 `ReleaseKeys` 恢复同一 Active Key，不必重新生成私钥。
- 许可证发行控制台新增 `5. 验证 BFLC2`，可独立核验签名、机器码、安装 ID 与有效期。
- 密钥轮换加入误操作保护：新源码尚未同步 Active Key、但本机已经存在发行密钥时，菜单 1 会阻止再次生成第二把密钥。
- 冻结 MachineCode 派生协议并增加固定向量回归测试，避免未来品牌文字调整让已签发设备许可证失效。
- 明确 BFID 是“具体安装实例”标识：签发时必须从真正准备激活的 Release 程序复制 BFM/BFID，不能拿另一份源码开发目录的 BFID。

### 已经在 0.1.69 生成过发行密钥的升级方式

运行：

```text
scripts\tools\license\AGMP-License-Admin.bat
```

**优先选择 `4. 同步本机发行公钥`**。只有第一次从未生成过发行密钥，或明确要执行密钥轮换时，才选择 `1. 初始化 / 轮换发行密钥`。

## 0.1.69 关键改动

- 正式建立 **AGMP 发行密钥生命周期**：私钥只允许在项目所有者本机、源码仓库外生成；GitHub 只保存公开发行公钥环。
- 新增 `scripts/tools/license/AGMP-License-Admin.bat`，提供发行密钥初始化/轮换、KeyID/指纹查看、BFLC2 离线证书签发。
- 0.1.68 旧公钥保留为 `legacy` 验证钥；新密钥轮换时旧 active key 会变成 `retired`，历史 BFLC2 不会仅因为正常轮换就全部失效。
- BFLC2 新增 `issuerKeyId`；签发工具会检查私钥必须与源码中的 `activeKeyId` 匹配。
- Windows Wails / Electron 和 Linux Desktop / Server 正式 Release 新增强制 **Release Key Gate**。
- 设置中心授权页新增 Active 发行 KeyID、公钥 SHA-256 指纹、可信公钥数量和当前许可证签发 KeyID，只显示公开信息，不暴露私钥。

## 0.1.68 关键改动

- 修复 Windows Wails / Electron 在授权状态、登录完成后加载许可证、进入安全中心等场景中偶发黑色终端窗口一闪而过的问题。机器码改为 **直接读取 Windows Registry API**，不再启动 `reg.exe` / `cmd.exe` / PowerShell。
- MachineCode 在 Core 生命周期内缓存；InstallID 首次读取/生成后也缓存。刷新许可证状态不会重复读取系统标识，也不会重新生成安装 ID。
- 设置中心与安全中心统一复用 Pinia 中的许可证快照。页面切换不再先清空机器码/安装 ID 再显示“读取中…”，手动刷新采用保留旧值的 stale-while-revalidate 行为。
- 明确授权术语：**短 CDK = 未来在线兑换码；当前离线授权 = BFLC2 证书**。在线 CDK 输入暂时禁用，避免误以为存在一个“离线短 CDK”。
- 修复 0.1.67 品牌迁移时误把 TypeScript 类型名替换成带空格的 `AI Game Manager PanelSettings` 的问题，统一为 `AGMPSettings`。

## 0.1.67 关键改动

- 修复 Electron 一键发布最后“晋升 Release”阶段因旧 EXE 被 Windows / Defender / 正在运行的程序占用而整次失败的问题。
- Release 改成“逐文件安全发布”：目标被锁定时重试；持续被占用则保留本次新产物并生成带时间戳的 `-Rebuild-...` 文件，不再让整次发布失败。
- 最终文件统一进入 `build/release/`；临时文件统一进入 `build/work/`。
- Electron 启用 `asar`、`maximum` 压缩，只保留 `zh-CN` / `en-US` Electron 语言包；Go sidecar 使用 `-trimpath -ldflags "-s -w"` 去除调试符号。
- 全局品牌、可执行文件、安装器、Electron/Wails 配置、Go Module、GitHub Safety 文档全部切换到 AI Game Manager Panel。
- 源码开发继续使用 `agmp_dev_license` Build Tag，不需要真实离线 CDK；正式 Release 不启用该 Tag，仍执行正式许可证校验。

## Electron 与 Wails 体积

Electron 必须携带 Chromium + Node.js Runtime，因此安装包天然会比 Wails 大。0.1.67 已做安全可维护的瘦身，但不会为了追求极端体积使用 UPX 或删除 Electron 必需资源。

如果优先考虑 **体积小**，使用 Wails Release；如果优先考虑 **Electron 运行一致性与兼容性**，使用 Electron Release。

## GitHub 安全

公开仓库只能提交源码与许可证公钥。禁止提交真实许可证私钥、CDK/BFLC/activation、管理员账户与安全密钥、DST/Steam Token、运行数据、日志、构建产物、`.env` 等敏感内容。

本地 `项目检查` 会运行 GitHub Safety Gate；GitHub Actions 也会再次检查。详细规则见 `docs/development/GITHUB-SAFETY.md`。
