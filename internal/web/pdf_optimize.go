package web

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"
)

const (
	maxPDFOptimizeRequestSize int64 = 1 << 20

	pdfOptimizeTimeout = 30 * time.Minute
)

type pdfOptimizeRequest struct {
	UploadID string `json:"upload_id"`

	Linearize bool `json:"linearize"`
}

type pdfOptimizeAnalysisResponse struct {
	OriginalSize int64 `json:"original_size"`

	ResultSize int64 `json:"result_size"`

	DeltaBytes int64 `json:"delta_bytes"`

	DeltaPercent float64 `json:"delta_percent"`

	Linearized bool `json:"linearized"`

	Unchanged bool `json:"unchanged"`
}

func (s *Server) handlePDFOptimizeAnalyze(
	w http.ResponseWriter,
	r *http.Request,
) {
	request, upload, ok :=
		s.readPDFOptimizeRequest(
			w,
			r,
		)

	if !ok {
		return
	}

	ctx, cancel :=
		context.WithTimeout(
			r.Context(),
			pdfOptimizeTimeout,
		)

	defer cancel()

	result, err :=
		s.ensurePDFOptimizationResult(
			ctx,
			upload,
			request.Linearize,
		)

	if err != nil {
		s.handlePDFOptimizeError(
			w,
			r,
			ctx,
			upload,
			err,
		)

		return
	}

	delta :=
		result.ResultSize -
			result.OriginalSize

	percent :=
		0.0

	if result.OriginalSize > 0 {
		percent =
			float64(
				delta,
			) /
				float64(
					result.OriginalSize,
				) *
				100
	}

	response :=
		pdfOptimizeAnalysisResponse{
			OriginalSize: result.OriginalSize,

			ResultSize: result.ResultSize,

			DeltaBytes: delta,

			DeltaPercent: percent,

			Linearized: result.Linearized,

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

	if err :=
		json.NewEncoder(
			w,
		).Encode(
			response,
		); err != nil {
		s.logError(
			r,
			"failed to encode PDF optimization analysis",
			err,
			"upload_id",
			upload.ID,
		)
	}
}

func (s *Server) handlePDFOptimize(
	w http.ResponseWriter,
	r *http.Request,
) {
	request, upload, ok :=
		s.readPDFOptimizeRequest(
			w,
			r,
		)

	if !ok {
		return
	}

	ctx, cancel :=
		context.WithTimeout(
			r.Context(),
			pdfOptimizeTimeout,
		)

	defer cancel()

	result, err :=
		s.ensurePDFOptimizationResult(
			ctx,
			upload,
			request.Linearize,
		)

	if err != nil {
		s.handlePDFOptimizeError(
			w,
			r,
			ctx,
			upload,
			err,
		)

		return
	}

	filename :=
		"optimiert.pdf"

	if result.Linearized {
		filename =
			"optimiert-web.pdf"
	}

	if !s.prepareDownloadResponse(
		w,
		r,
		result.Path,
		filename,
	) {
		return
	}

	s.pdfUploads.Delete(
		upload.ID,
	)

	s.logInfo(
		"PDF optimized",
		"linearized",
		result.Linearized,
		"input",
		result.OriginalSize,
		"output",
		result.ResultSize,
		"unchanged",
		result.Unchanged,
	)
}

func (s *Server) readPDFOptimizeRequest(
	w http.ResponseWriter,
	r *http.Request,
) (
	pdfOptimizeRequest,
	storedPDFUpload,
	bool,
) {
	r.Body =
		http.MaxBytesReader(
			w,
			r.Body,
			maxPDFOptimizeRequestSize,
		)

	defer r.Body.Close()

	var request pdfOptimizeRequest

	decoder :=
		json.NewDecoder(
			r.Body,
		)

	decoder.DisallowUnknownFields()

	if err :=
		decoder.Decode(
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

func (s *Server) handlePDFOptimizeError(
	w http.ResponseWriter,
	r *http.Request,
	ctx context.Context,
	upload storedPDFUpload,
	err error,
) bool {
	if errors.Is(
		err,
		errPDFOptimizationSignedDocument,
	) {
		s.logWarn(
			r,
			"PDF optimization rejected for signed document",
			"upload_id",
			upload.ID,
			"filename",
			upload.Filename,
		)

		http.Error(
			w,
			"Diese PDF enthält eine digitale Signatur. Eine Optimierung würde die Signatur ungültig machen.",
			http.StatusUnprocessableEntity,
		)

		return true
	}

	if errors.Is(
		ctx.Err(),
		context.Canceled,
	) {
		return true
	}

	if errors.Is(
		ctx.Err(),
		context.DeadlineExceeded,
	) {
		http.Error(
			w,
			"Die PDF-Optimierung hat zu lange gedauert.",
			http.StatusGatewayTimeout,
		)

		return true
	}

	s.logError(
		r,
		"PDF optimization failed",
		err,
		"upload_id",
		upload.ID,
		"filename",
		upload.Filename,
	)

	http.Error(
		w,
		"Die PDF konnte nicht optimiert werden.",
		http.StatusInternalServerError,
	)

	return true
}
