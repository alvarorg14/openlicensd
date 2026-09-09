package api

import (
	"context"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/alvarorg14/openlicensd/server/internal/auth"
	"github.com/alvarorg14/openlicensd/server/internal/logging"
	"github.com/alvarorg14/openlicensd/server/internal/ratelimit"
)

func principalRateLimitKey(principal *auth.Principal) string {
	if principal == nil {
		return ""
	}
	switch principal.AuthMethod {
	case auth.AuthMethodAPIToken:
		return "token:" + principal.TokenID.String()
	default:
		return "user:" + principal.UserID.String()
	}
}

func (s *Server) rateLimitPrincipal(scope ratelimit.Scope) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if s.limiter == nil {
				next.ServeHTTP(w, r)
				return
			}

			principal, ok := auth.PrincipalFromContext(r.Context())
			if !ok {
				next.ServeHTTP(w, r)
				return
			}

			key := principalRateLimitKey(principal)
			if key == "" {
				next.ServeHTTP(w, r)
				return
			}

			allowed, retryAfter := s.limiter.Allow(r.Context(), scope, key)
			if allowed {
				next.ServeHTTP(w, r)
				return
			}

			retrySeconds := ratelimit.RetryAfterSeconds(retryAfter)
			logging.FromContext(r.Context()).Warn("rate limit exceeded",
				slog.String("scope", string(scope)),
				slog.String("principal_key", key),
				slog.Int("retry_after_seconds", retrySeconds),
			)

			w.Header().Set("Retry-After", strconv.Itoa(retrySeconds))
			writeError(w, http.StatusTooManyRequests, "rate limit exceeded")
		})
	}
}

func (s *Server) rateLimit(scope ratelimit.Scope) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if s.limiter == nil {
				next.ServeHTTP(w, r)
				return
			}

			clientIP := s.clientIP.From(r)
			allowed, retryAfter := s.limiter.Allow(r.Context(), scope, clientIP)
			if allowed {
				next.ServeHTTP(w, r)
				return
			}

			retrySeconds := ratelimit.RetryAfterSeconds(retryAfter)
			logging.FromContext(r.Context()).Warn("rate limit exceeded",
				slog.String("scope", string(scope)),
				slog.String("client_ip", clientIP),
				slog.Int("retry_after_seconds", retrySeconds),
			)

			w.Header().Set("Retry-After", strconv.Itoa(retrySeconds))
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
