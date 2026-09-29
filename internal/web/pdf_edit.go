package web

import (
	"net/http"
	"os"
	"path/filepath"
	"time"
)

const maxPDFEditRequestSize int64 = 129 << 20

func (s *Server) handlePDFEdit(
	w http.ResponseWriter,
	r *http.Request,
) {
	started := time.Now()

	r.Body = http.MaxBytesReader(
		w,
		r.Body,
		maxPDFEditRequestSize,
	)

	if err := r.ParseMultipartForm(
		multipartMemoryLimit,
	); err != nil {
		s.logWarn(
			r,
			"PDF edit rejected",
			"reason",
			"invalid multipart request",
		)

		writeAPIError(
			w,
			http.StatusBadRequest,
			apiErrorInvalidRequest,
			"Ungültige Anfrage.",
		)

		return
	}

	if r.MultipartForm != nil {
		defer r.MultipartForm.RemoveAll()
	}

	request, err :=
		decodePDFEditRequest(
			r.FormValue(
				"metadata",
			),
		)

	if err != nil {
		s.logWarn(
			r,
			"PDF edit rejected",
			"reason",
			err.Error(),
		)

		writeAPIError(
			w,
			http.StatusBadRequest,
			apiErrorInvalidRequest,
			"Ungültige PDF-Änderungen.",
		)

		return
	}

	if request.UploadID == "" {
		writeAPIError(
			w,
			http.StatusBadRequest,
			apiErrorInvalidRequest,
			"Upload-ID fehlt.",
		)

		return
	}

	if len(request.Texts) == 0 &&
		len(request.Images) == 0 &&
		len(request.Drawings) == 0 {
		writeAPIError(
			w,
			http.StatusBadRequest,
			apiErrorInvalidRequest,
			"Es wurden keine Änderungen vorgenommen.",
		)

		return
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

		return
	}

	textEdits, err :=
		validatePDFTextEditRequest(
			request.Texts,
			upload.PageCount,
		)

	if err != nil {
		handlePDFEditValidationError(
			s,
			w,
			r,
			upload.ID,
			err,
		)

		return
	}

	drawEdits, err :=
		validatePDFDrawEditRequest(
			request.Drawings,
			upload.PageCount,
		)

	if err != nil {
		handlePDFEditValidationError(
			s,
			w,
			r,
			upload.ID,
			err,
		)

		return
	}

	imageHeaders :=
		r.MultipartForm.File["image"]

	if len(imageHeaders) !=
		len(request.Images) {
		handlePDFEditValidationError(
			s,
			w,
			r,
			upload.ID,
			errImageCountMismatch,
		)

		return
	}

	if len(imageHeaders) >
		maxPDFEditImages {
		handlePDFEditValidationError(
			s,
			w,
			r,
			upload.ID,
			errTooManyPDFEditImages,
		)

		return
	}

	tempDir, err :=
		os.MkdirTemp(
			"",
			"media-converter-pdf-edit-*",
		)

	if err != nil {
		s.logError(
			r,
			"PDF edit temporary directory failed",
			err,
		)

		writeAPIError(
			w,
			http.StatusInternalServerError,
			apiErrorInternal,
			"Temporäres Verzeichnis konnte nicht erstellt werden.",
		)

		return
	}

	defer func() {
		if err := os.RemoveAll(
			tempDir,
		); err != nil {
			s.logError(
				r,
				"PDF edit cleanup failed",
				err,
				"directory",
				tempDir,
			)
		}
	}()

	imageEdits, err :=
		savePDFEditImages(
			tempDir,
			imageHeaders,
			request.Images,
			upload.PageCount,
		)

	if err != nil {
		handlePDFEditValidationError(
			s,
			w,
			r,
			upload.ID,
			err,
		)

		return
	}

	if err :=
		s.acquireWorkload(
			r.Context(),
			workloadPDFCPU,
		); err != nil {
		s.logError(
			r,
			"PDF edit queue failed",
			err,
			"upload_id",
			upload.ID,
		)

		writeQueueAPIError(
			w,
			err,
		)

		return
	}

	defer s.releaseWorkload(
		workloadPDFCPU,
	)

	outputPath :=
		filepath.Join(
			tempDir,
			"bearbeitet.pdf",
		)

	if err :=
		s.pdf.ApplyEdits(
			upload.Path,
			outputPath,
			textEdits,
			imageEdits,
			drawEdits,
		); err != nil {
		s.logError(
			r,
			"PDF edit failed",
			err,
			"upload_id",
			upload.ID,
			"filename",
			upload.Filename,
			"texts",
			len(textEdits),
			"images",
			len(imageEdits),
			"drawings",
			len(drawEdits),
		)

		writeAPIError(
			w,
			http.StatusInternalServerError,
			apiErrorInternal,
			"Die PDF konnte nicht bearbeitet werden.",
		)

		return
	}

	outputSize, err :=
		downloadFileSize(
			outputPath,
		)

	if err != nil {
		s.logError(
			r,
			"PDF edit output inspection failed",
			err,
			"upload_id",
			upload.ID,
		)

		writeAPIError(
			w,
			http.StatusInternalServerError,
			apiErrorInternal,
			"Die erzeugte PDF konnte nicht gelesen werden.",
		)

		return
	}

	if !s.prepareDownloadResponse(
		w,
		r,
		outputPath,
		"bearbeitet.pdf",
	) {
		return
	}

	s.pdfUploads.Delete(
		upload.ID,
	)

	s.logInfo(
		"PDF edited",
		"pages",
		upload.PageCount,
		"texts",
		len(textEdits),
		"images",
		len(imageEdits),
		"drawings",
		len(drawEdits),
		"input",
		upload.Size,
		"output",
		outputSize,
		"duration",
		time.Since(started),
	)
}
