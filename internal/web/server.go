// Package web provides HTTP server and handlers
package web

import (
	"net/http"

	"github.com/jayzone91/media-converter/internal/converter"
)

type Server struct {
	imageMagick *converter.ImageMagick
	ffmpeg      *converter.FFmpeg
	mux         *http.ServeMux
}

func NewServer(imageMagick *converter.ImageMagick, ffmpeg *converter.FFmpeg) *Server {
	server := &Server{
		imageMagick: imageMagick,
		ffmpeg:      ffmpeg,
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
