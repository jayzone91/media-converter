package web

import (
	"fmt"
	"net"
	"net/http"
	"runtime/debug"
	"strings"
	"time"
)

type loggingResponseWriter struct {
	http.ResponseWriter

	status int
	bytes  int64
}

func (w *loggingResponseWriter) WriteHeader(
	status int,
) {
	if w.status != 0 {
		return
	}

	w.status = status

	w.ResponseWriter.WriteHeader(
		status,
	)
}

func (w *loggingResponseWriter) Write(
	data []byte,
) (int, error) {
	if w.status == 0 {
		w.WriteHeader(
			http.StatusOK,
		)
	}

	written, err := w.ResponseWriter.Write(
		data,
	)

	w.bytes += int64(written)

	return written, err
}

func (w *loggingResponseWriter) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}

func (s *Server) requestLoggingMiddleware(
	next http.Handler,
) http.Handler {
	return http.HandlerFunc(
		func(
			w http.ResponseWriter,
			r *http.Request,
		) {
			start := time.Now()

			writer := &loggingResponseWriter{
				ResponseWriter: w,
			}

			clientIP := requestClientIP(
				r,
			)

			s.logger.Debug(
				"http request started",
				"method",
				r.Method,
				"path",
				r.URL.Path,
				"client_ip",
				clientIP,
			)

			defer func() {
				if recovered := recover(); recovered != nil {
					s.logger.Error(
						"http handler panic",
						"method",
						r.Method,
						"path",
						r.URL.Path,
						"client_ip",
						clientIP,
						"user_agent",
						r.UserAgent(),
						"panic",
						fmt.Sprint(recovered),
						"stack",
						string(debug.Stack()),
					)

					if writer.status == 0 {
						http.Error(
							writer,
							"internal server error",
							http.StatusInternalServerError,
						)
					}
				}

				if writer.status == 0 {
					writer.status = http.StatusOK
				}

				duration := time.Since(
					start,
				)

				switch {
				case writer.status >= 500:
					s.logger.Error(
						"http request failed",
						"method",
						r.Method,
						"path",
						r.URL.Path,
						"status",
						writer.status,
						"bytes",
						writer.bytes,
						"duration",
						duration,
						"client_ip",
						clientIP,
						"user_agent",
						r.UserAgent(),
					)

				case writer.status >= 400:
					s.logger.Warn(
						"http request rejected",
						"method",
						r.Method,
						"path",
						r.URL.Path,
						"status",
						writer.status,
						"bytes",
						writer.bytes,
						"duration",
						duration,
						"client_ip",
						clientIP,
					)

				default:
					s.logger.Info(
						"http",
						"method",
						r.Method,
						"path",
						r.URL.Path,
						"status",
						writer.status,
						"duration",
						duration,
					)
				}
			}()

			next.ServeHTTP(
				writer,
				r,
			)
		},
	)
}

func requestClientIP(
	r *http.Request,
) string {
	if forwarded := strings.TrimSpace(
		r.Header.Get(
			"X-Forwarded-For",
		),
	); forwarded != "" {
		addresses := strings.Split(
			forwarded,
			",",
		)

		if len(addresses) > 0 {
			return strings.TrimSpace(
				addresses[0],
			)
		}
	}

	if realIP := strings.TrimSpace(
		r.Header.Get(
			"X-Real-IP",
		),
	); realIP != "" {
		return realIP
	}

	host, _, err := net.SplitHostPort(
		r.RemoteAddr,
	)
	if err == nil {
		return host
	}

	return r.RemoteAddr
}
