package cmd

import (
	"fmt"
	"io"
	"time"
)

type CommandOptions struct {
	Out io.Writer
}

const (
	kb = 1 << 10
	mb = 1 << 20
	gb = 1 << 30
)

// formatDurationMetrics formats a duration in seconds into a human-readable string with appropriate units (µs, ms, s).
func formatDurationMetrics(seconds float64) string {
	switch {
	case seconds < float64(time.Millisecond)/float64(time.Second):
		return fmt.Sprintf("%.2fµs", seconds*1e6)
	case seconds < 1.0:
		return fmt.Sprintf("%.2fms", seconds*1e3)
	default:
		return fmt.Sprintf("%.2fs", seconds)
	}
}

// formatSizeMetrics formats a file size in bytes into a human-readable string with appropriate units (KB, MB, GB).
func formatSizeMetrics(size int64) string {
	switch {
	case size >= gb:
		return fmt.Sprintf("%.2f GB", float64(size)/float64(gb))
	case size >= mb:
		return fmt.Sprintf("%.2f MB", float64(size)/float64(mb))
	case size >= kb:
		return fmt.Sprintf("%.2f KB", float64(size)/float64(kb))
	default:
		return fmt.Sprintf("%d Bytes", size)
	}
}

// formatNumberMetrics formats a large number into a human-readable string with appropriate units (K, M, B, T).
func formatNumberMetrics(number int64) string {
	switch {
	case number >= 1_000_000_000_000:
		return fmt.Sprintf("%.0fTr", float64(number)/1_000_000_000_000)
	case number >= 1_000_000_000:
		return fmt.Sprintf("%.0fBi", float64(number)/1_000_000_000)
	case number >= 1_000_000:
		return fmt.Sprintf("%.0fMi", float64(number)/1_000_000)
	case number >= 1_000:
		return fmt.Sprintf("%.0fK", float64(number)/1_000)
	default:
		return fmt.Sprintf("%d", number)
	}
}
