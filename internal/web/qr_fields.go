package web

import (
	"net/http"

	"github.com/jayzone91/media-converter/internal/web/view"
)

func (s *Server) handleQRFields(
	w http.ResponseWriter,
	r *http.Request,
) {
	kind := r.URL.Query().Get(
		"type",
	)

	if !validQRFieldType(kind) {
		http.Error(
			w,
			"unsupported QR type",
			http.StatusBadRequest,
		)

		return
	}

	w.Header().Set(
		"Content-Type",
		"text/html; charset=utf-8",
	)

	if err := view.QRFields(
		kind,
	).Render(
		r.Context(),
		w,
	); err != nil {
		http.Error(
			w,
			"failed to render QR fields",
			http.StatusInternalServerError,
		)
	}
}

func validQRFieldType(
	kind string,
) bool {
	switch kind {
	case
		"url",
		"text",
		"phone",
		"wifi",
		"vcard",
		"event":

		return true

	default:
		return false
	}
}
