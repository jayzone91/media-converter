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
			s.handlePDFWorkspaceRenderError(
				w,
				r,
				"merge",
				err,
			)
		}

	case "sort":
		if err := view.PDFSortWorkspace().Render(
			r.Context(),
			w,
		); err != nil {
			s.handlePDFWorkspaceRenderError(
				w,
				r,
				"sort",
				err,
			)
		}

	case "delete":
		if err := view.PDFDeleteWorkspace().Render(
			r.Context(),
			w,
		); err != nil {
			s.handlePDFWorkspaceRenderError(
				w,
				r,
				"delete",
				err,
			)
		}

	case "extract":
		if err := view.PDFExtractWorkspace().Render(
			r.Context(),
			w,
		); err != nil {
			s.handlePDFWorkspaceRenderError(
				w,
				r,
				"extract",
				err,
			)
		}

	case
		"split",
		"compress",
		"edit",
		"encrypt",
		"decrypt",
		"rotate",
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
			s.handlePDFWorkspaceRenderError(
				w,
				r,
				tool,
				err,
			)
		}

	default:
		http.NotFound(
			w,
			r,
		)
	}
}

func (s *Server) handlePDFWorkspaceRenderError(
	w http.ResponseWriter,
	r *http.Request,
	tool string,
	err error,
) {
	s.logError(
		r,
		"failed to render PDF workspace",
		err,
		"tool",
		tool,
	)

	http.Error(
		w,
		"PDF-Werkzeug konnte nicht geladen werden.",
		http.StatusInternalServerError,
	)
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
