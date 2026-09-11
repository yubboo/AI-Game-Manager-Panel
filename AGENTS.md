# AI Game Manager Panel AI / 人工开发规范

> 本文件是 AI Game Manager Panel 的强制开发规范。人工开发、ChatGPT、Codex 或其他 AI 修改项目时都必须遵守。

## 1. 项目定位与不可破坏的产品架构

AI Game Manager Panel（AI游戏管理器面板）定位为：**现代化、智能化、AI 驱动的一键游戏服务器部署与管理平台**。

以下是项目的最高级架构约束，优先级高于局部实现便利，后续任何 AI、人工开发、重构和多端适配都不得违反：

1. **AI Game Manager Panel 是唯一产品主体。** Web、Wails、Electron、未来 Rust Native 都只是同一产品的不同构建/运行端，不得拆成彼此独立、各自复制业务的产品。
2. **小鱼是 AGMP 内置的核心大脑。** Agent 负责理解用户意图、规划、调用 Tool、请求审批、执行、验证和汇报；它不是外挂产品、不是单独面向用户下载/启动的应用。即使底层为了稳定性使用 Rust 原生模块或内部子进程，也只能视为 AGMP 的内部实现细节。
3. **两个大脑，同一副身体；AI 辅助人。** 用户/管理员是最终人为大脑，小鱼是持续在线的智能大脑。用户可以手动控制，也可以把目标交给小鱼；Human override 永远优先。两者不得各造一套业务逻辑。
4. **AI 与人工必须调用同一能力层。** Steam、DST、实例、备份、日志、文件、网络、RCON、终端、更新等能力必须复用同一 Core/Service/Tool 契约，禁止“AI 一套实现、手动 UI 再写一套实现”。
5. **多端只做适配，不做业务分叉。** Web / Wails / Electron / Rust Native 的差异只允许存在于 Shell、Bridge、平台适配和打包层；Agent 能力、业务规则、权限、审计、数据模型必须尽量共用。
6. **内部组件不得变成用户心智中的独立产品。** `AI-Game-Manager-XiaoYu.exe`、Electron Go Core 等若因进程隔离而作为内部二进制存在，只允许随完整 AGMP 产物一起打包，不创建独立快捷方式、不单独作为普通用户 Release Asset、不要求用户手工启动。

当前阶段优先级固定为：**核心能力 > 稳定可用 > 性能 > 安全 > 省心运维 > 架构美化/目录重排**。在 Agent 核心闭环和主要服务器能力稳定前，禁止为了“看起来更漂亮”大规模移动根目录或重写总体项目布局。

## 1.1 开发环境与生产环境硬边界

必须严格区分“源码开发/发行构建环境”和“普通用户生产运行环境”：

- **开发环境**：面向开发者，可包含源码、Go、Node.js、pnpm、Wails/Electron 工具链；只有开发/修改 Rust AI Core 或从源码制作正式发行包时，才按需准备 Rust/Cargo/MSVC/Windows SDK。
- **发行构建机/CI**：属于开发/发布基础设施，不等于普通用户生产环境。它负责把 Go/Rust/Frontend/Desktop 预编译成最终可分发产物，并执行测试、签名、哈希、许可证和安全 Gate。
- **生产环境**：面向普通用户，只运行已经编译好的 AGMP。Windows 用户下载完整 Setup/Portable，Web 用户运行完整 Web 发行包；生产环境**不得现场编译**，不得要求安装 Rust、Cargo、MSVC、Go、Node.js、pnpm、Wails CLI 或 Visual Studio Build Tools。
- 根目录 `AI-Game-Manager-Panel.bat` 是**源码开发/发行构建助手**，不是生产启动器。普通用户不应运行它。
- 任何正式发布目标都必须自带该端运行所需的 AGMP 内部 AI Core 组件，用户只选择“使用哪个 AGMP 端”，不选择“要不要另外安装 Agent”。


## 2. 技术栈

