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
	maxPDFRotateRequestSize int64 = 4 << 20

	pdfRotateTimeout = 30 * time.Minute
)

type pdfRotateRequest struct {
	UploadID string `json:"upload_id"`
	Pages    []int  `json:"pages"`
	Angle    int    `json:"angle"`
}

func (s *Server) handlePDFRotatePages(
	w http.ResponseWriter,
	r *http.Request,
) {
	r.Body = http.MaxBytesReader(
		w,
		r.Body,
		maxPDFRotateRequestSize,
	)
	defer r.Body.Close()

	var request pdfRotateRequest

	decoder := json.NewDecoder(
		r.Body,
	)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(
		&request,
	); err != nil {
		s.logError(
			r,
			"failed to decode PDF rotate request",
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

	if err := validatePDFRotationAngle(
		request.Angle,
	); err != nil {
		s.logWarn(
			r,
			"invalid PDF rotation angle",
			"angle",
			request.Angle,
		)

		http.Error(
			w,
			"Ungültiger Drehwinkel.",
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
			"PDF page rotation requested for unknown upload",
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
			"invalid PDF page rotation request",
			"upload_id",
			upload.ID,
			"filename",
			upload.Filename,
			"page_count",
			upload.PageCount,
			"selected_pages",
			len(request.Pages),
			"angle",
			request.Angle,
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

	if err := s.acquireConversionSlot(
		r.Context(),
	); err != nil {
		s.logError(
			r,
			"failed to acquire PDF rotate conversion slot",
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

	tempDir, err := os.MkdirTemp(
		"",
		"media-converter-pdf-rotate-*",
	)
	if err != nil {
		s.logError(
			r,
			"failed to create PDF rotate temporary directory",
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
				"failed to remove PDF rotate temporary directory",
				err,
				"directory",
				tempDir,
			)
		}
	}()

	outputPath := filepath.Join(
		tempDir,
		"gedreht.pdf",
	)

	ctx, cancel := context.WithTimeout(
		r.Context(),
		pdfRotateTimeout,
	)
	defer cancel()

	if err := s.qpdf.Rotate(
		ctx,
		upload.Path,
		request.Pages,
		request.Angle,
		outputPath,
	); err != nil {
		s.logError(
			r,
			"PDF page rotation failed",
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
			"angle",
			request.Angle,
		)

		if errors.Is(
			ctx.Err(),
			context.DeadlineExceeded,
		) {
			http.Error(
				w,
				"Das Drehen der PDF-Seiten hat zu lange gedauert.",
				http.StatusGatewayTimeout,
			)
			return
		}

		http.Error(
			w,
			"Die ausgewählten Seiten konnten nicht gedreht werden.",
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
			"failed to open rotated PDF",
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

	info, err := file.Stat()
	if err != nil {
		s.logError(
			r,
			"failed to inspect rotated PDF",
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

	disposition := mime.FormatMediaType(
		"attachment",
		map[string]string{
			"filename": "gedreht.pdf",
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
		"PDF pages rotated",
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
		"rotated_page_count",
		len(request.Pages),
		"angle",
		request.Angle,
		"input_size_bytes",
		upload.Size,
		"output_size_bytes",
		info.Size(),
	)

	if _, err := io.Copy(
		w,
		file,
	); err != nil {
		s.logError(
			r,
			"failed to send rotated PDF",
			err,
			"filename",
			upload.Filename,
			"output_size_bytes",
			info.Size(),
		)
	}
}

func validatePDFRotationAngle(
	angle int,
) error {
	switch angle {
	case 90, 180, 270:
		return nil

	default:
		return fmt.Errorf(
			"unsupported rotation angle: %d",
			angle,
		)
	}
}
