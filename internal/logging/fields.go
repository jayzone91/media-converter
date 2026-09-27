package logging

import (
	"fmt"
	"log/slog"
	"strconv"
	"strings"
)

type field struct {
	key   string
	value string
}

func (h *Handler) writeInlineFields(
	fields []field,
) {
	for _, entry := range fields {
		value := compactFieldValue(
			entry.value,
		)

		if h.color {
			_, _ = fmt.Fprintf(
				h.writer,
				"  %s%s%s=%s",
				ansiDim,
				entry.key,
				ansiReset,
				value,
			)

			continue
		}

		_, _ = fmt.Fprintf(
			h.writer,
			"  %s=%s",
			entry.key,
			value,
		)
	}
}

func compactFieldValue(
	value string,
) string {
	value = normalizeFieldValue(
		value,
	)

	if strings.ContainsAny(
		value,
		" \t\n",
	) {
		return strconv.Quote(
			value,
		)
	}

	return value
}

func (h *Handler) writeFields(
	fields []field,
) {
	maxKeyLength := 0

	for _, entry := range fields {
		if len(entry.key) > maxKeyLength {
			maxKeyLength = len(
				entry.key,
			)
		}
	}

	for _, entry := range fields {
		value := normalizeFieldValue(
			entry.value,
		)

		lines := strings.Split(
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

func appendAttribute(
	fields []field,
	groups []string,
	attribute slog.Attr,
) []field {
	attribute.Value = attribute.Value.Resolve()

	if attribute.Equal(
		slog.Attr{},
	) {
		return fields
	}

	if attribute.Value.Kind() == slog.KindGroup {
		nextGroups := append(
			[]string(nil),
			groups...,
		)

		if attribute.Key != "" {
			nextGroups = append(
				nextGroups,
				attribute.Key,
			)
		}

		for _, child := range attribute.Value.Group() {
			fields = appendAttribute(
				fields,
				nextGroups,
				child,
			)
		}

		return fields
	}

	keyParts := append(
		[]string(nil),
		groups...,
	)

	if attribute.Key != "" {
		keyParts = append(
			keyParts,
			attribute.Key,
		)
	}

	key := strings.Join(
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

func normalizeFieldValue(
	value string,
) string {
	value = strings.ReplaceAll(
		value,
		"\r\n",
		"\n",
	)

	return strings.ReplaceAll(
		value,
		"\r",
		"\n",
	)
}