- 游戏与平台业务核心：Go。
- 小鱼 Intelligence Core：Rust（Brain / Planner / Workflow / Context / Memory / Tool 选择 / Result Validation）。最终 Tool 权限、审批、审计和业务执行由 Go Host 强制。
- Windows 桌面端：Wails + Vue 3 + TypeScript + Vite + Pinia；Rust 仅作为 小鱼核心 的底层实现之一，不作为独立产品暴露。
- Web 管理平台：复用同一套 Vue 前端，通过 HTTP/WebSocket Bridge 调用同一 Go Core。
- JavaScript/TypeScript 包管理只允许 **pnpm**。
- 正式用户产物按“完整 AGMP 运行端”发布：Web、Wails、Electron、未来 Rust Native。内部 Agent/Core 二进制可以存在于包内，但禁止单独作为普通用户产品发布。
- Linux 定位为一等 Headless Server Edition：完整 AGMP Web/Server + 预编译 XiaoYu Core + Model Center/Harness/Tool/审批/事件流 + `install.sh/systemd`；核心 AI 能力不得少于 Windows Desktop。

## 3. 中文规范

- UI、按钮、错误提示、README、开发文档、配置说明统一中文。
- **新增业务代码必须写必要中文注释**，重点说明边界、原因、并发/性能和安全假设。
- Steam AppID、API 字段、协议名称、Go/TypeScript 标识符可以保留英文。
- 错误信息必须告诉普通用户：发生了什么、为什么、下一步怎么做。

## 4. 版本规范

按连续小版本推进：`0.1.0 -> ... -> 0.1.100 -> 0.1.100 -> 0.2.0`。

每次交付：

- 源码：`AI-Game-Manager-Panel-<version>-source.zip`；
- 更新记录：`docs/releases/<version>.md`；
- Go / frontend / Wails / BAT / Installer 版本必须一致。

## 5. 阶段制开发

每个 Phase 必须经过：

`开发 -> 自检 -> 构建 -> 实机验证 -> 记录问题 -> 修复 -> 再验证 -> 冻结基线 -> 下一阶段`。

禁止跨阶段把多个没有验证的大功能同时并入稳定基线。占位功能必须明确显示“规划中/骨架已预留”，禁止伪装完成。

## 6. 0.1.81 冻结后的架构边界

源码业务骨架按领域聚合，禁止恢复 `internal/service` 平铺大仓库：

- `rust/crates/xiaoyu-core`：**小鱼超级大脑唯一实现**，负责理解上下文、规划、工作流、Tool 选择、智能状态机和结果验证；Brain 不直接读写业务文件、不创建 OS 进程、不执行游戏/系统 Tool。
- `rust/crates/xiaoyu-protocol`：小鱼与 Go Core 的稳定协议边界，当前协议 `xiaoyu.v1`。
- `internal/xiaoyu`：Go 侧只保留 `contract / control / runtime` 三组小鱼桥接职责；不得复制 Rust Brain，也不得承载游戏业务。
- `internal/games`：具体游戏特殊规则与 Adapter；公共能力禁止复制进游戏目录。
- `internal/server`：实例、服务器控制台、RCON、通用游戏工作区。
- `internal/ops`：文件、日志、备份、任务、计划任务等强关联运维能力。
- `internal/deploy`：部署、Steam/SteamCMD、环境检测、更新与构建。
- `internal/system`：认证、许可证、设置、节点、网络、安全、插件、通知和系统监控。
- `internal/platform`：最底层 OS/文件/网络/进程/终端/凭据/Telemetry/Steam 基础设施，不得依赖上层业务域。
- `internal/app`：应用装配与生命周期，不承载具体业务实现。
- `internal/bridge`：Wails / HTTP / WebSocket 适配，不写业务状态机。
- `frontend/src/features/xiaoyu`：小鱼用户入口；前端其余 feature 仅负责 UI 组合，业务规则仍归后端领域。

依赖方向固定为：`小鱼 -> Tool/Domain -> platform -> OS/外部程序`。`platform` 禁止反向依赖 `games/server/ops/deploy/system/xiaoyu`。

