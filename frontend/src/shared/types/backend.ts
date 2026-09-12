
export interface XiaoYuRuntimeStatus {
  enabled: boolean
  available: boolean
  ready: boolean
  name: string
  version: string
  protocol: string
  path: string
  capabilities: string[]
  toolCount: number
  message: string
}

export interface XiaoYuToolSpec {
  name: string
  description: string
  risk: 'read' | 'operate' | 'modify' | 'destructive' | 'system' | string
  category: string
  manual: boolean
  xiaoyu: boolean
  parameters?: Record<string, unknown>
  source?: string
}

export type XiaoYuPluginState = 'pending' | 'loading' | 'active' | 'failed' | 'unloading' | 'disposed' | string

export interface XiaoYuPluginManifest {
  id: string
  name: string
  version: string
  apiVersion: string
  source: string
  kinds: string[]
  requires?: string[]
  provides?: string[]
  permissions?: string[]
  configSchema?: Record<string, unknown>
  description?: string
  experimental?: boolean
}

export interface XiaoYuPluginSnapshot {
  manifest: XiaoYuPluginManifest
  state: XiaoYuPluginState
  error?: string
}

export interface XiaoYuCapabilitySnapshot {
  plugins: XiaoYuPluginSnapshot[]
  services: string[]
  tools: XiaoYuToolSpec[]
}

export interface XiaoYuTraceEvent {
  sequence: number
  time: string
  type: string
  pluginId?: string
  runId?: string
  summary?: string
  data?: Record<string, unknown>
}

export interface XiaoYuDSHBundle {
  packageName: string
  version: string
  directory: string
  kind: string
  patch?: string
}

export interface XiaoYuDSHMountRequest {
  directory: string
  trusted: boolean
  xiaoyu: boolean
  config?: Record<string, unknown>
}

export interface XiaoYuBrainInfo {
  id: string
  name: string
  model?: string
  source?: string
  ready: boolean
  message?: string
  capabilities?: XiaoYuModelCapabilities
}

export type XiaoYuModelProtocol = 'openai-responses' | 'deepseek' | 'openai-compatible' | 'anthropic' | 'gemini' | 'codex-app-server' | string
export type XiaoYuModelAuthMode = 'api-key' | 'subscription' | 'local' | string
export type XiaoYuModelProviderKind = 'model-api' | 'local-runtime' | 'agent-provider' | string

export interface XiaoYuModelCapabilities {
  adapter: string
  native: boolean
  reasoning: boolean
  toolCalling: boolean
  toolChoice?: boolean
  thinkingToolChoiceCompatible?: boolean
  parallelToolCalls?: boolean
  replay: boolean
  reasoningReplay?: boolean
  vision: boolean
  streaming: boolean
  inputModalities?: string[]
  nativeToolKinds?: string[]
  notes?: string[]
}

export interface XiaoYuModelPreset {
  id: string
  name: string
  description: string
  protocol: XiaoYuModelProtocol
  defaultBaseUrl: string
  local: boolean
  apiKeyOptional: boolean
  kind: XiaoYuModelProviderKind
  authModes: XiaoYuModelAuthMode[]
  defaultAuthMode: XiaoYuModelAuthMode
  executable?: string
  brainEligible: boolean
}

export interface XiaoYuModelProfile {
  id: string
  name: string
  provider: string
  protocol: XiaoYuModelProtocol
  authMode: XiaoYuModelAuthMode
  providerKind: XiaoYuModelProviderKind
  brainEligible: boolean
  baseUrl: string
  model: string
  enabled: boolean
  contextWindow: number
  maxOutputTokens: number
  thinkingMode: string
  reasoningEffort: string
  extra?: Record<string, unknown>
  capabilities: XiaoYuModelCapabilities
  hasApiKey: boolean
  hasCredential: boolean
  lastTestAt?: string
  lastTestOk?: boolean
  lastTestMessage?: string
  isDefault: boolean
  createdAt: string
  updatedAt: string
}

export interface XiaoYuModelCatalog {
  presets: XiaoYuModelPreset[]
  profiles: XiaoYuModelProfile[]
  defaultBrainId?: string
  brainReady: boolean
  message: string
}

export interface XiaoYuSaveModelRequest {
  id?: string
  name: string
  provider: string
  protocol: XiaoYuModelProtocol
  authMode?: XiaoYuModelAuthMode
  baseUrl: string
  model: string
  apiKey?: string
  enabled: boolean
  contextWindow: number
  maxOutputTokens: number
  thinkingMode: string
  reasoningEffort: string
  extra?: Record<string, unknown>
}

export interface XiaoYuModelConnectionRequest {
  id?: string
  provider: string
  protocol: XiaoYuModelProtocol
  authMode?: XiaoYuModelAuthMode
  baseUrl: string
  model: string
  apiKey?: string
  thinkingMode?: string
  reasoningEffort?: string
  extra?: Record<string, unknown>
}

export interface XiaoYuModelConnectionResult {
  ok: boolean
  message: string
  endpoint: string
  latencyMs: number
  models?: string[]
  capabilities?: XiaoYuModelCapabilities
}

export type XiaoYuIntelligenceVisibility = 'private' | 'group' | 'organization'
export type XiaoYuMemoryKind = 'session' | 'task' | 'user' | 'server' | 'instance' | 'experience'
export type XiaoYuMemorySensitivity = 'normal' | 'sensitive'

export interface XiaoYuMemoryScope {
  type: string
  id: string
}

export interface XiaoYuMemoryRecord {
  id: string
  organizationId: string
  groupId?: string
  ownerUserId: string
  kind: XiaoYuMemoryKind
  scope: XiaoYuMemoryScope
  content: string
  source: string
  sensitivity: XiaoYuMemorySensitivity
  confidence: number
  visibility: XiaoYuIntelligenceVisibility
  enabled: boolean
  expiresAt?: string
  createdAt: string
  updatedAt: string
}

export interface XiaoYuMemorySaveRequest {
  id?: string
  kind: XiaoYuMemoryKind
  scope: XiaoYuMemoryScope
  content: string
  source?: string
  sensitivity?: XiaoYuMemorySensitivity
  confidence?: number
  visibility?: XiaoYuIntelligenceVisibility
  enabled?: boolean
  expiresAt?: string
}

export interface XiaoYuSkillDefinition {
  id: string
  organizationId?: string
  groupId?: string
  ownerUserId?: string
  name: string
  description?: string
  prompt: string
  tags?: string[]
  gameIds?: string[]
  toolAllowlist?: string[]
  checklist?: string[]
  validators?: string[]
  enabled: boolean
  visibility: XiaoYuIntelligenceVisibility
  builtin: boolean
  createdAt: string
  updatedAt: string
}

export interface XiaoYuSkillSaveRequest {
  id?: string
  name: string
  description?: string
  prompt: string
  tags?: string[]
  gameIds?: string[]
  toolAllowlist?: string[]
  checklist?: string[]
  validators?: string[]
  enabled?: boolean
  visibility?: XiaoYuIntelligenceVisibility
}

export interface XiaoYuExpertDefinition {
  id: string
  organizationId?: string
  groupId?: string
  ownerUserId?: string
  name: string
  description?: string
  prompt: string
  domains?: string[]
  gameIds?: string[]
  knowledge?: string[]
  skillIds?: string[]
  toolAllowlist?: string[]
  checklist?: string[]
  validators?: string[]
  recoveryRules?: string[]
  enabled: boolean
  visibility: XiaoYuIntelligenceVisibility
  builtin: boolean
  createdAt: string
  updatedAt: string
}

