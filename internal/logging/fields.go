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

const maxCompactFields = 6

var compactFieldAliases = map[string]string{
	"input_size_bytes":      "input",
	"original_size_bytes":   "input",
	"before_size_bytes":     "input",
	"output_size_bytes":     "output",
	"compressed_size_bytes": "output",
	"optimized_size_bytes":  "output",
	"after_size_bytes":      "output",
	"size_bytes":            "size",
	"page_count":            "pages",
	"original_page_count":   "pages",
	"deleted_page_count":    "deleted",
	"result_page_count":     "remaining",
	"requested_pages":       "selected",
	"delete_count":          "selected",
	"selected_page_count":   "selected",
	"extracted_page_count":  "extracted",
	"rotated_page_count":    "rotated",
	"file_count":            "files",
	"files_count":           "files",
}

var compactIgnoredFields = map[string]struct{}{
	"method":     {},
	"path":       {},
	"upload_id":  {},
	"filename":   {},
	"client_ip":  {},
	"user_agent": {},
	"bytes":      {},
}

func compactRecordFields(
	message string,
	fields []field,
) []field {
	if message == "http" {
		return selectCompactFields(
			fields,
			[]string{
				"method",
				"path",
				"status",
				"duration",
			},
		)
	}

	capacity := len(fields)
	if capacity > maxCompactFields {
		capacity = maxCompactFields
	}

	result := make(
		[]field,
		0,
		capacity,
	)

	seen := make(
		map[string]struct{},
		maxCompactFields,
	)

	for _, entry := range fields {
		if _, ignored := compactIgnoredFields[entry.key]; ignored {
			continue
		}

		key := entry.key

		if alias, ok := compactFieldAliases[key]; ok {
			key = alias
		}

		if _, exists := seen[key]; exists {
			continue
		}

		seen[key] = struct{}{}

		result = append(
			result,
			field{
				key:   key,
				value: entry.value,
			},
		)

		if len(result) >= maxCompactFields {
			break
		}
	}

	return result
}

func selectCompactFields(
	fields []field,
	keys []string,
) []field {
	result := make(
		[]field,
		0,
		len(keys),
	)

	for _, key := range keys {
		for _, entry := range fields {
			if entry.key != key {
				continue
			}

			result = append(
				result,
				entry,
			)

			break
		}
	}

	return result
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
