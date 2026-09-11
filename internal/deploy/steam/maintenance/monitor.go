package maintenance

import (
	"context"
	"time"

	"github.com/yubboo/AI-Game-Manager-Panel/internal/platform/steam"
)

func (s *Service) monitor(id string, before steam.AppInstallation, cursor steam.ContentLogCursor, ioTracker validationIOTracker) {
	deadline := time.Now().Add(30 * time.Minute)
	tracker := logTracker{appID: uint32(before.AppID)}
	if tracker.appID == 0 {
		if task, ok := s.Get(id); ok {
			tracker.appID = task.AppID
		}
	}
	stableAfterCompletion := 0
	ioStartHits := 0
	var ioStartBytes uint64

	for {
		if time.Now().After(deadline) {
			s.update(id, func(t *TaskSnapshot) {
				t.State, t.Phase = StateTimeout, PhaseWaiting
				t.Message = "等待 Steam 完成超时。AI Game Manager Panel 没有收到可靠的 Steam 完成标记，因此不会假装校验/安装已经完成。"
				t.ProgressMode = ProgressIndeterminate
			})
			return
		}

		if lines, err := cursor.ReadNew(); err == nil {
			tracker.consume(lines)
		}

		ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
		current, found, err := s.steam.FindApp(ctx, steam.AppID(tracker.appID))
		cancel()
		if err != nil {
			s.update(id, func(t *TaskSnapshot) {
				t.State, t.Phase = StateMonitoring, PhaseWaiting
				t.Message = "Steam 正在处理；暂时无法读取 AppManifest"
				t.ProgressMode = ProgressIndeterminate
			})
			time.Sleep(time.Second)
			continue
		}

		if found {
			tracker.observeManifest(current)
		}
		operation := taskOperation(s, id)
		// AppManifest StateFlags may remain FullyInstalled (4) for the entire
		// Steam GUI verification. Once THIS task's content_log session has
		// entered validation, keep sampling Steam I/O until that same session
		// emits its validation-finished marker. Never require app.Busy() here.
		validationActive := operation == OperationValidate && (tracker.validationScanning() || (found && current.ValidationActive()))
		delta := ioTracker.observe(validationActive)

		// Some current Steam GUI builds keep AppManifest StateFlags=4 and may
		// delay/omit the target Validating start line. Because the user just
		// launched steam://validate/<AppID>, sustained real Steam disk reads are
		// accepted as a start-only fallback. They are never completion evidence.
		if operation == OperationValidate && !tracker.validationSessionStarted() {
			if delta >= validationIOStartMinDelta {
				ioStartHits++
				ioStartBytes += delta
			} else {
				ioStartHits = 0
				ioStartBytes = 0
			}
			if ioStartHits >= validationIOStartHits {
				tracker.startValidationFromIO()
				ioTracker.accumulated += ioStartBytes
				ioStartHits = 0
				ioStartBytes = 0
			}
		}
		validationProgress := ioTracker.estimate(current.SizeOnDisk)
		s.update(id, func(t *TaskSnapshot) {
			t.State = StateMonitoring
			if found {
				t.BuildID = current.BuildID
				libraryPath := current.LibraryPath
				if libraryPath != "" {
					t.LibraryPath = libraryPath
				}
				if current.InstallPath != "" {
					t.InstallPath = current.InstallPath
				}
				if current.ManifestPath != "" {
					t.ManifestPath = current.ManifestPath
				}
				t.StateFlags = current.StateFlags
			}
			t.ValidationFiles = tracker.validationFiles
			t.ValidationBytes = tracker.validationBytes
			t.MismatchedFiles = tracker.mismatchedFiles
			t.MismatchedBytes = tracker.mismatchedBytes
			applyManifestProgress(t, current, found, tracker, validationProgress)
		})

		if canComplete(operation, tracker, current, found) {
			stableAfterCompletion++
		} else {
			stableAfterCompletion = 0
		}
		if stableAfterCompletion >= 2 {
			source := "Steam AppManifest Fully Installed/idle (stable) + content_log activity"
			if operation == OperationValidate {
				source = "Steam content_log: this validation session reached a terminal event + AppManifest Fully Installed/idle (stable)"
				if tracker.schedulerFinishedSeq > tracker.validationStartSeq {
					source += " [scheduler finished]"
				} else {
					source += " [validation finished -> Update None -> Fully Installed]"
				}
			}
			s.update(id, func(t *TaskSnapshot) {
				t.State, t.Phase = StateCompleted, PhaseCompleted
				t.Message = completionMessage(t.Operation)
				t.Progress, t.ProgressMode = 100, ProgressDeterminate
				t.ProgressSource = "steam-terminal-confirmed"
				t.ProgressEstimated = false
				t.ValidationFiles = tracker.validationFiles
				t.ValidationBytes = tracker.validationBytes
				t.MismatchedFiles = tracker.mismatchedFiles
				t.MismatchedBytes = tracker.mismatchedBytes
				t.CompletionConfirmed, t.CompletionSource = true, source
				t.BytesDone, t.BytesTotal = 0, 0
			})
			return
		}

		time.Sleep(time.Second)
	}
}

