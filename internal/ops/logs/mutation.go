package loghub

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// ExportOne 把单个原始日志复制到配置日志根目录的 exports 子目录，便于桌面和本机 Web 使用同一路径。
func (s *Service) ExportOne(id string) (ExportResult, error) {
	meta, path, err := s.findByID(strings.TrimSpace(id))
	if err != nil {
		return ExportResult{}, err
	}
	if err := os.MkdirAll(s.exportRoot, 0o755); err != nil {
		return ExportResult{}, err
	}
	name := fmt.Sprintf("%s-%s", time.Now().Format("20060102-150405"), sanitizeFilename(meta.Name))
	dst := filepath.Join(s.exportRoot, name)
	if err := copyFile(path, dst); err != nil {
		return ExportResult{}, err
	}
	stat, err := os.Stat(dst)
	if err != nil {
		return ExportResult{}, err
	}
	return ExportResult{Path: dst, Name: filepath.Base(dst), Size: stat.Size()}, nil
}

// ExportFiltered 将筛选结果打成 ZIP，并附带 manifest.json 记录来源元数据。
func (s *Service) ExportFiltered(filter CatalogRequest) (ExportResult, error) {
	items, err := s.filteredAll(filter)
	if err != nil {
		return ExportResult{}, err
	}
	if len(items) == 0 {
		return ExportResult{}, fmt.Errorf("当前筛选条件下没有可导出的日志")
	}
	if err := os.MkdirAll(s.exportRoot, 0o755); err != nil {
		return ExportResult{}, err
	}
	name := fmt.Sprintf("AI Game Manager Panel-Logs-%s.zip", time.Now().Format("20060102-150405"))
	dst := filepath.Join(s.exportRoot, name)
	file, err := os.Create(dst)
	if err != nil {
		return ExportResult{}, err
	}
	zw := zip.NewWriter(file)
	manifest := make([]LogFile, 0, len(items))
	for _, item := range items {
		path, err := s.safePath(item.RelativePath)
		if err != nil {
			continue
		}
		source, err := os.Open(path)
		if err != nil {
			continue
		}
		entryName := filepath.ToSlash(filepath.Join("logs", item.RelativePath))
		entry, err := zw.Create(entryName)
		if err == nil {
			_, err = io.Copy(entry, source)
		}
		_ = source.Close()
		if err == nil {
			manifest = append(manifest, item)
		}
	}
	manifestWriter, manifestErr := zw.Create("manifest.json")
	if manifestErr == nil {
		payload, _ := json.MarshalIndent(map[string]any{
			"exportedAt": time.Now().Format(time.RFC3339),
			"files":      manifest,
		}, "", "  ")
		_, manifestErr = manifestWriter.Write(payload)
	}
	closeZipErr := zw.Close()
	closeFileErr := file.Close()
	if manifestErr != nil {
		_ = os.Remove(dst)
		return ExportResult{}, manifestErr
	}
	if closeZipErr != nil {
		_ = os.Remove(dst)
		return ExportResult{}, closeZipErr
	}
	if closeFileErr != nil {
		_ = os.Remove(dst)
		return ExportResult{}, closeFileErr
	}
	stat, err := os.Stat(dst)
	if err != nil {
		return ExportResult{}, err
	}
	return ExportResult{Path: dst, Name: name, Size: stat.Size()}, nil
}

func (s *Service) DeleteOne(id string) (MutationResult, error) {
	item, path, err := s.findByID(strings.TrimSpace(id))
	if err != nil {
		return MutationResult{}, err
	}
	if item.Active {
		return MutationResult{Skipped: 1, SkippedIDs: []string{item.ID}}, ErrLogActive
	}
	result := MutationResult{DeletedIDs: []string{}, SkippedIDs: []string{}, ErrorDetails: []string{}}
	s.deleteItem(item, path, &result)
	return result, nil
}

func (s *Service) DeleteFiltered(request DeleteFilteredRequest) (MutationResult, error) {
	items, err := s.filteredAll(request.Filter)
	if err != nil {
		return MutationResult{}, err
	}
	return s.deleteItems(items), nil
}

// ClearHistory 删除所有非活动日志。当前 AI Game Manager Panel Core、当天操作审计和正在运行的游戏 Session 会跳过。
func (s *Service) ClearHistory() (MutationResult, error) {
	items, err := s.scanAll()
	if err != nil {
		return MutationResult{}, err
	}
	return s.deleteItems(items), nil
}

func (s *Service) deleteItems(items []LogFile) MutationResult {
	result := MutationResult{DeletedIDs: []string{}, SkippedIDs: []string{}, ErrorDetails: []string{}}
	for _, item := range items {
		if item.Active {
			result.Skipped++
			result.SkippedIDs = append(result.SkippedIDs, item.ID)
			continue
		}
		path, err := s.safePath(item.RelativePath)
		if err != nil {
			result.ErrorDetails = append(result.ErrorDetails, fmt.Sprintf("%s: %v", item.RelativePath, err))
			continue
		}
		s.deleteItem(item, path, &result)
	}
	return result
}

func (s *Service) deleteItem(item LogFile, path string, result *MutationResult) {
	if err := os.Remove(path); err != nil {
		result.ErrorDetails = append(result.ErrorDetails, fmt.Sprintf("%s: %v", item.RelativePath, err))
		return
	}
	result.Deleted++
	result.BytesFreed += item.ByteSize
	result.DeletedIDs = append(result.DeletedIDs, item.ID)
	// DST Session 元数据和日志同名；删除历史日志时同步删除元数据，避免日志目录残留孤儿记录。
	metaPath := strings.TrimSuffix(path, filepath.Ext(path)) + ".json"
	if payload, err := os.ReadFile(metaPath); err == nil && strings.Contains(string(payload), `"clusterName"`) && strings.Contains(string(payload), `"shardName"`) {
		_ = os.Remove(metaPath)
	}
	s.cacheMu.Lock()
	delete(s.lineCache, path)
	s.cacheMu.Unlock()
}

func (s *Service) filteredAll(filter CatalogRequest) ([]LogFile, error) {
	items, err := s.scanAll()
	if err != nil {
		return nil, err
	}
	filtered := make([]LogFile, 0, len(items))
	for _, item := range items {
		if matchesCatalog(item, filter) {
			filtered = append(filtered, item)
		}
	}
	sort.Slice(filtered, func(i, j int) bool { return filtered[i].ModifiedAt > filtered[j].ModifiedAt })
	return filtered, nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(out, in)
	closeErr := out.Close()
	if copyErr != nil {
		return copyErr
	}
	return closeErr
}

func sanitizeFilename(value string) string {
	value = strings.TrimSpace(value)
	replacer := strings.NewReplacer("<", "_", ">", "_", ":", "_", `"`, "_", "/", "_", `\\`, "_", "|", "_", "?", "_", "*", "_")
	value = replacer.Replace(value)
	if value == "" {
		return "log.log"
	}
	return value
}
