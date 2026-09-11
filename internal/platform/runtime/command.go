package platformruntime

import (
	"errors"
	"os/exec"
	"strings"
)

var ErrExecutableRequired = errors.New("runtime executable is required")

func newCommand(spec Spec) (*exec.Cmd, error) {
	if strings.TrimSpace(spec.Executable) == "" {
		return nil, ErrExecutableRequired
	}
	cmd := exec.Command(spec.Executable, spec.Arguments...)
	cmd.Dir = spec.WorkingDirectory
	if spec.ReplaceEnvironment {
		cmd.Env = append([]string(nil), spec.Environment...)
	} else if len(spec.Environment) > 0 {
		// Cmd.Environ includes the process environment and the effective PWD.
		// os/exec uses the last value for duplicate keys, so caller overrides win.
		cmd.Env = append(cmd.Environ(), spec.Environment...)
	}
	configureProcess(cmd, spec.ShowWindow)
	return cmd, nil
}
