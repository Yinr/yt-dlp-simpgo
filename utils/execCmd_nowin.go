//go:build !windows
// +build !windows

package utils

import (
	"os/exec"
)

func ExecCmd(exePath string, arg ...string) (*exec.Cmd, error) {
	cmd := exec.Command(exePath, arg...)

	return cmd, nil
}
