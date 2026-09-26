package media

import (
	"path/filepath"
	"strings"
)

type Category string

const (
	CategoryImage    Category = "image"
	CategoryAudio    Category = "audio"
	CategoryVideo    Category = "video"
	CategoryDocument Category = "document"
	CategoryPDF      Category = "pdf"
)

type Format struct {
	ID         string   `json:"format"`
	MIME       string   `json:"mime"`
	Category   Category `json:"category"`
	Targets    []string `json:"targets"`
	Extensions []string `json:"-"`
}

var Formats = map[string]Format{
	"jpeg": {
		ID:         "jpeg",
		MIME:       "image/jpeg",
		Category:   CategoryImage,
		Targets:    []string{"png", "webp", "avif", "tiff", "bmp", "pdf"},
		Extensions: []string{".jpg", ".jpeg"},
	},
	"png": {
		ID:         "png",
		MIME:       "image/png",
		Category:   CategoryImage,
		Targets:    []string{"jpeg", "webp", "avif", "tiff", "bmp", "pdf"},
		Extensions: []string{".png"},
	},
	"webp": {
		ID:         "webp",
		MIME:       "image/webp",
		Category:   CategoryImage,
		Targets:    []string{"jpeg", "png", "avif", "tiff", "bmp", "pdf"},
		Extensions: []string{".webp"},
	},
	"avif": {
		ID:         "avif",
		MIME:       "image/avif",
		Category:   CategoryImage,
		Targets:    []string{"jpeg", "png", "webp", "tiff", "bmp", "pdf"},
		Extensions: []string{".avif"},
	},
	"heic": {
		ID:         "heic",
		MIME:       "image/heic",
		Category:   CategoryImage,
		Targets:    []string{"jpeg", "png", "webp", "avif", "tiff", "pdf"},
		Extensions: []string{".heic", ".heif"},
	},
	"tiff": {
		ID:         "tiff",
		MIME:       "image/tiff",
		Category:   CategoryImage,
		Targets:    []string{"jpeg", "png", "webp", "avif", "bmp", "pdf"},
		Extensions: []string{".tif", ".tiff"},
	},
	"bmp": {
		ID:         "bmp",
		MIME:       "image/bmp",
		Category:   CategoryImage,
		Targets:    []string{"jpeg", "png", "webp", "avif", "tiff", "pdf"},
		Extensions: []string{".bmp"},
	},
	"gif": {
		ID:         "gif",
		MIME:       "image/gif",
		Category:   CategoryImage,
		Targets:    []string{"png", "webp", "mp4", "webm"},
		Extensions: []string{".gif"},
	},
	"svg": {
		ID:         "svg",
		MIME:       "image/svg+xml",
		Category:   CategoryImage,
		Targets:    []string{"jpeg", "png", "webp", "avif", "tiff", "pdf"},
		Extensions: []string{".svg"},
	},

	"mp3": {
		ID:         "mp3",
		MIME:       "audio/mpeg",
		Category:   CategoryAudio,
		Targets:    []string{"wav", "flac", "ogg", "opus", "aac", "m4a"},
		Extensions: []string{".mp3"},
	},
	"wav": {
		ID:         "wav",
		MIME:       "audio/wav",
		Category:   CategoryAudio,
		Targets:    []string{"mp3", "flac", "ogg", "opus", "aac", "m4a"},
		Extensions: []string{".wav"},
	},
	"flac": {
		ID:         "flac",
		MIME:       "audio/flac",
		Category:   CategoryAudio,
		Targets:    []string{"mp3", "wav", "ogg", "opus", "aac", "m4a"},
		Extensions: []string{".flac"},
	},
	"ogg": {
		ID:         "ogg",
		MIME:       "audio/ogg",
		Category:   CategoryAudio,
		Targets:    []string{"mp3", "wav", "flac", "opus", "aac", "m4a"},
		Extensions: []string{".ogg"},
	},
	"opus": {
		ID:         "opus",
		MIME:       "audio/ogg",
		Category:   CategoryAudio,
		Targets:    []string{"mp3", "wav", "flac", "ogg", "aac", "m4a"},
		Extensions: []string{".opus"},
	},
	"aac": {
		ID:         "aac",
		MIME:       "audio/aac",
		Category:   CategoryAudio,
		Targets:    []string{"mp3", "wav", "flac", "ogg", "opus", "m4a"},
		Extensions: []string{".aac"},
	},
	"m4a": {
		ID:         "m4a",
		MIME:       "audio/mp4",
		Category:   CategoryAudio,
		Targets:    []string{"mp3", "wav", "flac", "ogg", "opus", "aac"},
		Extensions: []string{".m4a"},
	},
	"wma": {
		ID:         "wma",
		MIME:       "audio/x-ms-wma",
		Category:   CategoryAudio,
		Targets:    []string{"mp3", "wav", "flac", "ogg", "opus", "aac", "m4a"},
		Extensions: []string{".wma"},
	},

	"mp4": {
		ID:         "mp4",
		MIME:       "video/mp4",
		Category:   CategoryVideo,
		Targets:    []string{"mkv", "webm", "mov", "avi", "mp3", "wav", "flac", "ogg", "gif"},
		Extensions: []string{".mp4", ".m4v"},
	},
	"mkv": {
		ID:         "mkv",
		MIME:       "video/x-matroska",
		Category:   CategoryVideo,
		Targets:    []string{"mp4", "webm", "mov", "avi", "mp3", "wav", "flac", "ogg", "gif"},
		Extensions: []string{".mkv"},
	},
	"webm": {
		ID:         "webm",
		MIME:       "video/webm",
		Category:   CategoryVideo,
		Targets:    []string{"mp4", "mkv", "mov", "avi", "mp3", "wav", "flac", "ogg", "gif"},
		Extensions: []string{".webm"},
	},
	"mov": {
		ID:         "mov",
		MIME:       "video/quicktime",
		Category:   CategoryVideo,
		Targets:    []string{"mp4", "mkv", "webm", "avi", "mp3", "wav", "flac", "ogg", "gif"},
		Extensions: []string{".mov"},
	},
	"avi": {
		ID:         "avi",
		MIME:       "video/x-msvideo",
		Category:   CategoryVideo,
		Targets:    []string{"mp4", "mkv", "webm", "mov", "mp3", "wav", "flac", "ogg", "gif"},
		Extensions: []string{".avi"},
	},
	"mpeg": {
		ID:         "mpeg",
		MIME:       "video/mpeg",
		Category:   CategoryVideo,
		Targets:    []string{"mp4", "mkv", "webm", "mov", "avi", "mp3", "wav", "flac", "ogg", "gif"},
		Extensions: []string{".mpeg", ".mpg"},
	},
	"wmv": {
		ID:         "wmv",
		MIME:       "video/x-ms-wmv",
		Category:   CategoryVideo,
		Targets:    []string{"mp4", "mkv", "webm", "mov", "avi", "mp3", "wav", "flac", "ogg", "gif"},
		Extensions: []string{".wmv"},
	},
	"flv": {
		ID:         "flv",
		MIME:       "video/x-flv",
		Category:   CategoryVideo,
		Targets:    []string{"mp4", "mkv", "webm", "mov", "avi", "mp3", "wav", "flac", "ogg", "gif"},
		Extensions: []string{".flv"},
	},

	"docx": {
		ID:         "docx",
		MIME:       "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
		Category:   CategoryDocument,
		Targets:    []string{"pdf", "odt"},
		Extensions: []string{".docx"},
	},
	"odt": {
		ID:         "odt",
		MIME:       "application/vnd.oasis.opendocument.text",
		Category:   CategoryDocument,
		Targets:    []string{"pdf", "docx"},
		Extensions: []string{".odt"},
	},
	"xlsx": {
		ID:         "xlsx",
		MIME:       "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
		Category:   CategoryDocument,
		Targets:    []string{"pdf", "ods"},
		Extensions: []string{".xlsx"},
	},
	"ods": {
		ID:         "ods",
		MIME:       "application/vnd.oasis.opendocument.spreadsheet",
		Category:   CategoryDocument,
		Targets:    []string{"pdf", "xlsx"},
		Extensions: []string{".ods"},
	},
	"pptx": {
		ID:         "pptx",
		MIME:       "application/vnd.openxmlformats-officedocument.presentationml.presentation",
		Category:   CategoryDocument,
		Targets:    []string{"pdf", "odp"},
		Extensions: []string{".pptx"},
	},
	"odp": {
		ID:         "odp",
		MIME:       "application/vnd.oasis.opendocument.presentation",
		Category:   CategoryDocument,
		Targets:    []string{"pdf", "pptx"},
		Extensions: []string{".odp"},
	},
}

func FindByMIME(mime string) (Format, bool) {
	for _, format := range Formats {
		if format.MIME == mime {
			return format, true
		}
	}

	return Format{}, false
}

func FindImageByExtension(path string) (Format, bool) {
	extension := strings.ToLower(filepath.Ext(path))

	for _, format := range Formats {
		if format.Category != CategoryImage {
			continue
		}

		for _, candidate := range format.Extensions {
			if candidate == extension {
				return format, true
			}
		}
	}

	return Format{}, false
}
