package dedicated

const (
	DedicatedServerAppID uint32 = 343050
	InstallDirName              = "Don't Starve Together Dedicated Server"
	ExecutableAMD64             = "dontstarve_dedicated_server_nullrenderer_x64.exe"
	Executable386               = "dontstarve_dedicated_server_nullrenderer.exe"
	BinDirAMD64                 = "bin64"
	BinDir386                   = "bin"
)

type Architecture string

const (
	ArchitectureUnknown Architecture = "unknown"
	ArchitectureAMD64   Architecture = "windows-amd64"
	Architecture386     Architecture = "windows-386"
)

type InstallSource string

const (
	InstallSourceNone   InstallSource = "none"
	InstallSourceManual InstallSource = "manual"
	InstallSourceSteam  InstallSource = "steam"
)

// Installation is the validated DST Dedicated Server installation selected by
// the same priority as the Python baseline: remembered/manual path first, then
// every detected Steam library. A normal DST game client is deliberately not a
// valid dedicated-server installation even though it ships similarly named
// nullrenderer executables.
type Installation struct {
	Detected     bool          `json:"detected"`
	Valid        bool          `json:"valid"`
	AppID        uint32        `json:"appId"`
	RootDir      string        `json:"rootDir"`
	BinDir       string        `json:"binDir"`
	Executable   string        `json:"executable"`
	Architecture Architecture  `json:"architecture"`
	Bitness      int           `json:"bitness"`
	Source       InstallSource `json:"source"`
	LibraryPath  string        `json:"libraryPath"`
	Issues       []string      `json:"issues"`
}

type ConfDirInfo struct {
	DocumentsDir string `json:"documentsDir"`
	BaseDir      string `json:"baseDir"`
	KleiRoot     string `json:"kleiRoot"`
	Argument     string `json:"argument"`
	Default      bool   `json:"default"`
	Valid        bool   `json:"valid"`
	Error        string `json:"error"`
}

type ServerStatus string

const (
	StatusStarting ServerStatus = "starting"
	StatusRunning  ServerStatus = "running"
	StatusStopping ServerStatus = "stopping"
	StatusStopped  ServerStatus = "stopped"
	StatusCrashed  ServerStatus = "crashed"
)

type ShardRole string

const (
	ShardMaster    ShardRole = "Master"
	ShardSecondary ShardRole = "Secondary"
)

type LaunchRequest struct {
	ClusterName  string `json:"clusterName"`
	ShardName    string `json:"shardName"`
	KleiRoot     string `json:"kleiRoot"`
	UGCDirectory string `json:"ugcDirectory"`
	ExtraArgs    string `json:"extraArgs"`
}

type LaunchSpec struct {
	ClusterName      string       `json:"clusterName"`
	ShardName        string       `json:"shardName"`
	Role             ShardRole    `json:"role"`
	Executable       string       `json:"executable"`
	WorkingDirectory string       `json:"workingDirectory"`
	Arguments        []string     `json:"arguments"`
	ConfDirArgument  string       `json:"confDirArgument"`
	UGCDirectory     string       `json:"ugcDirectory"`
	ExtraArgs        string       `json:"extraArgs"`
	Architecture     Architecture `json:"architecture"`
}

type Snapshot struct {
	Installation Installation `json:"installation"`
	ConfDir      ConfDirInfo  `json:"confDir"`
	ManualPath   string       `json:"manualPath"`
	ExtraArgs    string       `json:"extraArgs"`
	Warnings     []string     `json:"warnings"`
}
