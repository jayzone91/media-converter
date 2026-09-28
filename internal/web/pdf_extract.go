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
		s.logError(
			r,
			"failed to decode PDF extract request",
			err,
		)

		http.Error(
			w,
			"Ungültige Anfrage.",
			http.StatusBadRequest,
		)

		return
	}

	if request.UploadID == "" {
		http.Error(
			w,
			"Upload-ID fehlt.",
			http.StatusBadRequest,
		)

		return
	}

	upload, ok := s.pdfUploads.Get(
		request.UploadID,
	)

	if !ok {
		s.logWarn(
			r,
			"PDF page extraction requested for unknown upload",
			"upload_id",
			request.UploadID,
		)

		http.Error(
			w,
			"Die PDF ist nicht mehr verfügbar. Bitte erneut hochladen.",
			http.StatusGone,
		)

		return
	}

	if err := validatePDFPageSelection(
		request.Pages,
		upload.PageCount,
	); err != nil {
		s.logWarn(
			r,
			"invalid PDF page extraction request",
			"upload_id",
			upload.ID,
			"filename",
			upload.Filename,
			"page_count",
			upload.PageCount,
			"selected_pages",
			len(request.Pages),
			"reason",
			err.Error(),
		)

		http.Error(
			w,
			"Die Seitenauswahl ist ungültig.",
			http.StatusBadRequest,
		)

		return
	}

	if err := s.acquireWorkload(
		r.Context(),
		workloadQPDF,
	); err != nil {
		s.logError(
			r,
			"failed to acquire PDF extract qpdf workload",
			err,
			"upload_id",
			upload.ID,
			"filename",
			upload.Filename,
		)

		http.Error(
			w,
			"Der Server ist momentan ausgelastet. Bitte später erneut versuchen.",
			http.StatusServiceUnavailable,
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
			"failed to create PDF extract temporary directory",
			err,
			"upload_id",
			upload.ID,
			"filename",
			upload.Filename,
		)

		http.Error(
			w,
			"Temporäres Verzeichnis konnte nicht erstellt werden.",
			http.StatusInternalServerError,
		)

		return
	}

	defer func() {
		if err := os.RemoveAll(
			tempDir,
		); err != nil {
			s.logError(
				r,
				"failed to remove PDF extract temporary directory",
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
		s.logError(
			r,
			"PDF page extraction failed",
			err,
			"upload_id",
			upload.ID,
			"filename",
			upload.Filename,
			"size_bytes",
			upload.Size,
			"page_count",
			upload.PageCount,
			"selected_pages",
			len(request.Pages),
		)

		if errors.Is(
			ctx.Err(),
			context.DeadlineExceeded,
		) {
			http.Error(
				w,
				"Das Extrahieren der Seiten hat zu lange gedauert.",
				http.StatusGatewayTimeout,
			)

			return
		}

		http.Error(
			w,
			"Die ausgewählten Seiten konnten nicht extrahiert werden.",
			http.StatusInternalServerError,
		)

		return
	}

	outputSize, err :=
		downloadFileSize(
			outputPath,
		)

	if err != nil {
		s.logError(
			r,
			"failed to inspect extracted PDF",
			err,
			"upload_id",
			upload.ID,
			"filename",
			upload.Filename,
		)

		http.Error(
			w,
			"Die erzeugte PDF konnte nicht gelesen werden.",
			http.StatusInternalServerError,
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
