package web

import (
	"encoding/json"
	"net/http"

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
	var request qrGenerateRequest

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&request); err != nil {
		s.logWarn(
			r,
			"invalid QR generate request",
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
		http.Error(
			w,
			err.Error(),
			http.StatusBadRequest,
		)
		return
	}

	style, err := buildQRStyle(request.Style)
	if err != nil {
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
			"QR code generation failed",
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
			"QR code rendering failed",
			err,
			"type",
			request.Type,
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
			"failed to encode QR generate response",
			err,
		)
	}
}
