# AI Game Manager Panel 项目规则

> 状态：**强制规则 / 0.1.88 双大脑 + XiaoYu Harness + 多端 AI 定型规则**  
> 适用范围：人工开发、ChatGPT、Codex、其他 AI、CI、Web/Wails/Electron/Rust Native 适配与正式发行。

本文件只定义不能被后续开发随意改变的产品与工程原则。0.1.83 完成第二轮核心架构收拢；本版通过 Gate 后冻结主要业务边界，后续功能开发不得随意增加一级域或恢复空占位包。

## 1. 唯一产品主体

**AI Game Manager Panel（AGMP）是唯一产品本体。**

Web、Wails、Electron、未来 Rust Native 只是同一个 AGMP 的不同构建/运行端。不同端允许有 Shell、Bridge、平台 API、打包方式差异，但不得演变成各自独立的业务产品，也不得复制出多套互不兼容的 Agent/Core 逻辑。


## 1.1 0.1.88 双大脑 / Web AI Parity 硬规则

- 产品控制模型固定为 **two-brains-one-body**：人为大脑决定目标/边界/审批/接管，XiaoYu 负责理解、规划、观察、验证与自动执行；二者共享 AGMP Host 和 Domain 能力。
- XiaoYu 是 **AGMP Core 能力，不是 Desktop 能力**。`internal/xiaoyu`、`rust/crates/xiaoyu-core`、模型中心和 Agent Harness 禁止依赖 Wails/Electron/Desktop Adapter。
- Web、Windows Wails、Windows Electron、Linux Server Web、Docker Web 均属于完整产品目标，核心 XiaoYu 能力不得阉割。正式 Linux/Docker 产物缺少预编译 XiaoYu Runtime 必须构建失败。
- 自主 Run 由服务器拥有；浏览器/WebView 断开不得取消 Run。事件通过有界 Sequence Trace + SSE/WebSocket 类流重新订阅。
- Human override 永远优先：`pause / takeover / cancel` 必须能中断当前可取消 Brain Turn；人工接管期间 XiaoYu 不得自动继续。
- 多用户 Web 中，Run 必须绑定发起人。Owner/Admin 可全局监督；普通 Operator 只能读取/控制自己的 Run 与对应 Event Stream。
- 人工按钮和 XiaoYu Tool 必须调用同一 Domain Action。禁止为了 AI 另建一套 DST/Minecraft/Steam/Backup 业务实现。
- Go RunManager/Host 只负责生命周期、安全和执行驱动；“下一步做什么、目标是否完成、如何语义恢复”由 Rust XiaoYu Brain 决定，禁止 Go 逐渐形成第二个 Planner/Brain。
- 第三方 DSH Tool 插件必须在独立受限进程运行，默认不得获得宿主环境秘密、任意文件写入或 child_process；插件必须通过 AGMP Tool/Capability 才能操作服务器资源。

## 2. 小鱼是内置核心大脑

小鱼是 AGMP 内置的智能核心，不是外挂、插件式第二产品，也不是要求用户另外下载和启动的软件。

产品主控制链固定为：

`用户意图 -> XiaoYu Brain/计划 -> AGMP Host Tool Registry -> Go 权限/审批 -> Domain 执行 -> 验证 -> 汇报`

为了稳定性、性能和故障隔离，底层可以使用 Rust 原生模块、内部进程、Go 服务或平台组件；这些只属于实现细节。任何内部可执行文件都不得被包装成普通用户需要理解或单独操作的产品。

## 3. 两个大脑，同一副身体

- **人为大脑**：用户/管理员决定目标、边界、审批与最终接管，始终拥有最高控制权。
- **XiaoYu 大脑**：负责理解目标、规划、观察、验证、恢复与自动执行，定位是帮助人更省心地完成开服和运维，不取代人的最终决定权。
- 手动服务器页、按钮、表单、日志、文件管理、终端等必须保留；用户既可以自己操作，也可以把目标交给 XiaoYu。
- 两个大脑共用一副 AGMP 身体：同一 Domain Action、权限模型、审计、数据模型与 Shared Runtime。禁止“AI 一套、人工一套”。
- AI 不可用时，核心服务器管理仍必须可手动使用；AI 恢复后必须重新观察真实状态，不得覆盖人工已经完成的操作。

