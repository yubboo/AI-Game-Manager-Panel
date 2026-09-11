package logcenter

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	dstruntime "github.com/yubboo/AI-Game-Manager-Panel/internal/games/dst/runtime"
)

var ErrSessionNotFound = errors.New("DST log session not found")

type Store struct {
	root       string
	exportRoot string
	mu         sync.RWMutex
	writers    map[string]*sessionWriter
}

// New creates a persistent DST log store. When exportRoot is provided, user-
// visible exports are written there instead of the operating system Downloads
// folder. This keeps AGMP portable and prevents unexpected writes to C:\Users.
func New(root string, exportRoots ...string) *Store {
	exportRoot := filepath.Join(root, "exports")
	if len(exportRoots) > 0 && strings.TrimSpace(exportRoots[0]) != "" {
		exportRoot = filepath.Clean(exportRoots[0])
	}
	return &Store{root: root, exportRoot: exportRoot, writers: make(map[string]*sessionWriter)}
}

func (s *Store) StartSession(request dstruntime.StartRequest) (dstruntime.LogSessionSink, error) {
	if err := os.MkdirAll(s.root, 0o755); err != nil {
		return nil, err
	}
	id := newSessionID()
	clusterDir := clusterDirectory(request.ClusterName, request.ClusterPath)
	shard := sanitizePart(request.ShardName)
	if shard == "" {
		shard = "Shard"
	}
	dir := filepath.Join(s.root, clusterDir, shard)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	stamp := time.Now().Format("20060102-150405")
	base := fmt.Sprintf("%s-%s-%s", stamp, shard, id)
	logPath := filepath.Join(dir, base+".log")
	metaPath := filepath.Join(dir, base+".json")
	file, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return nil, err
	}

	meta := Session{
		ID:          id,
		ClusterName: request.ClusterName,
		ClusterPath: request.ClusterPath,
		ShardName:   request.ShardName,
		Status:      "running",
		StartedAt:   time.Now().Unix(),
		LogPath:     logPath,
		MetaPath:    metaPath,
	}
	w := newSessionWriter(s, file, meta)
	s.mu.Lock()
	s.writers[id] = w
	s.mu.Unlock()
	if err := writeSessionMeta(meta); err != nil {
		w.abort()
		s.mu.Lock()
		delete(s.writers, id)
		s.mu.Unlock()
		return nil, err
	}
	go w.run()
	return w, nil
}

func (s *Store) removeWriter(id string) {
	s.mu.Lock()
	delete(s.writers, id)
	s.mu.Unlock()
}

func (s *Store) flushActive(id string) {
	s.mu.RLock()
	w := s.writers[strings.TrimSpace(id)]
	s.mu.RUnlock()
	if w != nil {
		w.flushSync(2 * time.Second)
	}
}

func (s *Store) List(request ListRequest) ([]Session, error) {
	limit := request.Limit
	if limit <= 0 {
		limit = DefaultListLimit
	}
	if limit > MaxListLimit {
		limit = MaxListLimit
	}
	clusterPath := filepath.Clean(strings.TrimSpace(request.ClusterPath))
	shardName := strings.TrimSpace(request.ShardName)
	items := make([]Session, 0, limit)
	roots := []string{s.root}
	if clusterPath != "." && clusterPath != "" {
		pattern := filepath.Join(s.root, "*-"+clusterPathHash(clusterPath))
		if matches, err := filepath.Glob(pattern); err == nil && len(matches) > 0 {
			roots = matches
		}
	}
	walk := func(root string) {
		_ = filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
			if err != nil || entry == nil || entry.IsDir() || !strings.HasSuffix(strings.ToLower(entry.Name()), ".json") {
				return nil
			}
			value, err := readSessionMeta(path)
			if err != nil {
				return nil
			}
			if clusterPath != "." && clusterPath != "" && !samePath(value.ClusterPath, clusterPath) {
				return nil
			}
			if shardName != "" && !strings.EqualFold(value.ShardName, shardName) {
				return nil
			}
			if stat, err := os.Stat(value.LogPath); err == nil {
				value.ByteSize = stat.Size()
			}
			items = append(items, value)
			return nil
		})
	}
	for _, root := range roots {
		walk(root)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].StartedAt > items[j].StartedAt })
	if len(items) > limit {
		items = items[:limit]
	}
	return items, nil
}

