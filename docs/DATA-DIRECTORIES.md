# AI Game Manager Panel 安装目录与数据目录

AI Game Manager Panel 0.1.65 将**源码目录**与**可写运行数据**明确分开，避免项目根目录随着开发不断增加一级文件夹。

## SourceRoot（源码）

源码根目录只保留稳定工程边界。安装器、Docker、第三方许可位于 `distribution/`；版本历史统一位于 `docs/PROJECT-HISTORY.md`。

## InstallRoot（程序安装目录）

Windows 默认：

```text
%LOCALAPPDATA%\Programs\AI Game Manager Panel
```

用户仍可选择 D/E 盘。程序文件、`configs/` 与安装后需要展示的 `licenses/` 位于安装目录。

## RuntimeRoot（运行根目录）

0.1.65 新项目/新安装的默认 `configs/paths.json` 使用：

```text
<RuntimeRoot>/runtime/
├─ data/
├─ log/
├─ backups/
├─ instances/
├─ temp/
├─ exports/
├─ plugins/
└─ cache/
```

这样源码或安装根目录只出现一个 `runtime/`，而不是八个并列运行目录。

## 兼容旧版本

0.1.64 及更早版本的 `configs/paths.json` 可能仍是 `data`、`log`、`backups` 等根目录相对路径。0.1.65 **继续尊重现有配置，不强制迁移、不自动移动用户数据**。Windows 安装器升级时也不会覆盖用户已有 `configs/`。

## Linux

Linux Server 继续通过 `AGMP_ROOT` 指定用户可写运行根目录；新默认配置在该根目录下使用 `runtime/*`。系统服务本体与运行数据仍分离，服务默认不以 root 长期运行。
