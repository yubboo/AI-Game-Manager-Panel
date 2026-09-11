# Steam 安装 / 校验任务（AI Game Manager Panel 0.1.32）

## 目标

AI Game Manager Panel 不接管用户 Steam 账号。游戏本体与 Steam Library 中的 Dedicated Server 仍由用户自己的 Steam 官方客户端安装/校验；AI Game Manager Panel 负责发起官方协议、读取真实安装位置、同步 AppManifest 状态与 Steam `content_log.txt`。

## 安装

- 游戏本体：`steam://install/322330`
- Dedicated Server：`steam://install/343050`

真实安装位置来自目标 AppID 所属 Steam Library 的 `steamapps/appmanifest_<appid>.acf`：

- `LibraryPath`
- `ManifestPath`
- `InstallDir`
- `InstallPath = <Library>/steamapps/common/<InstallDir>`

AI Game Manager Panel 不写死 `C:\Program Files (x86)\Steam`。

## 校验

- 游戏本体：`steam://validate/322330`
- Dedicated Server：`steam://validate/343050`

AI Game Manager Panel 发起校验前会记录 `<Steam>/logs/content_log.txt` 的文件偏移，只消费本次任务开始后的新日志。

## StateFlags 是同步状态的一等来源

0.1.29 开始不再只看 `StateFlags & 2`。AI Game Manager Panel 会识别 Steam AppManifest 中的运行状态位，例如：

- UpdateRunning
- UpdatePaused
- UpdateStarted
- Reconfiguring
- Validating
- AddingFiles
- Preallocating
- Downloading
- Staging
- Committing
- UpdateStopping

因此 Steam 仍处于 `Validating` 时，AppManifest 会被视为 **busy**，绝不能判定完成。

## 完成判定

校验任务首先必须真实进入目标 AppID 的验证流程。完成终态按可靠程度依次接受：

1. `File validation finished` 后出现目标 AppID 的 `scheduler finished : removed from schedule`；
2. 某些 Steam Client 构建不输出 scheduler 终点时，接受 `File validation finished` 后的目标 AppID `update changed : None` + 不带 Update Queued/Running 等状态的 `Fully Installed`；
3. 若 Steam Client 只暴露 AppManifest 的 `StateValidating` 转换，且本次验证结果为 **0 个不匹配文件**，允许在 `Validating -> idle`、`File validation finished`、安装目录存在、AppManifest 连续 3 次空闲确认后结束。

如果验证发现不匹配文件，则第 3 种兜底禁用，必须继续等待 Steam 的修复下载/暂存/提交与最终终态，避免扫描刚结束就提前 100%。

`Fully Installed` 绝不能单独作为校验完成依据。

## 进度真实性

### 可以显示精确百分比的阶段

Steam AppManifest 提供真实字节计数时：

- `BytesDownloaded / BytesToDownload`
- `BytesStaged / BytesToStage`

AI Game Manager Panel 直接计算真实百分比，并标记进度来源。

### Steam Client 本地文件扫描阶段

Steam Client 下载页自己的验证百分比没有通过 AI Game Manager Panel 当前使用的稳定公开接口（AppManifest / content_log / steam:// URI）直接暴露；`content_log.txt` 仍主要提供验证开始、错误与 `File validation finished`。

0.1.30 在 Windows 桌面端增加一个 **真实 I/O 驱动的估算层**：

1. AppManifest `StateValidating` 或本次目标 AppID 的 content_log 验证事件先确认“Steam 正在校验”；
2. 只有处于这个验证窗口时，AI Game Manager Panel 才累计 `steam.exe` / `steamservice.exe` 的 Windows `GetProcessIoCounters` 实际读取字节；
3. 用目标 AppManifest 的 `SizeOnDisk` 作为待扫描体积，换算为近似进度；
4. UI 明确显示“约 xx%”，进度来源标记为 `Steam Validating + Windows 进程实际读取量（估算）`。

这不是 `setInterval(progress++)` 或按时间模拟的假进度。它确实跟随 Steam 进程在验证期间发生的磁盘读取，因此通常会与 Steam 下载页的验证条一起向前推进。因为 Steam 进程还可能存在缓存、额外元数据读取或同时执行其他少量 I/O，所以允许与 Steam UI 有合理误差。

### 100% 终态门禁

