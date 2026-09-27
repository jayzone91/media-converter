package web

import (
	"embed"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"

	"github.com/jayzone91/media-converter/internal/media"
	"github.com/jayzone91/media-converter/internal/web/view"
)

//go:embed static/*
var uiFiles embed.FS

func (s *Server) handleIndex(
	w http.ResponseWriter,
	r *http.Request,
) {
	w.Header().Set(
		"Content-Type",
		"text/html; charset=utf-8",
	)

	if err := view.IndexPage().Render(
		r.Context(),
		w,
	); err != nil {
		http.Error(
			w,
			"failed to render page",
			http.StatusInternalServerError,
		)
	}
}

func (s *Server) handleDetect(
	w http.ResponseWriter,
	r *http.Request,
) {
	if !parseMultipartForm(w, r) {
		return
	}

	previousUploadID := r.FormValue(
		"upload_id",
	)

	headers := r.MultipartForm.File["file"]

	if len(headers) == 0 {
		renderDetectError(
			w,
			r,
			"Keine Datei ausgewählt.",
		)

		return
	}

	if len(headers) > maxBatchFiles {
		renderDetectError(
			w,
			r,
			fmt.Sprintf(
				"Maximal %d Dateien gleichzeitig.",
				maxBatchFiles,
			),
		)

		return
	}

	tempDir, err := os.MkdirTemp(
		"",
		"media-converter-upload-*",
	)
	if err != nil {
		renderDetectError(
			w,
			r,
			"Temporäres Verzeichnis konnte nicht erstellt werden.",
		)

		return
	}

	cleanup := true

	defer func() {
		if cleanup {
			_ = os.RemoveAll(tempDir)
		}
	}()

	files := make(
		[]storedUploadFile,
		0,
		len(headers),
	)

	filenames := make(
		[]string,
		0,
		len(headers),
	)

	var detectedFormat media.Format

	for index, header := range headers {
		if !validateFileSize(header) {
			w.WriteHeader(
				http.StatusRequestEntityTooLarge,
			)

			renderDetectError(
				w,
				r,
				"Eine Datei ist größer als 512 MiB.",
			)

			return
		}

		file, err := header.Open()
		if err != nil {
			renderDetectError(
				w,
				r,
				"Datei konnte nicht geöffnet werden.",
			)

			return
		}

		fileDir := filepath.Join(
			tempDir,
			fmt.Sprintf(
				"%03d",
				index+1,
			),
		)

		if err := os.MkdirAll(
			fileDir,
			0o700,
		); err != nil {
			file.Close()

			renderDetectError(
				w,
				r,
				"Temporäres Verzeichnis konnte nicht erstellt werden.",
			)

			return
		}

		inputPath, saveErr := saveUpload(
			file,
			header.Filename,
			fileDir,
		)

		closeErr := file.Close()

		if saveErr != nil || closeErr != nil {
			renderDetectError(
				w,
				r,
				"Datei konnte nicht gespeichert werden.",
			)

			return
		}

		format, err := detectFormat(
			r.Context(),
			inputPath,
			s.ffprobe,
		)
		if err != nil {
			renderDetectError(
				w,
				r,
				fmt.Sprintf(
					"Dateiformat von %q wird nicht unterstützt.",
					header.Filename,
				),
			)

			return
		}

		if index == 0 {
			detectedFormat = format
		} else if format.ID != detectedFormat.ID {
			renderDetectError(
				w,
				r,
				"Alle Dateien müssen denselben Dateityp haben.",
			)

			return
		}

		files = append(
			files,
			storedUploadFile{
				Path:     inputPath,
				Filename: header.Filename,
			},
		)

		filenames = append(
			filenames,
			header.Filename,
		)
	}

	upload, err := s.uploads.Add(
		tempDir,
		files,
		detectedFormat,
	)
	if err != nil {
		renderDetectError(
			w,
			r,
			"Upload konnte nicht gespeichert werden.",
		)

		return
	}

	cleanup = false

	s.uploads.Delete(
		previousUploadID,
	)

	w.Header().Set(
		"Content-Type",
		"text/html; charset=utf-8",
	)

	if err := view.DetectResult(
		upload.ID,
		filenames,
		detectedFormat,
	).Render(
		r.Context(),
		w,
	); err != nil {
		s.uploads.Delete(
			upload.ID,
		)

		http.Error(
			w,
			"failed to render detection result",
			http.StatusInternalServerError,
		)
	}
}

func renderDetectError(
	w http.ResponseWriter,
	r *http.Request,
	message string,
) {
	w.Header().Set(
		"Content-Type",
		"text/html; charset=utf-8",
	)

	_ = view.DetectError(
		message,
	).Render(
		r.Context(),
		w,
	)
}

func staticHandler() http.Handler {
	staticFS, err := fs.Sub(
		uiFiles,
		"static",
	)

	if err != nil {
		panic(err)
	}

	return http.FileServer(
		http.FS(staticFS),
	)
}
