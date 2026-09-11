package dst

import (
	"os"
	"path/filepath"
	"testing"
)

func TestListClustersRequiresClusterIniAndShard(t *testing.T) {
	root := t.TempDir()
	valid := filepath.Join(root, "Cluster_1")
	if err := os.MkdirAll(filepath.Join(valid, "Master"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(valid, "cluster.ini"), []byte("[NETWORK]\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(valid, "Master", "server.ini"), []byte("[SHARD]\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	partial := filepath.Join(root, "Cluster_2")
	if err := os.MkdirAll(partial, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(partial, "cluster.ini"), []byte("[NETWORK]\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	clusters := listClusters(root)
	if len(clusters) != 1 || clusters[0] != valid {
		t.Fatalf("expected only valid cluster, got %#v", clusters)
	}
}

func TestBuildClusterKeepsOriginalPaths(t *testing.T) {
	root := t.TempDir()
	clusterPath := filepath.Join(root, "Cluster_A")
	master := filepath.Join(clusterPath, "Master")
	if err := os.MkdirAll(master, 0o755); err != nil {
		t.Fatal(err)
	}
	for path, body := range map[string]string{
		filepath.Join(clusterPath, "cluster.ini"):       "[NETWORK]\n",
		filepath.Join(clusterPath, "cluster_token.txt"): "token",
		filepath.Join(clusterPath, "adminlist.txt"):     "KU_test",
		filepath.Join(master, "server.ini"):             "[SHARD]\n",
		filepath.Join(master, "leveldataoverride.lua"):  "return {}",
	} {
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	cluster := buildCluster(clusterPath, SaveSourceServer, DistributionSteam)
	if cluster.TokenPath == "" || cluster.AdminListPath == "" || len(cluster.Shards) != 1 {
		t.Fatalf("unexpected cluster: %#v", cluster)
	}
	if cluster.Shards[0].LevelDataPath == "" {
		t.Fatal("leveldataoverride.lua should be discovered")
	}
}
