//go:build agmp_dev_license

package license

import (
	"path/filepath"
	"testing"
)

func TestDevelopmentBuildAllowsFeatureWithoutCertificate(t *testing.T) {
	service := New(DefaultConfig(), filepath.Join(t.TempDir(), "license", "activation.json"))
	status := service.Status()
	if status.State != StateDevelopment || !status.Valid || status.Edition != "development" {
		t.Fatalf("unexpected development license status: %+v", status)
	}
	feature := service.HasFeature("ai.workbench")
	if !feature.Allowed {
		t.Fatalf("development build should allow feature: %+v", feature)
	}
}