func (s *Store) Session(id string) (Session, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return Session{}, ErrSessionNotFound
	}
	s.mu.RLock()
	w := s.writers[id]
	s.mu.RUnlock()
	if w != nil {
		return w.snapshot(), nil
	}
	var found Session
	errFound := errors.New("found")
	_ = filepath.WalkDir(s.root, func(path string, entry os.DirEntry, err error) error {
		if err != nil || entry == nil || entry.IsDir() || !strings.HasSuffix(strings.ToLower(entry.Name()), ".json") {
			return nil
		}
		value, err := readSessionMeta(path)
		if err != nil {
			return nil
		}
		if value.ID == id {
			found = value
			return errFound
		}
		return nil
	})
	if found.ID == "" {
		return Session{}, ErrSessionNotFound
	}
	if stat, err := os.Stat(found.LogPath); err == nil {
		found.ByteSize = stat.Size()
	}
	return found, nil
}

func (s *Store) Read(request ReadRequest) (ReadPage, error) {
	s.flushActive(request.SessionID)
	session, err := s.Session(request.SessionID)
	if err != nil {
		return ReadPage{}, err
	}
	limit := request.Limit
	if limit <= 0 {
		limit = DefaultReadLimit
	}
	if limit > MaxReadLimit {
		limit = MaxReadLimit
	}
	file, err := os.Open(session.LogPath)
	if err != nil {
		return ReadPage{}, err
	}
	defer file.Close()
	cursor := request.Cursor
	if cursor < 0 {
		cursor = 0
	}
	if _, err := file.Seek(cursor, io.SeekStart); err != nil {
		return ReadPage{}, err
	}
	reader := bufio.NewReaderSize(file, 128*1024)
	lineNo := request.StartLine
	if lineNo == 0 {
		lineNo = 1
	}
	page := ReadPage{Session: session, Lines: make([]LogLine, 0, limit), NextCursor: cursor, NextLine: lineNo}
	for len(page.Lines) < limit {
		text, readErr := reader.ReadString('\n')
		if len(text) > 0 {
			text = strings.TrimSuffix(strings.TrimSuffix(text, "\n"), "\r")
			page.Lines = append(page.Lines, makeLogLine(lineNo, text))
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
		// bufio may have read ahead. Subtract unread buffered bytes to return an exact cursor.
		pos -= int64(reader.Buffered())
		page.NextCursor = pos
	}
	page.NextLine = lineNo
	return page, nil
}

func (s *Store) Tail(request TailRequest) (ReadPage, error) {
	s.flushActive(request.SessionID)
	session, err := s.Session(request.SessionID)
	if err != nil {
		return ReadPage{}, err
	}
	limit := request.Limit
	if limit <= 0 {
		limit = DefaultReadLimit
	}
	if limit > MaxReadLimit {
		limit = MaxReadLimit
	}
	lines, err := tailLines(session.LogPath, limit)
	if err != nil {
		return ReadPage{}, err
	}
	start := uint64(1)
	if session.LineCount > uint64(len(lines)) {
		start = session.LineCount - uint64(len(lines)) + 1
	}
	result := make([]LogLine, 0, len(lines))
	for i, line := range lines {
		result = append(result, makeLogLine(start+uint64(i), line))
	}
	stat, _ := os.Stat(session.LogPath)
	cursor := int64(0)
	if stat != nil {
		cursor = stat.Size()
	}
	return ReadPage{Session: session, Lines: result, NextCursor: cursor, NextLine: session.LineCount + 1, EOF: true}, nil
}

func (s *Store) Search(request SearchRequest) (SearchResult, error) {
	s.flushActive(request.SessionID)
	started := time.Now()
	session, err := s.Session(request.SessionID)
	if err != nil {
		return SearchResult{}, err
	}
	limit := request.Limit
	if limit <= 0 {
		limit = DefaultSearchLimit
	}
	if limit > MaxSearchLimit {
		limit = MaxSearchLimit
	}
	query := strings.ToLower(strings.TrimSpace(request.Query))
	level := strings.ToLower(strings.TrimSpace(request.Level))
	category := strings.ToLower(strings.TrimSpace(request.Category))
	file, err := os.Open(session.LogPath)
	if err != nil {
		return SearchResult{}, err
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 64*1024), 4*1024*1024)
	result := SearchResult{Session: session, Matches: make([]LogLine, 0, minInt(limit, 64))}
	var lineNo uint64
	for scanner.Scan() {
		lineNo++
		result.Scanned = lineNo
		text := scanner.Text()
		classified := makeLogLine(lineNo, text)
		if query != "" && !strings.Contains(strings.ToLower(text), query) {
			continue
		}
		if level != "" && level != "all" && classified.Level != level {
			continue
		}
		if category != "" && category != "all" && classified.Category != category {
			continue
		}
		if len(result.Matches) < limit {
			result.Matches = append(result.Matches, classified)
		} else {
			result.Truncated = true
			break
		}
	}
	if err := scanner.Err(); err != nil {
		return SearchResult{}, err
	}
	result.DurationMs = time.Since(started).Milliseconds()
	return result, nil
}

