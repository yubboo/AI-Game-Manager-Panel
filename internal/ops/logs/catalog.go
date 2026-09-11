package loghub

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type dstSessionMeta struct {
	ID          string `json:"id"`
	ClusterName string `json:"clusterName"`
	ClusterPath string `json:"clusterPath"`
	ShardName   string `json:"shardName"`
	Status      string `json:"status"`
	StartedAt   int64  `json:"startedAt"`
	EndedAt     int64  `json:"endedAt"`
	LineCount   uint64 `json:"lineCount"`
}

// Catalog 扫描 log 根目录并按条件返回元数据。
// 目录刷新不会读取日志正文到内存，只为真实行数做流式计数并缓存结果。
func (s *Service) Catalog(request CatalogRequest) (CatalogPage, error) {
	started := time.Now()
	if err := s.ensureRoot(); err != nil {
		return CatalogPage{}, err
	}
	items, err := s.scanAll()
	if err != nil {
		return CatalogPage{}, err
	}
	filtered := make([]LogFile, 0, len(items))
	for _, item := range items {
		if matchesCatalog(item, request) {
			filtered = append(filtered, item)
		}
	}
	sort.Slice(filtered, func(i, j int) bool {
		if filtered[i].ModifiedAt == filtered[j].ModifiedAt {
			return filtered[i].RelativePath < filtered[j].RelativePath
		}
		return filtered[i].ModifiedAt > filtered[j].ModifiedAt
	})

	summary := Summary{}
	for _, item := range filtered {
		summary.Files++
		summary.Lines += item.LineCount
		summary.Bytes += item.ByteSize
		if item.Active {
			summary.ActiveFiles++
		}
	}

	limit := request.Limit
	if limit <= 0 {
		limit = defaultCatalogLimit
	}
	if limit > maxCatalogLimit {
		limit = maxCatalogLimit
	}
	offset := request.Offset
	if offset < 0 {
		offset = 0
	}
	if offset > len(filtered) {
		offset = len(filtered)
	}
	end := offset + limit
	if end > len(filtered) {
		end = len(filtered)
	}
	pageItems := append([]LogFile(nil), filtered[offset:end]...)
	return CatalogPage{
		Items:      pageItems,
		Summary:    summary,
		Total:      len(filtered),
		Offset:     offset,
		Limit:      limit,
		DurationMs: time.Since(started).Milliseconds(),
	}, nil
}

func (s *Service) scanAll() ([]LogFile, error) {
	items := make([]LogFile, 0, 64)
	err := filepath.WalkDir(s.root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil || entry == nil {
			// 单个历史文件权限异常不能让整个日志中心不可用。
			return nil
		}
		if entry.IsDir() {
			if samePath(path, s.exportRoot) {
				return filepath.SkipDir
			}
			return nil
		}
		if !isLogFile(entry.Name()) {
			return nil
		}
		stat, err := entry.Info()
		if err != nil {
			return nil
		}
		rel, err := filepath.Rel(s.root, path)
		if err != nil {
			return nil
		}
		value := s.describeFile(path, filepath.ToSlash(rel), stat)
		items = append(items, value)
		return nil
	})
	return items, err
}

func isLogFile(name string) bool {
	ext := strings.ToLower(filepath.Ext(name))
	return ext == ".log" || ext == ".txt" || ext == ".jsonl"
}

func (s *Service) describeFile(path, relative string, stat os.FileInfo) LogFile {
	value := LogFile{
		ID:           logID(relative),
		Name:         filepath.Base(path),
		RelativePath: relative,
		Status:       "history",
		ModifiedAt:   stat.ModTime().Unix(),
		ByteSize:     stat.Size(),
	}
	parts := strings.Split(strings.Trim(filepath.ToSlash(relative), "/"), "/")
	s.classifyLogPath(&value, parts)
	// 当前 Core 日志与当天操作审计仍可能继续写入，默认视为活动文件。
	if value.Source == "agmp" {
		value.Active = true
		value.Status = "running"
	}
	if value.Source == "operations" && s.recorder != nil && strings.EqualFold(value.Name, time.Now().Format("2006-01-02")+".log") {
		value.Active = true
		value.Status = "running"
	}

	// DST Session 的同名 .json 是事实元数据，优先使用其中的 Cluster/Shard/状态/时间。
	if meta, ok := readDSTMeta(strings.TrimSuffix(path, filepath.Ext(path)) + ".json"); ok {
		value.GameID = "steam.dst"
		value.GameName = "饥荒联机版"
		value.Kind = "game"
		value.Source = "dst"
		value.SourceLabel = "饥荒联机版"
		value.InstanceID = strings.TrimSpace(meta.ClusterPath)
		value.InstanceName = strings.TrimSpace(meta.ClusterName)
		value.Shard = strings.TrimSpace(meta.ShardName)
		value.Status = strings.TrimSpace(meta.Status)
		value.StartedAt = meta.StartedAt
		value.EndedAt = meta.EndedAt
		value.Active = strings.EqualFold(meta.Status, "running") || strings.EqualFold(meta.Status, "starting") || strings.EqualFold(meta.Status, "stopping")
	}
	value.LineCount = s.realLineCount(path, stat)
	return value
}

