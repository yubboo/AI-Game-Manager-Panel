package steam

import (
	"os"
	"path/filepath"
	"testing"
)

func TestContentLogCursorReadsOnlyAppendedLines(t *testing.T) {
	root := t.TempDir()
	logDir := filepath.Join(root, "logs")
	if err := os.MkdirAll(logDir, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(logDir, "content_log.txt")
	if err := os.WriteFile(path, []byte("old line\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	cursor := NewContentLogCursor(root)
	file, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := file.WriteString("new one\r\nnew two\n"); err != nil {
		t.Fatal(err)
	}
	file.Close()

	lines, err := cursor.ReadNew()
	if err != nil {
		t.Fatal(err)
	}
	if len(lines) != 2 || lines[0] != "new one" || lines[1] != "new two" {
		t.Fatalf("unexpected lines: %#v", lines)
	}
}

func TestContentLogCursorHandlesTruncate(t *testing.T) {
	root := t.TempDir()
	logDir := filepath.Join(root, "logs")
	if err := os.MkdirAll(logDir, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(logDir, "content_log.txt")
	if err := os.WriteFile(path, []byte("01234567890123456789\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cursor := NewContentLogCursor(root)
	if err := os.WriteFile(path, []byte("rotated\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	lines, err := cursor.ReadNew()
	if err != nil {
		t.Fatal(err)
	}
	if len(lines) != 1 || lines[0] != "rotated" {
		t.Fatalf("unexpected lines: %#v", lines)
	}
}
