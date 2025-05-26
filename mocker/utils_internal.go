package mocker

import (
	"fmt"
	"strings"
)

// Helper function to calculate checksum for CPF and CNPJ
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

// Extracts raw regex string from /.../ and unescapes \/ → /
func extractRegex(value string) (string, error) {
	if !strings.HasPrefix(value, "/") || !strings.HasSuffix(value, "/") {
		return "", fmt.Errorf("Value '%s' must be wrapped in /.../", value)
	}
	trimmed := value[1 : len(value)-1]
	unescaped := strings.ReplaceAll(trimmed, `\/`, `/`)
	return unescaped, nil
}