## 7. 配置规范（禁止散落硬编码）

项目可调整的平台默认值统一放在根目录 `configs/`：

- `app.json`：产品元信息；
- `ui.json`：导航和 小鱼文案；
- `ai.json`：AI 非敏感配置；
- `permissions.json`：AI 审批和权限策略；
- `paths.json`：运行目录；
- `logging.json`：日志策略；
- `games.json`：游戏模板目录；
- `modules.json`：功能模块和阶段矩阵；
- `release.json`：Windows/Linux Release 命名、架构与运行目录策略。

**API Key、Token、密码、Cookie 不允许写入 `configs/`。** 后续由系统凭据存储模块管理。

新增可调整值时优先扩展现有配置文件；禁止在多个 Vue/Go 文件里复制同一个默认值。

## 8. 复用规则

日志、备份、计划任务、终端、文件、Steam 校验、实例生命周期、权限、通知必须优先公共化。

禁止出现：

- 每个游戏复制一套 Steam 校验器；
- Desktop / Web 各写一套业务；
- 每个游戏复制一套日志/备份/文件/终端页面；
- 一个 `Manager` 或 Vue 页面承担整个系统。

## 8.1 日志系统规范

- 左侧 `日志中心` 是全平台唯一历史日志入口，具体游戏页面只能跳转并带筛选条件。
- 游戏模块负责产生日志和必要的游戏专用诊断，目录索引、分页、搜索、导出、删除统一由 `internal/ops/logs` 管理。
- 日志正文必须按需分页/流式读取，禁止把完整大日志一次发送给 Vue。
- `runtime/log/exports/` 是 0.1.65 新默认导出副本目录，不计入真实日志数量；旧配置的 `log/exports/` 继续兼容。
- 活动日志必须有删除保护；外部手工删除后的真实文件系统状态优先。
- 操作审计只能记录安全摘要，Token、密码、API Key、Cookie、完整控制台命令禁止写入。
- 日志目录名称、分页、自动刷新等可调整默认值统一由 `configs/logging.json` 提供。

## 9. 性能规则

- 禁止无界数组、日志 DOM、goroutine、Channel、缓存。
- 高频状态优先事件推送或单一自适应轮询，禁止多个页面重复轮询同一数据。
- 大日志、大文件必须分页/流式/增量处理。
- Go 承担文件、进程、压缩、哈希、扫描、日志等重任务，Vue 只负责交互。
- 长期后台任务必须可取消、可关闭、可观测。
- 单 Vue/Go 文件约 800 行时必须评估拆分，禁止继续形成巨型万能文件。

## 10. AI 权限与最高管理员

标准调用链：

`AI Provider -> XiaoYu Brain/Planner -> AGMP Host Tool Registry -> Go Permission/Approval -> Domain Executor -> platform/runtime -> OS / Game`

AI 只调用 AGMP Host 已注册 Tool。游戏服务器业务优先经 Go Domain API；当结构化领域能力不足时，可使用 Host 注册的 `shell.exec` 等通用后备 Tool。Shell/系统级工具属于显式高风险能力，必须经过 Go Host 的身份、权限、审批与审计边界，并受 `configs/permissions.json` 控制。Rust XiaoYu Brain 只能提出/编排 Tool 请求，不能自行执行 OS 能力。

审批模式由最高管理员决定：

- 请求批准：高风险操作逐次询问；
- 帮我批准：低风险自动执行，高风险询问；
- 完全访问权限：管理员主动开启后，对**已注册且已启用的 AI Game Manager Panel Tool**自动放行。

管理员可以在 `configs/permissions.json` 中关闭通用 Shell 等系统能力；默认能力存在，但 ask/risk/full 三种审批模式决定具体动作是否自动执行。AI Game Manager Panel 不替最高管理员做最终授权决定，必须清楚展示风险、保留审计、避免秘密泄露。

## 11. 安全边界

