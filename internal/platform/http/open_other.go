//go:build !windows

package platformhttp

import (
	"fmt"
	"runtime"

	platformruntime "github.com/yubboo/AI-Game-Manager-Panel/internal/platform/runtime"
)

func openSystemBrowser(target string) error {
	name := "xdg-open"
	if runtime.GOOS == "darwin" {
		name = "open"
	}
	if _, err := platformruntime.StartDetached(platformruntime.Spec{Executable: name, Arguments: []string{target}}); err != nil {
		return fmt.Errorf("open system browser: %w", err)
	}
	return nil
}
