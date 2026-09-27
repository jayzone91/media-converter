package media

import (
	"mime"
	"path/filepath"
	"strings"
)

func FindByExtension(
	path string,
) (Format, bool) {
	extension := strings.ToLower(
		filepath.Ext(path),
	)

	if extension == "" {
		return Format{}, false
	}

	for _, format := range Formats {
		for _, candidate := range format.Extensions {
			if candidate == extension {
				return format, true
			}
		}
	}

	return Format{}, false
}

func FormatMatchesExtension(
	path string,
	format Format,
) bool {
	expected, ok := FindByExtension(path)
	if !ok {
		return false
	}

	return expected.ID == format.ID
}

func FormatAcceptsMIME(
	format Format,
	contentType string,
) bool {
	mediaType := normalizeMIME(
		contentType,
	)

	switch format.ID {
	case "markdown":
		return mediaType == "text/plain" ||
			mediaType == "text/markdown"

	case "svg":
		return mediaType == "image/svg+xml" ||
			mediaType == "text/xml" ||
			mediaType == "application/xml"

	default:
		return mediaType == format.MIME
	}
}

func normalizeMIME(
	contentType string,
) string {
	mediaType, _, err := mime.ParseMediaType(
		contentType,
	)

	if err != nil {
		return strings.ToLower(
			strings.TrimSpace(contentType),
		)
	}

	return strings.ToLower(
		mediaType,
	)
}
