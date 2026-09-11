//go:build windows

package steam

import (
	"strings"
	"syscall"
	"unsafe"
)

const (
	th32csSnapProcess              = 0x00000002
	processQueryLimitedInformation = 0x1000
	maxPath                        = 260
)

type processEntry32 struct {
	Size            uint32
	CntUsage        uint32
	ProcessID       uint32
	DefaultHeapID   uintptr
	ModuleID        uint32
	CntThreads      uint32
	ParentProcessID uint32
	PriClassBase    int32
	Flags           uint32
	ExeFile         [maxPath]uint16
}

type ioCounters struct {
	ReadOperationCount  uint64
	WriteOperationCount uint64
	OtherOperationCount uint64
	ReadTransferCount   uint64
	WriteTransferCount  uint64
	OtherTransferCount  uint64
}

var (
	kernel32ClientIO           = syscall.NewLazyDLL("kernel32.dll")
	procCreateToolhelpSnapshot = kernel32ClientIO.NewProc("CreateToolhelp32Snapshot")
	procProcess32FirstW        = kernel32ClientIO.NewProc("Process32FirstW")
	procProcess32NextW         = kernel32ClientIO.NewProc("Process32NextW")
	procOpenProcessClientIO    = kernel32ClientIO.NewProc("OpenProcess")
	procGetProcessIoCounters   = kernel32ClientIO.NewProc("GetProcessIoCounters")
	procCloseHandleClientIO    = kernel32ClientIO.NewProc("CloseHandle")
)

// ClientReadBytes returns the cumulative OS read-transfer bytes for the Steam
// client processes that perform local content validation on Windows. AI Game Manager Panel
// samples this only while the target AppID is in Steam's Validating state, so
// it can derive a live, approximate verification percentage without inventing
// a timer-based progress bar.
//
// The value is deliberately process-wide instead of AppID-specific because
// Steam does not expose a stable public per-AppID verification byte counter.
// StateFlags remain the gate that decides when the counter is allowed to
// contribute to a target validation task.
func ClientReadBytes() (uint64, bool) {
	snapshot, _, _ := procCreateToolhelpSnapshot.Call(th32csSnapProcess, 0)
	if snapshot == 0 || snapshot == ^uintptr(0) {
		return 0, false
	}
	defer procCloseHandleClientIO.Call(snapshot) //nolint:errcheck

	entry := processEntry32{Size: uint32(unsafe.Sizeof(processEntry32{}))}
	ok, _, _ := procProcess32FirstW.Call(snapshot, uintptr(unsafe.Pointer(&entry)))
	if ok == 0 {
		return 0, false
	}

	var total uint64
	found := false
	for {
		name := strings.ToLower(syscall.UTF16ToString(entry.ExeFile[:]))
		if name == "steam.exe" || name == "steamservice.exe" {
			handle, _, _ := procOpenProcessClientIO.Call(processQueryLimitedInformation, 0, uintptr(entry.ProcessID))
			if handle != 0 {
				var counters ioCounters
				result, _, _ := procGetProcessIoCounters.Call(handle, uintptr(unsafe.Pointer(&counters)))
				procCloseHandleClientIO.Call(handle) //nolint:errcheck
				if result != 0 {
					total += counters.ReadTransferCount
					found = true
				}
			}
		}

		entry.Size = uint32(unsafe.Sizeof(processEntry32{}))
		next, _, _ := procProcess32NextW.Call(snapshot, uintptr(unsafe.Pointer(&entry)))
		if next == 0 {
			break
		}
	}
	return total, found
}
