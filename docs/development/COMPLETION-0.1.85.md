# AGMP 0.1.85 完成度说明

## 结论

**底层核心：可冻结。产品功能：仍处于早中期。XiaoYu 超级大脑：架构边界已定，智能闭环尚未完成。**

0.1.85 已把最容易导致未来“大换心”的部分收口：Rust `xiaoyu-core` 唯一负责 Brain/Policy/Protocol；AGMP Go Host 唯一负责 Tool 注册、最终权限/审批与领域分发；`internal/platform/runtime` 是唯一 Process/stdio 实现。

## 本轮底层完成

- 统一长生命周期 Session 与 Manager。
- stdout/stderr 分源、全局 Sequence。
- 固定容量环形历史，避免高日志量 O(n) 移动。
- 非阻塞实时订阅，慢 UI/WebSocket 不阻塞游戏进程输出。
- 串行 stdin / Send / SendLine / CloseInput。
- PID、状态、开始/退出时间、退出码与错误快照。
- Environment parent inheritance + Overlay / 显式 Replace。
- bounded Run、Shell、超时/取消、输出上限、Detached launch。
- Unix process-group TERM/KILL；Windows process group + hidden-window foundation。
- DST、XiaoYu Runtime、Updater 等现役进程入口统一复用 Shared Runtime。
- Gate 强制禁止业务域恢复 `os/exec`/stdio 私有实现。
- XiaoYu Rust Brain 清除直接 OS/file/process 执行；Host Tool Registry 拒绝重复名/非法名/非法风险等级。
- 工作区文件 Tool 使用 canonical + symlink 边界检查。

## 当前模块总体状态

`configs/modules.json` 目前共 33 个模块：

- implemented：1
- active：1
- partial：6
- skeleton：25

因此不能把“底层架构完成”误解为“AGMP 产品已经接近完成”。

## XiaoYu 下一阶段重点

- Model Provider 抽象与真实模型接入。
- Planner / Executor Agent Loop。
- Context Builder 与系统状态感知。
- Memory / Session / Task Context。
- 多步骤 Workflow、失败恢复、重试与结果验证。
- 将文件、备份、实例、服务器、Steam、环境、网络、日志等真实 Domain Tool 逐步注册到 AGMP Host Registry。

## 后续底层增强（不改架构）

- PTY / Windows ConPTY。
- 跨 AGMP 重启的外部进程发现和 Session Reattach。
- Windows Job Object / 进程树约束。
- 大规模长期运行压力测试与 WebSocket 终端恢复策略。

这些功能都在现有 `platform/runtime` 和 Host Tool 边界上扩展，不再允许重写第二套底层。
