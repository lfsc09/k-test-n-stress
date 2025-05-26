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

func formatDurationMetrics(duration time.Duration) string {
	switch {
	case duration < time.Millisecond:
		return fmt.Sprintf(" [%.2fµs] ", float64(duration.Microseconds()))
	case duration < time.Second:
		return fmt.Sprintf(" [%.2fms] ", float64(duration.Milliseconds()))
	default:
		return fmt.Sprintf(" [%.2fs] ", duration.Seconds())
	}
}

func formatSizeMetrics(size int64) string {
	switch {
	case size >= gb:
		return fmt.Sprintf(" [%.2f GB] ", float64(size)/float64(gb))
	case size >= mb:
		return fmt.Sprintf(" [%.2f MB] ", float64(size)/float64(mb))
	case size >= kb:
		return fmt.Sprintf(" [%.2f KB] ", float64(size)/float64(kb))
	default:
		return fmt.Sprintf(" [%d Bytes] ", size)
	}
}