func (s *Store) Diagnostics(id string) (Diagnostics, error) {
	s.flushActive(id)
	started := time.Now()
	session, err := s.Session(id)
	if err != nil {
		return Diagnostics{}, err
	}
	file, err := os.Open(session.LogPath)
	if err != nil {
		return Diagnostics{}, err
	}
	defer file.Close()

	result := Diagnostics{Session: session, Issues: []DiagnosticIssue{}}
	counts := map[string]int{}
	evidence := map[string][]string{}
	recent := make(map[string]struct{}, 700)
	recentOrder := make([]string, 0, 700)
	firewallWarnings := 0
	serverBound := false
	shardTransportReady := false
	addEvidence := func(code, text string) {
		items := evidence[code]
		if len(items) >= 3 {
			items = items[1:]
		}
		evidence[code] = append(items, strings.TrimSpace(text))
	}
	consume := func(text string) {
		lower := strings.ToLower(text)
		level := classifyLevel(lower)
		isFirewallWarning := strings.Contains(lower, "could not confirm port") && strings.Contains(lower, "firewall")
		if level == "error" {
			result.ErrorCount++
		}
		if level == "warning" && !isFirewallWarning {
			result.WarningCount++
		}
		if strings.Contains(text, "SteamGameServer_Init success") {
			result.SteamReady = true
		}
		if strings.Contains(lower, "server registered via geo dns") {
			result.Registered = true
		}
		if strings.Contains(text, "Token retrieved from:") {
			result.TokenLoaded = true
		}
		if strings.Contains(lower, "online server started on port:") {
			serverBound = true
		}
		if strings.Contains(lower, "shard server started on port:") ||
			strings.Contains(lower, "secondary shard is now ready") ||
			strings.Contains(lower, "secondary caves") && strings.Contains(lower, "ready") ||
			strings.Contains(lower, "world ") && strings.Contains(lower, " is now connected") {
			shardTransportReady = true
		}
		if strings.Contains(text, "Sim paused") || strings.Contains(text, "Sim unpaused") || strings.Contains(text, "DST_Master_Ready") || strings.Contains(lower, " is now ready!") {
			result.WorldReady = true
		}

		if strings.Contains(lower, "failed to run code from modoverrides.lua") {
			counts["modoverrides"]++
			addEvidence("modoverrides", text)
		}
		if strings.Contains(lower, "-console has been deprecated") {
			counts["console_deprecated"]++
			addEvidence("console_deprecated", text)
		}
		if isFirewallWarning {
			firewallWarnings++
			addEvidence("firewall", text)
		}
		if strings.Contains(lower, "e_rowid_exist") || ((strings.Contains(lower, "master server broadcast error") || strings.Contains(lower, "http_500")) && strings.Contains(lower, "token")) {
			counts["token_conflict"]++
			addEvidence("token_conflict", text)
		}
		if strings.Contains(lower, "token") && (strings.Contains(lower, "failed") || strings.Contains(lower, "invalid")) {
			counts["token"]++
			addEvidence("token", text)
		}
		if strings.Contains(lower, "socket_port_already_in_use") || strings.Contains(lower, "address already in use") || strings.Contains(lower, "port_already_in_use") || strings.Contains(lower, "bind failed") || strings.Contains(lower, "only one usage") {
			counts["port"]++
			addEvidence("port", text)
		}
		if strings.Contains(lower, "server failed to start") {
			counts["server_start"]++
			addEvidence("server_start", text)
		}
		if strings.Contains(lower, "vcruntime140.dll") || strings.Contains(lower, "msvcp140.dll") || strings.Contains(lower, "vcomp120.dll") || strings.Contains(lower, "cannot find the module") || strings.Contains(lower, "找不到指定模块") {
			counts["runtime"]++
			addEvidence("runtime", text)
		}
		if strings.Contains(lower, "access is denied") || strings.Contains(lower, "permission denied") || strings.Contains(lower, "拒绝访问") || strings.Contains(lower, "permissionerror") || strings.Contains(lower, "check for write access: false") || strings.Contains(lower, "check for read access: false") {
			counts["permission"]++
			addEvidence("permission", text)
		}
		if strings.Contains(lower, "must specify the task set for a level") || strings.Contains(lower, "error loading worldgen_main.lua") {
			counts["worldgen"]++
			addEvidence("worldgen", text)
		}
		if strings.Contains(lower, "lua error") || strings.Contains(lower, "stack traceback") {
			counts["lua"]++
			addEvidence("lua", text)
		}
	}

	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 64*1024), 4*1024*1024)
	var lineNo uint64
	for scanner.Scan() {
		lineNo++
		text := scanner.Text()
		consume(text)
		recent[text] = struct{}{}
		recentOrder = append(recentOrder, text)
		if len(recentOrder) > 700 {
			old := recentOrder[0]
			recentOrder = recentOrder[1:]
			delete(recent, old)
		}
	}
	if err := scanner.Err(); err != nil {
		return Diagnostics{}, err
	}

	// Some failures only fully flush to shard/server_log.txt. Merge its tail
	// without counting lines that are already present in the AGMP pipe log.
	nativePath := filepath.Join(session.ClusterPath, session.ShardName, "server_log.txt")
	if nativeLines, err := tailLines(nativePath, 700); err == nil {
		for _, text := range nativeLines {
			if _, exists := recent[text]; exists {
				continue
			}
			consume(text)
			lineNo++
		}
	}

	// DST frequently prints "Could not confirm port ... is open in the firewall"
	// even when the socket is subsequently bound successfully and the shard
	// becomes ready. Treat that as a resolved informational hint, not an active
	// startup warning. This prevents a healthy Master/Caves session from being
	// displayed as broken merely because Windows firewall introspection failed.
	result.NetworkReady = serverBound && (shardTransportReady || result.WorldReady)
	if firewallWarnings > 0 && !result.NetworkReady {
		counts["firewall"] = firewallWarnings
		result.WarningCount += firewallWarnings
	}

	// Health here means the dedicated-server startup chain is healthy, not that
	// every later runtime line is free of errors. A bad admin console command or
	// a non-fatal Mod error after World Ready must not retroactively turn a
	// successfully registered server into "startup failed".
	startupBlockers := counts["token_conflict"] + counts["runtime"] + counts["port"] + counts["server_start"] + counts["permission"] + counts["worldgen"] + counts["token"]
	isCaves := strings.EqualFold(session.ShardName, "Caves")
	if isCaves {
		result.Healthy = result.SteamReady && result.TokenLoaded && result.NetworkReady && result.WorldReady && startupBlockers == 0
	} else {
		result.Healthy = result.SteamReady && result.TokenLoaded && result.NetworkReady && result.Registered && result.WorldReady && startupBlockers == 0
	}

	issue := func(code, severity, title, detail string, certain bool, suggestions ...string) {
		n := counts[code]
		if n == 0 {
			return
		}
		result.Issues = append(result.Issues, DiagnosticIssue{Code: code, Severity: severity, Title: title, Detail: detail, Count: n, Suggestions: suggestions, Evidence: evidence[code], Certain: certain})
	}
	issue("token_conflict", "error", "令牌注册冲突", "服务器无法正常向 Klei 注册房间；地面与洞穴可能持续重试并无法完成建联。", true, "停止使用同一令牌的其他存档，或更换可用 Cluster Token。", "若房间刚崩溃，等待 Klei 释放旧注册后再重试。")
	issue("runtime", "error", "Visual C++ 运行库缺失", "Dedicated Server 启动时找不到所需的 Windows 运行库。", true, "确认专服位数与运行库位数匹配。", "LuaJIT 模式请检查相应 VC++ Runtime。")
	issue("port", "error", "服务器端口被占用", "服务器需要使用的网络端口已被其他进程占用或绑定失败。", true, "检查当前 Cluster 的 server.ini 端口配置。", "关闭占用端口的程序，或为当前存档分配新的端口。")
	issue("server_start", "error", "服务器网络监听启动失败", "DST 已进入启动阶段，但没有成功建立在线服务器监听。", true, "检查当前 Shard 的 UDP 端口是否被占用。", "确认 Windows 防火墙/安全软件没有阻止 Dedicated Server。")
	issue("permission", "error", "文件访问失败", "服务器无法读取或写入专服安装目录或 Klei 存档目录。", true, "确认当前用户对 Dedicated Server 与 Klei 目录有读写权限。", "检查安全软件是否拦截服务器进程。")
	issue("worldgen", "error", "世界生成配置错误", "世界生成阶段缺少有效任务集或 worldgen 配置无法加载。", true, "检查世界类型与生成预设是否匹配。", "若刚更新世界配置 Mod，请重新保存世界设置。")
	issue("lua", "error", "检测到 Lua 运行时错误", "可能来自 Mod、世界脚本或兼容性问题；应结合 stack traceback 定位。", false, "优先检查错误堆栈里明确出现的 workshop Mod。", "若问题来自最近新增/更新的 Mod，逐个停用后重启确认。")
	issue("modoverrides", "error", "modoverrides.lua 加载失败", "Mod 配置脚本执行失败；服务器仍可能继续启动，但 Mod 配置可能未生效。", true, "检查 modoverrides.lua 语法和文件编码。", "若没有启用 Mod，可重新生成干净的 modoverrides.lua。")
	issue("token", "error", "Cluster Token 异常", "日志中出现 Token 读取或校验失败信息。", true, "检查 cluster_token.txt 是否存在且内容有效。")
	issue("firewall", "warning", "端口防火墙状态无法确认", "DST 无法确认某些服务器端口是否已在 Windows 防火墙放行，而且当前日志里尚未观察到后续成功监听。", false, "若公网/局域网无法加入，再检查 Windows 防火墙与路由器端口。")
	issue("console_deprecated", "warning", "检测到已弃用的 -console 启动参数", "当前日志来自仍使用旧参数的启动方式；AGMP 0.1.34 起改用 [MISC] / console_enabled。", true, "若这是 AGMP 0.1.34 之后的新日志，请检查额外启动参数中是否手工加入了 -console。")
	if session.Status == "crashed" && len(result.Issues) == 0 {
		result.Issues = append(result.Issues, DiagnosticIssue{Code: "unknown_crash", Severity: "error", Title: "服务器异常退出", Detail: "进程异常退出，但当前日志证据不足以确定单一原因。", Count: 1, Suggestions: []string{"查看日志末尾并检查令牌、端口、存档权限和最近更新的 Mod。"}, Certain: false})
	}
	result.ScannedLines = lineNo
	result.DurationMs = time.Since(started).Milliseconds()
	return result, nil
}