export interface XiaoYuExpertSaveRequest {
  id?: string
  name: string
  description?: string
  prompt: string
  domains?: string[]
  gameIds?: string[]
  knowledge?: string[]
  skillIds?: string[]
  toolAllowlist?: string[]
  checklist?: string[]
  validators?: string[]
  recoveryRules?: string[]
  enabled?: boolean
  visibility?: XiaoYuIntelligenceVisibility
}

export interface XiaoYuIntelligenceCatalog {
  memories: XiaoYuMemoryRecord[]
  skills: XiaoYuSkillDefinition[]
  experts: XiaoYuExpertDefinition[]
}

export interface XiaoYuRunContext {
  sessionId?: string
  taskId?: string
  gameId?: string
  serverId?: string
  instanceId?: string
  uiRoute?: string
}

export interface XiaoYuLoopConfig {
  maxSteps: number
  maxToolCalls: number
  maxFailures: number
  repeatLimit: number
}

export type XiaoYuRunStatus = 'created' | 'running' | 'paused' | 'waiting_approval' | 'waiting_user' | 'completed' | 'failed' | 'cancelled' | string
export type XiaoYuControlOwner = 'xiaoyu' | 'human' | 'shared' | string

export interface XiaoYuToolCall {
  name: string
  arguments?: Record<string, unknown>
  approvalId?: string
}

export interface XiaoYuObservation {
  step: number
  tool?: string
  summary: string
  data?: unknown
  error?: string
  pending?: boolean
  denied?: boolean
  approvalId?: string
  time: string
}

export interface XiaoYuRunState {
  id: string
  goal: string
  status: XiaoYuRunStatus
  controlOwner: XiaoYuControlOwner
  initiatorId?: string
  initiatorName?: string
  context: XiaoYuRunContext
  pausedBy?: string
  pauseReason?: string
  step: number
  toolCalls: number
  failures: number
  message?: string
  phase?: string
  decisionSummary?: string
  approvalId?: string
  pendingCall?: XiaoYuToolCall
  observations: XiaoYuObservation[]
  createdAt: string
  updatedAt: string
}

export interface XiaoYuHarnessSnapshot {
  kernel: string
  pluginApi: string
  brain?: XiaoYuBrainInfo
  capabilities: XiaoYuCapabilitySnapshot
  runs: XiaoYuRunState[]
  limits: XiaoYuLoopConfig
}

export interface XiaoYuAttachmentRequest {
  name?: string
  mediaType: string
  dataBase64: string
}

export interface XiaoYuRunRequest {
  goal: string
  context?: Omit<XiaoYuRunContext, 'taskId'>
  attachments?: XiaoYuAttachmentRequest[]
}

export interface XiaoYuContinueRunRequest {
  message?: string
  attachments?: XiaoYuAttachmentRequest[]
}

export interface XiaoYuControlRequest {
  reason?: string
}


export interface XiaoYuToolCallRequest {
  tool: string
  arguments: Record<string, unknown>
  approvalId?: string
}

export interface XiaoYuToolCallResult {
  tool: string
  decision: 'allow' | 'confirm' | 'deny' | string
  summary: string
  data: unknown
  approvalId?: string
  pending?: boolean
  risk?: string
}

export interface XiaoYuCommandRequest {
  command: string
  workingDirectory: string
  approvalId?: string
}

export interface XiaoYuCommandResult {
  command: string
  cwd: string
  exitCode: number
  stdout: string
  stderr: string
  decision: 'allow' | 'confirm' | 'deny' | string
  approvalId?: string
  pending?: boolean
}

export interface XiaoYuApprovalState {
  mode: 'ask' | 'risk' | 'full' | string
  modeLabel: string
  updatedAt: number
  updatedBy?: string
  pendingCount: number
}

export interface XiaoYuApprovalRequest {
  id: string
  kind: 'tool' | 'command' | string
  subject: string
  risk: string
  summary: string
  requestHash: string
  state: 'pending' | 'approved' | 'rejected' | 'consumed' | string
  createdAt: number
  decidedAt?: number
  decidedBy?: string
  consumedAt?: number
}

export type LicenseState = 'UNLICENSED' | 'ACTIVE' | 'EXPIRING' | 'EXPIRED' | 'DEVICE_MISMATCH' | 'REVOKED' | 'SEAT_LIMIT' | 'OFFLINE_GRACE' | 'SERVER_UNREACHABLE' | string

export interface LicenseStatus {
  enabled: boolean
  activated: boolean
  valid: boolean
  state: LicenseState
  machineCode: string
  installId: string
  identificationCode: string
  licenseId: string
  activationId: string
  edition: string
  features: string[]
  seatLimit: number
  activeSeats: number
  issuedAt: number
  expiresAt: number
  activatedAt: number
  legacy: boolean
  issuerKeyId: string
  vendorKeyId: string
  vendorKeyFingerprint: string
  trustedKeyCount: number
  message: string
}

export interface LicenseActivateRequest {
  activationCode?: string
  certificate?: string
  cdk?: string
}

export interface LicenseFeatureStatus {
  feature: string
  allowed: boolean
  state: LicenseState
  message: string
}


export interface AuthBootstrapStatus {
  ownerExists: boolean
  registrationOpen: boolean
  invitationRegistrationOnly: boolean
  bootstrapLocked: boolean
  storeError: string
  userCount: number
  instanceId: string
  instanceFingerprint: string
}

export interface AuthUser {
  id: string
  username: string
  displayName: string
  role: 'owner' | 'administrator' | 'operator' | string
  organizationId: string
  groupId: string
  email?: string
  emailVerifiedAt?: number
  coreAccess: 'pending' | 'authorized' | 'suspended' | string
  coreAuthorizedAt?: number
  coreAuthorizedBy?: string
  riskState: 'normal' | 'challenge' | 'locked' | string
  riskReason?: string
  riskUpdatedAt?: number
  createdAt: number
}

export interface AuthSession { token: string; user: AuthUser; expiresAt: number }
export interface AuthCreateOwnerRequest { username: string; displayName: string; password: string; email?: string; securityKey: string; requireSecurityKey: boolean }
export interface AuthLoginRequest { username: string; password: string; securityKey: string }
export interface AuthSecurityKeyMaterial { key: string }
export interface AuthSecurityStatus { configured: boolean; verificationEnabled: boolean }
export interface AuthRotateSecurityKeyRequest { password: string }
export interface AuthSetSecurityKeyVerificationRequest { password: string; enabled: boolean }
export interface AuthUpdateDisplayNameRequest { displayName: string }

export interface AuthInstanceIdentity { id: string; fingerprint: string; publicKey: string; createdAt: number }
export interface AuthOrganization { id: string; name: string; defaultGroupId: string; createdAt: number }
export interface AuthInvitationView {
  id: string; role: string; groupId: string; targetUsername?: string; targetEmail?: string; verificationCode: string; createdBy: string; createdAt: number; expiresAt: number; usedAt?: number; usedBy?: string; revokedAt?: number; status: string
}
export interface AuthCreatedInvitation extends AuthInvitationView { token: string; instanceId: string; instanceFingerprint: string; organizationName: string }
export interface AuthCreateInvitationRequest { role: 'administrator' | 'operator'; targetUsername?: string; targetEmail?: string; expiresInHours?: number }
export interface AuthInvitationPreview { invitationId: string; organizationName: string; role: string; groupId: string; targetUsername?: string; targetEmail?: string; expiresAt: number; verificationCode: string; instanceId: string; instanceFingerprint: string }
export interface AuthRegisterInvitationRequest { invitationToken: string; username: string; displayName: string; password: string; email?: string; securityKey?: string; requireSecurityKey?: boolean }

