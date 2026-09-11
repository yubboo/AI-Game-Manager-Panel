# AGMP 0.1.83 当前完成度

> 本报告严格区分“架构优化完成度”和“产品功能完成度”。目录、接口或 Gate 存在不代表功能已经完成。

## 总体判断

**0.1.83 已完成本轮未完成架构优化，适合作为新的架构候选基线。** 这一版重点不是堆新功能，而是把 AGMP / XiaoYu 长期边界、公共 Process/Terminal Runtime、DST 归属、Application 编排和前端真实目录继续收干净。

架构已经接近冻结，但产品功能仍处于早中期：`configs/modules.json` 目前声明 33 个模块，其中 `implemented` 1 个、`active` 1 个、`partial` 6 个、`skeleton` 25 个。

## 1. XiaoYu 超级大脑边界

状态：**架构边界已定型，完整智能循环仍未完成。**

已完成：

- AGMP 是产品；小鱼 / XiaoYu / xiaoyu 是 AGMP 内置超级大脑和核心伙伴。
- Rust `xiaoyu-core` 是唯一 Brain；`xiaoyu-protocol` 提供 `xiaoyu.v1` 稳定边界。
- Go `internal/xiaoyu` 已从多散包收敛为 `contract / control / runtime` 三组职责。
- Go 侧禁止新增第二套 Brain / Planner / Memory；AI 必须通过 Tool / Domain API 调用系统能力。
- 受控人工终端与 XiaoYu 共用身份、审批和审计边界。

仍未完成：

- 真正可替换的模型 Provider 层。
- Planner / Executor 自动循环。
- Context Builder、会话/长期 Memory。
- 多步骤 Workflow、恢复、重试、取消和执行后验证闭环。
- 完整领域 Tool 覆盖和流式事件体系。

因此当前 XiaoYu 的准确定位仍是 **安全核心基础 + Tool/审批 Runtime**，还不是已经完全自主运行的超级大脑。

## 2. 项目结构优化

状态：**本轮目标已完成。**

- 禁止恢复 `internal/service`、`internal/core` 和 `internal/games/dst/service` 泛化中间层。
- 删除大量只有 `doc.go` 的后端规划空包；未实现能力改由 `configs/modules.json` 管理。
- 删除大量只有 `module.ts` / 包装占位页的前端目录；真实 `features` 现在主要是 `auth / dashboard / deployment / games / instances / logs / nodes / settings / terminal / users / xiaoyu`。
- `home -> dashboard`、`servers -> instances`，前后端术语更一致。
- 通用规划页面统一由 `ModulePlaceholderView` 渲染，不再维护第二套 FeaturePlaceholder。
- DST `service/*` 被收回 `dedicated / logcenter / runtime / setup / workspace` 自身，减少“游戏域里再套 Service 层”。
- 平台微包收拢为 `files / http / net / os / runtime / security / steam / telemetry` 等稳定组。
- 原 1397 行 `internal/app/app.go` 拆成同一 package 的 `app_*.go` 业务文件；最大业务文件目前约 526 行，没有新增额外架构层。

## 3. 通用 Process / Terminal Runtime

状态：**基础公共 Runtime 已真实落地，持久 PTY/会话体系待后续。**

0.1.83 新增真实 `internal/platform/runtime`：

- `Session` 通用进程会话；
- executable / arguments / working directory；
- stdin 命令写入；
- stdout + stderr 合流读取；
- PID；
- 真实进程退出等待；
- Terminate / Kill；
- Windows `CREATE_NO_WINDOW`；
- Line / ReadError / Exit Hook。

DST 已经实际改为复用该 Session，并删除 `internal/games/dst/runtime/process_windows.go` 与 `process_other.go`。DST 只保留 `c_shutdown()`、Ready 判定、Shard 状态和日志语义。

这意味着 Minecraft、Terraria 等未来只要满足“可执行程序 + 参数 + stdin/stdout”模型，就可以共用这一底座，而不是复制进程实现。

后续仍需：PTY、持久会话、会话恢复、WebSocket 实时 Session、多游戏 Console Adapter。

## 4. 前端终端

0.1.82 存在“真正能执行受控命令的 TerminalView 是死代码，而 Bottom Panel 是 disabled 占位”的结构/行为错位。

0.1.83 已改为唯一 `TerminalPanel.vue`，直接嵌入 Bottom Panel：

- 查询 XiaoYu Runtime 状态；
- 显示当前审批模式；
- 执行受控命令；
- Approve / Reject；
- `/terminal` 只负责打开 Bottom Panel。

以后禁止维护第二套独立 Terminal UI。

## 5. 真实模块状态

当前 33 个模块：

- `implemented`：1（logs）
- `active`：1（updater）
- `partial`：6（ai、deployment、terminal、environment、network、settings）
- `skeleton`：25

这说明**架构成熟度明显高于产品功能成熟度**。0.1.83 不是 RC，也不能把 skeleton 数量用目录占位“做少”。

## 6. 本轮验证结果

已真实通过：

- Project Layout Gate
- Module Boundary Gate
- XiaoYu Core Gate
- Product Architecture Gate
- Distribution Boundary Gate
- Windows Helper / UTF-8 / CRLF / Frontend Import Gate
- Windows Installer Gate
- Updater Gate
- GitHub Safety Gate
- Release Key Gate（development-only / allow-unconfigured）
- 42 个可用 `internal/...` Go package 的 `go test`
- 同范围 `go vet`
- `agmp_dev_license` 开发授权模式回归
- `cmd/aigame-manager-web` 构建
- `cmd/aigame-manager-license-admin` 构建
- `platform/runtime` Linux 实际测试
- `platform/runtime` 与 DST Runtime 的 Windows amd64 交叉编译测试
- DST Process 生命周期、Ready、命令、优雅停止、日志游标等原有测试继续通过

当前环境无法完成：

- Rust `cargo check/test/clippy`：执行环境没有 Cargo/Rust 工具链。
- Vue/pnpm build：执行环境没有 pnpm。
- Wails 根项目完整构建：源码仍没有 `go.sum`；本环境尝试下载 `github.com/wailsapp/wails/v2@v2.15.0` 时网络 DNS 被隔离，不能伪造校验文件。
- Windows Wails/Electron/Setup 真机启动验收：需 Windows 开发机执行。

## 7. 下一阶段应该做什么

架构不建议再做第三次大整理。0.1.83 通过 Windows 本机完整构建后，应优先进入 XiaoYu 功能阶段：Provider -> Planner/Executor -> Context/Memory -> Workflow -> Domain Tools；同时继续完成 PTY/持久 Console Session 和 Minecraft Adapter。
