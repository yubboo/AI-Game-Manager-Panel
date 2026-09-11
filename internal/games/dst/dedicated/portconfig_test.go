package dedicated

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReadPortConfigurationUsesRecommendedForMissingValues(t *testing.T) {
	root := t.TempDir()
	mustWritePortTest(t, filepath.Join(root, "cluster.ini"), "[SHARD]\nshard_enabled = true\nbind_ip = 127.0.0.1\nmaster_ip = 127.0.0.1\ncluster_key = hidden\n")
	mustWritePortTest(t, filepath.Join(root, "Master", "server.ini"), "[NETWORK]\nserver_port = 10999\n[SHARD]\nis_master = true\n")
	mustWritePortTest(t, filepath.Join(root, "Caves", "server.ini"), "[NETWORK]\nserver_port = 11000\n[SHARD]\nis_master = false\n")

	value, err := ReadPortConfiguration(root)
	if err != nil {
		t.Fatal(err)
	}
	if value.Valid {
		t.Fatal("missing Steam ports must require repair before startup")
	}
	if value.Effective.MasterSteamMasterPort != 27016 || value.Effective.CavesSteamAuthPort != 8767 || value.Effective.ShardMasterPort != 10888 {
		t.Fatalf("unexpected effective recommendation: %#v", value.Effective)
	}
}

func TestApplyPortSettingsWritesRecommendedProfile(t *testing.T) {
	root := t.TempDir()
	mustWritePortTest(t, filepath.Join(root, "cluster.ini"), "[MISC]\nconsole_enabled = true\n[SHARD]\nshard_enabled = true\nbind_ip = 127.0.0.1\nmaster_ip = 127.0.0.1\ncluster_key = hidden\n")
	mustWritePortTest(t, filepath.Join(root, "Master", "server.ini"), "[NETWORK]\nserver_port = 10999\n[SHARD]\nis_master = true\n")
	mustWritePortTest(t, filepath.Join(root, "Caves", "server.ini"), "[NETWORK]\nserver_port = 11000\n[SHARD]\nis_master = false\n")

	result, err := ApplyPortSettings(root, RecommendedPortSettings(true))
	if err != nil {
		t.Fatal(err)
	}
	if !result.Configuration.Valid {
		t.Fatalf("expected valid configuration: %#v", result.Configuration)
	}
	if result.BackupPath == "" {
		t.Fatal("expected backup path")
	}
	if _, err := os.Stat(filepath.Join(result.BackupPath, "Master", "server.ini")); err != nil {
		t.Fatalf("missing backup: %v", err)
	}

	master, _ := os.ReadFile(filepath.Join(root, "Master", "server.ini"))
	caves, _ := os.ReadFile(filepath.Join(root, "Caves", "server.ini"))
	cluster, _ := os.ReadFile(filepath.Join(root, "cluster.ini"))
	for _, pair := range []struct {
		body     []byte
		expected string
	}{
		{master, "master_server_port = 27016"}, {master, "authentication_port = 8766"},
		{caves, "master_server_port = 27017"}, {caves, "authentication_port = 8767"},
		{cluster, "master_port = 10888"},
	} {
		if !strings.Contains(string(pair.body), pair.expected) {
			t.Fatalf("missing %q in %s", pair.expected, string(pair.body))
		}
	}
}

func TestValidatePortSettingsRejectsDuplicate(t *testing.T) {
	settings := RecommendedPortSettings(true)
	settings.CavesServerPort = settings.MasterServerPort
	if err := ValidatePortSettings(settings, true); err == nil {
		t.Fatal("expected duplicate rejection")
	}
}