export interface AuthMemberAuthorizationView { id: string; userId: string; username: string; verificationCode: string; createdBy: string; createdAt: number; expiresAt: number; usedAt?: number; revokedAt?: number; status: string }
export interface AuthCreatedMemberAuthorization extends AuthMemberAuthorizationView { token: string; instanceId: string; instanceFingerprint: string }
export interface AuthCreateMemberAuthorizationRequest { userId: string; expiresInHours?: number }
export interface AuthRedeemMemberAuthorizationRequest { token: string }
export interface AuthCredentialStepUpRequest { password: string; securityKey?: string }

export interface AuthEmailSecurityStatus {
  email: string; bound: boolean; verified: boolean; verifiedAt?: number; riskState: string; riskReason?: string; riskUpdatedAt?: number; stepUpVerified: boolean; stepUpValidUntil?: number; coreAccess: string; smtpConfigured: boolean; emailFeaturesAvailable: boolean
}
export interface AuthBindEmailRequest { email: string; password: string }
export interface AuthRequestEmailVerificationRequest { purpose: 'bind' | 'risk' }
export interface AuthConfirmEmailVerificationRequest { code: string }
export interface AuthRequestPasswordResetRequest { username: string }
export interface AuthConfirmPasswordResetRequest { username: string; code: string; newPassword: string }
export interface AuthPasswordResetRequestStatus { accepted: boolean; message: string }

export interface AuthSMTPSettings { configured: boolean; host: string; port: number; username: string; from: string; tlsMode: 'starttls' | 'tls' | 'none' | string; hasPassword: boolean }
export interface AuthSaveSMTPSettingsRequest { host: string; port: number; username: string; password?: string; from: string; tlsMode: 'starttls' | 'tls' | 'none' | string; accountPassword: string; securityKey?: string }

export interface EnvironmentToolStatus {
  detected: boolean
  path: string
  root: string
  source: string
}

export interface EnvironmentGameInstallStatus {
  appId: number
  detected: boolean
  path: string
}

export interface EnvironmentStoragePaths {
  programDir: string
  dataDir: string
  runtimeRoot: string
  steamcmdRoot: string
  steamcmdPath: string
  gameLibraryRoot: string
  instanceConfigRoot: string
  gameSaveRoot: string
  cacheRoot: string
}

export interface EnvironmentSetupStatus {
  platform: string
  architecture: string
  initialized: boolean
  initializedAt: number
  skipped: boolean
  skippedAt: number
  steamcmd: EnvironmentToolStatus
  dedicatedServer: EnvironmentGameInstallStatus
  defaultInstallRoot: string
  paths: EnvironmentStoragePaths
  requiredDirs: string[]
  warnings: string[]
}

export interface EnvironmentInitializeRequest {
  steamcmdPath: string
  defaultInstallRoot: string
  steamcmdRoot?: string
  gameLibraryRoot?: string
}

export interface EnvironmentStoragePathsRequest {
  steamcmdRoot: string
  steamcmdPath: string
  gameLibraryRoot: string
  instanceConfigRoot: string
  gameSaveRoot: string
  cacheRoot: string
}


export type EnvironmentRuntimeKind = 'java' | 'steamcmd'
export interface EnvironmentRuntimeRecord {
  id: string; kind: EnvironmentRuntimeKind; name: string; version: string; major?: number; os: string; arch: string; root: string; executable: string; source: string; managed: boolean; default: boolean; checksum?: string; installedAt?: number; lastVerified?: number; metadata?: Record<string,string>
}
export interface EnvironmentRuntimeCatalog {
  platform: string; architecture: string; runtimeRoot: string; javaMajors: number[]; runtimes: EnvironmentRuntimeRecord[]; warnings: string[]
}
export interface EnvironmentJavaInstallRequest { major: number; targetRoot?: string; setDefault?: boolean }
export interface EnvironmentRegisterRuntimeRequest { kind: EnvironmentRuntimeKind; executable: string; major?: number; setDefault?: boolean }
export interface EnvironmentResolveRuntimeRequest { kind: EnvironmentRuntimeKind; major?: number }
export interface EnvironmentSetDefaultRuntimeRequest { id: string }
export interface EnvironmentGameRuntimeRequirement { id: string; kind: EnvironmentRuntimeKind; major?: number; required: boolean; satisfied: boolean; runtimeId?: string; message: string }
export interface EnvironmentSystemPrerequisite { id: string; name: string; required: boolean; detected: boolean; installable: boolean; packageManager?: string; message: string }
export interface EnvironmentGameRuntimeProfile { gameId: string; name: string; platform: string; architecture: string; ready: boolean; requirements: EnvironmentGameRuntimeRequirement[]; systemPrerequisites: EnvironmentSystemPrerequisite[] }
export interface EnvironmentGameRuntimeProfileRequest { gameId: string; javaMajor?: number }
export interface EnvironmentInstallSystemPrerequisiteRequest { id: string }

export interface SteamCMDInstallRequest { targetRoot: string }
export interface EnvironmentMigratePathRequest { targetRoot: string; removeSource: boolean }
export interface EnvironmentMigrationResult {
  source: string
  target: string
  copiedFiles: number
  copiedBytes: number
  removedOld: boolean
  message: string
}


export interface UpdateStatus {
  enabled: boolean
  currentVersion: string
  latestVersion: string
  updateAvailable: boolean
  channel: string
  releaseName: string
  releaseNotes: string
  releaseUrl: string
  publishedAt: string
  installerName: string
  installerSize: number
  installerSha256: string
  canInstall: boolean
  checkedAt: number
  message: string
}

export interface PreparedUpdate {
  version: string
  installerPath: string
  sha256: string
  size: number
}

export type PlatformSurfaceKind = 'web-client' | 'desktop-client' | 'node-runtime' | string

export interface PlatformSurfaceContract {
  id: string
  name: string
  kind: PlatformSurfaceKind
  state: string
  supportedOs?: string[]
  installRequired: boolean
  browser: boolean
  controlsNodes: boolean
  executesOnNode: boolean
}

export interface PlatformRuntimeContract {
  hostOs: string
  hostArch: string
  nodeKind: string
  nativeExecution: boolean
  remoteControl: boolean
  surfaces: PlatformSurfaceContract[]
}

export interface PlatformConfig {
  app: PlatformAppConfig
  ui: PlatformUIConfig
  ai: PlatformAIConfig
  permissions: PlatformPermissionsConfig
  paths: PlatformPathsConfig
  logging: PlatformLoggingConfig
  games: PlatformGamesConfig
  modules: PlatformModulesConfig
  update: PlatformUpdateConfig
}


export interface PlatformUpdateConfig {
  enabled: boolean
  provider: string
  repository: string
  apiBaseUrl: string
  channel: string
  checkOnStartup: boolean
  minimumCheckIntervalMinutes: number
  assetPattern: string
  requireSha256: boolean
  installMode: string
  preserveRuntimeData: boolean
}

export interface PlatformAppConfig {
  productName: string
  edition: string
  slogan: string
  defaultRoute: string
  locale: string
  telemetry: boolean
}

export interface PlatformNavigationItem {
  id: string
  to: string
  label: string
  icon: string
}

export interface PlatformNavigationGroup {
  id: string
  title: string
  items: PlatformNavigationItem[]
}

export interface PlatformWorkbenchConfig {
  title: string
  subtitle: string
  placeholder: string
  suggestions: string[]
}

