package platformsecurity

import "testing"

func TestVaultRoundTrip(t *testing.T) {
	vault := NewVault(t.TempDir())
	if err := vault.Put("xiaoyu:model:test", "sk-demo-secret"); err != nil {
		t.Fatal(err)
	}
	if !vault.Exists("xiaoyu:model:test") {
		t.Fatal("secret missing")
	}
	value, err := vault.Get("xiaoyu:model:test")
	if err != nil {
		t.Fatal(err)
	}
	if value != "sk-demo-secret" {
		t.Fatalf("unexpected value: %q", value)
	}
	if err := vault.Delete("xiaoyu:model:test"); err != nil {
		t.Fatal(err)
	}
	if vault.Exists("xiaoyu:model:test") {
		t.Fatal("secret survived delete")
	}
}
