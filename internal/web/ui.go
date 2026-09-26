package web

import (
	"embed"
	"html/template"
	"io/fs"
	"net/http"
	"os"
	"strings"

	"github.com/jayzone91/media-converter/internal/media"
)

//go:embed templates/*.html static/*
var uiFiles embed.FS

var uiTemplates = template.Must(
	template.New("").
		Funcs(template.FuncMap{
			"upper": strings.ToUpper,
		}).
		ParseFS(
			uiFiles,
			"templates/*.html",
		),
)

type detectTemplateData struct {
	UploadID string
	Filename string
	Format   media.Format
}

func (s *Server) handleIndex(
	w http.ResponseWriter,
	_ *http.Request,
) {
	w.Header().Set(
		"Content-Type",
		"text/html; charset=utf-8",
	)

	if err := uiTemplates.ExecuteTemplate(
		w,
		"index.html",
		nil,
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

	file, header, err := r.FormFile("file")
	if err != nil {
		renderDetectError(
			w,
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
			"Die Datei ist größer als 512 MiB.",
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
			"Temporäres Verzeichnis konnte nicht erstellt werden.",
		)
		return
	}

	inputPath, err := saveUpload(
		file,
		header.Filename,
		tempDir,
	)
	if err != nil {
		_ = os.RemoveAll(tempDir)

		renderDetectError(
			w,
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
		_ = os.RemoveAll(tempDir)

		renderDetectError(
			w,
			"Dieses Dateiformat wird nicht unterstützt.",
		)
		return
	}

	upload, err := s.uploads.Add(
		tempDir,
		inputPath,
		header.Filename,
		format,
	)
	if err != nil {
		_ = os.RemoveAll(tempDir)

		renderDetectError(
			w,
			"Upload konnte nicht gespeichert werden.",
		)
		return
	}

	// Erst löschen, nachdem der neue Upload sicher gespeichert wurde.
	s.uploads.Delete(previousUploadID)

	data := detectTemplateData{
		UploadID: upload.ID,
		Filename: header.Filename,
		Format:   format,
	}

	w.Header().Set(
		"Content-Type",
		"text/html; charset=utf-8",
	)

	if err := uiTemplates.ExecuteTemplate(
		w,
		"detect.html",
		data,
	); err != nil {
		s.uploads.Delete(upload.ID)

		http.Error(
			w,
			"failed to render detection result",
			http.StatusInternalServerError,
		)
	}
}

func renderDetectError(
	w http.ResponseWriter,
	message string,
) {
	_ = uiTemplates.ExecuteTemplate(
		w,
		"detect-error.html",
		message,
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
