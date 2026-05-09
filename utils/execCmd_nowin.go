//go:build !windows

package utils

import (
	"os/exec"
)

func ExecCmd(exePath string, arg ...string) *exec.Cmd {
	return exec.Command(exePath, arg...)
}
