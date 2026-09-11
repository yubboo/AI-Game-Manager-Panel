# AGMP 许可证发行工具（0.1.71+）

这里是 **AI Game Manager Panel 项目所有者 / 发行方** 的本地工具，不属于最终用户功能。

## 先分清三样东西

- **短 CDK**：`BF-XXXX-XXXX-XXXX-XXXX`。属于未来在线 License Server 的兑换码，当前尚未上线。
- **BFLC2**：`BFLC2.<payload>.<signature>`。当前正式 Release 使用的离线 Ed25519 许可证证书。
- **发行密钥**：Ed25519 私钥 + 公钥。私钥负责签发 BFLC2，公钥负责客户端验签。

## 推荐入口

Windows 直接运行：

```text
scripts\tools\license\AGMP-License-Admin.bat
```

菜单：

1. 初始化 / 轮换发行密钥
2. 查看 Active KeyID / 指纹 / 历史可信公钥
3. 签发 BFLC2 离线许可证（签发后自动回验）
4. 同步本机发行公钥（新源码目录恢复 Active Key，不复制私钥）
5. 验证 BFLC2 许可证（签名 + BFM + BFID + 有效期）

## 私钥保存位置

默认生成到：

```text
%LOCALAPPDATA%\AI-Game-Manager-Panel\ReleaseKeys\<时间戳-随机ID>\
├─ agmp-release-private.key   # 绝密，禁止上传 GitHub
├─ agmp-release-public.pub    # 公钥副本
└─ key-metadata.json           # KeyID / 指纹 / 公钥元数据
```

也可以通过环境变量指定仓库外目录：

```text
AGMP_RELEASE_KEY_DIR=D:\AGMP-Private\ReleaseKeys
```

`keygen` 本身也会检查路径：如果输出目录位于当前源码仓库内部，会直接拒绝生成私钥。

## 公开仓库里保存什么

只保存：

```text
internal/system/license/vendor_public_keys.json
```

这是 **公开发行公钥环**，允许进入 GitHub。0.1.70 起轮换新公钥时不会删除旧公钥，而是将旧 key 标记为 `retired/legacy`，这样以前已经签发的 BFLC2 仍然可以被后续客户端验证。

绝对不能提交：

- `agmp-release-private.key`
- 实际 BFLC2 证书
- activation.json / install.id
- 管理员账号、安全密钥、Token、`.env`
- `runtime/*`、日志和构建产物

## 为什么 0.1.70 源码第一次正式 Release 会被拦截

0.1.68 内置公钥对应的旧发行私钥已经不可用，因此 0.1.70 源码包只把旧公钥保留为 `legacy` 验证钥，不把它继续标记为 active。

源码开发仍然可以使用：

```text
AI-Game-Manager-Panel.bat -> 2. 开发模式
```

但正式 Wails / Electron / Linux Release 会运行：

```text
node scripts/common/check-release-key.mjs
```

如果没有 active 发行公钥，会明确拒绝 Release。若本机已经在旧源码版本生成过发行密钥，优先运行许可证发行控制台第 4 项恢复同一把公钥；只有真正需要首次初始化或轮换时才运行第 1 项。

## BFLC2 签发

签发工具会自动读取 `vendor_public_keys.json` 的 `activeKeyId`，然后从仓库外的 ReleaseKeys 目录寻找匹配私钥。私钥与 active 公钥不一致时会直接拒绝签发，避免拿错密钥。

签发后的证书默认保存到仓库外：

```text
%LOCALAPPDATA%\AI-Game-Manager-Panel\ReleaseKeys\IssuedLicenses\
```

签发完成后会立即使用当前公开密钥环回验；只有回验通过才提示完成。同时会输出完整 BFLC2，也可以直接把生成的 `.bflc` 文件交给对应机器导入。

## 源码开发

开发模式继续使用 `agmp_dev_license` Build Tag，显示 `DEVELOPMENT` 并放行开发调试功能。正式 Release 不带这个 Tag。
