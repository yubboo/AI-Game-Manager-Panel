//go:build agmp_dev_license

package httpapi

import (
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"testing"

	application "github.com/yubboo/AI-Game-Manager-Panel/internal/app"
	authservice "github.com/yubboo/AI-Game-Manager-Panel/internal/system/auth"
)

func TestXiaoYuHarnessRoutesExposeStatusAndFailClosedWithoutBrainInDevBuild(t *testing.T) {
	root := t.TempDir()
	app := application.NewWithOptions(application.Options{Root: root, DataDir: "data"})
	handler := New(app, Options{}).routes()
	owner, err := app.CreateInitialAdministrator(authservice.CreateOwnerRequest{Username: "owner", Password: "agmp-test-password"})
	if err != nil {
		t.Fatal(err)
	}

	statusRecorder := request(handler, http.MethodGet, "/api/v1/xiaoyu/harness", nil, owner.Token)
	if statusRecorder.Code != http.StatusOK {
		t.Fatalf("harness status=%d body=%s", statusRecorder.Code, statusRecorder.Body.String())
	}
	var status application.XiaoYuHarnessSnapshot
	if err := json.Unmarshal(statusRecorder.Body.Bytes(), &status); err != nil {
		t.Fatal(err)
	}
	if status.PluginAPI != "xiaoyu.plugin.v1" || len(status.Capabilities.Tools) < 10 {
		t.Fatalf("unexpected harness snapshot: %+v", status)
	}

	startRecorder := request(handler, http.MethodPost, "/api/v1/xiaoyu/runs", []byte(`{"goal":"检查 DST 状态"}`), owner.Token)
	if startRecorder.Code != http.StatusBadRequest {
		t.Fatalf("run without brain must fail closed: status=%d body=%s", startRecorder.Code, startRecorder.Body.String())
	}
}

func TestLinuxHeadlessWebCanReachPrecompiledXiaoYuRuntime(t *testing.T) {
	binary := strings.TrimSpace(os.Getenv("AGMP_HEADLESS_XIAOYU_RUNTIME"))
	if binary == "" {
		t.Skip("AGMP_HEADLESS_XIAOYU_RUNTIME is only set by the headless CI integration job")
	}
	if _, err := os.Stat(binary); err != nil {
		t.Fatalf("headless XiaoYu runtime unavailable: %v", err)
	}
	t.Setenv("AGMP_XIAOYU_RUNTIME", binary)
	root := t.TempDir()
	app := application.NewWithOptions(application.Options{Root: root, DataDir: "data"})
	handler := New(app, Options{}).routes()
	owner, err := app.CreateInitialAdministrator(authservice.CreateOwnerRequest{Username: "owner", Password: "agmp-test-password"})
	if err != nil {
		t.Fatal(err)
	}
	recorder := request(handler, http.MethodGet, "/api/v1/xiaoyu/runtime", nil, owner.Token)
	if recorder.Code != http.StatusOK {
		t.Fatalf("runtime status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	var status struct {
		Available bool   `json:"available"`
		Ready     bool   `json:"ready"`
		Protocol  string `json:"protocol"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &status); err != nil {
		t.Fatal(err)
	}
	if !status.Available || !status.Ready || status.Protocol != "xiaoyu.v1" {
		t.Fatalf("headless Web did not reach ready XiaoYu runtime: %+v body=%s", status, recorder.Body.String())
	}
}
