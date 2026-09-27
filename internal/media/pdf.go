package media

func init() {
	Formats["pdf"] = Format{
		ID:         "pdf",
		MIME:       "application/pdf",
		Category:   CategoryPDF,
		Targets:    []string{"docx", "png", "jpeg"},
		Extensions: []string{".pdf"},
	}
}
