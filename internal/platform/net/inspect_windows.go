//go:build windows

package platformnet

import (
	"errors"
	"path/filepath"
	"syscall"
	"unsafe"
)

const (
	afInet                  = 2
	errorInsufficientBuffer = syscall.Errno(122)
	udpTableOwnerPID        = 1
	tcpTableOwnerPIDAll     = 5
	processQueryLimitedInfo = 0x1000
	processTerminate        = 0x0001
	maxProcessImagePath     = 32768
)

var (
	iphlpapi                  = syscall.NewLazyDLL("iphlpapi.dll")
	procGetExtendedUdpTable   = iphlpapi.NewProc("GetExtendedUdpTable")
	procGetExtendedTcpTable   = iphlpapi.NewProc("GetExtendedTcpTable")
	kernel32                  = syscall.NewLazyDLL("kernel32.dll")
	procOpenProcess           = kernel32.NewProc("OpenProcess")
	procCloseHandle           = kernel32.NewProc("CloseHandle")
	procQueryFullProcessImage = kernel32.NewProc("QueryFullProcessImageNameW")
	procTerminateProcess      = kernel32.NewProc("TerminateProcess")
)

type mibUDPRowOwnerPID struct {
	LocalAddr uint32
	LocalPort uint32
	OwningPID uint32
}

type mibTCPRowOwnerPID struct {
	State      uint32
	LocalAddr  uint32
	LocalPort  uint32
	RemoteAddr uint32
	RemotePort uint32
	OwningPID  uint32
}

func InspectUDP(ports []int) ([]Status, error) {
	ports = UniquePorts(ports)
	wanted := make(map[int]struct{}, len(ports))
	result := make(map[int]*Status, len(ports))
	for _, port := range ports {
		wanted[port] = struct{}{}
		result[port] = &Status{Port: port, Owners: []Owner{}}
	}
	if len(ports) == 0 {
		return []Status{}, nil
	}
	owners, err := udpOwners(wanted)
	if err != nil {
		return nil, err
	}
	for _, owner := range owners {
		status := result[owner.Port]
		if status == nil {
			continue
		}
		status.Occupied = true
		status.Owners = append(status.Owners, owner)
	}
	out := make([]Status, 0, len(ports))
	for _, port := range ports {
		out = append(out, *result[port])
	}
	return out, nil
}

func Inspect(ports []int) ([]Status, error) {
	ports = UniquePorts(ports)
	wanted := make(map[int]struct{}, len(ports))
	result := make(map[int]*Status, len(ports))
	for _, port := range ports {
		wanted[port] = struct{}{}
		result[port] = &Status{Port: port, Owners: []Owner{}}
	}
	if len(ports) == 0 {
		return []Status{}, nil
	}
	udp, err := udpOwners(wanted)
	if err != nil {
		return nil, err
	}
	tcp, err := tcpOwners(wanted)
	if err != nil {
		return nil, err
	}
	for _, owner := range append(udp, tcp...) {
		status := result[owner.Port]
		if status == nil {
			continue
		}
		status.Occupied = true
		status.Owners = append(status.Owners, owner)
	}
	out := make([]Status, 0, len(ports))
	for _, port := range ports {
		out = append(out, *result[port])
	}
	return out, nil
}

func udpOwners(wanted map[int]struct{}) ([]Owner, error) {
	buffer, err := tableBuffer(procGetExtendedUdpTable, udpTableOwnerPID)
	if err != nil {
		return nil, err
	}
	if len(buffer) < 4 {
		return []Owner{}, nil
	}
	count := *(*uint32)(unsafe.Pointer(&buffer[0]))
	rowSize := unsafe.Sizeof(mibUDPRowOwnerPID{})
	result := make([]Owner, 0)
	for i := uint32(0); i < count; i++ {
		offset := uintptr(4) + uintptr(i)*rowSize
		if offset+rowSize > uintptr(len(buffer)) {
			break
		}
		row := *(*mibUDPRowOwnerPID)(unsafe.Pointer(&buffer[offset]))
		port := networkPort(row.LocalPort)
		if _, ok := wanted[port]; !ok {
			continue
		}
		result = append(result, ownerFor(port, ProtocolUDP, int(row.OwningPID)))
	}
	return result, nil
}

