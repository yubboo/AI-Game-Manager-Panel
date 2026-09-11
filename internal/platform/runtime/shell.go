package platformruntime

import (
	"context"
	"runtime"
)

// ShellSpec is an explicit opt-in wrapper for a human/admin shell command.
// Domain modules should prefer structured executable+argument calls; XiaoYu may
// reach this only through AGMP's approval/policy layer when arbitrary shell is
// explicitly allowed.
type ShellSpec struct {
	Command            string
	WorkingDirectory   string
	Environment        []string
	ReplaceEnvironment bool
	MaxOutputBytes     int
}

func RunShell(ctx context.Context, spec ShellSpec) (RunResult, error) {
	executable := "sh"
	arguments := []string{"-lc", spec.Command}
	if runtime.GOOS == "windows" {
		executable = "powershell.exe"
		arguments = []string{"-NoLogo", "-NoProfile", "-NonInteractive", "-Command", spec.Command}
	}
	return Run(ctx, RunSpec{
		Spec: Spec{
			Executable:         executable,
			Arguments:          arguments,
			WorkingDirectory:   spec.WorkingDirectory,
			Environment:        spec.Environment,
			ReplaceEnvironment: spec.ReplaceEnvironment,
		},
		MaxOutputBytes: spec.MaxOutputBytes,
	})
}
