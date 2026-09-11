package maintenance

import (
	"context"
	"sync"

	"github.com/yubboo/AI-Game-Manager-Panel/internal/platform/steam"
)

const (
	OperationValidate = "validate"
	OperationInstall  = "install"

	StateRequested  = "requested"
	StateMonitoring = "monitoring"
	StateCompleted  = "completed"
	StateFailed     = "failed"
	StateTimeout    = "timeout"

	PhaseWaiting     = "waiting"
	PhaseQueued      = "queued"
	PhaseValidating  = "validating"
	PhaseDownloading = "downloading"
	PhaseStaging     = "staging"
	PhaseCommitting  = "committing"
	PhaseFinalizing  = "finalizing"
	PhaseCompleted   = "completed"

	ProgressDeterminate   = "determinate"
	ProgressIndeterminate = "indeterminate"

	// Steam does not expose a public per-AppID validation progress API. On
	// clients that do not emit a prompt Validating StateFlag/log line, two
	// consecutive real Steam process reads of at least 1 MiB are used only as
	// a START fallback. Completion still requires this task's content_log
	// terminal event, so I/O can never promote a task to 100%.
	validationIOStartMinDelta = uint64(1 << 20)
	validationIOStartHits     = 2
)

type ValidateRequest struct {
	AppID uint32 `json:"appId"`
}

type InstallRequest struct {
	AppID uint32 `json:"appId"`
}

// TaskSnapshot is the single Steam maintenance task contract shared by every
// Steam game provider. Game-specific modules must not invent their own validate
// progress/state model.
type TaskSnapshot struct {
	ID                  string `json:"id"`
	AppID               uint32 `json:"appId"`
	Operation           string `json:"operation"`
	URI                 string `json:"uri"`
	State               string `json:"state"`
	Phase               string `json:"phase"`
	Message             string `json:"message"`
	StartedAt           int64  `json:"startedAt"`
	UpdatedAt           int64  `json:"updatedAt"`
	BuildID             string `json:"buildId"`
	Progress            int    `json:"progress"`
	ProgressMode        string `json:"progressMode"`
	ProgressSource      string `json:"progressSource"`
	ProgressEstimated   bool   `json:"progressEstimated"`
	BytesDone           uint64 `json:"bytesDone"`
	BytesTotal          uint64 `json:"bytesTotal"`
	SteamRoot           string `json:"steamRoot"`
	LibraryPath         string `json:"libraryPath"`
	InstallPath         string `json:"installPath"`
	ManifestPath        string `json:"manifestPath"`
	ContentLogPath      string `json:"contentLogPath"`
	StateFlags          uint64 `json:"stateFlags"`
	ValidationFiles     uint64 `json:"validationFiles"`
	ValidationBytes     uint64 `json:"validationBytes"`
	MismatchedFiles     uint64 `json:"mismatchedFiles"`
	MismatchedBytes     uint64 `json:"mismatchedBytes"`
	CompletionConfirmed bool   `json:"completionConfirmed"`
	CompletionSource    string `json:"completionSource"`
	Error               string `json:"error"`
}

type SteamPlatform interface {
	Detect(context.Context) (steam.Environment, error)
	FindApp(context.Context, steam.AppID) (steam.AppInstallation, bool, error)
}

type Launcher interface {
	Open(context.Context, string) error
}

type Service struct {
	steam    SteamPlatform
	launcher Launcher
	mu       sync.RWMutex
	tasks    map[string]TaskSnapshot
}
