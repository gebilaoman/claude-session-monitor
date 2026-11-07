package ui

import (
	"fmt"
	"time"
)

// HumanBytes 人类可读字节: 1.0GB / 938MB / 12KB / 96B。
func HumanBytes(b int64) string {
	switch {
	case b >= 1<<30:
		return fmt.Sprintf("%.1fGB", float64(b)/(1<<30))
	case b >= 1<<20:
		return fmt.Sprintf("%dMB", b/(1<<20))
	case b >= 1<<10:
		return fmt.Sprintf("%dKB", b/(1<<10))
	default:
		return fmt.Sprintf("%dB", b)
	}
}

// HumanSince 自 t 至今的时长: 23s / 6h34m / 2d6h。
func HumanSince(t time.Time) string {
	if t.IsZero() {
		return "-"
	}
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return fmt.Sprintf("%ds", int(d.Seconds()))
	case d < time.Hour:
		return fmt.Sprintf("%dm", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh%dm", int(d.Hours()), int(d.Minutes())%60)
	default:
		return fmt.Sprintf("%dd%dh", int(d.Hours())/24, int(d.Hours())%24)
	}
}

// Fit 截断过长显示名(末尾 …),保持列对齐。
func Fit(s string, max int) string {
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	return string(r[:max-1]) + "…"
}

// MB 字节数转 MB。
func MB(b int64) int { return int(b / 1024 / 1024) }
