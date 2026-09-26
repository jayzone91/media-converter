package main

import (
	"log"

	"github.com/jayzone91/media-converter/internal/converter"
	"github.com/jayzone91/media-converter/internal/media"
	"github.com/jayzone91/media-converter/internal/web"
)

func main() {
	imageMagick, err := converter.NewImageMagick()
	if err != nil {
		log.Fatal(err)
	}

	ffmpeg, err := converter.NewFFmpeg()
	if err != nil {
		log.Fatal(err)
	}

	ffprobe, err := media.NewFFProbe()
	if err != nil {
		log.Fatal(err)
	}

	libreoffice, err := converter.NewLibreOffice()

	server := web.NewServer(imageMagick, ffmpeg, ffprobe, libreoffice)

	log.Println("listening on :8080")

	if err := server.ListenAndServe(":8080"); err != nil {
		log.Fatal(err)
	}
}
