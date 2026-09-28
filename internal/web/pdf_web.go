package web

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jayzone91/media-converter/internal/converter"
)

const (
	maxPDFWebRequestSize int64 = 64 << 10

	pdfWebTimeout = 2 * time.Minute

	maxPDFWebWait = 10 * time.Second
)

type pdfWebRequest struct {
	URL string `json:"url"`

	PaperSize string `json:"paper_size"`

	RenderMode string `json:"render_mode"`

	Landscape bool `json:"landscape"`

	PrintBackground bool `json:"print_background"`

	WaitMilliseconds int `json:"wait_ms"`
}

func (s *Server) handlePDFWeb(
	w http.ResponseWriter,
	r *http.Request,
) {
	request, parsedURL, options, ok :=
		readPDFWebRequest(
			w,
			r,
		)

	if !ok {
		return
	}

	if err :=
		s.acquireWorkload(
			r.Context(),
			workloadChromium,
		); err != nil {
		http.Error(
			w,
			"Der Server ist momentan ausgelastet. Bitte später erneut versuchen.",
			http.StatusServiceUnavailable,
		)

		return
	}

	defer s.releaseWorkload(
		workloadChromium,
	)

	tempDir, err :=
		os.MkdirTemp(
			"",
			"media-converter-web-pdf-*",
		)

	if err != nil {
		s.logError(
			r,
			"failed to create webpage PDF temporary directory",
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
		if err :=
			os.RemoveAll(
				tempDir,
			); err != nil {
			s.logError(
				r,
				"failed to remove webpage PDF temporary directory",
				err,
				"directory",
				tempDir,
			)
		}
	}()

	outputPath :=
		filepath.Join(
			tempDir,
			"webseite.pdf",
		)

	ctx, cancel :=
		context.WithTimeout(
			r.Context(),
			pdfWebTimeout,
		)

	defer cancel()

	if err :=
		s.webPDF.Render(
			ctx,
			request.URL,
			outputPath,
			options,
		); err != nil {
		if errors.Is(
			ctx.Err(),
			context.DeadlineExceeded,
		) {
			http.Error(
				w,
				"Die Webseite konnte nicht rechtzeitig geladen werden.",
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
			"webpage PDF rendering failed",
			err,
			"host",
			parsedURL.Hostname(),
			"paper_size",
			request.PaperSize,
			"render_mode",
			request.RenderMode,
			"landscape",
			request.Landscape,
		)

		http.Error(
			w,
			"Die Webseite konnte nicht als PDF erstellt werden.",
			http.StatusBadGateway,
		)

		return
	}

	outputSize, err :=
		downloadFileSize(
			outputPath,
		)

	if err != nil {
		s.logError(
			r,
			"failed to stat webpage PDF",
			err,
		)

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
		outputPath,
		"webseite.pdf",
	) {
		return
	}

	s.logInfo(
		"webpage PDF created",
		"host",
		parsedURL.Hostname(),
		"paper_size",
		request.PaperSize,
		"render_mode",
		request.RenderMode,
		"output",
		outputSize,
	)
}

func readPDFWebRequest(
	w http.ResponseWriter,
	r *http.Request,
) (
	pdfWebRequest,
	*url.URL,
	converter.WebPDFOptions,
	bool,
) {
	r.Body =
		http.MaxBytesReader(
			w,
			r.Body,
			maxPDFWebRequestSize,
		)

	defer r.Body.Close()

	var request pdfWebRequest

	decoder :=
		json.NewDecoder(
			r.Body,
		)

	decoder.DisallowUnknownFields()

	if err :=
		decoder.Decode(
			&request,
		); err != nil {
		http.Error(
			w,
			"Ungültige Anfrage.",
			http.StatusBadRequest,
		)

		return request,
			nil,
			converter.WebPDFOptions{},
			false
	}

	request.URL =
		strings.TrimSpace(
			request.URL,
		)

	parsedURL, err :=
		validatePDFWebURL(
			request.URL,
		)

	if err != nil {
		http.Error(
			w,
			"Bitte eine gültige HTTP- oder HTTPS-Adresse eingeben.",
			http.StatusBadRequest,
		)

		return request,
			nil,
			converter.WebPDFOptions{},
			false
	}

	if err :=
		validatePDFWebTarget(
			r.Context(),
			parsedURL,
		); err != nil {
		http.Error(
			w,
			pdfWebTargetError(
				r,
				parsedURL,
			),
			http.StatusBadRequest,
		)

		return request,
			nil,
			converter.WebPDFOptions{},
			false
	}

	request.PaperSize =
		strings.ToLower(
			strings.TrimSpace(
				request.PaperSize,
			),
		)

	if request.PaperSize == "" {
		request.PaperSize =
			"a4"
	}

	switch request.PaperSize {
	case "a4", "letter":
	default:
		http.Error(
			w,
			"Ungültiges Papierformat.",
			http.StatusBadRequest,
		)

		return request,
			nil,
			converter.WebPDFOptions{},
			false
	}

	request.RenderMode =
		strings.ToLower(
			strings.TrimSpace(
				request.RenderMode,
			),
		)

	if request.RenderMode == "" {
		request.RenderMode =
			"desktop"
	}

	switch request.RenderMode {
	case
		"desktop",
		"tablet",
		"mobile",
		"print":

	default:
		http.Error(
			w,
			"Ungültige Darstellungsart.",
			http.StatusBadRequest,
		)

		return request,
			nil,
			converter.WebPDFOptions{},
			false
	}

	if request.WaitMilliseconds < 0 ||
		time.Duration(
			request.WaitMilliseconds,
		)*time.Millisecond >
			maxPDFWebWait {
		http.Error(
			w,
			"Die zusätzliche Wartezeit darf maximal 10 Sekunden betragen.",
			http.StatusBadRequest,
		)

		return request,
			nil,
			converter.WebPDFOptions{},
			false
	}

	options :=
		converter.WebPDFOptions{
			PaperSize: request.PaperSize,

			RenderMode: request.RenderMode,

			Landscape: request.Landscape,

			PrintBackground: request.PrintBackground,

			Wait: time.Duration(
				request.WaitMilliseconds,
			) *
				time.Millisecond,

			RequestValidator: validatePDFWebRequest,
		}

	return request,
		parsedURL,
		options,
		true
}
