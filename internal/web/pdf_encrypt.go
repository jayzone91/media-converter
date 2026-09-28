package web

import (
	"encoding/json"
	"net/http"
	"os"
)

type pdfEncryptRequest struct {
	UploadID string `json:"upload_id"`
	Password string `json:"password"`
}

func (s *Server) handlePDFEncrypt(
	w http.ResponseWriter,
	r *http.Request,
) {
	request, upload, ok :=
		s.readPDFEncryptRequest(
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
			"PDF encrypt queue failed",
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
			"media-converter-pdf-encrypt-*",
		)

	if err != nil {
		s.logError(
			r,
			"failed to create PDF encrypt temporary directory",
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
				"failed to remove PDF encrypt temporary directory",
				err,
				"directory",
				tempDir,
			)
		}
	}()

	ownerPassword, err :=
		generatePDFOwnerPassword()

	if err != nil {
		s.logError(
			r,
			"failed to generate PDF owner password",
			err,
			"upload_id",
			upload.ID,
		)

		writeAPIError(
			w,
			http.StatusInternalServerError,
			apiErrorInternal,
			"Verschlüsselung konnte nicht vorbereitet werden.",
		)

		return
	}

	outputPath :=
		pdfSecurityOutputPath(
			tempDir,
			"verschluesselt.pdf",
		)

	ctx, cancel :=
		pdfSecurityTimedContext(
			r.Context(),
		)

	defer cancel()

	if err :=
		s.qpdf.Encrypt(
			ctx,
			upload.Path,
			request.Password,
			ownerPassword,
			outputPath,
		); err != nil {
		if writeTimeoutAPIError(
			w,
			ctx,
			"Die Verschlüsselung hat zu lange gedauert.",
		) {
			return
		}

		s.logError(
			r,
			"PDF encryption failed",
			err,
			"upload_id",
			upload.ID,
			"filename",
			upload.Filename,
		)

		writeAPIError(
			w,
			http.StatusBadRequest,
			apiErrorInvalidRequest,
			"Die PDF konnte nicht verschlüsselt werden.",
		)

		return
	}

	s.writePDFSecurityResult(
		w,
		r,
		upload,
		outputPath,
		"verschluesselt.pdf",
		"encrypt",
	)
}

func (s *Server) readPDFEncryptRequest(
	w http.ResponseWriter,
	r *http.Request,
) (
	pdfEncryptRequest,
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

	var request pdfEncryptRequest

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
			false,
		); err != nil {
		writeAPIError(
			w,
			http.StatusBadRequest,
			apiErrorInvalidRequest,
			"Das Passwort muss zwischen 1 und 127 Zeichen lang sein und darf keine Zeilenumbrüche enthalten.",
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
