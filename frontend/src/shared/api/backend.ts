import type {
  UpdateStatus,
  PreparedUpdate,
  LicenseStatus,
  LicenseActivateRequest,
  LicenseFeatureStatus,
  XiaoYuRuntimeStatus,
  XiaoYuToolSpec,
  XiaoYuCapabilitySnapshot,
  XiaoYuHarnessSnapshot,
  XiaoYuModelCatalog,
  XiaoYuModelProfile,
  XiaoYuSaveModelRequest,
  XiaoYuModelConnectionRequest,
  XiaoYuModelConnectionResult,
  XiaoYuIntelligenceCatalog,
  XiaoYuMemoryRecord,
  XiaoYuMemorySaveRequest,
  XiaoYuSkillDefinition,
  XiaoYuSkillSaveRequest,
  XiaoYuExpertDefinition,
  XiaoYuExpertSaveRequest,
  XiaoYuRunRequest,
  XiaoYuContinueRunRequest,
  XiaoYuControlRequest,
  XiaoYuRunState,
  XiaoYuTraceEvent,
  XiaoYuDSHBundle,
  XiaoYuDSHMountRequest,
  XiaoYuPluginSnapshot,
  XiaoYuToolCallRequest,
  XiaoYuToolCallResult,
  XiaoYuCommandRequest,
  XiaoYuCommandResult,
  XiaoYuApprovalState,
  XiaoYuApprovalRequest,
  AuthBootstrapStatus,
  AuthUser,
  AuthSession,
  AuthCreateOwnerRequest,
  AuthLoginRequest,
  AuthUpdateDisplayNameRequest,
  AuthInstanceIdentity,
  AuthOrganization,
  AuthInvitationView,
  AuthCreatedInvitation,
  AuthCreateInvitationRequest,
  AuthInvitationPreview,
  AuthRegisterInvitationRequest,
  AuthMemberAuthorizationView,
  AuthCreatedMemberAuthorization,
  AuthCreateMemberAuthorizationRequest,
  AuthRedeemMemberAuthorizationRequest,
  AuthCredentialStepUpRequest,
  AuthEmailSecurityStatus,
  AuthBindEmailRequest,
  AuthRequestEmailVerificationRequest,
  AuthConfirmEmailVerificationRequest,
  AuthRequestPasswordResetRequest,
  AuthConfirmPasswordResetRequest,
  AuthPasswordResetRequestStatus,
  AuthSMTPSettings,
  AuthSaveSMTPSettingsRequest,
  AuthSecurityKeyMaterial,
  AuthSecurityStatus,
  AuthRotateSecurityKeyRequest,
  AuthSetSecurityKeyVerificationRequest,
  EnvironmentSetupStatus,
  EnvironmentInitializeRequest,
  EnvironmentStoragePathsRequest,
  SteamCMDInstallRequest,
  EnvironmentMigratePathRequest,
  EnvironmentMigrationResult,
  EnvironmentRuntimeCatalog,
  EnvironmentRuntimeRecord,
  EnvironmentJavaInstallRequest,
  EnvironmentRegisterRuntimeRequest,
  EnvironmentResolveRuntimeRequest,
  EnvironmentSetDefaultRuntimeRequest,
  EnvironmentGameRuntimeProfile,
  EnvironmentGameRuntimeProfileRequest,
  EnvironmentSystemPrerequisite,
  EnvironmentInstallSystemPrerequisiteRequest,
  AppInfo,
  PlatformConfig,
  PlatformRuntimeContract,
  GameInstance,
  GamePack,
  AGMPSettings,
  GlobalLogCatalogRequest,
  GlobalLogCatalogPage,
  GlobalLogReadRequest,
  GlobalLogReadPage,
  GlobalLogExportResult,
  GlobalLogMutationResult,
  GlobalLogDeleteFilteredRequest,
  DSTDedicatedPreferences,
  DSTDedicatedSnapshot,
  DSTEnvironment,
  DSTTokenStatus,
  DSTTokenSaveRequest,
  DSTTokenImportRequest,
  DSTKleiPackageInspectRequest,
  DSTKleiPackageImportRequest,
  DSTKleiPackagePreview,
  DSTKleiPackageImportResult,
  DSTPreflightResult,
  DSTClusterImportRequest,
  DSTClusterImportResult,
  DSTLaunchRequest,
  DSTLaunchSpec,
  DSTStartMasterRequest,
  DSTStartClusterRequest,
  DSTStartShardRequest,
  DSTClusterRequest,
  DSTClusterRuntimeSnapshot,
  DSTPortReport,
  DSTPortConfiguration,
  DSTPortConfigureRequest,
  DSTPortConfigureResult,
  DSTPortCleanupRequest,
  DSTPortCleanupResult,
  DSTProcessRequest,
  DSTProcessLookup,
  DSTProcessLogRequest,
  DSTProcessLogBatch,
  DSTCommandRequest,
  DSTProcessSnapshot,
  DSTWorkspaceSnapshot,
  DSTLogListRequest,
  DSTLogSession,
  DSTLogReadRequest,
  DSTLogReadPage,
  DSTLogTailRequest,
  DSTLogSearchRequest,
  DSTLogSearchResult,
  DSTLogDiagnostics,
  DSTLogExportResult,
  DSTLogBundleRequest,
  GameWorkspaceSnapshot,
  SteamAppInventory,
  SteamEnvironment,
  SteamSnapshot,
  SteamMaintenanceTask,
  SteamMaintenanceValidateRequest,
  SteamMaintenanceInstallRequest,
} from '../types/backend'

export type BackendMode = 'desktop' | 'web'
export type BackendAdapter = 'wails' | 'electron' | 'http'

const sessionStorageKey = 'agmp.auth.session'
let sessionToken = ''

// Wails calls are direct Go bridge invocations and therefore carry an explicit
// token. Web/Electron HTTP sessions use an HttpOnly cookie; never persist the
// bearer token in browser localStorage where injected JavaScript could read it.
try {
  if (window.go?.wailsbridge?.App) sessionToken = window.localStorage?.getItem(sessionStorageKey) ?? ''
  else window.localStorage?.removeItem(sessionStorageKey)
} catch { sessionToken = '' }

function setSessionToken(value: string) {
  if (window.go?.wailsbridge?.App) {
    sessionToken = value.trim()
    try {
      if (sessionToken) window.localStorage?.setItem(sessionStorageKey, sessionToken)
      else window.localStorage?.removeItem(sessionStorageKey)
    } catch { /* localStorage may be unavailable in hardened WebViews. */ }
    return
  }
  sessionToken = ''
  try { window.localStorage?.removeItem(sessionStorageKey) } catch { /* Best-effort legacy token cleanup. */ }
}

function cookieValue(name: string) {
  const prefix = `${encodeURIComponent(name)}=`
  for (const item of document.cookie.split(';')) {
    const value = item.trim()
    if (value.startsWith(prefix)) return decodeURIComponent(value.slice(prefix.length))
  }
  return ''
}

function getSessionToken() { return sessionToken }

function wailsApi() {
  return window.go?.wailsbridge?.App
}

function hasWailsBridge() { return Boolean(wailsApi()) }
function hasElectronShell() { return window.agmpElectron?.framework === 'electron' }

function mode(): BackendMode {
  return hasWailsBridge() || hasElectronShell() ? 'desktop' : 'web'
}

function adapter(): BackendAdapter {
  if (hasWailsBridge()) return 'wails'
  if (hasElectronShell()) return 'electron'
  return 'http'
}

