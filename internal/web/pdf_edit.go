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

		http.Error(
			w,
			"Ungültige Anfrage.",
			http.StatusBadRequest,
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

		http.Error(
			w,
			"Ungültige PDF-Änderungen.",
			http.StatusBadRequest,
		)

		return
	}

	if request.UploadID == "" {
		http.Error(
			w,
			"Upload-ID fehlt.",
			http.StatusBadRequest,
		)

		return
	}

	if len(request.Texts) == 0 &&
		len(request.Images) == 0 &&
		len(request.Drawings) == 0 {
		http.Error(
			w,
			"Es wurden keine Änderungen vorgenommen.",
			http.StatusBadRequest,
		)

		return
	}

	upload, ok :=
		s.pdfUploads.Get(
			request.UploadID,
		)

	if !ok {
		http.Error(
			w,
			"Die PDF ist nicht mehr verfügbar. Bitte erneut hochladen.",
			http.StatusGone,
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

	tempDir, err := os.MkdirTemp(
		"",
		"media-converter-pdf-edit-*",
	)
	if err != nil {
		s.logError(
			r,
			"PDF edit temporary directory failed",
			err,
		)

		http.Error(
			w,
			"Temporäres Verzeichnis konnte nicht erstellt werden.",
			http.StatusInternalServerError,
		)

		return
	}

	defer os.RemoveAll(
		tempDir,
	)

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

	if err := s.acquireConversionSlot(
		r.Context(),
	); err != nil {
		http.Error(
			w,
			"Der Server ist momentan ausgelastet. Bitte später erneut versuchen.",
			http.StatusServiceUnavailable,
		)

		return
	}

	defer s.releaseConversionSlot()

	outputPath := filepath.Join(
		tempDir,
		"bearbeitet.pdf",
	)

	if err := s.pdf.ApplyEdits(
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

		http.Error(
			w,
			"Die PDF konnte nicht bearbeitet werden.",
			http.StatusInternalServerError,
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

		http.Error(
			w,
			"Die erzeugte PDF konnte nicht gelesen werden.",
			http.StatusInternalServerError,
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
