//go:build windows
// +build windows

package utils

import (
	"os/exec"
	"syscall"
)

func ExecCmd(exePath string, arg ...string) (*exec.Cmd, error) {
	cmd := exec.Command(exePath, arg...)

	// hide console window on Windows (affects child process window)
	// if runtime.GOOS == "windows" {
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	// }

	return cmd, nil
}
