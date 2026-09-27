package main

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jayzone91/media-converter/internal/converter"
	"github.com/jayzone91/media-converter/internal/logging"
	"github.com/jayzone91/media-converter/internal/media"
	"github.com/jayzone91/media-converter/internal/web"
)

const shutdownTimeout = 2 * time.Minute

func main() {
	logger := slog.New(
		logging.NewPrettyHandler(
			os.Stdout,
			&logging.Options{
				Level: slog.LevelInfo,

				Color: logging.SupportsColor(
					os.Stdout,
				),
			},
		),
	)

	slog.SetDefault(
		logger,
	)

	imageMagick, err := converter.NewImageMagick()
	if err != nil {
		logger.Error(
			"failed to initialize ImageMagick",
			"error",
			err,
		)

		os.Exit(1)
	}

	ffmpeg, err := converter.NewFFmpeg()
	if err != nil {
		logger.Error(
			"failed to initialize FFmpeg",
			"error",
			err,
		)

		os.Exit(1)
	}

	ffprobe, err := media.NewFFProbe()
	if err != nil {
		logger.Error(
			"failed to initialize ffprobe",
			"error",
			err,
		)

		os.Exit(1)
	}

	libreOffice, err := converter.NewLibreOffice()
	if err != nil {
		logger.Error(
			"failed to initialize LibreOffice",
			"error",
			err,
		)

		os.Exit(1)
	}

	pdf, err := converter.NewPDF(
		libreOffice,
	)
	if err != nil {
		logger.Error(
			"failed to initialize PDF converter",
			"error",
			err,
		)

		os.Exit(1)
	}

	qpdf, err := converter.NewQPDF()
	if err != nil {
		logger.Error(
			"failed to initialize qpdf",
			"error",
			err,
		)

		os.Exit(1)
	}

	ghostscript, err := converter.NewGhostscript()
	if err != nil {
		logger.Error(
			"failed to initialize Ghostscript",
			"error",
			err,
		)

		os.Exit(1)
	}

	server := web.NewServer(
		logger,
		imageMagick,
		ffmpeg,
		ffprobe,
		libreOffice,
		pdf,
		qpdf,
		ghostscript,
	)

	signalCtx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()

	serverError := make(
		chan error,
		1,
	)

	go func() {
		logger.Info(
			"server listening",
			"address",
			":8080",
		)

		serverError <- server.ListenAndServe(
			":8080",
		)
	}()

	select {
	case err := <-serverError:
		if err != nil {
			logger.Error(
				"server stopped unexpectedly",
				"error",
				err,
			)

			os.Exit(1)
		}

	case <-signalCtx.Done():
		logger.Info(
			"server shutting down",
		)

		shutdownCtx, cancel := context.WithTimeout(
			context.Background(),
			shutdownTimeout,
		)
		defer cancel()

		if err := server.Shutdown(
			shutdownCtx,
		); err != nil {
			if !errors.Is(
				err,
				context.DeadlineExceeded,
			) {
				logger.Error(
					"graceful shutdown failed",
					"error",
					err,
				)
			}

			if err := server.Close(); err != nil {
				logger.Error(
					"forced shutdown failed",
					"error",
					err,
				)
			}
		}

		logger.Info(
			"server stopped",
		)
	}
}
