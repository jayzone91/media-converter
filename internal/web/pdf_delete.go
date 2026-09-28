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
	maxPDFDeleteRequestSize int64 = 4 << 20

	pdfDeleteTimeout = 30 * time.Minute
)

type pdfDeleteRequest struct {
	UploadID string `json:"upload_id"`

	Pages []int `json:"pages"`
}

func (s *Server) handlePDFDeletePages(
	w http.ResponseWriter,
	r *http.Request,
) {
	started :=
		time.Now()

	r.Body =
		http.MaxBytesReader(
			w,
			r.Body,
			maxPDFDeleteRequestSize,
		)

	defer r.Body.Close()

	var request pdfDeleteRequest

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
			"PDF delete rejected",
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
			"PDF delete rejected",
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

	upload, ok :=
		s.pdfUploads.Get(
			request.UploadID,
		)

	if !ok {
		s.logWarn(
			r,
			"PDF delete rejected",
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

	retainedPages, err :=
		pagesAfterDeletion(
			request.Pages,
			upload.PageCount,
		)

	if err != nil {
		s.logWarn(
			r,
			"PDF delete rejected",
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

	if err :=
		s.acquireWorkload(
			r.Context(),
			workloadQPDF,
		); err != nil {
		s.logError(
			r,
			"PDF delete queue failed",
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
			"media-converter-pdf-delete-*",
		)

	if err != nil {
		s.logError(
			r,
			"PDF delete temp directory failed",
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
				"PDF delete cleanup failed",
				err,
				"directory",
				tempDir,
			)
		}
	}()

	outputPath :=
		filepath.Join(
			tempDir,
			"seiten-entfernt.pdf",
		)

	ctx, cancel :=
		context.WithTimeout(
			r.Context(),
			pdfDeleteTimeout,
		)

	defer cancel()

	if err :=
		s.qpdf.Reorder(
			ctx,
			upload.Path,
			retainedPages,
			outputPath,
		); err != nil {
		if writeTimeoutAPIError(
			w,
			ctx,
			"Das Erstellen der PDF hat zu lange gedauert.",
		) {
			if ctx.Err() ==
				context.DeadlineExceeded {
				s.logError(
					r,
					"PDF delete timed out",
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
			"PDF delete failed",
			err,
			"filename",
			upload.Filename,
			"pages",
			upload.PageCount,
			"selected",
			len(request.Pages),
			"size",
			upload.Size,
		)

		writeAPIError(
			w,
			http.StatusInternalServerError,
			apiErrorInternal,
			"Die ausgewählten Seiten konnten nicht entfernt werden.",
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
			"PDF delete output stat failed",
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
		"seiten-entfernt.pdf",
	) {
		return
	}

	s.pdfUploads.Delete(
		upload.ID,
	)

	s.logInfo(
		"PDF delete",
		"deleted",
		len(request.Pages),
		"remaining",
		len(retainedPages),
		"input",
		upload.Size,
		"output",
		outputSize,
		"duration",
		time.Since(started),
	)
}

func pagesAfterDeletion(
	deletedPages []int,
	pageCount int,
) ([]int, error) {
	if pageCount < 1 {
		return nil,
			fmt.Errorf(
				"invalid page count",
			)
	}

	if len(deletedPages) == 0 {
		return nil,
			fmt.Errorf(
				"no pages selected",
			)
	}

	if len(deletedPages) >=
		pageCount {
		return nil,
			fmt.Errorf(
				"all pages selected",
			)
	}

	deleted :=
		make(
			[]bool,
			pageCount+1,
		)

	for _, page := range deletedPages {
		if page < 1 ||
			page > pageCount {
			return nil,
				fmt.Errorf(
					"page %d is outside range 1-%d",
					page,
					pageCount,
				)
		}

		if deleted[page] {
			return nil,
				fmt.Errorf(
					"page %d occurs more than once",
					page,
				)
		}

		deleted[page] = true
	}

	retained :=
		make(
			[]int,
			0,
			pageCount-
				len(
					deletedPages,
				),
		)

	for page :=
		1; page <= pageCount; page++ {
		if deleted[page] {
			continue
		}

		retained =
			append(
				retained,
				page,
			)
	}

	return retained, nil
}
