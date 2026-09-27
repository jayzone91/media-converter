package web

import (
	"log/slog"
	"net/http"
)

func (s *Server) logInfo(
	message string,
	attributes ...any,
) {
	s.logger.Info(
		message,
		attributes...,
	)
}

func (s *Server) logError(
	r *http.Request,
	message string,
	err error,
	attributes ...any,
) {
	args := []any{
		"method",
		r.Method,
		"path",
		r.URL.Path,
	}

	if err != nil {
		args = append(
			args,
			"error",
			err,
		)
	}

	args = append(
		args,
		attributes...,
	)

	s.logger.Error(
		message,
		args...,
	)
}

func (s *Server) logWarn(
	r *http.Request,
	message string,
	attributes ...any,
) {
	args := []any{
		"method",
		r.Method,
		"path",
		r.URL.Path,
	}

	args = append(
		args,
		attributes...,
	)

	s.logger.Warn(
		message,
		args...,
	)
}

func defaultLogger() *slog.Logger {
	return slog.Default()
}
