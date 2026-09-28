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
	maxPDFMergeFiles = 50

	maxPDFMergeTotalSize int64 = 1024 << 20

	maxPDFMergeRequestSize int64 = 64 << 10

	pdfMergeTimeout = 30 * time.Minute
)

type pdfMergeRequest struct {
	IDs []string `json:"ids"`
}

func (s *Server) handlePDFMerge(
	w http.ResponseWriter,
	r *http.Request,
) {
	started := time.Now()

	r.Body = http.MaxBytesReader(
		w,
		r.Body,
		maxPDFMergeRequestSize,
	)

	defer r.Body.Close()

	var request pdfMergeRequest

	decoder := json.NewDecoder(
		r.Body,
	)

	decoder.DisallowUnknownFields()

	if err := decoder.Decode(
		&request,
	); err != nil {
		writeAPIError(
			w,
			http.StatusBadRequest,
			apiErrorInvalidRequest,
			"Ungültige Merge-Anfrage.",
		)

		return
	}

	if len(request.IDs) < 2 {
		writeAPIError(
			w,
			http.StatusBadRequest,
			apiErrorInvalidRequest,
			"Bitte mindestens zwei PDF-Dateien auswählen.",
		)

		return
	}

	if len(request.IDs) >
		maxPDFMergeFiles {
		writeAPIError(
			w,
			http.StatusBadRequest,
			apiErrorInvalidRequest,
			fmt.Sprintf(
				"Es können maximal %d PDFs gleichzeitig zusammengefügt werden.",
				maxPDFMergeFiles,
			),
		)

		return
	}

	inputPaths := make(
		[]string,
		0,
		len(request.IDs),
	)

	seen := make(
		map[string]struct{},
		len(request.IDs),
	)

	var totalSize int64

	for _, id := range request.IDs {
		if id == "" {
			writeAPIError(
				w,
				http.StatusBadRequest,
				apiErrorInvalidRequest,
				"Ungültige PDF-ID.",
			)

			return
		}

		if _, exists := seen[id]; exists {
			writeAPIError(
				w,
				http.StatusBadRequest,
				apiErrorInvalidRequest,
				"Eine PDF wurde mehrfach angegeben.",
			)

			return
		}

		seen[id] = struct{}{}

		upload, ok :=
			s.pdfUploads.Get(
				id,
			)

		if !ok {
			writeAPIError(
				w,
				http.StatusGone,
				apiErrorUploadExpired,
				"Eine PDF ist nicht mehr verfügbar. Bitte erneut hochladen.",
			)

			return
		}

		totalSize += upload.Size

		if totalSize >
			maxPDFMergeTotalSize {
			writeAPIError(
				w,
				http.StatusRequestEntityTooLarge,
				apiErrorInvalidRequest,
				"Die PDFs dürfen zusammen maximal 1 GiB groß sein.",
			)

			return
		}

		inputPaths = append(
			inputPaths,
			upload.Path,
		)
	}

	if err := s.acquireWorkload(
		r.Context(),
		workloadQPDF,
	); err != nil {
		s.logError(
			r,
			"PDF merge queue failed",
			err,
			"files",
			len(request.IDs),
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
		"media-converter-pdf-merge-*",
	)

	if err != nil {
		s.logError(
			r,
			"PDF merge temp directory failed",
			err,
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
				"PDF merge cleanup failed",
				err,
				"directory",
				tempDir,
			)
		}
	}()

	ctx, cancel := context.WithTimeout(
		r.Context(),
		pdfMergeTimeout,
	)

	defer cancel()

	outputPath := filepath.Join(
		tempDir,
		"zusammengefuegt.pdf",
	)

	if err := s.qpdf.Merge(
		ctx,
		inputPaths,
		outputPath,
	); err != nil {
		if writeTimeoutAPIError(
			w,
			ctx,
			"Das Zusammenfügen der PDFs hat zu lange gedauert.",
		) {
			if ctx.Err() ==
				context.DeadlineExceeded {
				s.logError(
					r,
					"PDF merge timed out",
					ctx.Err(),
					"files",
					len(request.IDs),
					"input",
					totalSize,
				)
			}

			return
		}

		s.logError(
			r,
			"PDF merge failed",
			err,
			"files",
			len(request.IDs),
			"input_size",
			totalSize,
		)

		writeAPIError(
			w,
			http.StatusInternalServerError,
			apiErrorInternal,
			"Die PDF-Dateien konnten nicht zusammengefügt werden.",
		)

		return
	}

	outputSize, err := downloadFileSize(
		outputPath,
	)

	if err != nil {
		s.logError(
			r,
			"PDF merge output stat failed",
			err,
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
		"zusammengefuegt.pdf",
	) {
		return
	}

	for _, id := range request.IDs {
		s.pdfUploads.Delete(
			id,
		)
	}

	s.logInfo(
		"PDF merge",
		"files",
		len(request.IDs),
		"input",
		totalSize,
		"output",
		outputSize,
		"duration",
		time.Since(started),
	)
}