export interface PlatformUIConfig {
  theme: string
  accent: string
  sidebarWidth: number
  compactSidebarWidth: number
  showTopbar: boolean
  navigation: PlatformNavigationGroup[]
  workbench: PlatformWorkbenchConfig
}

export interface PlatformAIRuntimeConfig {
  enabled: boolean
  engine: string
  protocol: string
  transport: string
  binary: string
  manualFallback: boolean
}

export interface PlatformAIAgentLoopConfig {
  planner: string
  streaming: boolean
  maxSteps: number
  maxToolCalls: number
  maxFailures: number
  repeatLimit: number
  verifyAfterAction: boolean
}

export interface PlatformAIDSHCompatibilityConfig {
  enabled: boolean
  mode: string
  autoMount: boolean
  nodeExecutable: string
}

export interface PlatformAIHarnessConfig {
  kernel: string
  everythingAsPlugin: boolean
  traceMode: string
  deepseekHarness: PlatformAIDSHCompatibilityConfig
}

export interface PlatformAIConfig {
  enabled: boolean
  primaryInteraction: 'agent' | 'manual' | string
  stream: boolean
  approvalMode: string
  allowManualFallback: boolean
  maxConcurrentRuns: number
  toolTimeoutSeconds: number
  conversationHistoryLimit: number
  storeConversationHistory: boolean
  secretStorage: string
  runtime: PlatformAIRuntimeConfig
  agentLoop: PlatformAIAgentLoopConfig
  harness: PlatformAIHarnessConfig
  brainConfiguration: string
}

export interface PlatformApprovalMode {
  id: string
  label: string
  description: string
}

export interface PlatformApprovalPolicies {
  ask: Record<string, string>
  risk: Record<string, string>
  full: Record<string, string>
}

export interface PlatformPermissionsConfig {
  administratorRole: string
  allowFullAccessMode: boolean
  allowArbitraryShell: boolean
  approvalModes: PlatformApprovalMode[]
  policies: PlatformApprovalPolicies
  fileScope: string[]
}

export interface PlatformPathsConfig {
  dataDir: string
  logDir: string
  backupDir: string
  instanceDir: string
  tempDir: string
  exportDir: string
  pluginDir: string
  cacheDir: string
}

export interface PlatformLoggingConfig {
  level: string
  retentionDays: number
  maxFileSizeMB: number
  maxFilesPerSource: number
  compressRotated: boolean
  flushIntervalMs: number
  operationAudit: boolean
  redactSecrets: boolean
  exportSubdir: string
  catalogPageSize: number
  readPageSize: number
  autoRefreshSeconds: number
  directories: PlatformLoggingDirectories
}

export interface PlatformLoggingDirectories {
  core: string
  operations: string
  audit: string
  ai: string
  steam: string
  games: string
  nodes: string
}

export interface PlatformGameTemplate {
  id: string
  family: string
  nameZh: string
  nameEn: string
  state: string
  gameAppId?: number
  serverAppId?: number
  supportedOs?: string[]
  capabilities?: string[]
  uiPanels?: string[]
  installStrategy?: string
  factSources?: string[]
}

export type GameInstanceOrigin = 'discovered' | 'visual' | 'agent' | string

export interface GameInstance {
  id: string
  name: string
  gameId: string
  origin: GameInstanceOrigin
  nodeOs: string
  nodeArch: string
  installPath: string
  runtimeState: string
  desiredState?: string
  health?: string
  gameVersion?: string
  serverType?: string
  serverVersion?: string
  address?: string
  port?: number
  capabilities: string[]
  managed: boolean
  createdAt?: number
  updatedAt?: number
}

export interface PlatformGamesConfig { templates: PlatformGameTemplate[] }

export interface GamePack {
  id: string
  family: string
  nameZh: string
  nameEn: string
  state: string
  supportedOs?: string[]
  capabilities?: string[]
  uiPanels?: string[]
  installStrategy?: string
  factSources?: string[]
}

export type MinecraftSoftware = 'vanilla' | 'paper' | 'fabric'
export interface MinecraftArtifact { software: MinecraftSoftware; url: string; fileName: string; hashAlgorithm?: string; hash?: string; size?: number; build?: string; loader?: string; installer?: string; trust: string }
export interface MinecraftVersionFacts { requestedVersion?: string; version: string; latestRelease: string; javaMajor: number; artifact: MinecraftArtifact; sources: string[] }
export interface MinecraftPlanRequest { name: string; version?: string; software: MinecraftSoftware; memoryMb: number; port: number; onlineMode: boolean; whitelist: boolean; eulaAccepted: boolean; autoInstallJava: boolean; startAfterDeploy: boolean; origin?: GameInstanceOrigin }
export interface MinecraftPlan { id: string; gameId: string; name: string; origin: GameInstanceOrigin; installPath: string; versionFacts: MinecraftVersionFacts; memoryMb: number; port: number; onlineMode: boolean; whitelist: boolean; eulaAccepted: boolean; autoInstallJava: boolean; startAfterDeploy: boolean; steps: string[] }
export interface MinecraftRuntimeSnapshot { instanceId: string; state: string; pid?: number; ready: boolean; startedAt?: number; updatedAt?: number; exitCode?: number; error?: string; logCursor: number }
export interface MinecraftProbeResult { online: boolean; version?: string; protocol?: number; playersOnline?: number; playersMax?: number; description?: string; latencyMs: number }
export interface MinecraftDeploymentResult { plan: MinecraftPlan; instance: GameInstance; runtime: MinecraftRuntimeSnapshot; probe?: MinecraftProbeResult }
export interface MinecraftLogLine { sequence: number; timestamp: number; text: string }
export interface MinecraftLogBatch { lines: MinecraftLogLine[]; nextCursor: number; dropped: boolean }

export interface PlatformModuleConfig {
  id: string
  name: string
  category: string
  route: string
  status: string
  phase: string
  features: string[]
  sourceRefs?: string[]
}

export interface PlatformModulesConfig { modules: PlatformModuleConfig[] }

export interface AppInfo {
  name: string
  version: string
  slogan: string
  goVersion: string
  platform: string
  dataDir: string
}

export interface AGMPSettings {
  theme: 'dark' | 'light' | 'system'
  language: string
  debug: boolean
}

export interface SteamLibrary {
  path: string
  steamAppsPath: string
  primary: boolean
}

export interface SteamEnvironment {
  detected: boolean
  installPath: string
  executablePath: string
  libraries: SteamLibrary[]
  warnings: string[]
}

export interface SteamAppInstallation {
  appId: number
  name: string
  installDir: string
  installPath: string
  installPathExists: boolean
  libraryPath: string
  manifestPath: string
  buildId: string
  lastUpdated: number
  sizeOnDisk: number
  stateFlags: number
}

export interface SteamAppInventory {
  detected: boolean
  apps: SteamAppInstallation[]
  warnings: string[]
}

export interface SteamSnapshot {
  environment: SteamEnvironment
  inventory: SteamAppInventory
  scannedAt: number
  durationMs: number
}


export interface SteamMaintenanceValidateRequest {
  appId: number
}

export interface SteamMaintenanceInstallRequest {
  appId: number
}

export type SteamMaintenanceState = 'requested' | 'monitoring' | 'completed' | 'failed' | 'timeout' | string
export type SteamMaintenanceOperation = 'validate' | 'install' | string
export type SteamMaintenanceProgressMode = 'determinate' | 'indeterminate' | string

