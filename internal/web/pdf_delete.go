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
		s.logError(
			r,
			"failed to decode PDF delete request",
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

	upload, ok :=
		s.pdfUploads.Get(
			request.UploadID,
		)

	if !ok {
		s.logWarn(
			r,
			"PDF page deletion requested for unknown upload",
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

	retainedPages, err :=
		pagesAfterDeletion(
			request.Pages,
			upload.PageCount,
		)

	if err != nil {
		s.logWarn(
			r,
			"invalid PDF page deletion request",
			"upload_id",
			upload.ID,
			"filename",
			upload.Filename,
			"page_count",
			upload.PageCount,
			"delete_count",
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

	if err :=
		s.acquireConversionSlot(
			r.Context(),
		); err != nil {
		s.logError(
			r,
			"failed to acquire PDF delete conversion slot",
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
			"media-converter-pdf-delete-*",
		)

	if err != nil {
		s.logError(
			r,
			"failed to create PDF delete temporary directory",
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
				"failed to remove PDF delete temporary directory",
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
		s.logError(
			r,
			"PDF page deletion failed",
			err,
			"upload_id",
			upload.ID,
			"filename",
			upload.Filename,
			"size_bytes",
			upload.Size,
			"page_count",
			upload.PageCount,
			"deleted_pages",
			len(request.Pages),
		)

		if errors.Is(
			ctx.Err(),
			context.DeadlineExceeded,
		) {
			http.Error(
				w,
				"Das Erstellen der PDF hat zu lange gedauert.",
				http.StatusGatewayTimeout,
			)

			return
		}

		http.Error(
			w,
			"Die ausgewählten Seiten konnten nicht entfernt werden.",
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
			"failed to open PDF after page deletion",
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
			"failed to inspect PDF after page deletion",
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

	s.logger.Info(
		"PDF pages deleted",
		"method",
		r.Method,
		"path",
		r.URL.Path,
		"upload_id",
		upload.ID,
		"filename",
		upload.Filename,
		"original_page_count",
		upload.PageCount,
		"deleted_page_count",
		len(request.Pages),
		"result_page_count",
		len(retainedPages),
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
			"failed to send PDF after page deletion",
			err,
			"filename",
			upload.Filename,
			"output_size_bytes",
			info.Size(),
		)
	}
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

	if len(deletedPages) >=
		pageCount {
		return nil, fmt.Errorf(
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

		deleted[page] =
			true
	}

	retained :=
		make(
			[]int,
			0,
			pageCount-
				len(deletedPages),
		)

	for page := 1; page <= pageCount; page++ {
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
