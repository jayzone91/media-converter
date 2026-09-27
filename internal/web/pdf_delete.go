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
	started := time.Now()

	r.Body = http.MaxBytesReader(
		w,
		r.Body,
		maxPDFDeleteRequestSize,
	)

	defer r.Body.Close()

	var request pdfDeleteRequest

	decoder := json.NewDecoder(
		r.Body,
	)

	decoder.DisallowUnknownFields()

	if err := decoder.Decode(
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

		http.Error(
			w,
			"Ungültige Anfrage.",
			http.StatusBadRequest,
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
			"PDF delete rejected",
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

	retainedPages, err := pagesAfterDeletion(
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

		http.Error(
			w,
			"Die Seitenauswahl ist ungültig.",
			http.StatusBadRequest,
		)

		return
	}

	if err := s.acquireConversionSlot(
		r.Context(),
	); err != nil {
		s.logError(
			r,
			"PDF delete queue failed",
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

	tempDir, err := os.MkdirTemp(
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
				"PDF delete cleanup failed",
				err,
				"directory",
				tempDir,
			)
		}
	}()

	outputPath := filepath.Join(
		tempDir,
		"seiten-entfernt.pdf",
	)

	ctx, cancel := context.WithTimeout(
		r.Context(),
		pdfDeleteTimeout,
	)

	defer cancel()

	if err := s.qpdf.Reorder(
		ctx,
		upload.Path,
		retainedPages,
		outputPath,
	); err != nil {
		if errors.Is(
			ctx.Err(),
			context.DeadlineExceeded,
		) {
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

			http.Error(
				w,
				"Das Erstellen der PDF hat zu lange gedauert.",
				http.StatusGatewayTimeout,
			)

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

		http.Error(
			w,
			"Die ausgewählten Seiten konnten nicht entfernt werden.",
			http.StatusInternalServerError,
		)

		return
	}

	file, err := os.Open(
		outputPath,
	)

	if err != nil {
		s.logError(
			r,
			"PDF delete output open failed",
			err,
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

	info, err := file.Stat()

	if err != nil {
		s.logError(
			r,
			"PDF delete output stat failed",
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

	disposition := mime.FormatMediaType(
		"attachment",
		map[string]string{
			"filename": "seiten-entfernt.pdf",
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

	s.pdfUploads.Delete(
		upload.ID,
	)

	if _, err := io.Copy(
		w,
		file,
	); err != nil {
		s.logError(
			r,
			"PDF delete response failed",
			err,
			"filename",
			upload.Filename,
			"output_size",
			info.Size(),
		)

		return
	}

	s.logInfo(
		"PDF delete",
		"deleted",
		len(request.Pages),
		"remaining",
		len(retainedPages),
		"input",
		upload.Size,
		"output",
		info.Size(),
		"duration",
		time.Since(started),
	)
}

func pagesAfterDeletion(
	deletedPages []int,
	pageCount int,
) ([]int, error) {
	if pageCount < 1 {
		return nil, fmt.Errorf(
			"invalid page count",
		)
	}

	if len(deletedPages) == 0 {
		return nil, fmt.Errorf(
			"no pages selected",
		)
	}

	if len(deletedPages) >= pageCount {
		return nil, fmt.Errorf(
			"all pages selected",
		)
	}

	deleted := make(
		[]bool,
		pageCount+1,
	)

	for _, page := range deletedPages {
		if page < 1 ||
			page > pageCount {
			return nil, fmt.Errorf(
				"page %d is outside range 1-%d",
				page,
				pageCount,
			)
		}

		if deleted[page] {
			return nil, fmt.Errorf(
				"page %d occurs more than once",
				page,
			)
		}

		deleted[page] = true
	}

	retained := make(
		[]int,
		0,
		pageCount-len(deletedPages),
	)

	for page := 1; page <= pageCount; page++ {
		if deleted[page] {
			continue
		}

		retained = append(
			retained,
			page,
		)
	}

	return retained, nil
}
