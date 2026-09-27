package web

import (
	"context"
	"errors"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	maxPDFMergeFiles = 50

	maxPDFMergeTotalSize int64 = 1024 << 20

	pdfMergeRequestOverhead int64 = 16 << 20

	pdfMergeTimeout = 30 * time.Minute
)

func (s *Server) handlePDFMerge(
	w http.ResponseWriter,
	r *http.Request,
) {
	r.Body = http.MaxBytesReader(
		w,
		r.Body,
		maxPDFMergeTotalSize+
			pdfMergeRequestOverhead,
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
				"Die hochgeladenen PDFs sind zusammen zu groß.",
				http.StatusRequestEntityTooLarge,
			)

			return
		}

		http.Error(
			w,
			"Ungültiger Upload.",
			http.StatusBadRequest,
		)

		return
	}

	if r.MultipartForm != nil {
		defer r.MultipartForm.RemoveAll()
	}

	headers :=
		r.MultipartForm.File["files"]

	if len(headers) < 2 {
		http.Error(
			w,
			"Bitte mindestens zwei PDF-Dateien auswählen.",
			http.StatusBadRequest,
		)

		return
	}

	if len(headers) > maxPDFMergeFiles {
		http.Error(
			w,
			fmt.Sprintf(
				"Es können maximal %d PDFs gleichzeitig zusammengefügt werden.",
				maxPDFMergeFiles,
			),
			http.StatusBadRequest,
		)

		return
	}

	tempDir, err := os.MkdirTemp(
		"",
		"media-converter-pdf-merge-*",
	)
	if err != nil {
		http.Error(
			w,
			"Temporäres Verzeichnis konnte nicht erstellt werden.",
			http.StatusInternalServerError,
		)

		return
	}
	defer os.RemoveAll(tempDir)

	inputPaths := make(
		[]string,
		0,
		len(headers),
	)

	var totalSize int64

	for index, header := range headers {
		if err := validatePDFMergeFile(
			header,
		); err != nil {
			http.Error(
				w,
				err.Error(),
				http.StatusBadRequest,
			)

			return
		}

		totalSize += header.Size

		if totalSize >
			maxPDFMergeTotalSize {
			http.Error(
				w,
				"Die PDFs dürfen zusammen maximal 1 GiB groß sein.",
				http.StatusRequestEntityTooLarge,
			)

			return
		}

		path, err :=
			savePDFMergeUpload(
				header,
				tempDir,
				index,
			)
		if err != nil {
			http.Error(
				w,
				"Eine PDF-Datei konnte nicht gespeichert werden.",
				http.StatusInternalServerError,
			)

			return
		}

		inputPaths = append(
			inputPaths,
			path,
		)
	}

	if err :=
		s.acquireConversionSlot(
			r.Context(),
		); err != nil {
		http.Error(
			w,
			"Der Server ist momentan ausgelastet. Bitte später erneut versuchen.",
			http.StatusServiceUnavailable,
		)

		return
	}
	defer s.releaseConversionSlot()

	ctx, cancel :=
		context.WithTimeout(
			r.Context(),
			pdfMergeTimeout,
		)
	defer cancel()

	outputPath :=
		filepath.Join(
			tempDir,
			"zusammengefuegt.pdf",
		)

	if err :=
		s.qpdf.Merge(
			ctx,
			inputPaths,
			outputPath,
		); err != nil {
		if errors.Is(
			ctx.Err(),
			context.DeadlineExceeded,
		) {
			http.Error(
				w,
				"Das Zusammenfügen der PDFs hat zu lange gedauert.",
				http.StatusGatewayTimeout,
			)

			return
		}

		http.Error(
			w,
			"Die PDF-Dateien konnten nicht zusammengefügt werden.",
			http.StatusInternalServerError,
		)

		return
	}

	file, err :=
		os.Open(outputPath)
	if err != nil {
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
				"filename": "zusammengefuegt.pdf",
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

	if _, err :=
		io.Copy(
			w,
			file,
		); err != nil {
		return
	}
}

func validatePDFMergeFile(
	header *multipart.FileHeader,
) error {
	if header.Size <= 0 {
		return fmt.Errorf(
			"Eine der PDF-Dateien ist leer.",
		)
	}

	if header.Size >
		maxFileSize {
		return fmt.Errorf(
			"Eine einzelne PDF darf maximal 512 MiB groß sein.",
		)
	}

	extension :=
		strings.ToLower(
			filepath.Ext(
				header.Filename,
			),
		)

	if extension != ".pdf" {
		return fmt.Errorf(
			"Es können nur PDF-Dateien zusammengefügt werden.",
		)
	}

	return nil
}

func savePDFMergeUpload(
	header *multipart.FileHeader,
	tempDir string,
	index int,
) (string, error) {
	source, err :=
		header.Open()
	if err != nil {
		return "", err
	}
	defer source.Close()

	path :=
		filepath.Join(
			tempDir,
			fmt.Sprintf(
				"%03d.pdf",
				index,
			),
		)

	destination, err :=
		os.Create(path)
	if err != nil {
		return "", err
	}

	limited :=
		io.LimitReader(
			source,
			maxFileSize+1,
		)

	written, copyErr :=
		io.Copy(
			destination,
			limited,
		)

	closeErr :=
		destination.Close()

	if copyErr != nil {
		return "", copyErr
	}

	if closeErr != nil {
		return "", closeErr
	}

	if written > maxFileSize {
		_ = os.Remove(path)

		return "", fmt.Errorf(
			"file exceeds maximum size",
		)
	}

	return path, nil
}
