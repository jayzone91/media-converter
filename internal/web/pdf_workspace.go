package web

import (
	"net/http"

	"github.com/jayzone91/media-converter/internal/web/view"
)

func (s *Server) handlePDFWorkspace(
	w http.ResponseWriter,
	r *http.Request,
) {
	tool := r.PathValue(
		"tool",
	)

	w.Header().Set(
		"Content-Type",
		"text/html; charset=utf-8",
	)

	switch tool {
	case "merge":
		if err := view.PDFMergeWorkspace().Render(
			r.Context(),
			w,
		); err != nil {
			http.Error(
				w,
				"PDF-Werkzeug konnte nicht geladen werden.",
				http.StatusInternalServerError,
			)
		}

	case
		"split",
		"compress",
		"edit",
		"encrypt",
		"decrypt",
		"rotate",
		"delete",
		"extract",
		"sort",
		"web",
		"optimize",
		"redact",
		"create":

		if err := view.PDFPlaceholderWorkspace(
			pdfToolTitle(
				tool,
			),
		).Render(
			r.Context(),
			w,
		); err != nil {
			http.Error(
				w,
				"PDF-Werkzeug konnte nicht geladen werden.",
				http.StatusInternalServerError,
			)
		}

	default:
		http.NotFound(
			w,
			r,
		)
	}
}

func pdfToolTitle(
	tool string,
) string {
	switch tool {
	case "split":
		return "PDF trennen"

	case "compress":
		return "PDF komprimieren"

	case "edit":
		return "PDF bearbeiten"

	case "encrypt":
		return "PDF verschlüsseln"

	case "decrypt":
		return "Passwort entfernen"

	case "rotate":
		return "Seiten drehen"

	case "delete":
		return "Seiten löschen"

	case "extract":
		return "Seiten extrahieren"

	case "sort":
		return "Seiten sortieren"

	case "web":
		return "Webseite in PDF"

	case "optimize":
		return "PDF optimieren"

	case "redact":
		return "PDF schwärzen"

	case "create":
		return "PDF erstellen"

	default:
		return "PDF-Werkzeug"
	}
}
