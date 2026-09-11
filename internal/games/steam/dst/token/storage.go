package token

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	platformfiles "github.com/yubboo/AI-Game-Manager-Panel/internal/platform/files"
)

const maxTokenBytes = 16 * 1024

var (
	ErrEmptyToken   = errors.New("cluster token is empty")
	ErrTokenTooLong = errors.New("cluster token is unexpectedly large")
)

func Inspect(clusterPath string) (Status, error) {
	path := filepath.Join(filepath.Clean(clusterPath), FileName)
	info, err := os.Stat(path)
	if errors.Is(err, os.ErrNotExist) {
		return Status{State: StateMissing, Path: path, Message: "尚未配置 cluster_token.txt"}, nil
	}
	if err != nil {
		return Status{}, fmt.Errorf("读取令牌状态失败: %w", err)
	}
	if info.IsDir() {
		return Status{}, fmt.Errorf("令牌路径不是文件: %s", path)
	}
	if info.Size() <= 0 {
		return Status{State: StateEmpty, Path: path, ModifiedAt: info.ModTime().Unix(), Message: "cluster_token.txt 存在但为空"}, nil
	}

	content, err := os.ReadFile(path)
	if err != nil {
		return Status{}, fmt.Errorf("读取令牌文件失败: %w", err)
	}
	normalized := bytes.TrimPrefix(content, []byte{0xEF, 0xBB, 0xBF})
	if len(bytes.TrimSpace(normalized)) == 0 {
		return Status{State: StateEmpty, Path: path, Size: info.Size(), ModifiedAt: info.ModTime().Unix(), Message: "cluster_token.txt 只包含空白内容"}, nil
	}
	return configuredStatus(path, info.Size(), info.ModTime()), nil
}

// Save writes a Klei cluster token without ever returning the secret value.
// The temporary file is created in the Cluster directory and is replaced only
// after a successful fsync. File mode is 0600 on platforms that honor POSIX
// permission bits; Windows still receives the narrowest mode Go can request.
func Save(clusterPath, value string) (Status, error) {
	content, err := sanitize([]byte(value))
	if err != nil {
		return Status{}, err
	}
	return saveBytes(clusterPath, content)
}

// ValidateBytes validates token content without writing it or returning the secret.
func ValidateBytes(content []byte) error {
	_, err := sanitize(content)
	return err
}

// ImportBytes stores token content received from a trusted in-memory source such
// as the Klei official configuration ZIP parser. The secret is sanitized and is
// never returned to the caller.
func ImportBytes(clusterPath string, content []byte) (Status, error) {
	content, err := sanitize(content)
	if err != nil {
		return Status{}, err
	}
	return saveBytes(clusterPath, content)
}

func Import(clusterPath, sourcePath string) (Status, error) {
	sourcePath = filepath.Clean(strings.TrimSpace(sourcePath))
	if sourcePath == "." || sourcePath == "" {
		return Status{}, errors.New("缺少令牌文件路径")
	}
	info, err := os.Stat(sourcePath)
	if err != nil {
		return Status{}, fmt.Errorf("读取令牌文件失败: %w", err)
	}
	if info.IsDir() {
		return Status{}, errors.New("请选择 cluster_token.txt 文件，而不是文件夹")
	}
	if info.Size() > maxTokenBytes {
		return Status{}, ErrTokenTooLong
	}
	content, err := os.ReadFile(sourcePath)
	if err != nil {
		return Status{}, fmt.Errorf("读取令牌文件失败: %w", err)
	}
	content, err = sanitize(content)
	if err != nil {
		return Status{}, err
	}
	return saveBytes(clusterPath, content)
}

func sanitize(content []byte) ([]byte, error) {
	content = bytes.TrimPrefix(content, []byte{0xEF, 0xBB, 0xBF})
	content = bytes.TrimSpace(content)
	if len(content) == 0 {
		return nil, ErrEmptyToken
	}
	if len(content) > maxTokenBytes {
		return nil, ErrTokenTooLong
	}
	if bytes.IndexByte(content, 0) >= 0 {
		return nil, errors.New("令牌内容包含无效的 NUL 字节")
	}
	// Klei accepts the token as text. Normalize the final newline so the file is
	// predictable while preserving every non-whitespace token character.
	return append(content, '\n'), nil
}

func saveBytes(clusterPath string, content []byte) (Status, error) {
	clusterPath = filepath.Clean(strings.TrimSpace(clusterPath))
	info, err := os.Stat(clusterPath)
	if err != nil || !info.IsDir() {
		if err == nil {
			err = errors.New("not a directory")
		}
		return Status{}, fmt.Errorf("Cluster 目录不可用: %w", err)
	}
	target := filepath.Join(clusterPath, FileName)

	temp, err := os.CreateTemp(clusterPath, ".cluster-token-*.tmp")
	if err != nil {
		return Status{}, fmt.Errorf("创建令牌临时文件失败: %w", err)
	}
	tempPath := temp.Name()
	defer os.Remove(tempPath)

	if err := temp.Chmod(0o600); err != nil {
		temp.Close()
		return Status{}, fmt.Errorf("设置令牌文件权限失败: %w", err)
	}
	if _, err := temp.Write(content); err != nil {
		temp.Close()
		return Status{}, fmt.Errorf("写入令牌失败: %w", err)
	}
	if err := temp.Sync(); err != nil {
		temp.Close()
		return Status{}, fmt.Errorf("同步令牌文件失败: %w", err)
	}
	if err := temp.Close(); err != nil {
		return Status{}, fmt.Errorf("关闭令牌临时文件失败: %w", err)
	}

	if err := platformfiles.AtomicReplace(tempPath, target); err != nil {
		return Status{}, fmt.Errorf("保存令牌失败: %w", err)
	}
	_ = os.Chmod(target, 0o600)

	status, err := Inspect(clusterPath)
	if err != nil {
		return Status{}, err
	}
	return status, nil
}
