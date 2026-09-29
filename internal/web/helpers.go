package web

import (
	"context"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/jayzone91/media-converter/internal/media"
)

const (
	maxFileSize          int64 = 512 << 20
	maxRequestSize             = maxFileSize + (1 << 20)
	maxBatchFiles              = 20
	multipartMemoryLimit       = 32 << 20
	detectionTimeout           = 30 * time.Second
	conversionTimeout          = 30 * time.Minute
)

var (
	errUnsupportedExtension = errors.New(
		"unsupported file extension",
	)
	errContentMismatch = errors.New(
		"file content does not match extension",
	)
	errFormatMismatch = errors.New(
		"detected format does not match extension",
	)
	errDetectionFailed = errors.New(
		"media format could not be detected",
	)
)

func parseMultipartForm(
	w http.ResponseWriter,
	r *http.Request,
) bool {
	r.Body = http.MaxBytesReader(
		w,
		r.Body,
		maxRequestSize,
	)

	if err := r.ParseMultipartForm(
		multipartMemoryLimit,
	); err != nil {
		var maxBytesError *http.MaxBytesError

		if errors.As(
			err,
			&maxBytesError,
		) {
			writeAPIError(
				w,
				http.StatusRequestEntityTooLarge,
				apiErrorInvalidRequest,
				"Der Upload ist größer als 512 MiB.",
			)

			return false
		}

		writeAPIError(
			w,
			http.StatusBadRequest,
			apiErrorInvalidRequest,
			"Ungültiger Upload.",
		)

		return false
	}

	return true
}

func validateFileSize(
	header *multipart.FileHeader,
) bool {
	return header.Size <= maxFileSize
}

func saveUpload(
	file multipart.File,
	filename string,
	tempDir string,
) (string, error) {
	path := filepath.Join(
		tempDir,
		filepath.Base(filename),
	)

	dst, err := os.Create(path)
	if err != nil {
		return "", err
	}

	limited := io.LimitReader(
		file,
		maxFileSize+1,
	)

	written, err := io.Copy(
		dst,
		limited,
	)
	if err != nil {
		_ = dst.Close()

		return "", err
	}

	if err := dst.Close(); err != nil {
		return "", err
	}

	if written > maxFileSize {
		_ = os.Remove(path)

		return "", errors.New(
			"file exceeds maximum size",
		)
	}

	return path, nil
}

func detectFormat(
	ctx context.Context,
	path string,
	ffprobe *media.FFProbe,
) (media.Format, error) {
	ctx, cancel := context.WithTimeout(
		ctx,
		detectionTimeout,
	)
	defer cancel()

	expected, hasExpected :=
		media.FindByExtension(path)

	if !hasExpected {
		return media.Format{},
			fmt.Errorf(
				"%w: %q",
				errUnsupportedExtension,
				filepath.Ext(path),
			)
	}

	if format, err :=
		media.DetectDocument(
			path,
		); err == nil {
		return validateDetectedFormat(
			path,
			format,
		)
	}

	if format, err :=
		media.DetectSignature(
			path,
		); err == nil {
		return validateDetectedFormat(
			path,
			format,
		)
	}

	detection, detectionErr :=
		media.Detect(path)

	if detectionErr == nil &&
		media.FormatAcceptsMIME(
			expected,
			detection.MIME,
		) {
		switch expected.ID {
		case "txt", "markdown":
			return expected, nil
		}
	}

	switch expected.Category {
	case media.CategoryImage,
		media.CategoryPDF,
		media.CategoryDocument:
		return media.Format{},
			fmt.Errorf(
				"%w: expected %s",
				errContentMismatch,
				expected.ID,
			)

	case media.CategoryMarkdown:
		return media.Format{},
			fmt.Errorf(
				"%w: expected markdown",
				errContentMismatch,
			)
	}

	if ffprobe == nil {
		return media.Format{},
			errDetectionFailed
	}

	format, err := ffprobe.Detect(
		ctx,
		path,
	)
	if err != nil {
		return media.Format{},
			fmt.Errorf(
				"%w: %v",
				errDetectionFailed,
				err,
			)
	}

	return validateDetectedFormat(
		path,
		format,
	)
}

func validateDetectedFormat(
	path string,
	detected media.Format,
) (media.Format, error) {
	if media.FormatMatchesExtension(
		path,
		detected,
	) {
		return detected, nil
	}

	return media.Format{},
		formatMismatchError(
			path,
			detected,
		)
}

func formatMismatchError(
	path string,
	detected media.Format,
) error {
	expected, ok :=
		media.FindByExtension(path)

	if !ok {
		return fmt.Errorf(
			"%w: extension %q, detected %s",
			errUnsupportedExtension,
			filepath.Ext(path),
			detected.ID,
		)
	}

	return fmt.Errorf(
		"%w: extension indicates %s, content is %s",
		errFormatMismatch,
		expected.ID,
		detected.ID,
	)
}

func detectionErrorMessage(
	filename string,
	err error,
) string {
	switch {
	case errors.Is(
		err,
		errUnsupportedExtension,
	):
		extension := filepath.Ext(
			filename,
		)

		if extension == "" {
			return fmt.Sprintf(
				"Die Datei %q hat keine unterstützte Dateiendung.",
				filename,
			)
		}

		return fmt.Sprintf(
			"Dateien mit der Endung %q werden nicht unterstützt.",
			extension,
		)

	case errors.Is(
		err,
		errFormatMismatch,
	):
		return fmt.Sprintf(
			"Dateiendung und tatsächlicher Dateityp von %q stimmen nicht überein.",
			filename,
		)

	case errors.Is(
		err,
		errContentMismatch,
	):
		return fmt.Sprintf(
			"Der Inhalt von %q entspricht nicht dem angegebenen Dateityp.",
			filename,
		)

	case errors.Is(
		err,
		errDetectionFailed,
	):
		return fmt.Sprintf(
			"Der Dateityp von %q konnte nicht erkannt werden oder wird nicht unterstützt.",
			filename,
		)

	default:
		return fmt.Sprintf(
			"Die Datei %q konnte nicht als unterstütztes Format erkannt werden.",
			filename,
		)
	}
}