- Token、密码、API Key、Cookie 不得写日志、诊断包或普通前端持久化。
- 文件 API 必须校验允许根目录并防路径穿越。
- 压缩包解压必须限制路径、符号链接、文件数和总大小。
- Web 远程管理在开放到非 loopback 前必须先完成认证和权限系统。
- 删除、恢复、端口/防火墙、系统命令等操作必须进入操作审计。

## 12. UI 规范

- 主界面采用现代、克制、低干扰的 小鱼风格。
- 软件默认进入 `小鱼`。
- AI 未配置时，手动一键部署和全部传统管理功能仍可使用。
- 同一业务只保留一个信息源；游戏工作台中的日志/备份等入口应跳转到公共中心并带筛选条件。
- 不为了“炫”增加影响性能的持续动画、模糊和大面积重绘。

## 13. 文档规范

不允许每个目录都创建 README。核心维护文档集中在：

- `docs/PROJECT-ARCHITECTURE.md`：目录、关键文件、模块、配置、依赖和边界总说明；
- `docs/DEVELOPMENT-PLAN.md`：阶段开发计划；
- 具体已实现复杂功能可以保留专项文档，但禁止为了占位创建大量重复 MD。

新增/移动核心模块时必须同步 `PROJECT-ARCHITECTURE.md`。

## 14. 多平台安装与发布

正式 Windows 发布必须支持：

- `AI-Game-Manager-Panel-Setup.exe` 安装向导；
- 用户选择安装目录；
- 开始菜单快捷方式，可选桌面快捷方式；
- 支持卸载和固定 AppId 覆盖升级；
- 升级默认保留运行数据；
- 安装包禁止包含任何用户秘密。

正式 Linux 发布必须支持：

- 无 GUI Server Edition + Web 管理平台；
- `install.sh` / systemd 安装与管理流程；
- 可直接解压运行的 `tar.gz` 便携包；
- Debian/Ubuntu `.deb` 安装包；
- 系统安装后的运行数据必须进入当前用户可写目录，禁止把日志/配置写进只读 `/usr`；
- Linux 原生依赖必须由构建脚本检测，禁止在 Windows 里伪造“已成功交叉编译”。

Windows Wails Release 使用菜单 `4`，Electron Windows Release 使用菜单 `5`，完整双桌面发布使用菜单 `10 -> 3`；只有对应 Installer 与 Portable 都生成后才能报告 Release Success。Linux Server Edition 使用菜单 `6`（WSL）或 Linux 原生脚本。

## 15. 构建与测试

交付前至少执行：

```text
pnpm --dir frontend run build
go test ./...
go vet ./...
Windows 正式 Release
Linux Release（涉及 Linux 代码/发布脚本时）
```

平台相关功能还必须做对应 Windows 实机验收。编译通过不能冒充实机验证通过。

## 16. 更新记录

每版 `docs/releases/<version>.md` 至少包含：新增、修复、优化、架构调整、测试、已知问题、下一阶段。

## Linux Server Edition 规范（0.1.43 起）

- 云服务器/NAS/VPS 默认使用无界面的 Go Server Edition，不强制安装 Wails、GTK 或 WebKitGTK。
- 正式 Linux Server Release 必须同时覆盖 `linux/amd64` 与 `linux/arm64` 控制面二进制。
- ARM64 仅代表 AI Game Manager Panel 控制面可运行；具体游戏服务端是否支持 ARM64 必须由游戏模板单独声明，禁止笼统承诺“所有游戏全平台”。
- 安装脚本可以使用 root 完成系统用户、目录、systemd 等系统级变更，但 AI Game Manager Panel 服务默认必须以独立低权限用户运行。
- Docker Runtime 必须默认非 root，镜像运行阶段不得包含 Node/pnpm 构建工具。
- Linux Server 默认配置统一放在 `configs/server.json` 与 `configs/release.json`，业务代码不得散落写死安装路径、监听端口、服务用户名或架构列表。
- `bash scripts/build_linux.sh` 是 Linux Server Edition 的正式顶层构建入口；构建失败时不得生成“成功”状态文件。

