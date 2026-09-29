package web

import (
	"context"
	"fmt"
	"net/http"
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

	return s.prepareDownloadResponse(
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

		writeAPIError(
			w,
			http.StatusInternalServerError,
			apiErrorInternal,
			"Das Konvertierungsarchiv konnte nicht erstellt werden.",
		)

		return false
	}

	return s.prepareDownloadResponse(
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
	if writeTimeoutAPIError(
		w,
		ctx,
		"Die Konvertierung hat zu lange gedauert.",
	) {
		if ctx.Err() ==
			context.DeadlineExceeded {
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
		} else {
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
		}

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

		writeAPIError(
			w,
			http.StatusInternalServerError,
			apiErrorInternal,
			"Die Konvertierung ist fehlgeschlagen.",
		)

		return false
	}

	return true
}
