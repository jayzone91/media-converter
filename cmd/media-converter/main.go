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
	"github.com/jayzone91/media-converter/internal/dependencies"
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

	if !checkDependencies(
		logger,
	) {
		os.Exit(1)
	}

	imageMagick, err :=
		converter.NewImageMagick()

	if err != nil {
		startupError(
			logger,
			"ImageMagick",
			err,
		)
	}

	ffmpeg, err :=
		converter.NewFFmpeg()

	if err != nil {
		startupError(
			logger,
			"FFmpeg",
			err,
		)
	}

	ffprobe, err :=
		media.NewFFProbe()

	if err != nil {
		startupError(
			logger,
			"ffprobe",
			err,
		)
	}

	libreOffice, err :=
		converter.NewLibreOffice()

	if err != nil {
		startupError(
			logger,
			"LibreOffice",
			err,
		)
	}

	pdf, err :=
		converter.NewPDF(
			libreOffice,
		)

	if err != nil {
		startupError(
			logger,
			"PDF converter",
			err,
		)
	}

	qpdf, err :=
		converter.NewQPDF()

	if err != nil {
		startupError(
			logger,
			"qpdf",
			err,
		)
	}

	ghostscript, err :=
		converter.NewGhostscript()

	if err != nil {
		startupError(
			logger,
			"Ghostscript",
			err,
		)
	}

	webPDF, err :=
		converter.NewWebPDF()

	if err != nil {
		startupError(
			logger,
			"browser PDF converter",
			err,
		)
	}

	server :=
		web.NewServer(
			logger,
			imageMagick,
			ffmpeg,
			ffprobe,
			libreOffice,
			pdf,
			qpdf,
			ghostscript,
			webPDF,
		)

	runServer(
		logger,
		server,
	)
}

func checkDependencies(
	logger *slog.Logger,
) bool {
	ctx, cancel :=
		context.WithTimeout(
			context.Background(),
			30*time.Second,
		)
	defer cancel()

	tools, err :=
		dependencies.CheckAll(
			ctx,
		)

	for _, tool := range tools {
		logger.Info(
			"dependency",
			"name",
			tool.Name,
			"version",
			tool.Version,
			"path",
			tool.Path,
		)
	}

	if err != nil {
		logger.Error(
			"dependency check failed",
			"error",
			err,
		)

		return false
	}

	logger.Info(
		"dependency check completed",
		"tools",
		len(tools),
	)

	return true
}

func startupError(
	logger *slog.Logger,
	name string,
	err error,
) {
	logger.Error(
		"dependency initialization failed",
		"name",
		name,
		"error",
		err,
	)

	os.Exit(1)
}

func runServer(
	logger *slog.Logger,
	server *web.Server,
) {
	signalCtx, stop :=
		signal.NotifyContext(
			context.Background(),
			os.Interrupt,
			syscall.SIGTERM,
		)
	defer stop()

	serverError :=
		make(
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
	case err :=
		<-serverError:

		if err != nil {
			logger.Error(
				"server stopped unexpectedly",
				"error",
				err,
			)

			os.Exit(1)
		}

	case <-signalCtx.Done():
		shutdownServer(
			logger,
			server,
		)
	}
}

func shutdownServer(
	logger *slog.Logger,
	server *web.Server,
) {
	logger.Info(
		"server shutting down",
	)

	shutdownCtx, cancel :=
		context.WithTimeout(
			context.Background(),
			shutdownTimeout,
		)
	defer cancel()

	if err :=
		server.Shutdown(
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

		if err :=
			server.Close(); err != nil {
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
