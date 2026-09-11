//go:build windows

package license

import (
	"encoding/binary"
	"os"
	"runtime"
	"strings"
	"syscall"
)

// machineIdentity reads MachineGuid directly from the Windows registry.
// It intentionally does not launch reg.exe / cmd.exe / PowerShell, so GUI
// builds never flash a console window when license state is queried.
func machineIdentity() string {
	subkey, err := syscall.UTF16PtrFromString(`SOFTWARE\Microsoft\Cryptography`)
	if err == nil {
		var key syscall.Handle
		if err = syscall.RegOpenKeyEx(syscall.HKEY_LOCAL_MACHINE, subkey, 0, syscall.KEY_READ, &key); err == nil {
			defer syscall.RegCloseKey(key)
			name, nameErr := syscall.UTF16PtrFromString("MachineGuid")
			if nameErr == nil {
				var valueType uint32
				var size uint32
				if err = syscall.RegQueryValueEx(key, name, nil, &valueType, nil, &size); err == nil && size >= 2 {
					data := make([]byte, size)
					if err = syscall.RegQueryValueEx(key, name, nil, &valueType, &data[0], &size); err == nil {
						units := make([]uint16, 0, len(data)/2)
						for i := 0; i+1 < len(data); i += 2 {
							units = append(units, binary.LittleEndian.Uint16(data[i:i+2]))
						}
						if value := strings.TrimSpace(syscall.UTF16ToString(units)); value != "" {
							return value
						}
					}
				}
			}
		}
	}
	host, _ := os.Hostname()
	return "windows|" + runtime.GOARCH + "|" + host
}
