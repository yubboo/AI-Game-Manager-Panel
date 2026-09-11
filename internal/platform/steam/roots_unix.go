//go:build !windows

package steam

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
)

func platformSteamRoots(_ context.Context) []string {
	home, _ := os.UserHomeDir()
	if runtime.GOOS == "darwin" {
		return []string{filepath.Join(home, "Library", "Application Support", "Steam")}
	}
	return []string{
		filepath.Join(home, ".local", "share", "Steam"),
		filepath.Join(home, ".steam", "steam"),
	}
}
