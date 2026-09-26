package web

import (
	"io"
	"mime/multipart"
	"os"
	"path/filepath"

	"github.com/jayzone91/media-converter/internal/media"
)

const MAX_FILE_SIZE = 512 << 20

func saveUpload(file multipart.File, filename string, tempDir string) (string, error) {
	path := filepath.Join(tempDir, filepath.Base(filename))

	dst, err := os.Create(path)
	if err != nil {
		return "", err
	}

	if _, err := io.Copy(dst, file); err != nil {
		dst.Close()
		return "", err
	}

	if err := dst.Close(); err != nil {
		return "", err
	}

	return path, nil
}

func detectFormat(path string, ffprobe *media.FFProbe) (media.Format, error) {
	detection, err := media.Detect(path)
	if err == nil {
		if format, ok := media.FindByMIME(detection.MIME); ok {
			return format, nil
		}
	}

	if format, err := media.DetectDocument(path); err == nil {
		return format, nil
	}

	format, err := ffprobe.Detect(path)
	if err != nil {
		return media.Format{}, err
	}

	return format, nil
}
