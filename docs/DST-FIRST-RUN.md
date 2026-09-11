# DST 首次开服链路（AGMP 0.1.33）

AGMP 的 DST 首次开服流程固定为：

```text
Steam / 文件完整性
  ↓
Klei 服务器令牌
  ↓
SERVER Cluster
  ↓
启动前 Preflight
  ↓
Dedicated Server Runtime
  ↓
Steam Init → Klei 注册 → World Ready
```

## 1. Steam 文件

游戏本体 AppID `322330` 与 Dedicated Server AppID `343050` 继续复用公共 Steam Maintenance Service。

- 未安装：交给用户自己的 Steam 官方客户端安装；
- 已安装但不确定：允许再次使用 Steam 官方校验；
- Dedicated Server 运行时禁止安装/校验；
- Steam 校验的 Session/进度/99% 完成门禁属于 Steam 公共业务，不允许 DST 复制实现。

## 2. Klei 服务器令牌

令牌属于 DST/Klei 游戏业务，不属于 Steam 平台能力。

目录：

```text
internal/games/steam/dst/token/
```

支持：

- 前端选择 `cluster_token.txt` 并一次性读取后提交；
- 手动粘贴令牌；
- 高级用户通过本机文件路径导入；
- 保存位置固定为当前 SERVER Cluster 根目录的 `cluster_token.txt`；
- 保存采用同目录临时文件 + fsync + 替换；
- Token 正文不返回给 UI；
- Token 正文不写 AGMP 日志；
- Token 正文不进入 localStorage / Pinia 持久化；
- Token 正文不进入 DST 诊断包；
- 服务器运行时禁止替换令牌。

Klei 官方令牌页面：

`https://accounts.klei.com/account/game/servers?game=DontStarveTogether`

AGMP 只打开官方页面，不接管 Steam/Klei 密码、Cookie、2FA 或登录会话。

## 3. Cluster 导入

目录：

```text
internal/games/steam/dst/cluster/
```

导入要求：

- 源目录必须包含 `cluster.ini`；
- 必须包含 `Master/server.ini`；
- 复制到当前 Klei `DoNotStarveTogether` SERVER 根目录；
- 原世界不修改；
- 符号链接与特殊文件拒绝导入；
- 如果源 Cluster 有 `cluster_token.txt`，默认跳过，不复制服务器凭据；
- 服务器运行时禁止导入。

这允许把客户端 LOCAL 世界复制成 Dedicated Server 可使用的 SERVER Cluster。

## 4. 启动前 Preflight

目录：

```text
internal/games/steam/dst/preflight/
internal/games/dst/setup/
```

当前阻止启动的检查：

- Dedicated Server 是否有效；
- Klei `-conf_dir` 是否可用；
- 是否为 Steam SERVER Cluster；
- `cluster.ini`；
- `Master/server.ini`；
- `cluster_token.txt` 是否存在且非空。

Caves 文件夹存在但缺少 `server.ini` 时当前只警告，不阻止 Master 启动。

Preflight 不只是 UI。`internal/games/dst/runtime.StartMaster` 在真正创建进程前再次调用 Preflight；只要存在 blocker，就不允许 Dedicated Server 进程启动。

## 5. 目录职责

```text
internal/games/steam/dst/
├─ token/       # Token 规则、状态和安全文件写入
├─ cluster/     # Cluster 导入规则
└─ preflight/   # DST 启动检查规则

internal/games/dst/setup/
└─ service.go   # Workspace + Token + Cluster + Preflight 编排

internal/games/dst/runtime/
└─ service.go   # Runtime；启动前强制消费 Preflight
```

Steam 公共层不得出现 Klei Token、Cluster_1、Master/Caves 等 DST 专属业务。

## 0.1.34 Klei 官方配置包

Klei “下载设置”得到的 ZIP 可直接交给 AGMP。默认仅提取并安全保存 `cluster_token.txt`；用户明确选择“完整 Klei 配置”时，才会应用 `cluster.ini`、`Master/server.ini` 与可选的 `Caves/server.ini`，并先备份现有配置文件。世界存档不会被覆盖。

ZIP 导入属于 `games/steam/dst/kleiarchive` 游戏业务，不进入 `platform/steam`。
