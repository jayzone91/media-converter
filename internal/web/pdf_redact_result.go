package web

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
)

type pdfRedactResponse struct {
	DownloadURL string `json:"download_url"`
	Filename    string `json:"filename"`

	PreviewUploadID string   `json:"preview_upload_id"`
	Previews        []string `json:"previews"`
	PageCount       int      `json:"page_count"`
}

func (s *Server) preparePDFRedactResult(
	w http.ResponseWriter,
	r *http.Request,
	sourcePath string,
	size int64,
	pageCount int,
) (storedPDFUpload, bool) {
	resultUpload, err :=
		s.createPDFRedactResultUpload(
			sourcePath,
			size,
			pageCount,
		)
	if err != nil {
		s.logError(
			r,
			"PDF redaction result preview preparation failed",
			err,
		)

		http.Error(
			w,
			"Die Ergebnisvorschau konnte nicht vorbereitet werden.",
			http.StatusInternalServerError,
		)

		return storedPDFUpload{}, false
	}

	download, err :=
		s.downloads.Add(
			sourcePath,
			"geschwaerzt.pdf",
		)
	if err != nil {
		s.pdfUploads.Delete(
			resultUpload.ID,
		)

		s.logError(
			r,
			"PDF redaction download preparation failed",
			err,
		)

		http.Error(
			w,
			"Download konnte nicht vorbereitet werden.",
			http.StatusInternalServerError,
		)

		return storedPDFUpload{}, false
	}

	response :=
		pdfRedactResponse{
			DownloadURL: "/downloads/" +
				download.ID,

			Filename: download.Filename,

			PreviewUploadID: resultUpload.ID,

			Previews: buildPDFPreviewURLs(
				resultUpload,
			),

			PageCount: resultUpload.PageCount,
		}

	w.Header().Set(
		"Content-Type",
		"application/json; charset=utf-8",
	)

	w.Header().Set(
		"Cache-Control",
		"no-store",
	)

	if err :=
		json.NewEncoder(
			w,
		).Encode(
			response,
		); err != nil {
		s.logError(
			r,
			"PDF redaction result response failed",
			err,
		)

		return resultUpload, false
	}

	return resultUpload, true
}

func (s *Server) createPDFRedactResultUpload(
	sourcePath string,
	size int64,
	pageCount int,
) (storedPDFUpload, error) {
	directory, err :=
		os.MkdirTemp(
			"",
			"media-converter-pdf-redact-result-*",
		)
	if err != nil {
		return storedPDFUpload{},
			fmt.Errorf(
				"create redaction result directory: %w",
				err,
			)
	}

	keepDirectory := false

	defer func() {
		if !keepDirectory {
			_ = os.RemoveAll(
				directory,
			)
		}
	}()

	resultPath :=
		filepath.Join(
			directory,
			"result.pdf",
		)

	if err :=
		copyPDFRedactResult(
			sourcePath,
			resultPath,
		); err != nil {
		return storedPDFUpload{},
			err
	}

	previewDirectory :=
		filepath.Join(
			directory,
			"previews",
		)

	if err :=
		os.MkdirAll(
			previewDirectory,
			0700,
		); err != nil {
		return storedPDFUpload{},
			fmt.Errorf(
				"create redaction preview directory: %w",
				err,
			)
	}

	upload, err :=
		s.pdfUploads.Add(
			directory,
			resultPath,
			previewDirectory,
			"geschwaerzt.pdf",
			size,
			pageCount,
		)
	if err != nil {
		return storedPDFUpload{},
			fmt.Errorf(
				"register redaction result: %w",
				err,
			)
	}

	keepDirectory = true

	return upload, nil
}

func copyPDFRedactResult(
	sourcePath string,
	destinationPath string,
) error {
	source, err :=
		os.Open(
			sourcePath,
		)
	if err != nil {
		return fmt.Errorf(
			"open redaction result: %w",
			err,
		)
	}

	defer source.Close()

	destination, err :=
		os.OpenFile(
			destinationPath,
			os.O_CREATE|
				os.O_WRONLY|
				os.O_TRUNC,
			0600,
		)
	if err != nil {
		return fmt.Errorf(
			"create redaction result copy: %w",
			err,
		)
	}

	_, copyErr :=
		io.Copy(
			destination,
			source,
		)

	closeErr :=
		destination.Close()

	if copyErr != nil {
		return fmt.Errorf(
			"copy redaction result: %w",
			copyErr,
		)
	}

	if closeErr != nil {
		return fmt.Errorf(
			"close redaction result copy: %w",
			closeErr,
		)
	}

	return nil
}
