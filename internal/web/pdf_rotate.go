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

	"github.com/jayzone91/media-converter/internal/converter"
)

const (
	maxPDFRotateRequestSize int64 = 4 << 20

	pdfRotateTimeout = 30 * time.Minute
)

type pdfRotateRequest struct {
	UploadID  string                   `json:"upload_id"`
	Rotations []pdfPageRotationRequest `json:"rotations"`
}

type pdfPageRotationRequest struct {
	Page  int `json:"page"`
	Angle int `json:"angle"`
}

func (s *Server) handlePDFRotatePages(
	w http.ResponseWriter,
	r *http.Request,
) {
	started := time.Now()

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

	rotations, err := validatePDFRotations(
		request.Rotations,
		upload.PageCount,
	)

	if err != nil {
		s.logWarn(
			r,
			"invalid PDF page rotation request",
			"upload_id",
			upload.ID,
			"filename",
			upload.Filename,
			"page_count",
			upload.PageCount,
			"rotation_count",
			len(request.Rotations),
			"reason",
			err.Error(),
		)

		http.Error(
			w,
			"Die Seitendrehungen sind ungültig.",
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

	if err := s.qpdf.RotatePages(
		ctx,
		upload.Path,
		rotations,
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
			"rotation_count",
			len(rotations),
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
			"Die Seiten konnten nicht gedreht werden.",
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

	if !s.prepareDownloadResponse(
		w,
		r,
		outputPath,
		"gedreht.pdf",
	) {
		return
	}

	s.pdfUploads.Delete(
		upload.ID,
	)

	s.logInfo(
		"PDF pages rotated",
		"pages",
		upload.PageCount,
		"rotated",
		len(rotations),
		"input",
		upload.Size,
		"output",
		outputSize,
		"duration",
		time.Since(started),
	)
}

func validatePDFRotations(
	requests []pdfPageRotationRequest,
	pageCount int,
) ([]converter.PDFPageRotation, error) {
	if pageCount < 1 {
		return nil, fmt.Errorf(
			"invalid page count",
		)
	}

	if len(requests) == 0 {
		return nil, fmt.Errorf(
			"no rotations supplied",
		)
	}

	if len(requests) > pageCount {
		return nil, fmt.Errorf(
			"too many rotations supplied",
		)
	}

	seen := make(
		map[int]struct{},
		len(requests),
	)

	rotations := make(
		[]converter.PDFPageRotation,
		0,
		len(requests),
	)

	for _, request := range requests {
		if request.Page < 1 ||
			request.Page > pageCount {
			return nil, fmt.Errorf(
				"page %d is outside range 1-%d",
				request.Page,
				pageCount,
			)
		}

		if _, exists := seen[request.Page]; exists {
			return nil, fmt.Errorf(
				"page %d occurs more than once",
				request.Page,
			)
		}

		seen[request.Page] = struct{}{}

		switch request.Angle {
		case 90, 180, 270:
		default:
			return nil, fmt.Errorf(
				"invalid angle %d for page %d",
				request.Angle,
				request.Page,
			)
		}

		rotations = append(
			rotations,
			converter.PDFPageRotation{
				Page:  request.Page,
				Angle: request.Angle,
			},
		)
	}

	return rotations, nil
}
