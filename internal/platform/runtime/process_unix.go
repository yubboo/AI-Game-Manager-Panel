//go:build linux || darwin

package platformruntime

import (
	"errors"
	"os"
	"os/exec"
	"syscall"
)

func configureProcess(cmd *exec.Cmd, showWindow bool) {
	_ = showWindow
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}

func terminateProcess(process *os.Process) error {
	if process == nil {
		return ErrSessionNotRunning
	}
	if err := syscall.Kill(-process.Pid, syscall.SIGTERM); err != nil && !errors.Is(err, syscall.ESRCH) {
		return process.Signal(syscall.SIGTERM)
	}
	return nil
}

func killProcess(process *os.Process) error {
	if process == nil {
		return ErrSessionNotRunning
	}
	if err := syscall.Kill(-process.Pid, syscall.SIGKILL); err != nil && !errors.Is(err, syscall.ESRCH) {
		return process.Kill()
	}
	return nil
}
