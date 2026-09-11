package maintenance

import (
	"context"
	"sync"
	"testing"

	"github.com/yubboo/AI-Game-Manager-Panel/internal/platform/steam"
)

type fakeSteam struct {
	env   steam.Environment
	app   steam.AppInstallation
	found bool
}

func (f *fakeSteam) Detect(context.Context) (steam.Environment, error) { return f.env, nil }
func (f *fakeSteam) FindApp(context.Context, steam.AppID) (steam.AppInstallation, bool, error) {
	return f.app, f.found, nil
}

type fakeLauncher struct {
	mu  sync.Mutex
	uri string
}

func (f *fakeLauncher) Open(_ context.Context, uri string) error {
	f.mu.Lock()
	f.uri = uri
	f.mu.Unlock()
	return nil
}

func cleanInstalledApp() steam.AppInstallation {
	return steam.AppInstallation{InstallPathExists: true, StateFlags: steam.AppStateFullyInstalled}
}

func validatingInstalledApp() steam.AppInstallation {
	return steam.AppInstallation{InstallPathExists: true, StateFlags: steam.AppStateFullyInstalled | steam.AppStateUpdateRunning | steam.AppStateValidating}
}

func TestStartValidateUsesSteamProtocol(t *testing.T) {
	fs := &fakeSteam{
		env: steam.Environment{Detected: true, InstallPath: t.TempDir()},
		app: steam.AppInstallation{AppID: 1002, InstallPathExists: true, BuildID: "123", StateFlags: 4}, found: true,
	}
	launcher := &fakeLauncher{}
	svc := New(fs, launcher)
	task, err := svc.StartValidate(context.Background(), 1002)
	if err != nil {
		t.Fatal(err)
	}
	if task.URI != "steam://validate/1002" || task.Operation != OperationValidate {
		t.Fatalf("unexpected task %#v", task)
	}
	launcher.mu.Lock()
	got := launcher.uri
	launcher.mu.Unlock()
	if got != task.URI {
		t.Fatalf("launcher got %s", got)
	}
}

func TestActiveValidationRequestReusesExistingTask(t *testing.T) {
	fs := &fakeSteam{env: steam.Environment{Detected: true, InstallPath: t.TempDir()}}
	launcher := &fakeLauncher{}
	svc := New(fs, launcher)
	existing := newTask(1001, OperationValidate, "steam://validate/1001", "123")
	existing.State = StateMonitoring
	svc.put(existing)

	task, err := svc.StartValidate(context.Background(), 1001)
	if err != nil {
		t.Fatal(err)
	}
	if task.ID != existing.ID {
		t.Fatalf("expected existing task %s, got %s", existing.ID, task.ID)
	}
	launcher.mu.Lock()
	gotURI := launcher.uri
	launcher.mu.Unlock()
	if gotURI != "" {
		t.Fatalf("duplicate request must not relaunch Steam URI, got %q", gotURI)
	}
}

func TestCompletedValidationDoesNotBlockFreshSession(t *testing.T) {
	svc := New(&fakeSteam{}, &fakeLauncher{})
	done := newTask(1001, OperationValidate, "steam://validate/1001", "123")
	done.State = StateCompleted
	svc.put(done)
	if _, ok := svc.activeTask(1001, OperationValidate); ok {
		t.Fatal("completed validation must not block a fresh validation session")
	}
}

func TestStartInstallUsesUsersSteamClient(t *testing.T) {
	fs := &fakeSteam{env: steam.Environment{Detected: true, InstallPath: t.TempDir()}, found: false}
	launcher := &fakeLauncher{}
	svc := New(fs, launcher)
	task, err := svc.StartInstall(context.Background(), 1001)
	if err != nil {
		t.Fatal(err)
	}
	if task.URI != "steam://install/1001" || task.Operation != OperationInstall {
		t.Fatalf("unexpected task %#v", task)
	}
	launcher.mu.Lock()
	got := launcher.uri
	launcher.mu.Unlock()
	if got != task.URI {
		t.Fatalf("launcher got %s", got)
	}
}

