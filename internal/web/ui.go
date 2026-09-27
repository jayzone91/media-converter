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
		s.logError(
			r,
			"index render failed",
			err,
		)

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
		s.logWarn(
			r,
			"detect rejected",
			"reason",
			"invalid multipart form",
		)

		return
	}

	previousUploadID := r.FormValue(
		"upload_id",
	)

	headers := r.MultipartForm.File["file"]

	if len(headers) == 0 {
		s.logWarn(
			r,
			"detect rejected",
			"reason",
			"no files",
		)

		w.WriteHeader(
			http.StatusBadRequest,
		)

		renderDetectError(
			w,
			r,
			"Keine Datei ausgewählt.",
		)

		return
	}

	if len(headers) > maxBatchFiles {
		s.logWarn(
			r,
			"detect rejected",
			"reason",
			"too many files",
			"files",
			len(headers),
			"limit",
			maxBatchFiles,
		)

		w.WriteHeader(
			http.StatusBadRequest,
		)

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
		s.logError(
			r,
			"detect temp directory failed",
			err,
		)

		w.WriteHeader(
			http.StatusInternalServerError,
		)

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
			s.logWarn(
				r,
				"detect rejected",
				"reason",
				"file too large",
				"file",
				header.Filename,
				"size",
				header.Size,
			)

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
			s.logError(
				r,
				"detect file open failed",
				err,
				"file",
				header.Filename,
			)

			w.WriteHeader(
				http.StatusInternalServerError,
			)

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
			_ = file.Close()

			s.logError(
				r,
				"detect file directory failed",
				err,
				"file",
				header.Filename,
			)

			w.WriteHeader(
				http.StatusInternalServerError,
			)

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

		if saveErr != nil {
			s.logError(
				r,
				"detect upload save failed",
				saveErr,
				"file",
				header.Filename,
			)

			w.WriteHeader(
				http.StatusInternalServerError,
			)

			renderDetectError(
				w,
				r,
				"Datei konnte nicht gespeichert werden.",
			)

			return
		}

		if closeErr != nil {
			s.logError(
				r,
				"detect upload close failed",
				closeErr,
				"file",
				header.Filename,
			)

			w.WriteHeader(
				http.StatusInternalServerError,
			)

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
			s.logWarn(
				r,
				"detect unsupported",
				"file",
				header.Filename,
				"error",
				err,
			)

			w.WriteHeader(
				http.StatusUnsupportedMediaType,
			)

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
			s.logWarn(
				r,
				"detect mixed formats",
				"expected",
				detectedFormat.ID,
				"actual",
				format.ID,
				"file",
				header.Filename,
			)

			w.WriteHeader(
				http.StatusBadRequest,
			)

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
		s.logError(
			r,
			"detect upload store failed",
			err,
			"files",
			len(files),
			"format",
			detectedFormat.ID,
		)

		w.WriteHeader(
			http.StatusInternalServerError,
		)

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

		s.logError(
			r,
			"detect result render failed",
			err,
			"format",
			detectedFormat.ID,
			"files",
			len(files),
		)

		http.Error(
			w,
			"failed to render detection result",
			http.StatusInternalServerError,
		)

		return
	}

	s.logInfo(
		"detect",
		"format",
		detectedFormat.ID,
		"files",
		len(files),
	)
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
