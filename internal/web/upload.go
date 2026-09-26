package web

import (
	"encoding/json"
	"net/http"
	"os"
)

func (s *Server) handleUpload(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(MAX_FILE_SIZE); err != nil {
		http.Error(w, "invalid multipart form", http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "missing file", http.StatusBadGateway)
		return
	}
	defer file.Close()

	tempDir, err := os.MkdirTemp("", "media-converter-*")
	if err != nil {
		http.Error(w, "failed to create temp dir", http.StatusInternalServerError)
		return
	}
	defer os.RemoveAll(tempDir)

	inputPath, err := saveUpload(file, header.Filename, tempDir)
	if err != nil {
		http.Error(w, "failed to save upload", http.StatusInternalServerError)
		return
	}

	format, err := detectFormat(inputPath)
	if err != nil {
		http.Error(w, "unsupported media type", http.StatusUnsupportedMediaType)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(format); err != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
	}
}
