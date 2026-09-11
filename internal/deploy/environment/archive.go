package environment

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const maxArchiveFiles = 20000
const maxArchiveBytes int64 = 4 << 30

func secureJoin(root, name string) (string, error) {
	name = filepath.Clean(filepath.FromSlash(name))
	if name == "." || filepath.IsAbs(name) || name == ".." || strings.HasPrefix(name, ".."+string(os.PathSeparator)) {
		return "", errors.New("压缩包包含不安全路径")
	}
	dst := filepath.Join(root, name)
	rel, err := filepath.Rel(root, dst)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return "", errors.New("压缩包路径逃逸")
	}
	return dst, nil
}

func extractZIP(path, target string) error {
	reader, err := zip.OpenReader(path)
	if err != nil {
		return err
	}
	defer reader.Close()
	if len(reader.File) > maxArchiveFiles {
		return errors.New("压缩包文件数量超限")
	}
	var total int64
	for _, f := range reader.File {
		dst, err := secureJoin(target, f.Name)
		if err != nil {
			return err
		}
		mode := f.Mode()
		if mode&os.ModeSymlink != 0 || mode&os.ModeDevice != 0 || mode&os.ModeNamedPipe != 0 || mode&os.ModeSocket != 0 {
			return errors.New("压缩包包含不允许的特殊文件")
		}
		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(dst, 0o755); err != nil {
				return err
			}
			continue
		}
		total += int64(f.UncompressedSize64)
		if total > maxArchiveBytes {
			return errors.New("压缩包解压大小超限")
		}
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			return err
		}
		src, err := f.Open()
		if err != nil {
			return err
		}
		out, err := os.OpenFile(dst, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, mode.Perm()|0o600)
		if err != nil {
			src.Close()
			return err
		}
		_, copyErr := io.CopyN(out, src, int64(f.UncompressedSize64))
		closeErr := out.Close()
		src.Close()
		if copyErr != nil && !errors.Is(copyErr, io.EOF) {
			return copyErr
		}
		if closeErr != nil {
			return closeErr
		}
	}
	return nil
}

func extractTarGZ(path, target string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()
	gz, err := gzip.NewReader(file)
	if err != nil {
		return err
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	count := 0
	var total int64
	for {
		h, err := tr.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return err
		}
		count++
		if count > maxArchiveFiles {
			return errors.New("压缩包文件数量超限")
		}
		dst, err := secureJoin(target, h.Name)
		if err != nil {
			return err
		}
		switch h.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(dst, 0o755); err != nil {
				return err
			}
		case tar.TypeReg, tar.TypeRegA:
			if h.Size < 0 {
				return errors.New("压缩包文件大小无效")
			}
			total += h.Size
			if total > maxArchiveBytes {
				return errors.New("压缩包解压大小超限")
			}
			if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
				return err
			}
			mode := os.FileMode(h.Mode).Perm()
			if mode == 0 {
				mode = 0o644
			}
			out, err := os.OpenFile(dst, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, mode)
			if err != nil {
				return err
			}
			n, copyErr := io.CopyN(out, tr, h.Size)
			closeErr := out.Close()
			if copyErr != nil || n != h.Size {
				return fmt.Errorf("解压文件失败 %s: %w", h.Name, copyErr)
			}
			if closeErr != nil {
				return closeErr
			}
		default:
			return fmt.Errorf("压缩包包含不允许的特殊文件: %s", h.Name)
		}
	}
	return nil
}
