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

		http.Error(
			w,
			"Verschlüsselung konnte nicht vorbereitet werden.",
			http.StatusInternalServerError,
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

	if err := s.qpdf.Encrypt(
		ctx,
		upload.Path,
		request.Password,
		ownerPassword,
		outputPath,
	); err != nil {
		if isPDFSecurityTimeout(
			ctx,
			err,
		) {
			http.Error(
				w,
				"Die Verschlüsselung hat zu lange gedauert.",
				http.StatusGatewayTimeout,
			)
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

		http.Error(
			w,
			"Die PDF konnte nicht verschlüsselt werden.",
			http.StatusBadRequest,
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
	r.Body = http.MaxBytesReader(
		w,
		r.Body,
		maxPDFCompressRequestSize,
	)
	defer r.Body.Close()

	var request pdfEncryptRequest

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
		false,
	); err != nil {
		http.Error(
			w,
			"Das Passwort muss zwischen 1 und 256 Zeichen lang sein.",
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