func TestInstallRejectsAlreadyInstalledApp(t *testing.T) {
	fs := &fakeSteam{
		env: steam.Environment{Detected: true, InstallPath: t.TempDir()},
		app: steam.AppInstallation{AppID: 1001, InstallPathExists: true}, found: true,
	}
	svc := New(fs, &fakeLauncher{})
	if _, err := svc.StartInstall(context.Background(), 1001); err == nil {
		t.Fatal("expected already-installed error")
	}
}

func TestProgressUsesRealManifestCounters(t *testing.T) {
	task := TaskSnapshot{Operation: OperationInstall}
	app := steam.AppInstallation{BytesDownloaded: 25, BytesToDownload: 100}
	applyManifestProgress(&task, app, true, logTracker{}, validationEstimate{})
	if task.ProgressMode != ProgressDeterminate || task.Progress != 25 || task.BytesDone != 25 || task.BytesTotal != 100 {
		t.Fatalf("unexpected progress %#v", task)
	}
}

func TestValidationWithoutSteamCountersStaysIndeterminate(t *testing.T) {
	task := TaskSnapshot{Operation: OperationValidate}
	applyManifestProgress(&task, validatingInstalledApp(), true, logTracker{sawValidating: true}, validationEstimate{})
	if task.ProgressMode != ProgressIndeterminate || task.Phase != PhaseValidating {
		t.Fatalf("unexpected progress %#v", task)
	}
}

func TestValidationUsesSteamProcessIOEstimate(t *testing.T) {
	task := TaskSnapshot{Operation: OperationValidate}
	estimate := validationEstimate{Available: true, BytesDone: 470, BytesTotal: 1000, Progress: 47}
	applyManifestProgress(&task, validatingInstalledApp(), true, logTracker{sawValidating: true}, estimate)
	if task.ProgressMode != ProgressDeterminate || task.Progress != 47 || !task.ProgressEstimated {
		t.Fatalf("expected live estimated validation progress, got %#v", task)
	}
	if task.BytesDone != 470 || task.BytesTotal != 1000 || task.ProgressSource != "steam-windows-process-io-estimate" {
		t.Fatalf("unexpected validation progress source %#v", task)
	}
}

func TestValidationIOTrackerOnlyCountsWhileTargetIsValidating(t *testing.T) {
	tracker := validationIOTracker{last: 1000, available: true, sampled: true}
	tracker.observeValue(false, 1200, true)
	tracker.observeValue(true, 1500, true)
	estimate := tracker.estimate(1000)
	if !estimate.Available || estimate.BytesDone != 300 || estimate.Progress != 30 {
		t.Fatalf("unexpected IO estimate %#v / tracker %#v", estimate, tracker)
	}
}

func TestValidationIOEstimateNeverReachesHundredBeforeSteamTerminalState(t *testing.T) {
	tracker := validationIOTracker{last: 0, accumulated: 5000, available: true, sampled: true}
	estimate := tracker.estimate(1000)
	if !estimate.Available || estimate.Progress != 99 || estimate.BytesDone >= estimate.BytesTotal {
		t.Fatalf("validation estimate must be capped below terminal completion: %#v", estimate)
	}
}

func TestTrackerRequiresTargetAppCompletion(t *testing.T) {
	tracker := logTracker{appID: 1001}
	tracker.consume([]string{
		"[x] AppID 570 update changed : Running,Validating,",
		"[x] AppID 570 scheduler finished : removed from schedule",
	})
	if tracker.sawSchedulerFinished || tracker.sawValidating {
		t.Fatal("other app must not complete this task")
	}
	tracker.consume([]string{
		"[x] AppID 1001 update changed : Running,Validating,",
		"[x] Validating files (all active,full) ...",
		"[x] AppID 1001 state changed : Fully Installed,",
		"[x] AppID 1001 scheduler finished : removed from schedule",
	})
	if !tracker.sawValidating || !tracker.sawFullyInstalled || !tracker.sawSchedulerFinished {
		t.Fatalf("target app was not tracked: %#v", tracker)
	}
}

