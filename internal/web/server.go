// Package web provides HTTP server and handlers
package web

import (
	"context"
	"errors"
	"log/slog"
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
	logger *slog.Logger

	imageMagick *converter.ImageMagick
	ffmpeg      *converter.FFmpeg
	ffprobe     *media.FFProbe
	libreOffice *converter.LibreOffice
	pdf         *converter.PDF
	qpdf        *converter.QPDF
	ghostscript *converter.Ghostscript
	webPDF      *converter.WebPDF

	uploads    *uploadStore
	pdfUploads *pdfUploadStore
	downloads  *downloadStore

	conversionSlots chan struct{}

	mux        *http.ServeMux
	httpServer *http.Server
}

func NewServer(
	logger *slog.Logger,
	imageMagick *converter.ImageMagick,
	ffmpeg *converter.FFmpeg,
	ffprobe *media.FFProbe,
	libreOffice *converter.LibreOffice,
	pdf *converter.PDF,
	qpdf *converter.QPDF,
	ghostscript *converter.Ghostscript,
	webPDF *converter.WebPDF,
) *Server {
	if logger == nil {
		logger = defaultLogger()
	}

	server := &Server{
		logger: logger,

		imageMagick: imageMagick,
		ffmpeg:      ffmpeg,
		ffprobe:     ffprobe,
		libreOffice: libreOffice,
		pdf:         pdf,
		qpdf:        qpdf,
		ghostscript: ghostscript,
		webPDF:      webPDF,

		uploads:    newUploadStore(),
		pdfUploads: newPDFUploadStore(),
		downloads:  newDownloadStore(),

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
		"GET /downloads/{id}",
		s.handleDownload,
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
		"GET /pdf/tools/{tool}",
		s.handlePDFWorkspace,
	)

	s.mux.HandleFunc(
		"POST /pdf/uploads",
		s.handlePDFUpload,
	)

	s.mux.HandleFunc(
		"POST /pdf/uploads/raw",
		s.handlePDFRawUpload,
	)

	s.mux.HandleFunc(
		"GET /pdf/uploads/{id}/pages/{page}",
		s.handlePDFPreview,
	)

	s.mux.HandleFunc(
		"DELETE /pdf/uploads/{id}",
		s.handlePDFUploadDelete,
	)

	s.mux.HandleFunc(
		"POST /pdf/merge",
		s.handlePDFMerge,
	)

	s.mux.HandleFunc(
		"POST /pdf/sort",
		s.handlePDFSort,
	)

	s.mux.HandleFunc(
		"POST /pdf/delete-pages",
		s.handlePDFDeletePages,
	)

	s.mux.HandleFunc(
		"POST /pdf/extract-pages",
		s.handlePDFExtractPages,
	)

	s.mux.HandleFunc(
		"POST /pdf/rotate-pages",
		s.handlePDFRotatePages,
	)

	s.mux.HandleFunc(
		"POST /pdf/split",
		s.handlePDFSplit,
	)

	s.mux.HandleFunc(
		"POST /pdf/compress/analyze",
		s.handlePDFCompressionAnalyze,
	)

	s.mux.HandleFunc(
		"POST /pdf/compress",
		s.handlePDFCompress,
	)

	s.mux.HandleFunc(
		"POST /pdf/encrypt",
		s.handlePDFEncrypt,
	)

	s.mux.HandleFunc(
		"POST /pdf/decrypt",
		s.handlePDFDecrypt,
	)

	s.mux.HandleFunc(
		"POST /pdf/optimize/analyze",
		s.handlePDFOptimizeAnalyze,
	)

	s.mux.HandleFunc(
		"POST /pdf/optimize",
		s.handlePDFOptimize,
	)

	s.mux.HandleFunc(
		"POST /pdf/web",
		s.handlePDFWeb,
	)

	s.mux.HandleFunc(
		"POST /pdf/create",
		s.handlePDFCreate,
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
	s.httpServer = &http.Server{
		Addr: addr,

		Handler: s.requestLoggingMiddleware(
			s.mux,
		),

		ReadHeaderTimeout: readHeaderTimeout,

		ReadTimeout: readTimeout,

		IdleTimeout: idleTimeout,
	}

	s.logger.Info(
		"http server starting",
		"address",
		addr,
		"max_concurrent_conversions",
		maxConcurrentConversions,
		"conversion_queue_timeout",
		conversionQueueTimeout,
	)

	err := s.httpServer.ListenAndServe()

	if errors.Is(
		err,
		http.ErrServerClosed,
	) {
		s.logger.Info(
			"http server stopped",
			"address",
			addr,
		)

		return nil
	}

	if err != nil {
		s.logger.Error(
			"http server failed",
			"address",
			addr,
			"error",
			err,
		)
	}

	return err
}

func (s *Server) Shutdown(
	ctx context.Context,
) error {
	s.logger.Info(
		"http server shutdown started",
	)

	if s.httpServer == nil {
		s.closeStores()

		s.logger.Info(
			"http server shutdown completed",
		)

		return nil
	}

	err := s.httpServer.Shutdown(
		ctx,
	)

	s.closeStores()

	if err != nil {
		s.logger.Error(
			"http server shutdown failed",
			"error",
			err,
		)

		return err
	}

	s.logger.Info(
		"http server shutdown completed",
	)

	return nil
}

func (s *Server) Close() error {
	s.logger.Info(
		"http server forced close started",
	)

	s.closeStores()

	if s.httpServer == nil {
		s.logger.Info(
			"http server forced close completed",
		)

		return nil
	}

	err := s.httpServer.Close()

	if err != nil {
		s.logger.Error(
			"http server forced close failed",
			"error",
			err,
		)

		return err
	}

	s.logger.Info(
		"http server forced close completed",
	)

	return nil
}

func (s *Server) closeStores() {
	s.logger.Debug(
		"closing upload stores",
	)

	s.uploads.Close()
	s.pdfUploads.Close()
	s.downloads.Close()

	s.logger.Debug(
		"upload stores closed",
	)
}
