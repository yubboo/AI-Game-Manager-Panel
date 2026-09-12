package instance

import "testing"

func TestStableIDIsDeterministicAndScopedByGame(t *testing.T) {
	a := StableID("steam.dst", "/srv/game")
	b := StableID("steam.dst", "/srv/game")
	c := StableID("minecraft.java", "/srv/game")
	if a != b || a == c || len(a) != len("inst_")+16 {
		t.Fatalf("unexpected ids: %q %q %q", a, b, c)
	}
}
