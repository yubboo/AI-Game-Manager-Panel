# Steam 公共平台层

当前实现版本：AI Game Manager Panel-0.1.16

## 边界

`internal/platform/steam` 只提供所有 Steam 游戏都能复用的能力，不能出现 DST、Palworld、Valheim 等具体游戏规则。

```text
Steam Root
    ↓
libraryfolders.vdf
    ↓
Steam Libraries
    ↓
appmanifest_*.acf
    ↓
AppInventory / AppInstallation
```

## Steam 根目录发现

Windows 当前顺序：

1. `STEAM_PATH` 环境变量。
2. `HKCU\Software\Valve\Steam` / `SteamPath`。
3. `HKLM\SOFTWARE\WOW6432Node\Valve\Steam` / `InstallPath`。
4. `HKLM\SOFTWARE\Valve\Steam` / `InstallPath`。
5. `%ProgramFiles(x86)%\Steam`。
6. `%ProgramFiles%\Steam`。

候选目录必须至少存在 `steamapps` 或 Steam 可执行文件才视为有效。

## Steam Library

主 Steam 根目录作为 Primary Library，再读取：

```text
<Steam>/steamapps/libraryfolders.vdf
```

AI Game Manager Panel 去重并验证每个 Library 的 `steamapps` 目录。

## 0.1.11 AppManifest Inventory

AI Game Manager Panel 扫描：

```text
<Library>/steamapps/appmanifest_*.acf
```

通用字段：

- AppID
- name
- installdir
- buildid
- LastUpdated
- SizeOnDisk
- StateFlags
- ManifestPath
- LibraryPath
- InstallPath
- InstallPathExists

### 容错

单个 manifest 出现以下问题不会让整个扫描失败：

- 数值字段损坏。
- AppID 内容与文件名不一致。
- `installdir` 缺失。
- `installdir` 不安全。
- 重复 AppID。

问题进入 `AppInventory.Warnings`，其他应用继续返回。

### 路径安全

`installdir` 只允许相对目录。AI Game Manager Panel 拒绝：

- `../` / `..\\` 目录穿越。
- Windows 绝对路径。
- Unix 绝对路径。
- UNC 路径。

Steam manifest 是本地输入，但公共平台层仍不应盲目信任路径数据。

## FindApp

具体游戏以后只需要：

```text
FindApp(AppID)
```

例如未来 DST Game Provider 可以查询其 Dedicated Server AppID，但 AppID 本身和 DST 校验规则必须放在 DST 模块，而不是 Steam 公共层。


## 0.1.14 原子 Steam Snapshot

UI 刷新不再分别调用“检测环境”和“扫描 AppManifest”。现在只调用一次：

```text
SteamSnapshot
    ↓
Detect Steam Root / Libraries（一次）
    ↓
基于同一 Environment 扫描 appmanifest_*.acf
    ↓
Environment + AppInventory + 扫描时间 + 耗时
```

前端采用 stale-while-revalidate：刷新时继续显示上一次成功数据，只改变“扫描中”状态；新快照完成后一次性替换，避免大块 DOM 被卸载/重建导致 WebView2 重绘闪烁。

## 后续真实需求再加入

- SteamCMD
- InstallApp
- UpdateApp
- VerifyApp
- Workshop
- 下载 / 更新任务进度

不提前创建空模块。

## 0.1.16 定向 AppID 检测

Game Workspace 不调用完整 Steam AppInventory 扫描。当前选中的 Game Provider 提供所需 Steam AppID，平台层只尝试读取：

```text
<Library>/steamapps/appmanifest_<appid>.acf
```

因此点击 DST 时只检测 322330 / 343050，不会把 Counter-Strike、Wallpaper Engine 等无关 Steam 应用加载到用户界面。

完整 AppManifest 枚举能力仍保留，供“运行环境 / Steam 诊断”使用。

当前 DST Catalog：

```text
游戏本体 AppID: 322330
Dedicated Server AppID: 343050
```

## 0.1.32：Platform 与 Maintenance Service 分层

`internal/platform/steam` 继续只做低层 Steam 能力；安装/校验的长任务状态机不放在 Platform 文件夹中，而统一位于：

```text
internal/deploy/steam/maintenance
```

这样可以同时满足：

- Steam 的解析/协议/I/O 能力可被多个 Service 使用；
- 校验 Session 状态机只有一份；
- DST/Palworld/Valheim 等游戏模块只声明 AppID 与游戏侧安全策略；
- Application/Bridge/UI 不复制底层判定。
