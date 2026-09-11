package cluster

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"unicode"
)

func Import(kleiRoot string, request ImportRequest) (ImportResult, error) {
	source := filepath.Clean(strings.TrimSpace(request.SourcePath))
	root := filepath.Clean(strings.TrimSpace(kleiRoot))
	if source == "." || source == "" {
		return ImportResult{}, errors.New("缺少待导入的 Cluster 路径")
	}
	if root == "." || root == "" {
		return ImportResult{}, errors.New("Klei 根目录不可用")
	}
	if !isFile(filepath.Join(source, "cluster.ini")) || !isFile(filepath.Join(source, "Master", "server.ini")) {
		return ImportResult{}, errors.New("所选目录不是完整 DST Cluster：需要 cluster.ini 和 Master/server.ini")
	}
	name := strings.TrimSpace(request.TargetName)
	if name == "" {
		name = filepath.Base(source)
	}
	if err := validateName(name); err != nil {
		return ImportResult{}, err
	}
	if err := os.MkdirAll(root, 0o755); err != nil {
		return ImportResult{}, fmt.Errorf("创建 Klei 根目录失败: %w", err)
	}
	target := filepath.Join(root, name)
	if !pathWithin(root, target) {
		return ImportResult{}, errors.New("目标 Cluster 路径越界")
	}
	if _, err := os.Stat(target); err == nil {
		return ImportResult{}, fmt.Errorf("目标 Cluster 已存在: %s", target)
	} else if !errors.Is(err, os.ErrNotExist) {
		return ImportResult{}, err
	}

	result := ImportResult{Name: name, Path: target, SourcePath: source, Warnings: []string{}}
	if err := copyTree(source, target, &result); err != nil {
		_ = os.RemoveAll(target)
		return ImportResult{}, err
	}
	if !isFile(filepath.Join(target, "cluster.ini")) || !isFile(filepath.Join(target, "Master", "server.ini")) {
		_ = os.RemoveAll(target)
		return ImportResult{}, errors.New("导入后完整性复查失败")
	}
	return result, nil
}

func validateName(name string) error {
	if name == "." || name == ".." || len([]rune(name)) > 64 {
		return errors.New("Cluster 名称无效或过长")
	}
	for _, r := range name {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' || r == '-' || r == ' ' {
			continue
		}
		return fmt.Errorf("Cluster 名称包含不允许的字符: %q", r)
	}
	return nil
}

func pathWithin(root, target string) bool {
	rel, err := filepath.Rel(root, target)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(os.PathSeparator)) && !filepath.IsAbs(rel)
}

func copyTree(source, target string, result *ImportResult) error {
	return filepath.WalkDir(source, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return os.MkdirAll(target, 0o755)
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("为安全起见不导入符号链接: %s", path)
		}
		if strings.EqualFold(filepath.Base(path), "cluster_token.txt") {
			result.TokenSkipped = true
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		dstPath := filepath.Join(target, rel)
		if entry.IsDir() {
			return os.MkdirAll(dstPath, 0o755)
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("不支持导入特殊文件: %s", path)
		}
		if err := copyFile(path, dstPath, info.Mode().Perm()); err != nil {
			return err
		}
		result.CopiedFiles++
		result.CopiedBytes += info.Size()
		return nil
	})
}

func copyFile(source, target string, mode os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
	}
	in, err := os.Open(source)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
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

func isFile(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.Mode().IsRegular()
}
