// Package auth 提供 AI Game Manager Panel 本地控制面的身份认证、账号和会话服务。
//
// 0.1.45 建立首次 Owner Bootstrap、PBKDF2 密码存储、Bearer Session、
// 管理员创建账号与 bootstrap.lock 失效关闭门禁。0.1.50 增加每账号独立的
// 登录安全密钥：密钥明文只在生成时返回一次，账号库仅保存 SHA-256 校验值，
// 用户可以选择是否把安全密钥作为账号密码之外的第二层登录验证。
package auth