func taskOperation(s *Service, id string) string {
	if task, ok := s.Get(id); ok {
		return task.Operation
	}
	return ""
}

func canComplete(operation string, tracker logTracker, app steam.AppInstallation, found bool) bool {
	installedAndStable := found && app.InstallPathExists && app.FullyInstalled() && !app.UpdatePending() && !app.Busy()
	if !installedAndStable {
		return false
	}
	switch operation {
	case OperationValidate:
		// CRITICAL: AppManifest is NOT a validation completion signal. Current
		// Steam clients can keep StateFlags=4 (Fully Installed) while the GUI is
		// still visibly validating. A validation may complete only after a
		// terminal content_log event that occurred after THIS task's start event.
		return tracker.validationTerminalSeen()
	case OperationInstall:
		return tracker.sawActivity
	default:
		return false
	}
}

func applyManifestProgress(task *TaskSnapshot, app steam.AppInstallation, found bool, tracker logTracker, validationProgress validationEstimate) {
	task.CompletionConfirmed = false
	task.CompletionSource = ""
	task.ProgressSource = "steam-content-log-phase"
	task.ProgressEstimated = false
	task.BytesDone, task.BytesTotal = 0, 0

	// For validation, the per-task content_log session is authoritative for
	// whether Steam is still scanning. Some Steam builds keep AppManifest
	// StateFlags=4 throughout validation, so manifest idle must never override
	// a live session started after this AI Game Manager Panel request.
	if task.Operation == OperationValidate && tracker.validationScanning() {
		task.Phase = PhaseValidating
		if validationProgress.Available {
			if tracker.sawValidationIOStart {
				task.Message = "Steam 正在产生连续真实文件读取；AI Game Manager Panel 已按本次校验请求进入扫描跟踪，并根据 Steam 实际磁盘读取量估算进度。完成仍必须等待本次 content_log 终态。"
			} else {
				task.Message = "Steam content_log 已确认本次校验正在进行；AI Game Manager Panel 正根据 Steam 实际磁盘读取量同步估算扫描进度。"
			}
			task.BytesDone, task.BytesTotal = validationProgress.BytesDone, validationProgress.BytesTotal
			task.Progress = validationProgress.Progress
			task.ProgressMode = ProgressDeterminate
			task.ProgressSource = "steam-windows-process-io-estimate"
			task.ProgressEstimated = true
		} else {
			if tracker.sawValidationIOStart {
				task.Message = "Steam 已出现连续真实文件读取，AI Game Manager Panel 正在等待可用的进度采样；不会使用计时器伪造进度。"
			} else {
				task.Message = "Steam content_log 已确认本次校验正在进行，正在等待 Windows Steam I/O 进度采样。"
			}
			task.ProgressMode = ProgressIndeterminate
			task.ProgressSource = "steam-content-log-validating"
		}
		return
	}

	// Prefer CURRENT AppManifest state over historical log flags. Historical
	// flags are sticky evidence ("this phase happened before"), not the current
	// phase. This prevents a past Committing line from pinning the UI forever
	// after Steam has already returned to StateFlags=4 (FullyInstalled + idle).
	if found {
		if app.BytesToDownload > 0 && app.BytesDownloaded <= app.BytesToDownload && app.BytesDownloaded < app.BytesToDownload {
			task.Phase, task.Message = PhaseDownloading, "Steam 正在下载/修复文件"
			task.BytesDone, task.BytesTotal = app.BytesDownloaded, app.BytesToDownload
			task.Progress = percent(app.BytesDownloaded, app.BytesToDownload)
			task.ProgressMode = ProgressDeterminate
			task.ProgressSource = "steam-appmanifest-bytesdownloaded"
			return
		}
		if app.BytesToStage > 0 && app.BytesStaged <= app.BytesToStage && app.BytesStaged < app.BytesToStage {
			task.Phase, task.Message = PhaseStaging, "Steam 正在暂存/安装文件"
			task.BytesDone, task.BytesTotal = app.BytesStaged, app.BytesToStage
			task.Progress = percent(app.BytesStaged, app.BytesToStage)
			task.ProgressMode = ProgressDeterminate
			task.ProgressSource = "steam-appmanifest-bytesstaged"
			return
		}
		if app.ValidationActive() {
			task.Phase = PhaseValidating
			if validationProgress.Available {
				task.Message = "Steam 正在验证文件。AI Game Manager Panel 根据 Validating 状态期间 Steam 进程的实际磁盘读取量同步估算当前扫描进度。"
				task.BytesDone, task.BytesTotal = validationProgress.BytesDone, validationProgress.BytesTotal
				task.Progress = validationProgress.Progress
				task.ProgressMode = ProgressDeterminate
				task.ProgressSource = "steam-windows-process-io-estimate"
				task.ProgressEstimated = true
			} else {
				task.Message = "Steam 正在验证文件。AI Game Manager Panel 已确认目标 AppID 处于真实 Validating 状态，正在等待可用的进度采样。"
				task.ProgressMode = ProgressIndeterminate
				task.ProgressSource = "steam-appmanifest-stateflags"
			}
			return
		}
		if app.DownloadActive() {
			task.Phase, task.Message = PhaseDownloading, "Steam 已进入下载/修复流程，等待 AppManifest 提供真实字节计数"
			task.ProgressMode = ProgressIndeterminate
			task.ProgressSource = "steam-appmanifest-stateflags"
			return
		}
		if app.StagingActive() {
			task.Phase, task.Message = PhaseStaging, "Steam 已进入文件暂存流程"
			task.ProgressMode = ProgressIndeterminate
			task.ProgressSource = "steam-appmanifest-stateflags"
			return
		}
		if app.CommittingActive() {
			task.Phase, task.Message = PhaseCommitting, "Steam 正在提交安装结果"
			task.ProgressMode = ProgressIndeterminate
			task.ProgressSource = "steam-appmanifest-stateflags"
			return
		}
		// After content_log reports File validation finished, Steam may still
		// commit repairs or remove the task from its scheduler. Hold at 99 until
		// a terminal event from this same session is observed.
		if task.Operation == OperationValidate && tracker.validationFinishedSeq > tracker.validationStartSeq && !tracker.validationTerminalSeen() {
			task.Phase, task.Message = PhaseFinalizing, "Steam 已完成文件扫描，正在等待本次校验任务真正退出 Steam 调度；确认前不会显示 100%"
			task.Progress = 99
			task.ProgressMode = ProgressDeterminate
			task.ProgressSource = "steam-terminal-gate"
			task.ProgressEstimated = true
			task.BytesDone, task.BytesTotal = 0, 0
			return
		}

		// AppManifest Validating can help with phase display on clients that do
		// expose the flag, but it is never sufficient to complete the task.
		if task.Operation == OperationValidate && tracker.sawManifestValidating && !tracker.validationSessionStarted() && !app.Busy() {
			task.Phase, task.Message = PhaseFinalizing, "Steam AppManifest 曾进入 Validating，但尚未收到本次 content_log 终态；AI Game Manager Panel 将继续等待，不会提前完成"
			task.Progress = 99
			task.ProgressMode = ProgressDeterminate
			task.ProgressSource = "steam-terminal-gate"
			task.ProgressEstimated = true
			return
		}
	}

	// If AppManifest is temporarily unavailable, content_log history is still a
	// useful fallback for showing what Steam most recently did. It is never used
	// to override an available current manifest state.
	if !found {
		if tracker.sawValidating {
			task.Phase, task.Message = PhaseValidating, "Steam 正在验证文件；暂时无法读取 AppManifest"
			task.ProgressMode = ProgressIndeterminate
			return
		}
		if tracker.sawDownloading {
			task.Phase, task.Message = PhaseDownloading, "Steam 已进入下载/修复流程；暂时无法读取 AppManifest"
			task.ProgressMode = ProgressIndeterminate
			return
		}
		if tracker.sawStaging {
			task.Phase, task.Message = PhaseStaging, "Steam 已进入文件暂存流程；暂时无法读取 AppManifest"
			task.ProgressMode = ProgressIndeterminate
			return
		}
		if tracker.sawCommitting {
			task.Phase, task.Message = PhaseCommitting, "Steam 曾进入提交阶段；暂时无法读取 AppManifest"
			task.ProgressMode = ProgressIndeterminate
			return
		}
	}

	task.Phase = PhaseWaiting
	if task.Operation == OperationInstall {
		task.Message = "等待 Steam 用户确认安装并进入下载队列"
	} else {
		task.Message = "等待 Steam 真正开始文件校验"
	}
	task.ProgressMode = ProgressIndeterminate
}

func completionMessage(operation string) string {
	if operation == OperationInstall {
		return "Steam 官方客户端已完成安装，AI Game Manager Panel 已从 Steam 日志和本地 AppManifest 双重确认"
	}
	return "Steam 官方文件完整性校验已完成，AI Game Manager Panel 已从 Steam 日志和本地 AppManifest 双重确认"
}

func (s *Service) put(task TaskSnapshot) { s.mu.Lock(); s.tasks[task.ID] = task; s.mu.Unlock() }
func (s *Service) update(id string, fn func(*TaskSnapshot)) {
	s.mu.Lock()
	task, ok := s.tasks[id]
	if ok {
		fn(&task)
		task.UpdatedAt = time.Now().Unix()
		s.tasks[id] = task
	}
	s.mu.Unlock()
}
