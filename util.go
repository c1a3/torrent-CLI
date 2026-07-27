package main

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
)

// THis function allows to convert bytes to human-readable format
func formatBytes(bytes int64) string {
	const (
		KB = 1 << 10
		MB = 1 << 20
		GB = 1 << 30
		TB = 1 << 40
	)

	switch {
	case bytes >= TB:
		return fmt.Sprintf("%.2f TB", float64(bytes)/TB)
	case bytes >= GB:
		return fmt.Sprintf("%.2f GB", float64(bytes)/GB)
	case bytes >= MB:
		return fmt.Sprintf("%.2f MB", float64(bytes)/MB)
	case bytes >= KB:
		return fmt.Sprintf("%.2f KB", float64(bytes)/KB)
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}

// formatSpeed formats a bytes-per-second value into a human-readable speed string.
func formatSpeed(bytesPerSec int64) string {
	if bytesPerSec <= 0 {
		return "0 B/s"
	}
	return formatBytes(bytesPerSec) + "/s"
}

// formatETA calculates and formats the estimated time remaining.
func formatETA(remainingBytes, bytesPerSec int64) string {
	if bytesPerSec <= 0 {
		return "∞"
	}
	seconds := remainingBytes / bytesPerSec
	if seconds < 60 {
		return fmt.Sprintf("%ds", seconds)
	}
	if seconds < 3600 {
		return fmt.Sprintf("%dm %ds", seconds/60, seconds%60)
	}
	hours := seconds / 3600
	minutes := (seconds % 3600) / 60
	return fmt.Sprintf("%dh %dm", hours, minutes)
}

// generateID creates a short random hex ID for identifying torrents.
func generateID() string {
	b := make([]byte, 4)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