## 0.1.52 双桌面 Adapter 规则

- Wails 与 Electron 只允许拥有不同桌面 Adapter / Shell，不允许复制业务逻辑。
- SteamCMD、DST、认证、日志、文件、实例、任务、网络等实现必须继续落在 Go Core。
- Electron Renderer 禁止 `nodeIntegration=true`，必须保持 `contextIsolation=true` 与 `sandbox=true`。
- Electron preload 不得暴露通用 Node、任意 Shell 或通用 `ipcRenderer`；新增能力先进入 Go Application/HTTP API。
- Wails 与 Electron 必须拥有独立 Release 入口：菜单 4 只构建 Wails，菜单 5 只构建 Electron；任何一方失败不得修改另一方最近一次成功产物。
- 双桌面完整发布统一由菜单 `10 -> 3` 顺序调用两个已独立验证的 Release，不得维护第三套重复构建逻辑。任一 Adapter 的 bug 修复优先修 Core/公共 Vue，而不是复制补丁。

## 0.1.53 首次启动布局规则

- 首次启动整个舞台必须在可用窗口内保持视觉居中；内容超高时必须允许完整滚动，不能通过强制顶对齐破坏大屏居中效果。
- AI Game Manager Panel 品牌标题必须脱离实际表单卡片，放在顶部左侧独立区域。
- 当前步骤标题必须脱离实际表单卡片，放在顶部右侧独立区域，与品牌区构成左右布局。
- 注册、登录、环境初始化的真正表单内容必须作为独立卡片在标题区下方居中。
- 禁止在内容卡片内再次重复 `auth-step` 标题。
- 小屏允许收窄与滚动，但不允许丢失底部操作按钮或破坏表单可访问性。

## 授权保险规则

- CDK / 机器码 / BFID 属于“系统设置 → 授权与设备安全”的独立保险层，不得放在 Owner 注册或登录之前阻断首次使用。
- Owner 注册/登录继续使用账号 + 密码 + 可选安全密钥；授权保险与身份认证必须保持职责分离。
- BFLC2 离线许可证必须使用 Ed25519 签名，不能使用可逆“算法 CDK”或客户端内置共享密钥；短 CDK 仅作为未来在线 License Server 的兑换码。
- 发行私钥永远不得进入 AI Game Manager Panel 源码、客户端、安装包、日志或配置目录。
- 发行私钥只能由项目所有者在仓库外本机生成并离线备份；正式 Release 必须通过 Release Key Gate，确认源码存在一个合法 `active` 发行公钥。
- 公钥环允许 `active / retired / legacy` 共存；密钥轮换不得默认删除仍用于验证历史 BFLC2 的旧公钥。
- 许可证公钥必须编译进 Go Core，不能从用户可编辑 JSON 读取作为信任根。
- Windows 机器码只使用稳定系统标识派生结果，UI/日志不得输出原始 MachineGuid。

## pnpm Store 与依赖恢复规则

- Windows 开发/打包依赖默认使用 AI Game Manager Panel 专用 Store：`%LOCALAPPDATA%\AI Game Manager Panel\DevTools\pnpm-store`，不得依赖用户全局 pnpm Store。
- 正常安装只允许使用经过项目审核的最小参数集：`pnpm install --store-dir <AI Game Manager PanelStore>`。禁止添加未经当前 pnpm 官方文档/帮助确认的参数。
- 首次安装失败时，只允许重置当前 AI Game Manager Panel 项目的 `node_modules` 与 AI Game Manager Panel 专用 Store；不得删除或修改用户其他项目的全局 pnpm Store。
- Electron Runtime 使用 `install-electron`，缓存位于 `%LOCALAPPDATA%\AI Game Manager Panel\DevTools\electron-cache`。

## 0.1.62 Windows 开发助手与编码硬规则

