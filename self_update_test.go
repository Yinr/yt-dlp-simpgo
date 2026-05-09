package main

import (
	"bytes"
	"crypto/sha256"
	"runtime"
	"testing"
)

func TestCompareSemver(t *testing.T) {
	tests := []struct {
		name string
		a    string
		b    string
		want int
		ok   bool
	}{
		{"less", "v0.3.0", "v1.0.0", -1, true},
		{"equal", "1.0.0", "v1.0.0", 0, true},
		{"greater", "v1.2.0", "v1.1.9", 1, true},
		{"dirty version", "v1.0.0-dirty", "v1.0.1", 0, false},
		{"dev version", "dev", "v1.0.0", 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := compareSemver(tt.a, tt.b)
			if ok != tt.ok {
				t.Fatalf("ok = %v, want %v", ok, tt.ok)
			}
			if got != tt.want {
				t.Fatalf("compare = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestCurrentPlatformAssetName(t *testing.T) {
	name := currentPlatformAssetName()
	want := "yt-dlp-simpgo-" + runtime.GOOS + "-" + runtime.GOARCH
	if runtime.GOOS == "windows" {
		want += ".exe"
	}
	if name != want {
		t.Fatalf("asset name = %s, want %s", name, want)
	}
}

func TestVerifyAssetDigest(t *testing.T) {
	sum := sha256.Sum256([]byte("hello"))
	if err := verifyAssetDigest("sha256:2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824", sum[:]); err != nil {
		t.Fatalf("expected digest to match: %v", err)
	}

	if err := verifyAssetDigest("sha256:bad", sum[:]); err == nil {
		t.Fatal("expected digest mismatch")
	}

	if err := verifyAssetDigest("", sum[:]); err != nil {
		t.Fatalf("empty digest should be ignored: %v", err)
	}
}

func TestCopyWithProgress(t *testing.T) {
	var dst bytes.Buffer
	var calls int
	var lastReceived int64
	var lastTotal int64
	err := copyWithProgress(&dst, bytes.NewBufferString("hello"), 5, func(received, total int64) {
		calls++
		lastReceived = received
		lastTotal = total
	})
	if err != nil {
		t.Fatalf("copyWithProgress failed: %v", err)
	}
	if dst.String() != "hello" {
		t.Fatalf("copied content = %q", dst.String())
	}
	if calls == 0 {
		t.Fatal("progress callback was not called")
	}
	if lastReceived != 5 || lastTotal != 5 {
		t.Fatalf("progress = (%d, %d), want (5, 5)", lastReceived, lastTotal)
	}
}
