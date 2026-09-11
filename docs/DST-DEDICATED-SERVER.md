# DST Dedicated Server 迁移说明（更新至 0.1.25）

行为基线：DSTCamp 1.3.5 `dstools/features/local_service/dedicated_server.py`。

## 本版迁移范围

### 1. Dedicated Server 严格验证

Bonfire 不再只依赖 Steam AppManifest 判断“已安装”。有效专服目录必须：

1. 目录真实存在；
2. 目录名包含 `Dedicated Server`；
3. 至少存在以下一组专用 EXE：
   - `bin64/dontstarve_dedicated_server_nullrenderer_x64.exe`
   - `bin/dontstarve_dedicated_server_nullrenderer.exe`

这条目录名规则来自原 DSTCamp，用于防止把普通 DST 游戏客户端误判成专用服务器，因为游戏本体目录也可能包含类似 nullrenderer EXE。

### 2. 位数选择

优先：64 位；不存在时回退 32 位。

统一输出：安装根目录、bin 目录、EXE、bitness、architecture、来源。

### 3. 路径优先级

顺序保持原行为：

```text
用户手动确认路径
  ↓ 无效则继续
Steam Library 1
Steam Library 2
...
```

手动路径持久化在 Bonfire 自己的 DST 偏好中，不写入 Klei/Steam 文件。

### 4. -conf_dir

DST `-conf_dir` 是相对于真实 Windows `Documents/Klei` 的相对路径：

- 默认 `<Documents>/Klei/DoNotStarveTogether`：不传 `-conf_dir`；
- 自定义同盘路径：计算相对路径；
- 跨盘：返回明确错误，不伪造不可用参数。

### 5. LaunchSpec

Go Core 统一生成：

```text
[-conf_dir <relative>]
-cluster <cluster>
-shard <shard>
[-ugc_directory <path>]
[extra args...]
```

Desktop 与 Web 共用同一生成器。

Bonfire 0.1.34 起不再传入已弃用的 `-console` 参数。启动前会确保当前 Cluster 根目录 `cluster.ini` 的 `[MISC]` 中存在 `console_enabled = true`，Master/Caves 的 `server.ini` 继续只承担 Shard、端口等配置。

### 6. Ready 日志标记

正式启动分界线：

- `About to start a shard with these settings`
- `About to start a server with the following settings`

Master Ready：

- `Sim paused`
- `Sim unpaused`
- `DST_Master_Ready`（旧版兼容）

Secondary Ready：

- `is now ready!`

`Reset() returning` **不作为 Master Ready**，因为临时 worldgen 流程也可能打印它。

## 0.1.24 新增：真实 Master Runtime

本版继续对照 `ServerProcess` / `ServerManager`：

- 直接创建 Dedicated Server 子进程；
- Windows 使用隐藏控制台创建参数，不弹独立 CMD/服务器黑框；
- stdin 管道发送 DST 命令；
- stdout + stderr 合流后实时读取；
- Master 日志实际消费 `AdvanceWorldReadyMarker`，只有正式 `Sim paused/Sim unpaused/DST_Master_Ready` 后 `worldReady=true`；
- PID、退出码、意外退出/崩溃状态；
- 完整 `c_shutdown()` 命令识别，公告文本中出现 `c_shutdown()` 不会误判；
- 停止顺序保持 Python 基线：`c_shutdown()` → 等待 → terminate → 等待 → kill → 最后等待；
- Manager key 使用 `(cluster.path, shard)`，不同目录中的同名 Cluster 不会互相覆盖；
- 重复启动同一运行中 Shard 时复用已有进程引用，不产生孤儿进程；
- Application 退出时会收口 Bonfire 自己启动的 DST 子进程，避免遗留孤儿进程。

为同时支持 Desktop 和 Web，实时日志采用带 sequence/cursor 的有限环形流缓冲；同时保留最近 500 行诊断快照。这样两个客户端不会因为其中一个读取日志而把另一个客户端的日志“取走”。

## 仍未迁移

- Caves/Secondary 真实进程联动；
- 控制台命令历史 UI；
- Mod 启用/加载/失败集合与 `missing_mods` 完整性诊断；
- LuaJIT `bin64_override`；
- WeGame 外部 DST 进程识别；
- PID/UDP 端口映射；
- 启动前 Token、端口、Legacy Mod、Lobby Accel 等完整 preflight；
- 退出程序时“有服务器运行”的交互确认框（当前安全策略为退出时自动优雅停止由 Bonfire 管理的进程）。

这些不会假装完成，会在后续版本继续对照原 Python 迁移。

## 0.1.24：工作台与 Steam 文件校验

DST 工作台改为“全局导航 + DST 二级导航 + 主工作区”的三栏结构，主工作区在宽屏下居中并扩展。
“概览”与 Master 实时控制台合并，避免服务器状态与日志分散在两个页面。

文件完整性校验继续保持 DSTCamp 的 Steam 客户端托管方式：对已安装 App 发送
`steam://validate/<appid>`，然后轮询本机 AppManifest；URI 请求本身不能被当作完成回调。
当前支持 322330 游戏本体与 343050 Dedicated Server。343050 校验在 Bonfire 管理的 DST
进程运行时会被拒绝。


## 0.1.25：日志持久化与诊断

Master Runtime 的 stdout/stderr 现在同时进入有限实时缓冲与非阻塞持久化队列。完整设计、性能边界、下载/搜索和 DSTCamp 日志包迁移见 `DST-LOG-CENTER.md`。Caves 尚未接入真实进程，但日志层已按 Shard 通用设计，可直接复用。


## 0.1.35 Master + Caves 与端口生命周期

Bonfire 的 Dedicated Runtime 现在以 Cluster 为默认管理单位：

```text
Cluster
├─ Master · 地面
└─ Caves  · 洞穴
```

- 一键启动会在启动任何进程之前读取 Master/Caves 的显式 UDP 端口计划并检查系统实际占用。
- 双 Shard 要求 `cluster.ini [SHARD]` 与两个 `server.ini` 的 Shard/端口配置明确且互不冲突。
- 停服顺序固定为 Caves → Master；只有 `cmd.Wait()` 真正确认进程退出后才允许报告 stopped。
- 端口不是独立“关闭”的对象；端口由进程持有，Bonfire 通过停止正确的进程来释放端口。
- 自动清理只针对本次 Bonfire 已知、同 Cluster 的托管残留；未知进程必须由用户明确处理。
- 显式强制清理也只允许结束 DST Dedicated Server 进程，避免误杀其他占用相同端口的应用。