func (s *Service) classifyLogPath(value *LogFile, parts []string) {
	if len(parts) == 0 {
		value.Source = "unknown"
		value.SourceLabel = "其他日志"
		value.Kind = "system"
		return
	}

	// 日志目录名称由 configs/logging.json 决定。这里按“路径前缀”匹配，
	// 因此管理员即使把某一类日志放到 system/core 这类子目录，也不会破坏分类。
	matchDir := func(configured, legacy string) (bool, int) {
		configuredParts := splitConfiguredDir(configured)
		if len(configuredParts) > 0 && len(parts) >= len(configuredParts) {
			matched := true
			for i, part := range configuredParts {
				if !strings.EqualFold(parts[i], part) {
					matched = false
					break
				}
			}
			if matched {
				return true, len(configuredParts)
			}
		}
		if legacy != "" && strings.EqualFold(parts[0], legacy) {
			return true, 1
		}
		return false, 0
	}

	if ok, _ := matchDir(s.coreDir, "agmp"); ok {
		value.Source, value.SourceLabel, value.Kind = "agmp", "AI Game Manager Panel Core", "system"
	} else if ok, _ := matchDir(s.operationDir, "operations"); ok {
		value.Source, value.SourceLabel, value.Kind = "operations", "操作记录", "operation"
	} else if ok, _ := matchDir(s.auditDir, "audit"); ok {
		value.Source, value.SourceLabel, value.Kind = "audit", "安全审计", "audit"
	} else if ok, _ := matchDir(s.aiDir, "ai"); ok {
		value.Source, value.SourceLabel, value.Kind = "ai", "小鱼", "ai"
	} else if ok, _ := matchDir(s.steamDir, "steam"); ok {
		value.Source, value.SourceLabel, value.Kind = "steam", "Steam", "steam"
	} else if ok, _ := matchDir(s.nodesDir, "nodes"); ok {
		value.Source, value.SourceLabel, value.Kind = "nodes", "节点", "node"
	} else if strings.EqualFold(parts[0], "dst") { // 兼容 0.1.41 之前的 log/dst 目录。
		value.Source, value.SourceLabel, value.Kind = "dst", "饥荒联机版", "game"
		value.GameID, value.GameName = "steam.dst", "饥荒联机版"
		inferDSTParts(value, parts[1:])
	} else if ok, consumed := matchDir(s.gamesDir, "games"); ok {
		value.Kind = "game"
		remaining := parts[consumed:]
		if len(remaining) > 0 {
			game := strings.ToLower(remaining[0])
			value.Source = game
			value.GameID = game
			value.GameName = game
			value.SourceLabel = game
			if game == "dst" || game == "steam.dst" {
				value.Source, value.GameID, value.GameName, value.SourceLabel = "dst", "steam.dst", "饥荒联机版", "饥荒联机版"
				inferDSTParts(value, remaining[1:])
			}
		}
	} else {
		first := strings.ToLower(parts[0])
		// 兼容旧版根目录 agmp.log。
		if strings.EqualFold(value.Name, "agmp.log") {
			value.Source, value.SourceLabel, value.Kind = "agmp", "AI Game Manager Panel Core", "system"
		} else {
			value.Source, value.SourceLabel, value.Kind = first, first, "system"
		}
	}
	if value.SourceLabel == "" {
		value.SourceLabel = value.Source
	}
	if value.Kind == "" {
		value.Kind = "system"
	}
}

func splitConfiguredDir(value string) []string {
	cleaned := strings.Trim(filepath.ToSlash(filepath.Clean(value)), "/")
	if cleaned == "" || cleaned == "." {
		return nil
	}
	parts := strings.Split(cleaned, "/")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" && part != "." {
			result = append(result, part)
		}
	}
	return result
}

func inferDSTParts(value *LogFile, parts []string) {
	if len(parts) > 0 {
		value.InstanceName = strings.TrimSpace(strings.Split(parts[0], "-")[0])
	}
	for _, part := range parts {
		if strings.EqualFold(part, "Master") || strings.EqualFold(part, "Caves") {
			value.Shard = part
			break
		}
	}
}

