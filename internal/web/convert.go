package web

import (
	"context"
	"errors"
	"net/http"
	"os"
	"slices"
	"strings"
	"time"
)

func (s *Server) handleConvert(
	w http.ResponseWriter,
	r *http.Request,
) {
	contentType := r.Header.Get(
		"Content-Type",
	)

	if !strings.HasPrefix(
		contentType,
		"application/x-www-form-urlencoded",
	) {
		s.logWarn(
			r,
			"convert rejected",
			"reason",
			"invalid content type",
			"content_type",
			contentType,
		)

		http.Error(
			w,
			"direct conversion upload is not supported",
			http.StatusBadRequest,
		)

		return
	}

	if err := r.ParseForm(); err != nil {
		s.logWarn(
			r,
			"convert rejected",
			"reason",
			"invalid form",
		)

		http.Error(
			w,
			"invalid form",
			http.StatusBadRequest,
		)

		return
	}

	uploadID := r.FormValue(
		"upload_id",
	)

	if uploadID == "" {
		s.logWarn(
			r,
			"convert rejected",
			"reason",
			"missing upload id",
		)

		http.Error(
			w,
			"missing upload id",
			http.StatusBadRequest,
		)

		return
	}

	upload, ok := s.uploads.Take(
		uploadID,
	)
	if !ok {
		s.logWarn(
			r,
			"convert rejected",
			"reason",
			"upload expired",
		)

		http.Error(
			w,
			"upload expired or not found",
			http.StatusGone,
		)

		return
	}

	defer os.RemoveAll(
		upload.Directory,
	)

	if len(upload.Files) == 0 {
		s.logError(
			r,
			"convert upload empty",
			nil,
			"upload_id",
			uploadID,
		)

		http.Error(
			w,
			"upload contains no files",
			http.StatusBadRequest,
		)

		return
	}

	target := strings.ToLower(
		r.FormValue("target"),
	)

	if target == "" {
		s.logWarn(
			r,
			"convert rejected",
			"reason",
			"missing target",
			"source",
			upload.Format.ID,
			"files",
			len(upload.Files),
		)

		http.Error(
			w,
			"missing target format",
			http.StatusBadRequest,
		)

		return
	}

	if !slices.Contains(
		upload.Format.Targets,
		target,
	) {
		s.logWarn(
			r,
			"convert rejected",
			"reason",
			"unsupported target",
			"source",
			upload.Format.ID,
			"target",
			target,
		)

		http.Error(
			w,
			"unsupported conversion",
			http.StatusBadRequest,
		)

		return
	}

	if err := s.acquireConversionSlot(
		r.Context(),
	); err != nil {
		s.logWarn(
			r,
			"convert queue failed",
			"source",
			upload.Format.ID,
			"target",
			target,
			"files",
			len(upload.Files),
			"error",
			err,
		)

		if errors.Is(
			err,
			context.DeadlineExceeded,
		) {
			http.Error(
				w,
				"conversion queue full",
				http.StatusServiceUnavailable,
			)
		}

		return
	}
	defer s.releaseConversionSlot()

	started := time.Now()

	conversionCtx, cancel := context.WithTimeout(
		r.Context(),
		conversionTimeout,
	)
	defer cancel()

	var success bool

	if len(upload.Files) == 1 {
		success = s.convertSingleUpload(
			w,
			r,
			conversionCtx,
			upload,
			target,
		)
	} else {
		success = s.convertBatchUpload(
			w,
			r,
			conversionCtx,
			upload,
			target,
		)
	}

	if !success {
		return
	}

	s.logInfo(
		"convert",
		"source",
		upload.Format.ID,
		"target",
		target,
		"files",
		len(upload.Files),
		"duration",
		time.Since(started),
	)
}
