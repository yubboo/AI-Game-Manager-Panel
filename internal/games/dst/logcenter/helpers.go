package logcenter

import (
	"bytes"
	"io"
	"os"
	"strings"
)

func makeLogLine(number uint64, text string) LogLine {
	lower := strings.ToLower(text)
	return LogLine{LineNumber: number, Text: text, Level: classifyLevel(lower), Category: classifyCategory(lower)}
}

func classifyLevel(lower string) string {
	if strings.Contains(lower, "error:") || strings.Contains(lower, "[error]") || strings.Contains(lower, "failed to run code") || strings.Contains(lower, "fatal") {
		return "error"
	}
	if strings.Contains(lower, "[warning]") || strings.Contains(lower, "[warn]") || strings.Contains(lower, "warning:") || strings.Contains(lower, "could not confirm port") {
		return "warning"
	}
	return "info"
}

func classifyCategory(lower string) string {
	switch {
	case strings.Contains(lower, "steam") || strings.Contains(lower, "account communication"):
		return "steam"
	case strings.Contains(lower, "mod") || strings.Contains(lower, "workshop"):
		return "mod"
	case strings.Contains(lower, "network") || strings.Contains(lower, "port") || strings.Contains(lower, "firewall") || strings.Contains(lower, "shard"):
		return "network"
	case strings.Contains(lower, "token") || strings.Contains(lower, "permission") || strings.Contains(lower, "adminlist") || strings.Contains(lower, "whitelist") || strings.Contains(lower, "blocklist"):
		return "security"
	case strings.Contains(lower, "save") || strings.Contains(lower, "session") || strings.Contains(lower, "world"):
		return "world"
	default:
		return "general"
	}
}

func tailLines(path string, limit int) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	stat, err := file.Stat()
	if err != nil {
		return nil, err
	}
	if stat.Size() == 0 {
		return []string{}, nil
	}
	const chunkSize int64 = 64 * 1024
	position := stat.Size()
	chunks := make([][]byte, 0, 8)
	newlineCount := 0
	total := 0
	for position > 0 && newlineCount <= limit {
		readSize := chunkSize
		if position < readSize {
			readSize = position
		}
		position -= readSize
		chunk := make([]byte, readSize)
		if _, err := file.ReadAt(chunk, position); err != nil && err != io.EOF {
			return nil, err
		}
		chunks = append(chunks, chunk)
		newlineCount += bytes.Count(chunk, []byte{'\n'})
		total += len(chunk)
		if total > 16*1024*1024 && newlineCount == 0 {
			break
		}
	}
	buffer := make([]byte, 0, total)
	for i := len(chunks) - 1; i >= 0; i-- {
		buffer = append(buffer, chunks[i]...)
	}
	parts := strings.Split(strings.ReplaceAll(string(buffer), "\r\n", "\n"), "\n")
	if len(parts) > 0 && parts[len(parts)-1] == "" {
		parts = parts[:len(parts)-1]
	}
	if len(parts) > limit {
		parts = parts[len(parts)-limit:]
	}
	return parts, nil
}
