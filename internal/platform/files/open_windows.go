//go:build windows

package platformfiles

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"unsafe"
)

var (
	shell32          = syscall.NewLazyDLL("shell32.dll")
	procShellExecute = shell32.NewProc("ShellExecuteW")
)

// OpenFolder 使用 Windows Explorer 打开指定目录。
func OpenFolder(path string) error {
	path = filepath.Clean(path)
	if err := os.MkdirAll(path, 0o755); err != nil {
		return err
	}
	verb, _ := syscall.UTF16PtrFromString("open")
	value, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return err
	}
	result, _, callErr := procShellExecute.Call(0, uintptr(unsafe.Pointer(verb)), uintptr(unsafe.Pointer(value)), 0, 0, 1)
	if result <= 32 {
		return fmt.Errorf("打开文件夹失败 (%d): %v", result, callErr)
	}
	return nil
}
