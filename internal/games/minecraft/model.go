package minecraft

import serverinstance "github.com/yubboo/AI-Game-Manager-Panel/internal/server/instance"

const GameID = "minecraft.java"

type Software string

const (
	SoftwareVanilla Software = "vanilla"
	SoftwarePaper   Software = "paper"
	SoftwareFabric  Software = "fabric"
)

type Artifact struct {
	Software      Software `json:"software"`
	URL           string   `json:"url"`
	FileName      string   `json:"fileName"`
	HashAlgorithm string   `json:"hashAlgorithm,omitempty"`
	Hash          string   `json:"hash,omitempty"`
	Size          int64    `json:"size,omitempty"`
	Build         string   `json:"build,omitempty"`
	Loader        string   `json:"loader,omitempty"`
	Installer     string   `json:"installer,omitempty"`
	Trust         string   `json:"trust"`
}

type VersionFacts struct {
	RequestedVersion string   `json:"requestedVersion,omitempty"`
	Version          string   `json:"version"`
	LatestRelease    string   `json:"latestRelease"`
	JavaMajor        int      `json:"javaMajor"`
	Artifact         Artifact `json:"artifact"`
	Sources          []string `json:"sources"`
}

type PlanRequest struct {
	Name             string                `json:"name"`
	Version          string                `json:"version,omitempty"`
	Software         Software              `json:"software"`
	MemoryMB         int                   `json:"memoryMb"`
	Port             int                   `json:"port"`
	OnlineMode       *bool                 `json:"onlineMode"`
	Whitelist        bool                  `json:"whitelist"`
	EULAAccepted     bool                  `json:"eulaAccepted"`
	AutoInstallJava  bool                  `json:"autoInstallJava"`
	StartAfterDeploy bool                  `json:"startAfterDeploy"`
	Origin           serverinstance.Origin `json:"origin,omitempty"`
}

type Plan struct {
	ID               string                `json:"id"`
	GameID           string                `json:"gameId"`
	Name             string                `json:"name"`
	Origin           serverinstance.Origin `json:"origin"`
	InstallPath      string                `json:"installPath"`
	VersionFacts     VersionFacts          `json:"versionFacts"`
	MemoryMB         int                   `json:"memoryMb"`
	Port             int                   `json:"port"`
	OnlineMode       bool                  `json:"onlineMode"`
	Whitelist        bool                  `json:"whitelist"`
	EULAAccepted     bool                  `json:"eulaAccepted"`
	AutoInstallJava  bool                  `json:"autoInstallJava"`
	StartAfterDeploy bool                  `json:"startAfterDeploy"`
	Steps            []string              `json:"steps"`
}

type DeploymentResult struct {
	Plan     Plan                    `json:"plan"`
	Instance serverinstance.Instance `json:"instance"`
	Runtime  RuntimeSnapshot         `json:"runtime"`
	Probe    *ProbeResult            `json:"probe,omitempty"`
}

type RuntimeSnapshot struct {
	InstanceID string `json:"instanceId"`
	State      string `json:"state"`
	PID        int    `json:"pid,omitempty"`
	Ready      bool   `json:"ready"`
	StartedAt  int64  `json:"startedAt,omitempty"`
	UpdatedAt  int64  `json:"updatedAt,omitempty"`
	ExitCode   *int   `json:"exitCode,omitempty"`
	Error      string `json:"error,omitempty"`
	LogCursor  uint64 `json:"logCursor"`
}

type LogLine struct {
	Sequence  uint64 `json:"sequence"`
	Timestamp int64  `json:"timestamp"`
	Text      string `json:"text"`
}
type LogBatch struct {
	Lines      []LogLine `json:"lines"`
	NextCursor uint64    `json:"nextCursor"`
	Dropped    bool      `json:"dropped"`
}

type ProbeResult struct {
	Online        bool   `json:"online"`
	Version       string `json:"version,omitempty"`
	Protocol      int    `json:"protocol,omitempty"`
	PlayersOnline int    `json:"playersOnline,omitempty"`
	PlayersMax    int    `json:"playersMax,omitempty"`
	Description   string `json:"description,omitempty"`
	LatencyMs     int64  `json:"latencyMs"`
}

type manifest struct {
	Version        int      `json:"version"`
	InstanceID     string   `json:"instanceId"`
	GameVersion    string   `json:"gameVersion"`
	Software       Software `json:"software"`
	ServerVersion  string   `json:"serverVersion,omitempty"`
	JavaMajor      int      `json:"javaMajor"`
	JavaRuntimeID  string   `json:"javaRuntimeId"`
	JavaExecutable string   `json:"javaExecutable"`
	MemoryMB       int      `json:"memoryMb"`
	Port           int      `json:"port"`
	Artifact       Artifact `json:"artifact"`
}
