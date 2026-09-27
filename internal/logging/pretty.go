package logging

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	ansiReset  = "\x1b[0m"
	ansiDim    = "\x1b[2m"
	ansiGray   = "\x1b[90m"
	ansiGreen  = "\x1b[32m"
	ansiYellow = "\x1b[33m"
	ansiRed    = "\x1b[31m"
)

type Options struct {
	Level slog.Leveler
	Color bool
}

type Handler struct {
	writer io.Writer

	level slog.Leveler
	color bool

	mutex *sync.Mutex

	attributes []slog.Attr
	groups     []string
}

type field struct {
	key   string
	value string
}

func NewPrettyHandler(
	writer io.Writer,
	options *Options,
) *Handler {
	level := slog.Leveler(
		slog.LevelInfo,
	)

	color := false

	if options != nil {
		if options.Level != nil {
			level = options.Level
		}

		color = options.Color
	}

	return &Handler{
		writer: writer,

		level: level,
		color: color,

		mutex: &sync.Mutex{},
	}
}

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

	info, err :=
		file.Stat()

	if err != nil {
		return false
	}

	return info.Mode()&
		os.ModeCharDevice != 0
}

func (h *Handler) Enabled(
	_ context.Context,
	level slog.Level,
) bool {
	return level >=
		h.level.Level()
}

func (h *Handler) Handle(
	_ context.Context,
	record slog.Record,
) error {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	timestamp :=
		record.Time

	if timestamp.IsZero() {
		timestamp =
			time.Now()
	}

	levelLabel :=
		formatLevel(
			record.Level,
		)

	levelColor :=
		h.levelColor(
			record.Level,
		)

	if h.color {
		_, _ = fmt.Fprintf(
			h.writer,
			"%s%s%s  %s%-5s%s  %s\n",
			ansiDim,
			timestamp.Format(
				"15:04:05.000",
			),
			ansiReset,
			levelColor,
			levelLabel,
			ansiReset,
			record.Message,
		)
	} else {
		_, _ = fmt.Fprintf(
			h.writer,
			"%s  %-5s  %s\n",
			timestamp.Format(
				"15:04:05.000",
			),
			levelLabel,
			record.Message,
		)
	}

	fields :=
		make(
			[]field,
			0,
			len(h.attributes)+
				record.NumAttrs(),
		)

	for _, attribute := range h.attributes {
		fields =
			appendAttribute(
				fields,
				h.groups,
				attribute,
			)
	}

	record.Attrs(
		func(attribute slog.Attr) bool {
			fields =
				appendAttribute(
					fields,
					h.groups,
					attribute,
				)

			return true
		},
	)

	if len(fields) > 0 {
		h.writeFields(
			fields,
		)
	}

	_, _ = fmt.Fprintln(
		h.writer,
	)

	return nil
}

func (h *Handler) WithAttrs(
	attributes []slog.Attr,
) slog.Handler {
	clone :=
		*h

	clone.attributes =
		append(
			append(
				[]slog.Attr(nil),
				h.attributes...,
			),
			attributes...,
		)

	clone.groups =
		append(
			[]string(nil),
			h.groups...,
		)

	return &clone
}

func (h *Handler) WithGroup(
	name string,
) slog.Handler {
	if name == "" {
		return h
	}

	clone :=
		*h

	clone.attributes =
		append(
			[]slog.Attr(nil),
			h.attributes...,
		)

	clone.groups =
		append(
			append(
				[]string(nil),
				h.groups...,
			),
			name,
		)

	return &clone
}

func (h *Handler) writeFields(
	fields []field,
) {
	maxKeyLength := 0

	for _, entry := range fields {
		if len(entry.key) >
			maxKeyLength {
			maxKeyLength =
				len(entry.key)
		}
	}

	for _, entry := range fields {
		value :=
			strings.ReplaceAll(
				entry.value,
				"\r\n",
				"\n",
			)

		value =
			strings.ReplaceAll(
				value,
				"\r",
				"\n",
			)

		lines :=
			strings.Split(
				value,
				"\n",
			)

		if len(lines) <= 1 {
			h.writeSingleLineField(
				entry.key,
				value,
				maxKeyLength,
			)

			continue
		}

		h.writeMultiLineField(
			entry.key,
			lines,
		)
	}
}

func (h *Handler) writeSingleLineField(
	key string,
	value string,
	width int,
) {
	if h.color {
		_, _ = fmt.Fprintf(
			h.writer,
			"    %s%-*s%s  %s\n",
			ansiDim,
			width,
			key,
			ansiReset,
			value,
		)

		return
	}

	_, _ = fmt.Fprintf(
		h.writer,
		"    %-*s  %s\n",
		width,
		key,
		value,
	)
}

func (h *Handler) writeMultiLineField(
	key string,
	lines []string,
) {
	if h.color {
		_, _ = fmt.Fprintf(
			h.writer,
			"    %s%s%s\n",
			ansiDim,
			key,
			ansiReset,
		)
	} else {
		_, _ = fmt.Fprintf(
			h.writer,
			"    %s\n",
			key,
		)
	}

	for _, line := range lines {
		if strings.TrimSpace(
			line,
		) == "" {
			continue
		}

		_, _ = fmt.Fprintf(
			h.writer,
			"      %s\n",
			line,
		)
	}
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

func appendAttribute(
	fields []field,
	groups []string,
	attribute slog.Attr,
) []field {
	attribute.Value =
		attribute.Value.Resolve()

	if attribute.Equal(
		slog.Attr{},
	) {
		return fields
	}

	if attribute.Value.Kind() ==
		slog.KindGroup {
		groupName :=
			attribute.Key

		nextGroups :=
			append(
				[]string(nil),
				groups...,
			)

		if groupName != "" {
			nextGroups =
				append(
					nextGroups,
					groupName,
				)
		}

		for _, child := range attribute.Value.Group() {
			fields =
				appendAttribute(
					fields,
					nextGroups,
					child,
				)
		}

		return fields
	}

	keyParts :=
		append(
			[]string(nil),
			groups...,
		)

	if attribute.Key != "" {
		keyParts =
			append(
				keyParts,
				attribute.Key,
			)
	}

	key :=
		strings.Join(
			keyParts,
			".",
		)

	if key == "" {
		key = "value"
	}

	return append(
		fields,
		field{
			key: key,

			value: formatValue(
				attribute.Value,
			),
		},
	)
}

func formatValue(
	value slog.Value,
) string {
	value =
		value.Resolve()

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
		if err, ok :=
			value.Any().(error); ok {
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
