package utils

import "runtime"

// MemSnapshotBytes returns the currently used memory and total memory in bytes by forcing a GC and reading the memory stats.
func MemSnapshotBytes() (uint64, uint64) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	return m.Alloc, m.Sys
}
