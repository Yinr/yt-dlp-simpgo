package main

import "testing"

func TestFriendlyYtDlpLine(t *testing.T) {
	tests := []struct {
		name        string
		line        string
		want        string
		rewriteLast bool
		ok          bool
	}{
		{
			name:        "download progress",
			line:        "[download]  42.1% of 25.0MiB at 1.2MiB/s ETA 00:12",
			want:        "下载中: 42.1%，总大小 25.0MiB，速度 1.2MiB/s，剩余 00:12",
			rewriteLast: true,
			ok:          true,
		},
		{
			name: "destination",
			line: "[download] Destination: video.mp4",
			want: "保存文件: video.mp4",
			ok:   true,
		},
		{
			name: "warning",
			line: "WARNING: something happened",
			want: "WARNING: something happened",
			ok:   true,
		},
		{
			name: "ignored noise",
			line: "Deleting original file video.f137.mp4 (pass -k to keep)",
			ok:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ev := friendlyYtDlpLine(tt.line)
			if ev.Ok != tt.ok {
				t.Fatalf("ok = %v, want %v", ev.Ok, tt.ok)
			}
			if ev.Text != tt.want {
				t.Fatalf("text = %q, want %q", ev.Text, tt.want)
			}
			if ev.RewriteLast != tt.rewriteLast {
				t.Fatalf("rewriteLast = %v, want %v", ev.RewriteLast, tt.rewriteLast)
			}
		})
	}
}
