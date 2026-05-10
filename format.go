package main

import "fmt"

func formatBytes(n int64) string {
	if n < 1024 {
		return fmt.Sprintf("%d B", n)
	}

	value := float64(n)
	units := []string{"KB", "MB", "GB", "TB"}
	for i, unit := range units {
		value /= 1024
		if value < 1024 || i == len(units)-1 {
			return fmt.Sprintf("%.1f %s", value, unit)
		}
	}

	return fmt.Sprintf("%d B", n)
}

func formatProgress(prefix string, received, total int64) (string, int) {
	if total > 0 {
		pct := int(float64(received) * 100.0 / float64(total))
		return fmt.Sprintf("%s: %d%% (%s/%s)", prefix, pct, formatBytes(received), formatBytes(total)), pct
	}
	return fmt.Sprintf("%s: %s", prefix, formatBytes(received)), 0
}

func logMarker(title string) string {
	return "========== " + title + " =========="
}
