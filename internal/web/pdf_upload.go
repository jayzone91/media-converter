package web

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	pdfUploadRequestOverhead int64 = 1 << 20

	pdfPreviewTimeout = 10 * time.Minute
)

type pdfUploadResponse struct {
	ID string `json:"id"`

	Filename string `json:"filename"`

	Size int64 `json:"size"`

	PageCount int `json:"page_count"`

	Previews []string `json:"previews"`
}

func (s *Server) handlePDFUpload(
	w http.ResponseWriter,
	r *http.Request,
) {
	r.Body = http.MaxBytesReader(
		w,
		r.Body,
		maxFileSize+
			pdfUploadRequestOverhead,
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
				"Die PDF ist größer als 512 MiB.",
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

	file, header, err :=
		r.FormFile("file")

	if err != nil {
		http.Error(
			w,
			"Keine PDF-Datei ausgewählt.",
			http.StatusBadRequest,
		)

		return
	}

	defer file.Close()

	if err := validatePDFUpload(
		header,
	); err != nil {
		http.Error(
			w,
			err.Error(),
			http.StatusBadRequest,
		)

		return
	}

	tempDir, err := os.MkdirTemp(
		"",
		"media-converter-pdf-*",
	)
	if err != nil {
		http.Error(
			w,
			"Temporäres Verzeichnis konnte nicht erstellt werden.",
			http.StatusInternalServerError,
		)

		return
	}

	keepTempDir := false

	defer func() {
		if !keepTempDir {
			_ = os.RemoveAll(
				tempDir,
			)
		}
	}()

	inputPath := filepath.Join(
		tempDir,
		"input.pdf",
	)

	size, err := savePDFUpload(
		file,
		inputPath,
	)
	if err != nil {
		http.Error(
			w,
			"PDF konnte nicht gespeichert werden.",
			http.StatusInternalServerError,
		)

		return
	}

	if err := validatePDFSignature(
		inputPath,
	); err != nil {
		http.Error(
			w,
			"Die Datei ist keine gültige PDF.",
			http.StatusBadRequest,
		)

		return
	}

	if err := s.acquireConversionSlot(
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

	ctx, cancel := context.WithTimeout(
		r.Context(),
		pdfPreviewTimeout,
	)

	defer cancel()

	previewDir := filepath.Join(
		tempDir,
		"previews",
	)

	previewPaths, err :=
		s.pdf.RenderPreviews(
			ctx,
			inputPath,
			previewDir,
		)

	if err != nil {
		if errors.Is(
			ctx.Err(),
			context.DeadlineExceeded,
		) {
			http.Error(
				w,
				"Die PDF-Vorschau konnte nicht rechtzeitig erstellt werden.",
				http.StatusGatewayTimeout,
			)

			return
		}

		http.Error(
			w,
			"Die PDF konnte nicht gelesen oder dargestellt werden.",
			http.StatusBadRequest,
		)

		return
	}

	upload, err :=
		s.pdfUploads.Add(
			tempDir,
			inputPath,
			header.Filename,
			size,
			previewPaths,
		)

	if err != nil {
		http.Error(
			w,
			"PDF-Upload konnte nicht gespeichert werden.",
			http.StatusInternalServerError,
		)

		return
	}

	keepTempDir = true

	previews := make(
		[]string,
		upload.PageCount(),
	)

	for index := range previews {
		previews[index] =
			fmt.Sprintf(
				"/pdf/uploads/%s/pages/%d",
				upload.ID,
				index+1,
			)
	}

	response := pdfUploadResponse{
		ID: upload.ID,

		Filename: upload.Filename,

		Size: upload.Size,

		PageCount: upload.PageCount(),

		Previews: previews,
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
		s.pdfUploads.Delete(
			upload.ID,
		)

		return
	}
}

func (s *Server) handlePDFPreview(
	w http.ResponseWriter,
	r *http.Request,
) {
	id := r.PathValue(
		"id",
	)

	page, err := strconv.Atoi(
		r.PathValue(
			"page",
		),
	)

	if err != nil ||
		page < 1 {
		http.NotFound(
			w,
			r,
		)

		return
	}

	upload, ok :=
		s.pdfUploads.Get(
			id,
		)

	if !ok {
		http.NotFound(
			w,
			r,
		)

		return
	}

	if page >
		len(upload.PreviewPaths) {
		http.NotFound(
			w,
			r,
		)

		return
	}

	previewPath :=
		upload.PreviewPaths[page-1]

	w.Header().Set(
		"Content-Type",
		"image/jpeg",
	)

	w.Header().Set(
		"Cache-Control",
		"private, max-age=1800",
	)

	http.ServeFile(
		w,
		r,
		previewPath,
	)
}

func (s *Server) handlePDFUploadDelete(
	w http.ResponseWriter,
	r *http.Request,
) {
	id := r.PathValue(
		"id",
	)

	if id == "" {
		http.NotFound(
			w,
			r,
		)

		return
	}

	if _, ok :=
		s.pdfUploads.Get(
			id,
		); !ok {
		http.NotFound(
			w,
			r,
		)

		return
	}

	s.pdfUploads.Delete(
		id,
	)

	w.WriteHeader(
		http.StatusNoContent,
	)
}

func validatePDFUpload(
	header *multipart.FileHeader,
) error {
	if header.Size <= 0 {
		return fmt.Errorf(
			"Die PDF-Datei ist leer.",
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
			"Es können nur PDF-Dateien hochgeladen werden.",
		)
	}

	return nil
}

func savePDFUpload(
	source multipart.File,
	path string,
) (int64, error) {
	destination, err :=
		os.OpenFile(
			path,
			os.O_WRONLY|
				os.O_CREATE|
				os.O_EXCL,
			0600,
		)

	if err != nil {
		return 0, err
	}

	limited := io.LimitReader(
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
		return 0, copyErr
	}

	if closeErr != nil {
		return 0, closeErr
	}

	if written >
		maxFileSize {
		return 0, fmt.Errorf(
			"PDF exceeds maximum size",
		)
	}

	return written, nil
}

func validatePDFSignature(
	path string,
) error {
	file, err := os.Open(
		path,
	)
	if err != nil {
		return err
	}

	defer file.Close()

	buffer := make(
		[]byte,
		1024,
	)

	n, err := file.Read(
		buffer,
	)

	if err != nil &&
		!errors.Is(
			err,
			io.EOF,
		) {
		return err
	}

	if !bytes.Contains(
		buffer[:n],
		[]byte("%PDF-"),
	) {
		return fmt.Errorf(
			"PDF signature not found",
		)
	}

	return nil
}
