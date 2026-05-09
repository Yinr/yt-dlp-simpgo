//go:build windows

package utils

import (
	"os/exec"
	"syscall"
)

func ExecCmd(exePath string, arg ...string) *exec.Cmd {
	cmd := exec.Command(exePath, arg...)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	return cmd
}
