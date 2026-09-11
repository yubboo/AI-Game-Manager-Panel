//go:build !windows && !linux && !darwin

package platformruntime

import (
	"os"
	"os/exec"
)

func configureProcess(_ *exec.Cmd, _ bool) {}
func terminateProcess(process *os.Process) error {
	if process == nil {
		return ErrSessionNotRunning
	}
	return process.Signal(os.Interrupt)
}
func killProcess(process *os.Process) error {
	if process == nil {
		return ErrSessionNotRunning
	}
	return process.Kill()
}
