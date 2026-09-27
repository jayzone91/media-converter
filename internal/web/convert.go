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
)

func (s *Server) handleConvert(
	w http.ResponseWriter,
	r *http.Request,
) {
	contentType := r.Header.Get(
		"Content-Type",
	)

	if !strings.HasPrefix(
		contentType,
		"application/x-www-form-urlencoded",
	) {
		http.Error(
			w,
			"direct conversion upload is not supported",
			http.StatusBadRequest,
		)

		return
	}

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

	defer os.RemoveAll(
		upload.Directory,
	)

	if len(upload.Files) == 0 {
		http.Error(
			w,
			"upload contains no files",
			http.StatusBadRequest,
		)

		return
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
		upload.Format.Targets,
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

	conversionCtx, cancel := context.WithTimeout(
		r.Context(),
		conversionTimeout,
	)
	defer cancel()

	if len(upload.Files) == 1 {
		s.convertSingleUpload(
			w,
			r,
			conversionCtx,
			upload,
			target,
		)

		return
	}

	s.convertBatchUpload(
		w,
		r,
		conversionCtx,
		upload,
		target,
	)
}

func (s *Server) convertSingleUpload(
	w http.ResponseWriter,
	r *http.Request,
	ctx context.Context,
	upload storedUpload,
	target string,
) {
	file := upload.Files[0]

	outputDir := filepath.Join(
		upload.Directory,
		"output",
	)

	outputPath, outputName, err := s.convertFile(
		ctx,
		upload.Format,
		target,
		file.Path,
		outputDir,
		file.Filename,
	)

	if !handleConversionError(
		w,
		ctx,
		err,
	) {
		return
	}

	serveConvertedFile(
		w,
		r,
		outputPath,
		outputName,
	)
}

func (s *Server) convertBatchUpload(
	w http.ResponseWriter,
	r *http.Request,
	ctx context.Context,
	upload storedUpload,
	target string,
) {
	archiveFiles := make(
		[]convertedArchiveFile,
		0,
		len(upload.Files),
	)

	for index, file := range upload.Files {
		outputDir := filepath.Join(
			upload.Directory,
			"output",
			fmt.Sprintf(
				"%03d",
				index+1,
			),
		)

		outputPath, outputName, err := s.convertFile(
			ctx,
			upload.Format,
			target,
			file.Path,
			outputDir,
			file.Filename,
		)

		if !handleConversionError(
			w,
			ctx,
			err,
		) {
			return
		}

		archiveFiles = append(
			archiveFiles,
			convertedArchiveFile{
				Path: outputPath,
				Name: outputName,
			},
		)
	}

	archiveName := fmt.Sprintf(
		"converted-%s.zip",
		target,
	)

	archivePath := filepath.Join(
		upload.Directory,
		archiveName,
	)

	if err := createConversionArchive(
		archivePath,
		archiveFiles,
	); err != nil {
		http.Error(
			w,
			err.Error(),
			http.StatusInternalServerError,
		)

		return
	}

	serveConvertedFile(
		w,
		r,
		archivePath,
		archiveName,
	)
}

func handleConversionError(
	w http.ResponseWriter,
	ctx context.Context,
	err error,
) bool {
	if errors.Is(
		ctx.Err(),
		context.DeadlineExceeded,
	) {
		http.Error(
			w,
			"conversion timed out",
			http.StatusGatewayTimeout,
		)

		return false
	}

	if errors.Is(
		ctx.Err(),
		context.Canceled,
	) {
		return false
	}

	if err != nil {
		http.Error(
			w,
			err.Error(),
			http.StatusInternalServerError,
		)

		return false
	}

	return true
}

func serveConvertedFile(
	w http.ResponseWriter,
	r *http.Request,
	path string,
	filename string,
) {
	outputFile, err := os.Open(path)
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
			filepath.Base(filename),
		),
	)

	http.ServeContent(
		w,
		r,
		filepath.Base(filename),
		stat.ModTime(),
		outputFile,
	)
}
