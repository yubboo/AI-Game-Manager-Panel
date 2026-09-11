package dedicated

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReadPortPlanMasterAndCaves(t *testing.T) {
	root := t.TempDir()
	mustWritePortTest(t, filepath.Join(root, "cluster.ini"), "[SHARD]\nshard_enabled = true\nmaster_port = 10888\n")
	mustWritePortTest(t, filepath.Join(root, "Master", "server.ini"), "[NETWORK]\nserver_port = 10999\n[STEAM]\nmaster_server_port = 27016\nauthentication_port = 8766\n")
	mustWritePortTest(t, filepath.Join(root, "Caves", "server.ini"), "[NETWORK]\nserver_port = 11000\n[STEAM]\nmaster_server_port = 27017\nauthentication_port = 8767\n")
	plan, err := ReadPortPlan(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Uses) != 7 {
		t.Fatalf("expected 7 port uses, got %d: %#v", len(plan.Uses), plan.Uses)
	}
	if len(plan.Collisions) != 0 {
		t.Fatalf("unexpected collisions: %#v", plan.Collisions)
	}
}

func TestReadPortPlanDetectsCollision(t *testing.T) {
	root := t.TempDir()
	mustWritePortTest(t, filepath.Join(root, "cluster.ini"), "[SHARD]\nshard_enabled = true\nmaster_port = 10888\n")
	mustWritePortTest(t, filepath.Join(root, "Master", "server.ini"), "[NETWORK]\nserver_port = 10999\n")
	mustWritePortTest(t, filepath.Join(root, "Caves", "server.ini"), "[NETWORK]\nserver_port = 10999\n")
	plan, err := ReadPortPlan(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Collisions) != 1 || plan.Collisions[0].Port != 10999 {
		t.Fatalf("expected 10999 collision, got %#v", plan.Collisions)
	}
}

func mustWritePortTest(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
