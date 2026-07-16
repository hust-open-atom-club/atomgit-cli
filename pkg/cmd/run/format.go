package run

import (
	"fmt"
	"strings"
	"unicode"

	"atomgit.com/hust-open-atom-club/atomgit-cli/internal/api/actions"
)

func fallback(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func singleLine(value string) string {
	value = strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return ' '
		}
		return r
	}, value)
	return strings.Join(strings.Fields(value), " ")
}

func formatTimestamp(value actions.Timestamp) string {
	timestamp := value.Time()
	if timestamp.IsZero() {
		return "-"
	}
	return timestamp.Local().Format("2006-01-02 15:04:05 MST")
}

func formatBytes(size int64) string {
	if size < 0 {
		return "-"
	}
	const unit = 1024
	if size < unit {
		return fmt.Sprintf("%d B", size)
	}
	divisor := int64(unit)
	exponent := 0
	for quotient := size / unit; quotient >= unit && exponent < len("KMGTPE")-1; quotient /= unit {
		divisor *= unit
		exponent++
	}
	return fmt.Sprintf("%.1f %ciB", float64(size)/float64(divisor), "KMGTPE"[exponent])
}
