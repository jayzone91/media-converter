package web

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"
)

const pdfMetadataTimeout = 30 * time.Second

func (s *Server) readPDFPageCount(
	r *http.Request,
	inputPath string,
	filename string,
	size int64,
) (int, error) {
	ctx, cancel :=
		context.WithTimeout(
			r.Context(),
			pdfMetadataTimeout,
		)

	defer cancel()

	metadata, err :=
		s.qpdf.PageCount(
			ctx,
			inputPath,
		)

	if err != nil {
		s.logError(
			r,
			"PDF metadata analysis failed",
			err,
			"filename",
			filename,
			"size_bytes",
			size,
			"timeout",
			pdfMetadataTimeout.String(),
		)

		if errors.Is(
			ctx.Err(),
			context.DeadlineExceeded,
		) ||
			errors.Is(
				err,
				context.DeadlineExceeded,
			) {
			return 0,
				context.DeadlineExceeded
		}

		return 0, err
	}

	if metadata.Warnings != "" {
		s.logWarn(
			r,
			"qpdf recovered PDF with warnings",
			"filename",
			filename,
			"size_bytes",
			size,
			"page_count",
			metadata.Count,
			"warnings",
			metadata.Warnings,
		)
	}

	return metadata.Count, nil
}

func buildPDFPreviewURLs(
	upload storedPDFUpload,
) []string {
	previews := make(
		[]string,
		upload.PageCount,
	)

	for index := range previews {
		previews[index] =
			fmt.Sprintf(
				"/pdf/uploads/%s/pages/%d",
				upload.ID,
				index+1,
			)
	}

	return previews
}

func (s *Server) handlePDFUploadDelete(
	w http.ResponseWriter,
	r *http.Request,
) {
	id :=
		r.PathValue(
			"id",
		)

	if id == "" {
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
		http.NotFound(
			w,
			r,
		)

		return
	}

	s.pdfUploads.Delete(
		id,
	)

	s.logger.Info(
		"PDF upload deleted",
		"method",
		r.Method,
		"path",
		r.URL.Path,
		"upload_id",
		id,
		"filename",
		upload.Filename,
	)

	w.WriteHeader(
		http.StatusNoContent,
	)
}
