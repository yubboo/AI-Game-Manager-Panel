package protocol

import "testing"

func TestBuildValidateURI(t *testing.T) {
	if got := BuildValidateURI(343050); got != "steam://validate/343050" {
		t.Fatalf("unexpected validate URI: %s", got)
	}
	if got := BuildValidateURI(322330); got != "steam://validate/322330" {
		t.Fatalf("unexpected game validate URI: %s", got)
	}
}

func TestBuildInstallURI(t *testing.T) {
	if got := BuildInstallURI(322330); got != "steam://install/322330" {
		t.Fatalf("unexpected install URI: %s", got)
	}
	if got := BuildInstallURI(343050); got != "steam://install/343050" {
		t.Fatalf("unexpected dedicated server install URI: %s", got)
	}
}
