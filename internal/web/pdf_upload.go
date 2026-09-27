package web

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

const (
	pdfUploadRequestOverhead int64 = 1 << 20
	pdfMetadataTimeout             = 30 * time.Second
)

type pdfUploadResponse struct {
	ID        string   `json:"id"`
	Filename  string   `json:"filename"`
	Size      int64    `json:"size"`
	PageCount int      `json:"page_count"`
	Previews  []string `json:"previews"`
}

func (s *Server) handlePDFUpload(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(
		w,
		r.Body,
		maxFileSize+pdfUploadRequestOverhead,
	)

	if err := r.ParseMultipartForm(multipartMemoryLimit); err != nil {
		var maxBytesError *http.MaxBytesError

		if errors.As(err, &maxBytesError) {
			s.logWarn(
				r,
				"PDF upload rejected because request is too large",
				"limit_bytes",
				maxFileSize,
			)

			http.Error(
				w,
				"Die PDF ist größer als 512 MiB.",
				http.StatusRequestEntityTooLarge,
			)
			return
		}

		s.logError(
			r,
			"failed to parse PDF upload",
			err,
		)

		http.Error(
			w,
			"Ungültiger Upload.",
			http.StatusBadRequest,
		)
		return
	}

	if r.MultipartForm != nil {
		defer r.MultipartForm.RemoveAll()
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		s.logWarn(
			r,
			"PDF upload contains no file",
		)

		http.Error(
			w,
			"Keine PDF-Datei ausgewählt.",
			http.StatusBadRequest,
		)
		return
	}
	defer file.Close()

	if err := validatePDFUpload(header); err != nil {
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

		http.Error(
			w,
			err.Error(),
			http.StatusBadRequest,
		)
		return
	}

	tempDir, err := os.MkdirTemp("", "media-converter-pdf-*")
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

		http.Error(
			w,
			"Temporäres Verzeichnis konnte nicht erstellt werden.",
			http.StatusInternalServerError,
		)
		return
	}

	keepTempDir := false

	defer func() {
		if keepTempDir {
			return
		}

		if err := os.RemoveAll(tempDir); err != nil {
			s.logError(
				r,
				"failed to remove PDF temporary directory",
				err,
				"directory",
				tempDir,
			)
		}
	}()

	inputPath := filepath.Join(
		tempDir,
		"input.pdf",
	)

	size, err := savePDFUpload(
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

		http.Error(
			w,
			"PDF konnte nicht gespeichert werden.",
			http.StatusInternalServerError,
		)
		return
	}

	if err := validatePDFSignature(inputPath); err != nil {
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

		http.Error(
			w,
			"Die Datei ist keine gültige PDF.",
			http.StatusBadRequest,
		)
		return
	}

	pageCount, err := s.readPDFPageCount(
		r,
		inputPath,
		header.Filename,
		size,
	)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			http.Error(
				w,
				"Die PDF konnte nicht rechtzeitig analysiert werden.",
				http.StatusGatewayTimeout,
			)
			return
		}

		http.Error(
			w,
			"Die PDF konnte nicht gelesen werden.",
			http.StatusBadRequest,
		)
		return
	}

	previewDirectory := filepath.Join(
		tempDir,
		"previews",
	)

	if err := os.MkdirAll(previewDirectory, 0700); err != nil {
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

		http.Error(
			w,
			"Vorschau-Verzeichnis konnte nicht erstellt werden.",
			http.StatusInternalServerError,
		)
		return
	}

	upload, err := s.pdfUploads.Add(
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

		http.Error(
			w,
			"PDF-Upload konnte nicht gespeichert werden.",
			http.StatusInternalServerError,
		)
		return
	}

	keepTempDir = true

	response := pdfUploadResponse{
		ID:        upload.ID,
		Filename:  upload.Filename,
		Size:      upload.Size,
		PageCount: upload.PageCount,
		Previews:  buildPDFPreviewURLs(upload),
	}

	w.Header().Set(
		"Content-Type",
		"application/json; charset=utf-8",
	)
	w.Header().Set(
		"Cache-Control",
		"no-store",
	)

	if err := json.NewEncoder(w).Encode(response); err != nil {
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

		s.pdfUploads.Delete(upload.ID)
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

func (s *Server) readPDFPageCount(
	r *http.Request,
	inputPath string,
	filename string,
	size int64,
) (int, error) {
	ctx, cancel := context.WithTimeout(
		r.Context(),
		pdfMetadataTimeout,
	)
	defer cancel()

	metadata, err := s.qpdf.PageCount(
		ctx,
		inputPath,
	)
	if err != nil {
		s.logError(
			r,
			"PDF metadata analysis failed",
			err,
			"filename",
			filename,
			"size_bytes",
			size,
			"timeout",
			pdfMetadataTimeout.String(),
		)

		if errors.Is(ctx.Err(), context.DeadlineExceeded) ||
			errors.Is(err, context.DeadlineExceeded) {
			return 0, context.DeadlineExceeded
		}

		return 0, err
	}

	if metadata.Warnings != "" {
		s.logWarn(
			r,
			"qpdf recovered PDF with warnings",
			"filename",
			filename,
			"size_bytes",
			size,
			"page_count",
			metadata.Count,
			"warnings",
			metadata.Warnings,
		)
	}

	return metadata.Count, nil
}

func buildPDFPreviewURLs(
	upload storedPDFUpload,
) []string {
	previews := make(
		[]string,
		upload.PageCount,
	)

	for index := range previews {
		previews[index] = fmt.Sprintf(
			"/pdf/uploads/%s/pages/%d",
			upload.ID,
			index+1,
		)
	}

	return previews
}

func (s *Server) handlePDFUploadDelete(
	w http.ResponseWriter,
	r *http.Request,
) {
	id := r.PathValue("id")
	if id == "" {
		http.NotFound(w, r)
		return
	}

	upload, ok := s.pdfUploads.Get(id)
	if !ok {
		http.NotFound(w, r)
		return
	}

	s.pdfUploads.Delete(id)

	s.logger.Info(
		"PDF upload deleted",
		"method",
		r.Method,
		"path",
		r.URL.Path,
		"upload_id",
		id,
		"filename",
		upload.Filename,
	)

	w.WriteHeader(http.StatusNoContent)
}
