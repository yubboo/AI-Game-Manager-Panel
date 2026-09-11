package logcenter

import (
	"archive/zip"
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

const (
	maxBundleLogBytes     = int64(32 * 1024 * 1024)
	maxBackupLogsPerShard = 5
)

var workshopLineRE = regexp.MustCompile(`(?i)(?:inserting\s+modname,|Loading\s+mod:|Mod:)\s*(workshop-\d+)(?:\s*\(([^)]*)\))?`)

var errorMarkers = []string{
	"lua error", "stack traceback", "server failed to start", "unhandled exception",
	"error loading", "failed msimulation", "socket_port_already_in_use",
}

type bundleFile struct{ source, archive string }

func (s *Store) Bundle(request BundleRequest) (ExportResult, error) {
	cluster, shards, files, err := prepareBundle(request)
	if err != nil {
		return ExportResult{}, err
	}
	outDir := s.defaultExportDir()
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return ExportResult{}, err
	}
	name := bundleName(cluster)
	path := filepath.Join(outDir, name)
	for suffix := 2; ; suffix++ {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			break
		}
		ext := filepath.Ext(name)
		base := strings.TrimSuffix(name, ext)
		path = filepath.Join(outDir, fmt.Sprintf("%s_%d%s", base, suffix, ext))
	}
	file, err := os.Create(path)
	if err != nil {
		return ExportResult{}, err
	}
	if err := writeBundleArchive(file, cluster, shards, files); err != nil {
		_ = file.Close()
		_ = os.Remove(path)
		return ExportResult{}, err
	}
	if err := file.Close(); err != nil {
		_ = os.Remove(path)
		return ExportResult{}, err
	}
	info, err := os.Stat(path)
	if err != nil {
		return ExportResult{}, err
	}
	return ExportResult{Path: path, Name: filepath.Base(path), Size: info.Size()}, nil
}

// BundleName validates the request and returns a safe download file name.
// It lets the HTTP bridge set Content-Disposition before streaming bytes.
func (s *Store) BundleName(request BundleRequest) (string, error) {
	cluster, _, _, err := prepareBundle(request)
	if err != nil {
		return "", err
	}
	return bundleName(cluster), nil
}

// StreamBundle writes a diagnostic archive directly to writer. The Web bridge
// uses this path so a large bundle is never materialized into browser memory or
// duplicated into the server user's Downloads directory first.
func (s *Store) StreamBundle(request BundleRequest, writer io.Writer) (string, error) {
	cluster, shards, files, err := prepareBundle(request)
	if err != nil {
		return "", err
	}
	name := bundleName(cluster)
	if err := writeBundleArchive(writer, cluster, shards, files); err != nil {
		return "", err
	}
	return name, nil
}

func prepareBundle(request BundleRequest) (string, []string, []bundleFile, error) {
	cluster := filepath.Clean(strings.TrimSpace(request.ClusterPath))
	stat, err := os.Stat(cluster)
	if err != nil || !stat.IsDir() {
		return "", nil, nil, fmt.Errorf("服务器存档不存在: %s", cluster)
	}
	shards := normalizeShardNames(request.ShardNames)
	if len(shards) == 0 {
		entries, _ := os.ReadDir(cluster)
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			if _, err := os.Stat(filepath.Join(cluster, entry.Name(), "server_log.txt")); err == nil {
				shards = append(shards, entry.Name())
			}
		}
		sort.Strings(shards)
	}
	files := make([]bundleFile, 0, 16)
	for _, shard := range shards {
		files = append(files, collectShardFiles(filepath.Join(cluster, shard), shard)...)
	}
	if len(files) == 0 {
		return "", nil, nil, fmt.Errorf("当前存档没有可收集的服务器日志")
	}
	return cluster, shards, files, nil
}

func bundleName(cluster string) string {
	stamp := time.Now().Format("20060102_150405")
	return fmt.Sprintf("AGMP_日志_%s_%s.zip", sanitizePart(filepath.Base(cluster)), stamp)
}

func writeBundleArchive(output io.Writer, cluster string, shards []string, files []bundleFile) error {
	archive := zip.NewWriter(output)
	manifest := []string{
		"AGMP · DST 本地服务器日志包",
		"生成时间：" + time.Now().Format("2006-01-02 15:04:05"),
		"存档目录：" + cluster,
		"世界：" + strings.Join(shards, "、"),
		"说明：已排除 cluster_token.txt、管理员/白名单/黑名单等敏感文件。",
		"",
	}
	mods := make(map[string]string)
	for _, item := range files {
		info, err := os.Stat(item.source)
		if err != nil {
			continue
		}
		if err := addFileToZip(archive, item.source, item.archive); err != nil {
			_ = archive.Close()
			return err
		}
		manifest = append(manifest, fmt.Sprintf("%s (%d bytes)", item.archive, info.Size()))
		if strings.Contains(strings.ToLower(item.archive), "server_log") && strings.HasSuffix(strings.ToLower(item.archive), ".txt") {
			collectMods(item.source, mods)
		}
	}
	if err := writeZipText(archive, "manifest.txt", strings.Join(manifest, "\n")+"\n"); err != nil {
		_ = archive.Close()
		return err
	}
	if err := writeZipText(archive, "mod_list.txt", formatModList(mods)); err != nil {
		_ = archive.Close()
		return err
	}
	return archive.Close()
}

