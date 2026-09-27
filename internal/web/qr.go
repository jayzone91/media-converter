package web

import (
	"encoding/json"
	"net/http"
	"time"

	qrservice "github.com/jayzone91/media-converter/internal/qr"
)

type qrGenerateResponse struct {
	SVG             string `json:"svg"`
	Version         int    `json:"version"`
	ErrorCorrection string `json:"error_correction"`
}

func (s *Server) handleQRGenerate(
	w http.ResponseWriter,
	r *http.Request,
) {
	started := time.Now()

	var request qrGenerateRequest

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&request); err != nil {
		s.logWarn(
			r,
			"QR generate rejected",
			"reason",
			"invalid request",
			"error",
			err,
		)

		http.Error(
			w,
			"Ungültige QR-Code-Anfrage.",
			http.StatusBadRequest,
		)
		return
	}

	payload, err := buildQRPayload(request)
	if err != nil {
		s.logWarn(
			r,
			"QR generate rejected",
			"type",
			request.Type,
			"reason",
			err.Error(),
		)

		http.Error(
			w,
			err.Error(),
			http.StatusBadRequest,
		)
		return
	}

	style, err := buildQRStyle(request.Style)
	if err != nil {
		s.logWarn(
			r,
			"QR generate rejected",
			"type",
			request.Type,
			"reason",
			err.Error(),
		)

		http.Error(
			w,
			err.Error(),
			http.StatusBadRequest,
		)
		return
	}

	matrix, err := qrservice.Generate(
		payload,
		style,
	)
	if err != nil {
		s.logError(
			r,
			"QR generation failed",
			err,
			"type",
			request.Type,
		)

		http.Error(
			w,
			"QR Code konnte nicht erzeugt werden.",
			http.StatusInternalServerError,
		)
		return
	}

	svg, err := qrservice.RenderSVG(
		matrix,
		style,
	)
	if err != nil {
		s.logError(
			r,
			"QR render failed",
			err,
			"type",
			request.Type,
			"version",
			matrix.Version,
		)

		http.Error(
			w,
			"QR Code konnte nicht gerendert werden.",
			http.StatusInternalServerError,
		)
		return
	}

	response := qrGenerateResponse{
		SVG:             string(svg),
		Version:         matrix.Version,
		ErrorCorrection: string(matrix.ErrorCorrection),
	}

	w.Header().Set(
		"Content-Type",
		"application/json; charset=utf-8",
	)

	if err := json.NewEncoder(w).Encode(response); err != nil {
		s.logError(
			r,
			"QR response failed",
			err,
			"type",
			request.Type,
		)

		return
	}

	s.logInfo(
		"QR generate",
		"type",
		request.Type,
		"version",
		matrix.Version,
		"ecc",
		matrix.ErrorCorrection,
		"duration",
		time.Since(started),
	)
}