func readDSTMeta(path string) (dstSessionMeta, bool) {
	payload, err := os.ReadFile(path)
	if err != nil {
		return dstSessionMeta{}, false
	}
	var value dstSessionMeta
	if err := json.Unmarshal(payload, &value); err != nil {
		return dstSessionMeta{}, false
	}
	if strings.TrimSpace(value.ID) == "" && strings.TrimSpace(value.ClusterName) == "" {
		return dstSessionMeta{}, false
	}
	return value, true
}

func (s *Service) realLineCount(path string, stat os.FileInfo) uint64 {
	mod := stat.ModTime().UnixNano()
	s.cacheMu.Lock()
	cached, ok := s.lineCache[path]
	s.cacheMu.Unlock()
	if ok && cached.Size == stat.Size() && cached.ModifiedAt == mod {
		return cached.Lines
	}

	// 如果文件只是在尾部增长，从上次 size 继续数，避免长期服务器每次刷新都重扫整个日志。
	if ok && stat.Size() > cached.Size && cached.Size >= 0 && cached.EndedWithNewline {
		if file, err := os.Open(path); err == nil {
			if _, err = file.Seek(cached.Size, io.SeekStart); err == nil {
				added, endedWithNewline, countErr := countReaderLines(file)
				_ = file.Close()
				if countErr == nil {
					value := lineCacheEntry{Size: stat.Size(), ModifiedAt: mod, Lines: cached.Lines + added, EndedWithNewline: endedWithNewline}
					s.cacheMu.Lock()
					s.lineCache[path] = value
					s.cacheMu.Unlock()
					return value.Lines
				}
			} else {
				_ = file.Close()
			}
		}
	}

	file, err := os.Open(path)
	if err != nil {
		return 0
	}
	lines, endedWithNewline, err := countReaderLines(file)
	_ = file.Close()
	if err != nil {
		return 0
	}
	value := lineCacheEntry{Size: stat.Size(), ModifiedAt: mod, Lines: lines, EndedWithNewline: endedWithNewline}
	s.cacheMu.Lock()
	s.lineCache[path] = value
	s.cacheMu.Unlock()
	return lines
}

func countReaderLines(reader io.Reader) (uint64, bool, error) {
	buf := bufio.NewReaderSize(reader, 256*1024)
	var lines uint64
	var sawBytes bool
	var last byte
	chunk := make([]byte, 256*1024)
	for {
		n, err := buf.Read(chunk)
		if n > 0 {
			sawBytes = true
			last = chunk[n-1]
			for _, b := range chunk[:n] {
				if b == '\n' {
					lines++
				}
			}
		}
		if err != nil {
			if err == io.EOF {
				break
			}
			return 0, false, err
		}
	}
	if sawBytes && last != '\n' {
		lines++
	}
	return lines, sawBytes && last == '\n', nil
}

func matchesCatalog(item LogFile, request CatalogRequest) bool {
	q := strings.ToLower(strings.TrimSpace(request.Query))
	if q != "" {
		hay := strings.ToLower(strings.Join([]string{item.Name, item.RelativePath, item.SourceLabel, item.GameName, item.InstanceName, item.Shard}, " "))
		if !strings.Contains(hay, q) {
			return false
		}
	}
	if !matchOptional(request.Source, item.Source) || !matchOptional(request.Kind, item.Kind) || !matchOptional(request.GameID, item.GameID) || !matchOptional(request.Shard, item.Shard) || !matchOptional(request.Status, item.Status) {
		return false
	}
	instance := strings.TrimSpace(request.InstanceID)
	if instance != "" && instance != "all" && !strings.EqualFold(instance, item.InstanceID) && !strings.EqualFold(instance, item.InstanceName) {
		return false
	}
	if request.DateFrom > 0 && item.ModifiedAt < request.DateFrom {
		return false
	}
	if request.DateTo > 0 && item.ModifiedAt > request.DateTo {
		return false
	}
	return true
}

func matchOptional(filter, value string) bool {
	filter = strings.TrimSpace(filter)
	return filter == "" || strings.EqualFold(filter, "all") || strings.EqualFold(filter, strings.TrimSpace(value))
}

func logID(relative string) string {
	sum := sha256.Sum256([]byte(strings.ToLower(filepath.ToSlash(relative))))
	return hex.EncodeToString(sum[:8])
}

func samePath(left, right string) bool {
	leftAbs, _ := filepath.Abs(left)
	rightAbs, _ := filepath.Abs(right)
	return strings.EqualFold(filepath.Clean(leftAbs), filepath.Clean(rightAbs))
}
