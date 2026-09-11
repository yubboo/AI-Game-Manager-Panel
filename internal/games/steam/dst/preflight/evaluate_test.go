package preflight

import (
	"os"
	"path/filepath"
	"testing"

	dstdomain "github.com/yubboo/AI-Game-Manager-Panel/internal/games/dst"
	"github.com/yubboo/AI-Game-Manager-Panel/internal/games/dst/dedicated"
	"github.com/yubboo/AI-Game-Manager-Panel/internal/games/steam/dst/token"
)

func TestTokenBlocksStart(t *testing.T) {
	clusterPath := t.TempDir()
	_ = os.WriteFile(filepath.Join(clusterPath, "cluster.ini"), []byte("x"), 0o644)
	_ = os.MkdirAll(filepath.Join(clusterPath, "Master"), 0o755)
	_ = os.WriteFile(filepath.Join(clusterPath, "Master", "server.ini"), []byte("x"), 0o644)
	cluster := dstdomain.Cluster{Name: "Cluster_1", Path: clusterPath, Source: dstdomain.SaveSourceServer, Distribution: dstdomain.DistributionSteam, Shards: []dstdomain.Shard{{Name: "Master", Path: filepath.Join(clusterPath, "Master")}}}
	result := Evaluate(Input{Dedicated: dedicated.Snapshot{Installation: dedicated.Installation{Valid: true, Bitness: 64}, ConfDir: dedicated.ConfDirInfo{Valid: true, KleiRoot: filepath.Dir(clusterPath)}}, Cluster: &cluster, Wanted: clusterPath, Token: token.Status{State: token.StateMissing}})
	if result.Ready || result.Blockers != 1 {
		t.Fatalf("expected token blocker, got %#v", result)
	}
}
