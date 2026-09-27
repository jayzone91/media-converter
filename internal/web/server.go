// Package web provides HTTP server and handlers
package web

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/jayzone91/media-converter/internal/converter"
	"github.com/jayzone91/media-converter/internal/media"
)

const (
	maxConcurrentConversions = 2
	conversionQueueTimeout   = 60 * time.Second

	readHeaderTimeout = 10 * time.Second
	readTimeout       = 15 * time.Minute
	idleTimeout       = 2 * time.Minute
)

type Server struct {
	imageMagick *converter.ImageMagick
	ffmpeg      *converter.FFmpeg
	ffprobe     *media.FFProbe
	libreOffice *converter.LibreOffice
	pdf         *converter.PDF
	qpdf        *converter.QPDF

	uploads *uploadStore

	conversionSlots chan struct{}

	mux        *http.ServeMux
	httpServer *http.Server
}

func NewServer(
	imageMagick *converter.ImageMagick,
	ffmpeg *converter.FFmpeg,
	ffprobe *media.FFProbe,
	libreOffice *converter.LibreOffice,
	pdf *converter.PDF,
	qpdf *converter.QPDF,
) *Server {
	server := &Server{
		imageMagick: imageMagick,
		ffmpeg:      ffmpeg,
		ffprobe:     ffprobe,
		libreOffice: libreOffice,
		pdf:         pdf,
		qpdf:        qpdf,

		uploads: newUploadStore(),

		conversionSlots: make(
			chan struct{},
			maxConcurrentConversions,
		),

		mux: http.NewServeMux(),
	}

	server.routes()

	return server
}

func (s *Server) routes() {
	s.mux.HandleFunc(
		"GET /",
		s.handleIndex,
	)

	s.mux.HandleFunc(
		"POST /detect",
		s.handleDetect,
	)

	s.mux.HandleFunc(
		"POST /convert",
		s.handleConvert,
	)

	s.mux.HandleFunc(
		"GET /qr/fields",
		s.handleQRFields,
	)

	s.mux.HandleFunc(
		"POST /qr/generate",
		s.handleQRGenerate,
	)

	s.mux.HandleFunc(
		"POST /pdf/merge",
		s.handlePDFMerge,
	)

	s.mux.Handle(
		"GET /static/",
		http.StripPrefix(
			"/static/",
			staticHandler(),
		),
	)
}

func (s *Server) acquireConversionSlot(
	ctx context.Context,
) error {
	queueCtx, cancel :=
		context.WithTimeout(
			ctx,
			conversionQueueTimeout,
		)

	defer cancel()

	select {
	case s.conversionSlots <- struct{}{}:
		return nil

	case <-queueCtx.Done():
		return queueCtx.Err()
	}
}

func (s *Server) releaseConversionSlot() {
	<-s.conversionSlots
}

func (s *Server) ListenAndServe(
	addr string,
) error {
	s.httpServer =
		&http.Server{
			Addr: addr,

			Handler: s.mux,

			ReadHeaderTimeout: readHeaderTimeout,

			ReadTimeout: readTimeout,

			IdleTimeout: idleTimeout,
		}

	err :=
		s.httpServer.
			ListenAndServe()

	if errors.Is(
		err,
		http.ErrServerClosed,
	) {
		return nil
	}

	return err
}

func (s *Server) Shutdown(
	ctx context.Context,
) error {
	if s.httpServer == nil {
		s.uploads.Close()

		return nil
	}

	err :=
		s.httpServer.
			Shutdown(ctx)

	s.uploads.Close()

	return err
}

func (s *Server) Close() error {
	s.uploads.Close()

	if s.httpServer == nil {
		return nil
	}

	return s.httpServer.Close()
}