func (s *Store) File(id string) (FileRef, error) {
	s.flushActive(id)
	session, err := s.Session(id)
	if err != nil {
		return FileRef{}, err
	}
	stat, err := os.Stat(session.LogPath)
	if err != nil {
		return FileRef{}, err
	}
	return FileRef{Path: session.LogPath, Name: filepath.Base(session.LogPath), Size: stat.Size(), ModTime: stat.ModTime()}, nil
}

func (s *Store) Export(id string) (ExportResult, error) {
	ref, err := s.File(id)
	if err != nil {
		return ExportResult{}, err
	}
	outDir := s.exportDir()
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return ExportResult{}, err
	}
	name := ref.Name
	dst := filepath.Join(outDir, name)
	if _, err := os.Stat(dst); err == nil {
		ext := filepath.Ext(name)
		base := strings.TrimSuffix(name, ext)
		dst = filepath.Join(outDir, fmt.Sprintf("%s-export-%s%s", base, time.Now().Format("150405"), ext))
	}
	srcFile, err := os.Open(ref.Path)
	if err != nil {
		return ExportResult{}, err
	}
	defer srcFile.Close()
	dstFile, err := os.OpenFile(dst, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
	if err != nil {
		return ExportResult{}, err
	}
	_, copyErr := io.CopyBuffer(dstFile, srcFile, make([]byte, 256*1024))
	closeErr := dstFile.Close()
	if copyErr != nil {
		return ExportResult{}, copyErr
	}
	if closeErr != nil {
		return ExportResult{}, closeErr
	}
	stat, err := os.Stat(dst)
	if err != nil {
		return ExportResult{}, err
	}
	return ExportResult{Path: dst, Name: filepath.Base(dst), Size: stat.Size()}, nil
}

