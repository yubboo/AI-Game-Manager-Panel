//go:build windows

package platformhttp

import (
	"fmt"
	"syscall"
	"unsafe"
)

var (
	shell32          = syscall.NewLazyDLL("shell32.dll")
	procShellExecute = shell32.NewProc("ShellExecuteW")
)

func openSystemBrowser(target string) error {
	verb, _ := syscall.UTF16PtrFromString("open")
	value, err := syscall.UTF16PtrFromString(target)
	if err != nil {
		return err
	}
	result, _, callErr := procShellExecute.Call(
		0,
		uintptr(unsafe.Pointer(verb)),
		uintptr(unsafe.Pointer(value)),
		0,
		0,
		1,
	)
	if result <= 32 {
		return fmt.Errorf("ShellExecuteW failed (%d): %v", result, callErr)
	}
	return nil
}
