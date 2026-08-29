package api

import (
	"context"
	"net/http"
	"strconv"

	"github.com/alvarorg14/openlicensd/server/internal/ratelimit"
)

func (s *Server) rateLimit(scope ratelimit.Scope) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if s.limiter == nil {
				next.ServeHTTP(w, r)
				return
			}

			clientIP := s.clientIP.From(r)
			allowed, retryAfter := s.limiter.Allow(scope, clientIP)
			if allowed {
				next.ServeHTTP(w, r)
				return
			}

			w.Header().Set("Retry-After", strconv.Itoa(ratelimit.RetryAfterSeconds(retryAfter)))
			writeError(w, http.StatusTooManyRequests, "rate limit exceeded")
		})
	}
}

// StartBackground launches background tasks owned by the API server.
func (s *Server) StartBackground(ctx context.Context) {
	if s.limiter != nil {
		go s.limiter.Run(ctx)
	}
}