## 4. 开发环境与生产环境严格分离

### 开发环境

面向源码开发者。基础开发只准备当前工作所需工具，例如 Go、Node.js、pnpm。Rust/Cargo/MSVC/Windows SDK 只在开发 Rust AI Core 或从源码构建需要它的正式目标时按需准备。

根目录 `AI-Game-Manager-Panel.bat` 是**开发/发行构建助手**，不是普通用户启动器。

### 发行构建环境

发布机或 CI 负责测试并预编译 Go、Rust、Frontend 和各桌面 Shell，然后把该运行端所需内部能力封装成完整 AGMP 发行产物。这里可以存在编译器、SDK、签名和打包工具，但它们不得进入普通用户运行依赖。

### 生产环境

普通用户只运行预编译成品：

- Web：完整 AGMP Web 发行包；
- Wails：完整 AGMP Windows Wails 安装包/便携包；
- Electron：完整 AGMP Electron 安装包/便携包；
- Rust Native：未来完整 AGMP Rust Native 发行包。

生产环境禁止现场编译，禁止要求用户安装 Rust、Cargo、MSVC、Go、Node.js、pnpm、Wails CLI、Visual Studio Build Tools 等开发工具链。

## 5. 多端必须是完整 AGMP

每个正式运行端都必须呈现同一个产品身份，并尽量具有一致的：

- 小鱼与 Agent 能力；
- 游戏/实例管理；
- Steam、日志、备份、文件、终端、网络、RCON 等核心能力；
- 权限、审批、审计、配置和更新规则。

内部组件即使是独立进程，也只能随完整 AGMP 包交付，不创建独立快捷方式，不单独列入普通用户 Release，不要求用户手动启动。

## 6. 当前阶段优先级

0.1.83 完成核心目录定型后冻结骨架。后续开发优先级固定为：

1. **核心能力真实可用**：Agent 能理解、规划、调用工具、执行、验证；服务器核心能力可被 AI/人工共同调用。
2. **稳定**：失败可恢复、状态一致、长任务可取消、关键流程有测试和实机验证。
3. **性能**：流式/增量处理，避免重复轮询、无界日志、无界 goroutine/缓存与不必要子进程。
4. **安全**：最小权限、危险操作审批、路径边界、秘密保护、完整审计、生产发行不泄露开发密钥。
5. **省心**：普通用户下载完整产品即可使用；升级保留数据；开发工具按需安装；能自动检测的问题不要让用户手工排查。
6. **架构美化/目录重排**：在前五项稳定后单独立项处理。

## 7. 开发完成定义

任何“已完成”的核心功能至少满足：

`实现 -> 自检/单测 -> 构建 -> 实机验证 -> 故障场景验证 -> 安全检查 -> 记录 -> 冻结基线`

只存在界面、占位 API、静态 Gate 或未执行的代码，不得宣称核心能力已经完成。

## 8. 模块自治与 XiaoYu 调用规则（0.1.83 定型）

功能按**业务模块**组织，不按页面、语言或临时需求随意散落。一个模块负责自己的业务规则、服务接口、配置、测试和平台适配；其他模块只能通过公开 API / Service / Tool 契约调用它。

