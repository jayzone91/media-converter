package web

import (
	"context"
	"encoding/json"
	"errors"
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

	Markdown bool `json:"markdown"`
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
			"markdown",
			request.Markdown,
		)

		http.Error(
			w,
			"Das PDF-Dokument konnte nicht erstellt werden.",
			http.StatusInternalServerError,
		)

		return
	}

	writeCreatedPDF(
		s,
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
			request.Markdown,
		)

	if err :=
		validatePDFCreateDocument(
			document,
		); err != nil {
		if errors.Is(
			err,
			errPDFCreateMarkdownImage,
		) {
			http.Error(
				w,
				"Bilder sind im Markdown-Modus aus Sicherheitsgründen nicht erlaubt.",
				http.StatusBadRequest,
			)

			return request,
				pdfCreateDocument{},
				false
		}

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

func writeCreatedPDF(
	s *Server,
	w http.ResponseWriter,
	r *http.Request,
	path string,
	request pdfCreateRequest,
) {
	outputSize, err :=
		downloadFileSize(
			path,
		)

	if err != nil {
		http.Error(
			w,
			"Die erzeugte PDF konnte nicht gelesen werden.",
			http.StatusInternalServerError,
		)

		return
	}

	if !s.prepareDownloadResponse(
		w,
		r,
		path,
		"dokument.pdf",
	) {
		return
	}

	s.logInfo(
		"PDF document created",
		"paper_size",
		request.PaperSize,
		"landscape",
		request.Landscape,
		"markdown",
		request.Markdown,
		"output",
		outputSize,
	)
}
