# AGMP 0.1.84 当前完成度

## 结论

0.1.84 不再做第三次大骨架搬迁，而是在 0.1.83 已冻结架构上把 **共享 Process / Terminal Runtime** 做到可以作为后续游戏控制台的稳定地基。

对“可执行程序 + 参数 + stdin/stdout/stderr”这一类游戏服务器，底层公共控制链已经成立：

```text
Game Adapter / XiaoYu / AGMP Module
                ↓
       platform/runtime
       ├─ Session
       ├─ Manager
       ├─ Run
       └─ StartDetached
                ↓
        Windows / Linux / macOS
```

DST 已实际复用该 Runtime；XiaoYu Runtime 的内部子进程调用、Updater 外部安装器启动、非 Windows 系统打开器也已收口进同一底层。业务域中已无直接 `exec.Command/CommandContext` 或自行 stdio pipe 的现役实现。

## 共享 Runtime 已完成

### 长生命周期 Session

- 进程启动与参数传递；
- Working Directory；
- 环境变量继承 + Overlay；
- 显式 ReplaceEnvironment；
- stdin raw `Send` / `SendLine`；
- 并发 stdin 串行保护；
- stdout / stderr 独立读取并保留来源；
- 4 MiB 单行 Scanner 防护；
- PID；
- created / starting / running / stopping / exited / failed 状态；
- StartedAt / ExitedAt / ExitCode / Error 快照；
- Done / Wait / WaitContext；
- CloseInput；
- Terminate / Kill；
- 真实 `cmd.Wait()` 回收后才发布退出状态。

### 命名 Session Manager

- ID 唯一性；
- Start / Get / Stop / Delete；
- 活跃 Session 禁止误删；
- Terminate 超时后 Kill；
- 完成 Session 可保留快照，显式删除；
- 多 Session 快照稳定排序。

### 一次性 Run

用于 XiaoYu Core 等内部受控 CLI 调用：

- 不经过 Shell；
- stdout / stderr 独立捕获；
- 默认单流 8 MiB 上限；
- 可配置输出上限；
- Context 超时/取消；
- 超时主动 Kill；
- 输出超限显式返回错误，防止无界内存增长或截断 JSON 被误当成功。

### StartDetached

用于安装器、系统打开器等无需交互的外部进程：

- 统一平台进程参数；
- stdout/stderr 丢弃；
- 后台 `Wait()` 回收，避免僵尸/句柄泄露；
- Windows GUI 安装器可显式 `ShowWindow`，不会被共享 Runtime 的默认隐藏窗口策略误伤。

### 平台终止策略

- Windows：新进程组 + 默认隐藏控制台；GUI 可显式 ShowWindow；Terminate/Kill 使用进程句柄终止。
- Linux/macOS：新进程组；Terminate 使用 SIGTERM，Kill 使用 SIGKILL，作用于整个组，避免只杀父进程留下子进程。

## 已经消除的重复实现

- DST 私有 Windows/Other Process 实现：已在 0.1.83 删除。
- XiaoYu Runtime 私有 `process_windows.go / process_other.go`：0.1.84 删除。
- Updater `exec.Command`：改为 `StartDetached`。
- 非 Windows 文件夹/浏览器打开器：改为 `StartDetached`。

Module Gate 已阻止这些实现重新长回来。

## 还没有完成，但不会要求重构底层

### PTY / ConPTY

当前 `Session` 是可靠的 pipe-backed console，已经足够服务 DST、Minecraft Paper 等标准 stdin/stdout 游戏服务器。真正 PTY/ConPTY 仍需后续实现，主要用于：

- 完整 ANSI/光标控制；
- 交互式 Shell；
- 依赖 TTY 检测的程序；
- 更接近原生 PowerShell/cmd/bash 体验。

PTY 会作为现有 Runtime 的 Transport 扩展，**不会再创建第二套 Terminal Core**。

### 跨 AGMP 重启的会话恢复

当前 Session 在一个 AGMP 进程生命周期内可稳定管理。AGMP 自身重启后要重新接回旧控制台，需要进一步设计守护进程/IPC/Named Pipe 或持久 PTY；仅靠 PID 无法安全恢复 stdin/stdout。

这也是后续增强，不需要改变现有 Game Adapter -> platform/runtime 边界。

## XiaoYu 超级大脑状态

0.1.84 继续保证：

```text
Rust xiaoyu-core = 唯一 Brain
Go internal/xiaoyu = contract / control / runtime
```

小鱼当前已有 Rust Core/Protocol、Tool Registry、审批/权限基础和 Go Bridge，但真正智能闭环仍需继续：Provider -> Planner/Executor -> Context/Memory -> Workflow -> Result Validation -> Domain Tools。

因此“底层运行时已经成熟”和“整个 AGMP 已经完成”不能混为一谈。

## 产品模块矩阵

`configs/modules.json` 当前共 33 个模块：

- implemented: 1
- active: 1
- partial: 6
- skeleton: 25

骨架/Runtime 成熟度明显高于业务功能完成度。

## 下一阶段建议

0.1.85 起不再继续搬架构。优先：

1. XiaoYu Provider 与真正 Agent Loop；
2. Planner / Executor / Result Validator；
3. Context / Memory / Workflow；
4. 把现有 AGMP Domain Service 逐步注册为正式 XiaoYu Tools；
5. Minecraft Adapter 直接复用 0.1.84 Shared Runtime；
6. 之后再做 PTY/ConPTY 与跨重启 Session Agent。
