package runtime

import (
	"testing"

	dstdomain "github.com/yubboo/AI-Game-Manager-Panel/internal/games/dst"
	"github.com/yubboo/AI-Game-Manager-Panel/internal/games/dst/dedicated"
)

func TestFindClusterUsesFullPathNotDisplayName(t *testing.T) {
	clusters := []dstdomain.Cluster{
		{Name: "Cluster_1", Path: `D:\Klei\DoNotStarveTogether\Cluster_1`},
		{Name: "Cluster_1", Path: `E:\Klei\DoNotStarveTogether\Cluster_1`},
	}
	got, ok := findCluster(clusters, `E:\Klei\DoNotStarveTogether\Cluster_1`)
	if !ok || got.Path != clusters[1].Path {
		t.Fatalf("wrong cluster selected: %+v ok=%v", got, ok)
	}
}

func TestFindMasterShardIsCaseInsensitive(t *testing.T) {
	cluster := dstdomain.Cluster{Shards: []dstdomain.Shard{{Name: "Caves"}, {Name: "master"}}}
	got, ok := findMasterShard(cluster)
	if !ok || got.Name != "master" {
		t.Fatalf("Master not resolved: %+v ok=%v", got, ok)
	}
}

func TestOverallRuntimeWithMasterAndCaves(t *testing.T) {
	master := LookupResult{Found: true, Process: ProcessSnapshot{Status: dedicated.StatusRunning, WorldReady: true}}
	caves := LookupResult{Found: true, Process: ProcessSnapshot{Status: dedicated.StatusRunning, WorldReady: true}}
	if got := overallRuntime(master, caves, true); got != "running" {
		t.Fatalf("expected running, got %s", got)
	}
	caves.Process.WorldReady = false
	if got := overallRuntime(master, caves, true); got != "starting" {
		t.Fatalf("expected starting while Caves is not ready, got %s", got)
	}
	caves = LookupResult{}
	if got := overallRuntime(master, caves, true); got != "partial" {
		t.Fatalf("expected partial when Caves is missing, got %s", got)
	}
}

func TestOnlyDSTProcessesAreEligibleForForceCleanup(t *testing.T) {
	if !isDSTDedicatedProcess("dontstarve_dedicated_server_nullrenderer_x64.exe", `D:\\steam\\dontstarve_dedicated_server_nullrenderer_x64.exe`) {
		t.Fatal("DST dedicated process should be recognized")
	}
	if isDSTDedicatedProcess("chrome.exe", `C:\\Program Files\\Google\\Chrome\\Application\\chrome.exe`) {
		t.Fatal("unrelated process must never be eligible for force cleanup")
	}
}
