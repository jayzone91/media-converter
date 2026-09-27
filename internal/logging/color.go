package logging

import (
	"log/slog"
	"os"
	"strings"
)

const (
	ansiReset  = "\x1b[0m"
	ansiDim    = "\x1b[2m"
	ansiGray   = "\x1b[90m"
	ansiGreen  = "\x1b[32m"
	ansiYellow = "\x1b[33m"
	ansiRed    = "\x1b[31m"
)

func SupportsColor(
	file *os.File,
) bool {
	if file == nil {
		return false
	}

	if os.Getenv(
		"NO_COLOR",
	) != "" {
		return false
	}

	if strings.EqualFold(
		os.Getenv("TERM"),
		"dumb",
	) {
		return false
	}

	info, err := file.Stat()
	if err != nil {
		return false
	}

	return info.Mode()&
		os.ModeCharDevice != 0
}

func (h *Handler) levelColor(
	level slog.Level,
) string {
	if !h.color {
		return ""
	}

	switch {
	case level >= slog.LevelError:
		return ansiRed

	case level >= slog.LevelWarn:
		return ansiYellow

	case level >= slog.LevelInfo:
		return ansiGreen

	default:
		return ansiGray
	}
}
