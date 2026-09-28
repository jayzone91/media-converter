package web

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	maxPDFPasswordLength = 127

	pdfSecurityTimeout = 30 * time.Minute
)

func validatePDFPassword(
	password string,
	allowEmpty bool,
) error {
	if password == "" &&
		!allowEmpty {
		return fmt.Errorf(
			"password must not be empty",
		)
	}

	if len([]rune(password)) >
		maxPDFPasswordLength {
		return fmt.Errorf(
			"password exceeds maximum length",
		)
	}

	if strings.ContainsAny(
		password,
		"\r\n\x00",
	) {
		return fmt.Errorf(
			"password contains unsupported control characters",
		)
	}

	return nil
}

func generatePDFOwnerPassword() (
	string,
	error,
) {
	buffer := make(
		[]byte,
		32,
	)

	if _, err :=
		rand.Read(
			buffer,
		); err != nil {
		return "",
			fmt.Errorf(
				"generate owner password: %w",
				err,
			)
	}

	return hex.EncodeToString(
		buffer,
	), nil
}

func (s *Server) writePDFSecurityResult(
	w http.ResponseWriter,
	r *http.Request,
	upload storedPDFUpload,
	path string,
	filename string,
	operation string,
) {
	outputSize, err :=
		downloadFileSize(
			path,
		)

	if err != nil {
		s.logError(
			r,
			"failed to inspect PDF security result",
			err,
			"operation",
			operation,
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
		path,
		filename,
	) {
		return
	}

	s.pdfUploads.Delete(
		upload.ID,
	)

	s.logInfo(
		"PDF security",
		"operation",
		operation,
		"input",
		upload.Size,
		"output",
		outputSize,
	)
}

func createPDFSecurityTempDirectory(
	prefix string,
) (string, error) {
	directory, err :=
		os.MkdirTemp(
			"",
			prefix,
		)

	if err != nil {
		return "",
			fmt.Errorf(
				"create temporary directory: %w",
				err,
			)
	}

	return directory,
		nil
}

func pdfSecurityOutputPath(
	directory string,
	filename string,
) string {
	return filepath.Join(
		directory,
		filename,
	)
}

func pdfSecurityTimedContext(
	parent context.Context,
) (
	context.Context,
	context.CancelFunc,
) {
	return context.WithTimeout(
		parent,
		pdfSecurityTimeout,
	)
}
