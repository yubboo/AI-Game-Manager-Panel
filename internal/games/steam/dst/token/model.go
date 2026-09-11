package token

import "time"

const (
	FileName           = "cluster_token.txt"
	OfficialServerPage = "https://accounts.klei.com/account/game/servers?game=DontStarveTogether"
)

type State string

const (
	StateMissing    State = "missing"
	StateEmpty      State = "empty"
	StateConfigured State = "configured"
)

type Status struct {
	State      State  `json:"state"`
	Configured bool   `json:"configured"`
	Path       string `json:"path"`
	Size       int64  `json:"size"`
	ModifiedAt int64  `json:"modifiedAt"`
	Message    string `json:"message"`
}

type SaveRequest struct {
	ClusterPath string `json:"clusterPath"`
	Value       string `json:"value"`
}

type ImportRequest struct {
	ClusterPath string `json:"clusterPath"`
	SourcePath  string `json:"sourcePath"`
}

func configuredStatus(path string, size int64, modified time.Time) Status {
	return Status{
		State:      StateConfigured,
		Configured: true,
		Path:       path,
		Size:       size,
		ModifiedAt: modified.Unix(),
		Message:    "令牌已配置；实际有效性将在 Dedicated Server 向 Klei 注册时确认",
	}
}
