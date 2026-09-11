# AI Game Manager Panel 许可证、CDK 与设备绑定

> 适用版本：AI Game Manager Panel 0.1.74。许可证属于 Go Core；Wails、Electron 与本机 Web 读取同一个状态。

## 五个核心 ID

- **UserID**：账号身份，回答“谁在使用”。
- **LicenseID**：许可证唯一编号。
- **MachineCode (`BFM-...`)**：设备指纹。Windows 使用稳定系统标识的 AI Game Manager Panel SHA-256 派生值，不展示原始 MachineGuid。
- **InstallID (`BFID-...`)**：当前 AI Game Manager Panel 安装实例 ID。升级会兼容迁移旧版 `data/license/device.id`；0.1.65+ 新默认位置位于 `runtime/data/license/install.id`，避免已有许可证因 BFID 改变失效。
- **ActivationID**：一次 License 与 MachineCode + InstallID 的设备绑定记录。

## CDK 与许可证证书不是同一个东西

- **短 CDK**：`BF-XXXX-XXXX-XXXX-XXXX`，用于未来 License Server 在线兑换。0.1.70 仍未上线 License Server，客户端中的在线 CDK 输入会明确显示“尚未上线”并保持禁用。
- **License Certificate**：`BFLC2.<payload>.<signature>`，是当前可用的离线 Ed25519 签名许可证。
- 客户端只内置发行公钥；发行私钥不得进入源码、Git、客户端、安装包、日志或最终用户目录。

`BFLC2` Payload 至少包含：LicenseID、ActivationID、Product、Edition、Features、SeatLimit、MachineCode、InstallID、IssuedAt、ExpiresAt。

## 授权与登录分离

Owner 创建/登录只负责身份认证。License 不得放到登录前阻塞。未授权用户仍可进入设置、许可证、环境配置和基础诊断；高级功能通过 Go Core Feature Entitlement Gate 判定。

授权状态包括：未激活、已激活、即将到期、已过期、设备不匹配、已吊销、席位已满、离线宽限期、许可证服务器不可达。

## 设备绑定 / 解绑

当前版本支持本地证书绑定和“解绑当前设备”。解绑只删除本机许可证记录，不删除账号和服务器数据。未来云端 License Server 负责设备列表、Seat 管理、远程解绑和旧电脑损坏后的自助释放；这些能力在服务未上线前不得在客户端伪装成功。

## 三端归属

同一台机器：

```text
License Service
      │
   Go Core
   ├─ Wails
   ├─ Electron
   └─ localhost Web
```

浏览器本身不拥有 MachineCode。未来远程 Web 平台的设备绑定对象应是运行 Go Core / Agent 的服务器节点。

## 离线签发

开发阶段可使用 `cmd/aigame-manager-license-admin` 生成 Ed25519 签名 BFLC2 证书。发行私钥必须离线保存；客户端只负责公钥验签。历史 BFCDK1/BFCDK2 保留只读迁移兼容，不再作为新版 CDK 定义。

## 源码开发模式（0.1.67+）

为了避免开发者必须持有真实离线 CDK 才能调试源码，0.1.67 新增 `agmp_dev_license` Go Build Tag。该模式只由开发入口启用，Feature Entitlement 返回开发态全功能；正式 Wails/Electron/Web Release 不带该 Tag，继续执行正式许可证校验。

这不是一个可传播的“万能 CDK”，也不会要求把发行私钥放入源码。发行私钥继续必须离线保存并被 GitHub Safety Gate 排除。



## 0.1.70 发行密钥生命周期

0.1.70 不再把“单一硬编码公钥 + 不知道私钥在哪里”当作可持续发行方案，而是引入公开发行公钥环 `vendor_public_keys.json`。旧公钥可以标记为 `legacy/retired` 继续验证历史 BFLC2，新公钥标记为 `active` 用于签发新证书。

项目所有者必须通过 `scripts/tools/license/AGMP-License-Admin.bat` 在 **源码仓库外** 本机生成 Ed25519 私钥。默认位置是 `%LOCALAPPDATA%\AI-Game-Manager-Panel\ReleaseKeys`。源码只更新公开 KeyID、指纹和公钥；私钥绝不能进入 GitHub。

BFLC2 0.1.70 起会记录 `issuerKeyId`。正式 Release 会运行 Release Key Gate：没有 active 发行公钥时拒绝构建；源码 Development 模式仍使用 `agmp_dev_license`，不要求发行私钥或 BFLC2。

## 0.1.68 Windows 黑框与设备标识修复

旧版本为了取得 Windows `MachineGuid` 会启动 `reg.exe query ...`。Wails 属于 GUI 进程，Windows 在某些机器上会为这个子进程短暂创建控制台窗口，因此在登录后加载许可证、进入授权页、刷新状态或 Feature Gate 查询时会看到黑框一闪。这个现象与 CDK 是否有效没有关系。

0.1.68 改为直接调用 Windows Registry API 读取 `MachineGuid`，不再创建 `reg.exe` 子进程；同时 MachineCode 与 InstallID 都进行进程内缓存。前端设置中心/安全中心复用同一份许可证状态，不再因为组件重新挂载而把标识临时清空成“读取中…”。
