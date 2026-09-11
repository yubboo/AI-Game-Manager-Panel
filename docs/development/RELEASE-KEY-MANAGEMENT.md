# AI Game Manager Panel 发行密钥管理

> 适用版本：0.1.70+

## 目标

发行许可证使用 Ed25519 非对称签名：项目所有者持有私钥，客户端只持有公钥。公开 GitHub 仓库被攻破、源码被完整复制，也不能仅凭公钥伪造新的 BFLC2。

## 信任模型

```text
项目所有者本机 / 离线介质
  agmp-release-private.key
             │
             │ Ed25519 Sign
             ▼
  BFLC2.<payload>.<signature>
             │
             ▼
AI Game Manager Panel 客户端
  vendor_public_keys.json
             │
             └─ Ed25519 Verify
```

客户端内的公钥不是秘密；发行私钥才是根秘密。

## 0.1.70 密钥环

客户端不再只硬编码一个 `DefaultVendorPublicKey`，而是编译公开密钥环：

```text
internal/system/license/vendor_public_keys.json
```

每一项包含：

- `keyId`
- `fingerprint`
- `publicKey`
- `status`: `active` / `retired` / `legacy`
- `createdAt`

`active` 用于签发新证书；`retired/legacy` 继续用于验证以前已经签发的证书。

这意味着以后正常轮换发行密钥时，不需要强制让旧许可证全部失效。

## 第一次初始化

运行：

```text
scripts\tools\license\AGMP-License-Admin.bat
```

选择：

```text
1. 初始化 / 轮换发行密钥
```

工具会：

1. 在 `%LOCALAPPDATA%\AI-Game-Manager-Panel\ReleaseKeys` 下创建新的独立目录。
2. 生成 Ed25519 私钥/公钥。
3. 尝试把 Windows 私钥 ACL 收紧为当前账号。
4. 仅把公钥、KeyID 和指纹加入源码里的公开密钥环。
5. 将之前的 active key 改成 `retired`，历史 `legacy` key 继续保留。
6. 执行 Release Key Gate 与 GitHub Safety Gate。

## 私钥备份

生成后至少做一份 **离线备份**。建议使用你自己控制的加密存储或离线介质。不要把私钥发送到公开 GitHub、Issue、聊天截图、日志、安装包或公共网盘链接。

如果 active 私钥真的永久丢失：

- 已签发许可证仍可由对应旧公钥验证；
- 但无法继续用该 key 签发新许可证；
- 需要生成新 active key，并发布包含新公钥环的新客户端版本。

## 正式 Release 门禁

Windows Wails / Electron、Linux Desktop / Server 正式构建都必须通过：

```text
node scripts/common/check-release-key.mjs
```

开发检查可使用 `--allow-unconfigured`，因此公开源码即使尚未初始化发行私钥，也仍可进行 Development 模式开发。

## GitHub

应该提交：

- `vendor_public_keys.json`
- 发行工具源码
- 文档、测试、Gate

禁止提交：

- `agmp-release-private.key`
- 任何 `.key/.priv/.seed/.pem/.p12/.pfx`
- 实际 BFLC2
- 用户/设备 activation 数据
