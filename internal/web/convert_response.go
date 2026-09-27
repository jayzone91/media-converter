package web

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
)

func (s *Server) convertSingleUpload(
	w http.ResponseWriter,
	r *http.Request,
	ctx context.Context,
	upload storedUpload,
	target string,
) bool {
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

	if !s.handleConversionError(
		w,
		r,
		ctx,
		err,
		upload.Format.ID,
		target,
		file.Filename,
	) {
		return false
	}

	return s.serveConvertedFile(
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
) bool {
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

		if !s.handleConversionError(
			w,
			r,
			ctx,
			err,
			upload.Format.ID,
			target,
			file.Filename,
		) {
			return false
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
		s.logError(
			r,
			"conversion archive failed",
			err,
			"source",
			upload.Format.ID,
			"target",
			target,
			"files",
			len(upload.Files),
		)

		http.Error(
			w,
			"failed to create conversion archive",
			http.StatusInternalServerError,
		)

		return false
	}

	return s.serveConvertedFile(
		w,
		r,
		archivePath,
		archiveName,
	)
}

func (s *Server) handleConversionError(
	w http.ResponseWriter,
	r *http.Request,
	ctx context.Context,
	err error,
	source string,
	target string,
	filename string,
) bool {
	if errors.Is(
		ctx.Err(),
		context.DeadlineExceeded,
	) {
		s.logError(
			r,
			"conversion timed out",
			ctx.Err(),
			"source",
			source,
			"target",
			target,
			"file",
			filename,
		)

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
		s.logWarn(
			r,
			"conversion canceled",
			"source",
			source,
			"target",
			target,
			"file",
			filename,
		)

		return false
	}

	if err != nil {
		s.logError(
			r,
			"conversion failed",
			err,
			"source",
			source,
			"target",
			target,
			"file",
			filename,
		)

		http.Error(
			w,
			"conversion failed",
			http.StatusInternalServerError,
		)

		return false
	}

	return true
}

func (s *Server) serveConvertedFile(
	w http.ResponseWriter,
	r *http.Request,
	path string,
	filename string,
) bool {
	outputFile, err := os.Open(path)
	if err != nil {
		s.logError(
			r,
			"converted file open failed",
			err,
			"file",
			filename,
		)

		http.Error(
			w,
			"failed to open converted file",
			http.StatusInternalServerError,
		)

		return false
	}
	defer outputFile.Close()

	stat, err := outputFile.Stat()
	if err != nil {
		s.logError(
			r,
			"converted file stat failed",
			err,
			"file",
			filename,
		)

		http.Error(
			w,
			"failed to read converted file",
			http.StatusInternalServerError,
		)

		return false
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

	return true
}
