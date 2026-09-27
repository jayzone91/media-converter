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

		http.Error(
			w,
			"Temporäres Verzeichnis konnte nicht erstellt werden.",
			http.StatusInternalServerError,
		)
		return
	}

	defer func() {
		if err := os.RemoveAll(
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

	if err := s.qpdf.Decrypt(
		ctx,
		upload.Path,
		request.Password,
		outputPath,
	); err != nil {
		if isPDFSecurityTimeout(
			ctx,
			err,
		) {
			http.Error(
				w,
				"Das Entfernen des Passworts hat zu lange gedauert.",
				http.StatusGatewayTimeout,
			)
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

		http.Error(
			w,
			"Das Passwort ist falsch oder die PDF kann nicht entschlüsselt werden.",
			http.StatusBadRequest,
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
	r.Body = http.MaxBytesReader(
		w,
		r.Body,
		maxPDFCompressRequestSize,
	)
	defer r.Body.Close()

	var request pdfDecryptRequest

	decoder := json.NewDecoder(
		r.Body,
	)

	decoder.DisallowUnknownFields()

	if err := decoder.Decode(
		&request,
	); err != nil {
		http.Error(
			w,
			"Ungültige Anfrage.",
			http.StatusBadRequest,
		)

		return request,
			storedPDFUpload{},
			false
	}

	if request.UploadID == "" {
		http.Error(
			w,
			"Upload-ID fehlt.",
			http.StatusBadRequest,
		)

		return request,
			storedPDFUpload{},
			false
	}

	if err := validatePDFPassword(
		request.Password,
		true,
	); err != nil {
		http.Error(
			w,
			"Das Passwort darf maximal 256 Zeichen lang sein.",
			http.StatusBadRequest,
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
		http.Error(
			w,
			"Die PDF ist nicht mehr verfügbar. Bitte erneut hochladen.",
			http.StatusGone,
		)

		return request,
			storedPDFUpload{},
			false
	}

	return request,
		upload,
		true
}
