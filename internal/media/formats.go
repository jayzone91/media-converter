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
		Targets:    []string{"png", "webp", "avif", "tiff", "pdf"},
		ProbeNames: []string{"image2"},
	},
	"png": {
		ID:         "png",
		MIME:       "image/png",
		Category:   CategoryImage,
		Targets:    []string{"jpeg", "webp", "avif", "tiff", "pdf"},
		ProbeNames: []string{"png_pipe"},
	},
	"webp": {
		ID:         "webp",
		MIME:       "image/webp",
		Category:   CategoryImage,
		Targets:    []string{"jpeg", "png", "avif", "tiff", "pdf"},
		ProbeNames: []string{"webp_pipe"},
	},
	"mp3": {
		ID:         "mp3",
		MIME:       "audio/mpeg",
		Category:   CategoryAudio,
		Targets:    []string{"wav", "flac", "ogg", "opus"},
		ProbeNames: []string{"mp3"},
	},
	"wav": {
		ID:         "wav",
		MIME:       "audio/wav",
		Category:   CategoryAudio,
		Targets:    []string{"mp3", "flac", "ogg", "opus"},
		ProbeNames: []string{"wav"},
	},
	"flac": {
		ID:         "flac",
		MIME:       "audio/flac",
		Category:   CategoryAudio,
		Targets:    []string{"mp3", "wav", "ogg", "opus"},
		ProbeNames: []string{"flac"},
	},
	"ogg": {
		ID:         "ogg",
		MIME:       "audio/ogg",
		Category:   CategoryAudio,
		Targets:    []string{"mp3", "wav", "flac", "opus"},
		ProbeNames: []string{"ogg"},
	},
	"opus": {
		ID:         "opus",
		MIME:       "audio/ogg",
		Category:   CategoryAudio,
		Targets:    []string{"mp3", "wav", "flac", "ogg"},
		ProbeNames: []string{"ogg"},
	},
	"mp4": {
		ID:         "mp4",
		MIME:       "video/mp4",
		Category:   CategoryVideo,
		Targets:    []string{"mkv", "webm", "mov", "mp3", "wav", "flac", "gif"},
		ProbeNames: []string{"mp4"},
	},
	"mkv": {
		ID:         "mkv",
		MIME:       "video/x-matroska",
		Category:   CategoryVideo,
		Targets:    []string{"mp4", "webm", "mov", "mp3", "wav", "flac", "gif"},
		ProbeNames: []string{"matroska"},
	},
	"webm": {
		ID:         "webm",
		MIME:       "video/webm",
		Category:   CategoryVideo,
		Targets:    []string{"mp4", "mkv", "mov", "mp3", "wav", "flac", "gif"},
		ProbeNames: []string{"webm"},
	},
	"mov": {
		ID:         "mov",
		MIME:       "video/quicktime",
		Category:   CategoryVideo,
		Targets:    []string{"mp4", "mkv", "webm", "mp3", "wav", "flac", "gif"},
		ProbeNames: []string{"mov"},
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
