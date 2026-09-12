package game

import "testing"

func TestPlannedGamePackIsNotExecutable(t *testing.T) {
	if (Pack{State: PackPlanned}).Executable() {
		t.Fatal("planned Game Pack must never be presented as executable")
	}
	if !(Pack{State: PackSupported}).Executable() {
		t.Fatal("supported Game Pack should be executable")
	}
}
