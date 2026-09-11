# DST Master + Caves Runtime（0.1.35）

## 目标

同一个 Cluster 只维护一套 Runtime/Console 能力，Master 与 Caves 只是不同 Shard 实例：

```text
Cluster_1
├─ Master · 地面
└─ Caves  · 洞穴
```

前端控制台使用两个 Tab，共用筛选、结构化日志、原始日志、命令输入、自动滚动和日志下载。

## 一键启动

```text
Startup Preflight
  ↓
Shard 拓扑检查
  ↓
端口配置去重
  ↓
Windows 实际 UDP 占用检查
  ↓
安全清理本 Cluster 的 Bonfire 托管残留
  ↓
Master
  ↓
Caves（存在 Caves/server.ini 时）
  ↓
分别监听 Ready
```

任何未知端口占用都不会被自动结束。用户显式选择“强制清理 DST 残留”时，也只允许结束 DST Dedicated Server 进程。

## 一键停止

```text
Caves c_shutdown()
  ↓
等待退出
  ↓
必要时 terminate / kill
  ↓
Master c_shutdown()
  ↓
等待退出
  ↓
确认 UDP 端口释放
```

没有确认 `cmd.Wait()` 返回时，Runtime 保持 `stopping`，不得伪报 `stopped`。

## 双 Shard 必要配置

`cluster.ini`：

- `[SHARD] shard_enabled = true`
- `bind_ip`
- `master_ip`
- `master_port`
- `cluster_key`（Bonfire 只检查“已配置”，绝不返回正文）

`Master/server.ini`：

- `[SHARD] is_master = true`
- `[NETWORK] server_port`
- `[STEAM] master_server_port`
- `[STEAM] authentication_port`

`Caves/server.ini`：

- `[SHARD] is_master = false`
- `[NETWORK] server_port`
- `[STEAM] master_server_port`
- `[STEAM] authentication_port`

双 Shard 模式要求这些端口显式配置且互不重复，以避免两个实例同时落到同一默认端口。
