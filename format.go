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
