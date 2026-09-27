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
	started := time.Now()

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

		http.Error(
			w,
			"Ungültige Sortier-Anfrage.",
			http.StatusBadRequest,
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

		http.Error(
			w,
			"Die PDF ist nicht mehr verfügbar. Bitte erneut hochladen.",
			http.StatusGone,
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

		http.Error(
			w,
			"Die Seitenreihenfolge ist ungültig.",
			http.StatusBadRequest,
		)

		return
	}

	if err :=
		s.acquireConversionSlot(
			r.Context(),
		); err != nil {
		s.logError(
			r,
			"PDF sort queue failed",
			err,
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

	defer s.releaseConversionSlot()

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

		http.Error(
			w,
			"Temporäres Verzeichnis konnte nicht erstellt werden.",
			http.StatusInternalServerError,
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
		if errors.Is(
			ctx.Err(),
			context.DeadlineExceeded,
		) {
			s.logError(
				r,
				"PDF sort timed out",
				ctx.Err(),
				"filename",
				upload.Filename,
				"pages",
				upload.PageCount,
			)

			http.Error(
				w,
				"Das Sortieren der PDF hat zu lange gedauert.",
				http.StatusGatewayTimeout,
			)

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

		http.Error(
			w,
			"Die PDF-Seiten konnten nicht sortiert werden.",
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
			"PDF sort output stat failed",
			err,
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
