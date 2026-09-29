package web

import (
	"context"
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

		writeAPIError(
			w,
			http.StatusBadRequest,
			apiErrorInvalidRequest,
			"Direkter Datei-Upload ist an diesem Endpunkt nicht erlaubt.",
		)

		return
	}

	if err :=
		r.ParseForm(); err != nil {
		s.logWarn(
			r,
			"convert rejected",
			"reason",
			"invalid form",
		)

		writeAPIError(
			w,
			http.StatusBadRequest,
			apiErrorInvalidRequest,
			"Ungültige Anfrage.",
		)

		return
	}

	uploadID :=
		r.FormValue(
			"upload_id",
		)

	if uploadID == "" {
		s.logWarn(
			r,
			"convert rejected",
			"reason",
			"missing upload id",
		)

		writeAPIError(
			w,
			http.StatusBadRequest,
			apiErrorInvalidRequest,
			"Upload-ID fehlt.",
		)

		return
	}

	upload, ok :=
		s.uploads.Take(
			uploadID,
		)

	if !ok {
		s.logWarn(
			r,
			"convert rejected",
			"reason",
			"upload expired",
		)

		writeAPIError(
			w,
			http.StatusGone,
			apiErrorUploadExpired,
			"Der Upload ist nicht mehr verfügbar. Bitte erneut hochladen.",
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

		writeAPIError(
			w,
			http.StatusBadRequest,
			apiErrorInvalidRequest,
			"Der Upload enthält keine Dateien.",
		)

		return
	}

	target :=
		strings.ToLower(
			r.FormValue(
				"target",
			),
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

		writeAPIError(
			w,
			http.StatusBadRequest,
			apiErrorInvalidRequest,
			"Zielformat fehlt.",
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

		writeAPIError(
			w,
			http.StatusBadRequest,
			apiErrorUnsupportedConversion,
			"Diese Konvertierung wird nicht unterstützt.",
		)

		return
	}

	workloadPlan, err :=
		conversionWorkloads(
			upload.Format,
			target,
		)

	if err != nil {
		s.logError(
			r,
			"conversion workload mapping missing",
			err,
			"source",
			upload.Format.ID,
			"target",
			target,
		)

		writeAPIError(
			w,
			http.StatusInternalServerError,
			apiErrorInternal,
			"Die Konvertierung konnte nicht vorbereitet werden.",
		)

		return
	}

	releaseWorkloads, err :=
		s.acquireConversionWorkloads(
			r.Context(),
			workloadPlan,
		)

	if err != nil {
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

		writeQueueAPIError(
			w,
			err,
		)

		return
	}

	defer releaseWorkloads()

	started :=
		time.Now()

	conversionCtx, cancel :=
		context.WithTimeout(
			r.Context(),
			conversionTimeout,
		)

	defer cancel()

	var success bool

	if len(upload.Files) == 1 {
		success =
			s.convertSingleUpload(
				w,
				r,
				conversionCtx,
				upload,
				target,
			)
	} else {
		success =
			s.convertBatchUpload(
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
		time.Since(
			started,
		),
	)
}
