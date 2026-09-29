package web

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"path/filepath"
)

const (
	pdfUploadRequestOverhead int64 = 1 << 20
)

type pdfUploadResponse struct {
	ID        string   `json:"id"`
	Filename  string   `json:"filename"`
	Size      int64    `json:"size"`
	PageCount int      `json:"page_count"`
	Previews  []string `json:"previews"`
}

func (s *Server) handlePDFUpload(
	w http.ResponseWriter,
	r *http.Request,
) {
	r.Body = http.MaxBytesReader(
		w,
		r.Body,
		maxFileSize+pdfUploadRequestOverhead,
	)

	if err := r.ParseMultipartForm(
		multipartMemoryLimit,
	); err != nil {
		var maxBytesError *http.MaxBytesError

		if errors.As(
			err,
			&maxBytesError,
		) {
			s.logWarn(
				r,
				"PDF upload rejected because request is too large",
				"limit_bytes",
				maxFileSize,
			)

			writeAPIError(
				w,
				http.StatusRequestEntityTooLarge,
				apiErrorInvalidRequest,
				"Die PDF ist größer als 512 MiB.",
			)

			return
		}

		s.logError(
			r,
			"failed to parse PDF upload",
			err,
		)

		writeAPIError(
			w,
			http.StatusBadRequest,
			apiErrorInvalidRequest,
			"Ungültiger Upload.",
		)

		return
	}

	if r.MultipartForm != nil {
		defer r.MultipartForm.RemoveAll()
	}

	file, header, err :=
		r.FormFile(
			"file",
		)

	if err != nil {
		s.logWarn(
			r,
			"PDF upload contains no file",
		)

		writeAPIError(
			w,
			http.StatusBadRequest,
			apiErrorInvalidRequest,
			"Keine PDF-Datei ausgewählt.",
		)

		return
	}

	defer file.Close()

	if err := validatePDFUpload(
		header,
	); err != nil {
		s.logWarn(
			r,
			"PDF upload validation failed",
			"filename",
			header.Filename,
			"size_bytes",
			header.Size,
			"reason",
			err.Error(),
		)

		writeAPIError(
			w,
			http.StatusBadRequest,
			apiErrorInvalidRequest,
			err.Error(),
		)

		return
	}

	tempDir, err :=
		os.MkdirTemp(
			"",
			"media-converter-pdf-*",
		)

	if err != nil {
		s.logError(
			r,
			"failed to create PDF temporary directory",
			err,
			"filename",
			header.Filename,
			"size_bytes",
			header.Size,
		)

		writeAPIError(
			w,
			http.StatusInternalServerError,
			apiErrorInternal,
			"Temporäres Verzeichnis konnte nicht erstellt werden.",
		)

		return
	}

	keepTempDir := false

	defer func() {
		if keepTempDir {
			return
		}

		if err := os.RemoveAll(
			tempDir,
		); err != nil {
			s.logError(
				r,
				"failed to remove PDF temporary directory",
				err,
				"directory",
				tempDir,
			)
		}
	}()

	inputPath :=
		filepath.Join(
			tempDir,
			"input.pdf",
		)

	size, err :=
		savePDFUpload(
			file,
			inputPath,
		)

	if err != nil {
		s.logError(
			r,
			"failed to save PDF upload",
			err,
			"filename",
			header.Filename,
			"declared_size_bytes",
			header.Size,
		)

		writeAPIError(
			w,
			http.StatusInternalServerError,
			apiErrorInternal,
			"PDF konnte nicht gespeichert werden.",
		)

		return
	}

	if err := validatePDFSignature(
		inputPath,
	); err != nil {
		s.logWarn(
			r,
			"PDF signature validation failed",
			"filename",
			header.Filename,
			"size_bytes",
			size,
			"reason",
			err.Error(),
		)

		writeAPIError(
			w,
			http.StatusBadRequest,
			apiErrorInvalidRequest,
			"Die Datei ist keine gültige PDF.",
		)

		return
	}

	pageCount, err :=
		s.readPDFPageCount(
			r,
			inputPath,
			header.Filename,
			size,
		)

	if err != nil {
		if errors.Is(
			err,
			context.DeadlineExceeded,
		) {
			writeAPIError(
				w,
				http.StatusGatewayTimeout,
				apiErrorTimeout,
				"Die PDF konnte nicht rechtzeitig analysiert werden.",
			)

			return
		}

		writeAPIError(
			w,
			http.StatusBadRequest,
			apiErrorInvalidRequest,
			"Die PDF konnte nicht gelesen werden.",
		)

		return
	}

	previewDirectory :=
		filepath.Join(
			tempDir,
			"previews",
		)

	if err := os.MkdirAll(
		previewDirectory,
		0700,
	); err != nil {
		s.logError(
			r,
			"failed to create PDF preview directory",
			err,
			"filename",
			header.Filename,
			"size_bytes",
			size,
			"page_count",
			pageCount,
		)

		writeAPIError(
			w,
			http.StatusInternalServerError,
			apiErrorInternal,
			"Vorschau-Verzeichnis konnte nicht erstellt werden.",
		)

		return
	}

	upload, err :=
		s.pdfUploads.Add(
			tempDir,
			inputPath,
			previewDirectory,
			header.Filename,
			size,
			pageCount,
		)

	if err != nil {
		s.logError(
			r,
			"failed to register PDF upload",
			err,
			"filename",
			header.Filename,
			"size_bytes",
			size,
			"page_count",
			pageCount,
		)

		writeAPIError(
			w,
			http.StatusInternalServerError,
			apiErrorInternal,
			"PDF-Upload konnte nicht gespeichert werden.",
		)

		return
	}

	keepTempDir = true

	response :=
		pdfUploadResponse{
			ID: upload.ID,

			Filename: upload.Filename,

			Size: upload.Size,

			PageCount: upload.PageCount,

			Previews: buildPDFPreviewURLs(
				upload,
			),
		}

	w.Header().Set(
		"Content-Type",
		"application/json; charset=utf-8",
	)

	w.Header().Set(
		"Cache-Control",
		"no-store",
	)

	if err :=
		json.NewEncoder(
			w,
		).Encode(
			response,
		); err != nil {
		s.logError(
			r,
			"failed to encode PDF upload response",
			err,
			"upload_id",
			upload.ID,
			"filename",
			upload.Filename,
			"size_bytes",
			upload.Size,
			"page_count",
			upload.PageCount,
		)

		s.pdfUploads.Delete(
			upload.ID,
		)

		return
	}

	s.logger.Info(
		"PDF upload ready",
		"method",
		r.Method,
		"path",
		r.URL.Path,
		"upload_id",
		upload.ID,
		"filename",
		upload.Filename,
		"size_bytes",
		upload.Size,
		"page_count",
		upload.PageCount,
	)
}
