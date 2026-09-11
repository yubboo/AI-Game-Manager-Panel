import fs from 'node:fs'

const failures = []
const read = p => fs.readFileSync(p, 'utf8')
const wails = read('internal/bridge/wails/app.go')
const backend = read('frontend/src/shared/api/backend.ts')
const types = read('frontend/src/shared/types/backend.ts')
const http = read('internal/bridge/httpapi/server.go')

const publicWails = new Set([
  'Ping', 'GetAuthBootstrapStatus', 'GenerateSecurityKey', 'CreateInitialAdministrator',
  'Login', 'GetAuthInstanceIdentity', 'InspectUserInvitation', 'RegisterInvitedUser',
  'RequestPasswordReset', 'ConfirmPasswordReset',
])
const exported = [...wails.matchAll(/^func \(a \*App\) ([A-Z]\w*)\(([^)]*)\)/gm)]
for (const [, name, args] of exported) {
  if (publicWails.has(name)) continue
  if (!/^token(?:\s+string|,)/.test(args.trim())) {
    failures.push(`Wails.${name} 未携带组织成员 session token；桌面端可能绕过 Web Session Gate`)
  }
}

const protectedClientCalls = [...backend.matchAll(/requireWails\(\)\.(\w+)\(([^)]*)\)/g)]
for (const [, name, args] of protectedClientCalls) {
  if (publicWails.has(name) || name === 'Logout') continue
  if (!args.trim().startsWith('sessionToken')) {
    failures.push(`frontend backend.${name} 调用 Wails 时未传 sessionToken`)
  }
}

const typeBlock = types.includes('GetEnvironmentSetupStatus: (token: string)') &&
  types.includes('GetSettings: (token: string)') &&
  types.includes('GetDSTEnvironment: (token: string)') &&
  types.includes('GetRecentLogs: (token: string, limit: number)')
if (!typeBlock) failures.push('Wails TypeScript bridge 声明尚未统一到 token-first Default-Deny')

for (const name of ['InstallLatestUpdate','InitializeEnvironment','SkipEnvironmentSetup','InstallSteamCMD','InstallSteamCMDAt','UpdateEnvironmentPaths','MigrateSteamCMD','MigrateGameLibrary','SaveSettings','DeleteGlobalLog','DeleteGlobalLogs','ClearGlobalLogHistory','OpenGlobalLogFolder']) {
  const start = wails.indexOf(`func (a *App) ${name}(`)
  if (start < 0) { failures.push(`缺少 Wails.${name}`); continue }
  const next = wails.indexOf('\nfunc (a *App)', start + 1)
  const block = wails.slice(start, next < 0 ? wails.length : next)
  if (!block.includes('requireAdministrator(token)')) failures.push(`Wails.${name} 必须经过 administrator gate`)
}

if (!wails.includes('ActivateLicenseForUser(token, request)')) failures.push('Wails 许可证激活必须使用 Owner 认证包装器')
if (!wails.includes('UnbindLicenseForUser(token)')) failures.push('Wails 许可证解绑必须使用 Owner + step-up 包装器')
if (!http.includes('ActivateLicenseForUser(bearerToken(r), request)')) failures.push('HTTP 许可证激活必须使用 Owner 认证包装器')
if (!http.includes('UnbindLicenseForUser(bearerToken(r))')) failures.push('HTTP 许可证解绑必须使用 Owner + step-up 包装器')
if (!http.includes('return withSecurityHeaders(withNoCacheAPI(withOriginProtection(s.requireSession(mux))))')) failures.push('HTTP API 必须统一经过 Security Headers → no-cache → Origin Protection → requireSession middleware 链')
if (!http.includes('需要有效的组织成员登录会话')) failures.push('HTTP Default-Deny 缺少组织成员会话拒绝提示')

if (failures.length) {
  console.error(`AGMP Bridge Auth Boundary Gate FAIL (${failures.length})`)
  for (const failure of failures) console.error(` - ${failure}`)
  process.exit(1)
}
console.log('AGMP Bridge Auth Boundary Gate PASS (HTTP session middleware · Wails token-first · admin mutations · owner license mutation)')
