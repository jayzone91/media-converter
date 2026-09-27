package web

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
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
		s.logError(
			r,
			"failed to decode PDF sort request",
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
			"PDF sort requested for unknown upload",
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
			"invalid PDF sort page order",
			"upload_id",
			upload.ID,
			"filename",
			upload.Filename,
			"page_count",
			upload.PageCount,
			"requested_pages",
			len(request.Pages),
			"reason",
			err.Error(),
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
			"failed to acquire PDF sort conversion slot",
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

	defer s.releaseConversionSlot()

	tempDir, err :=
		os.MkdirTemp(
			"",
			"media-converter-pdf-sort-*",
		)

	if err != nil {
		s.logError(
			r,
			"failed to create PDF sort temporary directory",
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
		if err :=
			os.RemoveAll(
				tempDir,
			); err != nil {
			s.logError(
				r,
				"failed to remove PDF sort temporary directory",
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
		s.logError(
			r,
			"PDF page sorting failed",
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

		if errors.Is(
			ctx.Err(),
			context.DeadlineExceeded,
		) {
			http.Error(
				w,
				"Das Sortieren der PDF hat zu lange gedauert.",
				http.StatusGatewayTimeout,
			)

			return
		}

		http.Error(
			w,
			"Die PDF-Seiten konnten nicht sortiert werden.",
			http.StatusInternalServerError,
		)

		return
	}

	file, err :=
		os.Open(
			outputPath,
		)

	if err != nil {
		s.logError(
			r,
			"failed to open sorted PDF",
			err,
			"upload_id",
			upload.ID,
			"filename",
			upload.Filename,
		)

		http.Error(
			w,
			"Die erzeugte PDF konnte nicht geöffnet werden.",
			http.StatusInternalServerError,
		)

		return
	}

	defer file.Close()

	info, err :=
		file.Stat()

	if err != nil {
		s.logError(
			r,
			"failed to inspect sorted PDF",
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

	disposition :=
		mime.FormatMediaType(
			"attachment",
			map[string]string{
				"filename": "sortiert.pdf",
			},
		)

	w.Header().Set(
		"Content-Type",
		"application/pdf",
	)

	w.Header().Set(
		"Content-Disposition",
		disposition,
	)

	w.Header().Set(
		"Content-Length",
		fmt.Sprintf(
			"%d",
			info.Size(),
		),
	)

	w.Header().Set(
		"Cache-Control",
		"no-store",
	)

	/*
		Die Quelldatei kann jetzt entfernt werden.
		Die Ausgabe liegt in einem separaten Temp-Verzeichnis.
	*/
	s.pdfUploads.Delete(
		upload.ID,
	)

	s.logger.Info(
		"PDF pages sorted",
		"method",
		r.Method,
		"path",
		r.URL.Path,
		"upload_id",
		upload.ID,
		"filename",
		upload.Filename,
		"page_count",
		upload.PageCount,
		"input_size_bytes",
		upload.Size,
		"output_size_bytes",
		info.Size(),
	)

	if _, err :=
		io.Copy(
			w,
			file,
		); err != nil {
		s.logError(
			r,
			"failed to send sorted PDF",
			err,
			"filename",
			upload.Filename,
			"output_size_bytes",
			info.Size(),
		)

		return
	}
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