无论估算读取量是多少，扫描阶段最高只显示 **99%**。Steam 未确认退出验证/更新状态时，AI Game Manager Panel 禁止显示 `100%` 或“校验完成”。

当 AppManifest 从验证/busy 返回 `FullyInstalled + idle` 后，AI Game Manager Panel 先进入 `99% / Steam 收尾中`，连续确认稳定终态后才切换为 `100% / 校验完成`。

Dedicated Server 后续接入 SteamCMD Core 后，还可以使用 `app_update 343050 validate` stdout 的原生 verifying progress 作为更精确的托管模式来源；Steam Client 模式继续保留本节的真实 I/O 估算方案。

## 0.1.29：Steam 完成同步修复

0.1.28 的问题不是 Steam 未完成，而是 AI Game Manager Panel 把 `sawCommitting` 等历史事件当成了当前状态，并且完成条件仍可能被某一种 `content_log` 终止格式卡死。

0.1.29 改为：

- 当前 AppManifest StateFlags 是实时阶段的第一事实来源；
- `content_log.txt` 只用于证明“本次任务确实进入过验证”、补充错误/扫描结果，不再成为必须出现某条 scheduler 文本的完成锁；
- 一旦本次任务真实观察到 Validating/Busy，随后目标 AppID 回到 `FullyInstalled + idle`，并连续两个轮询保持稳定，即同步完成；
- 历史 `committing` 不再覆盖当前已经 idle 的 AppManifest；
- Steam 校验扫描的精确 GUI 百分比没有稳定公开接口，AI Game Manager Panel 在 Validating 阶段保持真实的“不确定进度”，下载/暂存阶段仍使用 AppManifest 的真实字节百分比。

## 0.1.31：校验会话状态机修复

Steam GUI 在文件校验期间并不保证 `appmanifest_*.acf` 的 `StateFlags` 一定带 `Validating`；实机已观察到 Steam 仍显示 15%/42% 校验时 AppManifest 仍为 `4 (Fully Installed)`。因此 0.1.31 明确调整证据优先级：

```text
本次 steam://validate 请求
  ↓
本次请求后的 content_log Validating 事件
  或连续 Steam 真实读取 I/O（仅作为开始兜底）
  ↓
validating / 近似百分比（最大 99%）
  ↓
本次会话的 File validation finished
  ↓
本次会话的 scheduler finished
  或 validation finished -> Update None -> Fully Installed
  ↓
AppManifest Fully Installed + idle 连续稳定
  ↓
100% / 校验完成
```

硬规则：

- `StateFlags=4` 不能单独完成校验；
- Windows Steam 进程 I/O 只能证明“扫描实际开始并在读盘”，不能证明完成；
- 旧校验会话的终态日志不能完成新的重复校验；
- 没有本次会话终态证据时宁可继续等待，也不能提前 100%。

## 0.1.30：Steam Client 校验进度同步修复

- 新增 Windows Steam 进程实际读取字节采样，不使用按时间递增的模拟百分比。
- 只有目标 AppID 被确认处于本次验证流程时才累计 I/O，避免普通 Steam 后台活动直接驱动校验条。
- 估算值以 AppManifest `SizeOnDisk` 为基准并硬限制 `< 100%`。
- `Validating -> idle` 后先显示 99% 收尾，Steam 稳定终态确认后才显示 100%。
- API 新增 `progressEstimated`，前端会把此类进度显示为“约 xx%”并标注来源。

## 0.1.32：Steam Maintenance 公共化基线

0.1.31 已通过 Windows + Steam 实机验证后，校验 Session 状态机正式冻结为所有 Steam App 的公共实现。

后端唯一入口移动到：

```text
internal/deploy/steam/maintenance/
├─ model.go       # Task/State/Phase/公共接口
├─ service.go     # StartInstall / StartValidate / 防重复 Session
├─ tracker.go     # 本次 content_log Session 事件跟踪
├─ progress.go    # Steam I/O 估算与 99% 上限
├─ monitor.go     # 轮询、阶段推进、终态门禁
└─ service_test.go
```

DST 不再拥有任何 Steam 校验状态机。`343050` 运行中禁止维护的规则回到 `internal/games/steam/dst/maintenance.go`。

前端公共控制器：

```text
frontend/src/games/steam/shared/useSteamMaintenance.ts
```

以后 Palworld、Valheim、Project Zomboid 等 Steam Provider 接入时必须直接复用，不得复制校验逻辑。
