package token

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSaveDoesNotExposeValueAndInspectConfigured(t *testing.T) {
	cluster := t.TempDir()
	secret := "pds-g^SUPER-SECRET-TOKEN"
	status, err := Save(cluster, "  "+secret+"  ")
	if err != nil {
		t.Fatal(err)
	}
	if !status.Configured || status.State != StateConfigured {
		t.Fatalf("unexpected status: %#v", status)
	}
	if strings.Contains(status.Message, secret) || strings.Contains(status.Path, secret) {
		t.Fatalf("status leaked token: %#v", status)
	}
	content, err := os.ReadFile(filepath.Join(cluster, FileName))
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != secret+"\n" {
		t.Fatalf("unexpected token file content: %q", content)
	}
}

func TestSaveReplacesExistingTokenWithoutBackup(t *testing.T) {
	cluster := t.TempDir()
	if _, err := Save(cluster, "OLD-SECRET"); err != nil {
		t.Fatal(err)
	}
	if _, err := Save(cluster, "NEW-SECRET"); err != nil {
		t.Fatal(err)
	}
	content, err := os.ReadFile(filepath.Join(cluster, FileName))
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "NEW-SECRET\n" {
		t.Fatalf("unexpected replacement content: %q", content)
	}
	if _, err := os.Stat(filepath.Join(cluster, FileName+".bonfire-backup")); !os.IsNotExist(err) {
		t.Fatalf("unexpected token backup: %v", err)
	}
}

func TestInspectMissingAndEmpty(t *testing.T) {
	cluster := t.TempDir()
	status, err := Inspect(cluster)
	if err != nil || status.State != StateMissing {
		t.Fatalf("missing status=%#v err=%v", status, err)
	}
	if err := os.WriteFile(filepath.Join(cluster, FileName), []byte("  \n"), 0o600); err != nil {
		t.Fatal(err)
	}
	status, err = Inspect(cluster)
	if err != nil || status.State != StateEmpty {
		t.Fatalf("empty status=%#v err=%v", status, err)
	}
}