export interface SteamMaintenanceTask {
  id: string
  appId: number
  operation: SteamMaintenanceOperation
  uri: string
  state: SteamMaintenanceState
  phase: string
  message: string
  startedAt: number
  updatedAt: number
  buildId: string
  progress: number
  progressMode: SteamMaintenanceProgressMode
  progressSource: string
  progressEstimated: boolean
  bytesDone: number
  bytesTotal: number
  steamRoot: string
  libraryPath: string
  installPath: string
  manifestPath: string
  contentLogPath: string
  stateFlags: number
  validationFiles: number
  validationBytes: number
  mismatchedFiles: number
  mismatchedBytes: number
  completionConfirmed: boolean
  completionSource: string
  error: string
}

export interface GameSteamMetadata {
  gameAppId: number
  serverAppId: number
}

export interface GameCatalogEntry {
  id: string
  family: 'steam' | 'minecraft' | string
  nameZh: string
  nameEn: string
  description: string
  aliases: string[]
  steam?: GameSteamMetadata
}

export interface GameAppState {
  appId: number
  installed: boolean
  name: string
  installPath: string
  installPathExists: boolean
  libraryPath: string
  buildId: string
  lastUpdated: number
  sizeOnDisk: number
}

export interface GameWorkspaceGameState {
  catalog: GameCatalogEntry
  gameApp?: GameAppState
  serverApp?: GameAppState
}

export interface GameWorkspaceSnapshot {
  game: GameWorkspaceGameState
  steam: SteamEnvironment
  warnings: string[]
  scannedAt: number
  durationMs: number
}


export interface DSTShard {
  name: string
  path: string
  modOverridesPath: string
  levelDataPath: string
}

export interface DSTCluster {
  name: string
  path: string
  source: 'server' | 'local'
  distribution: 'steam' | 'wegame'
  shards: DSTShard[]
  modOverridesPath: string
  adminListPath: string
  tokenPath: string
  blockListPath: string
}


export interface DSTDedicatedInstallation {
  detected: boolean
  valid: boolean
  appId: number
  rootDir: string
  binDir: string
  executable: string
  architecture: 'unknown' | 'windows-amd64' | 'windows-386' | string
  bitness: number
  source: 'none' | 'manual' | 'steam' | string
  libraryPath: string
  issues: string[]
}

export interface DSTConfDirInfo {
  documentsDir: string
  baseDir: string
  kleiRoot: string
  argument: string
  default: boolean
  valid: boolean
  error: string
}

export interface DSTDedicatedSnapshot {
  installation: DSTDedicatedInstallation
  confDir: DSTConfDirInfo
  manualPath: string
  extraArgs: string
  warnings: string[]
}

export interface DSTDedicatedPreferences {
  dedicatedServerPath: string
  dedicatedServerExtraArgs: string
}

export interface DSTWorkspaceSnapshot {
  workspace: GameWorkspaceSnapshot
  dedicated: DSTDedicatedSnapshot
  environment: DSTEnvironment
}



export type DSTTokenState = 'missing' | 'empty' | 'configured' | string

export interface DSTTokenStatus {
  state: DSTTokenState
  configured: boolean
  path: string
  size: number
  modifiedAt: number
  message: string
}

export interface DSTTokenSaveRequest {
  clusterPath: string
  value: string
}

export interface DSTTokenImportRequest {
  clusterPath: string
  sourcePath: string
}

export type DSTKleiPackageImportMode = 'token_only' | 'full_config'

export interface DSTKleiPackageInspectRequest {
  archiveName: string
  archiveBase64: string
}

export interface DSTKleiPackageImportRequest extends DSTKleiPackageInspectRequest {
  clusterPath: string
  mode: DSTKleiPackageImportMode
}

export interface DSTKleiPackagePreview {
  archiveName: string
  rootPrefix: string
  hasToken: boolean
  hasClusterIni: boolean
  hasMasterServerIni: boolean
  hasCavesServerIni: boolean
  serverName: string
  maxPlayers: string
  gameMode: string
  description: string
  passworded: boolean
  entryCount: number
  uncompressedBytes: number
}

export interface DSTKleiPackageImportResult {
  mode: DSTKleiPackageImportMode | string
  clusterPath: string
  tokenConfigured: boolean
  configFilesCopied: number
  backupPath: string
  warnings: string[]
  preview: DSTKleiPackagePreview
}

export type DSTPreflightSeverity = 'ok' | 'warning' | 'blocker' | string

export interface DSTPreflightCheck {
  code: string
  label: string
  severity: DSTPreflightSeverity
  message: string
  action: string
}

export interface DSTPreflightResult {
  clusterPath: string
  ready: boolean
  blockers: number
  warnings: number
  checks: DSTPreflightCheck[]
  checkedAt: number
}

export interface DSTClusterImportRequest {
  sourcePath: string
  targetName: string
}

export interface DSTClusterImportResult {
  name: string
  path: string
  sourcePath: string
  copiedFiles: number
  copiedBytes: number
  tokenSkipped: boolean
  warnings: string[]
}

export type DSTServerStatus = 'starting' | 'running' | 'stopping' | 'stopped' | 'crashed'

export interface DSTProcessSnapshot {
  clusterName: string
  clusterPath: string
  shardName: string
  role: 'Master' | 'Secondary' | string
  status: DSTServerStatus | string
  pid: number
  worldReady: boolean
  intentionalShutdown: boolean
  exitCode?: number | null
  startedAt: number
  updatedAt: number
  error: string
  logCursor: number
  logSessionId: string
  logPersistenceError: string
}

export interface DSTProcessLookup {
  found: boolean
  process: DSTProcessSnapshot
}

export interface DSTProcessLogLine {
  sequence: number
  timestamp: number
  text: string
}

export interface DSTProcessLogBatch {
  lines: DSTProcessLogLine[]
  nextCursor: number
  dropped: boolean
}

export interface DSTStartMasterRequest {
  clusterPath: string
  ugcDirectory: string
}

export interface DSTStartClusterRequest {
  clusterPath: string
  ugcDirectory: string
}

export interface DSTStartShardRequest extends DSTStartClusterRequest {
  shardName: string
}

export interface DSTClusterRequest {
  clusterPath: string
}

export interface DSTPortUse {
  port: number
  shardName: string
  purpose: string
  source: string
}

export interface DSTPortCollision {
  port: number
  uses: DSTPortUse[]
}

export interface DSTPortPlan {
  clusterPath: string
  uses: DSTPortUse[]
  collisions: DSTPortCollision[]
  warnings: string[]
  missing: string[]
}

export interface DSTPortOwner {
  port: number
  protocol: string
  pid: number
  processPath: string
  processName: string
  managed: boolean
  managedCluster: string
  managedShard: string
}

export interface DSTPortCheck {
  port: number
  uses: DSTPortUse[]
  occupied: boolean
  owners: DSTPortOwner[]
}

export interface DSTPortReport {
  ready: boolean
  plan: DSTPortPlan
  checks: DSTPortCheck[]
  blockers: string[]
  inspectedAt: number
}

export interface DSTPortSettings {
  shardMasterPort: number
  masterServerPort: number
  masterSteamMasterPort: number
  masterSteamAuthPort: number
  cavesServerPort: number
  cavesSteamMasterPort: number
  cavesSteamAuthPort: number
}

export interface DSTPortConfiguration {
  clusterPath: string
  hasCaves: boolean
  configured: DSTPortSettings
  effective: DSTPortSettings
  recommended: DSTPortSettings
  missing: string[]
  collisions: DSTPortCollision[]
  valid: boolean
}

export interface DSTPortConfigureRequest {
  clusterPath: string
  useRecommended: boolean
  settings: DSTPortSettings
}

