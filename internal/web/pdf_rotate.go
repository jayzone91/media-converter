package web

import (
	"context"
	"encoding/json"
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
		s.logWarn(
			r,
			"PDF rotate rejected",
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
			"PDF rotate rejected",
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
			"PDF rotate rejected",
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

	rotations, err := validatePDFRotations(
		request.Rotations,
		upload.PageCount,
	)

	if err != nil {
		s.logWarn(
			r,
			"PDF rotate rejected",
			"reason",
			err.Error(),
			"filename",
			upload.Filename,
			"pages",
			upload.PageCount,
			"rotations",
			len(request.Rotations),
		)

		writeAPIError(
			w,
			http.StatusBadRequest,
			apiErrorInvalidRequest,
			"Die Seitendrehungen sind ungültig.",
		)

		return
	}

	if err := s.acquireWorkload(
		r.Context(),
		workloadQPDF,
	); err != nil {
		s.logError(
			r,
			"PDF rotate queue failed",
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
		"media-converter-pdf-rotate-*",
	)

	if err != nil {
		s.logError(
			r,
			"PDF rotate temp directory failed",
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
				"PDF rotate cleanup failed",
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
		if writeTimeoutAPIError(
			w,
			ctx,
			"Das Drehen der PDF-Seiten hat zu lange gedauert.",
		) {
			if ctx.Err() ==
				context.DeadlineExceeded {
				s.logError(
					r,
					"PDF rotate timed out",
					ctx.Err(),
					"filename",
					upload.Filename,
					"pages",
					upload.PageCount,
					"rotations",
					len(rotations),
				)
			}

			return
		}

		s.logError(
			r,
			"PDF rotate failed",
			err,
			"filename",
			upload.Filename,
			"size",
			upload.Size,
			"pages",
			upload.PageCount,
			"rotations",
			len(rotations),
		)

		writeAPIError(
			w,
			http.StatusInternalServerError,
			apiErrorInternal,
			"Die Seiten konnten nicht gedreht werden.",
		)

		return
	}

	outputSize, err := downloadFileSize(
		outputPath,
	)

	if err != nil {
		s.logError(
			r,
			"PDF rotate output stat failed",
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
		return nil,
			fmt.Errorf(
				"invalid page count",
			)
	}

	if len(requests) == 0 {
		return nil,
			fmt.Errorf(
				"no rotations supplied",
			)
	}

	if len(requests) > pageCount {
		return nil,
			fmt.Errorf(
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
			return nil,
				fmt.Errorf(
					"page %d is outside range 1-%d",
					request.Page,
					pageCount,
				)
		}

		if _, exists := seen[request.Page]; exists {
			return nil,
				fmt.Errorf(
					"page %d occurs more than once",
					request.Page,
				)
		}

		seen[request.Page] = struct{}{}

		switch request.Angle {
		case 90, 180, 270:

		default:
			return nil,
				fmt.Errorf(
					"invalid angle %d for page %d",
					request.Angle,
					request.Page,
				)
		}

		rotations = append(
			rotations,
			converter.PDFPageRotation{
				Page: request.Page,

				Angle: request.Angle,
			},
		)
	}

	return rotations, nil
}
