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
			http.Error(
				w,
				"upload too large",
				http.StatusRequestEntityTooLarge,
			)

			return false
		}

		http.Error(
			w,
			"invalid multipart form",
			http.StatusBadRequest,
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

	/*
		Strukturierte Dokumente müssen vor der generischen
		MIME-Erkennung geprüft werden.

		RTF wird von net/http häufig als text/plain erkannt.
	*/
	if format, err := media.DetectDocument(
		path,
	); err == nil {
		if !media.FormatMatchesExtension(
			path,
			format,
		) {
			return media.Format{},
				formatMismatchError(
					path,
					format,
				)
		}

		return format, nil
	}

	detection, detectionErr :=
		media.Detect(path)

	if detectionErr == nil {
		if hasExpected &&
			media.FormatAcceptsMIME(
				expected,
				detection.MIME,
			) {
			return expected, nil
		}

		if detected, ok :=
			media.FindByMIME(
				detection.MIME,
			); ok {
			if !hasExpected {
				return media.Format{},
					fmt.Errorf(
						"unsupported file extension %q for detected format %s",
						filepath.Ext(path),
						detected.ID,
					)
			}

			return media.Format{},
				fmt.Errorf(
					"file extension %q does not match detected format %s",
					filepath.Ext(path),
					detected.ID,
				)
		}
	}

	/*
		Textformate dürfen niemals ausschließlich anhand ihrer
		Dateiendung akzeptiert werden.
	*/
	if hasExpected {
		switch expected.ID {
		case "txt", "markdown":
			return media.Format{},
				fmt.Errorf(
					"file content does not match %s",
					expected.ID,
				)
		}
	}

	/*
		Einige Bildformate werden von http.DetectContentType
		nicht zuverlässig erkannt.

		Diese bleiben vorerst über den bestehenden
		Extension-Fallback kompatibel.
	*/
	if format, ok :=
		media.FindImageByExtension(
			path,
		); ok {
		return format, nil
	}

	if ffprobe == nil {
		return media.Format{},
			errors.New(
				"media format could not be detected",
			)
	}

	format, err := ffprobe.Detect(
		ctx,
		path,
	)
	if err != nil {
		return media.Format{}, err
	}

	if !media.FormatMatchesExtension(
		path,
		format,
	) {
		return media.Format{},
			formatMismatchError(
				path,
				format,
			)
	}

	return format, nil
}

func formatMismatchError(
	path string,
	detected media.Format,
) error {
	expected, ok :=
		media.FindByExtension(path)

	if !ok {
		return fmt.Errorf(
			"unsupported file extension %q for detected format %s",
			filepath.Ext(path),
			detected.ID,
		)
	}

	return fmt.Errorf(
		"file extension indicates %s but content is %s",
		expected.ID,
		detected.ID,
	)
}
