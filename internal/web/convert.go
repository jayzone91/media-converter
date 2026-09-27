package web

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/jayzone91/media-converter/internal/converter"
	"github.com/jayzone91/media-converter/internal/media"
)

func (s *Server) handleConvert(
	w http.ResponseWriter,
	r *http.Request,
) {
	var (
		inputPath string
		baseName  string
		tempDir   string
		format    media.Format
		err       error
	)

	contentType := r.Header.Get(
		"Content-Type",
	)

	if strings.HasPrefix(
		contentType,
		"application/x-www-form-urlencoded",
	) {
		if err := r.ParseForm(); err != nil {
			http.Error(
				w,
				"invalid form",
				http.StatusBadRequest,
			)
			return
		}

		uploadID := r.FormValue(
			"upload_id",
		)

		if uploadID == "" {
			http.Error(
				w,
				"missing upload id",
				http.StatusBadRequest,
			)
			return
		}

		upload, ok := s.uploads.Take(
			uploadID,
		)
		if !ok {
			http.Error(
				w,
				"upload expired or not found",
				http.StatusGone,
			)
			return
		}

		tempDir = upload.Directory
		inputPath = upload.Path
		format = upload.Format

		baseName = strings.TrimSuffix(
			filepath.Base(upload.Filename),
			filepath.Ext(upload.Filename),
		)

		defer os.RemoveAll(tempDir)
	} else {
		if !parseMultipartForm(w, r) {
			return
		}

		file, header, err := r.FormFile("file")
		if err != nil {
			http.Error(
				w,
				"missing file",
				http.StatusBadRequest,
			)
			return
		}
		defer file.Close()

		if !validateFileSize(header) {
			http.Error(
				w,
				"upload too large",
				http.StatusRequestEntityTooLarge,
			)
			return
		}

		tempDir, err = os.MkdirTemp(
			"",
			"media-converter-*",
		)
		if err != nil {
			http.Error(
				w,
				"failed to create temp dir",
				http.StatusInternalServerError,
			)
			return
		}
		defer os.RemoveAll(tempDir)

		inputPath, err = saveUpload(
			file,
			header.Filename,
			tempDir,
		)
		if err != nil {
			http.Error(
				w,
				"failed to save upload",
				http.StatusInternalServerError,
			)
			return
		}

		format, err = detectFormat(
			r.Context(),
			inputPath,
			s.ffprobe,
		)
		if err != nil {
			if errors.Is(
				err,
				context.DeadlineExceeded,
			) {
				http.Error(
					w,
					"media detection timed out",
					http.StatusGatewayTimeout,
				)
				return
			}

			http.Error(
				w,
				"unsupported media type",
				http.StatusUnsupportedMediaType,
			)
			return
		}

		baseName = strings.TrimSuffix(
			filepath.Base(header.Filename),
			filepath.Ext(header.Filename),
		)
	}

	target := strings.ToLower(
		r.FormValue("target"),
	)

	if target == "" {
		http.Error(
			w,
			"missing target format",
			http.StatusBadRequest,
		)
		return
	}

	if !slices.Contains(
		format.Targets,
		target,
	) {
		http.Error(
			w,
			"unsupported conversion",
			http.StatusBadRequest,
		)
		return
	}

	if err := s.acquireConversionSlot(
		r.Context(),
	); err != nil {
		if errors.Is(
			err,
			context.DeadlineExceeded,
		) {
			http.Error(
				w,
				"conversion queue full",
				http.StatusServiceUnavailable,
			)
		}

		return
	}
	defer s.releaseConversionSlot()

	outputPath := filepath.Join(
		tempDir,
		baseName+"."+target,
	)

	conversionCtx, cancel := context.WithTimeout(
		r.Context(),
		conversionTimeout,
	)
	defer cancel()

	switch format.Category {
	case media.CategoryImage:
		switch {
		case format.ID == "gif" &&
			(target == "mp4" || target == "webm"):
			err = s.ffmpeg.Convert(
				conversionCtx,
				inputPath,
				outputPath,
			)

		default:
			err = s.imageMagick.Convert(
				conversionCtx,
				inputPath,
				outputPath,
			)
		}

	case media.CategoryAudio, media.CategoryVideo:
		err = s.ffmpeg.Convert(
			conversionCtx,
			inputPath,
			outputPath,
		)

	case media.CategoryDocument:
		err = s.libreOffice.Convert(
			conversionCtx,
			inputPath,
			outputPath,
		)

	case media.CategoryMarkdown:
		switch target {
		case "html":
			err = converter.MarkdownToHTML(
				inputPath,
				outputPath,
			)

		default:
			http.Error(
				w,
				"unsupported Markdown conversion",
				http.StatusBadRequest,
			)
			return
		}

	case media.CategoryPDF:
		switch target {
		case "docx":
			err = s.pdf.ConvertToDOCX(
				conversionCtx,
				inputPath,
				outputPath,
			)

		case "png", "jpeg":
			outputPath = filepath.Join(
				tempDir,
				baseName+"-"+target+".zip",
			)

			err = s.pdf.ConvertToImages(
				conversionCtx,
				inputPath,
				outputPath,
				target,
			)

		default:
			http.Error(
				w,
				"unsupported PDF conversion",
				http.StatusBadRequest,
			)
			return
		}

	default:
		http.Error(
			w,
			"converter not implemented for this media type",
			http.StatusNotImplemented,
		)
		return
	}

	if errors.Is(
		conversionCtx.Err(),
		context.DeadlineExceeded,
	) {
		http.Error(
			w,
			"conversion timed out",
			http.StatusGatewayTimeout,
		)
		return
	}

	if errors.Is(
		conversionCtx.Err(),
		context.Canceled,
	) {
		return
	}

	if err != nil {
		http.Error(
			w,
			err.Error(),
			http.StatusInternalServerError,
		)
		return
	}

	outputFile, err := os.Open(outputPath)
	if err != nil {
		http.Error(
			w,
			"failed to open converted file",
			http.StatusInternalServerError,
		)
		return
	}
	defer outputFile.Close()

	stat, err := outputFile.Stat()
	if err != nil {
		http.Error(
			w,
			"failed to read converted file",
			http.StatusInternalServerError,
		)
		return
	}

	w.Header().Set(
		"Content-Disposition",
		fmt.Sprintf(
			`attachment; filename="%s"`,
			filepath.Base(outputPath),
		),
	)

	http.ServeContent(
		w,
		r,
		filepath.Base(outputPath),
		stat.ModTime(),
		outputFile,
	)
}
