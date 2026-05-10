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

func TestFormatProgress(t *testing.T) {
	tests := []struct {
		name      string
		prefix    string
		received  int64
		total     int64
		wantText  string
		wantPct   int
	}{
		{"with total", "下载", 512, 1024, "下载: 50% (512 B/1.0 KB)", 50},
		{"zero total", "下载", 512, 0, "下载: 512 B", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			text, pct := formatProgress(tt.prefix, tt.received, tt.total)
			if text != tt.wantText {
				t.Fatalf("text = %q, want %q", text, tt.wantText)
			}
			if pct != tt.wantPct {
				t.Fatalf("pct = %d, want %d", pct, tt.wantPct)
			}
		})
	}
}
