package web

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"path/filepath"
)

type conversionDownloadResponse struct {
	DownloadURL string `json:"download_url"`

	Filename string `json:"filename"`
}

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

	outputPath, outputName, err :=
		s.convertFile(
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

	return s.prepareConvertedDownload(
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

		outputPath, outputName, err :=
			s.convertFile(
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

	return s.prepareConvertedDownload(
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

func (s *Server) prepareConvertedDownload(
	w http.ResponseWriter,
	r *http.Request,
	path string,
	filename string,
) bool {
	download, err :=
		s.downloads.Add(
			path,
			filename,
		)

	if err != nil {
		s.logError(
			r,
			"download preparation failed",
			err,
			"filename",
			filename,
		)

		http.Error(
			w,
			"Download konnte nicht vorbereitet werden.",
			http.StatusInternalServerError,
		)

		return false
	}

	response :=
		conversionDownloadResponse{
			DownloadURL: "/downloads/" +
				download.ID,

			Filename: download.Filename,
		}

	w.Header().Set(
		"Content-Type",
		"application/json; charset=utf-8",
	)

	w.Header().Set(
		"Cache-Control",
		"no-store",
	)

	if err := json.NewEncoder(
		w,
	).Encode(
		response,
	); err != nil {
		s.logError(
			r,
			"download response failed",
			err,
			"filename",
			filename,
		)

		return false
	}

	return true
}
