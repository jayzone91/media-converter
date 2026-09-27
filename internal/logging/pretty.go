package logging

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"sync"
	"time"
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
		level:  level,
		color:  color,
		mutex:  &sync.Mutex{},
	}
}

func (h *Handler) Enabled(
	_ context.Context,
	level slog.Level,
) bool {
	return level >= h.level.Level()
}

func (h *Handler) Handle(
	_ context.Context,
	record slog.Record,
) error {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	timestamp := record.Time
	if timestamp.IsZero() {
		timestamp = time.Now()
	}

	fields := make(
		[]field,
		0,
		len(h.attributes)+record.NumAttrs(),
	)

	for _, attribute := range h.attributes {
		fields = appendAttribute(
			fields,
			h.groups,
			attribute,
		)
	}

	record.Attrs(
		func(attribute slog.Attr) bool {
			fields = appendAttribute(
				fields,
				h.groups,
				attribute,
			)

			return true
		},
	)

	if record.Level < slog.LevelWarn {
		h.writeCompactRecord(
			timestamp,
			record.Level,
			record.Message,
			fields,
		)

		return nil
	}

	h.writeHeader(
		timestamp,
		record.Level,
		record.Message,
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
	clone := *h

	clone.attributes = append(
		append(
			[]slog.Attr(nil),
			h.attributes...,
		),
		attributes...,
	)

	clone.groups = append(
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

	clone := *h

	clone.attributes = append(
		[]slog.Attr(nil),
		h.attributes...,
	)

	clone.groups = append(
		append(
			[]string(nil),
			h.groups...,
		),
		name,
	)

	return &clone
}

func (h *Handler) writeCompactRecord(
	timestamp time.Time,
	level slog.Level,
	message string,
	fields []field,
) {
	levelLabel := formatLevel(
		level,
	)

	if h.color {
		_, _ = fmt.Fprintf(
			h.writer,
			"%s%s%s  %s%-5s%s  %s",
			ansiDim,
			timestamp.Format(
				"15:04:05.000",
			),
			ansiReset,
			h.levelColor(level),
			levelLabel,
			ansiReset,
			message,
		)
	} else {
		_, _ = fmt.Fprintf(
			h.writer,
			"%s  %-5s  %s",
			timestamp.Format(
				"15:04:05.000",
			),
			levelLabel,
			message,
		)
	}

	if len(fields) > 0 {
		h.writeInlineFields(
			fields,
		)
	}

	_, _ = fmt.Fprintln(
		h.writer,
	)
}

func (h *Handler) writeHeader(
	timestamp time.Time,
	level slog.Level,
	message string,
) {
	levelLabel := formatLevel(
		level,
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
			h.levelColor(level),
			levelLabel,
			ansiReset,
			message,
		)

		return
	}

	_, _ = fmt.Fprintf(
		h.writer,
		"%s  %-5s  %s\n",
		timestamp.Format(
			"15:04:05.000",
		),
		levelLabel,
		message,
	)
}
