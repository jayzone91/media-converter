package web

import (
	"embed"
	"io/fs"
	"net/http"
	"os"

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

	previousUploadID :=
		r.FormValue(
			"upload_id",
		)

	file, header, err :=
		r.FormFile("file")

	if err != nil {
		renderDetectError(
			w,
			r,
			"Keine Datei ausgewählt.",
		)

		return
	}

	defer file.Close()

	if !validateFileSize(header) {
		w.WriteHeader(
			http.StatusRequestEntityTooLarge,
		)

		renderDetectError(
			w,
			r,
			"Die Datei ist größer als 512 MiB.",
		)

		return
	}

	tempDir, err :=
		os.MkdirTemp(
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

	inputPath, err :=
		saveUpload(
			file,
			header.Filename,
			tempDir,
		)

	if err != nil {
		_ = os.RemoveAll(
			tempDir,
		)

		renderDetectError(
			w,
			r,
			"Datei konnte nicht gespeichert werden.",
		)

		return
	}

	format, err :=
		detectFormat(
			r.Context(),
			inputPath,
			s.ffprobe,
		)

	if err != nil {
		_ = os.RemoveAll(
			tempDir,
		)

		renderDetectError(
			w,
			r,
			"Dieses Dateiformat wird nicht unterstützt.",
		)

		return
	}

	upload, err :=
		s.uploads.Add(
			tempDir,
			inputPath,
			header.Filename,
			format,
		)

	if err != nil {
		_ = os.RemoveAll(
			tempDir,
		)

		renderDetectError(
			w,
			r,
			"Upload konnte nicht gespeichert werden.",
		)

		return
	}

	s.uploads.Delete(
		previousUploadID,
	)

	w.Header().Set(
		"Content-Type",
		"text/html; charset=utf-8",
	)

	if err := view.DetectResult(
		upload.ID,
		header.Filename,
		format,
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
	staticFS, err :=
		fs.Sub(
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
