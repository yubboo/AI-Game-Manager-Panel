import fs from 'node:fs'
import path from 'node:path'

const root = process.cwd()
const failures = []
const req = (ok, message) => { if (!ok) failures.push(message) }
const read = rel => fs.readFileSync(path.join(root, rel), 'utf8')

try {
  const webSecurity = read('internal/bridge/httpapi/web_security.go')
  const server = read('internal/bridge/httpapi/server.go')
  const backend = read('frontend/src/shared/api/backend.ts')
  const tests = read('internal/bridge/httpapi/web_security_test.go')
  const authGate = read('frontend/src/features/auth/AuthGate.vue')
  const appView = read('frontend/src/app/App.vue')
  const indexHtml = read('frontend/index.html')
  const appStore = read('frontend/src/shared/store/app.ts')

  req(webSecurity.includes('agmp_session') && webSecurity.includes('HttpOnly: true'), 'Web Session 必须使用 HttpOnly agmp_session Cookie')
  req(webSecurity.includes('agmp_csrf') && webSecurity.includes('X-AGMP-CSRF'), 'Web Cookie Session 必须配置双提交 CSRF Token')
  req(webSecurity.includes('SameSite: http.SameSiteStrictMode'), '认证 Cookie 必须使用 SameSite=Strict')
  req(webSecurity.includes('subtle.ConstantTimeCompare'), 'CSRF 比较必须使用常量时间比较')
  req(webSecurity.includes('requestOriginAllowed') && webSecurity.includes('Sec-Fetch-Site') && webSecurity.includes('Origin'), '所有浏览器 API 写请求必须经过 Origin / Sec-Fetch-Site 同源检查')
  req(webSecurity.includes('ip.IsLoopback()') && webSecurity.includes('X-Forwarded-Proto'), '只有 loopback 反向代理可以提供可信 X-Forwarded-Proto')
  req(webSecurity.includes('Content-Security-Policy') && webSecurity.includes("frame-ancestors 'none'") && webSecurity.includes("object-src 'none'"), 'Web 必须设置 CSP 并禁止 frame/object 注入面')
  req(webSecurity.includes('X-Frame-Options') && webSecurity.includes('DENY'), 'Web 必须禁止 iframe 点击劫持')
  req(webSecurity.includes('X-Content-Type-Options') && webSecurity.includes('nosniff'), 'Web 必须启用 nosniff')
  req(webSecurity.includes('Referrer-Policy') && webSecurity.includes('no-referrer'), 'Web 必须避免敏感 URL Referrer 泄露')
  req(server.includes('writeAuthenticatedSession') && server.includes('clearAuthenticatedSession'), '登录/注册/登出必须统一经过安全 Cookie Session helper')
  req(server.includes('withSecurityHeaders') && server.includes('withOriginProtection') && server.includes('requireSession'), 'HTTP 路由链必须启用 Security Headers、Origin Protection 与 Session Gate')
  req(backend.includes("credentials: 'same-origin'"), 'Web fetch 必须显式携带同源 HttpOnly Cookie')
  req(backend.includes("cookieValue('agmp_csrf')") && backend.includes("'X-AGMP-CSRF': csrf"), '前端写请求必须从 CSRF Cookie 回送 X-AGMP-CSRF')
  req(backend.includes("if (window.go?.wailsbridge?.App) sessionToken = window.localStorage?.getItem(sessionStorageKey)"), '只有 Wails bridge 可以从 localStorage 恢复显式 Session Token')
  req(backend.includes('else window.localStorage?.removeItem(sessionStorageKey)'), 'Web/Electron HTTP 模式必须清理旧 localStorage Session Token')
  req(!backend.includes("else sessionToken = window.localStorage?.getItem(sessionStorageKey)"), 'Web/Electron HTTP 模式禁止从 localStorage 恢复 Session Token')
  req(backend.includes("credentials: 'same-origin'") && backend.includes('if (response.status === 401) {'), 'XiaoYu SSE 必须使用 Cookie Session，并在 Web sessionToken 为空时仍处理 401 会话失效')

  req(authGate.includes('SILENT_SESSION_RESTORE') && authGate.includes('SILENT_AUTH_LOADING'), '刷新必须静默恢复会话，禁止 AuthGate loading 登录卡闪现')
  req(authGate.includes('user.value = await backend.currentUser()') && authGate.includes("backend.adapter() !== 'wails' || backend.getSessionToken()"), 'HTTP/Wails 已有会话必须优先 currentUser 恢复')
  req(appView.includes('SHELL_FIRST_HYDRATION') && appView.includes('ready.value = true') && appView.includes('void hydrateApp()'), '认证成功后必须先恢复工作台 Shell，再后台 Hydrate')
  req(indexHtml.includes('PREPAINT_THEME_RESTORE') && indexHtml.includes('agmp.ui.theme-choice') && indexHtml.includes('agmp-preload'), 'index.html 必须在 Vue 首帧前恢复主题并关闭首帧过渡')
  req(appStore.includes("UI_THEME_CHOICE_KEY = 'agmp.ui.theme-choice'") && appStore.includes('readPersistedThemeChoice'), '主题选择必须本地持久化用于刷新首帧')

  req(tests.includes('TestWebSessionCookieRequiresCSRF') || tests.includes('CSRF'), '必须有 Web Cookie/CSRF 回归测试')
  req(tests.includes('TestCrossSite') || tests.includes('Origin'), '必须有跨站 Origin 防护回归测试')
  req(tests.includes('X-Forwarded-Proto') || tests.includes('SecureTransport'), '必须测试 X-Forwarded-Proto 信任边界')
} catch (error) {
  failures.push(`Web Session Security Gate 检查失败：${error instanceof Error ? error.message : String(error)}`)
}

if (failures.length) {
  console.error('AGMP Web Session Security Gate FAIL')
  for (const failure of failures) console.error(` - ${failure}`)
  process.exit(1)
}
console.log('AGMP Web Session Security Gate PASS (HttpOnly session · CSRF · origin protection · CSP · no Web localStorage bearer)')
