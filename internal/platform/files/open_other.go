//go:build !windows

package platformfiles

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	platformruntime "github.com/yubboo/AI-Game-Manager-Panel/internal/platform/runtime"
)

// OpenFolder 在非 Windows 开发环境使用系统文件管理器打开目录。
func OpenFolder(path string) error {
	path = filepath.Clean(path)
	if err := os.MkdirAll(path, 0o755); err != nil {
		return err
	}
	command := "xdg-open"
	if runtime.GOOS == "darwin" {
		command = "open"
	}
	if _, err := platformruntime.StartDetached(platformruntime.Spec{Executable: command, Arguments: []string{path}}); err != nil {
		return fmt.Errorf("打开文件夹失败: %w", err)
	}
	return nil
}