func tcpOwners(wanted map[int]struct{}) ([]Owner, error) {
	buffer, err := tableBuffer(procGetExtendedTcpTable, tcpTableOwnerPIDAll)
	if err != nil {
		return nil, err
	}
	if len(buffer) < 4 {
		return []Owner{}, nil
	}
	count := *(*uint32)(unsafe.Pointer(&buffer[0]))
	rowSize := unsafe.Sizeof(mibTCPRowOwnerPID{})
	result := make([]Owner, 0)
	for i := uint32(0); i < count; i++ {
		offset := uintptr(4) + uintptr(i)*rowSize
		if offset+rowSize > uintptr(len(buffer)) {
			break
		}
		row := *(*mibTCPRowOwnerPID)(unsafe.Pointer(&buffer[offset]))
		port := networkPort(row.LocalPort)
		if _, ok := wanted[port]; !ok {
			continue
		}
		result = append(result, ownerFor(port, ProtocolTCP, int(row.OwningPID)))
	}
	return result, nil
}

func tableBuffer(proc *syscall.LazyProc, class uintptr) ([]byte, error) {
	var size uint32
	ret, _, callErr := proc.Call(0, uintptr(unsafe.Pointer(&size)), 1, afInet, class, 0)
	if ret != 0 && syscall.Errno(ret) != errorInsufficientBuffer {
		if callErr != nil && callErr != syscall.Errno(0) {
			return nil, callErr
		}
		return nil, syscall.Errno(ret)
	}
	if size == 0 {
		return []byte{}, nil
	}
	buffer := make([]byte, size)
	ret, _, callErr = proc.Call(uintptr(unsafe.Pointer(&buffer[0])), uintptr(unsafe.Pointer(&size)), 1, afInet, class, 0)
	if ret != 0 {
		if callErr != nil && callErr != syscall.Errno(0) {
			return nil, callErr
		}
		return nil, syscall.Errno(ret)
	}
	return buffer, nil
}

func networkPort(value uint32) int {
	return int(((value & 0xff) << 8) | ((value >> 8) & 0xff))
}

func ownerFor(port int, protocol Protocol, pid int) Owner {
	path, _ := ProcessPath(pid)
	return Owner{Port: port, Protocol: protocol, PID: pid, ProcessPath: path, ProcessName: filepath.Base(path)}
}

func ProcessPath(pid int) (string, error) {
	if pid <= 0 {
		return "", errors.New("invalid process id")
	}
	handle, _, callErr := procOpenProcess.Call(processQueryLimitedInfo, 0, uintptr(uint32(pid)))
	if handle == 0 {
		if callErr != nil && callErr != syscall.Errno(0) {
			return "", callErr
		}
		return "", errors.New("OpenProcess failed")
	}
	defer procCloseHandle.Call(handle)
	buffer := make([]uint16, maxProcessImagePath)
	size := uint32(len(buffer))
	ret, _, callErr := procQueryFullProcessImage.Call(handle, 0, uintptr(unsafe.Pointer(&buffer[0])), uintptr(unsafe.Pointer(&size)))
	if ret == 0 {
		if callErr != nil && callErr != syscall.Errno(0) {
			return "", callErr
		}
		return "", errors.New("QueryFullProcessImageNameW failed")
	}
	return syscall.UTF16ToString(buffer[:size]), nil
}

func IsProcessAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	handle, _, _ := procOpenProcess.Call(processQueryLimitedInfo, 0, uintptr(uint32(pid)))
	if handle == 0 {
		return false
	}
	procCloseHandle.Call(handle)
	return true
}

func TerminatePID(pid int) error {
	if pid <= 0 {
		return errors.New("invalid process id")
	}
	handle, _, callErr := procOpenProcess.Call(processTerminate|processQueryLimitedInfo, 0, uintptr(uint32(pid)))
	if handle == 0 {
		if callErr != nil && callErr != syscall.Errno(0) {
			return callErr
		}
		return errors.New("OpenProcess failed")
	}
	defer procCloseHandle.Call(handle)
	ret, _, callErr := procTerminateProcess.Call(handle, 1)
	if ret == 0 {
		if callErr != nil && callErr != syscall.Errno(0) {
			return callErr
		}
		return errors.New("TerminateProcess failed")
	}
	return nil
}
