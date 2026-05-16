package mock

import (
	"context"
	"fmt"
	"io"
	"sync/atomic"
	"time"

	"github.com/lfsc09/k-test-n-stress/internal/utils"
)

type DebugStats struct {
	StartTime        time.Time
	GenerateTotal    uint
	Generated        atomic.Uint32
	UsedMemPeakBytes atomic.Uint64
	BytesWritten     atomic.Uint32
}

// StartDebugRoutine launches a goroutine that prints live progress to out (always
// os.Stderr in production) at 200ms intervals. Call the returned stop function to
// terminate the display and print a final newline.
func StartDebugRoutine(ctx context.Context, stats *DebugStats, out io.Writer) (stop func()) {
	innerCtx, cancel := context.WithCancel(ctx)
	done := make(chan struct{})

	render := func(final bool) {
		elapsed := time.Since(stats.StartTime).Seconds()
		generated := stats.Generated.Load()
		outputSizeBytes := stats.BytesWritten.Load()
		percentDone := 0.0
		if stats.GenerateTotal > 0 {
			percentDone = float64(generated) / float64(stats.GenerateTotal) * 100
		}
		usedMemPeakBytes := stats.UsedMemPeakBytes.Load()
		usedMemBytes, reservedMemSystemBytes := utils.MemSnapshotBytes()

		if usedMemBytes > usedMemPeakBytes {
			usedMemPeakBytes = usedMemBytes
			stats.UsedMemPeakBytes.Store(usedMemPeakBytes)
		}

		// '\r' at the start to overwrite the previous line, making it look like a live-updating single line of output
		strOutput := fmt.Sprintf("\r[debug] Elapsed: %s | Progress: %s / %s (%.1f%%) | Mem: %s ⌈%s⌉ [%s] | Output Size: ~%s",
			utils.FormatDurationMetrics(elapsed),
			utils.FormatNumberMetrics(uint64(generated)),
			utils.FormatNumberMetrics(uint64(stats.GenerateTotal)),
			percentDone,
			utils.FormatSizeMetrics(usedMemBytes),
			utils.FormatSizeMetrics(usedMemPeakBytes),
			utils.FormatSizeMetrics(reservedMemSystemBytes),
			utils.FormatSizeMetrics(uint64(outputSizeBytes)),
		)

		fmt.Fprint(out, strOutput)
		if final {
			fmt.Fprintln(out)
		}
	}

	go func() {
		defer close(done)
		ticker := time.NewTicker(200 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				render(false)
			case <-innerCtx.Done():
				render(true)
				return
			}
		}
	}()

	return func() {
		cancel()
		<-done
	}
}
