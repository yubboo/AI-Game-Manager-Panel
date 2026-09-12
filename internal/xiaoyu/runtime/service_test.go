package xiaoyuruntime

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestMissingRuntimeReturnsStableStatus(t *testing.T) {
	service := New(Options{Root: t.TempDir(), BinaryOverride: filepath.Join(t.TempDir(), "missing-agent")})
	status := service.Status()
	if status.Available || status.Ready {
		t.Fatalf("missing runtime must not report ready: %+v", status)
	}
	if status.Protocol != ProtocolVersion {
		t.Fatalf("unexpected protocol: %q", status.Protocol)
	}
}

func TestWindowsRuntimeNameHasExeSuffix(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("windows-specific runtime path behavior")
	}
	service := New(Options{Root: t.TempDir()})
	if _, err := service.binaryPath(); err == nil {
		t.Fatal("unexpected runtime discovered in isolated temp root")
	}
}

func TestInternalAICoreRuntimePathIsPreferred(t *testing.T) {
	root := t.TempDir()
	name := "AI-Game-Manager-XiaoYu"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	internalDir := filepath.Join(root, "internal", "xiaoyu")
	if err := os.MkdirAll(internalDir, 0o755); err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(internalDir, name)
	if err := os.WriteFile(want, []byte("internal-agmp-ai-core"), 0o755); err != nil {
		t.Fatal(err)
	}
	service := New(Options{Root: root})
	got, err := service.binaryPath()
	if err != nil {
		t.Fatal(err)
	}
	gotAbs, _ := filepath.Abs(got)
	wantAbs, _ := filepath.Abs(want)
	if filepath.Clean(gotAbs) != filepath.Clean(wantAbs) {
		t.Fatalf("internal AI Core path = %q, want %q", gotAbs, wantAbs)
	}
}

func TestTailWriterKeepsBoundedSuffix(t *testing.T) {
	writer := newTailWriter(8)
	_, _ = writer.Write([]byte("12345"))
	_, _ = writer.Write([]byte("67890"))
	if got := writer.String(); got != "34567890" {
		t.Fatalf("tail writer = %q, want bounded suffix", got)
	}
}

func TestStartJobRejectsMissingHostAuthorizationBeforeRuntime(t *testing.T) {
	service := New(Options{Root: t.TempDir(), BinaryOverride: filepath.Join(t.TempDir(), "missing-runtime")})
	_, err := service.StartJob(nil, JobStartRequest{Executable: "example", HostAuthorized: false})
	if err == nil {
		t.Fatal("unauthorized Rust job must be rejected before contacting the runtime")
	}
}

func TestClosePersistentWorkerIsIdempotent(t *testing.T) {
	service := New(Options{Root: t.TempDir()})
	service.Close()
	service.Close()
}

func TestStartTerminalRejectsMissingHostAuthorizationBeforeRuntime(t *testing.T) {
	service := New(Options{Root: t.TempDir(), BinaryOverride: filepath.Join(t.TempDir(), "missing-runtime")})
	_, err := service.StartTerminal(nil, TerminalStartRequest{Executable: "example", HostAuthorized: false})
	if err == nil {
		t.Fatal("unauthorized Rust terminal must be rejected before contacting the runtime")
	}
}

func TestStartTerminalRejectsMissingCapabilityLeaseBeforeRuntime(t *testing.T) {
	service := New(Options{Root: t.TempDir(), BinaryOverride: filepath.Join(t.TempDir(), "missing-runtime")})
	_, err := service.StartTerminal(nil, TerminalStartRequest{Executable: "example", HostAuthorized: true})
	if err == nil {
		t.Fatal("host-authorized Rust terminal without a capability lease must be rejected before contacting the runtime")
	}
}

func TestWriteTerminalRejectsMissingHostAuthorizationBeforeRuntime(t *testing.T) {
	service := New(Options{Root: t.TempDir(), BinaryOverride: filepath.Join(t.TempDir(), "missing-runtime")})
	_, err := service.WriteTerminal(nil, TerminalWriteRequest{ID: "XYT-test", Data: "echo blocked", HostAuthorized: false})
	if err == nil {
		t.Fatal("unauthorized Rust terminal input must be rejected before contacting the runtime")
	}
}

func TestResizeTerminalRejectsMissingHostAuthorizationBeforeRuntime(t *testing.T) {
	service := New(Options{Root: t.TempDir(), BinaryOverride: filepath.Join(t.TempDir(), "missing-runtime")})
	_, err := service.ResizeTerminal(nil, TerminalResizeRequest{ID: "XYT-test", Rows: 40, Cols: 120, HostAuthorized: false})
	if err == nil {
		t.Fatal("unauthorized Rust terminal resize must be rejected before contacting the runtime")
	}
}
