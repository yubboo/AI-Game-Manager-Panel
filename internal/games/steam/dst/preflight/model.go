package preflight

import "time"

type Severity string

const (
	SeverityOK      Severity = "ok"
	SeverityWarning Severity = "warning"
	SeverityBlocker Severity = "blocker"
)

type Check struct {
	Code     string   `json:"code"`
	Label    string   `json:"label"`
	Severity Severity `json:"severity"`
	Message  string   `json:"message"`
	Action   string   `json:"action"`
}

type Result struct {
	ClusterPath string  `json:"clusterPath"`
	Ready       bool    `json:"ready"`
	Blockers    int     `json:"blockers"`
	Warnings    int     `json:"warnings"`
	Checks      []Check `json:"checks"`
	CheckedAt   int64   `json:"checkedAt"`
}

func NewResult(clusterPath string, checks []Check) Result {
	result := Result{ClusterPath: clusterPath, Ready: true, Checks: checks, CheckedAt: time.Now().Unix()}
	for _, check := range checks {
		switch check.Severity {
		case SeverityBlocker:
			result.Blockers++
			result.Ready = false
		case SeverityWarning:
			result.Warnings++
		}
	}
	return result
}
