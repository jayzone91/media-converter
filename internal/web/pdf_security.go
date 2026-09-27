package web

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

const (
	maxPDFPasswordLength = 256

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

	if _, err := rand.Read(
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
	file, err := os.Open(
		path,
	)
	if err != nil {
		s.logError(
			r,
			"failed to open PDF security result",
			err,
			"operation",
			operation,
			"upload_id",
			upload.ID,
		)

		http.Error(
			w,
			"Die erzeugte PDF konnte nicht geöffnet werden.",
			http.StatusInternalServerError,
		)
		return
	}
	defer file.Close()

	info, err := file.Stat()
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

		http.Error(
			w,
			"Die erzeugte PDF konnte nicht gelesen werden.",
			http.StatusInternalServerError,
		)
		return
	}

	disposition :=
		mime.FormatMediaType(
			"attachment",
			map[string]string{
				"filename": filename,
			},
		)

	w.Header().Set(
		"Content-Type",
		"application/pdf",
	)

	w.Header().Set(
		"Content-Disposition",
		disposition,
	)

	w.Header().Set(
		"Content-Length",
		fmt.Sprintf(
			"%d",
			info.Size(),
		),
	)

	w.Header().Set(
		"Cache-Control",
		"no-store",
	)

	if _, err := io.Copy(
		w,
		file,
	); err != nil {
		s.logError(
			r,
			"failed to send PDF security result",
			err,
			"operation",
			operation,
			"upload_id",
			upload.ID,
		)

		return
	}

	s.pdfUploads.Delete(
		upload.ID,
	)

	s.logger.Info(
		"PDF security operation completed",
		"method",
		r.Method,
		"path",
		r.URL.Path,
		"operation",
		operation,
		"upload_id",
		upload.ID,
		"filename",
		upload.Filename,
		"input_size_bytes",
		upload.Size,
		"output_size_bytes",
		info.Size(),
	)
}

func createPDFSecurityTempDirectory(
	prefix string,
) (string, error) {
	directory, err := os.MkdirTemp(
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

func isPDFSecurityTimeout(
	ctx context.Context,
	err error,
) bool {
	return errors.Is(
		ctx.Err(),
		context.DeadlineExceeded,
	) ||
		errors.Is(
			err,
			context.DeadlineExceeded,
		)
}
