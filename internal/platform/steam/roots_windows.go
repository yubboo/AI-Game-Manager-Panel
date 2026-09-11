//go:build windows

package steam

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"syscall"
)

type registryQuery struct {
	root  syscall.Handle
	key   string
	value string
}

func platformSteamRoots(ctx context.Context) []string {
	roots := []string{}

	if explicit := strings.TrimSpace(os.Getenv("STEAM_PATH")); explicit != "" {
		roots = append(roots, explicit)
	}

	// Read Windows Registry directly through the Win32 API.
	// Do not shell out to reg.exe: AI Game Manager Panel is a GUI application and spawning
	// console-subsystem tools can cause a visible black console flash.
	queries := []registryQuery{
		{root: syscall.HKEY_CURRENT_USER, key: `Software\Valve\Steam`, value: "SteamPath"},
		{root: syscall.HKEY_LOCAL_MACHINE, key: `SOFTWARE\WOW6432Node\Valve\Steam`, value: "InstallPath"},
		{root: syscall.HKEY_LOCAL_MACHINE, key: `SOFTWARE\Valve\Steam`, value: "InstallPath"},
	}
	for _, query := range queries {
		if ctx != nil && ctx.Err() != nil {
			break
		}
		if value := queryRegistryString(query.root, query.key, query.value); value != "" {
			roots = append(roots, value)
		}
	}

	if programFilesX86 := os.Getenv("ProgramFiles(x86)"); programFilesX86 != "" {
		roots = append(roots, filepath.Join(programFilesX86, "Steam"))
	}
	if programFiles := os.Getenv("ProgramFiles"); programFiles != "" {
		roots = append(roots, filepath.Join(programFiles, "Steam"))
	}

	return roots
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
	defer syscall.RegCloseKey(key) //nolint:errcheck -- nothing useful can be done on close failure

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

	value := strings.TrimSpace(decodeRegistryUTF16LE(buffer[:size]))
	if valueType == syscall.REG_EXPAND_SZ {
		value = os.ExpandEnv(value)
	}
	return value
}
