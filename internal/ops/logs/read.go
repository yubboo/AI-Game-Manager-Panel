package loghub

import (
	"bufio"
	"errors"
	"io"
	"os"
	"strings"
	"time"
)

// Read 按需读取一个日志文件。默认单次 600 行，硬上限 2000 行。
func (s *Service) Read(request ReadRequest) (ReadPage, error) {
	started := time.Now()
	fileMeta, path, err := s.findByID(strings.TrimSpace(request.ID))
	if err != nil {
		return ReadPage{}, err
	}
	limit := normalizeReadLimit(request.Limit)
	direction := strings.ToLower(strings.TrimSpace(request.Direction))
	if direction == "" {
		direction = "tail"
	}

	var page ReadPage
	if direction == "tail" {
		page, err = s.readTail(fileMeta, path, request, limit)
	} else {
		page, err = s.readForward(fileMeta, path, request, limit)
	}
	if err != nil {
		return ReadPage{}, err
	}
	page.DurationMs = time.Since(started).Milliseconds()
	return page, nil
}

func normalizeReadLimit(value int) int {
	if value <= 0 {
		return defaultReadLimit
	}
	if value > maxReadLimit {
		return maxReadLimit
	}
	return value
}

func (s *Service) readForward(meta LogFile, path string, request ReadRequest, limit int) (ReadPage, error) {
	file, err := os.Open(path)
	if err != nil {
		return ReadPage{}, err
	}
	defer file.Close()
	cursor := request.Cursor
	if strings.EqualFold(request.Direction, "head") {
		cursor = 0
	}
	if cursor < 0 {
		cursor = 0
	}
	if _, err := file.Seek(cursor, io.SeekStart); err != nil {
		return ReadPage{}, err
	}
	reader := bufio.NewReaderSize(file, 128*1024)
	lineNo := request.StartLine
	if lineNo == 0 || strings.EqualFold(request.Direction, "head") {
		lineNo = 1
	}
	page := ReadPage{File: meta, Lines: make([]LogLine, 0, limit), NextCursor: cursor, NextLine: lineNo}
	for len(page.Lines) < limit {
		text, readErr := reader.ReadString('\n')
		if len(text) > 0 {
			text = strings.TrimSuffix(strings.TrimSuffix(text, "\n"), "\r")
			line := classifyLine(lineNo, text)
			page.Scanned++
			if lineMatches(line, request.Query, request.Level, request.Category) {
				page.Lines = append(page.Lines, line)
				page.Matched++
			}
			lineNo++
		}
		if readErr != nil {
			if errors.Is(readErr, io.EOF) {
				page.EOF = true
				break
			}
			return ReadPage{}, readErr
		}
	}
	pos, err := file.Seek(0, io.SeekCurrent)
	if err == nil {
		pos -= int64(reader.Buffered())
		page.NextCursor = pos
	}
	page.NextLine = lineNo
	return page, nil
}

func (s *Service) readTail(meta LogFile, path string, request ReadRequest, limit int) (ReadPage, error) {
	// 有内容筛选时需要保证返回“最后 N 条匹配记录”，因此流式扫描整文件并只保留有限环形结果。
	if strings.TrimSpace(request.Query) != "" || (strings.TrimSpace(request.Level) != "" && !strings.EqualFold(request.Level, "all")) || (strings.TrimSpace(request.Category) != "" && !strings.EqualFold(request.Category, "all")) {
		return readFilteredTail(meta, path, request, limit)
	}
	lines, err := tailRawLines(path, limit)
	if err != nil {
		return ReadPage{}, err
	}
	start := uint64(1)
	if meta.LineCount > uint64(len(lines)) {
		start = meta.LineCount - uint64(len(lines)) + 1
	}
	result := make([]LogLine, 0, len(lines))
	for i, text := range lines {
		result = append(result, classifyLine(start+uint64(i), text))
	}
	stat, _ := os.Stat(path)
	cursor := int64(0)
	if stat != nil {
		cursor = stat.Size()
	}
	return ReadPage{File: meta, Lines: result, NextCursor: cursor, NextLine: meta.LineCount + 1, EOF: true, Scanned: uint64(len(lines)), Matched: uint64(len(lines))}, nil
}

func readFilteredTail(meta LogFile, path string, request ReadRequest, limit int) (ReadPage, error) {
	file, err := os.Open(path)
	if err != nil {
		return ReadPage{}, err
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 64*1024), 4*1024*1024)
	ring := make([]LogLine, 0, limit)
	var lineNo uint64
	var matched uint64
	for scanner.Scan() {
		lineNo++
		line := classifyLine(lineNo, scanner.Text())
		if !lineMatches(line, request.Query, request.Level, request.Category) {
			continue
		}
		matched++
		if len(ring) < limit {
			ring = append(ring, line)
		} else {
			copy(ring, ring[1:])
			ring[len(ring)-1] = line
		}
	}
	if err := scanner.Err(); err != nil {
		return ReadPage{}, err
	}
	stat, _ := file.Stat()
	cursor := int64(0)
	if stat != nil {
		cursor = stat.Size()
	}
	return ReadPage{File: meta, Lines: ring, NextCursor: cursor, NextLine: lineNo + 1, EOF: true, Scanned: lineNo, Matched: matched}, nil
}

func tailRawLines(path string, limit int) ([]string, error) {
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
	newlines := 0
	for position > 0 && newlines <= limit {
		size := chunkSize
		if position < size {
			size = position
		}
		position -= size
		chunk := make([]byte, size)
		if _, err := file.ReadAt(chunk, position); err != nil && !errors.Is(err, io.EOF) {
			return nil, err
		}
		chunks = append(chunks, chunk)
		for _, b := range chunk {
			if b == '\n' {
				newlines++
			}
		}
	}
	var builder strings.Builder
	for i := len(chunks) - 1; i >= 0; i-- {
		builder.Write(chunks[i])
	}
	parts := strings.Split(strings.ReplaceAll(builder.String(), "\r\n", "\n"), "\n")
	if len(parts) > 0 && parts[len(parts)-1] == "" {
		parts = parts[:len(parts)-1]
	}
	if len(parts) > limit {
		parts = parts[len(parts)-limit:]
	}
	return parts, nil
}

func (s *Service) findByID(id string) (LogFile, string, error) {
	if id == "" {
		return LogFile{}, "", ErrLogNotFound
	}
	items, err := s.scanAll()
	if err != nil {
		return LogFile{}, "", err
	}
	for _, item := range items {
		if item.ID != id {
			continue
		}
		path, err := s.safePath(item.RelativePath)
		if err != nil {
			return LogFile{}, "", err
		}
		return item, path, nil
	}
	return LogFile{}, "", ErrLogNotFound
}
