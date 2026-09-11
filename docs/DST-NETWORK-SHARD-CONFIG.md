# DST 网络与 Shard 配置向导（AGMP 0.1.36）

## 目标

普通用户不应该为了启动 Master + Caves 手工理解或编辑 `cluster.ini` / `server.ini`。

AGMP 将网络配置分成两层：

1. **推荐配置**：一键修复缺失/重复端口，适合绝大多数单机单 Cluster 场景。
2. **高级配置**：多开服务器、端口映射或特殊网络环境才需要使用。

## AGMP 推荐配置

| 用途 | 配置文件 | 推荐端口 |
| --- | --- | ---: |
| Master / Caves 内部 Shard 联动 | `cluster.ini [SHARD] master_port` | 10888 |
| Master 玩家连接 | `Master/server.ini [NETWORK] server_port` | 10999 |
| Master Steam master | `Master/server.ini [STEAM] master_server_port` | 27016 |
| Master Steam auth | `Master/server.ini [STEAM] authentication_port` | 8766 |
| Caves 玩家连接 | `Caves/server.ini [NETWORK] server_port` | 11000 |
| Caves Steam master | `Caves/server.ini [STEAM] master_server_port` | 27017 |
| Caves Steam auth | `Caves/server.ini [STEAM] authentication_port` | 8767 |

这些是 AGMP 为单个两层 Cluster 提供的推荐组合。官方要求多层 Cluster 的各 Server 使用不同玩家端口，并要求同机多个 Server 的 Steam 内部端口不要冲突。

## 一键修复流程

```text
Preflight 发现端口缺失 / 重复
        ↓
网络与端口
        ↓
一键修复并应用
        ↓
备份原 cluster.ini / server.ini
        ↓
校验 1024-65535 + 配置内无重复
        ↓
原子写入
        ↓
重新 Preflight
        ↓
检查 Windows 实际 UDP 占用
        ↓
允许启动 Master + Caves
```

## 安全规则

- Master / Caves 运行期间禁止修改端口。
- 保存前始终检查端口范围和配置内部重复。
- 保存前创建本地备份；备份不进入前端，也不主动进入诊断包。
- 实际端口占用仍由 `platform/netports` 检查；配置正确不等于端口当前一定空闲。
- AGMP 不会因为端口冲突自动结束未知程序。

## 业务归类

```text
internal/games/dst/dedicated/
├─ ports.go          # 读取实际 PortPlan
└─ portconfig.go     # 推荐配置、可视化配置读写、备份与校验

internal/games/dst/runtime/
└─ cluster.go        # 运行期禁止修改、应用后实际 UDP 检查

frontend/src/games/steam/dst/components/
└─ NetworkConfig.vue # 普通一键修复 + 高级可视化配置
```

`platform/steam` 不承载 DST 网络配置；这是 DST Dedicated Server 业务。
