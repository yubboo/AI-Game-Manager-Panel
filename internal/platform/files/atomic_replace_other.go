//go:build !windows

package platformfiles

import "os"

func AtomicReplace(source, target string) error { return os.Rename(source, target) }
