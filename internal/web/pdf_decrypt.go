package web

import (
	"encoding/json"
	"net/http"
	"os"
)

type pdfDecryptRequest struct {
	UploadID string `json:"upload_id"`
	Password string `json:"password"`
}

func (s *Server) handlePDFDecrypt(
	w http.ResponseWriter,
	r *http.Request,
) {
	request, upload, ok :=
		s.readPDFDecryptRequest(
			w,
			r,
		)

	if !ok {
		return
	}

	if err :=
		s.acquireWorkload(
			r.Context(),
			workloadQPDF,
		); err != nil {
		s.logError(
			r,
			"PDF decrypt queue failed",
			err,
			"filename",
			upload.Filename,
		)

		writeQueueAPIError(
			w,
			err,
		)

		return
	}

	defer s.releaseWorkload(
		workloadQPDF,
	)

	tempDir, err :=
		createPDFSecurityTempDirectory(
			"media-converter-pdf-decrypt-*",
		)

	if err != nil {
		s.logError(
			r,
			"failed to create PDF decrypt temporary directory",
			err,
			"upload_id",
			upload.ID,
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
		if err :=
			os.RemoveAll(
				tempDir,
			); err != nil {
			s.logError(
				r,
				"failed to remove PDF decrypt temporary directory",
				err,
				"directory",
				tempDir,
			)
		}
	}()

	outputPath :=
		pdfSecurityOutputPath(
			tempDir,
			"ohne-passwort.pdf",
		)

	ctx, cancel :=
		pdfSecurityTimedContext(
			r.Context(),
		)

	defer cancel()

	if err :=
		s.qpdf.Decrypt(
			ctx,
			upload.Path,
			request.Password,
			outputPath,
		); err != nil {
		if writeTimeoutAPIError(
			w,
			ctx,
			"Das Entfernen des Passworts hat zu lange gedauert.",
		) {
			return
		}

		s.logWarn(
			r,
			"PDF decryption failed",
			"upload_id",
			upload.ID,
			"filename",
			upload.Filename,
		)

		writeAPIError(
			w,
			http.StatusBadRequest,
			apiErrorInvalidRequest,
			"Das Passwort ist falsch oder die PDF kann nicht entschlüsselt werden.",
		)

		return
	}

	s.writePDFSecurityResult(
		w,
		r,
		upload,
		outputPath,
		"ohne-passwort.pdf",
		"decrypt",
	)
}

func (s *Server) readPDFDecryptRequest(
	w http.ResponseWriter,
	r *http.Request,
) (
	pdfDecryptRequest,
	storedPDFUpload,
	bool,
) {
	r.Body =
		http.MaxBytesReader(
			w,
			r.Body,
			maxPDFCompressRequestSize,
		)

	defer r.Body.Close()

	var request pdfDecryptRequest

	decoder :=
		json.NewDecoder(
			r.Body,
		)

	decoder.DisallowUnknownFields()

	if err :=
		decoder.Decode(
			&request,
		); err != nil {
		writeAPIError(
			w,
			http.StatusBadRequest,
			apiErrorInvalidRequest,
			"Ungültige Anfrage.",
		)

		return request,
			storedPDFUpload{},
			false
	}

	if request.UploadID == "" {
		writeAPIError(
			w,
			http.StatusBadRequest,
			apiErrorInvalidRequest,
			"Upload-ID fehlt.",
		)

		return request,
			storedPDFUpload{},
			false
	}

	if err :=
		validatePDFPassword(
			request.Password,
			true,
		); err != nil {
		writeAPIError(
			w,
			http.StatusBadRequest,
			apiErrorInvalidRequest,
			"Das Passwort darf maximal 127 Zeichen lang sein und keine Zeilenumbrüche enthalten.",
		)

		return request,
			storedPDFUpload{},
			false
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

		return request,
			storedPDFUpload{},
			false
	}

	return request,
		upload,
		true
}
