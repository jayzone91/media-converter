package web

import (
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"path/filepath"
)

type rawPDFUploadResponse struct {
	ID       string `json:"id"`
	Filename string `json:"filename"`
	Size     int64  `json:"size"`
}

func (s *Server) handlePDFRawUpload(
	w http.ResponseWriter,
	r *http.Request,
) {
	r.Body =
		http.MaxBytesReader(
			w,
			r.Body,
			maxFileSize+pdfUploadRequestOverhead,
		)

	if err :=
		r.ParseMultipartForm(
			multipartMemoryLimit,
		); err != nil {
		var maxBytesError *http.MaxBytesError

		if errors.As(
			err,
			&maxBytesError,
		) {
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
			"failed to parse raw PDF upload",
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
		writeAPIError(
			w,
			http.StatusBadRequest,
			apiErrorInvalidRequest,
			"Keine PDF-Datei ausgewählt.",
		)

		return
	}

	defer file.Close()

	if err :=
		validatePDFUpload(
			header,
		); err != nil {
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
			"media-converter-pdf-raw-*",
		)

	if err != nil {
		s.logError(
			r,
			"failed to create raw PDF temporary directory",
			err,
			"filename",
			header.Filename,
		)

		writeAPIError(
			w,
			http.StatusInternalServerError,
			apiErrorInternal,
			"Temporäres Verzeichnis konnte nicht erstellt werden.",
		)

		return
	}

	keepTempDir :=
		false

	defer func() {
		if keepTempDir {
			return
		}

		if err :=
			os.RemoveAll(
				tempDir,
			); err != nil {
			s.logError(
				r,
				"failed to remove raw PDF temporary directory",
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
			"failed to save raw PDF upload",
			err,
			"filename",
			header.Filename,
		)

		writeAPIError(
			w,
			http.StatusInternalServerError,
			apiErrorInternal,
			"PDF konnte nicht gespeichert werden.",
		)

		return
	}

	if err :=
		validatePDFSignature(
			inputPath,
		); err != nil {
		s.logWarn(
			r,
			"raw PDF signature validation failed",
			"filename",
			header.Filename,
			"size_bytes",
			size,
		)

		writeAPIError(
			w,
			http.StatusBadRequest,
			apiErrorInvalidRequest,
			"Die Datei ist keine gültige PDF.",
		)

		return
	}

	previewDirectory :=
		filepath.Join(
			tempDir,
			"previews",
		)

	upload, err :=
		s.pdfUploads.Add(
			tempDir,
			inputPath,
			previewDirectory,
			header.Filename,
			size,
			0,
		)

	if err != nil {
		s.logError(
			r,
			"failed to register raw PDF upload",
			err,
			"filename",
			header.Filename,
		)

		writeAPIError(
			w,
			http.StatusInternalServerError,
			apiErrorInternal,
			"PDF-Upload konnte nicht gespeichert werden.",
		)

		return
	}

	keepTempDir =
		true

	response :=
		rawPDFUploadResponse{
			ID: upload.ID,

			Filename: upload.Filename,

			Size: upload.Size,
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
			"failed to encode raw PDF upload response",
			err,
			"upload_id",
			upload.ID,
		)

		s.pdfUploads.Delete(
			upload.ID,
		)

		return
	}

	s.logger.Info(
		"raw PDF upload ready",
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
	)
}
