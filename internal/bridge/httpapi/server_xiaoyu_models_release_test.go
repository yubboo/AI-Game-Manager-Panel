//go:build !agmp_dev_license

package httpapi

import (
	"net/http"
	"strings"
	"testing"

	application "github.com/yubboo/AI-Game-Manager-Panel/internal/app"
	authservice "github.com/yubboo/AI-Game-Manager-Panel/internal/system/auth"
)

func TestXiaoYuModelCenterRequiresOfficialLicenseInReleaseBuild(t *testing.T) {
	app := application.NewWithOptions(application.Options{Root: t.TempDir(), DataDir: "data"})
	handler := New(app, Options{}).routes()
	owner, err := app.CreateInitialAdministrator(authservice.CreateOwnerRequest{Username: "owner", Password: "agmp-test-password"})
	if err != nil {
		t.Fatal(err)
	}
	recorder := request(handler, http.MethodGet, "/api/v1/xiaoyu/models", nil, owner.Token)
	if recorder.Code == http.StatusOK || !strings.Contains(recorder.Body.String(), "许可证") {
		t.Fatalf("release XiaoYu model center must fail closed without official license: status=%d body=%s", recorder.Code, recorder.Body.String())
	}
}
