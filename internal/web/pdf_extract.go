package web

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

const (
	maxPDFExtractRequestSize int64 = 4 << 20

	pdfExtractTimeout = 30 * time.Minute
)

type pdfExtractRequest struct {
	UploadID string `json:"upload_id"`
	Pages    []int  `json:"pages"`
}

func (s *Server) handlePDFExtractPages(
	w http.ResponseWriter,
	r *http.Request,
) {
	started := time.Now()

	r.Body = http.MaxBytesReader(
		w,
		r.Body,
		maxPDFExtractRequestSize,
	)

	defer r.Body.Close()

	var request pdfExtractRequest

	decoder := json.NewDecoder(
		r.Body,
	)

	decoder.DisallowUnknownFields()

	if err := decoder.Decode(
		&request,
	); err != nil {
		s.logWarn(
			r,
			"PDF extract rejected",
			"reason",
			"invalid request",
			"error",
			err,
		)

		writeAPIError(
			w,
			http.StatusBadRequest,
			apiErrorInvalidRequest,
			"Ungültige Anfrage.",
		)

		return
	}

	if request.UploadID == "" {
		s.logWarn(
			r,
			"PDF extract rejected",
			"reason",
			"missing upload id",
		)

		writeAPIError(
			w,
			http.StatusBadRequest,
			apiErrorInvalidRequest,
			"Upload-ID fehlt.",
		)

		return
	}

	upload, ok := s.pdfUploads.Get(
		request.UploadID,
	)

	if !ok {
		s.logWarn(
			r,
			"PDF extract rejected",
			"reason",
			"unknown upload",
			"upload_id",
			request.UploadID,
		)

		writeAPIError(
			w,
			http.StatusGone,
			apiErrorUploadExpired,
			"Die PDF ist nicht mehr verfügbar. Bitte erneut hochladen.",
		)

		return
	}

	if err := validatePDFPageSelection(
		request.Pages,
		upload.PageCount,
	); err != nil {
		s.logWarn(
			r,
			"PDF extract rejected",
			"reason",
			err.Error(),
			"filename",
			upload.Filename,
			"pages",
			upload.PageCount,
			"selected",
			len(request.Pages),
		)

		writeAPIError(
			w,
			http.StatusBadRequest,
			apiErrorInvalidRequest,
			"Die Seitenauswahl ist ungültig.",
		)

		return
	}

	if err := s.acquireWorkload(
		r.Context(),
		workloadQPDF,
	); err != nil {
		s.logError(
			r,
			"PDF extract queue failed",
			err,
			"filename",
			upload.Filename,
		)

		writeQueueAPIError(
			w,
			err,
		)

		return
	}

	defer s.releaseWorkload(
		workloadQPDF,
	)

	tempDir, err := os.MkdirTemp(
		"",
		"media-converter-pdf-extract-*",
	)

	if err != nil {
		s.logError(
			r,
			"PDF extract temp directory failed",
			err,
			"filename",
			upload.Filename,
		)

		writeAPIError(
			w,
			http.StatusInternalServerError,
			apiErrorInternal,
			"Temporäres Verzeichnis konnte nicht erstellt werden.",
		)

		return
	}

	defer func() {
		if err := os.RemoveAll(
			tempDir,
		); err != nil {
			s.logError(
				r,
				"PDF extract cleanup failed",
				err,
				"directory",
				tempDir,
			)
		}
	}()

	outputPath := filepath.Join(
		tempDir,
		"extrahiert.pdf",
	)

	ctx, cancel := context.WithTimeout(
		r.Context(),
		pdfExtractTimeout,
	)

	defer cancel()

	if err := s.qpdf.Reorder(
		ctx,
		upload.Path,
		request.Pages,
		outputPath,
	); err != nil {
		if writeTimeoutAPIError(
			w,
			ctx,
			"Das Extrahieren der Seiten hat zu lange gedauert.",
		) {
			if ctx.Err() ==
				context.DeadlineExceeded {
				s.logError(
					r,
					"PDF extract timed out",
					ctx.Err(),
					"filename",
					upload.Filename,
					"pages",
					upload.PageCount,
					"selected",
					len(request.Pages),
				)
			}

			return
		}

		s.logError(
			r,
			"PDF extract failed",
			err,
			"filename",
			upload.Filename,
			"size",
			upload.Size,
			"pages",
			upload.PageCount,
			"selected",
			len(request.Pages),
		)

		writeAPIError(
			w,
			http.StatusInternalServerError,
			apiErrorInternal,
			"Die ausgewählten Seiten konnten nicht extrahiert werden.",
		)

		return
	}

	outputSize, err := downloadFileSize(
		outputPath,
	)

	if err != nil {
		s.logError(
			r,
			"PDF extract output stat failed",
			err,
			"filename",
			upload.Filename,
		)

		writeAPIError(
			w,
			http.StatusInternalServerError,
			apiErrorInternal,
			"Die erzeugte PDF konnte nicht gelesen werden.",
		)

		return
	}

	if !s.prepareDownloadResponse(
		w,
		r,
		outputPath,
		"extrahiert.pdf",
	) {
		return
	}

	s.pdfUploads.Delete(
		upload.ID,
	)

	s.logInfo(
		"PDF pages extracted",
		"pages",
		upload.PageCount,
		"extracted",
		len(request.Pages),
		"input",
		upload.Size,
		"output",
		outputSize,
		"duration",
		time.Since(started),
	)
}

func validatePDFPageSelection(
	pages []int,
	pageCount int,
) error {
	if pageCount < 1 {
		return fmt.Errorf(
			"invalid page count",
		)
	}

	if len(pages) == 0 {
		return fmt.Errorf(
			"no pages selected",
		)
	}

	if len(pages) > pageCount {
		return fmt.Errorf(
			"too many pages selected",
		)
	}

	seen := make(
		[]bool,
		pageCount+1,
	)

	for _, page := range pages {
		if page < 1 ||
			page > pageCount {
			return fmt.Errorf(
				"page %d is outside range 1-%d",
				page,
				pageCount,
			)
		}

		if seen[page] {
			return fmt.Errorf(
				"page %d occurs more than once",
				page,
			)
		}

		seen[page] = true
	}

	return nil
}
