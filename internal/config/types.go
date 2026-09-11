package config

// PlatformConfig 是提供给 Desktop/Web 前端的只读平台配置快照。
// 这些值来自项目根目录 configs/，用于避免导航、AI 权限模式、模块规划等散落硬编码。
type PlatformConfig struct {
	App         AppConfig         `json:"app"`
	UI          UIConfig          `json:"ui"`
	AI          AIConfig          `json:"ai"`
	Permissions PermissionsConfig `json:"permissions"`
	Paths       PathsConfig       `json:"paths"`
	Logging     LoggingConfig     `json:"logging"`
	Games       GamesConfig       `json:"games"`
	Modules     ModulesConfig     `json:"modules"`
	Server      ServerConfig      `json:"server"`
	Update      UpdateConfig      `json:"update"`
}

type AppConfig struct {
	ProductName  string `json:"productName"`
	Edition      string `json:"edition"`
	Slogan       string `json:"slogan"`
	DefaultRoute string `json:"defaultRoute"`
	Locale       string `json:"locale"`
	Telemetry    bool   `json:"telemetry"`
}

type UIConfig struct {
	Theme               string            `json:"theme"`
	Accent              string            `json:"accent"`
	SidebarWidth        int               `json:"sidebarWidth"`
	CompactSidebarWidth int               `json:"compactSidebarWidth"`
	ShowTopbar          bool              `json:"showTopbar"`
	Navigation          []NavigationGroup `json:"navigation"`
	Workbench           WorkbenchUIConfig `json:"workbench"`
}

type NavigationGroup struct {
	ID    string           `json:"id"`
	Title string           `json:"title"`
	Items []NavigationItem `json:"items"`
}

type NavigationItem struct {
	ID    string `json:"id"`
	To    string `json:"to"`
	Label string `json:"label"`
	Icon  string `json:"icon"`
}

type WorkbenchUIConfig struct {
	Title       string   `json:"title"`
	Subtitle    string   `json:"subtitle"`
	Placeholder string   `json:"placeholder"`
	Suggestions []string `json:"suggestions"`
}

type AIConfig struct {
	Enabled                  bool              `json:"enabled"`
	PrimaryInteraction       string            `json:"primaryInteraction"`
	Stream                   bool              `json:"stream"`
	ApprovalMode             string            `json:"approvalMode"`
	AllowManualFallback      bool              `json:"allowManualFallback"`
	MaxConcurrentRuns        int               `json:"maxConcurrentRuns"`
	ToolTimeoutSeconds       int               `json:"toolTimeoutSeconds"`
	ConversationHistoryLimit int               `json:"conversationHistoryLimit"`
	StoreConversationHistory bool              `json:"storeConversationHistory"`
	SecretStorage            string            `json:"secretStorage"`
	Runtime                  AIRuntimeConfig   `json:"runtime"`
	AgentLoop                AIAgentLoopConfig `json:"agentLoop"`
	Harness                  AIHarnessConfig   `json:"harness"`
	BrainConfiguration       string            `json:"brainConfiguration"`
}

type AIRuntimeConfig struct {
	Enabled        bool   `json:"enabled"`
	Engine         string `json:"engine"`
	Protocol       string `json:"protocol"`
	Transport      string `json:"transport"`
	Binary         string `json:"binary"`
	ManualFallback bool   `json:"manualFallback"`
}

type AIAgentLoopConfig struct {
	Planner           string `json:"planner"`
	Streaming         bool   `json:"streaming"`
	MaxSteps          int    `json:"maxSteps"`
	MaxToolCalls      int    `json:"maxToolCalls"`
	MaxFailures       int    `json:"maxFailures"`
	RepeatLimit       int    `json:"repeatLimit"`
	VerifyAfterAction bool   `json:"verifyAfterAction"`
}

type AIHarnessConfig struct {
	Kernel             string                   `json:"kernel"`
	EverythingAsPlugin bool                     `json:"everythingAsPlugin"`
	TraceMode          string                   `json:"traceMode"`
	DSH                AIDSHCompatibilityConfig `json:"deepseekHarness"`
}

type AIDSHCompatibilityConfig struct {
	Enabled        bool   `json:"enabled"`
	Mode           string `json:"mode"`
	AutoMount      bool   `json:"autoMount"`
	NodeExecutable string `json:"nodeExecutable"`
}

type PermissionsConfig struct {
	AdministratorRole   string           `json:"administratorRole"`
	AllowFullAccessMode bool             `json:"allowFullAccessMode"`
	AllowArbitraryShell bool             `json:"allowArbitraryShell"`
	ApprovalModes       []ApprovalMode   `json:"approvalModes"`
	Policies            ApprovalPolicies `json:"policies"`
	FileScope           []string         `json:"fileScope"`
}

// ApprovalPolicies 使用和审批模式相同的短 ID，便于人工和 AI 一眼看懂。
// ask=请求批准，risk=帮我批准，full=完全访问权限。
type ApprovalPolicies struct {
	Ask  map[string]string `json:"ask"`
	Risk map[string]string `json:"risk"`
	Full map[string]string `json:"full"`
}

