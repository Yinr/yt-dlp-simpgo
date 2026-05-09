package utils

import (
	"runtime"
	"testing"
)

func TestExecCmd(t *testing.T) {
	// 使用 echo 命令测试
	cmd := ExecCmd("echo", "hello")

	if cmd == nil {
		t.Fatal("ExecCmd 返回的命令不应为 nil")
	}

	if len(cmd.Args) != 2 {
		t.Errorf("参数数量应为 2，实际为 %d", len(cmd.Args))
	}

	if cmd.Args[0] != "echo" {
		t.Errorf("第一个参数应为 echo，实际为 %s", cmd.Args[0])
	}

	if cmd.Args[1] != "hello" {
		t.Errorf("第二个参数应为 hello，实际为 %s", cmd.Args[1])
	}
}

func TestExecCmd_WindowsHideWindow(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("跳过 Windows 特定测试")
	}

	cmd := ExecCmd("echo", "test")

	if cmd.SysProcAttr == nil {
		t.Error("Windows 上 SysProcAttr 不应为 nil")
	}

	if !cmd.SysProcAttr.HideWindow {
		t.Error("Windows 上 HideWindow 应为 true")
	}
}
