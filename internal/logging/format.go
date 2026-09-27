package logging

import (
	"fmt"
	"log/slog"
	"strconv"
	"time"
)

func formatValue(
	value slog.Value,
) string {
	value = value.Resolve()

	switch value.Kind() {
	case slog.KindString:
		return value.String()

	case slog.KindBool:
		return strconv.FormatBool(
			value.Bool(),
		)

	case slog.KindInt64:
		return strconv.FormatInt(
			value.Int64(),
			10,
		)

	case slog.KindUint64:
		return strconv.FormatUint(
			value.Uint64(),
			10,
		)

	case slog.KindFloat64:
		return strconv.FormatFloat(
			value.Float64(),
			'f',
			-1,
			64,
		)

	case slog.KindDuration:
		return value.Duration().
			String()

	case slog.KindTime:
		return value.Time().
			Format(
				time.RFC3339Nano,
			)

	case slog.KindAny:
		if err, ok := value.Any().(error); ok {
			return err.Error()
		}

		return fmt.Sprint(
			value.Any(),
		)

	default:
		return value.String()
	}
}

func formatLevel(
	level slog.Level,
) string {
	switch {
	case level >= slog.LevelError:
		return "ERROR"

	case level >= slog.LevelWarn:
		return "WARN"

	case level >= slog.LevelInfo:
		return "INFO"

	case level >= slog.LevelDebug:
		return "DEBUG"

	default:
		return level.String()
	}
}
