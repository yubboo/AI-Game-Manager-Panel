# AGMP 0.1.82 当前完成度

> 本报告区分“架构完成度”和“真实功能完成度”。目录存在或页面存在不等于功能已经完成。

## 总体判断

0.1.82 已经完成一轮核心架构定型与命名收口，适合作为后续开发候选基线；但**还不是功能完成版，也不是最终 Release Candidate**。

## 1. 架构与工程边界

状态：**高完成度**。

已完成：

- 产品与超级大脑身份分离：AGMP = 产品，XiaoYu = 内置超级大脑。
- Rust `xiaoyu-core` 明确为唯一 Brain，Go `internal/xiaoyu` 定位为桥接/Tool/审批持久化等过渡能力，不允许发展第二套 Planner/Brain。
- Go 后端从平铺 Service/Core 收敛到 `xiaoyu / games / server / ops / deploy / system / platform` 主要领域。
- Module Gate 强制检查一级领域、禁止恢复 `internal/service` / `internal/core`、禁止旧产品缩写重新进入源码。
- Windows Helper / Installer / Updater / GitHub Safety Gate 均通过。

仍需继续：

- `internal/xiaoyu` 中部分 Go 审批/权限/Registry 仍属于过渡实现，最终权威决策应逐步收回 Rust XiaoYu Core。
- `internal/platform` 内部仍有若干小型子包，但已限制在明确基础设施组内；后续只在真实维护痛点出现时继续合并，避免为了目录漂亮再次大迁移。

## 2. XiaoYu 超级大脑

状态：**核心基础已落地，完整智能循环尚未完成**。

已落地：

- Rust Workspace：`xiaoyu-core` + `xiaoyu-protocol`。
- `xiaoyu.v1` JSON-RPC stdio 协议。
- Runtime status / session foundation / Tool Registry。
- Allow / Confirm / Deny 审批基础。
- 首批 `system.info`、`fs.list`、`fs.read`、受控 `process.run`。
- Go ↔ XiaoYu Runtime bridge。
- 用户审批状态、单次批准消费、审计边界基础。

关键未完成：

- 真正模型 Provider 接入与可替换 Provider 管理。
- Planner / Executor 自动循环。
- Context Builder、长期/会话 Memory。
- 多步骤 Workflow 恢复、重试、取消与结果验证闭环。
- Tool 并行、流式 reasoning/event、故障恢复。
- XiaoYu 对全部领域 Tool 的完整覆盖。

因此当前 XiaoYu 更准确的定位是：**安全可控的 Intelligence Core Foundation，而不是已经完成的自主超级大脑。**

## 3. 通用终端与服务器控制

状态：**Partial**。

架构边界已经固定：

```text
XiaoYu / 人工 UI
    -> server/terminal
    -> platform/runtime/terminal
    -> platform/runtime/process
    -> Game Server
```

但 DST 当前仍保留 `internal/games/dst/runtime` 的既有进程/stdin/stdout 实现。后续需要在不破坏 DST 行为的前提下逐步抽到公共 Runtime，再让 Minecraft/其他游戏复用。

## 4. 模块状态

`configs/modules.json` 当前共声明 **33 个模块**：

- `implemented`：1 个（日志中心）
- `active`：1 个（更新中心）
- `partial`：6 个（包括 XiaoYu、部署、终端、环境、网络、设置）
- `skeleton`：25 个

这说明目前项目的**架构成熟度明显高于功能成熟度**。如果仅按模块状态观察，产品仍处于早中期开发阶段，不能把目录齐全误认为功能接近完成。

## 5. 本轮真实验证

已通过：

- Project Layout Gate
- Module Boundary Gate
- XiaoYu Core Gate
- Product Architecture Gate
- Windows Helper / UTF-8 / CRLF / Frontend Import Gate
- Windows Installer Gate
- Updater Gate
- GitHub Safety Gate
- 85 个 Go package 的 `go test`（仅排除依赖 Wails 外部模块的 `internal/bridge/wails`）
- 同范围 `go vet`
- `agmp_dev_license` 开发许可证模式测试
- `cmd/aigame-manager-web` Go 构建
- `cmd/aigame-manager-license-admin` Go 构建
- 许可证命名迁移兼容回归测试

当前环境无法完成：

- Rust `cargo build/test/clippy`：当前执行环境没有 Cargo/Rust 工具链。
- Vue/pnpm build：当前执行环境没有 pnpm 与前端依赖。
- Wails 根项目完整构建：源码当前未带 `go.sum`，同时 `frontend/dist` 需要先由前端构建生成；当前隔离环境无法联网下载 Wails 模块。
- Windows 真机 Setup/Wails/Electron 启动验收：需要在 Windows 开发机完成。

## 6. 下一阶段优先级

1. Windows 本机完成 0.1.82 全构建验收并补齐 `go.sum`。
2. 把 XiaoYu 的 Provider + Planner/Executor Loop 做成真正的超级大脑主循环。
3. 完成公共 Terminal Runtime，先迁移 DST，再接 Minecraft。
4. 按领域 Tool Contract 给 XiaoYu 接入实例、备份、文件、Steam、环境、日志等真实能力。
5. 功能阶段继续遵守现有七领域边界，不再做总体骨架大手术。
