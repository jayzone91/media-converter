package main

import (
	"context"
	"errors"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jayzone91/media-converter/internal/converter"
	"github.com/jayzone91/media-converter/internal/media"
	"github.com/jayzone91/media-converter/internal/web"
)

const shutdownTimeout = 2 * time.Minute

func main() {
	imageMagick, err :=
		converter.NewImageMagick()
	if err != nil {
		log.Fatal(err)
	}

	ffmpeg, err :=
		converter.NewFFmpeg()
	if err != nil {
		log.Fatal(err)
	}

	ffprobe, err :=
		media.NewFFProbe()
	if err != nil {
		log.Fatal(err)
	}

	libreOffice, err :=
		converter.NewLibreOffice()
	if err != nil {
		log.Fatal(err)
	}

	pdf, err :=
		converter.NewPDF(
			libreOffice,
		)
	if err != nil {
		log.Fatal(err)
	}

	qpdf, err :=
		converter.NewQPDF()
	if err != nil {
		log.Fatal(err)
	}

	server :=
		web.NewServer(
			imageMagick,
			ffmpeg,
			ffprobe,
			libreOffice,
			pdf,
			qpdf,
		)

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
		log.Println(
			"listening on :8080",
		)

		serverError <- server.ListenAndServe(
			":8080",
		)
	}()

	select {
	case err := <-serverError:
		if err != nil {
			log.Fatal(err)
		}

	case <-signalCtx.Done():
		log.Println(
			"shutting down",
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
				log.Printf(
					"graceful shutdown failed: %v",
					err,
				)
			}

			if err :=
				server.Close(); err != nil {
				log.Printf(
					"forced shutdown failed: %v",
					err,
				)
			}
		}

		log.Println(
			"server stopped",
		)
	}
}