async function httpJSON<T>(path: string, init?: RequestInit): Promise<T> {
  const method = (init?.method || 'GET').toUpperCase()
  const csrf = ['POST', 'PUT', 'PATCH', 'DELETE'].includes(method) ? cookieValue('agmp_csrf') : ''
  const response = await fetch(path, {
    ...init,
    credentials: 'same-origin',
    headers: {
      Accept: 'application/json',
      ...(init?.body ? { 'Content-Type': 'application/json' } : {}),
      ...(sessionToken ? { Authorization: `Bearer ${sessionToken}` } : {}),
      ...(csrf ? { 'X-AGMP-CSRF': csrf } : {}),
      ...(init?.headers ?? {}),
    },
  })
  if (!response.ok) {
    if (response.status === 401 && !path.endsWith('/auth/login')) {
      setSessionToken('')
      window.dispatchEvent(new Event('agmp-auth-expired'))
    }
    let message = `HTTP ${response.status}`
    try {
      const body = await response.json() as { error?: string }
      if (body.error) message = body.error
    } catch {
      // Keep the HTTP status when the response is not JSON.
    }
    throw new Error(message)
  }
  if (response.status === 204) return undefined as T
  return response.json() as Promise<T>
}

function requireWails() {
  const api = wailsApi()
  if (!api) throw new Error('未检测到AI游戏管理器面板 Wails 桌面后端。')
  return api
}


function streamXiaoYuEvents(after: number, onEvent: (event: XiaoYuTraceEvent) => void, onError?: (error: Error) => void): () => void {
  // Wails has no HTTP event stream in-process; it consumes the exact same
  // server-owned Trace through bounded polling. Closing the window/poller never
  // cancels a XiaoYu Run.
  if (hasWailsBridge()) {
    let stopped = false
    let cursor = after
    let inFlight = false
    const poll = async () => {
      if (stopped || inFlight) return
      inFlight = true
      try {
        const events = await requireWails().GetXiaoYuTrace(sessionToken, cursor, 200)
        for (const event of events) {
          cursor = Math.max(cursor, event.sequence)
          onEvent(event)
        }
      } catch (error) {
        if (!stopped && onError) onError(error instanceof Error ? error : new Error(String(error)))
      } finally {
        inFlight = false
      }
    }
    void poll()
    const timer = window.setInterval(() => void poll(), 1000)
    return () => { stopped = true; window.clearInterval(timer) }
  }

  const controller = new AbortController()
  let cursor = after
  const sleep = (ms: number) => new Promise<void>(resolve => window.setTimeout(resolve, ms))
  void (async () => {
    while (!controller.signal.aborted) {
      try {
        const response = await fetch(`/api/v1/xiaoyu/events?after=${encodeURIComponent(cursor)}`, {
          method: 'GET',
          credentials: 'same-origin',
          headers: {
            Accept: 'text/event-stream',
            ...(sessionToken ? { Authorization: `Bearer ${sessionToken}` } : {}),
          },
          signal: controller.signal,
        })
        // Web HTTP mode intentionally keeps sessionToken empty because the real
        // session lives in an HttpOnly cookie. A 401 must therefore expire the
        // UI regardless of whether an explicit Wails bearer exists.
        if (response.status === 401) {
          setSessionToken('')
          window.dispatchEvent(new Event('agmp-auth-expired'))
          return
        }
        if (!response.ok) throw new Error(`XiaoYu Event Stream HTTP ${response.status}`)
        if (!response.body) throw new Error('当前浏览器不支持 XiaoYu 实时事件流。')
        const reader = response.body.getReader()
        const decoder = new TextDecoder()
        let buffer = ''
        while (!controller.signal.aborted) {
          const { value, done } = await reader.read()
          if (done) break
          buffer += decoder.decode(value, { stream: true })
          let boundary = buffer.indexOf('\n\n')
          while (boundary >= 0) {
            const block = buffer.slice(0, boundary)
            buffer = buffer.slice(boundary + 2)
            const data = block.split('\n').filter(line => line.startsWith('data:')).map(line => line.slice(5).trim()).join('\n')
            if (data) {
              try {
                const event = JSON.parse(data) as XiaoYuTraceEvent
                cursor = Math.max(cursor, event.sequence)
                onEvent(event)
              } catch { /* malformed extension event: ignore without killing stream */ }
            }
            boundary = buffer.indexOf('\n\n')
          }
        }
      } catch (error) {
        if (controller.signal.aborted) return
        if (onError) onError(error instanceof Error ? error : new Error(String(error)))
      }
      if (!controller.signal.aborted) await sleep(1200)
    }
  })()
  return () => controller.abort()
}

