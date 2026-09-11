package dedicated

import (
	"path/filepath"
	"testing"
)

func TestInspectShardTopologyReady(t *testing.T) {
	root := t.TempDir()
	mustWritePortTest(t, filepath.Join(root, "cluster.ini"), "[SHARD]\nshard_enabled = true\nbind_ip = 127.0.0.1\nmaster_ip = 127.0.0.1\nmaster_port = 10888\ncluster_key = hidden-value\n")
	mustWritePortTest(t, filepath.Join(root, "Master", "server.ini"), "[SHARD]\nis_master = true\n")
	mustWritePortTest(t, filepath.Join(root, "Caves", "server.ini"), "[SHARD]\nis_master = false\n")
	value, err := InspectShardTopology(root)
	if err != nil {
		t.Fatal(err)
	}
	if !value.Ready || !value.HasCaves || value.MasterPort != 10888 {
		t.Fatalf("unexpected topology: %+v", value)
	}
}

func TestInspectShardTopologyNeverExposesClusterKey(t *testing.T) {
	root := t.TempDir()
	mustWritePortTest(t, filepath.Join(root, "cluster.ini"), "[SHARD]\nshard_enabled = true\nbind_ip = 127.0.0.1\nmaster_ip = 127.0.0.1\nmaster_port = 10888\ncluster_key = super-secret\n")
	mustWritePortTest(t, filepath.Join(root, "Master", "server.ini"), "[SHARD]\nis_master = true\n")
	mustWritePortTest(t, filepath.Join(root, "Caves", "server.ini"), "[SHARD]\nis_master = false\n")
	value, err := InspectShardTopology(root)
	if err != nil {
		t.Fatal(err)
	}
	if !value.ClusterKeyConfigured {
		t.Fatal("expected key configured flag")
	}
	if got := value.Summary(); got == "super-secret" {
		t.Fatal("cluster key must never be exposed")
	}
}