func (s *Store) exportDir() string {
	if strings.TrimSpace(s.exportRoot) != "" {
		return filepath.Clean(s.exportRoot)
	}
	return filepath.Join(s.root, "exports")
}

func writeSessionMeta(value Session) error {
	if err := os.MkdirAll(filepath.Dir(value.MetaPath), 0o755); err != nil {
		return err
	}
	payload, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	tmp := value.MetaPath + ".tmp"
	if err := os.WriteFile(tmp, payload, 0o644); err != nil {
		return err
	}
	// Windows cannot atomically rename over an existing file. Remove the tiny
	// metadata target first; the log file itself is append-only and unaffected.
	_ = os.Remove(value.MetaPath)
	return os.Rename(tmp, value.MetaPath)
}

func readSessionMeta(path string) (Session, error) {
	payload, err := os.ReadFile(path)
	if err != nil {
		return Session{}, err
	}
	var value Session
	if err := json.Unmarshal(payload, &value); err != nil {
		return Session{}, err
	}
	value.MetaPath = path
	if value.LogPath == "" {
		value.LogPath = strings.TrimSuffix(path, filepath.Ext(path)) + ".log"
	}
	return value, nil
}

func newSessionID() string {
	seed := fmt.Sprintf("%d-%d", time.Now().UnixNano(), os.Getpid())
	sum := sha256.Sum256([]byte(seed))
	return hex.EncodeToString(sum[:6])
}

func clusterDirectory(name, path string) string {
	slug := sanitizePart(name)
	if slug == "" {
		slug = "Cluster"
	}
	return fmt.Sprintf("%s-%s", slug, clusterPathHash(path))
}

func clusterPathHash(path string) string {
	sum := sha256.Sum256([]byte(strings.ToLower(filepath.Clean(path))))
	return hex.EncodeToString(sum[:4])
}

func sanitizePart(value string) string {
	value = strings.TrimSpace(value)
	replacer := strings.NewReplacer("<", "_", ">", "_", ":", "_", "\"", "_", "/", "_", "\\", "_", "|", "_", "?", "_", "*", "_")
	value = replacer.Replace(value)
	value = strings.Trim(value, ". ")
	if len(value) > 64 {
		value = value[:64]
	}
	return value
}

func samePath(left, right string) bool {
	return strings.EqualFold(filepath.Clean(left), filepath.Clean(right))
}
func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
