//go:build windows

package platformruntime

import (
	"os"
	"os/exec"
	"syscall"
)

const (
	createNewProcessGroup = 0x00000200
	createNoWindow        = 0x08000000
)

func configureProcess(cmd *exec.Cmd, showWindow bool) {
	flags := uint32(createNewProcessGroup)
	if !showWindow {
		flags |= createNoWindow
	}
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: !showWindow, CreationFlags: flags}
}

func terminateProcess(process *os.Process) error { return process.Kill() }
func killProcess(process *os.Process) error      { return process.Kill() }