- 根目录只允许一个用户入口 `AI-Game-Manager-Panel.bat`；该文件必须 ASCII-safe + CRLF，只负责切换 UTF-8 code page、定位项目目录并启动 `scripts/windows/AIGameManagerPanel.ps1`。
- `scripts/windows/` 下禁止再出现 `.bat`；Windows 菜单、任务编排、依赖、检查与 Release 统一使用 PowerShell。
- 所有 `.ps1` 必须保存为 **UTF-8 with BOM + CRLF**，兼容 Windows PowerShell 5.1 与 PowerShell 7。
- PowerShell 启动后必须显式设置 Console Input/OutputEncoding 为 UTF-8；任何 `�`、非法字节、中文残片被 CMD 当作命令、菜单行乱码，都视为 **Gate FAIL**，禁止进入 Release。
- 禁止 `Invoke-Expression`、禁止字符串拼接后执行外部命令、禁止 Node `shell:true`。外部程序统一使用 `& $exe @args` / 参数数组调用，必须支持中文路径和包含空格的路径。
- 初始化（菜单 1）只负责工具链、Go Modules 与 Frontend 依赖；Electron 依赖/Runtime 在菜单 2 的 Electron 开发或菜单 5 的 Electron Release 时按需准备。初始化不得把 TypeScript/业务代码错误混进“环境初始化失败”；代码质量统一由菜单 3 检查。
- 主菜单保持精简，当前固定 0-10；新增菜单前必须先证明不能通过现有子菜单合并。
- Wails、Electron、Web 必须可独立开发/检查/发布；一键发布只能调度已存在的独立 Release，不允许维护重复构建逻辑。
- `scripts/common/check-windows-helper.mjs` 与 `scripts/windows/Test-WindowsHelper.ps1` 是强制门禁：检查编码、路径、禁用模式、Frontend 相对导入与菜单映射。
- 每次交付 Windows Helper 变更，必须执行菜单 1-10 Dry-Run 自检；在 Windows 实机上还必须至少验证菜单 1、3（完整检查）、4、5、8、10。
- 任何 Windows Helper 交付包若出现 `�`、乱码菜单、CMD 把中文残片当命令、路径被空格截断或 native command 输出编码错误，必须立即判定该版本不可交付；禁止以“用户环境差异”绕过编码 Gate。

## 0.1.63 开发提示词归档硬规则

- `docs/prompts/` 是 AI Game Manager Panel 唯一正式开发提示词归档目录。进入正式开发、重构、修复或阶段任务前，只要存在用户确认的提示词/实施约束，就必须先归档到该目录。
- 文件命名统一使用 `<version>-<topic>.md`；例如 `docs/prompts/0.1.63-five-foundation-fixes.md`。
- 提示词归档用于追溯“为什么改、按什么要求改”；`AGENTS.md` 继续负责长期有效的永久开发规范，两者职责不得混淆。
- 已归档提示词视为历史需求证据，禁止在后续版本直接覆盖改写。新增或变更需求必须新建新的版本/主题提示词文件。
- 每版 `validation.md` 必须明确写出对应提示词路径，并逐项核对提示词中的验收要求。
- 禁止只在聊天中保留正式需求而不进入仓库；未归档的正式提示词视为开发流程未开始，不得冻结版本基线。

## 0.1.64 工作台、PowerShell 与 Electron 发布回归硬规则

