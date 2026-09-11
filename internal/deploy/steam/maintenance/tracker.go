package maintenance

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/yubboo/AI-Game-Manager-Panel/internal/platform/steam"
)

type logTracker struct {
	appID                  uint32
	seq                    uint64
	sawActivity            bool
	sawValidationStarted   bool
	sawValidationIOStart   bool
	sawValidating          bool
	sawManifestValidating  bool
	sawValidationFinished  bool
	sawDownloading         bool
	sawStaging             bool
	sawCommitting          bool
	sawFullyInstalled      bool
	sawCleanFullyInstalled bool
	sawUpdateNone          bool
	sawSchedulerFinished   bool
	validationApp          uint32
	validationStartSeq     uint64
	validationFinishedSeq  uint64
	updateNoneSeq          uint64
	cleanFullyInstalledSeq uint64
	schedulerFinishedSeq   uint64
	lastBusySeq            uint64
	validationFiles        uint64
	validationBytes        uint64
	mismatchedFiles        uint64
	mismatchedBytes        uint64
}

func (t logTracker) validationSessionStarted() bool {
	return t.sawValidationStarted
}

func (t logTracker) validationScanning() bool {
	return t.sawValidationStarted && t.validationFinishedSeq <= t.validationStartSeq
}

func (t *logTracker) startValidationFromIO() {
	if t.sawValidationStarted {
		return
	}
	t.sawActivity = true
	t.sawValidationStarted = true
	t.sawValidationIOStart = true
	t.sawValidating = true
	t.validationApp = t.appID
	t.validationStartSeq = t.seq
}

func (t logTracker) validationTerminalSeen() bool {
	if !t.sawValidationStarted {
		return false
	}
	if t.schedulerFinishedSeq > t.validationStartSeq {
		return true
	}
	if t.validationFinishedSeq > t.validationStartSeq &&
		t.updateNoneSeq > t.validationFinishedSeq &&
		t.cleanFullyInstalledSeq > t.validationFinishedSeq {
		return true
	}
	return false
}

var (
	appIDLine          = regexp.MustCompile(`(?i)AppID\s+([0-9]+)(?:\s+|$)`)
	startValidating    = regexp.MustCompile(`(?i)start validating appid\s+([0-9]+)(?:\s+|$)`)
	updateStarted      = regexp.MustCompile(`(?i)update started\s*:\s*download\s+([0-9]+)/([0-9]+).*stage\s+([0-9]+)/([0-9]+)`)
	validationFinished = regexp.MustCompile(`(?i)file validation finished:\s*([0-9]+) files \(([0-9]+) bytes\) total,\s*([0-9]+) files \(([0-9]+) bytes\) mismatched`)
)

func (t *logTracker) consume(lines []string) {
	for _, line := range lines {
		t.seq++
		lower := strings.ToLower(strings.TrimSpace(line))

		if match := startValidating.FindStringSubmatch(lower); len(match) == 2 {
			if value, err := strconv.ParseUint(match[1], 10, 32); err == nil {
				t.validationApp = uint32(value)
				if t.validationApp == t.appID {
					t.sawActivity = true
					t.sawValidationStarted = true
					t.sawValidating = true
					t.validationStartSeq = t.seq
					t.lastBusySeq = t.seq
				}
			}
		}

		lineApp := uint32(0)
		if match := appIDLine.FindStringSubmatch(lower); len(match) == 2 {
			if value, err := strconv.ParseUint(match[1], 10, 32); err == nil {
				lineApp = uint32(value)
			}
		}
		targetLine := lineApp == t.appID
		if targetLine {
			if strings.Contains(lower, "validating") {
				t.sawActivity, t.sawValidationStarted, t.sawValidating = true, true, true
				t.validationApp = t.appID
				t.validationStartSeq = t.seq
				t.lastBusySeq = t.seq
			}
			if strings.Contains(lower, "downloading") {
				t.sawActivity, t.sawDownloading = true, true
				t.lastBusySeq = t.seq
			}
			if strings.Contains(lower, "staging") {
				t.sawActivity, t.sawStaging = true, true
				t.lastBusySeq = t.seq
			}
			if strings.Contains(lower, "committing") || strings.Contains(lower, "preallocating") || strings.Contains(lower, "reconfiguring") {
				t.sawActivity, t.sawCommitting = true, true
				t.lastBusySeq = t.seq
			}
			if strings.Contains(lower, "state changed") && strings.Contains(lower, "fully installed") {
				t.sawActivity, t.sawFullyInstalled = true, true
				busyState := strings.Contains(lower, "update queued") || strings.Contains(lower, "update running") || strings.Contains(lower, "update paused") || strings.Contains(lower, "update required") || strings.Contains(lower, "validating") || strings.Contains(lower, "downloading") || strings.Contains(lower, "staging") || strings.Contains(lower, "committing")
				if !busyState {
					t.sawCleanFullyInstalled = true
					t.cleanFullyInstalledSeq = t.seq
				}
			}
			if strings.Contains(lower, "update changed") && strings.Contains(lower, ": none") {
				t.sawActivity, t.sawUpdateNone = true, true
				t.updateNoneSeq = t.seq
			}
			if strings.Contains(lower, "scheduler finished") && strings.Contains(lower, "removed from schedule") {
				t.sawActivity, t.sawSchedulerFinished = true, true
				t.schedulerFinishedSeq = t.seq
			}
			if match := updateStarted.FindStringSubmatch(lower); len(match) == 5 {
				t.sawActivity = true
				t.lastBusySeq = t.seq
			}
		}

		if t.validationApp == t.appID {
			if strings.Contains(lower, "validating files") {
				t.sawActivity, t.sawValidationStarted, t.sawValidating = true, true, true
				if t.validationStartSeq == 0 || t.validationFinishedSeq >= t.validationStartSeq {
					t.validationStartSeq = t.seq
				}
				t.lastBusySeq = t.seq
			}
			if match := validationFinished.FindStringSubmatch(lower); len(match) == 5 {
				t.sawActivity = true
				t.sawValidationFinished = true
				t.validationFinishedSeq = t.seq
				t.validationFiles, _ = strconv.ParseUint(match[1], 10, 64)
				t.validationBytes, _ = strconv.ParseUint(match[2], 10, 64)
				t.mismatchedFiles, _ = strconv.ParseUint(match[3], 10, 64)
				t.mismatchedBytes, _ = strconv.ParseUint(match[4], 10, 64)
			}
		}
	}
}

func (t *logTracker) observeManifest(app steam.AppInstallation) {
	if app.ValidationActive() {
		t.sawActivity = true
		t.sawValidating = true
		t.sawManifestValidating = true
	}
	if app.DownloadActive() {
		t.sawActivity, t.sawDownloading = true, true
	}
	if app.StagingActive() {
		t.sawActivity, t.sawStaging = true, true
	}
	if app.CommittingActive() {
		t.sawActivity, t.sawCommitting = true, true
	}
}
