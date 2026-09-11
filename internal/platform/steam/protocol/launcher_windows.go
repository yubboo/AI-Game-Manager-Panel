//go:build windows

package protocol

import (
	"context"
	"fmt"
	"syscall"
	"unsafe"
)

var (
	shell32          = syscall.NewLazyDLL("shell32.dll")
	procShellExecute = shell32.NewProc("ShellExecuteW")
)

func openURI(ctx context.Context, uri string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	action, _ := syscall.UTF16PtrFromString("open")
	target, err := syscall.UTF16PtrFromString(uri)
	if err != nil {
		return err
	}
	result, _, callErr := procShellExecute.Call(
		0,
		uintptr(unsafe.Pointer(action)),
		uintptr(unsafe.Pointer(target)),
		0,
		0,
		1,
	)
	if result <= 32 {
		return fmt.Errorf("Steam protocol launch failed (%d): %v", result, callErr)
	}
	return nil
}
