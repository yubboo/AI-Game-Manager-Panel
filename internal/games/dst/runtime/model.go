package runtime

import (
	"time"

	"github.com/yubboo/AI-Game-Manager-Panel/internal/games/dst/dedicated"
)

const (
	RecentLogLimit = 500
	StreamLogLimit = 3000
)

type ProcessSnapshot struct {
	ClusterName         string                 `json:"clusterName"`
	ClusterPath         string                 `json:"clusterPath"`
	ShardName           string                 `json:"shardName"`
	Role                dedicated.ShardRole    `json:"role"`
	Status              dedicated.ServerStatus `json:"status"`
	PID                 int                    `json:"pid"`
	WorldReady          bool                   `json:"worldReady"`
	IntentionalShutdown bool                   `json:"intentionalShutdown"`
	ExitCode            *int                   `json:"exitCode"`
	StartedAt           int64                  `json:"startedAt"`
	UpdatedAt           int64                  `json:"updatedAt"`
	Error               string                 `json:"error"`
	LogCursor           uint64                 `json:"logCursor"`
	LogSessionID        string                 `json:"logSessionId"`
	LogPersistenceError string                 `json:"logPersistenceError"`
}

type LookupResult struct {
	Found   bool            `json:"found"`
	Process ProcessSnapshot `json:"process"`
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

type StartRequest struct {
	ClusterName string               `json:"clusterName"`
	ClusterPath string               `json:"clusterPath"`
	ShardName   string               `json:"shardName"`
	Spec        dedicated.LaunchSpec `json:"spec"`
}

type StopOptions struct {
	GracefulTimeout time.Duration
	TermTimeout     time.Duration
}

func DefaultStopOptions() StopOptions {
	return StopOptions{
		GracefulTimeout: 30 * time.Second,
		TermTimeout:     5 * time.Second,
	}
}

// LogSessionSink receives runtime lines without coupling the runtime package to
// a concrete persistence implementation. Implementations must keep Append fast
// and non-blocking so a slow disk can never stall the dedicated server stdout pipe.
type LogSessionSink interface {
	ID() string
	Append(LogLine)
	Close(ProcessSnapshot)
}

type LogSessionFactory interface {
	StartSession(StartRequest) (LogSessionSink, error)
}