export interface DSTPortConfigureResult {
  configuration: DSTPortConfiguration
  report: DSTPortReport
  backupPath: string
}

export interface DSTClusterRuntimeSnapshot {
  clusterPath: string
  clusterName: string
  hasCaves: boolean
  overall: 'stopped' | 'starting' | 'running' | 'partial' | 'stopping' | 'crashed' | string
  shardLink: 'not_applicable' | 'stopped' | 'waiting' | 'ready' | 'disconnected' | string
  master: DSTProcessLookup
  caves: DSTProcessLookup
  ports: DSTPortReport
}

export interface DSTPortCleanupRequest {
  clusterPath: string
  force: boolean
}

export interface DSTPortCleanupResult {
  terminatedPids: number[]
  skippedPids: number[]
  report: DSTPortReport
}

export interface DSTProcessRequest {
  clusterPath: string
  shardName: string
}

export interface DSTProcessLogRequest extends DSTProcessRequest {
  after: number
  limit: number
}

export interface DSTCommandRequest extends DSTProcessRequest {
  command: string
}

export interface DSTLaunchRequest {
  clusterName: string
  shardName: string
  kleiRoot: string
  ugcDirectory: string
  extraArgs: string
}

export interface DSTLaunchSpec {
  clusterName: string
  shardName: string
  role: 'Master' | 'Secondary' | string
  executable: string
  workingDirectory: string
  arguments: string[]
  confDirArgument: string
  ugcDirectory: string
  extraArgs: string
  architecture: string
}


export interface GlobalLogCatalogRequest {
  query: string
  source: string
  kind: string
  gameId: string
  instanceId: string
  shard: string
  status: string
  dateFrom: number
  dateTo: number
  offset: number
  limit: number
}

export interface GlobalLogSummary {
  files: number
  lines: number
  bytes: number
  activeFiles: number
}

export interface GlobalLogFile {
  id: string
  name: string
  relativePath: string
  source: string
  sourceLabel: string
  kind: string
  gameId: string
  gameName: string
  instanceId: string
  instanceName: string
  shard: string
  status: string
  active: boolean
  startedAt: number
  endedAt: number
  modifiedAt: number
  lineCount: number
  byteSize: number
}

export interface GlobalLogCatalogPage {
  items: GlobalLogFile[]
  summary: GlobalLogSummary
  total: number
  offset: number
  limit: number
  durationMs: number
}

export interface GlobalLogReadRequest {
  id: string
  direction: 'head' | 'tail' | 'next' | string
  cursor: number
  startLine: number
  limit: number
  query: string
  level: string
  category: string
}

export interface GlobalLogLine {
  lineNumber: number
  text: string
  level: string
  category: string
  timestamp: string
}

export interface GlobalLogReadPage {
  file: GlobalLogFile
  lines: GlobalLogLine[]
  nextCursor: number
  nextLine: number
  eof: boolean
  scanned: number
  matched: number
  durationMs: number
}

export interface GlobalLogExportResult {
  path: string
  name: string
  size: number
}

export interface GlobalLogMutationResult {
  deleted: number
  skipped: number
  bytesFreed: number
  deletedIds: string[]
  skippedIds: string[]
  errorDetails: string[]
}

export interface GlobalLogDeleteFilteredRequest {
  filter: GlobalLogCatalogRequest
}

export interface DSTLogSession {
  id: string
  clusterName: string
  clusterPath: string
  shardName: string
  status: string
  pid: number
  worldReady: boolean
  startedAt: number
  endedAt: number
  exitCode?: number | null
  error: string
  lineCount: number
  byteSize: number
  droppedLines: number
  persistenceError: string
}

export interface DSTLogListRequest {
  clusterPath: string
  shardName: string
  limit: number
}

export interface DSTPersistentLogLine {
  lineNumber: number
  text: string
  level: 'info' | 'warning' | 'error' | string
  category: 'general' | 'steam' | 'mod' | 'network' | 'security' | 'world' | string
}

export interface DSTLogReadRequest {
  sessionId: string
  cursor: number
  startLine: number
  limit: number
}

export interface DSTLogTailRequest {
  sessionId: string
  limit: number
}

export interface DSTLogReadPage {
  session: DSTLogSession
  lines: DSTPersistentLogLine[]
  nextCursor: number
  nextLine: number
  eof: boolean
}

export interface DSTLogSearchRequest {
  sessionId: string
  query: string
  level: string
  category: string
  limit: number
}

export interface DSTLogSearchResult {
  session: DSTLogSession
  matches: DSTPersistentLogLine[]
  scanned: number
  truncated: boolean
  durationMs: number
}

export interface DSTLogDiagnosticIssue {
  code: string
  severity: 'warning' | 'error' | string
  title: string
  detail: string
  count: number
  suggestions: string[]
  evidence: string[]
  certain: boolean
}

export interface DSTLogDiagnostics {
  session: DSTLogSession
  errorCount: number
  warningCount: number
  steamReady: boolean
  registered: boolean
  tokenLoaded: boolean
  networkReady: boolean
  worldReady: boolean
  healthy: boolean
  issues: DSTLogDiagnosticIssue[]
  scannedLines: number
  durationMs: number
}


export interface DSTLogBundleRequest {
  clusterPath: string
  shardNames: string[]
}
export interface DSTLogExportResult {
  path: string
  name: string
  size: number
}

export interface DSTEnvironment {
  documentsDir: string
  kleiRoot: string
  wegameKleiRoot: string
  userId: string
  wegameUserId: string
  clientConfig: string
  clusters: DSTCluster[]
}

