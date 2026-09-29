package web

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"
)

const pdfPreviewTimeout = 2 * time.Minute

var errPDFPreviewQueue = errors.New(
	"PDF preview queue failed",
)

func (s *Server) handlePDFPreview(
	w http.ResponseWriter,
	r *http.Request,
) {
	id := r.PathValue("id")
	pageValue := r.PathValue("page")

	page, err :=
		strconv.Atoi(
			pageValue,
		)

	if err != nil ||
		page < 1 {
		s.logWarn(
			r,
			"invalid PDF preview page requested",
			"upload_id",
			id,
			"page_value",
			pageValue,
		)

		http.NotFound(
			w,
			r,
		)

		return
	}

	upload, ok :=
		s.pdfUploads.Get(
			id,
		)

	if !ok {
		s.logWarn(
			r,
			"PDF preview requested for unknown upload",
			"upload_id",
			id,
			"page",
			page,
		)

		http.NotFound(
			w,
			r,
		)

		return
	}

	if page >
		upload.PageCount {
		s.logWarn(
			r,
			"PDF preview page exceeds document page count",
			"upload_id",
			upload.ID,
			"filename",
			upload.Filename,
			"page",
			page,
			"page_count",
			upload.PageCount,
		)

		http.NotFound(
			w,
			r,
		)

		return
	}

	previewPath :=
		upload.PreviewPath(
			page,
		)

	if _, err :=
		os.Stat(
			previewPath,
		); errors.Is(
		err,
		os.ErrNotExist,
	) {
		if err :=
			s.renderPDFPreview(
				r,
				upload,
				page,
				previewPath,
			); err != nil {
			s.handlePDFPreviewError(
				w,
				r,
				upload,
				page,
				err,
			)

			return
		}
	} else if err != nil {
		s.logError(
			r,
			"failed to inspect PDF preview cache",
			err,
			"upload_id",
			upload.ID,
			"filename",
			upload.Filename,
			"page",
			page,
		)

		writeAPIError(
			w,
			http.StatusInternalServerError,
			apiErrorInternal,
			"PDF-Vorschau konnte nicht gelesen werden.",
		)

		return
	}

	w.Header().Set(
		"Content-Type",
		"image/jpeg",
	)

	w.Header().Set(
		"Cache-Control",
		"private, max-age=1800",
	)

	http.ServeFile(
		w,
		r,
		previewPath,
	)
}

func (s *Server) handlePDFPreviewError(
	w http.ResponseWriter,
	r *http.Request,
	upload storedPDFUpload,
	page int,
	err error,
) {
	s.logError(
		r,
		"PDF preview rendering failed",
		err,
		"upload_id",
		upload.ID,
		"filename",
		upload.Filename,
		"size_bytes",
		upload.Size,
		"page",
		page,
		"page_count",
		upload.PageCount,
	)

	if errors.Is(
		err,
		errPDFPreviewQueue,
	) {
		writeQueueAPIError(
			w,
			err,
		)

		return
	}

	if errors.Is(
		err,
		context.DeadlineExceeded,
	) {
		writeAPIError(
			w,
			http.StatusGatewayTimeout,
			apiErrorTimeout,
			"PDF-Vorschau konnte nicht rechtzeitig erstellt werden.",
		)

		return
	}

	if errors.Is(
		err,
		context.Canceled,
	) {
		return
	}

	writeAPIError(
		w,
		http.StatusInternalServerError,
		apiErrorInternal,
		"PDF-Vorschau konnte nicht erstellt werden.",
	)
}

func (s *Server) renderPDFPreview(
	r *http.Request,
	upload storedPDFUpload,
	page int,
	outputPath string,
) error {
	if err :=
		s.acquireWorkload(
			r.Context(),
			workloadPDFPreview,
		); err != nil {
		return fmt.Errorf(
			"%w: %w",
			errPDFPreviewQueue,
			err,
		)
	}

	defer s.releaseWorkload(
		workloadPDFPreview,
	)

	ctx, cancel :=
		context.WithTimeout(
			r.Context(),
			pdfPreviewTimeout,
		)

	defer cancel()

	if err :=
		s.pdf.RenderPreviewPage(
			ctx,
			upload.Path,
			outputPath,
			page,
		); err != nil {
		if ctxErr :=
			ctx.Err(); ctxErr != nil {
			return fmt.Errorf(
				"PDF preview rendering failed: %w",
				ctxErr,
			)
		}

		return err
	}

	return nil
}
