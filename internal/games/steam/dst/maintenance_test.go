package dst

import "testing"

func TestRequiresStoppedRuntimeForSteamMaintenance(t *testing.T) {
	if RequiresStoppedRuntimeForSteamMaintenance(GameAppID) {
		t.Fatal("DST client validation must not be tied to Dedicated Server runtime")
	}
	if !RequiresStoppedRuntimeForSteamMaintenance(ServerAppID) {
		t.Fatal("DST Dedicated Server maintenance must require stopped runtime")
	}
	if RequiresStoppedRuntimeForSteamMaintenance(570) {
		t.Fatal("DST policy must not affect unrelated Steam AppIDs")
	}
}
