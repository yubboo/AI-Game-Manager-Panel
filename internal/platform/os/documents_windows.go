//go:build windows

package platformos

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"unicode/utf16"
)

// DocumentsDir returns the user's actual Windows "Documents" special-folder
// location. It follows the same source used by the Python baseline instead of
// assuming %USERPROFILE%\Documents, so redirected folders on another drive are
// preserved.
func DocumentsDir() string {
	const keyPath = `Software\Microsoft\Windows\CurrentVersion\Explorer\User Shell Folders`
	if value := queryRegistryString(syscall.HKEY_CURRENT_USER, keyPath, "Personal"); value != "" {
		value = os.ExpandEnv(value)
		if info, err := os.Stat(value); err == nil && info.IsDir() {
			return filepath.Clean(value)
		}
	}
	return fallbackDocumentsDir()
}

func fallbackDocumentsDir() string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return "Documents"
	}
	return filepath.Join(home, "Documents")
}

func queryRegistryString(root syscall.Handle, keyPath, valueName string) string {
	keyPathPtr, err := syscall.UTF16PtrFromString(keyPath)
	if err != nil {
		return ""
	}
	var key syscall.Handle
	if err := syscall.RegOpenKeyEx(root, keyPathPtr, 0, syscall.KEY_READ, &key); err != nil {
		return ""
	}
	defer syscall.RegCloseKey(key) //nolint:errcheck

	valueNamePtr, err := syscall.UTF16PtrFromString(valueName)
	if err != nil {
		return ""
	}
	var valueType uint32
	var size uint32
	if err := syscall.RegQueryValueEx(key, valueNamePtr, nil, &valueType, nil, &size); err != nil || size == 0 {
		return ""
	}
	if valueType != syscall.REG_SZ && valueType != syscall.REG_EXPAND_SZ {
		return ""
	}
	buffer := make([]byte, size)
	if err := syscall.RegQueryValueEx(key, valueNamePtr, nil, &valueType, &buffer[0], &size); err != nil {
		return ""
	}
	return strings.TrimSpace(decodeRegistryUTF16LE(buffer[:size]))
}

func decodeRegistryUTF16LE(data []byte) string {
	if len(data)%2 != 0 {
		data = data[:len(data)-1]
	}
	values := make([]uint16, 0, len(data)/2)
	for i := 0; i+1 < len(data); i += 2 {
		value := binary.LittleEndian.Uint16(data[i : i+2])
		if value == 0 {
			break
		}
		values = append(values, value)
	}
	return string(utf16.Decode(values))
}
