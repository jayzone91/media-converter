package web

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jayzone91/media-converter/internal/converter"
)

const (
	maxPDFCreateRequestSize int64 = 256 << 10

	pdfCreateTimeout = 2 * time.Minute
)

type pdfCreateRequest struct {
	Title string `json:"title"`

	Body string `json:"body"`

	PaperSize string `json:"paper_size"`

	Landscape bool `json:"landscape"`

	FontSize int `json:"font_size"`

	IncludeDate bool `json:"include_date"`
}

func (s *Server) handlePDFCreate(
	w http.ResponseWriter,
	r *http.Request,
) {
	request, document, ok :=
		readPDFCreateRequest(
			w,
			r,
		)

	if !ok {
		return
	}

	html, err :=
		createPDFDocumentHTML(
			document,
		)

	if err != nil {
		s.logError(
			r,
			"failed to create PDF document HTML",
			err,
		)

		http.Error(
			w,
			"Das PDF-Dokument konnte nicht vorbereitet werden.",
			http.StatusInternalServerError,
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

	tempDir, err :=
		os.MkdirTemp(
			"",
			"media-converter-pdf-create-*",
		)

	if err != nil {
		s.logError(
			r,
			"failed to create PDF document temporary directory",
			err,
		)

		http.Error(
			w,
			"Temporäres Verzeichnis konnte nicht erstellt werden.",
			http.StatusInternalServerError,
		)
		return
	}

	defer func() {
		if err := os.RemoveAll(
			tempDir,
		); err != nil {
			s.logError(
				r,
				"failed to remove PDF document temporary directory",
				err,
				"directory",
				tempDir,
			)
		}
	}()

	outputPath :=
		filepath.Join(
			tempDir,
			"dokument.pdf",
		)

	ctx, cancel :=
		context.WithTimeout(
			r.Context(),
			pdfCreateTimeout,
		)
	defer cancel()

	options :=
		converter.WebPDFOptions{
			PaperSize: request.PaperSize,

			RenderMode: "print",

			Landscape: request.Landscape,

			PrintBackground: true,
		}

	if err := s.webPDF.RenderHTML(
		ctx,
		html,
		outputPath,
		options,
	); err != nil {
		if errors.Is(
			ctx.Err(),
			context.DeadlineExceeded,
		) {
			http.Error(
				w,
				"Die PDF-Erstellung hat zu lange gedauert.",
				http.StatusGatewayTimeout,
			)
			return
		}

		if errors.Is(
			ctx.Err(),
			context.Canceled,
		) {
			return
		}

		s.logError(
			r,
			"PDF document rendering failed",
			err,
			"paper_size",
			request.PaperSize,
			"landscape",
			request.Landscape,
		)

		http.Error(
			w,
			"Das PDF-Dokument konnte nicht erstellt werden.",
			http.StatusInternalServerError,
		)
		return
	}

	s.writeCreatedPDF(
		w,
		r,
		outputPath,
		request,
	)
}

func readPDFCreateRequest(
	w http.ResponseWriter,
	r *http.Request,
) (
	pdfCreateRequest,
	pdfCreateDocument,
	bool,
) {
	r.Body = http.MaxBytesReader(
		w,
		r.Body,
		maxPDFCreateRequestSize,
	)
	defer r.Body.Close()

	var request pdfCreateRequest

	decoder :=
		json.NewDecoder(
			r.Body,
		)

	decoder.DisallowUnknownFields()

	if err := decoder.Decode(
		&request,
	); err != nil {
		http.Error(
			w,
			"Ungültige Anfrage.",
			http.StatusBadRequest,
		)

		return request,
			pdfCreateDocument{},
			false
	}

	request.PaperSize =
		strings.ToLower(
			strings.TrimSpace(
				request.PaperSize,
			),
		)

	switch request.PaperSize {
	case "a4", "letter":
	default:
		http.Error(
			w,
			"Ungültiges Papierformat.",
			http.StatusBadRequest,
		)

		return request,
			pdfCreateDocument{},
			false
	}

	document :=
		newPDFCreateDocument(
			request.Title,
			request.Body,
			request.FontSize,
			request.IncludeDate,
		)

	if err :=
		validatePDFCreateDocument(
			document,
		); err != nil {
		http.Error(
			w,
			"Das Dokument ist leer oder enthält ungültige Angaben.",
			http.StatusBadRequest,
		)

		return request,
			pdfCreateDocument{},
			false
	}

	return request,
		document,
		true
}

func (s *Server) writeCreatedPDF(
	w http.ResponseWriter,
	r *http.Request,
	path string,
	request pdfCreateRequest,
) {
	file, err :=
		os.Open(
			path,
		)

	if err != nil {
		http.Error(
			w,
			"Die erzeugte PDF konnte nicht geöffnet werden.",
			http.StatusInternalServerError,
		)
		return
	}
	defer file.Close()

	info, err :=
		file.Stat()

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
				"filename": "dokument.pdf",
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

	w.Header().Set(
		"Cache-Control",
		"no-store",
	)

	if _, err := io.Copy(
		w,
		file,
	); err != nil {
		s.logError(
			r,
			"failed to send created PDF",
			err,
		)

		return
	}

	s.logger.Info(
		"PDF document created",
		"method",
		r.Method,
		"path",
		r.URL.Path,
		"paper_size",
		request.PaperSize,
		"landscape",
		request.Landscape,
		"font_size",
		request.FontSize,
		"include_date",
		request.IncludeDate,
		"output_size_bytes",
		info.Size(),
	)
}
