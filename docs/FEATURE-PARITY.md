# AI游戏管理器面板功能等价验收清单

本清单来自 DSTCamp 1.3.5 README、源码模块与测试入口。只有“实现 + 自动测试 + Windows 实机验证”全部通过，才可标记为完成。

## 平台公共能力状态（0.1.67）

- [x] Windows Desktop / Web 共用 Go Core
- [x] Electron Windows Desktop Adapter（0.1.52：与 Wails/Web 共用 Vue UI + Go Core，仅增加桌面 Shell）
- [x] Linux Server Edition amd64/arm64 静态控制面二进制
- [x] Linux systemd 安装骨架与交互式管理脚本
- [x] Docker 多架构 Server Edition 基础镜像
- [x] 首次 Owner Bootstrap 与公开注册永久关闭（0.1.45 自动测试通过，待 Windows UI 实机验收）
- [x] PBKDF2 密码存储与 Bearer Session（0.1.45 自动测试通过）
- [x] Web `/api/v1/*` 默认 Session 门禁（0.1.45；远程监听仍默认关闭）
- [x] 首次环境初始化状态与 SteamCMD / DST Dedicated Server 探测（0.1.45 自动测试通过）
- [x] 登录安全密钥二次验证（0.1.51：每账号独立配置、密钥哈希保存、轮换与登录门禁）
- [x] 首次环境初始化可持久化跳过并在运行环境页补初始化（0.1.51）
- [ ] RCON 控制台（当前仅模块骨架）
- [ ] GIF/WEBP 自定义背景（当前仅模块骨架）
- [ ] 通用 SteamCMD 一键安装中心（0.1.45 已完成首次启动所需的 Windows SteamCMD 托管安装；完整通用 SteamCMD Manager 尚未完成）
- [ ] Docker 环境依赖可视化安装/检测（当前已有镜像，不等于环境管理中心完成）
- [ ] 全局 CPU/内存/磁盘/网络/实例资源监控
- [ ] 全局活跃端口/活跃进程中心（DST 端口预检已部分实现）
- [ ] 通用游戏配置 Schema 可视化编辑器
- [ ] 在线 Mod 管理中心
- [ ] 跨游戏/跨节点多实例统一管理
- [ ] 全局备份/快照恢复中心
- [ ] 通用玩家白名单/黑名单/管理员中心
- [x] DST 实时日志/实时控制台基础能力
- [ ] 通用游戏自动更新检测与更新策略中心

> 注意：AI Game Manager Panel 控制面支持 linux/arm64，不代表每个游戏服务端都支持 ARM64；具体由游戏模板声明。

## P0：应用与发布

- [x] Windows 桌面入口（Wails）
- [x] 浏览器 Web 入口（HTTP + 同一 Vue 前端）
- [x] Desktop/Web 共用 Go Application/Core
- [ ] 单实例与已有窗口激活
- [ ] 系统托盘
- [ ] 中英文完整文案
- [ ] 五套主题
- [ ] 三种字体
- [ ] 自定义背景
- [ ] 窗口状态记忆
- [ ] 启动更新检查
- [ ] 自动更新 + SHA-256 + 回滚

## P0：DST/平台发现

- [x] Steam 根目录与多 Library
- [x] AppManifest 定向识别
- [x] Windows 真实 Documents 特殊目录
- [x] Steam `DoNotStarveTogether` 根目录发现
- [x] WeGame `DoNotStarveTogetherRail` 根目录发现
- [x] SERVER / LOCAL Cluster 区分
- [x] Shard `server.ini` 有效性扫描
- [x] Dedicated Server 32/64 位严格验证（0.1.22）
- [x] 手动确认专服路径优先级（0.1.22）
- [x] `-conf_dir` 跨盘行为（0.1.22）
- [ ] UGC/Workshop 共享目录大小写兼容

## P0：本地专服