declare global {
  interface Window {
    go?: {
      wailsbridge?: {
        App?: {
          CheckForUpdates: (token: string, force: boolean) => Promise<UpdateStatus>
          InstallLatestUpdate: (token: string) => Promise<PreparedUpdate>
          GetLicenseStatus: (token: string) => Promise<LicenseStatus>
          ActivateLicense: (token: string, request: LicenseActivateRequest) => Promise<LicenseStatus>
          UnbindLicense: (token: string) => Promise<LicenseStatus>
          GetLicenseFeatureStatus: (token: string, feature: string) => Promise<LicenseFeatureStatus>
          GetXiaoYuRuntimeStatus: (token: string) => Promise<XiaoYuRuntimeStatus>
          GetXiaoYuTools: (token: string) => Promise<XiaoYuToolSpec[]>
          GetXiaoYuCapabilities: (token: string) => Promise<XiaoYuCapabilitySnapshot>
          GetXiaoYuHarnessStatus: (token: string) => Promise<XiaoYuHarnessSnapshot>
          GetXiaoYuModelCatalog: (token: string) => Promise<XiaoYuModelCatalog>
          SaveXiaoYuModel: (token: string, request: XiaoYuSaveModelRequest) => Promise<XiaoYuModelProfile>
          DeleteXiaoYuModel: (token: string, id: string) => Promise<void>
          SetXiaoYuDefaultModel: (token: string, id: string) => Promise<XiaoYuModelProfile>
          TestXiaoYuModel: (token: string, request: XiaoYuModelConnectionRequest) => Promise<XiaoYuModelConnectionResult>
          DiscoverXiaoYuModels: (token: string, request: XiaoYuModelConnectionRequest) => Promise<XiaoYuModelConnectionResult>
          GetXiaoYuIntelligenceCatalog: (token: string) => Promise<XiaoYuIntelligenceCatalog>
          SaveXiaoYuMemory: (token: string, request: XiaoYuMemorySaveRequest) => Promise<XiaoYuMemoryRecord>
          SaveXiaoYuSkill: (token: string, request: XiaoYuSkillSaveRequest) => Promise<XiaoYuSkillDefinition>
          SaveXiaoYuExpert: (token: string, request: XiaoYuExpertSaveRequest) => Promise<XiaoYuExpertDefinition>
          DeleteXiaoYuIntelligence: (token: string, kind: 'memory' | 'skill' | 'expert', id: string) => Promise<void>
          StartXiaoYuRun: (token: string, request: XiaoYuRunRequest) => Promise<XiaoYuRunState>
          ContinueXiaoYuRun: (token: string, id: string, request: XiaoYuContinueRunRequest) => Promise<XiaoYuRunState>
          GetXiaoYuRunState: (token: string, id: string) => Promise<XiaoYuRunState>
          CancelXiaoYuRun: (token: string, id: string) => Promise<XiaoYuRunState>
          PauseXiaoYuRun: (token: string, id: string, request: XiaoYuControlRequest) => Promise<XiaoYuRunState>
          TakeoverXiaoYuRun: (token: string, id: string, request: XiaoYuControlRequest) => Promise<XiaoYuRunState>
          ResumeXiaoYuRun: (token: string, id: string) => Promise<XiaoYuRunState>
          GetXiaoYuTrace: (token: string, after: number, limit: number) => Promise<XiaoYuTraceEvent[]>
          GetXiaoYuDSHPlugins: (token: string) => Promise<XiaoYuDSHBundle[]>
          MountXiaoYuDSHPlugin: (token: string, request: XiaoYuDSHMountRequest) => Promise<XiaoYuPluginSnapshot>
          UnmountXiaoYuPlugin: (token: string, id: string) => Promise<void>
          GetXiaoYuApprovalState: (token: string) => Promise<XiaoYuApprovalState>
          SetXiaoYuApprovalMode: (token: string, request: { mode: string }) => Promise<XiaoYuApprovalState>
          GetXiaoYuPendingApprovals: (token: string) => Promise<XiaoYuApprovalRequest[]>
          ResolveXiaoYuApproval: (token: string, id: string, request: { decision: 'approve' | 'reject' }) => Promise<XiaoYuApprovalRequest>
          CallXiaoYuTool: (token: string, request: XiaoYuToolCallRequest) => Promise<XiaoYuToolCallResult>
          RunXiaoYuCommand: (token: string, request: XiaoYuCommandRequest) => Promise<XiaoYuCommandResult>
          GetAuthBootstrapStatus: () => Promise<AuthBootstrapStatus>
          GenerateSecurityKey: () => Promise<AuthSecurityKeyMaterial>
          CreateInitialAdministrator: (request: AuthCreateOwnerRequest) => Promise<AuthSession>
          Login: (request: AuthLoginRequest) => Promise<AuthSession>
          ValidateSession: (token: string) => Promise<AuthUser>
          Logout: (token: string) => Promise<void>
          ListUsers: (token: string) => Promise<AuthUser[]>
          UpdateMyDisplayName: (token: string, request: AuthUpdateDisplayNameRequest) => Promise<AuthUser>
          GetAuthInstanceIdentity: () => Promise<AuthInstanceIdentity>
          GetAuthOrganization: (token: string) => Promise<AuthOrganization>
          InspectUserInvitation: (token: string) => Promise<AuthInvitationPreview>
          RegisterInvitedUser: (request: AuthRegisterInvitationRequest) => Promise<AuthSession>
          CreateUserInvitation: (token: string, request: AuthCreateInvitationRequest) => Promise<AuthCreatedInvitation>
          ListUserInvitations: (token: string) => Promise<AuthInvitationView[]>
          RevokeUserInvitation: (token: string, id: string) => Promise<AuthInvitationView>
          GetMyEmailSecurityStatus: (token: string) => Promise<AuthEmailSecurityStatus>
          BindMyEmail: (token: string, request: AuthBindEmailRequest) => Promise<AuthEmailSecurityStatus>
          UnbindMyEmail: (token: string, password: string) => Promise<AuthEmailSecurityStatus>
          RequestMyEmailVerification: (token: string, request: AuthRequestEmailVerificationRequest) => Promise<void>
          ConfirmMyEmailVerification: (token: string, request: AuthConfirmEmailVerificationRequest) => Promise<AuthEmailSecurityStatus>
          ConfirmMyCredentialStepUp: (token: string, request: AuthCredentialStepUpRequest) => Promise<AuthUser>
          RequestPasswordReset: (request: AuthRequestPasswordResetRequest) => Promise<AuthPasswordResetRequestStatus>
          ConfirmPasswordReset: (request: AuthConfirmPasswordResetRequest) => Promise<void>
          GetSMTPSettings: (token: string) => Promise<AuthSMTPSettings>
          SaveSMTPSettings: (token: string, request: AuthSaveSMTPSettingsRequest) => Promise<AuthSMTPSettings>
          CreateMemberCoreAuthorization: (token: string, request: AuthCreateMemberAuthorizationRequest) => Promise<AuthCreatedMemberAuthorization>
          ListMemberCoreAuthorizations: (token: string) => Promise<AuthMemberAuthorizationView[]>
          RevokeMemberCoreAuthorization: (token: string, id: string) => Promise<AuthMemberAuthorizationView>
          RedeemMyCoreAuthorization: (token: string, request: AuthRedeemMemberAuthorizationRequest) => Promise<AuthUser>
          RevokeMemberCoreAccess: (token: string, userId: string) => Promise<AuthUser>
          ClearMemberRisk: (token: string, userId: string) => Promise<AuthUser>
          RemoveOrganizationMember: (token: string, userId: string) => Promise<void>
          GetMySecurityStatus: (token: string) => Promise<AuthSecurityStatus>
          RotateMySecurityKey: (token: string, request: AuthRotateSecurityKeyRequest) => Promise<AuthSecurityKeyMaterial>
          SetMySecurityKeyVerification: (token: string, request: AuthSetSecurityKeyVerificationRequest) => Promise<AuthSecurityStatus>
          GetEnvironmentSetupStatus: (token: string) => Promise<EnvironmentSetupStatus>
          InitializeEnvironment: (token: string, request: EnvironmentInitializeRequest) => Promise<EnvironmentSetupStatus>
          SkipEnvironmentSetup: (token: string) => Promise<EnvironmentSetupStatus>
          InstallSteamCMD: (token: string) => Promise<EnvironmentSetupStatus>
          InstallSteamCMDAt: (token: string, request: SteamCMDInstallRequest) => Promise<EnvironmentSetupStatus>
          UpdateEnvironmentPaths: (token: string, request: EnvironmentStoragePathsRequest) => Promise<EnvironmentSetupStatus>
          MigrateSteamCMD: (token: string, request: EnvironmentMigratePathRequest) => Promise<EnvironmentMigrationResult>
          MigrateGameLibrary: (token: string, request: EnvironmentMigratePathRequest) => Promise<EnvironmentMigrationResult>
          GetEnvironmentRuntimeCatalog: (token: string) => Promise<EnvironmentRuntimeCatalog>
          InstallJavaRuntime: (token: string, request: EnvironmentJavaInstallRequest) => Promise<EnvironmentRuntimeRecord>
          RegisterEnvironmentRuntime: (token: string, request: EnvironmentRegisterRuntimeRequest) => Promise<EnvironmentRuntimeRecord>
          ResolveEnvironmentRuntime: (token: string, request: EnvironmentResolveRuntimeRequest) => Promise<EnvironmentRuntimeRecord>
          SetDefaultEnvironmentRuntime: (token: string, request: EnvironmentSetDefaultRuntimeRequest) => Promise<EnvironmentRuntimeRecord>
          RemoveEnvironmentRuntime: (token: string, request: { id: string }) => Promise<void>
          GetEnvironmentGameRuntimeProfiles: (token: string) => Promise<EnvironmentGameRuntimeProfile[]>
          GetEnvironmentGameRuntimeProfile: (token: string, request: EnvironmentGameRuntimeProfileRequest) => Promise<EnvironmentGameRuntimeProfile>
          InstallEnvironmentSystemPrerequisite: (token: string, request: EnvironmentInstallSystemPrerequisiteRequest) => Promise<EnvironmentSystemPrerequisite>
          SelectDirectory: (token: string, title: string, initial: string) => Promise<string>
          Ping: () => Promise<string>
          GetAppInfo: (token: string) => Promise<AppInfo>
          GetPlatformConfig: (token: string) => Promise<PlatformConfig>
          GetPlatformRuntimeContract: (token: string) => Promise<PlatformRuntimeContract>
          GetGamePacks: (token: string) => Promise<GamePack[]>
          GetGameInstances: (token: string) => Promise<GameInstance[]>
          GetMinecraftPlan: (token: string, request: MinecraftPlanRequest) => Promise<MinecraftPlan>
          DeployMinecraft: (token: string, request: MinecraftPlanRequest) => Promise<MinecraftDeploymentResult>
          StartMinecraft: (token: string, id: string) => Promise<MinecraftRuntimeSnapshot>
          StopMinecraft: (token: string, id: string) => Promise<MinecraftRuntimeSnapshot>
          GetMinecraftStatus: (token: string, id: string) => Promise<MinecraftRuntimeSnapshot>
          GetMinecraftLogs: (token: string, id: string, after: number, limit: number) => Promise<MinecraftLogBatch>
          ProbeMinecraft: (token: string, id: string) => Promise<MinecraftProbeResult>
          GetSettings: (token: string) => Promise<AGMPSettings>
          SaveSettings: (token: string, value: AGMPSettings) => Promise<void>
          GetRecentLogs: (token: string, limit: number) => Promise<string[]>
          GetGlobalLogCatalog: (token: string, request: GlobalLogCatalogRequest) => Promise<GlobalLogCatalogPage>
          ReadGlobalLog: (token: string, request: GlobalLogReadRequest) => Promise<GlobalLogReadPage>
          ExportGlobalLog: (token: string, id: string) => Promise<GlobalLogExportResult>
          ExportGlobalLogs: (token: string, request: GlobalLogCatalogRequest) => Promise<GlobalLogExportResult>
          DeleteGlobalLog: (token: string, id: string) => Promise<GlobalLogMutationResult>
          DeleteGlobalLogs: (token: string, request: GlobalLogDeleteFilteredRequest) => Promise<GlobalLogMutationResult>
          ClearGlobalLogHistory: (token: string) => Promise<GlobalLogMutationResult>
          OpenGlobalLogFolder: (token: string) => Promise<void>
          CurrentTime: (token: string) => Promise<string>
          GetSteamEnvironment: (token: string) => Promise<SteamEnvironment>
          GetSteamApps: (token: string) => Promise<SteamAppInventory>
          GetSteamSnapshot: (token: string) => Promise<SteamSnapshot>
          StartSteamInstall: (token: string, request: SteamMaintenanceInstallRequest) => Promise<SteamMaintenanceTask>
          StartSteamValidation: (token: string, request: SteamMaintenanceValidateRequest) => Promise<SteamMaintenanceTask>
          GetSteamMaintenanceTask: (token: string, id: string) => Promise<SteamMaintenanceTask>
          GetGameWorkspaceSnapshot: (token: string, id: string) => Promise<GameWorkspaceSnapshot>
          GetDSTEnvironment: (token: string) => Promise<DSTEnvironment>
          GetDSTWorkspaceSnapshot: (token: string) => Promise<DSTWorkspaceSnapshot>
          GetDSTDedicatedServer: (token: string) => Promise<DSTDedicatedSnapshot>
          GetDSTDedicatedPreferences: (token: string) => Promise<DSTDedicatedPreferences>
          SaveDSTDedicatedPreferences: (token: string, value: DSTDedicatedPreferences) => Promise<void>
          OpenDSTTokenPage: (token: string) => Promise<void>
          GetDSTTokenStatus: (token: string, clusterPath: string) => Promise<DSTTokenStatus>
          SaveDSTToken: (token: string, request: DSTTokenSaveRequest) => Promise<DSTTokenStatus>
          ImportDSTToken: (token: string, request: DSTTokenImportRequest) => Promise<DSTTokenStatus>
          InspectDSTKleiPackage: (token: string, request: DSTKleiPackageInspectRequest) => Promise<DSTKleiPackagePreview>
          ImportDSTKleiPackage: (token: string, request: DSTKleiPackageImportRequest) => Promise<DSTKleiPackageImportResult>
          GetDSTPreflight: (token: string, clusterPath: string) => Promise<DSTPreflightResult>
          ImportDSTCluster: (token: string, request: DSTClusterImportRequest) => Promise<DSTClusterImportResult>
          StartDSTMaster: (token: string, request: DSTStartMasterRequest) => Promise<DSTProcessSnapshot>
          StartDSTCluster: (token: string, request: DSTStartClusterRequest) => Promise<DSTClusterRuntimeSnapshot>
          StartDSTShard: (token: string, request: DSTStartShardRequest) => Promise<DSTProcessSnapshot>
          GetDSTClusterStatus: (token: string, request: DSTClusterRequest) => Promise<DSTClusterRuntimeSnapshot>
          GetDSTPortStatus: (token: string, request: DSTClusterRequest) => Promise<DSTPortReport>
          GetDSTPortConfiguration: (token: string, request: DSTClusterRequest) => Promise<DSTPortConfiguration>
          ConfigureDSTPorts: (token: string, request: DSTPortConfigureRequest) => Promise<DSTPortConfigureResult>
          CleanupDSTPorts: (token: string, request: DSTPortCleanupRequest) => Promise<DSTPortCleanupResult>
          StopDSTCluster: (token: string, request: DSTClusterRequest) => Promise<DSTClusterRuntimeSnapshot>
          GetDSTProcessStatus: (token: string, request: DSTProcessRequest) => Promise<DSTProcessLookup>
          ReadDSTProcessLogs: (token: string, request: DSTProcessLogRequest) => Promise<DSTProcessLogBatch>
          SendDSTCommand: (token: string, request: DSTCommandRequest) => Promise<DSTProcessSnapshot>
          StopDSTProcess: (token: string, request: DSTProcessRequest) => Promise<DSTProcessSnapshot>
          GetDSTLogSessions: (token: string, request: DSTLogListRequest) => Promise<DSTLogSession[]>
          ReadDSTLog: (token: string, request: DSTLogReadRequest) => Promise<DSTLogReadPage>
          TailDSTLog: (token: string, request: DSTLogTailRequest) => Promise<DSTLogReadPage>
          SearchDSTLog: (token: string, request: DSTLogSearchRequest) => Promise<DSTLogSearchResult>
          GetDSTLogDiagnostics: (token: string, id: string) => Promise<DSTLogDiagnostics>
          ExportDSTLog: (token: string, id: string) => Promise<DSTLogExportResult>
          ExportDSTLogBundle: (token: string, request: DSTLogBundleRequest) => Promise<DSTLogExportResult>
          BuildDSTLaunchSpec: (token: string, request: DSTLaunchRequest) => Promise<DSTLaunchSpec>
        }
      }
    }
  }
}

export {}