- 文件管理归文件模块；设置归设置模块；节点归节点模块；授权归授权模块；备份、日志、终端、Steam、实例等同理。
- UI 可以把某模块嵌在另一个页面里，例如“授权与许可证”显示在设置中心，但授权业务仍归 `internal/system/license`，不得把许可证核心逻辑搬进设置模块。
- Rust `xiaoyu-core` 是唯一超级大脑，负责理解、规划、工作流、Tool 编排与结果验证；Go `internal/xiaoyu` 只保留 `contract / control / runtime` 三组桥接职责，不复制 Brain 或领域业务。尚未实现的领域能力以 Tool 契约和 `configs/modules.json` 状态存在，不为占位预建空目录。
- 业务域可以依赖 `internal/platform`，但禁止为了省事直接修改其他业务域的私有状态；`internal/platform` 不得反向依赖任何业务域。
- 新功能较小时不要过度拆分。强关联代码可以保留在同一文件；确实需要拆分时使用短而一致的前缀，例如 `file_scan.go`、`file_scan_rules.go`、`file_scan_win.go`，或 `file_scan_one.go` / `file_scan_two.go`。
- 文件名、目录名和 API 名优先使用短、明确、可搜索的业务词。禁止为了“显得专业”制造超长命名；人和 AI 都应能从名字判断归属与用途。
- 同一功能需要拆成多个文件时，必须保留统一短前缀，优先 `xxx.go`、`xxx_rule.go`、`xxx_win.go`、`xxx_test.go`；确有顺序关系时可用 `xxx_one.go`、`xxx_two.go`。禁止同一功能拆成互不相关、看不出关联的文件名。
- 新增目录原则上使用 1~2 个清晰业务词；新增源码文件名应尽量控制在 32 个字符以内。超过 48 个字符必须说明理由并通过 Module Gate。
- 0.1.83 已完成第二轮聚合：禁止恢复 `internal/service`、`internal/core`、`internal/games/dst/service` 等泛化中间层；新增功能优先进入 `xiaoyu/games/server/ops/deploy/system/platform` 既有边界。
- 规划功能不得只创建 `doc.go` 或前端 `module.ts` 占位目录；没有真实实现时由 `configs/modules.json` 记录阶段。
- `internal/app` 保持单包编排，但大型文件按统一 `app_*.go` 前缀按业务分组，避免一个 1000+ 行总文件。
- 手动终端唯一 UI 为 Bottom Panel 的 `TerminalPanel`；不得再维护第二套独立终端页面。
- `internal/platform/runtime` 是跨游戏统一的进程终端底座，负责 Process/stdin/stdout/stderr/PID/退出等待/Terminate/Kill、非阻塞输出订阅与固定容量历史；游戏域只保留命令语义、Ready 判定和专属状态解析，禁止复制 OS 进程实现。
- 0.1.85 起所有真正的子进程创建必须收口到 `internal/platform/runtime`：长生命周期用 `Session`，一次性受控命令用 `Run`，非交互启动用 `StartDetached`。业务域、XiaoYu Bridge、Updater 不得重新直接 `exec.Command` 或持有 stdio pipe；`exec.LookPath` 仅用于发现可执行文件，不视为进程创建。
- Rust `xiaoyu-core` 是 Brain-only，禁止 `std::process` / `std::fs` 和本地 `fs.read/fs.list/process.run` 执行。可执行 Tool 必须注册在 Go Host `internal/xiaoyu/contract.Registry`，最终 Handler 归所属业务域。
- 共享 Runtime 必须保持有界：一次性输出有大小上限，超时必须终止进程；stdin 并发写必须串行；stdout/stderr 保留来源；Session 状态、PID、退出码和退出时间必须可快照。

详细模块归属见 `docs/development/MODULES.md`。

## 9. AI 权限的硬安全语义（0.1.81 起强制）

小鱼只允许以下三个固定名称，禁止改名：**请求批准 / 帮我批准 / 完全访问权限**。

- **请求批准**：只读检查可直接执行；需要操作、修改、删除或系统权限的下一步必须向用户请求批准。未回答时保持 pending，不自动继续。
- **帮我批准**：小鱼可以主动规划并给出推荐，但只有读取自动完成；任何会改变系统状态的 Tool 都必须等待用户明确批准。推荐意见不能替代批准。
- **完全访问权限**：用户授予 AI 最高 AGMP Tool 权限，可自主规划并执行已注册、已启用的系统能力，不逐项请求批准。

“完全访问权限”不是“关闭安全”。无论哪种模式，以下保护都不能被权限模式绕过：身份/RBAC、模块 API 边界、参数校验、路径作用域、目标存在性检查、并发/幂等保护、必要备份与回滚条件、秘密保护、操作审计、执行后验证。Full 权限表示用户已授权已注册能力在当前策略范围内自动执行，不等于绕过 Host。AI 仍应优先使用模块 Service / Domain Tool；当领域能力不足时可以调用 Host 注册的 `shell.exec` 通用后备能力，但不能通过 Rust Brain、前端或未注册旁路直接执行 OS 操作。

AI 的主要职责始终围绕 AGMP：理解用户的开服/运维意图，自动调用系统内置能力完成任务。通用聊天可以提供，但优先级低于“安全、准确地操控 AGMP 完成用户目标”。
