package kleiarchive

import (
	"archive/zip"
	"bytes"
	"encoding/base64"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func makeZip(t *testing.T, files map[string]string) []byte {
	t.Helper()
	var buffer bytes.Buffer
	writer := zip.NewWriter(&buffer)
	for name, body := range files {
		entry, err := writer.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := entry.Write([]byte(body)); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	return buffer.Bytes()
}

func TestInspectOfficialPackageNeverReturnsToken(t *testing.T) {
	secret := "pds-g^SUPER_SECRET_TOKEN"
	data := makeZip(t, map[string]string{
		"MyDediServer/cluster_token.txt": secret,
		"MyDediServer/cluster.ini":       "[GAMEPLAY]\ngame_mode = survival\nmax_players = 6\n[NETWORK]\ncluster_name = Super Server\ncluster_description = Hello\ncluster_password = abc\n",
		"MyDediServer/Master/server.ini": "[NETWORK]\nserver_port = 10999\n[SHARD]\nis_master = true\n",
		"MyDediServer/Caves/server.ini":  "[NETWORK]\nserver_port = 11000\n",
	})
	preview, err := Inspect("MyDediServer.zip", data)
	if err != nil {
		t.Fatal(err)
	}
	if !preview.HasToken || !preview.HasClusterINI || !preview.HasMasterServerINI || !preview.HasCavesServerINI {
		t.Fatalf("unexpected preview: %+v", preview)
	}
	if preview.ServerName != "Super Server" || preview.MaxPlayers != "6" || preview.GameMode != "survival" || !preview.Passworded {
		t.Fatalf("metadata mismatch: %+v", preview)
	}
	if strings.Contains(strings.Join([]string{preview.ArchiveName, preview.RootPrefix, preview.ServerName, preview.Description}, "|"), secret) {
		t.Fatal("preview leaked token")
	}
}

func TestInspectRootLevelOfficialPackage(t *testing.T) {
	data := makeZip(t, map[string]string{
		"cluster_token.txt": "pds-g^TOKEN",
		"cluster.ini":       "[NETWORK]\ncluster_name = Root Package\n",
		"Master/server.ini": "[NETWORK]\nserver_port = 10999\n[SHARD]\nis_master = true\n",
	})
	preview, err := Inspect("Root.zip", data)
	if err != nil {
		t.Fatal(err)
	}
	if preview.RootPrefix != "" || preview.ServerName != "Root Package" {
		t.Fatalf("unexpected preview: %+v", preview)
	}
}

func TestRejectsZipSlip(t *testing.T) {
	data := makeZip(t, map[string]string{"../cluster_token.txt": "secret"})
	if _, err := Inspect("bad.zip", data); err == nil {
		t.Fatal("expected zip-slip rejection")
	}
}

func TestRejectsSymlinkEntry(t *testing.T) {
	var buffer bytes.Buffer
	writer := zip.NewWriter(&buffer)
	header := &zip.FileHeader{Name: "pkg/cluster_token.txt", Method: zip.Store}
	header.SetMode(os.ModeSymlink | 0o777)
	entry, err := writer.CreateHeader(header)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := entry.Write([]byte("target")); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := Inspect("bad.zip", buffer.Bytes()); err == nil {
		t.Fatal("expected symlink rejection")
	}
}

func TestApplyTokenOnlyPreservesExistingConfig(t *testing.T) {
	cluster := t.TempDir()
	if err := os.MkdirAll(filepath.Join(cluster, "Master"), 0o755); err != nil {
		t.Fatal(err)
	}
	original := "[NETWORK]\ncluster_name = Existing\n"
	if err := os.WriteFile(filepath.Join(cluster, "cluster.ini"), []byte(original), 0o644); err != nil {
		t.Fatal(err)
	}
	data := makeZip(t, map[string]string{
		"MyDediServer/cluster_token.txt": "pds-g^TOKEN",
		"MyDediServer/cluster.ini":       "[NETWORK]\ncluster_name = Imported\n",
		"MyDediServer/Master/server.ini": "[SHARD]\nis_master = true\n",
	})
	result, err := Apply(ImportRequest{ClusterPath: cluster, ArchiveName: "MyDediServer.zip", ArchiveBase64: base64.StdEncoding.EncodeToString(data), Mode: ModeTokenOnly})
	if err != nil {
		t.Fatal(err)
	}
	if !result.TokenConfigured || result.ConfigFilesCopied != 0 {
		t.Fatalf("unexpected result: %+v", result)
	}
	body, _ := os.ReadFile(filepath.Join(cluster, "cluster.ini"))
	if string(body) != original {
		t.Fatal("token-only import changed cluster.ini")
	}
	tokenBody, _ := os.ReadFile(filepath.Join(cluster, "cluster_token.txt"))
	if strings.TrimSpace(string(tokenBody)) != "pds-g^TOKEN" {
		t.Fatal("token not imported")
	}
}

func TestApplyFullConfigBacksUpExistingConfig(t *testing.T) {
	cluster := t.TempDir()
	if err := os.MkdirAll(filepath.Join(cluster, "Master"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cluster, "cluster.ini"), []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cluster, "Master", "server.ini"), []byte("old-master"), 0o644); err != nil {
		t.Fatal(err)
	}
	data := makeZip(t, map[string]string{
		"pkg/cluster_token.txt": "pds-g^TOKEN",
		"pkg/cluster.ini":       "new",
		"pkg/Master/server.ini": "new-master",
	})
	result, err := Apply(ImportRequest{ClusterPath: cluster, ArchiveName: "pkg.zip", ArchiveBase64: base64.StdEncoding.EncodeToString(data), Mode: ModeFullConfig})
	if err != nil {
		t.Fatal(err)
	}
	if result.ConfigFilesCopied != 2 || result.BackupPath == "" {
		t.Fatalf("unexpected result: %+v", result)
	}
	backup, err := os.ReadFile(filepath.Join(result.BackupPath, "cluster.ini"))
	if err != nil || string(backup) != "old" {
		t.Fatalf("backup missing: %v %q", err, backup)
	}
}
