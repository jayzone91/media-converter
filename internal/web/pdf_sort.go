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
	maxPDFSortRequestSize int64 = 4 << 20

	pdfSortTimeout = 30 * time.Minute
)

type pdfSortRequest struct {
	UploadID string `json:"upload_id"`

	Pages []int `json:"pages"`
}

func (s *Server) handlePDFSort(
	w http.ResponseWriter,
	r *http.Request,
) {
	started :=
		time.Now()

	r.Body =
		http.MaxBytesReader(
			w,
			r.Body,
			maxPDFSortRequestSize,
		)

	defer r.Body.Close()

	var request pdfSortRequest

	decoder :=
		json.NewDecoder(
			r.Body,
		)

	decoder.DisallowUnknownFields()

	if err :=
		decoder.Decode(
			&request,
		); err != nil {
		s.logWarn(
			r,
			"PDF sort rejected",
			"reason",
			"invalid request",
			"error",
			err,
		)

		writeAPIError(
			w,
			http.StatusBadRequest,
			apiErrorInvalidRequest,
			"Ungültige Sortier-Anfrage.",
		)

		return
	}

	upload, ok :=
		s.pdfUploads.Get(
			request.UploadID,
		)

	if !ok {
		s.logWarn(
			r,
			"PDF sort rejected",
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

	if err :=
		validatePDFPageOrder(
			request.Pages,
			upload.PageCount,
		); err != nil {
		s.logWarn(
			r,
			"PDF sort rejected",
			"reason",
			err.Error(),
			"filename",
			upload.Filename,
			"pages",
			upload.PageCount,
			"requested_pages",
			len(request.Pages),
		)

		writeAPIError(
			w,
			http.StatusBadRequest,
			apiErrorInvalidRequest,
			"Die Seitenreihenfolge ist ungültig.",
		)

		return
	}

	if err :=
		s.acquireWorkload(
			r.Context(),
			workloadQPDF,
		); err != nil {
		s.logError(
			r,
			"PDF sort queue failed",
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

	tempDir, err :=
		os.MkdirTemp(
			"",
			"media-converter-pdf-sort-*",
		)

	if err != nil {
		s.logError(
			r,
			"PDF sort temp directory failed",
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
		if err :=
			os.RemoveAll(
				tempDir,
			); err != nil {
			s.logError(
				r,
				"PDF sort cleanup failed",
				err,
				"directory",
				tempDir,
			)
		}
	}()

	outputPath :=
		filepath.Join(
			tempDir,
			"sortiert.pdf",
		)

	ctx, cancel :=
		context.WithTimeout(
			r.Context(),
			pdfSortTimeout,
		)

	defer cancel()

	if err :=
		s.qpdf.Reorder(
			ctx,
			upload.Path,
			request.Pages,
			outputPath,
		); err != nil {
		if writeTimeoutAPIError(
			w,
			ctx,
			"Das Sortieren der PDF hat zu lange gedauert.",
		) {
			if ctx.Err() ==
				context.DeadlineExceeded {
				s.logError(
					r,
					"PDF sort timed out",
					ctx.Err(),
					"filename",
					upload.Filename,
					"pages",
					upload.PageCount,
				)
			}

			return
		}

		s.logError(
			r,
			"PDF sort failed",
			err,
			"filename",
			upload.Filename,
			"pages",
			upload.PageCount,
			"size",
			upload.Size,
		)

		writeAPIError(
			w,
			http.StatusInternalServerError,
			apiErrorInternal,
			"Die PDF-Seiten konnten nicht sortiert werden.",
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
			"PDF sort output stat failed",
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
		"sortiert.pdf",
	) {
		return
	}

	s.pdfUploads.Delete(
		upload.ID,
	)

	s.logInfo(
		"PDF sort",
		"pages",
		upload.PageCount,
		"input",
		upload.Size,
		"output",
		outputSize,
		"duration",
		time.Since(started),
	)
}

func validatePDFPageOrder(
	pages []int,
	pageCount int,
) error {
	if len(pages) !=
		pageCount {
		return fmt.Errorf(
			"expected %d pages, got %d",
			pageCount,
			len(pages),
		)
	}

	seen :=
		make(
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
