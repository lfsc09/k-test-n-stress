package mocker

import (
	"fmt"
	"strings"
	"time"
)

// calculateChecksum calculates checksum for CPF and CNPJ
func calculateChecksum(digits []int, multipliers []int) int {
	sum := 0
	for i := range digits {
		sum += digits[i] * multipliers[i]
	}
	remainder := sum % 11
	if remainder < 2 {
		return 0
	}
	return 11 - remainder
}

// extractRegex extracts raw regex string from /.../ and unescapes \/ → /
func extractRegex(value string) (string, error) {
	if !strings.HasPrefix(value, "/") || !strings.HasSuffix(value, "/") {
		return "", fmt.Errorf("Value '%s' must be wrapped in /.../", value)
	}
	trimmed := value[1 : len(value)-1]
	unescaped := strings.ReplaceAll(trimmed, `\/`, `/`)
	return unescaped, nil
}

// formatDatetime formats a time.Time value using a human-readable token format string.
// Supported tokens: YYYY, MM, DD, hh, mm, ss, sss (milliseconds).
// sss is replaced before ss to avoid partial collision.
func formatDatetime(t time.Time, format string) string {
	result := format
	result = strings.ReplaceAll(result, "sss", fmt.Sprintf("%03d", t.Nanosecond()/1_000_000))
	result = strings.ReplaceAll(result, "YYYY", fmt.Sprintf("%04d", t.Year()))
	result = strings.ReplaceAll(result, "MM", fmt.Sprintf("%02d", int(t.Month())))
	result = strings.ReplaceAll(result, "DD", fmt.Sprintf("%02d", t.Day()))
	result = strings.ReplaceAll(result, "hh", fmt.Sprintf("%02d", t.Hour()))
	result = strings.ReplaceAll(result, "mm", fmt.Sprintf("%02d", t.Minute()))
	result = strings.ReplaceAll(result, "ss", fmt.Sprintf("%02d", t.Second()))
	return result
}

// parseDateOnly parses a date string in YYYY-MM-DD format.
func parseDateOnly(s string) (time.Time, error) {
	return time.Parse("2006-01-02", s)
}

// parseTimeOnly parses a time string in hh:mm format.
func parseTimeOnly(s string) (time.Time, error) {
	return time.Parse("15:04", s)
}

// parseDatetimeFull parses a datetime string, trying YYYY-MM-DDThh:mm first, then YYYY-MM-DD.
func parseDatetimeFull(s string) (time.Time, error) {
	t, err := time.Parse("2006-01-02T15:04", s)
	if err == nil {
		return t, nil
	}
	return time.Parse("2006-01-02", s)
}
