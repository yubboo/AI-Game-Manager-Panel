package telemetry

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type Logger struct {
	mu   sync.Mutex
	path string
}

func New(path string) *Logger {
	return &Logger{path: path}
}

func (l *Logger) Info(message string, fields ...any)  { l.write("INFO", message, fields...) }
func (l *Logger) Warn(message string, fields ...any)  { l.write("WARN", message, fields...) }
func (l *Logger) Error(message string, fields ...any) { l.write("ERROR", message, fields...) }

func (l *Logger) write(level, message string, fields ...any) {
	l.mu.Lock()
	defer l.mu.Unlock()

	line := fmt.Sprintf("%s [%s] %s", time.Now().Format("2006-01-02 15:04:05"), level, message)
	for i := 0; i+1 < len(fields); i += 2 {
		key := fmt.Sprint(fields[i])
		value := fmt.Sprint(fields[i+1])
		line += fmt.Sprintf(" %s=%q", key, value)
	}
	line += "\n"

	if err := os.MkdirAll(filepath.Dir(l.path), 0o755); err != nil {
		fmt.Print(line)
		return
	}
	file, err := os.OpenFile(l.path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		fmt.Print(line)
		return
	}
	defer file.Close()
	_, _ = file.WriteString(line)
}

func (l *Logger) ReadRecent(limit int) []string {
	l.mu.Lock()
	defer l.mu.Unlock()

	file, err := os.Open(l.path)
	if err != nil {
		return []string{}
	}
	defer file.Close()

	lines := make([]string, 0, limit)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		lines = append(lines, line)
		if len(lines) > limit {
			lines = lines[1:]
		}
	}
	return lines
}