func TestFullyInstalledDuringValidationIsNotCompletion(t *testing.T) {
	tracker := logTracker{appID: 1001}
	tracker.consume([]string{
		"[2026-09-08 13:08:00] Start validating appID 1001",
		"[2026-09-08 13:08:00] AppID 1001 state changed : Fully Installed,Update Queued,",
		"[2026-09-08 13:08:00] AppID 1001 state changed : Fully Installed,Update Queued,Update Running,",
		"[2026-09-08 13:08:00] AppID 1001 update changed : Running,Validating,",
	})
	if !tracker.sawFullyInstalled || !tracker.sawValidationStarted {
		t.Fatalf("expected Steam start signals: %#v", tracker)
	}
	if canComplete(OperationValidate, tracker, validatingInstalledApp(), true) {
		t.Fatal("Fully Installed while Steam is still validating must never complete the task")
	}
}

func TestValidationCanFinishWithoutSchedulerWhenSteamReachedCleanIdleTerminalState(t *testing.T) {
	tracker := logTracker{appID: 1001}
	tracker.consume([]string{
		"[x] Start validating appID 1001",
		"[x] AppID 1001 update changed : Running,Validating,",
		"[x] Validating files (all active,full) ...",
		"[x] File validation finished: 123 files (456 bytes) total, 0 files (0 bytes) mismatched (1000 msec).",
		"[x] AppID 1001 update changed : None",
		"[x] AppID 1001 state changed : Fully Installed,",
	})
	if !tracker.sawValidationFinished {
		t.Fatalf("expected validation-finished signal: %#v", tracker)
	}
	if !canComplete(OperationValidate, tracker, cleanInstalledApp(), true) {
		t.Fatal("current Steam GUI may omit scheduler-finished; explicit validation-finished + Update None + clean Fully Installed + idle manifest must be accepted")
	}
}

func TestValidationCompletesOnlyAfterTargetSchedulerFinished(t *testing.T) {
	tracker := logTracker{appID: 1001}
	tracker.consume([]string{
		"[x] Start validating appID 1001",
		"[x] AppID 1001 update changed : Running,Validating,",
		"[x] Validating files (all active,full) ...",
		"[x] File validation finished: 123 files (456 bytes) total, 0 files (0 bytes) mismatched (1000 msec).",
		"[x] AppID 1001 update changed : None",
		"[x] AppID 1001 state changed : Fully Installed,",
		"[x] AppID 1001 scheduler finished : removed from schedule (result No Error, state 0x4)",
	})
	if !canComplete(OperationValidate, tracker, cleanInstalledApp(), true) {
		t.Fatalf("expected true terminal validation state: %#v", tracker)
	}
}

func TestSchedulerFinishedWithoutThisValidationDoesNotComplete(t *testing.T) {
	tracker := logTracker{appID: 1001}
	tracker.consume([]string{
		"[x] AppID 1001 state changed : Fully Installed,",
		"[x] AppID 1001 scheduler finished : removed from schedule (result No Error, state 0x4)",
	})
	if canComplete(OperationValidate, tracker, cleanInstalledApp(), true) {
		t.Fatal("stale/non-validation scheduler completion must not complete a validate task")
	}
}

func TestGenericValidationLinesAreBoundToExplicitValidationApp(t *testing.T) {
	tracker := logTracker{appID: 1001}
	tracker.consume([]string{
		"[x] Start validating appID 730",
		"[x] Validating files (all active,full) ...",
		"[x] File validation finished: 10 files (100 bytes) total, 0 files (0 bytes) mismatched (1 msec).",
		"[x] AppID 1001 state changed : Fully Installed,",
		"[x] AppID 1001 scheduler finished : removed from schedule (result No Error, state 0x4)",
	})
	if tracker.sawValidationStarted || tracker.sawValidationFinished {
		t.Fatalf("generic validation output from another AppID leaked into target: %#v", tracker)
	}
	if canComplete(OperationValidate, tracker, cleanInstalledApp(), true) {
		t.Fatal("another AppID's validation must never complete this task")
	}
}

