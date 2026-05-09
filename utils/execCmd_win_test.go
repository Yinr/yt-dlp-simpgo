//go:build windows

package utils

import "testing"

func TestExecCmd_WindowsHideWindow(t *testing.T) {
	cmd := ExecCmd("echo", "test")

	if cmd.SysProcAttr == nil {
		t.Fatal("Windows 上 SysProcAttr 不应为 nil")
	}

	if !cmd.SysProcAttr.HideWindow {
		t.Error("Windows 上 HideWindow 应为 true")
	}
}
