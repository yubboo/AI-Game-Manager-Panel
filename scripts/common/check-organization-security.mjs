import fs from 'node:fs'
import path from 'node:path'

const root = process.cwd()
const failures = []
const read = rel => fs.readFileSync(path.join(root, rel), 'utf8')
const requireText = (name, text, needle) => { if (!text.includes(needle)) failures.push(`${name} 缺少：${needle}`) }

try {
  const auth = read('internal/system/auth/service.go')
  const invite = read('internal/system/auth/identity_invite.go')
  const member = read('internal/system/auth/member_security.go')
  const email = read('internal/system/auth/email_security.go')
  const http = read('internal/bridge/httpapi/server.go')
  const rate = read('internal/bridge/httpapi/auth_rate_limit.go')
  const users = read('frontend/src/features/users/UsersView.vue')
  const gate = read('frontend/src/features/auth/AuthGate.vue')
  const settings = read('frontend/src/features/settings/EmailServiceSection.vue')
  const xiaoyu = read('internal/app/app_xiaoyu.go')

  requireText('Auth', auth, 'ErrInvitationRequired')
  requireText('Auth', auth, 'OrganizationID')
  requireText('Auth', auth, 'AuthEpoch')
  requireText('Invitation', invite, 'ed25519.Sign')
  requireText('Invitation', invite, 'ed25519.Verify')
  requireText('Invitation', invite, 'InstanceFingerprint')
  requireText('Invitation', invite, 'UsedAt')
  requireText('Invitation', invite, 'RevokedAt')
  requireText('Member Authorization', member, 'memberAuthorizationPrefix = "AGMPU1"')
  requireText('Member Authorization', member, 'RequireCoreAccess')
  requireText('Member Authorization', member, 'RequireSensitiveAction')
  requireText('Member Authorization', member, 'ErrCredentialStepUpRequired')
  requireText('XiaoYu', xiaoyu, 'requireXiaoYuMember(token)')
  if (http.includes('HandleFunc("POST /api/v1/users"')) failures.push('HTTP 禁止恢复管理员直接创建成员账号；只能通过签名邀请注册')
  requireText('HTTP', http, 'POST /api/v1/auth/invitation/inspect')
  requireText('HTTP', http, 'POST /api/v1/auth/invitation/register')
  requireText('HTTP', http, 'POST /api/v1/auth/core/redeem')
  requireText('HTTP', http, 'POST /api/v1/auth/step-up/credentials')
  requireText('Rate Limit', rate, 'ip.IsLoopback()')
  requireText('Rate Limit', rate, 'X-Real-IP')
  requireText('UsersView', users, '#/?invite=')
  requireText('AuthGate', gate, "get('invite')")
  if (gate.includes("searchParams.get('invite')")) failures.push('AuthGate 禁止从 Query 读取邀请令牌；只允许 URL Fragment')
  requireText('UsersView', users, '签发成员核心授权码')
  requireText('UsersView', users, 'coreAccess')

  // Email is optional. SMTP only enables email-related recovery/verification.
  requireText('Email Service', email, 'ErrEmailDeliveryUnavailable')
  requireText('Email Service', email, 'EmailFeaturesAvailable')
  requireText('Email Service', email, 'RequestPasswordReset')
  requireText('Email Settings', settings, '邮箱是可选安全增强')
  requireText('Email Settings', settings, '不影响 AGMP')
  if (member.includes('ErrEmailRequired') || member.includes('EmailVerifiedAt == 0')) failures.push('成员核心授权链禁止把邮箱绑定/验证作为必要条件')
  if (gate.includes('&& ownerSecurityKey.value.startsWith') && !gate.includes('!ownerRequireSecurityKey.value ||')) failures.push('Owner 安全密钥不能成为首次注册强制项')
} catch (error) {
  failures.push(error instanceof Error ? error.message : String(error))
}

if (failures.length) {
  console.error(`AGMP Organization & Email Security Gate FAIL (${failures.length})`)
  for (const item of failures) console.error(` - ${item}`)
  process.exit(1)
}
console.log('AGMP Organization & Email Security Gate PASS (invite-only · instance identity · member core grant · optional email · SMTP-admin · step-up · default deny)')