func TestValidationFinishedShowsFinalizingUntilSchedulerFinished(t *testing.T) {
	task := TaskSnapshot{Operation: OperationValidate}
	tracker := logTracker{appID: 1001}
	tracker.consume([]string{
		"[x] AppID 1001 update changed : Running,Validating,",
		"[x] Validating files (all active,full) ...",
		"[x] File validation finished: 123 files (456 bytes) total, 0 files (0 bytes) mismatched (1000 msec).",
	})
	applyManifestProgress(&task, cleanInstalledApp(), true, tracker, validationEstimate{})
	if task.Phase != PhaseFinalizing || task.ProgressMode != ProgressDeterminate || task.Progress != 99 {
		t.Fatalf("expected 99%% terminal gate until Steam completion is confirmed, got %#v", task)
	}
}

func TestManifestValidatingFlagIsBusyAndPending(t *testing.T) {
	app := steam.AppInstallation{StateFlags: steam.AppStateFullyInstalled | steam.AppStateUpdateRunning | steam.AppStateValidating}
	if !app.ValidationActive() || !app.Busy() || !app.UpdatePending() {
		t.Fatalf("validating app must be busy/pending: %#v", app)
	}
}

func TestValidationResultCountsAreParsed(t *testing.T) {
	tracker := logTracker{appID: 1001}
	tracker.consume([]string{
		"Start validating appID 1001",
		"Validating files (all active,full) ...",
		"File validation finished: 123 files (456789 bytes) total, 2 files (1024 bytes) mismatched (1000 msec).",
	})
	if tracker.validationFiles != 123 || tracker.validationBytes != 456789 || tracker.mismatchedFiles != 2 || tracker.mismatchedBytes != 1024 {
		t.Fatalf("unexpected validation result: %#v", tracker)
	}
}

func TestManifestIdleFallbackDoesNotCompleteWhenValidationFoundMismatches(t *testing.T) {
	tracker := logTracker{appID: 1001}
	tracker.consume([]string{
		"[x] AppID 1001 update changed : Running,Validating,",
		"[x] Validating files (all active,full) ...",
		"[x] File validation finished: 123 files (456 bytes) total, 2 files (1024 bytes) mismatched (1000 msec).",
	})
	if canComplete(OperationValidate, tracker, cleanInstalledApp(), true) {
		t.Fatal("mismatched validation must wait for Steam repair/terminal evidence")
	}
}

func TestHistoricalCommittingDoesNotPinIdleManifestPhase(t *testing.T) {
	task := TaskSnapshot{Operation: OperationValidate}
	tracker := logTracker{appID: 1001}
	tracker.consume([]string{
		"[x] AppID 1001 update changed : Running,Validating,",
		"[x] Validating files (all active,full) ...",
		"[x] File validation finished: 123 files (456 bytes) total, 0 files (0 bytes) mismatched (1000 msec).",
		"[x] AppID 1001 update changed : Running,Committing,",
	})
	applyManifestProgress(&task, cleanInstalledApp(), true, tracker, validationEstimate{})
	if task.Phase != PhaseFinalizing {
		t.Fatalf("historical committing must not override current idle manifest, got %#v", task)
	}
}

func TestManifestIdleNeverCompletesLiveValidationWithoutTerminalLog(t *testing.T) {
	tracker := logTracker{appID: 1001}
	tracker.consume([]string{
		"[x] AppID 1001 update changed : Running,Validating,",
		"[x] Validating files (all active,full) ...",
	})
	if canComplete(OperationValidate, tracker, cleanInstalledApp(), true) {
		t.Fatal("StateFlags=4 while Steam is visibly validating must never complete the task")
	}
}

