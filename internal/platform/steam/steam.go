package steam

import "context"

// AppID is a Steam application identifier.
type AppID uint32

const (
	AppStateUpdateRequired uint64 = 2
	AppStateFullyInstalled uint64 = 4
	AppStateUpdateRunning  uint64 = 256
	AppStateUpdatePaused   uint64 = 512
	AppStateUpdateStarted  uint64 = 1024
	AppStateReconfiguring  uint64 = 65536
	AppStateValidating     uint64 = 131072
	AppStateAddingFiles    uint64 = 262144
	AppStatePreallocating  uint64 = 524288
	AppStateDownloading    uint64 = 1048576
	AppStateStaging        uint64 = 2097152
	AppStateCommitting     uint64 = 4194304
	AppStateUpdateStopping uint64 = 8388608
)

const appStateBusyMask = AppStateUpdateRunning | AppStateUpdatePaused | AppStateUpdateStarted | AppStateReconfiguring | AppStateValidating | AppStateAddingFiles | AppStatePreallocating | AppStateDownloading | AppStateStaging | AppStateCommitting | AppStateUpdateStopping

// Library describes one Steam library root.
type Library struct {
	Path          string `json:"path"`
	SteamAppsPath string `json:"steamAppsPath"`
	Primary       bool   `json:"primary"`
}

// Environment is the Steam-wide state discovered on the current node.
// Missing Steam is a normal, non-error state: Detected will simply be false.
type Environment struct {
	Detected       bool      `json:"detected"`
	InstallPath    string    `json:"installPath"`
	ExecutablePath string    `json:"executablePath"`
	Libraries      []Library `json:"libraries"`
	Warnings       []string  `json:"warnings"`
}

// AppInstallation is generic Steam installation information shared by every
// Steam-hosted dedicated server. It intentionally contains no game-specific data.
type AppInstallation struct {
	AppID             AppID  `json:"appId"`
	Name              string `json:"name"`
	InstallDir        string `json:"installDir"`
	InstallPath       string `json:"installPath"`
	InstallPathExists bool   `json:"installPathExists"`
	LibraryPath       string `json:"libraryPath"`
	ManifestPath      string `json:"manifestPath"`
	BuildID           string `json:"buildId"`
	LastUpdated       int64  `json:"lastUpdated"`
	SizeOnDisk        uint64 `json:"sizeOnDisk"`
	StateFlags        uint64 `json:"stateFlags"`
	BytesDownloaded   uint64 `json:"bytesDownloaded"`
	BytesToDownload   uint64 `json:"bytesToDownload"`
	BytesStaged       uint64 `json:"bytesStaged"`
	BytesToStage      uint64 `json:"bytesToStage"`
	TargetBuildID     string `json:"targetBuildId"`
	Branch            string `json:"branch"`
}

func (a AppInstallation) UpdatePending() bool {
	if a.StateFlags&AppStateUpdateRequired != 0 || a.StateFlags&appStateBusyMask != 0 {
		return true
	}
	if a.TargetBuildID != "" && a.TargetBuildID != "0" && a.BuildID != "" && a.TargetBuildID != a.BuildID {
		return true
	}
	if a.BytesToDownload > 0 && a.BytesDownloaded < a.BytesToDownload {
		return true
	}
	if a.BytesToStage > 0 && a.BytesStaged < a.BytesToStage {
		return true
	}
	return false
}

func (a AppInstallation) ValidationActive() bool { return a.StateFlags&AppStateValidating != 0 }
func (a AppInstallation) DownloadActive() bool   { return a.StateFlags&AppStateDownloading != 0 }
func (a AppInstallation) StagingActive() bool    { return a.StateFlags&AppStateStaging != 0 }
func (a AppInstallation) CommittingActive() bool { return a.StateFlags&AppStateCommitting != 0 }
func (a AppInstallation) Busy() bool             { return a.StateFlags&appStateBusyMask != 0 }
func (a AppInstallation) FullyInstalled() bool   { return a.StateFlags&AppStateFullyInstalled != 0 }

// AppInventory is a fault-tolerant snapshot of every appmanifest_*.acf found
// across all detected Steam libraries. A broken manifest is reported through
// Warnings and does not prevent other apps from being returned.
type AppInventory struct {
	Detected bool              `json:"detected"`
	Apps     []AppInstallation `json:"apps"`
	Warnings []string          `json:"warnings"`
}

// Snapshot is one atomic Steam refresh result. Environment and app inventory
// are produced from the same discovery pass so the UI never needs to issue
// duplicate Steam scans or briefly render mismatched data.
type Snapshot struct {
	Environment Environment  `json:"environment"`
	Inventory   AppInventory `json:"inventory"`
	ScannedAt   int64        `json:"scannedAt"`
	DurationMs  int64        `json:"durationMs"`
}

// TargetedSnapshot checks only the requested AppIDs. It is used by the user-facing
// game library so AI Game Manager Panel does not enumerate every Steam application just to
// determine whether supported games are installed.
type TargetedSnapshot struct {
	Environment Environment       `json:"environment"`
	Apps        []AppInstallation `json:"apps"`
	Warnings    []string          `json:"warnings"`
	ScannedAt   int64             `json:"scannedAt"`
	DurationMs  int64             `json:"durationMs"`
}

// Platform defines Steam-wide operations only. It must never contain game-specific rules.
type Platform interface {
	Detect(ctx context.Context) (Environment, error)
	Libraries(ctx context.Context) ([]Library, error)
	Apps(ctx context.Context) (AppInventory, error)
	Snapshot(ctx context.Context) (Snapshot, error)
	FindApps(ctx context.Context, appIDs []AppID) (TargetedSnapshot, error)
	FindApp(ctx context.Context, appID AppID) (AppInstallation, bool, error)
}
