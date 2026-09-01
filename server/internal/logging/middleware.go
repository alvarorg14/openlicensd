package logging

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5/middleware"
)

func isProbePath(path string) bool {
	return path == "/healthz" || path == "/readyz"
}

// RequestLogger logs one line per HTTP request and injects a request-scoped logger into the context.
func RequestLogger(base *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			requestID := middleware.GetReqID(r.Context())
			logger := base.With(slog.String("request_id", requestID))
			ctx := ContextWithLogger(r.Context(), logger)

			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
			next.ServeHTTP(ww, r.WithContext(ctx))

			duration := time.Since(start)
			attrs := []any{
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.Int("status", ww.Status()),
				slog.Int("bytes", ww.BytesWritten()),
				slog.Int64("duration_ms", duration.Milliseconds()),
				slog.String("remote_addr", r.RemoteAddr),
			}

			switch {
			case ww.Status() >= 500:
				logger.Warn("request completed", attrs...)
			case isProbePath(r.URL.Path):
				logger.Debug("request completed", attrs...)
			default:
				logger.Info("request completed", attrs...)
			}
		})
	}
}
