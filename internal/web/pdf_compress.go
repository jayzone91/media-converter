package web

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
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

	ctx, cancel :=
		context.WithTimeout(
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
		if s.handlePDFCompressionSpecialError(
			w,
			r,
			ctx,
			upload,
			request.Mode,
			err,
		) {
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

		writeAPIError(
			w,
			http.StatusInternalServerError,
			apiErrorInternal,
			"Die mögliche Kompression konnte nicht berechnet werden.",
		)

		return
	}

	savings :=
		result.OriginalSize -
			result.ResultSize

	if savings < 0 {
		savings = 0
	}

	percent :=
		0.0

	if result.OriginalSize > 0 {
		percent =
			float64(
				savings,
			) /
				float64(
					result.OriginalSize,
				) *
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

	if err :=
		json.NewEncoder(
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

	s.logInfo(
		"PDF compression analyzed",
		"mode",
		request.Mode,
		"input",
		result.OriginalSize,
		"output",
		result.ResultSize,
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

	ctx, cancel :=
		context.WithTimeout(
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

	if !s.prepareDownloadResponse(
		w,
		r,
		result.Path,
		"komprimiert.pdf",
	) {
		return
	}

	s.pdfUploads.Delete(
		upload.ID,
	)

	s.logInfo(
		"PDF compression",
		"mode",
		request.Mode,
		"input",
		result.OriginalSize,
		"output",
		result.ResultSize,
		"unchanged",
		result.Unchanged,
	)
}

func (s *Server) parsePDFCompressionRequest(
	w http.ResponseWriter,
	r *http.Request,
) (
	pdfCompressRequest,
	storedPDFUpload,
	bool,
) {
	r.Body =
		http.MaxBytesReader(
			w,
			r.Body,
			maxPDFCompressRequestSize,
		)

	defer r.Body.Close()

	var request pdfCompressRequest

	decoder :=
		json.NewDecoder(
			r.Body,
		)

	decoder.DisallowUnknownFields()

	if err :=
		decoder.Decode(
			&request,
		); err != nil {
		writeAPIError(
			w,
			http.StatusBadRequest,
			apiErrorInvalidRequest,
			"Ungültige Anfrage.",
		)

		return request,
			storedPDFUpload{},
			false
	}

	if request.UploadID == "" {
		writeAPIError(
			w,
			http.StatusBadRequest,
			apiErrorInvalidRequest,
			"Upload-ID fehlt.",
		)

		return request,
			storedPDFUpload{},
			false
	}

	if !validPDFCompressionMode(
		request.Mode,
	) {
		writeAPIError(
			w,
			http.StatusBadRequest,
			apiErrorInvalidRequest,
			"Ungültiger Kompressionsmodus.",
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
		writeAPIError(
			w,
			http.StatusGone,
			apiErrorUploadExpired,
			"Die PDF ist nicht mehr verfügbar. Bitte erneut hochladen.",
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
	if s.handlePDFCompressionSpecialError(
		w,
		r,
		ctx,
		upload,
		mode,
		err,
	) {
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

	writeAPIError(
		w,
		http.StatusInternalServerError,
		apiErrorInternal,
		"Die PDF konnte nicht komprimiert werden.",
	)
}

func (s *Server) handlePDFCompressionSpecialError(
	w http.ResponseWriter,
	r *http.Request,
	ctx context.Context,
	upload storedPDFUpload,
	mode string,
	err error,
) bool {
	if errors.Is(
		err,
		errPDFCompressionInteractiveForm,
	) {
		s.logWarn(
			r,
			"lossy PDF compression rejected for interactive form",
			"upload_id",
			upload.ID,
			"filename",
			upload.Filename,
			"mode",
			mode,
		)

		writeAPIError(
			w,
			http.StatusUnprocessableEntity,
			apiErrorInvalidRequest,
			"Diese PDF enthält interaktive Formularfelder. Verwende den Modus „Verlustfrei“, damit das Formular erhalten bleibt.",
		)

		return true
	}

	return writeOperationContextAPIError(
		w,
		ctx,
		err,
		"Die PDF-Komprimierung hat zu lange gedauert.",
	)
}