func TestContentLogValidationOverridesIdleManifestAndKeepsLiveProgress(t *testing.T) {
	task := TaskSnapshot{Operation: OperationValidate}
	tracker := logTracker{appID: 1001}
	tracker.consume([]string{
		"[x] AppID 1001 update changed : Running,Validating,",
		"[x] Validating files (all active,full) ...",
	})
	estimate := validationEstimate{Available: true, BytesDone: 420, BytesTotal: 1000, Progress: 42}
	applyManifestProgress(&task, cleanInstalledApp(), true, tracker, estimate)
	if task.Phase != PhaseValidating || task.ProgressMode != ProgressDeterminate || task.Progress != 42 {
		t.Fatalf("live content_log validation must override idle StateFlags=4: %#v", task)
	}
	if canComplete(OperationValidate, tracker, cleanInstalledApp(), true) {
		t.Fatal("live validation with idle manifest must stay incomplete")
	}
}

func TestIOFallbackStartsValidationButCanNeverCompleteIt(t *testing.T) {
	tracker := logTracker{appID: 1001}
	tracker.startValidationFromIO()
	if !tracker.validationScanning() || !tracker.sawValidationIOStart {
		t.Fatalf("I/O fallback did not start validation session: %#v", tracker)
	}
	if canComplete(OperationValidate, tracker, cleanInstalledApp(), true) {
		t.Fatal("I/O fallback is start evidence only and must never complete validation")
	}
	tracker.consume([]string{
		"[x] File validation finished: 123 files (456 bytes) total, 0 files (0 bytes) mismatched (1000 msec).",
		"[x] AppID 1001 update changed : None",
		"[x] AppID 1001 state changed : Fully Installed,",
	})
	if !canComplete(OperationValidate, tracker, cleanInstalledApp(), true) {
		t.Fatal("I/O-started session should complete only after new content_log terminal evidence")
	}
}

func TestRepeatedValidationRequiresFreshSessionTerminal(t *testing.T) {
	first := logTracker{appID: 1001}
	first.consume([]string{
		"[first] AppID 1001 update changed : Running,Validating,",
		"[first] Validating files (all active,full) ...",
		"[first] File validation finished: 123 files (456 bytes) total, 0 files (0 bytes) mismatched (1000 msec).",
		"[first] AppID 1001 update changed : None",
		"[first] AppID 1001 state changed : Fully Installed,",
		"[first] AppID 1001 scheduler finished : removed from schedule (result No Error, state 0x4)",
	})
	if !canComplete(OperationValidate, first, cleanInstalledApp(), true) {
		t.Fatal("first validation should complete")
	}

	second := logTracker{appID: 1001}
	if canComplete(OperationValidate, second, cleanInstalledApp(), true) {
		t.Fatal("second validation must not reuse first validation terminal state")
	}
	second.consume([]string{
		"[second] AppID 1001 update changed : Running,Validating,",
		"[second] Validating files (all active,full) ...",
	})
	if !second.validationScanning() || canComplete(OperationValidate, second, cleanInstalledApp(), true) {
		t.Fatal("second validation must remain active until its own terminal event")
	}
	second.consume([]string{
		"[second] File validation finished: 123 files (456 bytes) total, 0 files (0 bytes) mismatched (1000 msec).",
		"[second] AppID 1001 update changed : None",
		"[second] AppID 1001 state changed : Fully Installed,",
	})
	if !canComplete(OperationValidate, second, cleanInstalledApp(), true) {
		t.Fatal("second validation should complete only from its fresh terminal sequence")
	}
}

func TestValidateDoesNotCompleteBeforeValidationWasObserved(t *testing.T) {
	tracker := logTracker{appID: 1001}
	if canComplete(OperationValidate, tracker, cleanInstalledApp(), true) {
		t.Fatal("an already installed idle app must not complete a newly requested validation before Steam actually starts it")
	}
}
