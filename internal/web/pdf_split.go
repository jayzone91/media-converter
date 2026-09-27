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
	maxPDFSplitRequestSize int64 = 4 << 20

	maxPDFSplitParts = 200

	pdfSplitTimeout = 30 * time.Minute
)

type pdfSplitRequest struct {
	UploadID   string `json:"upload_id"`
	SplitAfter []int  `json:"split_after"`
}

func (s *Server) handlePDFSplit(
	w http.ResponseWriter,
	r *http.Request,
) {
	started := time.Now()

	r.Body = http.MaxBytesReader(
		w,
		r.Body,
		maxPDFSplitRequestSize,
	)
	defer r.Body.Close()

	var request pdfSplitRequest

	decoder := json.NewDecoder(
		r.Body,
	)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(
		&request,
	); err != nil {
		s.logError(
			r,
			"failed to decode PDF split request",
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

	if len(request.SplitAfter) >=
		maxPDFSplitParts {
		http.Error(
			w,
			"Es können maximal 200 Teildokumente erzeugt werden.",
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
			"PDF split requested for unknown upload",
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

	ranges, err := buildPDFSplitRanges(
		upload.PageCount,
		request.SplitAfter,
	)

	if err != nil {
		s.logWarn(
			r,
			"invalid PDF split request",
			"upload_id",
			upload.ID,
			"filename",
			upload.Filename,
			"page_count",
			upload.PageCount,
			"split_points",
			len(request.SplitAfter),
			"reason",
			err.Error(),
		)

		http.Error(
			w,
			"Die Trennpunkte sind ungültig.",
			http.StatusBadRequest,
		)

		return
	}

	if err := s.acquireConversionSlot(
		r.Context(),
	); err != nil {
		s.logError(
			r,
			"failed to acquire PDF split conversion slot",
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
		"media-converter-pdf-split-*",
	)

	if err != nil {
		s.logError(
			r,
			"failed to create PDF split temporary directory",
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
				"failed to remove PDF split temporary directory",
				err,
				"directory",
				tempDir,
			)
		}
	}()

	ctx, cancel := context.WithTimeout(
		r.Context(),
		pdfSplitTimeout,
	)
	defer cancel()

	files := make(
		[]string,
		0,
		len(ranges),
	)

	for index, pageRange := range ranges {
		outputPath := filepath.Join(
			tempDir,
			fmt.Sprintf(
				"part-%03d.pdf",
				index+1,
			),
		)

		if err := s.qpdf.ExtractRange(
			ctx,
			upload.Path,
			pageRange.Start,
			pageRange.End,
			outputPath,
		); err != nil {
			s.logError(
				r,
				"PDF split part creation failed",
				err,
				"upload_id",
				upload.ID,
				"filename",
				upload.Filename,
				"part",
				index+1,
				"start_page",
				pageRange.Start,
				"end_page",
				pageRange.End,
			)

			if errors.Is(
				ctx.Err(),
				context.DeadlineExceeded,
			) {
				http.Error(
					w,
					"Das Trennen der PDF hat zu lange gedauert.",
					http.StatusGatewayTimeout,
				)

				return
			}

			http.Error(
				w,
				"Die PDF konnte nicht getrennt werden.",
				http.StatusInternalServerError,
			)

			return
		}

		files = append(
			files,
			outputPath,
		)
	}

	archivePath := filepath.Join(
		tempDir,
		"getrennte-pdfs.zip",
	)

	if err := createPDFSplitArchive(
		archivePath,
		files,
		ranges,
	); err != nil {
		s.logError(
			r,
			"failed to create PDF split archive",
			err,
			"upload_id",
			upload.ID,
			"filename",
			upload.Filename,
			"part_count",
			len(ranges),
		)

		http.Error(
			w,
			"ZIP-Datei konnte nicht erstellt werden.",
			http.StatusInternalServerError,
		)

		return
	}

	outputSize, err :=
		downloadFileSize(
			archivePath,
		)

	if err != nil {
		s.logError(
			r,
			"failed to inspect PDF split archive",
			err,
		)

		http.Error(
			w,
			"ZIP-Datei konnte nicht gelesen werden.",
			http.StatusInternalServerError,
		)

		return
	}

	if !s.prepareDownloadResponse(
		w,
		r,
		archivePath,
		"getrennte-pdfs.zip",
	) {
		return
	}

	s.pdfUploads.Delete(
		upload.ID,
	)

	s.logInfo(
		"PDF split",
		"pages",
		upload.PageCount,
		"parts",
		len(ranges),
		"input",
		upload.Size,
		"output",
		outputSize,
		"duration",
		time.Since(started),
	)
}
