package media

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
	ProbeNames []string `json:"-"`
}

var Formats = map[string]Format{
	"jpeg": {
		ID:         "jpeg",
		MIME:       "image/jpeg",
		Category:   CategoryImage,
		Targets:    []string{"png", "webp", "avif", "tiff", "bmp", "pdf"},
		ProbeNames: []string{"image2"},
	},
	"png": {
		ID:         "png",
		MIME:       "image/png",
		Category:   CategoryImage,
		Targets:    []string{"jpeg", "webp", "avif", "tiff", "bmp", "pdf"},
		ProbeNames: []string{"png_pipe"},
	},
	"webp": {
		ID:         "webp",
		MIME:       "image/webp",
		Category:   CategoryImage,
		Targets:    []string{"jpeg", "png", "avif", "tiff", "bmp", "pdf"},
		ProbeNames: []string{"webp_pipe"},
	},
	"avif": {
		ID:         "avif",
		MIME:       "image/avif",
		Category:   CategoryImage,
		Targets:    []string{"jpeg", "png", "webp", "tiff", "bmp", "pdf"},
		ProbeNames: []string{"avif"},
	},
	"tiff": {
		ID:         "tiff",
		MIME:       "image/tiff",
		Category:   CategoryImage,
		Targets:    []string{"jpeg", "png", "webp", "avif", "bmp", "pdf"},
		ProbeNames: []string{"tiff_pipe"},
	},
	"bmp": {
		ID:         "bmp",
		MIME:       "image/bmp",
		Category:   CategoryImage,
		Targets:    []string{"jpeg", "png", "webp", "avif", "tiff", "pdf"},
		ProbeNames: []string{"bmp_pipe"},
	},
	"gif": {
		ID:         "gif",
		MIME:       "image/gif",
		Category:   CategoryImage,
		Targets:    []string{"png", "webp", "mp4", "webm"},
		ProbeNames: []string{"gif"},
	},

	"mp3": {
		ID:         "mp3",
		MIME:       "audio/mpeg",
		Category:   CategoryAudio,
		Targets:    []string{"wav", "flac", "ogg", "opus", "aac", "m4a"},
		ProbeNames: []string{"mp3"},
	},
	"wav": {
		ID:         "wav",
		MIME:       "audio/wav",
		Category:   CategoryAudio,
		Targets:    []string{"mp3", "flac", "ogg", "opus", "aac", "m4a"},
		ProbeNames: []string{"wav"},
	},
	"flac": {
		ID:         "flac",
		MIME:       "audio/flac",
		Category:   CategoryAudio,
		Targets:    []string{"mp3", "wav", "ogg", "opus", "aac", "m4a"},
		ProbeNames: []string{"flac"},
	},
	"ogg": {
		ID:         "ogg",
		MIME:       "audio/ogg",
		Category:   CategoryAudio,
		Targets:    []string{"mp3", "wav", "flac", "opus", "aac", "m4a"},
		ProbeNames: []string{"ogg"},
	},
	"opus": {
		ID:         "opus",
		MIME:       "audio/ogg",
		Category:   CategoryAudio,
		Targets:    []string{"mp3", "wav", "flac", "ogg", "aac", "m4a"},
		ProbeNames: []string{"ogg"},
	},
	"aac": {
		ID:         "aac",
		MIME:       "audio/aac",
		Category:   CategoryAudio,
		Targets:    []string{"mp3", "wav", "flac", "ogg", "opus", "m4a"},
		ProbeNames: []string{"aac"},
	},
	"m4a": {
		ID:         "m4a",
		MIME:       "audio/mp4",
		Category:   CategoryAudio,
		Targets:    []string{"mp3", "wav", "flac", "ogg", "opus", "aac"},
		ProbeNames: []string{"mov", "mp4", "m4a", "3gp", "3g2", "mj2"},
	},

	"mp4": {
		ID:         "mp4",
		MIME:       "video/mp4",
		Category:   CategoryVideo,
		Targets:    []string{"mkv", "webm", "mov", "avi", "mp3", "wav", "flac", "ogg", "gif"},
		ProbeNames: []string{"mov", "mp4", "m4a", "3gp", "3g2", "mj2"},
	},
	"mkv": {
		ID:         "mkv",
		MIME:       "video/x-matroska",
		Category:   CategoryVideo,
		Targets:    []string{"mp4", "webm", "mov", "avi", "mp3", "wav", "flac", "ogg", "gif"},
		ProbeNames: []string{"matroska"},
	},
	"webm": {
		ID:         "webm",
		MIME:       "video/webm",
		Category:   CategoryVideo,
		Targets:    []string{"mp4", "mkv", "mov", "avi", "mp3", "wav", "flac", "ogg", "gif"},
		ProbeNames: []string{"webm"},
	},
	"mov": {
		ID:         "mov",
		MIME:       "video/quicktime",
		Category:   CategoryVideo,
		Targets:    []string{"mp4", "mkv", "webm", "avi", "mp3", "wav", "flac", "ogg", "gif"},
		ProbeNames: []string{"mov"},
	},
	"avi": {
		ID:         "avi",
		MIME:       "video/x-msvideo",
		Category:   CategoryVideo,
		Targets:    []string{"mp4", "mkv", "webm", "mov", "mp3", "wav", "flac", "ogg", "gif"},
		ProbeNames: []string{"avi"},
	},
	"mpeg": {
		ID:         "mpeg",
		MIME:       "video/mpeg",
		Category:   CategoryVideo,
		Targets:    []string{"mp4", "mkv", "webm", "mov", "avi", "mp3", "wav", "flac", "ogg", "gif"},
		ProbeNames: []string{"mpeg"},
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

func FindByProbeName(name string) (Format, bool) {
	for _, format := range Formats {
		for _, probeName := range format.ProbeNames {
			if probeName == name {
				return format, true
			}
		}
	}

	return Format{}, false
}
