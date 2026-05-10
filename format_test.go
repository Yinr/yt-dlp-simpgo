package main

import "testing"

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		name string
		in   int64
		want string
	}{
		{"bytes", 512, "512 B"},
		{"one kb", 1024, "1.0 KB"},
		{"mb", 1536 * 1024, "1.5 MB"},
		{"gb", 2 * 1024 * 1024 * 1024, "2.0 GB"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := formatBytes(tt.in); got != tt.want {
				t.Fatalf("formatBytes(%d) = %s, want %s", tt.in, got, tt.want)
			}
		})
	}
}

func TestLogMarker(t *testing.T) {
	got := logMarker("下载完成")
	want := "========== 下载完成 =========="
	if got != want {
		t.Fatalf("logMarker() = %q, want %q", got, want)
	}
}
