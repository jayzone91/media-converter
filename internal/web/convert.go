package web

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/jayzone91/media-converter/internal/media"
)

func (s *Server) handleConvert(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(MAX_FILE_SIZE); err != nil {
		http.Error(w, "invalid multipart form", http.StatusBadRequest)
		return
	}

	target := strings.ToLower(r.FormValue("target"))
	if target == "" {
		http.Error(w, "missing target format", http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "missing file", http.StatusBadRequest)
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

	if !slices.Contains(format.Targets, target) {
		http.Error(w, "unsupported conversion", http.StatusBadRequest)
		return
	}

	outputPath := filepath.Join(
		tempDir,
		strings.TrimSuffix(
			filepath.Base(header.Filename),
			filepath.Ext(header.Filename),
		)+"."+target,
	)

	switch format.Category {
	case media.CategoryImage:
		err = s.imageMagick.Convert(r.Context(), inputPath, outputPath)

	case media.CategoryAudio:
		err = s.ffmpeg.Convert(r.Context(), inputPath, outputPath)

	default:
		http.Error(w, "converter not implemented for this media type", http.StatusNotImplemented)
		return
	}

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	outputFile, err := os.Open(outputPath)
	if err != nil {
		http.Error(w, "failed to open converted file", http.StatusInternalServerError)
		return
	}
	defer outputFile.Close()

	stat, err := outputFile.Stat()
	if err != nil {
		http.Error(w, "failed to read converted file", http.StatusInternalServerError)
		return
	}

	w.Header().Set(
		"Content-Disposition",
		fmt.Sprintf(`attachment; filename"%s"`, filepath.Base(outputPath)),
	)

	http.ServeContent(w, r, filepath.Base(outputPath), stat.ModTime(), outputFile)
}
