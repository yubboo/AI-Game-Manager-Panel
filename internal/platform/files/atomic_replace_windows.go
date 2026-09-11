//go:build windows

package platformfiles

import (
	"fmt"
	"syscall"
	"unsafe"
)

const (
	moveFileReplaceExisting = 0x1
	moveFileWriteThrough    = 0x8
)

var (
	kernel32MoveFile = syscall.NewLazyDLL("kernel32.dll")
	procMoveFileExW  = kernel32MoveFile.NewProc("MoveFileExW")
)

// Replace atomically replaces target with source on Windows and asks the OS to
// flush the rename metadata before returning.
func AtomicReplace(source, target string) error {
	sourcePtr, err := syscall.UTF16PtrFromString(source)
	if err != nil {
		return err
	}
	targetPtr, err := syscall.UTF16PtrFromString(target)
	if err != nil {
		return err
	}
	result, _, callErr := procMoveFileExW.Call(
		uintptr(unsafe.Pointer(sourcePtr)),
		uintptr(unsafe.Pointer(targetPtr)),
		uintptr(moveFileReplaceExisting|moveFileWriteThrough),
	)
	if result == 0 {
		return fmt.Errorf("MoveFileExW: %w", callErr)
	}
	return nil
}
