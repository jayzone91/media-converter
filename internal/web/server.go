// Package web provides HTTP server and handlers
package web

import (
	"net/http"

	"github.com/jayzone91/media-converter/internal/converter"
	"github.com/jayzone91/media-converter/internal/media"
)

type Server struct {
	imageMagick *converter.ImageMagick
	ffmpeg      *converter.FFmpeg
	ffprobe     *media.FFProbe
	libreOffice *converter.LibreOffice
	pdf         *converter.PDF
	mux         *http.ServeMux
}

func NewServer(
	imageMagick *converter.ImageMagick,
	ffmpeg *converter.FFmpeg,
	ffprobe *media.FFProbe,
	libreOffice *converter.LibreOffice,
	pdf *converter.PDF,
) *Server {
	server := &Server{
		imageMagick: imageMagick,
		ffmpeg:      ffmpeg,
		ffprobe:     ffprobe,
		libreOffice: libreOffice,
		pdf:         pdf,
		mux:         http.NewServeMux(),
	}

	server.routes()

	return server
}

func (s *Server) routes() {
	s.mux.HandleFunc("GET /", s.handleIndex)
	s.mux.HandleFunc("POST /upload", s.handleUpload)
	s.mux.HandleFunc("POST /convert", s.handleConvert)
}

func (s *Server) handleIndex(w http.ResponseWriter, _ *http.Request) {
	_, _ = w.Write([]byte("media-converter"))
}

func (s *Server) ListenAndServe(addr string) error {
	return http.ListenAndServe(addr, s.mux)
}