type ApprovalMode struct {
	ID          string `json:"id"`
	Label       string `json:"label"`
	Description string `json:"description"`
}

type PathsConfig struct {
	DataDir     string `json:"dataDir"`
	LogDir      string `json:"logDir"`
	BackupDir   string `json:"backupDir"`
	InstanceDir string `json:"instanceDir"`
	TempDir     string `json:"tempDir"`
	ExportDir   string `json:"exportDir"`
	PluginDir   string `json:"pluginDir"`
	CacheDir    string `json:"cacheDir"`
}

type LoggingConfig struct {
	Level              string             `json:"level"`
	RetentionDays      int                `json:"retentionDays"`
	MaxFileSizeMB      int                `json:"maxFileSizeMB"`
	MaxFilesPerSource  int                `json:"maxFilesPerSource"`
	CompressRotated    bool               `json:"compressRotated"`
	FlushIntervalMS    int                `json:"flushIntervalMs"`
	OperationAudit     bool               `json:"operationAudit"`
	RedactSecrets      bool               `json:"redactSecrets"`
	ExportSubdir       string             `json:"exportSubdir"`
	CatalogPageSize    int                `json:"catalogPageSize"`
	ReadPageSize       int                `json:"readPageSize"`
	AutoRefreshSeconds int                `json:"autoRefreshSeconds"`
	Directories        LoggingDirectories `json:"directories"`
}

// LoggingDirectories 集中定义配置日志根目录下的公共目录名称，避免路径散落在业务代码。
type LoggingDirectories struct {
	Core       string `json:"core"`
	Operations string `json:"operations"`
	Audit      string `json:"audit"`
	AI         string `json:"ai"`
	Steam      string `json:"steam"`
	Games      string `json:"games"`
	Nodes      string `json:"nodes"`
}

type GamesConfig struct {
	Templates []GameTemplate `json:"templates"`
}

type GameTemplate struct {
	ID          string `json:"id"`
	Family      string `json:"family"`
	NameZh      string `json:"nameZh"`
	NameEn      string `json:"nameEn"`
	State       string `json:"state"`
	GameAppID   int    `json:"gameAppId,omitempty"`
	ServerAppID int    `json:"serverAppId,omitempty"`
}

type ModulesConfig struct {
	Modules []ModuleConfig `json:"modules"`
}

type ModuleConfig struct {
	ID         string   `json:"id"`
	Name       string   `json:"name"`
	Category   string   `json:"category"`
	Route      string   `json:"route"`
	Status     string   `json:"status"`
	Phase      string   `json:"phase"`
	Features   []string `json:"features"`
	SourceRefs []string `json:"sourceRefs,omitempty"`
}

// ServerConfig 定义 Linux/容器等无界面 Server Edition 的部署边界。
// 该配置只包含非敏感默认值，管理员口令、API Key 等秘密不得写入此文件。
type ServerConfig struct {
	Listen                   string             `json:"listen"`
	ServiceName              string             `json:"serviceName"`
	ServiceUser              string             `json:"serviceUser"`
	InstallRoot              string             `json:"installRoot"`
	ConfigRoot               string             `json:"configRoot"`
	RuntimeRoot              string             `json:"runtimeRoot"`
	LogRoot                  string             `json:"logRoot"`
	SupportedDistributions   []string           `json:"supportedDistributions"`
	Architectures            []string           `json:"architectures"`
	RunAsRoot                bool               `json:"runAsRoot"`
	EnableSystemd            bool               `json:"enableSystemd"`
	EnableOnBoot             bool               `json:"enableOnBoot"`
	AllowRemoteWeb           bool               `json:"allowRemoteWeb"`
	RequireInitialAdminSetup bool               `json:"requireInitialAdminSetup"`
	Docker                   DockerServerConfig `json:"docker"`
}

type DockerServerConfig struct {
	Enabled       bool   `json:"enabled"`
	Image         string `json:"image"`
	ContainerPort int    `json:"containerPort"`
	RunAsNonRoot  bool   `json:"runAsNonRoot"`
}

// UpdateConfig 定义公开稳定版更新源。它不包含 GitHub Token 等秘密。
type UpdateConfig struct {
	Enabled                     bool   `json:"enabled"`
	Provider                    string `json:"provider"`
	Repository                  string `json:"repository"`
	APIBaseURL                  string `json:"apiBaseUrl"`
	Channel                     string `json:"channel"`
	CheckOnStartup              bool   `json:"checkOnStartup"`
	MinimumCheckIntervalMinutes int    `json:"minimumCheckIntervalMinutes"`
	AssetPattern                string `json:"assetPattern"`
	RequireSHA256               bool   `json:"requireSha256"`
	InstallMode                 string `json:"installMode"`
	PreserveRuntimeData         bool   `json:"preserveRuntimeData"`
}