export const backend = {
  mode,
  adapter,
  getSessionToken,
  setSessionToken,
  checkForUpdates(force = false): Promise<UpdateStatus> {
    if (hasWailsBridge()) return requireWails().CheckForUpdates(sessionToken, force)
    return httpJSON<UpdateStatus>(`/api/v1/updater/check?force=${force ? '1' : '0'}`)
  },
  installLatestUpdate(): Promise<PreparedUpdate> {
    if (hasWailsBridge()) return requireWails().InstallLatestUpdate(sessionToken)
    return Promise.reject(new Error('自动安装更新仅支持 Windows Wails 桌面端；Web 管理端请下载正式安装器升级。'))
  },
  licenseStatus(): Promise<LicenseStatus> {
    if (hasWailsBridge()) return requireWails().GetLicenseStatus(sessionToken)
    return httpJSON<LicenseStatus>('/api/v1/license/status')
  },
  activateLicense(request: LicenseActivateRequest): Promise<LicenseStatus> {
    if (hasWailsBridge()) return requireWails().ActivateLicense(sessionToken, request)
    return httpJSON<LicenseStatus>('/api/v1/license/activate', { method: 'POST', body: JSON.stringify(request) })
  },
  unbindLicense(): Promise<LicenseStatus> {
    if (hasWailsBridge()) return requireWails().UnbindLicense(sessionToken)
    return httpJSON<LicenseStatus>('/api/v1/license/unbind', { method: 'POST' })
  },
  licenseFeatureStatus(feature: string): Promise<LicenseFeatureStatus> {
    if (hasWailsBridge()) return requireWails().GetLicenseFeatureStatus(sessionToken, feature)
    return httpJSON<LicenseFeatureStatus>(`/api/v1/license/features/${encodeURIComponent(feature)}`)
  },
  xiaoyuRuntimeStatus(): Promise<XiaoYuRuntimeStatus> {
    if (hasWailsBridge()) return requireWails().GetXiaoYuRuntimeStatus(sessionToken)
    return httpJSON<XiaoYuRuntimeStatus>('/api/v1/xiaoyu/runtime')
  },
  xiaoyuTools(): Promise<XiaoYuToolSpec[]> {
    if (hasWailsBridge()) return requireWails().GetXiaoYuTools(sessionToken)
    return httpJSON<XiaoYuToolSpec[]>('/api/v1/xiaoyu/tools')
  },
  xiaoyuCapabilities(): Promise<XiaoYuCapabilitySnapshot> {
    if (hasWailsBridge()) return requireWails().GetXiaoYuCapabilities(sessionToken)
    return httpJSON<XiaoYuCapabilitySnapshot>('/api/v1/xiaoyu/capabilities')
  },
  xiaoyuHarnessStatus(): Promise<XiaoYuHarnessSnapshot> {
    if (hasWailsBridge()) return requireWails().GetXiaoYuHarnessStatus(sessionToken)
    return httpJSON<XiaoYuHarnessSnapshot>('/api/v1/xiaoyu/harness')
  },
  xiaoyuModelCatalog(): Promise<XiaoYuModelCatalog> {
    if (hasWailsBridge()) return requireWails().GetXiaoYuModelCatalog(sessionToken)
    return httpJSON<XiaoYuModelCatalog>('/api/v1/xiaoyu/models')
  },
  saveXiaoYuModel(request: XiaoYuSaveModelRequest): Promise<XiaoYuModelProfile> {
    if (hasWailsBridge()) return requireWails().SaveXiaoYuModel(sessionToken, request)
    return httpJSON<XiaoYuModelProfile>('/api/v1/xiaoyu/models', { method: 'POST', body: JSON.stringify(request) })
  },
  deleteXiaoYuModel(id: string): Promise<void> {
    if (hasWailsBridge()) return requireWails().DeleteXiaoYuModel(sessionToken, id)
    return httpJSON<void>(`/api/v1/xiaoyu/models/${encodeURIComponent(id)}`, { method: 'DELETE' })
  },
  setXiaoYuDefaultModel(id: string): Promise<XiaoYuModelProfile> {
    if (hasWailsBridge()) return requireWails().SetXiaoYuDefaultModel(sessionToken, id)
    return httpJSON<XiaoYuModelProfile>(`/api/v1/xiaoyu/models/${encodeURIComponent(id)}/default`, { method: 'PUT' })
  },
  testXiaoYuModel(request: XiaoYuModelConnectionRequest): Promise<XiaoYuModelConnectionResult> {
    if (hasWailsBridge()) return requireWails().TestXiaoYuModel(sessionToken, request)
    return httpJSON<XiaoYuModelConnectionResult>('/api/v1/xiaoyu/models/test', { method: 'POST', body: JSON.stringify(request) })
  },
  discoverXiaoYuModels(request: XiaoYuModelConnectionRequest): Promise<XiaoYuModelConnectionResult> {
    if (hasWailsBridge()) return requireWails().DiscoverXiaoYuModels(sessionToken, request)
    return httpJSON<XiaoYuModelConnectionResult>('/api/v1/xiaoyu/models/discover', { method: 'POST', body: JSON.stringify(request) })
  },
  xiaoyuIntelligenceCatalog(): Promise<XiaoYuIntelligenceCatalog> {
    if (hasWailsBridge()) return requireWails().GetXiaoYuIntelligenceCatalog(sessionToken)
    return httpJSON<XiaoYuIntelligenceCatalog>('/api/v1/xiaoyu/intelligence')
  },
  saveXiaoYuMemory(request: XiaoYuMemorySaveRequest): Promise<XiaoYuMemoryRecord> {
    if (hasWailsBridge()) return requireWails().SaveXiaoYuMemory(sessionToken, request)
    return httpJSON<XiaoYuMemoryRecord>('/api/v1/xiaoyu/intelligence/memories', { method: 'POST', body: JSON.stringify(request) })
  },
  saveXiaoYuSkill(request: XiaoYuSkillSaveRequest): Promise<XiaoYuSkillDefinition> {
    if (hasWailsBridge()) return requireWails().SaveXiaoYuSkill(sessionToken, request)
    return httpJSON<XiaoYuSkillDefinition>('/api/v1/xiaoyu/intelligence/skills', { method: 'POST', body: JSON.stringify(request) })
  },
  saveXiaoYuExpert(request: XiaoYuExpertSaveRequest): Promise<XiaoYuExpertDefinition> {
    if (hasWailsBridge()) return requireWails().SaveXiaoYuExpert(sessionToken, request)
    return httpJSON<XiaoYuExpertDefinition>('/api/v1/xiaoyu/intelligence/experts', { method: 'POST', body: JSON.stringify(request) })
  },
  deleteXiaoYuIntelligence(kind: 'memory' | 'skill' | 'expert', id: string): Promise<void> {
    if (hasWailsBridge()) return requireWails().DeleteXiaoYuIntelligence(sessionToken, kind, id)
    return httpJSON<void>(`/api/v1/xiaoyu/intelligence/${encodeURIComponent(kind)}/${encodeURIComponent(id)}`, { method: 'DELETE' })
  },
  xiaoyuStartRun(request: XiaoYuRunRequest): Promise<XiaoYuRunState> {
    if (hasWailsBridge()) return requireWails().StartXiaoYuRun(sessionToken, request)
    return httpJSON<XiaoYuRunState>('/api/v1/xiaoyu/runs', { method: 'POST', body: JSON.stringify(request) })
  },
  xiaoyuContinueRun(id: string, request: XiaoYuContinueRunRequest = {}): Promise<XiaoYuRunState> {
    if (hasWailsBridge()) return requireWails().ContinueXiaoYuRun(sessionToken, id, request)
    return httpJSON<XiaoYuRunState>(`/api/v1/xiaoyu/runs/${encodeURIComponent(id)}/continue`, { method: 'POST', body: JSON.stringify(request) })
  },
  xiaoyuRunState(id: string): Promise<XiaoYuRunState> {
    if (hasWailsBridge()) return requireWails().GetXiaoYuRunState(sessionToken, id)
    return httpJSON<XiaoYuRunState>(`/api/v1/xiaoyu/runs/${encodeURIComponent(id)}`)
  },
  xiaoyuCancelRun(id: string): Promise<XiaoYuRunState> {
    if (hasWailsBridge()) return requireWails().CancelXiaoYuRun(sessionToken, id)
    return httpJSON<XiaoYuRunState>(`/api/v1/xiaoyu/runs/${encodeURIComponent(id)}/cancel`, { method: 'POST' })
  },
  xiaoyuPauseRun(id: string, request: XiaoYuControlRequest = {}): Promise<XiaoYuRunState> {
    if (hasWailsBridge()) return requireWails().PauseXiaoYuRun(sessionToken, id, request)
    return httpJSON<XiaoYuRunState>(`/api/v1/xiaoyu/runs/${encodeURIComponent(id)}/pause`, { method: 'POST', body: JSON.stringify(request) })
  },
  xiaoyuTakeoverRun(id: string, request: XiaoYuControlRequest = {}): Promise<XiaoYuRunState> {
    if (hasWailsBridge()) return requireWails().TakeoverXiaoYuRun(sessionToken, id, request)
    return httpJSON<XiaoYuRunState>(`/api/v1/xiaoyu/runs/${encodeURIComponent(id)}/takeover`, { method: 'POST', body: JSON.stringify(request) })
  },
  xiaoyuResumeRun(id: string): Promise<XiaoYuRunState> {
    if (hasWailsBridge()) return requireWails().ResumeXiaoYuRun(sessionToken, id)
    return httpJSON<XiaoYuRunState>(`/api/v1/xiaoyu/runs/${encodeURIComponent(id)}/resume`, { method: 'POST' })
  },
  xiaoyuSubscribeEvents(after: number, onEvent: (event: XiaoYuTraceEvent) => void, onError?: (error: Error) => void): () => void {
    return streamXiaoYuEvents(after, onEvent, onError)
  },
  xiaoyuTrace(after = 0, limit = 100): Promise<XiaoYuTraceEvent[]> {
    if (hasWailsBridge()) return requireWails().GetXiaoYuTrace(sessionToken, after, limit)
    return httpJSON<XiaoYuTraceEvent[]>(`/api/v1/xiaoyu/trace?after=${encodeURIComponent(after)}&limit=${encodeURIComponent(limit)}`)
  },
  xiaoyuDSHPlugins(): Promise<XiaoYuDSHBundle[]> {
    if (hasWailsBridge()) return requireWails().GetXiaoYuDSHPlugins(sessionToken)
    return httpJSON<XiaoYuDSHBundle[]>('/api/v1/xiaoyu/plugins/dsh')
  },
  mountXiaoYuDSHPlugin(request: XiaoYuDSHMountRequest): Promise<XiaoYuPluginSnapshot> {
    if (hasWailsBridge()) return requireWails().MountXiaoYuDSHPlugin(sessionToken, request)
    return httpJSON<XiaoYuPluginSnapshot>('/api/v1/xiaoyu/plugins/dsh/mount', { method: 'POST', body: JSON.stringify(request) })
  },
  unmountXiaoYuPlugin(id: string): Promise<void> {
    if (hasWailsBridge()) return requireWails().UnmountXiaoYuPlugin(sessionToken, id)
    return httpJSON<void>(`/api/v1/xiaoyu/plugins/${encodeURIComponent(id)}`, { method: 'DELETE' })
  },
  xiaoyuApprovalState(): Promise<XiaoYuApprovalState> {
    if (hasWailsBridge()) return requireWails().GetXiaoYuApprovalState(sessionToken)
    return httpJSON<XiaoYuApprovalState>('/api/v1/xiaoyu/approval')
  },
  setXiaoYuApprovalMode(mode: string): Promise<XiaoYuApprovalState> {
    const request = { mode }
    if (hasWailsBridge()) return requireWails().SetXiaoYuApprovalMode(sessionToken, request)
    return httpJSON<XiaoYuApprovalState>('/api/v1/xiaoyu/approval/mode', { method: 'PUT', body: JSON.stringify(request) })
  },
  xiaoyuPendingApprovals(): Promise<XiaoYuApprovalRequest[]> {
    if (hasWailsBridge()) return requireWails().GetXiaoYuPendingApprovals(sessionToken)
    return httpJSON<XiaoYuApprovalRequest[]>('/api/v1/xiaoyu/approvals')
  },
  resolveXiaoYuApproval(id: string, decision: 'approve' | 'reject'): Promise<XiaoYuApprovalRequest> {
    const request = { decision }
    if (hasWailsBridge()) return requireWails().ResolveXiaoYuApproval(sessionToken, id, request)
    return httpJSON<XiaoYuApprovalRequest>(`/api/v1/xiaoyu/approvals/${encodeURIComponent(id)}`, { method: 'POST', body: JSON.stringify(request) })
  },
  callXiaoYuTool(request: XiaoYuToolCallRequest): Promise<XiaoYuToolCallResult> {
    if (hasWailsBridge()) return requireWails().CallXiaoYuTool(sessionToken, request)
    return httpJSON<XiaoYuToolCallResult>('/api/v1/xiaoyu/tool', { method: 'POST', body: JSON.stringify(request) })
  },
  runXiaoYuCommand(request: XiaoYuCommandRequest): Promise<XiaoYuCommandResult> {
    if (hasWailsBridge()) return requireWails().RunXiaoYuCommand(sessionToken, request)
    return httpJSON<XiaoYuCommandResult>('/api/v1/xiaoyu/command', { method: 'POST', body: JSON.stringify(request) })
  },
  authBootstrapStatus(): Promise<AuthBootstrapStatus> {
    if (hasWailsBridge()) return requireWails().GetAuthBootstrapStatus()
    return httpJSON<AuthBootstrapStatus>('/api/v1/auth/bootstrap')
  },
  generateSecurityKey(): Promise<AuthSecurityKeyMaterial> {
    if (hasWailsBridge()) return requireWails().GenerateSecurityKey()
    return httpJSON<AuthSecurityKeyMaterial>('/api/v1/auth/security-key/generate', { method: 'POST' })
  },
  createInitialAdministrator(request: AuthCreateOwnerRequest): Promise<AuthSession> {
    if (hasWailsBridge()) return requireWails().CreateInitialAdministrator(request).then(session => { setSessionToken(session.token); return session })
    return httpJSON<AuthSession>('/api/v1/auth/bootstrap/owner', { method: 'POST', body: JSON.stringify(request) }).then(session => { setSessionToken(session.token); return session })
  },
  login(request: AuthLoginRequest): Promise<AuthSession> {
    if (hasWailsBridge()) return requireWails().Login(request).then(session => { setSessionToken(session.token); return session })
    return httpJSON<AuthSession>('/api/v1/auth/login', { method: 'POST', body: JSON.stringify(request) }).then(session => { setSessionToken(session.token); return session })
  },
  currentUser(): Promise<AuthUser> {
    if (hasWailsBridge()) return requireWails().ValidateSession(sessionToken)
    return httpJSON<AuthUser>('/api/v1/auth/me')
  },
  async logout(): Promise<void> {
    const token = sessionToken
    try {
      if (hasWailsBridge()) await requireWails().Logout(token)
      else await httpJSON<void>('/api/v1/auth/logout', { method: 'POST' })
    } finally {
      setSessionToken('')
    }
  },
  listUsers(): Promise<AuthUser[]> {
    if (hasWailsBridge()) return requireWails().ListUsers(sessionToken)
    return httpJSON<AuthUser[]>('/api/v1/users')
  },
  updateMyDisplayName(request: AuthUpdateDisplayNameRequest): Promise<AuthUser> {
    if (hasWailsBridge()) return requireWails().UpdateMyDisplayName(sessionToken, request)
    return httpJSON<AuthUser>('/api/v1/auth/me/display-name', { method: 'PUT', body: JSON.stringify(request) })
  },
  authInstanceIdentity(): Promise<AuthInstanceIdentity> {
    if (hasWailsBridge()) return requireWails().GetAuthInstanceIdentity()
    return this.authBootstrapStatus().then(value => ({ id: value.instanceId, fingerprint: value.instanceFingerprint, publicKey: '', createdAt: 0 }))
  },
  authOrganization(): Promise<AuthOrganization> {
    if (hasWailsBridge()) return requireWails().GetAuthOrganization(sessionToken)
    return httpJSON<AuthOrganization>('/api/v1/auth/organization')
  },
  inspectInvitation(token: string): Promise<AuthInvitationPreview> {
    if (hasWailsBridge()) return requireWails().InspectUserInvitation(token)
    return httpJSON<AuthInvitationPreview>('/api/v1/auth/invitation/inspect', { method: 'POST', body: JSON.stringify({ token }) })
  },
  registerInvitedUser(request: AuthRegisterInvitationRequest): Promise<AuthSession> {
    if (hasWailsBridge()) return requireWails().RegisterInvitedUser(request).then(session => { setSessionToken(session.token); return session })
    return httpJSON<AuthSession>('/api/v1/auth/invitation/register', { method: 'POST', body: JSON.stringify(request) }).then(session => { setSessionToken(session.token); return session })
  },
  listUserInvitations(): Promise<AuthInvitationView[]> {
    if (hasWailsBridge()) return requireWails().ListUserInvitations(sessionToken)
    return httpJSON<AuthInvitationView[]>('/api/v1/users/invitations')
  },
  createUserInvitation(request: AuthCreateInvitationRequest): Promise<AuthCreatedInvitation> {
    if (hasWailsBridge()) return requireWails().CreateUserInvitation(sessionToken, request)
    return httpJSON<AuthCreatedInvitation>('/api/v1/users/invitations', { method: 'POST', body: JSON.stringify(request) })
  },
  revokeUserInvitation(id: string): Promise<AuthInvitationView> {
    if (hasWailsBridge()) return requireWails().RevokeUserInvitation(sessionToken, id)
    return httpJSON<AuthInvitationView>(`/api/v1/users/invitations/${encodeURIComponent(id)}`, { method: 'DELETE' })
  },
  myEmailSecurityStatus(): Promise<AuthEmailSecurityStatus> {
    if (hasWailsBridge()) return requireWails().GetMyEmailSecurityStatus(sessionToken)
    return httpJSON<AuthEmailSecurityStatus>('/api/v1/auth/email/status')
  },
  bindMyEmail(request: AuthBindEmailRequest): Promise<AuthEmailSecurityStatus> {
    if (hasWailsBridge()) return requireWails().BindMyEmail(sessionToken, request)
    return httpJSON<AuthEmailSecurityStatus>('/api/v1/auth/email', { method: 'PUT', body: JSON.stringify(request) })
  },
  unbindMyEmail(password: string): Promise<AuthEmailSecurityStatus> {
    if (hasWailsBridge()) return requireWails().UnbindMyEmail(sessionToken, password)
    return httpJSON<AuthEmailSecurityStatus>('/api/v1/auth/email', { method: 'DELETE', body: JSON.stringify({ password }) })
  },
  requestMyEmailVerification(request: AuthRequestEmailVerificationRequest): Promise<void> {
    if (hasWailsBridge()) return requireWails().RequestMyEmailVerification(sessionToken, request)
    return httpJSON<void>('/api/v1/auth/email/verification/request', { method: 'POST', body: JSON.stringify(request) })
  },
  confirmMyEmailVerification(request: AuthConfirmEmailVerificationRequest): Promise<AuthEmailSecurityStatus> {
    if (hasWailsBridge()) return requireWails().ConfirmMyEmailVerification(sessionToken, request)
    return httpJSON<AuthEmailSecurityStatus>('/api/v1/auth/email/verification/confirm', { method: 'POST', body: JSON.stringify(request) })
  },
  confirmMyCredentialStepUp(request: AuthCredentialStepUpRequest): Promise<AuthUser> {
    if (hasWailsBridge()) return requireWails().ConfirmMyCredentialStepUp(sessionToken, request)
    return httpJSON<AuthUser>('/api/v1/auth/step-up/credentials', { method: 'POST', body: JSON.stringify(request) })
  },
  requestPasswordReset(request: AuthRequestPasswordResetRequest): Promise<AuthPasswordResetRequestStatus> {
    if (hasWailsBridge()) return requireWails().RequestPasswordReset(request)
    return httpJSON<AuthPasswordResetRequestStatus>('/api/v1/auth/password-reset/request', { method: 'POST', body: JSON.stringify(request) })
  },
  confirmPasswordReset(request: AuthConfirmPasswordResetRequest): Promise<void> {
    if (hasWailsBridge()) return requireWails().ConfirmPasswordReset(request)
    return httpJSON<void>('/api/v1/auth/password-reset/confirm', { method: 'POST', body: JSON.stringify(request) })
  },
  smtpSettings(): Promise<AuthSMTPSettings> {
    if (hasWailsBridge()) return requireWails().GetSMTPSettings(sessionToken)
    return httpJSON<AuthSMTPSettings>('/api/v1/settings/email')
  },
  saveSMTPSettings(request: AuthSaveSMTPSettingsRequest): Promise<AuthSMTPSettings> {
    if (hasWailsBridge()) return requireWails().SaveSMTPSettings(sessionToken, request)
    return httpJSON<AuthSMTPSettings>('/api/v1/settings/email', { method: 'PUT', body: JSON.stringify(request) })
  },
  listMemberCoreAuthorizations(): Promise<AuthMemberAuthorizationView[]> {
    if (hasWailsBridge()) return requireWails().ListMemberCoreAuthorizations(sessionToken)
    return httpJSON<AuthMemberAuthorizationView[]>('/api/v1/users/core-authorizations')
  },
  createMemberCoreAuthorization(request: AuthCreateMemberAuthorizationRequest): Promise<AuthCreatedMemberAuthorization> {
    if (hasWailsBridge()) return requireWails().CreateMemberCoreAuthorization(sessionToken, request)
    return httpJSON<AuthCreatedMemberAuthorization>('/api/v1/users/core-authorizations', { method: 'POST', body: JSON.stringify(request) })
  },
  revokeMemberCoreAuthorization(id: string): Promise<AuthMemberAuthorizationView> {
    if (hasWailsBridge()) return requireWails().RevokeMemberCoreAuthorization(sessionToken, id)
    return httpJSON<AuthMemberAuthorizationView>(`/api/v1/users/core-authorizations/${encodeURIComponent(id)}`, { method: 'DELETE' })
  },
  redeemMyCoreAuthorization(request: AuthRedeemMemberAuthorizationRequest): Promise<AuthUser> {
    if (hasWailsBridge()) return requireWails().RedeemMyCoreAuthorization(sessionToken, request)
    return httpJSON<AuthUser>('/api/v1/auth/core/redeem', { method: 'POST', body: JSON.stringify(request) })
  },
  revokeMemberCoreAccess(userId: string): Promise<AuthUser> {
    if (hasWailsBridge()) return requireWails().RevokeMemberCoreAccess(sessionToken, userId)
    return httpJSON<AuthUser>(`/api/v1/users/${encodeURIComponent(userId)}/core/revoke`, { method: 'POST' })
  },
  clearMemberRisk(userId: string): Promise<AuthUser> {
    if (hasWailsBridge()) return requireWails().ClearMemberRisk(sessionToken, userId)
    return httpJSON<AuthUser>(`/api/v1/users/${encodeURIComponent(userId)}/risk/clear`, { method: 'POST' })
  },
  removeOrganizationMember(userId: string): Promise<void> {
    if (hasWailsBridge()) return requireWails().RemoveOrganizationMember(sessionToken, userId)
    return httpJSON<void>(`/api/v1/users/${encodeURIComponent(userId)}`, { method: 'DELETE' })
  },
  mySecurityStatus(): Promise<AuthSecurityStatus> {
    if (hasWailsBridge()) return requireWails().GetMySecurityStatus(sessionToken)
    return httpJSON<AuthSecurityStatus>('/api/v1/auth/security-key/status')
  },
  rotateMySecurityKey(request: AuthRotateSecurityKeyRequest): Promise<AuthSecurityKeyMaterial> {
    if (hasWailsBridge()) return requireWails().RotateMySecurityKey(sessionToken, request)
    return httpJSON<AuthSecurityKeyMaterial>('/api/v1/auth/security-key/rotate', { method: 'POST', body: JSON.stringify(request) })
  },
  setMySecurityKeyVerification(request: AuthSetSecurityKeyVerificationRequest): Promise<AuthSecurityStatus> {
    if (hasWailsBridge()) return requireWails().SetMySecurityKeyVerification(sessionToken, request)
    return httpJSON<AuthSecurityStatus>('/api/v1/auth/security-key/verification', { method: 'PUT', body: JSON.stringify(request) })
  },
  environmentSetupStatus(): Promise<EnvironmentSetupStatus> {
    if (hasWailsBridge()) return requireWails().GetEnvironmentSetupStatus(sessionToken)
    return httpJSON<EnvironmentSetupStatus>('/api/v1/environment/setup')
  },
  initializeEnvironment(request: EnvironmentInitializeRequest): Promise<EnvironmentSetupStatus> {
    if (hasWailsBridge()) return requireWails().InitializeEnvironment(sessionToken, request)
    return httpJSON<EnvironmentSetupStatus>('/api/v1/environment/setup', { method: 'POST', body: JSON.stringify(request) })
  },
  skipEnvironmentSetup(): Promise<EnvironmentSetupStatus> {
    if (hasWailsBridge()) return requireWails().SkipEnvironmentSetup(sessionToken)
    return httpJSON<EnvironmentSetupStatus>('/api/v1/environment/setup/skip', { method: 'POST' })
  },
  installSteamCMD(request: SteamCMDInstallRequest = { targetRoot: '' }): Promise<EnvironmentSetupStatus> {
    if (hasWailsBridge()) return requireWails().InstallSteamCMDAt(sessionToken, request)
    return httpJSON<EnvironmentSetupStatus>('/api/v1/environment/steamcmd/install', { method: 'POST', body: JSON.stringify(request) })
  },
  updateEnvironmentPaths(request: EnvironmentStoragePathsRequest): Promise<EnvironmentSetupStatus> {
    if (hasWailsBridge()) return requireWails().UpdateEnvironmentPaths(sessionToken, request)
    return httpJSON<EnvironmentSetupStatus>('/api/v1/environment/paths', { method: 'PUT', body: JSON.stringify(request) })
  },
  migrateSteamCMD(request: EnvironmentMigratePathRequest): Promise<EnvironmentMigrationResult> {
    if (hasWailsBridge()) return requireWails().MigrateSteamCMD(sessionToken, request)
    return httpJSON<EnvironmentMigrationResult>('/api/v1/environment/steamcmd/migrate', { method: 'POST', body: JSON.stringify(request) })
  },
  migrateGameLibrary(request: EnvironmentMigratePathRequest): Promise<EnvironmentMigrationResult> {
    if (hasWailsBridge()) return requireWails().MigrateGameLibrary(sessionToken, request)
    return httpJSON<EnvironmentMigrationResult>('/api/v1/environment/games/migrate', { method: 'POST', body: JSON.stringify(request) })
  },
  environmentRuntimeCatalog(): Promise<EnvironmentRuntimeCatalog> {
    if (hasWailsBridge()) return requireWails().GetEnvironmentRuntimeCatalog(sessionToken)
    return httpJSON<EnvironmentRuntimeCatalog>('/api/v1/environment/runtime/catalog')
  },
  installJavaRuntime(request: EnvironmentJavaInstallRequest): Promise<EnvironmentRuntimeRecord> {
    if (hasWailsBridge()) return requireWails().InstallJavaRuntime(sessionToken, request)
    return httpJSON<EnvironmentRuntimeRecord>('/api/v1/environment/runtime/java/install', { method: 'POST', body: JSON.stringify(request) })
  },
  registerEnvironmentRuntime(request: EnvironmentRegisterRuntimeRequest): Promise<EnvironmentRuntimeRecord> {
    if (hasWailsBridge()) return requireWails().RegisterEnvironmentRuntime(sessionToken, request)
    return httpJSON<EnvironmentRuntimeRecord>('/api/v1/environment/runtime/register', { method: 'POST', body: JSON.stringify(request) })
  },
  resolveEnvironmentRuntime(request: EnvironmentResolveRuntimeRequest): Promise<EnvironmentRuntimeRecord> {
    if (hasWailsBridge()) return requireWails().ResolveEnvironmentRuntime(sessionToken, request)
    return httpJSON<EnvironmentRuntimeRecord>('/api/v1/environment/runtime/resolve', { method: 'POST', body: JSON.stringify(request) })
  },
  setDefaultEnvironmentRuntime(request: EnvironmentSetDefaultRuntimeRequest): Promise<EnvironmentRuntimeRecord> {
    if (hasWailsBridge()) return requireWails().SetDefaultEnvironmentRuntime(sessionToken, request)
    return httpJSON<EnvironmentRuntimeRecord>('/api/v1/environment/runtime/default', { method: 'PUT', body: JSON.stringify(request) })
  },
  removeEnvironmentRuntime(id: string): Promise<void> {
    if (hasWailsBridge()) return requireWails().RemoveEnvironmentRuntime(sessionToken, { id })
    return httpJSON<void>(`/api/v1/environment/runtime/${encodeURIComponent(id)}`, { method: 'DELETE' })
  },
  environmentGameRuntimeProfiles(): Promise<EnvironmentGameRuntimeProfile[]> {
    if (hasWailsBridge()) return requireWails().GetEnvironmentGameRuntimeProfiles(sessionToken)
    return httpJSON<EnvironmentGameRuntimeProfile[]>('/api/v1/environment/games/profiles')
  },
  environmentGameRuntimeProfile(request: EnvironmentGameRuntimeProfileRequest): Promise<EnvironmentGameRuntimeProfile> {
    if (hasWailsBridge()) return requireWails().GetEnvironmentGameRuntimeProfile(sessionToken, request)
    return httpJSON<EnvironmentGameRuntimeProfile>('/api/v1/environment/games/profile', { method: 'POST', body: JSON.stringify(request) })
  },
  installEnvironmentSystemPrerequisite(request: EnvironmentInstallSystemPrerequisiteRequest): Promise<EnvironmentSystemPrerequisite> {
    if (hasWailsBridge()) return requireWails().InstallEnvironmentSystemPrerequisite(sessionToken, request)
    return httpJSON<EnvironmentSystemPrerequisite>('/api/v1/environment/system-prerequisites/install', { method: 'POST', body: JSON.stringify(request) })
  },
  async selectDirectory(title: string, initial = ''): Promise<string> {
    if (hasWailsBridge()) return requireWails().SelectDirectory(sessionToken, title, initial)
    if (hasElectronShell() && window.agmpElectron?.selectDirectory) return window.agmpElectron.selectDirectory(title, initial)
    return ''
  },
  async openLocalPath(target: string): Promise<string> {
    if (hasElectronShell() && window.agmpElectron?.openPath) return window.agmpElectron.openPath(target)
    return ''
  },
  ping(): Promise<string> {
    if (hasWailsBridge()) return requireWails().Ping()
    return httpJSON<{ message: string }>('/api/v1/health').then(value => value.message)
  },
  appInfo(): Promise<AppInfo> {
    if (hasWailsBridge()) return requireWails().GetAppInfo(sessionToken)
    return httpJSON<AppInfo>('/api/v1/info')
  },
  platformConfig(): Promise<PlatformConfig> {
    if (hasWailsBridge()) return requireWails().GetPlatformConfig(sessionToken)
    return httpJSON<PlatformConfig>('/api/v1/platform/config')
  },
  platformRuntime(): Promise<PlatformRuntimeContract> {
    if (hasWailsBridge()) return requireWails().GetPlatformRuntimeContract(sessionToken)
    return httpJSON<PlatformRuntimeContract>('/api/v1/platform/runtime')
  },
  gamePacks(): Promise<GamePack[]> {
    if (hasWailsBridge()) return requireWails().GetGamePacks(sessionToken)
    return httpJSON<GamePack[]>('/api/v1/game-packs')
  },
  gameInstances(): Promise<GameInstance[]> {
    if (hasWailsBridge()) return requireWails().GetGameInstances(sessionToken)
    return httpJSON<GameInstance[]>('/api/v1/instances')
  },
  settings(): Promise<AGMPSettings> {
    if (hasWailsBridge()) return requireWails().GetSettings(sessionToken)
    return httpJSON<AGMPSettings>('/api/v1/settings')
  },
  saveSettings(value: AGMPSettings): Promise<void> {
    if (hasWailsBridge()) return requireWails().SaveSettings(sessionToken, value)
    return httpJSON<void>('/api/v1/settings', { method: 'PUT', body: JSON.stringify(value) })
  },
  logs(limit = 100): Promise<string[]> {
    if (hasWailsBridge()) return requireWails().GetRecentLogs(sessionToken, limit)
    return httpJSON<string[]>(`/api/v1/logs?limit=${encodeURIComponent(limit)}`)
  },
  globalLogCatalog(request: GlobalLogCatalogRequest): Promise<GlobalLogCatalogPage> {
    if (hasWailsBridge()) return requireWails().GetGlobalLogCatalog(sessionToken, request)
    return httpJSON<GlobalLogCatalogPage>('/api/v1/loghub/catalog', { method: 'POST', body: JSON.stringify(request) })
  },
  readGlobalLog(request: GlobalLogReadRequest): Promise<GlobalLogReadPage> {
    if (hasWailsBridge()) return requireWails().ReadGlobalLog(sessionToken, request)
    return httpJSON<GlobalLogReadPage>('/api/v1/loghub/read', { method: 'POST', body: JSON.stringify(request) })
  },
  exportGlobalLog(id: string): Promise<GlobalLogExportResult> {
    if (hasWailsBridge()) return requireWails().ExportGlobalLog(sessionToken, id)
    return httpJSON<GlobalLogExportResult>(`/api/v1/loghub/logs/${encodeURIComponent(id)}/export`, { method: 'POST' })
  },
  exportGlobalLogs(request: GlobalLogCatalogRequest): Promise<GlobalLogExportResult> {
    if (hasWailsBridge()) return requireWails().ExportGlobalLogs(sessionToken, request)
    return httpJSON<GlobalLogExportResult>('/api/v1/loghub/export', { method: 'POST', body: JSON.stringify(request) })
  },
  deleteGlobalLog(id: string): Promise<GlobalLogMutationResult> {
    if (hasWailsBridge()) return requireWails().DeleteGlobalLog(sessionToken, id)
    return httpJSON<GlobalLogMutationResult>(`/api/v1/loghub/logs/${encodeURIComponent(id)}`, { method: 'DELETE' })
  },
  deleteGlobalLogs(request: GlobalLogDeleteFilteredRequest): Promise<GlobalLogMutationResult> {
    if (hasWailsBridge()) return requireWails().DeleteGlobalLogs(sessionToken, request)
    return httpJSON<GlobalLogMutationResult>('/api/v1/loghub/delete-filtered', { method: 'POST', body: JSON.stringify(request) })
  },
  clearGlobalLogHistory(): Promise<GlobalLogMutationResult> {
    if (hasWailsBridge()) return requireWails().ClearGlobalLogHistory(sessionToken)
    return httpJSON<GlobalLogMutationResult>('/api/v1/loghub/history', { method: 'DELETE' })
  },
  openGlobalLogFolder(): Promise<void> {
    if (hasWailsBridge()) return requireWails().OpenGlobalLogFolder(sessionToken)
    return httpJSON<void>('/api/v1/loghub/open-folder', { method: 'POST' })
  },
  currentTime(): Promise<string> {
    if (hasWailsBridge()) return requireWails().CurrentTime(sessionToken)
    return httpJSON<{ value: string }>('/api/v1/time').then(value => value.value)
  },
  steamEnvironment(): Promise<SteamEnvironment> {
    if (hasWailsBridge()) return requireWails().GetSteamEnvironment(sessionToken)
    return httpJSON<SteamEnvironment>('/api/v1/steam/environment')
  },
  steamApps(): Promise<SteamAppInventory> {
    if (hasWailsBridge()) return requireWails().GetSteamApps(sessionToken)
    return httpJSON<SteamAppInventory>('/api/v1/steam/apps')
  },
  steamSnapshot(): Promise<SteamSnapshot> {
    if (hasWailsBridge()) return requireWails().GetSteamSnapshot(sessionToken)
    return httpJSON<SteamSnapshot>('/api/v1/steam/snapshot', { method: 'POST' })
  },
  startSteamInstall(request: SteamMaintenanceInstallRequest): Promise<SteamMaintenanceTask> {
    if (hasWailsBridge()) return requireWails().StartSteamInstall(sessionToken, request)
    return httpJSON<SteamMaintenanceTask>('/api/v1/steam/maintenance/install', { method: 'POST', body: JSON.stringify(request) })
  },
  startSteamValidation(request: SteamMaintenanceValidateRequest): Promise<SteamMaintenanceTask> {
    if (hasWailsBridge()) return requireWails().StartSteamValidation(sessionToken, request)
    return httpJSON<SteamMaintenanceTask>('/api/v1/steam/maintenance/validate', { method: 'POST', body: JSON.stringify(request) })
  },
  steamMaintenanceTask(id: string): Promise<SteamMaintenanceTask> {
    if (hasWailsBridge()) return requireWails().GetSteamMaintenanceTask(sessionToken, id)
    return httpJSON<SteamMaintenanceTask>(`/api/v1/steam/maintenance/tasks/${encodeURIComponent(id)}`)
  },
  gameWorkspace(id: string): Promise<GameWorkspaceSnapshot> {
    if (hasWailsBridge()) return requireWails().GetGameWorkspaceSnapshot(sessionToken, id)
    return httpJSON<GameWorkspaceSnapshot>(`/api/v1/games/${encodeURIComponent(id)}/snapshot`, { method: 'POST' })
  },
  dstEnvironment(): Promise<DSTEnvironment> {
    if (hasWailsBridge()) return requireWails().GetDSTEnvironment(sessionToken)
    return httpJSON<DSTEnvironment>('/api/v1/dst/environment')
  },
  dstWorkspace(): Promise<DSTWorkspaceSnapshot> {
    if (hasWailsBridge()) return requireWails().GetDSTWorkspaceSnapshot(sessionToken)
    return httpJSON<DSTWorkspaceSnapshot>('/api/v1/dst/workspace', { method: 'POST' })
  },
  dstDedicated(): Promise<DSTDedicatedSnapshot> {
    if (hasWailsBridge()) return requireWails().GetDSTDedicatedServer(sessionToken)
    return httpJSON<DSTDedicatedSnapshot>('/api/v1/dst/dedicated')
  },
  dstDedicatedPreferences(): Promise<DSTDedicatedPreferences> {
    if (hasWailsBridge()) return requireWails().GetDSTDedicatedPreferences(sessionToken)
    return httpJSON<DSTDedicatedPreferences>('/api/v1/dst/dedicated/preferences')
  },
  saveDSTDedicatedPreferences(value: DSTDedicatedPreferences): Promise<void> {
    if (hasWailsBridge()) return requireWails().SaveDSTDedicatedPreferences(sessionToken, value)
    return httpJSON<void>('/api/v1/dst/dedicated/preferences', { method: 'PUT', body: JSON.stringify(value) })
  },
  openDSTTokenPage(): Promise<void> {
    if (hasWailsBridge()) return requireWails().OpenDSTTokenPage(sessionToken)
    return httpJSON<void>('/api/v1/dst/token/page', { method: 'POST' })
  },
  dstTokenStatus(clusterPath: string): Promise<DSTTokenStatus> {
    if (hasWailsBridge()) return requireWails().GetDSTTokenStatus(sessionToken, clusterPath)
    const query = new URLSearchParams({ clusterPath })
    return httpJSON<DSTTokenStatus>(`/api/v1/dst/token/status?${query.toString()}`)
  },
  saveDSTToken(request: DSTTokenSaveRequest): Promise<DSTTokenStatus> {
    if (hasWailsBridge()) return requireWails().SaveDSTToken(sessionToken, request)
    return httpJSON<DSTTokenStatus>('/api/v1/dst/token', { method: 'PUT', body: JSON.stringify(request) })
  },
  importDSTToken(request: DSTTokenImportRequest): Promise<DSTTokenStatus> {
    if (hasWailsBridge()) return requireWails().ImportDSTToken(sessionToken, request)
    return httpJSON<DSTTokenStatus>('/api/v1/dst/token/import', { method: 'POST', body: JSON.stringify(request) })
  },
  inspectDSTKleiPackage(request: DSTKleiPackageInspectRequest): Promise<DSTKleiPackagePreview> {
    if (hasWailsBridge()) return requireWails().InspectDSTKleiPackage(sessionToken, request)
    return httpJSON<DSTKleiPackagePreview>('/api/v1/dst/klei-package/inspect', { method: 'POST', body: JSON.stringify(request) })
  },
  importDSTKleiPackage(request: DSTKleiPackageImportRequest): Promise<DSTKleiPackageImportResult> {
    if (hasWailsBridge()) return requireWails().ImportDSTKleiPackage(sessionToken, request)
    return httpJSON<DSTKleiPackageImportResult>('/api/v1/dst/klei-package/import', { method: 'POST', body: JSON.stringify(request) })
  },
  dstPreflight(clusterPath: string): Promise<DSTPreflightResult> {
    if (hasWailsBridge()) return requireWails().GetDSTPreflight(sessionToken, clusterPath)
    const query = new URLSearchParams({ clusterPath })
    return httpJSON<DSTPreflightResult>(`/api/v1/dst/preflight?${query.toString()}`)
  },
  importDSTCluster(request: DSTClusterImportRequest): Promise<DSTClusterImportResult> {
    if (hasWailsBridge()) return requireWails().ImportDSTCluster(sessionToken, request)
    return httpJSON<DSTClusterImportResult>('/api/v1/dst/clusters/import', { method: 'POST', body: JSON.stringify(request) })
  },
  startDSTMaster(request: DSTStartMasterRequest): Promise<DSTProcessSnapshot> {
    if (hasWailsBridge()) return requireWails().StartDSTMaster(sessionToken, request)
    return httpJSON<DSTProcessSnapshot>('/api/v1/dst/runtime/master/start', { method: 'POST', body: JSON.stringify(request) })
  },
  startDSTCluster(request: DSTStartClusterRequest): Promise<DSTClusterRuntimeSnapshot> {
    if (hasWailsBridge()) return requireWails().StartDSTCluster(sessionToken, request)
    return httpJSON<DSTClusterRuntimeSnapshot>('/api/v1/dst/runtime/cluster/start', { method: 'POST', body: JSON.stringify(request) })
  },
  startDSTShard(request: DSTStartShardRequest): Promise<DSTProcessSnapshot> {
    if (hasWailsBridge()) return requireWails().StartDSTShard(sessionToken, request)
    return httpJSON<DSTProcessSnapshot>('/api/v1/dst/runtime/shard/start', { method: 'POST', body: JSON.stringify(request) })
  },
  dstClusterStatus(request: DSTClusterRequest): Promise<DSTClusterRuntimeSnapshot> {
    if (hasWailsBridge()) return requireWails().GetDSTClusterStatus(sessionToken, request)
    return httpJSON<DSTClusterRuntimeSnapshot>('/api/v1/dst/runtime/cluster/status', { method: 'POST', body: JSON.stringify(request) })
  },
  dstPortStatus(request: DSTClusterRequest): Promise<DSTPortReport> {
    if (hasWailsBridge()) return requireWails().GetDSTPortStatus(sessionToken, request)
    return httpJSON<DSTPortReport>('/api/v1/dst/runtime/ports/status', { method: 'POST', body: JSON.stringify(request) })
  },
  dstPortConfiguration(request: DSTClusterRequest): Promise<DSTPortConfiguration> {
    if (hasWailsBridge()) return requireWails().GetDSTPortConfiguration(sessionToken, request)
    return httpJSON<DSTPortConfiguration>('/api/v1/dst/runtime/ports/configuration', { method: 'POST', body: JSON.stringify(request) })
  },
  configureDSTPorts(request: DSTPortConfigureRequest): Promise<DSTPortConfigureResult> {
    if (hasWailsBridge()) return requireWails().ConfigureDSTPorts(sessionToken, request)
    return httpJSON<DSTPortConfigureResult>('/api/v1/dst/runtime/ports/configuration', { method: 'PUT', body: JSON.stringify(request) })
  },
  cleanupDSTPorts(request: DSTPortCleanupRequest): Promise<DSTPortCleanupResult> {
    if (hasWailsBridge()) return requireWails().CleanupDSTPorts(sessionToken, request)
    return httpJSON<DSTPortCleanupResult>('/api/v1/dst/runtime/ports/cleanup', { method: 'POST', body: JSON.stringify(request) })
  },
  stopDSTCluster(request: DSTClusterRequest): Promise<DSTClusterRuntimeSnapshot> {
    if (hasWailsBridge()) return requireWails().StopDSTCluster(sessionToken, request)
    return httpJSON<DSTClusterRuntimeSnapshot>('/api/v1/dst/runtime/cluster/stop', { method: 'POST', body: JSON.stringify(request) })
  },
  dstProcessStatus(request: DSTProcessRequest): Promise<DSTProcessLookup> {
    if (hasWailsBridge()) return requireWails().GetDSTProcessStatus(sessionToken, request)
    return httpJSON<DSTProcessLookup>('/api/v1/dst/runtime/status', { method: 'POST', body: JSON.stringify(request) })
  },
  dstProcessLogs(request: DSTProcessLogRequest): Promise<DSTProcessLogBatch> {
    if (hasWailsBridge()) return requireWails().ReadDSTProcessLogs(sessionToken, request)
    return httpJSON<DSTProcessLogBatch>('/api/v1/dst/runtime/logs', { method: 'POST', body: JSON.stringify(request) })
  },
  sendDSTCommand(request: DSTCommandRequest): Promise<DSTProcessSnapshot> {
    if (hasWailsBridge()) return requireWails().SendDSTCommand(sessionToken, request)
    return httpJSON<DSTProcessSnapshot>('/api/v1/dst/runtime/command', { method: 'POST', body: JSON.stringify(request) })
  },
  stopDSTProcess(request: DSTProcessRequest): Promise<DSTProcessSnapshot> {
    if (hasWailsBridge()) return requireWails().StopDSTProcess(sessionToken, request)
    return httpJSON<DSTProcessSnapshot>('/api/v1/dst/runtime/stop', { method: 'POST', body: JSON.stringify(request) })
  },
  dstLogSessions(request: DSTLogListRequest): Promise<DSTLogSession[]> {
    if (hasWailsBridge()) return requireWails().GetDSTLogSessions(sessionToken, request)
    return httpJSON<DSTLogSession[]>('/api/v1/dst/logs/sessions', { method: 'POST', body: JSON.stringify(request) })
  },
  readDSTLog(request: DSTLogReadRequest): Promise<DSTLogReadPage> {
    if (hasWailsBridge()) return requireWails().ReadDSTLog(sessionToken, request)
    return httpJSON<DSTLogReadPage>('/api/v1/dst/logs/read', { method: 'POST', body: JSON.stringify(request) })
  },
  tailDSTLog(request: DSTLogTailRequest): Promise<DSTLogReadPage> {
    if (hasWailsBridge()) return requireWails().TailDSTLog(sessionToken, request)
    return httpJSON<DSTLogReadPage>('/api/v1/dst/logs/tail', { method: 'POST', body: JSON.stringify(request) })
  },
  searchDSTLog(request: DSTLogSearchRequest): Promise<DSTLogSearchResult> {
    if (hasWailsBridge()) return requireWails().SearchDSTLog(sessionToken, request)
    return httpJSON<DSTLogSearchResult>('/api/v1/dst/logs/search', { method: 'POST', body: JSON.stringify(request) })
  },
  dstLogDiagnostics(id: string): Promise<DSTLogDiagnostics> {
    if (hasWailsBridge()) return requireWails().GetDSTLogDiagnostics(sessionToken, id)
    return httpJSON<DSTLogDiagnostics>(`/api/v1/dst/logs/sessions/${encodeURIComponent(id)}/diagnostics`)
  },
  async exportDSTLog(id: string): Promise<DSTLogExportResult> {
    if (hasWailsBridge()) return requireWails().ExportDSTLog(sessionToken, id)
    return httpJSON<DSTLogExportResult>(`/api/v1/dst/logs/sessions/${encodeURIComponent(id)}/export`, { method: 'POST' })
  },
  async exportDSTLogBundle(request: DSTLogBundleRequest): Promise<DSTLogExportResult> {
    if (hasWailsBridge()) return requireWails().ExportDSTLogBundle(sessionToken, request)
    return httpJSON<DSTLogExportResult>('/api/v1/dst/logs/bundle/export', { method: 'POST', body: JSON.stringify(request) })
  },
  buildDSTLaunchSpec(request: DSTLaunchRequest): Promise<DSTLaunchSpec> {
    if (hasWailsBridge()) return requireWails().BuildDSTLaunchSpec(sessionToken, request)
    return httpJSON<DSTLaunchSpec>('/api/v1/dst/launch-spec', { method: 'POST', body: JSON.stringify(request) })
  },
}
