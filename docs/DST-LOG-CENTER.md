# DST 日志中心（当前实现基线：AGMP 0.1.38）

行为参考：DSTCamp 1.3.5 `dstools/features/local_service/log_bundle.py`、`server_diagnostics.py` 与本地服务器页面的日志诊断流程。

## 目标

日志功能不能反过来拖慢 Dedicated Server。AGMP 将“实时控制台”和“历史日志”分成两个层次：

1. Runtime 内存环形缓冲只服务当前实时 UI；
2. 持久化日志通过非阻塞队列由独立 goroutine 写盘；
3. 历史查看按 cursor 分页；
4. 搜索在 Go 后端流式扫描；
5. 导出统一由 Go 后端写入配置的日志根目录；0.1.65 新默认是 `runtime/log/`，保留 HTTP 流式下载端点用于显式下载场景。

## 性能边界

- Runtime 实时流缓冲：3000 行；
- Runtime 近期诊断快照：500 行；
- 持久化队列：16384 行；
- 持久化 bufio：256 KiB；
- 常规刷盘周期：1 秒；元数据周期：5 秒；
- 单页后端读取上限：2000 行；
- 单次搜索后端上限：1000 条匹配；当前 Vue 搜索默认只取 300 条；
- Vue 历史日志查看器最多保留 1500 行；
- 单个诊断包日志文件最多 32 MiB。

`Append` 永不等待磁盘。如果持久化队列在极端磁盘阻塞下被写满，AGMP 会增加 `droppedLines` 并继续消费 Dedicated Server stdout，优先防止服务器因为日志管道背压卡死。日志会话同时记录持久化错误，UI 可明确提示。

## 当前运行日志的一致性

当前会话仍可能有内容停留在 bufio 中。`Read/Tail/Search/Diagnostics/Download` 会请求一个有界 flush barrier：先处理 barrier 被接受时已经排队的日志，再刷盘，然后返回。新的服务器输出仍可继续入队，不会让读取一直等待“队列彻底为空”。

Runtime 结束时会尽量把队列完整落盘；持久化收尾最多等待 5 秒，不允许损坏/卡死的磁盘让服务器生命周期永久挂起。

## 存储

默认根目录：

```text
<AGMP 项目/可执行文件根目录>\runtime\log\
  bonfire.log
  dst\
    <Cluster>-<path-hash>\
      Master\
        YYYYMMDD-HHMMSS-Master-<id>.log
        YYYYMMDD-HHMMSS-Master-<id>.json
      Caves\
        ...
  <手动导出的日志>.log
  AGMP_日志_<Cluster>_<时间>.zip
```

路径 hash 用于区分不同目录中同名 Cluster。元数据包含状态、PID、Ready、退出码、行数、字节数、丢弃行数和持久化错误。

## 日志中心能力

- 运行会话列表；
- 从头读取；
- 尾部读取；
- cursor 继续读取；
- 关键字 + 等级 + 分类搜索；
- 单日志导出到当前配置的日志根目录（0.1.65 新默认 `runtime/log/`）；
- 诊断摘要；
- Cluster 安全诊断包。

日志等级为启发式分类，不修改原始文本。分类包括：`steam`、`mod`、`network`、`security`、`world`、`general`。

## 诊断策略

当前已覆盖：

- Steam GameServer 初始化；
- Klei geo DNS 注册；
- Cluster Token 读取；
- Master/Secondary Ready；
- `modoverrides.lua` 执行失败；
- Token 注册冲突；
- 端口占用/绑定失败；
- VC++ Runtime/DLL 缺失；
- 文件读写权限；
- worldgen 任务集/加载异常；
- Lua traceback；
- Windows 防火墙端口状态无法确认；如果后续已成功监听端口并 Ready，则该提示自动判定为已解决，不保留为当前故障；
- `-console` 参数弃用警告；
- 无明确证据时的异常退出兜底。

为保持原 DSTCamp 的安全网，Diagnostics 除 AGMP 自己持久化的 stdout/stderr 外，还会读取 `<Cluster>/<Shard>/server_log.txt` 最后 700 行；最近已经出现过的相同文本不会重复计数。

这还**不等于完整迁移**原 `server_diagnostics.py`：更深的 Mod 归因、missing_mods、Legacy V1、Token 调度与端口 preflight 仍留在迁移矩阵中。

## 诊断包安全边界

包含：

- 当前 Shard `server_log.txt`；
- `backup/server_log/server_log_*.txt` 中最多 5 个日志（错误特征优先，其次按修改时间）；
- `server.ini`；
- `modoverrides.lua`；
- `leveldataoverride.lua`；
- `manifest.txt`；
- `mod_list.txt`。

明确排除：

- `cluster_token.txt`；
- `adminlist.txt`；
- `blocklist.txt`；
- `whitelist.txt`。

后续如果诊断包需要增加新文件，必须先审计是否含账号、Token、隐私或授权信息。

## Desktop / Web

两端共享 `internal/games/dst/logcenter`：

- Desktop 通过 Wails Bridge；
- Web 通过 `/api/v1/dst/logs/*`；
- Desktop/Wails 与本机 Web 的“导出日志/诊断包”都调用同一个 Go Export 服务并写入当前配置的日志根目录（0.1.65 新默认 `runtime/log/`）；
- HTTP 仍保留 `ServeContent` 流式下载端点给显式下载/未来远程场景；
- 不复制一套浏览器专属日志实现。

当前 Web 仍只绑定 localhost；开放远程访问前必须先完成认证与权限边界。
