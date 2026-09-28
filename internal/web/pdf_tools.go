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
		http.Error(
			w,
			"Ungültige Merge-Anfrage.",
			http.StatusBadRequest,
		)

		return
	}

	if len(request.IDs) < 2 {
		http.Error(
			w,
			"Bitte mindestens zwei PDF-Dateien auswählen.",
			http.StatusBadRequest,
		)

		return
	}

	if len(request.IDs) >
		maxPDFMergeFiles {
		http.Error(
			w,
			fmt.Sprintf(
				"Es können maximal %d PDFs gleichzeitig zusammengefügt werden.",
				maxPDFMergeFiles,
			),
			http.StatusBadRequest,
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
			http.Error(
				w,
				"Ungültige PDF-ID.",
				http.StatusBadRequest,
			)

			return
		}

		if _, exists := seen[id]; exists {
			http.Error(
				w,
				"Eine PDF wurde mehrfach angegeben.",
				http.StatusBadRequest,
			)

			return
		}

		seen[id] = struct{}{}

		upload, ok :=
			s.pdfUploads.Get(
				id,
			)

		if !ok {
			http.Error(
				w,
				"Eine PDF ist nicht mehr verfügbar. Bitte erneut hochladen.",
				http.StatusGone,
			)

			return
		}

		totalSize += upload.Size

		if totalSize >
			maxPDFMergeTotalSize {
			http.Error(
				w,
				"Die PDFs dürfen zusammen maximal 1 GiB groß sein.",
				http.StatusRequestEntityTooLarge,
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
		http.Error(
			w,
			"Der Server ist momentan ausgelastet. Bitte später erneut versuchen.",
			http.StatusServiceUnavailable,
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
		if errors.Is(
			ctx.Err(),
			context.DeadlineExceeded,
		) {
			http.Error(
				w,
				"Das Zusammenfügen der PDFs hat zu lange gedauert.",
				http.StatusGatewayTimeout,
			)

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

		http.Error(
			w,
			"Die PDF-Dateien konnten nicht zusammengefügt werden.",
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
			"PDF merge output stat failed",
			err,
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
