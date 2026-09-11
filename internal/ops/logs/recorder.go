package loghub

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type operationLine struct {
	TS       string `json:"ts"`
	Level    string `json:"level"`
	Category string `json:"category"`
	Source   string `json:"source"`
	Action   string `json:"action"`
	Target   string `json:"target,omitempty"`
	Result   string `json:"result,omitempty"`
	Detail   string `json:"detail,omitempty"`
	GameID   string `json:"gameId,omitempty"`
	Instance string `json:"instance,omitempty"`
}

// Recorder 采用有界队列和后台 Buffered I/O 写操作审计，业务线程不会等待每一次磁盘刷写。
type Recorder struct {
	root          string
	flushInterval time.Duration
	queue         chan OperationRecord
	stop          chan struct{}
	done          chan struct{}
	once          sync.Once
}

func NewRecorder(root string, flushIntervalMS int) *Recorder {
	if flushIntervalMS <= 0 {
		flushIntervalMS = 500
	}
	r := &Recorder{
		root:          filepath.Clean(root),
		flushInterval: time.Duration(flushIntervalMS) * time.Millisecond,
		queue:         make(chan OperationRecord, 256),
		stop:          make(chan struct{}),
		done:          make(chan struct{}),
	}
	go r.run()
	return r
}

func (r *Recorder) Record(value OperationRecord) {
	value.Detail = redactOperationText(value.Detail)
	value.Target = redactOperationText(value.Target)
	select {
	case r.queue <- value:
	default:
		// 日志审计不能阻塞服务器生命周期；队列满时宁可丢一条审计，也不能拖死 Core。
	}
}

func (r *Recorder) Close() {
	r.once.Do(func() { close(r.stop) })
	<-r.done
}

func (r *Recorder) run() {
	defer close(r.done)
	var file *os.File
	var writer *bufio.Writer
	var currentDay string
	closeCurrent := func() {
		if writer != nil {
			_ = writer.Flush()
		}
		if file != nil {
			_ = file.Close()
		}
		writer, file, currentDay = nil, nil, ""
	}
	defer closeCurrent()

	openForToday := func() bool {
		day := time.Now().Format("2006-01-02")
		if writer != nil && currentDay == day {
			return true
		}
		closeCurrent()
		if err := os.MkdirAll(r.root, 0o755); err != nil {
			return false
		}
		path := filepath.Join(r.root, day+".log")
		opened, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
		if err != nil {
			return false
		}
		file = opened
		writer = bufio.NewWriterSize(opened, 64*1024)
		currentDay = day
		return true
	}
	write := func(value OperationRecord) {
		if !openForToday() {
			return
		}
		level := normalizeLevel(value.Level)
		if level == "" {
			level = "info"
		}
		line := operationLine{
			TS:       time.Now().Format(time.RFC3339),
			Level:    level,
			Category: "operation",
			Source:   strings.TrimSpace(value.Source),
			Action:   strings.TrimSpace(value.Action),
			Target:   strings.TrimSpace(value.Target),
			Result:   strings.TrimSpace(value.Result),
			Detail:   strings.TrimSpace(value.Detail),
			GameID:   strings.TrimSpace(value.GameID),
			Instance: strings.TrimSpace(value.Instance),
		}
		payload, err := json.Marshal(line)
		if err == nil {
			_, _ = writer.Write(payload)
			_ = writer.WriteByte('\n')
		}
	}

	ticker := time.NewTicker(r.flushInterval)
	defer ticker.Stop()
	for {
		select {
		case value := <-r.queue:
			write(value)
		case <-ticker.C:
			if writer != nil {
				_ = writer.Flush()
			}
		case <-r.stop:
			for {
				select {
				case value := <-r.queue:
					write(value)
				default:
					if writer != nil {
						_ = writer.Flush()
					}
					return
				}
			}
		}
	}
}

func redactOperationText(value string) string {
	// 审计记录只接收我们控制的安全摘要。再做一层关键词防护，避免未来误把秘密原文塞入 Detail。
	lower := strings.ToLower(value)
	for _, key := range []string{"token=", "password=", "api_key=", "apikey=", "authorization:", "bearer "} {
		if strings.Contains(lower, key) {
			return "[已脱敏：内容包含敏感凭据标记]"
		}
	}
	if len(value) > 1024 {
		return value[:1024] + "…"
	}
	return value
}
