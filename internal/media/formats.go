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
	ID       string   `json:"format"`
	MIME     string   `json:"mime"`
	Category Category `json:"category"`
	Targets  []string `json:"targets"`
}

var Formats = map[string]Format{
	"jpeg": {
		ID:       "jpeg",
		MIME:     "image/jpeg",
		Category: CategoryImage,
		Targets:  []string{"png", "webp", "avif", "tiff", "pdf"},
	},
	"png": {
		ID:       "png",
		MIME:     "image/png",
		Category: CategoryImage,
		Targets:  []string{"jpeg", "webp", "avif", "tiff", "pdf"},
	},
	"webp": {
		ID:       "webp",
		MIME:     "image/webp",
		Category: CategoryImage,
		Targets:  []string{"jpeg", "png", "avif", "tiff", "pdf"},
	},
	"mp3": {
		ID:       "mp3",
		MIME:     "audio/mpeg",
		Category: CategoryAudio,
		Targets:  []string{"wav", "flac", "ogg", "opus"},
	},
	"wav": {
		ID:       "wav",
		MIME:     "audio/wav",
		Category: CategoryAudio,
		Targets:  []string{"mp3", "flac", "ogg", "opus"},
	},
	"flac": {
		ID:       "flac",
		MIME:     "audio/flac",
		Category: CategoryAudio,
		Targets:  []string{"mp3", "wav", "ogg", "opus"},
	},
	"ogg": {
		ID:       "ogg",
		MIME:     "audio/ogg",
		Category: CategoryAudio,
		Targets:  []string{"mp3", "wav", "flac", "opus"},
	},
	"mp4": {
		ID:       "mp4",
		MIME:     "video/mp4",
		Category: CategoryVideo,
		Targets:  []string{"mkv", "webm", "mov", "mp3", "wav", "flac", "gif"},
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
