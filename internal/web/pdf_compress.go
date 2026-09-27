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
	"time"
)

const (
	maxPDFCompressRequestSize int64 = 1 << 20

	pdfCompressTimeout = 30 * time.Minute
)

type pdfCompressRequest struct {
	UploadID string `json:"upload_id"`
	Mode     string `json:"mode"`
}

type pdfCompressionAnalysisResponse struct {
	OriginalSize int64 `json:"original_size"`
	ResultSize   int64 `json:"result_size"`
	SavingsBytes int64 `json:"savings_bytes"`

	SavingsPercent float64 `json:"savings_percent"`

	Unchanged bool `json:"unchanged"`
}

func (s *Server) handlePDFCompressionAnalyze(
	w http.ResponseWriter,
	r *http.Request,
) {
	request, upload, ok :=
		s.parsePDFCompressionRequest(
			w,
			r,
		)

	if !ok {
		return
	}

	ctx, cancel := context.WithTimeout(
		r.Context(),
		pdfCompressTimeout,
	)
	defer cancel()

	result, err :=
		s.ensurePDFCompressionResult(
			ctx,
			upload,
			request.Mode,
		)

	if err != nil {
		if errors.Is(
			ctx.Err(),
			context.Canceled,
		) {
			return
		}

		if errors.Is(
			ctx.Err(),
			context.DeadlineExceeded,
		) {
			http.Error(
				w,
				"Die Berechnung der Kompression hat zu lange gedauert.",
				http.StatusGatewayTimeout,
			)
			return
		}

		s.logError(
			r,
			"PDF compression analysis failed",
			err,
			"upload_id",
			upload.ID,
			"filename",
			upload.Filename,
			"mode",
			request.Mode,
		)

		http.Error(
			w,
			"Die mögliche Kompression konnte nicht berechnet werden.",
			http.StatusInternalServerError,
		)
		return
	}

	savings :=
		result.OriginalSize -
			result.ResultSize

	if savings < 0 {
		savings = 0
	}

	percent := 0.0

	if result.OriginalSize > 0 {
		percent =
			float64(savings) /
				float64(result.OriginalSize) *
				100
	}

	response :=
		pdfCompressionAnalysisResponse{
			OriginalSize: result.OriginalSize,

			ResultSize: result.ResultSize,

			SavingsBytes: savings,

			SavingsPercent: percent,

			Unchanged: result.Unchanged,
		}

	w.Header().Set(
		"Content-Type",
		"application/json; charset=utf-8",
	)

	w.Header().Set(
		"Cache-Control",
		"no-store",
	)

	if err := json.NewEncoder(
		w,
	).Encode(
		response,
	); err != nil {
		s.logError(
			r,
			"failed to encode PDF compression analysis",
			err,
			"upload_id",
			upload.ID,
		)
		return
	}

	s.logger.Info(
		"PDF compression analyzed",
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
		result.OriginalSize,
		"output_size_bytes",
		result.ResultSize,
		"savings_bytes",
		savings,
		"savings_percent",
		percent,
		"unchanged",
		result.Unchanged,
	)
}

func (s *Server) handlePDFCompress(
	w http.ResponseWriter,
	r *http.Request,
) {
	request, upload, ok :=
		s.parsePDFCompressionRequest(
			w,
			r,
		)

	if !ok {
		return
	}

	ctx, cancel := context.WithTimeout(
		r.Context(),
		pdfCompressTimeout,
	)
	defer cancel()

	result, err :=
		s.ensurePDFCompressionResult(
			ctx,
			upload,
			request.Mode,
		)

	if err != nil {
		s.handlePDFCompressionError(
			w,
			r,
			ctx,
			upload,
			request.Mode,
			err,
		)
		return
	}

	file, err := os.Open(
		result.Path,
	)
	if err != nil {
		s.logError(
			r,
			"failed to open PDF compression result",
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

	disposition :=
		mime.FormatMediaType(
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
			result.ResultSize,
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
			result.OriginalSize,
		),
	)

	w.Header().Set(
		"X-Result-Size",
		fmt.Sprintf(
			"%d",
			result.ResultSize,
		),
	)

	if result.Unchanged {
		w.Header().Set(
			"X-Compression-Unchanged",
			"true",
		)
	}

	s.logger.Info(
		"PDF compression download started",
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
		result.OriginalSize,
		"output_size_bytes",
		result.ResultSize,
		"unchanged",
		result.Unchanged,
	)

	if _, err := io.Copy(
		w,
		file,
	); err != nil {
		s.logError(
			r,
			"failed to send compressed PDF",
			err,
			"filename",
			upload.Filename,
			"output_size_bytes",
			result.ResultSize,
		)
		return
	}

	s.pdfUploads.Delete(
		upload.ID,
	)
}

func (s *Server) parsePDFCompressionRequest(
	w http.ResponseWriter,
	r *http.Request,
) (pdfCompressRequest, storedPDFUpload, bool) {
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
		http.Error(
			w,
			"Ungültige Anfrage.",
			http.StatusBadRequest,
		)

		return request,
			storedPDFUpload{},
			false
	}

	if request.UploadID == "" {
		http.Error(
			w,
			"Upload-ID fehlt.",
			http.StatusBadRequest,
		)

		return request,
			storedPDFUpload{},
			false
	}

	if !validPDFCompressionMode(
		request.Mode,
	) {
		http.Error(
			w,
			"Ungültiger Kompressionsmodus.",
			http.StatusBadRequest,
		)

		return request,
			storedPDFUpload{},
			false
	}

	upload, ok :=
		s.pdfUploads.Get(
			request.UploadID,
		)

	if !ok {
		http.Error(
			w,
			"Die PDF ist nicht mehr verfügbar. Bitte erneut hochladen.",
			http.StatusGone,
		)

		return request,
			storedPDFUpload{},
			false
	}

	return request,
		upload,
		true
}

func (s *Server) handlePDFCompressionError(
	w http.ResponseWriter,
	r *http.Request,
	ctx context.Context,
	upload storedPDFUpload,
	mode string,
	err error,
) {
	if compressionWasCancelled(
		err,
		ctx,
	) {
		if errors.Is(
			ctx.Err(),
			context.DeadlineExceeded,
		) {
			http.Error(
				w,
				"Die PDF-Komprimierung hat zu lange gedauert.",
				http.StatusGatewayTimeout,
			)
		}

		return
	}

	s.logError(
		r,
		"PDF compression failed",
		err,
		"upload_id",
		upload.ID,
		"filename",
		upload.Filename,
		"mode",
		mode,
		"size_bytes",
		upload.Size,
	)

	http.Error(
		w,
		"Die PDF konnte nicht komprimiert werden.",
		http.StatusInternalServerError,
	)
}
