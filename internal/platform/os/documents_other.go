//go:build !windows

package platformos

import (
	"os"
	"path/filepath"
)

func DocumentsDir() string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return "Documents"
	}
	return filepath.Join(home, "Documents")
}
