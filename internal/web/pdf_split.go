package web

import (
	"context"
	"encoding/json"
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
	UploadID string `json:"upload_id"`

	SplitAfter []int `json:"split_after"`
}

func (s *Server) handlePDFSplit(
	w http.ResponseWriter,
	r *http.Request,
) {
	started :=
		time.Now()

	r.Body =
		http.MaxBytesReader(
			w,
			r.Body,
			maxPDFSplitRequestSize,
		)

	defer r.Body.Close()

	var request pdfSplitRequest

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
			"PDF split rejected",
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
			"PDF split rejected",
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

	if len(request.SplitAfter) >=
		maxPDFSplitParts {
		writeAPIError(
			w,
			http.StatusBadRequest,
			apiErrorInvalidRequest,
			"Es können maximal 200 Teildokumente erzeugt werden.",
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
			"PDF split rejected",
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

	ranges, err :=
		buildPDFSplitRanges(
			upload.PageCount,
			request.SplitAfter,
		)

	if err != nil {
		s.logWarn(
			r,
			"PDF split rejected",
			"reason",
			err.Error(),
			"filename",
			upload.Filename,
			"pages",
			upload.PageCount,
			"split_points",
			len(request.SplitAfter),
		)

		writeAPIError(
			w,
			http.StatusBadRequest,
			apiErrorInvalidRequest,
			"Die Trennpunkte sind ungültig.",
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
			"PDF split queue failed",
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
			"media-converter-pdf-split-*",
		)

	if err != nil {
		s.logError(
			r,
			"PDF split temp directory failed",
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
				"PDF split cleanup failed",
				err,
				"directory",
				tempDir,
			)
		}
	}()

	ctx, cancel :=
		context.WithTimeout(
			r.Context(),
			pdfSplitTimeout,
		)

	defer cancel()

	files, err :=
		s.createPDFSplitParts(
			ctx,
			upload.Path,
			tempDir,
			ranges,
		)

	if err != nil {
		if writeTimeoutAPIError(
			w,
			ctx,
			"Das Trennen der PDF hat zu lange gedauert.",
		) {
			if ctx.Err() ==
				context.DeadlineExceeded {
				s.logError(
					r,
					"PDF split timed out",
					ctx.Err(),
					"filename",
					upload.Filename,
					"parts",
					len(ranges),
				)
			}

			return
		}

		s.logError(
			r,
			"PDF split part creation failed",
			err,
			"filename",
			upload.Filename,
			"part_count",
			len(ranges),
		)

		writeAPIError(
			w,
			http.StatusInternalServerError,
			apiErrorInternal,
			"Die PDF konnte nicht getrennt werden.",
		)

		return
	}

	archivePath :=
		filepath.Join(
			tempDir,
			"getrennte-pdfs.zip",
		)

	if err :=
		createPDFSplitArchive(
			archivePath,
			files,
			ranges,
		); err != nil {
		s.logError(
			r,
			"PDF split archive failed",
			err,
			"filename",
			upload.Filename,
			"parts",
			len(ranges),
		)

		writeAPIError(
			w,
			http.StatusInternalServerError,
			apiErrorInternal,
			"ZIP-Datei konnte nicht erstellt werden.",
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
			"PDF split output stat failed",
			err,
			"filename",
			upload.Filename,
		)

		writeAPIError(
			w,
			http.StatusInternalServerError,
			apiErrorInternal,
			"ZIP-Datei konnte nicht gelesen werden.",
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
		time.Since(
			started,
		),
	)
}
