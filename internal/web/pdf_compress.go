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

	"github.com/jayzone91/media-converter/internal/converter"
)

const (
	maxPDFCompressRequestSize int64 = 1 << 20

	pdfCompressTimeout = 30 * time.Minute
)

type pdfCompressRequest struct {
	UploadID string `json:"upload_id"`
	Mode     string `json:"mode"`
}

func (s *Server) handlePDFCompress(
	w http.ResponseWriter,
	r *http.Request,
) {
	r.Body = http.MaxBytesReader(
		w,
		r.Body,
		maxPDFCompressRequestSize,
	)
	defer r.Body.Close()

	var request pdfCompressRequest

	decoder := json.NewDecoder(
		r.Body,
	)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(
		&request,
	); err != nil {
		s.logError(
			r,
			"failed to decode PDF compression request",
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

	if !validPDFCompressionMode(
		request.Mode,
	) {
		http.Error(
			w,
			"Ungültiger Kompressionsmodus.",
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
			"PDF compression requested for unknown upload",
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

	if err := s.acquireConversionSlot(
		r.Context(),
	); err != nil {
		s.logError(
			r,
			"failed to acquire PDF compression slot",
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
		"media-converter-pdf-compress-*",
	)
	if err != nil {
		s.logError(
			r,
			"failed to create PDF compression temporary directory",
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
				"failed to remove PDF compression temporary directory",
				err,
				"directory",
				tempDir,
			)
		}
	}()

	outputPath := filepath.Join(
		tempDir,
		"komprimiert.pdf",
	)

	ctx, cancel := context.WithTimeout(
		r.Context(),
		pdfCompressTimeout,
	)
	defer cancel()

	if err := s.compressPDF(
		ctx,
		upload.Path,
		outputPath,
		request.Mode,
	); err != nil {
		s.logError(
			r,
			"PDF compression failed",
			err,
			"upload_id",
			upload.ID,
			"filename",
			upload.Filename,
			"mode",
			request.Mode,
			"size_bytes",
			upload.Size,
		)

		if errors.Is(
			ctx.Err(),
			context.DeadlineExceeded,
		) {
			http.Error(
				w,
				"Die PDF-Komprimierung hat zu lange gedauert.",
				http.StatusGatewayTimeout,
			)
			return
		}

		http.Error(
			w,
			"Die PDF konnte nicht komprimiert werden.",
			http.StatusInternalServerError,
		)
		return
	}

	outputInfo, err := os.Stat(
		outputPath,
	)
	if err != nil {
		s.logError(
			r,
			"failed to inspect compressed PDF",
			err,
			"upload_id",
			upload.ID,
		)

		http.Error(
			w,
			"Die komprimierte PDF konnte nicht gelesen werden.",
			http.StatusInternalServerError,
		)
		return
	}

	resultPath := outputPath
	resultSize := outputInfo.Size()
	usedOriginal := false

	if resultSize >= upload.Size {
		resultPath = upload.Path
		resultSize = upload.Size
		usedOriginal = true
	}

	file, err := os.Open(
		resultPath,
	)
	if err != nil {
		s.logError(
			r,
			"failed to open compressed PDF result",
			err,
			"upload_id",
			upload.ID,
		)

		http.Error(
			w,
			"Die erzeugte PDF konnte nicht geöffnet werden.",
			http.StatusInternalServerError,
		)
		return
	}
	defer file.Close()

	disposition := mime.FormatMediaType(
		"attachment",
		map[string]string{
			"filename": "komprimiert.pdf",
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
			resultSize,
		),
	)

	w.Header().Set(
		"Cache-Control",
		"no-store",
	)

	w.Header().Set(
		"X-Original-Size",
		fmt.Sprintf(
			"%d",
			upload.Size,
		),
	)

	w.Header().Set(
		"X-Result-Size",
		fmt.Sprintf(
			"%d",
			resultSize,
		),
	)

	if usedOriginal {
		w.Header().Set(
			"X-Compression-Unchanged",
			"true",
		)
	}

	s.logger.Info(
		"PDF compression completed",
		"method",
		r.Method,
		"path",
		r.URL.Path,
		"upload_id",
		upload.ID,
		"filename",
		upload.Filename,
		"mode",
		request.Mode,
		"input_size_bytes",
		upload.Size,
		"output_size_bytes",
		resultSize,
		"used_original",
		usedOriginal,
	)

	_, copyErr := io.Copy(
		w,
		file,
	)

	if copyErr != nil {
		s.logError(
			r,
			"failed to send compressed PDF",
			copyErr,
			"filename",
			upload.Filename,
			"output_size_bytes",
			resultSize,
		)

		return
	}

	s.pdfUploads.Delete(
		upload.ID,
	)
}

func (s *Server) compressPDF(
	ctx context.Context,
	input string,
	output string,
	mode string,
) error {
	switch mode {
	case "lossless":
		return s.qpdf.Optimize(
			ctx,
			input,
			output,
		)

	case "balanced":
		return s.ghostscript.CompressPDF(
			ctx,
			input,
			output,
			converter.PDFCompressionBalanced,
		)

	case "strong":
		return s.ghostscript.CompressPDF(
			ctx,
			input,
			output,
			converter.PDFCompressionStrong,
		)

	default:
		return fmt.Errorf(
			"unsupported PDF compression mode: %s",
			mode,
		)
	}
}

func validPDFCompressionMode(
	mode string,
) bool {
	switch mode {
	case
		"lossless",
		"balanced",
		"strong":

		return true

	default:
		return false
	}
}