- [ ] 启动/停止 Master（0.1.24 已实现 Go Runtime + Desktop/Web 接口，待 Windows 实机验收后勾选）
- [ ] 启动/停止 Caves/Secondary
- [x] Master Ready 日志判定纯解析逻辑（0.1.22）
- [x] Master Ready 真实进程日志消费（0.1.24 已由 Windows 实机日志确认 `Sim paused` / Ready）
- [x] Secondary Ready 日志判定纯解析逻辑（0.1.22；Caves 真实进程尚未开放）
- [ ] 崩溃状态（0.1.24 已实现意外退出/退出码状态收口，待 Windows 实机验收）
- [ ] 控制台输入与历史（0.1.24 已实现 stdin 命令输入；历史浏览仍待迁移）
- [ ] 公告
- [ ] 玩家列表
- [ ] 回档
- [ ] 端口预检与多 Cluster 冲突
- [ ] Startup Preflight（0.1.33 已实现 Dedicated/conf_dir/Cluster/Master/Token 基础门禁，待 Windows 实机验收后勾选）
- [ ] VC++ 运行库诊断
- [ ] 日志诊断与日志打包（0.1.25 已实现持久化日志中心、流式搜索/下载、主要诊断与安全诊断包；待 Windows 实机验收，完整 Mod 归因仍待迁移）
- [ ] Token 调度/冲突保护（0.1.33 已完成基础 Token 安全写入、运行时替换保护与启动前缺失拦截；轮换/调度仍待迁移）
- [ ] Steam 客户端专服安装/更新/校验流程

## P0：服务器配置

- [ ] `cluster.ini` 读写
- [ ] `server.ini` 读写
- [ ] 已确认默认值回填
- [ ] Token（0.1.33 已实现上传/粘贴/路径导入、隐私保护与 Preflight，待 Windows 实机验收后勾选）
- [ ] adminlist
- [ ] blocklist
- [ ] whitelist

## P1：存档与备份

- [ ] 存档浏览
- [ ] 玩家角色状态读取
- [ ] 手动备份
- [ ] 自动备份
- [ ] 恢复
- [ ] 复制为服务器存档（0.1.33 已实现 LOCAL/外部 Cluster → SERVER Cluster 复制，待 Windows 实机验收后勾选）
- [ ] 多世界存档创建
- [ ] 完整存档打包分享

## P1：世界设置

- [ ] 森林独立配置
- [ ] 洞穴独立配置
- [ ] 图标与取值说明
- [ ] 创建世界
- [ ] 岛屿冒险/猪镇等 Mod 世界设置
- [ ] 世界设置兼容审计

## P1：Mod 管理

- [ ] Steam Mod 扫描
- [ ] WeGame Mod 扫描
- [ ] 启用/停用
- [ ] 配置集
- [ ] 目录联接/共享 UGC
- [ ] Workshop Manifest 状态
- [ ] 本地版本证据
- [ ] Lua 5.1 受限沙箱
- [ ] Mod 图标
- [ ] 中文翻译
- [ ] 大型 Mod 库异步分批刷新
- [ ] V1 Legacy 校验/部署/回滚
- [ ] 残留目录清理
- [ ] 启动前 Mod 完整性诊断

## P1：网络与穿透

- [ ] SakuraFrp
- [ ] frpc 缺失恢复
- [ ] 自建 frps
- [ ] SSH 部署
- [ ] 线路连通性诊断
- [ ] Mihomo TUN
- [ ] WireGuard 大厅加速
- [ ] 公网 IP 回退查询
- [ ] Windows Defender 排除项检测/入口
- [ ] 局域网/公网模式冲突保护

## 自动化基线

原项目 17 个 `tests/test_*.py` 必须逐项建立 Go/前端等价测试，不允许只删除旧测试：

- [ ] `test_auto_update.py` → 等价测试待迁移
- [ ] `test_build_pipeline.py` → 等价测试待迁移
- [ ] `test_e2e.py` → 等价测试待迁移
- [ ] `test_e2e_phase2.py` → 等价测试待迁移
- [ ] `test_gui_cursors.py` → 等价测试待迁移
- [ ] `test_legacy_v1.py` → 等价测试待迁移
- [ ] `test_lobby_accel.py` → 等价测试待迁移
- [ ] `test_lobby_diagnostics.py` → 等价测试待迁移
- [ ] `test_mod_shared.py` → 等价测试待迁移
- [ ] `test_multi_cluster_ports.py` → 等价测试待迁移
- [ ] `test_sakura_frpc_recovery.py` → 等价测试待迁移
- [ ] `test_server_diagnostics.py` → 等价测试待迁移
- [ ] `test_server_mod_status.py` → 等价测试待迁移
- [ ] `test_steam_client_updater.py` → 等价测试待迁移
- [ ] `test_token_scheduling.py` → 等价测试待迁移
- [ ] `test_windows_defender.py` → 等价测试待迁移
- [ ] `test_world_mod_compat.py` → 等价测试待迁移

- [x] Steam 官方文件完整性校验（0.1.24：322330/343050 validate URI + AppManifest 监控，已由用户 Windows 实机验证）
