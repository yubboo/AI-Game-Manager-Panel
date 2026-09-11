package steam

import (
	"bufio"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// ContentLogCursor tracks appended lines in Steam's logs/content_log.txt without
// rereading the whole file on every maintenance poll. It is intentionally small
// and allocation-light because validation/install monitoring runs once per second.
type ContentLogCursor struct {
	Path   string
	Offset int64
	tail   string
}

func NewContentLogCursor(steamRoot string) ContentLogCursor {
	path := filepath.Join(steamRoot, "logs", "content_log.txt")
	cursor := ContentLogCursor{Path: path}
	if info, err := os.Stat(path); err == nil && !info.IsDir() {
		cursor.Offset = info.Size()
	}
	return cursor
}

func (c *ContentLogCursor) ReadNew() ([]string, error) {
	if strings.TrimSpace(c.Path) == "" {
		return nil, errors.New("Steam content_log.txt path is empty")
	}
	file, err := os.Open(c.Path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return nil, err
	}
	// Steam may rotate/truncate the log while a task is running.
	if info.Size() < c.Offset {
		c.Offset = 0
		c.tail = ""
	}
	if _, err := file.Seek(c.Offset, io.SeekStart); err != nil {
		return nil, err
	}

	reader := bufio.NewReaderSize(file, 64*1024)
	chunk, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}
	c.Offset += int64(len(chunk))
	if len(chunk) == 0 {
		return nil, nil
	}

	text := c.tail + strings.ReplaceAll(string(chunk), "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")
	parts := strings.Split(text, "\n")
	if !strings.HasSuffix(text, "\n") {
		c.tail = parts[len(parts)-1]
		parts = parts[:len(parts)-1]
	} else {
		c.tail = ""
	}
	lines := make([]string, 0, len(parts))
	for _, line := range parts {
		if line = strings.TrimSpace(line); line != "" {
			lines = append(lines, line)
		}
	}
	return lines, nil
}
