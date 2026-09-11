package dst_test

import (
	"os"
	"path/filepath"
	"testing"

	dstdomain "github.com/yubboo/AI-Game-Manager-Panel/internal/games/dst"
	"github.com/yubboo/AI-Game-Manager-Panel/internal/games/dst/dedicated"
	clusterops "github.com/yubboo/AI-Game-Manager-Panel/internal/games/steam/dst/cluster"
	"github.com/yubboo/AI-Game-Manager-Panel/internal/games/steam/dst/preflight"
	"github.com/yubboo/AI-Game-Manager-Panel/internal/games/steam/dst/token"
)

func TestFirstServerFlowImportTokenPreflight(t *testing.T) {
	source := filepath.Join(t.TempDir(), "ClientWorld")
	write(t, filepath.Join(source, "cluster.ini"), "[GAMEPLAY]\n")
	write(t, filepath.Join(source, "Master", "server.ini"), "[SHARD]\n")
	write(t, filepath.Join(source, "Master", "save", "session", "0001"), "world-data")
	write(t, filepath.Join(source, "cluster_token.txt"), "OLD-TOKEN-MUST-NOT-COPY")

	kleiRoot := filepath.Join(t.TempDir(), "DoNotStarveTogether")
	imported, err := clusterops.Import(kleiRoot, clusterops.ImportRequest{SourcePath: source, TargetName: "My_Server"})
	if err != nil {
		t.Fatal(err)
	}
	if !imported.TokenSkipped {
		t.Fatal("expected imported client token to be skipped")
	}

	status, err := token.Save(imported.Path, "NEW-KLEI-TOKEN")
	if err != nil {
		t.Fatal(err)
	}
	cluster := dstdomain.Cluster{
		Name:         imported.Name,
		Path:         imported.Path,
		Source:       dstdomain.SaveSourceServer,
		Distribution: dstdomain.DistributionSteam,
		Shards: []dstdomain.Shard{{
			Name: "Master",
			Path: filepath.Join(imported.Path, "Master"),
		}},
	}
	result := preflight.Evaluate(preflight.Input{
		Dedicated: dedicated.Snapshot{
			Installation: dedicated.Installation{Valid: true, Bitness: 64},
			ConfDir:      dedicated.ConfDirInfo{Valid: true, KleiRoot: kleiRoot},
		},
		Cluster: &cluster,
		Wanted:  imported.Path,
		Token:   status,
	})
	if !result.Ready || result.Blockers != 0 {
		t.Fatalf("expected ready first-run flow, got %#v", result)
	}
}

func write(t *testing.T, path, value string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(value), 0o644); err != nil {
		t.Fatal(err)
	}
}