- PowerShell 变量名大小写不敏感。`scripts/windows/` 内禁止把 `$home`、`$host`、`$pid` 等名称用作可写局部变量；新增脚本必须通过自动变量/只读变量冲突 Gate。
- 只负责显示信息的 PowerShell 函数不得把 Hashtable / PSObject 意外写入成功输出管道；如需内部返回对象，调用端必须显式接收或 `$null = ...`。
- Electron 原生 Application Menu 必须显式中文化；禁止回退 Electron 默认英文菜单，也禁止“帮助”跳转无关占位网址。
- Electron Windows Release 在 AI Game Manager Panel 已准备本地 Electron Runtime 后，必须优先使用经验证的本地 `electronDist`；Release 前先验证 `electron.exe` 与 builder 配置，不允许重新把 Electron distribution 下载链当作默认兜底。
- AI Game Manager Panel 主界面采用可拖拽工作台能力：左右侧栏可独立调整宽度并完全收起；底部面板可调整高度并完全收起；布局状态持久化。终端入口归属 Bottom Panel，不得重新放回左侧一级导航。
- Theme 必须覆盖 `html/body/#app`、左右侧栏、Topbar、Bottom Panel 和全部主要 Surface。任何“浅色卡片 + 深色外围壳”的混合状态都判定 Theme Gate FAIL。
- 设置/功能页不得通过窄 `max-width` 人为浪费工作区；大屏中央区域默认只保留约 24~32px 安全边距，并使用统一 Typography/Spacing Token。
- 0.1.64 十项统一修复的正式需求以 `docs/prompts/0.1.64-ten-issue-consolidated-fix.md` 为准；后续不得覆盖该历史提示词。

## 0.1.74 Agent-first / Rust Runtime 规则

- 小鱼是默认主入口；传统可视化页面和手动终端是辅助/兜底，不得反过来让 AI 变成装饰聊天框。
- 小鱼核心 Runtime 与 Go Core 必须保持职责分离：Rust 不复制 Steam/DST/实例/授权/文件/Process 业务，Go 不重复实现 Brain/Planner/Memory。Process/stdio/PTY 的平台实现统一归 `internal/platform/runtime`。
- Rust wire protocol 当前固定为 `xiaoyu.v1`，UI/Go/Rust 三层字段变更必须同步并有 Gate。
- Tool 必须声明风险级别；Agent 不得直接执行未注册能力。
- `ask/risk/full` 是审批策略，不等于“模型想做什么都能做”；`full` 仍只对明确注册并允许的 Tool 生效。
- 0.1.74 的 `process.run` 是一次性受控命令，不得在文档中冒充持久 PTY。
- 后续可参考 Codex / OpenCode / Claude Code / ChatGPT 的公开交互与架构思想，但禁止直接复制不兼容授权或非开源实现。

## 17. 0.1.83 核心架构定型与 XiaoYu 能力接口硬规则

- 一个业务模块一个明确归属。文件、设置、节点、授权、备份、日志、终端、Steam、实例等各管各的，禁止把 A 模块业务逻辑塞进 B 模块目录。
- 页面归属不等于业务归属。例如授权 UI 可以显示在设置中心，但授权核心仍归 `internal/system/license`。
- AI 只能调用 Host 注册的 Tool / Service / API；`internal/xiaoyu` 不得复制业务实现。已有可靠 Domain Tool 时优先使用它；没有合适领域能力或现实异常超出预设时，允许转向受控通用 Tool / `shell.exec`，但不得绕过 Host 执行边界。
- 小功能不要为了“模块化”拆得过碎；强关联代码可同文件。需要拆时使用相同短前缀，例如 `node_scan.go`、`node_scan_win.go`，让人工和 AI 一眼看出是一组。
- 未实现功能不得创建只有 `doc.go`、`module.ts` 或空 README 的占位包；规划状态统一由 `configs/modules.json` 管理，出现真实代码时再创建目录。
- `internal/app` 允许按 `app_core.go / app_games.go / app_xiaoyu.go / app_auth.go` 等同包文件聚合，禁止重新长成一个 1000+ 行全能文件。
- 终端只维护一套用户入口：Bottom Panel `TerminalPanel`；系统能力统一走 XiaoYu/Domain 安全边界，不得出现死代码式第二套 Terminal UI。
- `internal/platform/runtime` 是共享 Process/Terminal Session 实现。DST、Minecraft 和未来满足 stdin/stdout 模型的游戏必须复用它；游戏只实现命令/解析适配，不得再次出现各游戏自己的 `process_windows.go`。
- 新增文件/目录/接口名称必须短、明确、可搜索。优先 `files`、`nodes`、`license`、`settings`、`backup` 等稳定业务词，避免冗长重复命名。
- 同一功能拆成多个文件时必须使用统一短前缀，例如 `file_scan.go`、`file_scan_rule.go`、`file_scan_win.go`；有明确顺序时才使用 `xxx_one.go`、`xxx_two.go`。不得把同一功能拆成看不出关联的一堆文件。
- 新增源码文件名优先控制在 32 个字符以内；超过 48 个字符必须有明确理由并由 Module Gate 拦截。
- 当前阶段不做大规模项目搬家；只做不破坏功能的轻量归属纠偏。核心能力、稳定、性能、安全和省心优先。

