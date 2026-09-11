package platformhttp

import "testing"

func TestRejectsUnsafeSchemesBeforeLaunch(t *testing.T) {
	for _, value := range []string{"javascript:alert(1)", "file:///tmp/a", "steam://validate/1", ""} {
		if err := OpenExternalURL(value); err == nil {
			t.Fatalf("expected %q to be rejected", value)
		}
	}
}
