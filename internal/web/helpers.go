package web

import (
	"context"
	"errors"
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
	multipartMemoryLimit       = 32 << 20
	detectionTimeout           = 30 * time.Second
	conversionTimeout          = 30 * time.Minute
)

func parseMultipartForm(w http.ResponseWriter, r *http.Request) bool {
	r.Body = http.MaxBytesReader(
		w,
		r.Body,
		maxRequestSize,
	)

	if err := r.ParseMultipartForm(
		multipartMemoryLimit,
	); err != nil {
		var maxBytesError *http.MaxBytesError

		if errors.As(err, &maxBytesError) {
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

func validateFileSize(header *multipart.FileHeader) bool {
	return header.Size <= maxFileSize
}

func saveUpload(file multipart.File, filename, tempDir string) (string, error) {
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

	written, err := io.Copy(dst, limited)
	if err != nil {
		dst.Close()
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

func detectFormat(ctx context.Context, path string, ffprobe *media.FFProbe) (media.Format, error) {
	ctx, cancel := context.WithTimeout(
		ctx,
		detectionTimeout,
	)
	defer cancel()

	detection, err := media.Detect(path)
	if err == nil {
		if format, ok := media.FindByMIME(
			detection.MIME,
		); ok {
			return format, nil
		}
	}

	if format, err := media.DetectDocument(
		path,
	); err == nil {
		return format, nil
	}

	if format, ok := media.FindImageByExtension(
		path,
	); ok {
		return format, nil
	}

	format, err := ffprobe.Detect(
		ctx,
		path,
	)
	if err != nil {
		return media.Format{}, err
	}

	return format, nil
}