func (s *Store) defaultExportDir() string {
	return s.exportDir()
}

func normalizeShardNames(values []string) []string {
	result := make([]string, 0, len(values))
	seen := map[string]bool{}
	for _, value := range values {
		name := strings.TrimSpace(value)
		if name == "" || name != filepath.Base(name) || name == "." || name == ".." {
			continue
		}
		key := strings.ToLower(name)
		if seen[key] {
			continue
		}
		seen[key] = true
		result = append(result, name)
	}
	return result
}

func collectShardFiles(shardPath, shardName string) []bundleFile {
	files := []bundleFile{}
	add := func(path, archive string) {
		info, err := os.Stat(path)
		if err == nil && info.Mode().IsRegular() && info.Size() <= maxBundleLogBytes {
			files = append(files, bundleFile{path, archive})
		}
	}
	add(filepath.Join(shardPath, "server_log.txt"), filepath.ToSlash(filepath.Join(shardName, "server_log.txt")))
	backupDir := filepath.Join(shardPath, "backup", "server_log")
	entries, _ := os.ReadDir(backupDir)
	type candidate struct {
		path, name string
		mod        time.Time
		errorLike  bool
	}
	candidates := []candidate{}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasPrefix(strings.ToLower(entry.Name()), "server_log_") || !strings.HasSuffix(strings.ToLower(entry.Name()), ".txt") {
			continue
		}
		path := filepath.Join(backupDir, entry.Name())
		info, err := entry.Info()
		if err != nil || info.Size() > maxBundleLogBytes {
			continue
		}
		candidates = append(candidates, candidate{path: path, name: entry.Name(), mod: info.ModTime(), errorLike: looksLikeError(path)})
	}
	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].errorLike != candidates[j].errorLike {
			return candidates[i].errorLike
		}
		return candidates[i].mod.After(candidates[j].mod)
	})
	if len(candidates) > maxBackupLogsPerShard {
		candidates = candidates[:maxBackupLogsPerShard]
	}
	for _, c := range candidates {
		add(c.path, filepath.ToSlash(filepath.Join(shardName, "backup", "server_log", c.name)))
	}
	for _, name := range []string{"server.ini", "modoverrides.lua", "leveldataoverride.lua"} {
		add(filepath.Join(shardPath, name), filepath.ToSlash(filepath.Join(shardName, name)))
	}
	return files
}

func looksLikeError(path string) bool {
	file, err := os.Open(path)
	if err != nil {
		return false
	}
	defer file.Close()
	scanner := bufio.NewScanner(io.LimitReader(file, maxBundleLogBytes))
	scanner.Buffer(make([]byte, 64*1024), 4*1024*1024)
	for scanner.Scan() {
		lower := strings.ToLower(scanner.Text())
		for _, marker := range errorMarkers {
			if strings.Contains(lower, marker) {
				return true
			}
		}
	}
	return false
}

func addFileToZip(archive *zip.Writer, source, name string) error {
	input, err := os.Open(source)
	if err != nil {
		return err
	}
	defer input.Close()
	writer, err := archive.Create(name)
	if err != nil {
		return err
	}
	_, err = io.CopyBuffer(writer, io.LimitReader(input, maxBundleLogBytes+1), make([]byte, 256*1024))
	return err
}
func writeZipText(archive *zip.Writer, name, text string) error {
	writer, err := archive.Create(name)
	if err != nil {
		return err
	}
	_, err = io.WriteString(writer, text)
	return err
}
func collectMods(path string, mods map[string]string) {
	file, err := os.Open(path)
	if err != nil {
		return
	}
	defer file.Close()
	scanner := bufio.NewScanner(io.LimitReader(file, maxBundleLogBytes))
	scanner.Buffer(make([]byte, 64*1024), 4*1024*1024)
	for scanner.Scan() {
		match := workshopLineRE.FindStringSubmatch(scanner.Text())
		if len(match) < 2 {
			continue
		}
		id := strings.ToLower(match[1])
		name := ""
		if len(match) > 2 {
			name = strings.TrimSpace(match[2])
		}
		if old, ok := mods[id]; !ok || (old == "" && name != "") {
			mods[id] = name
		}
	}
}
func formatModList(mods map[string]string) string {
	lines := []string{"AGMP · DST Mod 列表", "", "格式：Mod ID（名称）", ""}
	if len(mods) == 0 {
		lines = append(lines, "未从当前日志中识别到创意工坊 Mod。")
		return strings.Join(lines, "\n") + "\n"
	}
	ids := make([]string, 0, len(mods))
	for id := range mods {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		if mods[id] != "" {
			lines = append(lines, fmt.Sprintf("%s（%s）", id, mods[id]))
		} else {
			lines = append(lines, id)
		}
	}
	return strings.Join(lines, "\n") + "\n"
}