AI 权限名称固定为 **请求批准 / 帮我批准 / 完全访问权限**。完全访问权限只减少审批，不取消任何硬安全检查。即使 Full Access，AI 仍必须通过模块能力接口，并执行必要的校验、保护和结果验证。

### 0.1.88 双大脑 / 多端 XiaoYu 硬规则

- AGMP 产品模型固定为 **Human + XiaoYu / two-brains-one-body**。用户拥有最终控制权；XiaoYu 负责智能规划与自动执行，两者共享 Host/Domain/Runtime，禁止两套业务身体。
- XiaoYu 属于 AGMP Core，不属于桌面端。Web/Linux/Docker 必须拥有与 Windows Desktop 等价的核心 AI：Model Center、Harness、Tool、审批、Run、事件流。
- Headless 是强制架构能力：`internal/xiaoyu` 与 Rust Brain 禁止依赖 Wails/Electron；Linux/Docker 正式包缺预编译 XiaoYu Runtime 直接失败。
- Autonomous Run 归 Server 所有，客户端断线不取消；Run 绑定 Initiator，多用户 Operator 只能查看/控制自己的 Run，Owner/Admin 可监督全部。
- Human takeover/pause/cancel 永远高于 XiaoYu；人工接管后自动循环停止，直到明确交还。
- Go Host/RunManager 只能负责生命周期、安全、Tool 执行驱动和模型通信，不能复制 Rust 的 Planner/Memory/语义 Decision。
- 第三方 DSH Tool 插件默认以 Node Permission 受限子进程运行，只读插件自身目录，不继承 AGMP 秘密环境，不允许直接 child_process/文件写；插件入口与 patch 必须通过 symlink 真实路径边界检查。

### 0.1.85 Shared Runtime 硬规则

- Rust `xiaoyu-core` 是 **Brain-only**：禁止 `std::process` / `std::fs` 等直接 OS/业务执行；可执行 Tool 的注册、最终权限/审批与 Handler 全部归 AGMP Go Host。
- `internal/xiaoyu/contract.Registry` 是唯一 Host Tool Catalog/Dispatch 边界；同名 Tool 禁止覆盖，名称和风险级别必须通过校验。
- 工作区文件 Tool 必须经过 `internal/ops/files` 的 canonical path + symlink containment 校验，不能由 Brain 自己读文件。

- `internal/platform/runtime` 是 AGMP 唯一 Process/stdio 实现边界。业务域不得直接 `exec.Command`、`exec.CommandContext` 或自行创建 stdin/stdout/stderr pipe。
- 长生命周期游戏/终端会话统一使用 `Session`；一次性内部命令统一使用有输出上限和 Context 取消的 `Run`；无需交互的外部启动使用 `StartDetached` 并异步回收。
- Session 必须保留 stdout/stderr 来源、串行化 stdin、PID/状态/退出码快照；Unix 终止以进程组为单位，Windows 保持隐藏控制台并允许显式 `ShowWindow` 的 GUI 安装器例外。
- PTY/ConPTY 和跨 AGMP 重启后的会话重连仍是独立增强项；实现时只能扩展当前 Runtime，不得再创建另一套 Process Core。

## 18. 0.1.81 架构冻结

0.1.81 完成核心目录聚合后冻结骨架。后续新增能力优先进入既有领域，禁止为了单一小功能新增一级业务域。
