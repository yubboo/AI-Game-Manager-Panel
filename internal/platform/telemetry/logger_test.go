package telemetry

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestLoggerAcceptsTypedFields(t *testing.T) {
	logger := New(filepath.Join(t.TempDir(), "agmp.log"))
	logger.Info("settings saved", "theme", "dark", "debug", true, "count", 3)

	lines := logger.ReadRecent(10)
	if len(lines) != 1 {
		t.Fatalf("expected 1 log line, got %d", len(lines))
	}
	if !strings.Contains(lines[0], `theme="dark"`) {
		t.Fatalf("theme field missing: %s", lines[0])
	}
	if !strings.Contains(lines[0], `debug="true"`) {
		t.Fatalf("bool field missing: %s", lines[0])
	}
	if !strings.Contains(lines[0], `count="3"`) {
		t.Fatalf("numeric field missing: %s", lines[0])
	}
}
