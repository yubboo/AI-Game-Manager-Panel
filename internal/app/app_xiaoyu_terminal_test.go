package app

import (
	"context"
	"runtime"
	"strings"
	"testing"
	"unicode/utf8"

	opsfiles "github.com/yubboo/AI-Game-Manager-Panel/internal/ops/files"
	xiaoyuruntime "github.com/yubboo/AI-Game-Manager-Panel/internal/xiaoyu/runtime"
)

type fakeNativeTerminalRuntime struct {
	startRequest xiaoyuruntime.TerminalStartRequest
	started      bool
	closed       bool
}

func (f *fakeNativeTerminalRuntime) Start(context.Context) error { return nil }
func (f *fakeNativeTerminalRuntime) Close()                      {}
func (f *fakeNativeTerminalRuntime) Status() xiaoyuruntime.Status {
	return xiaoyuruntime.Status{Ready: true}
}
func (f *fakeNativeTerminalRuntime) SearchTools(context.Context, string, []xiaoyuruntime.ToolSpec, int) (xiaoyuruntime.ToolSearchResponse, error) {
	return xiaoyuruntime.ToolSearchResponse{}, nil
}
func (f *fakeNativeTerminalRuntime) StartTerminal(_ context.Context, request xiaoyuruntime.TerminalStartRequest) (xiaoyuruntime.TerminalSnapshot, error) {
	f.started = true
	f.startRequest = request
	return xiaoyuruntime.TerminalSnapshot{ID: "XYT-test", State: "running", Backend: "native-test"}, nil
}
func (f *fakeNativeTerminalRuntime) GetTerminal(context.Context, string) (xiaoyuruntime.TerminalSnapshot, error) {
	exit := 0
	return xiaoyuruntime.TerminalSnapshot{ID: "XYT-test", State: "exited", Backend: "native-test", ExitCode: &exit}, nil
}
func (f *fakeNativeTerminalRuntime) TerminalOutput(context.Context, string, uint64, int) (xiaoyuruntime.TerminalOutputResponse, error) {
	return xiaoyuruntime.TerminalOutputResponse{
		ID:         "XYT-test",
		NextCursor: 1,
		Chunks:     []xiaoyuruntime.JobOutputChunk{{Sequence: 1, Stream: "pty", Text: "native-output\n"}},
	}, nil
}
func (f *fakeNativeTerminalRuntime) CloseTerminal(context.Context, string) (xiaoyuruntime.TerminalSnapshot, error) {
	f.closed = true
	return xiaoyuruntime.TerminalSnapshot{ID: "XYT-test", State: "closed", Backend: "native-test"}, nil
}

func TestApprovedAgentShellUsesHostAuthorizedNativeTerminal(t *testing.T) {
	root := t.TempDir()
	files, err := opsfiles.New(root)
	if err != nil {
		t.Fatal(err)
	}
	runtimeClient := &fakeNativeTerminalRuntime{}
	app := &Application{workspaceFiles: files, xiaoyuRuntime: runtimeClient}
	ctx := withXiaoYuInvocationContext(context.Background(), xiaoyuInvocationContext{RunID: "RUN-1"})

	result, err := app.runApprovedAgentTerminal(ctx, "echo native", "")
	if err != nil {
		t.Fatal(err)
	}
	if !runtimeClient.started || !runtimeClient.closed {
		t.Fatalf("native terminal lifecycle was not completed: started=%v closed=%v", runtimeClient.started, runtimeClient.closed)
	}
	if !runtimeClient.startRequest.HostAuthorized {
		t.Fatal("approved Agent terminal start must carry HostAuthorized=true")
	}
	if runtimeClient.startRequest.Cwd != root {
		t.Fatalf("native terminal cwd must stay inside resolved workspace: got %q want %q", runtimeClient.startRequest.Cwd, root)
	}
	if result.ExitCode != 0 || !strings.Contains(result.Stdout, "native-output") {
		t.Fatalf("unexpected native terminal receipt: %+v", result)
	}
	joined := strings.Join(runtimeClient.startRequest.Arguments, " ")
	if !strings.Contains(joined, "echo native") {
		t.Fatalf("approved command was not bound to native terminal start: %q", joined)
	}
	if runtime.GOOS == "windows" && runtimeClient.startRequest.Executable != "powershell.exe" {
		t.Fatalf("Windows Agent terminal must use PowerShell one-shot shell, got %q", runtimeClient.startRequest.Executable)
	}
	if runtime.GOOS != "windows" && runtimeClient.startRequest.Executable != "sh" {
		t.Fatalf("Unix Agent terminal must use sh one-shot shell, got %q", runtimeClient.startRequest.Executable)
	}
}

func TestApprovedAgentShellRejectsNonRunCallerBeforeNativeRuntime(t *testing.T) {
	root := t.TempDir()
	files, err := opsfiles.New(root)
	if err != nil {
		t.Fatal(err)
	}
	runtimeClient := &fakeNativeTerminalRuntime{}
	app := &Application{workspaceFiles: files, xiaoyuRuntime: runtimeClient}

	if _, err := app.runApprovedAgentTerminal(context.Background(), "echo blocked", ""); err == nil {
		t.Fatal("manual/non-Run caller must not enter the Agent Native Terminal path")
	}
	if runtimeClient.started {
		t.Fatal("native terminal must not start before a server-owned Run authorization context exists")
	}
}

func TestBoundedTerminalOutputKeepsUTF8Boundary(t *testing.T) {
	var output strings.Builder
	truncated := false
	appendBoundedTerminalOutput(&output, "你好世界", 7, &truncated)
	if !truncated {
		t.Fatal("bounded output must report truncation")
	}
	if !utf8.ValidString(output.String()) {
		t.Fatalf("truncated output must remain valid UTF-8: %q", output.String())
	}
	if output.Len() > 7 {
		t.Fatalf("bounded output exceeded limit: %d", output.Len())
	}
}
