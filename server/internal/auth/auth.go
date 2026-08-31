package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/alvarorg14/openlicensd/server/internal/store"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

const (
	SessionCookieName = "openlicensd_session"
	CSRFCookieName    = "openlicensd_csrf"
	CSRFHeaderName    = "X-CSRF-Token"

	maxFailedAttempts = 5
	lockoutDuration   = 15 * time.Minute
	sessionTouchMin   = 5 * time.Minute

	// bcrypt hash of "dummy-password-for-timing" — used when user not found.
	dummyPasswordHash = "$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy"
)

type contextKey string

const principalContextKey contextKey = "principal"

type Principal struct {
	UserID       uuid.UUID
	Email        string
	Name         string
	Role         store.Role
	AuthProvider string
	PictureURL   *string
	SessionID    uuid.UUID
	HasPassword  bool
}

type Service struct {
	store        *store.Store
	sessionTTL   time.Duration
	cookieSecure bool
	logger       *slog.Logger
}

func NewService(st *store.Store, sessionTTL time.Duration, cookieSecure bool, logger *slog.Logger) *Service {
	return &Service{
		store:        st,
		sessionTTL:   sessionTTL,
		cookieSecure: cookieSecure,
		logger:       logger,
	}
}

func (s *Service) Login(ctx context.Context, email, password, userAgent, clientIP string) (*store.User, *SessionTokens, error) {
	user, err := s.store.GetUserByEmail(ctx, email)
	if err != nil {
		return nil, nil, err
	}

	hashToCompare := dummyPasswordHash
	if user != nil && user.PasswordHash != nil {
		hashToCompare = *user.PasswordHash
	}

	if user != nil && user.PasswordHash == nil {
		s.logLoginFailure(email, clientIP, "no_password_set")
		return nil, nil, errors.New("invalid credentials")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(hashToCompare), []byte(password)); err != nil {
		if user != nil {
			if recordErr := s.store.RecordFailedLogin(ctx, user.ID, maxFailedAttempts, lockoutDuration); recordErr != nil {
				s.logger.Warn("record failed login failed",
					slog.String("user_id", user.ID.String()),
					slog.Any("err", recordErr),
				)
			} else if updated, fetchErr := s.store.GetUserByID(ctx, user.ID); fetchErr == nil && updated != nil &&
				updated.LockedUntil != nil && updated.LockedUntil.After(time.Now()) &&
				updated.FailedLoginAttempts >= maxFailedAttempts {
				s.logger.Warn("account locked",
					slog.String("email", email),
					slog.String("client_ip", clientIP),
					slog.String("user_id", user.ID.String()),
				)
			}
		}
		s.logLoginFailure(email, clientIP, "bad_password")
		return nil, nil, errors.New("invalid credentials")
	}

	if user == nil {
		s.logLoginFailure(email, clientIP, "unknown_user")
		return nil, nil, errors.New("invalid credentials")
	}

	if user.DisabledAt != nil {
		s.logLoginFailure(email, clientIP, "user_disabled")
		return nil, nil, errors.New("invalid credentials")
	}

	if user.LockedUntil != nil && user.LockedUntil.After(time.Now()) {
		s.logLoginFailure(email, clientIP, "account_locked")
		return nil, nil, errors.New("invalid credentials")
	}

	if err := s.store.ClearFailedLogin(ctx, user.ID); err != nil {
		return nil, nil, err
	}

	tokens, err := s.createSession(ctx, user, user.AuthProvider, userAgent, clientIP)
	if err != nil {
		return nil, nil, err
	}

	s.logger.Info("login succeeded",
		slog.String("user_id", user.ID.String()),
		slog.String("email", email),
		slog.String("auth_provider", user.AuthProvider),
		slog.String("client_ip", clientIP),
	)

	return user, tokens, nil
}

func (s *Service) logLoginFailure(email, clientIP, reason string) {
	s.logger.Warn("login failed",
		slog.String("email", email),
		slog.String("client_ip", clientIP),
		slog.String("reason", reason),
	)
}

// CreateSessionForUser creates a session for an already-authenticated user (e.g. OIDC in Phase 2).
func (s *Service) CreateSessionForUser(ctx context.Context, user *store.User, authProvider, userAgent, clientIP string) (*SessionTokens, error) {
	if user.DisabledAt != nil {
		return nil, errors.New("user is disabled")
	}
	return s.createSession(ctx, user, authProvider, userAgent, clientIP)
}

type SessionTokens struct {
	SessionToken string
	CSRFToken    string
	SessionID    uuid.UUID
}

func (s *Service) createSession(ctx context.Context, user *store.User, authProvider, userAgent, clientIP string) (*SessionTokens, error) {
	sessionToken, err := generateToken()
	if err != nil {
		return nil, err
	}
	csrfToken, err := generateToken()
	if err != nil {
		return nil, err
	}

	tokenHash := hashToken(sessionToken)
	expiresAt := time.Now().Add(s.sessionTTL)

	var ua, ip *string
	if userAgent != "" {
		ua = &userAgent
	}
	if clientIP != "" {
		ip = &clientIP
	}

	sess, err := s.store.CreateSession(ctx, user.ID, tokenHash, authProvider, ua, ip, expiresAt)
	if err != nil {
		return nil, err
	}

	return &SessionTokens{
		SessionToken: sessionToken,
		CSRFToken:    csrfToken,
		SessionID:    sess.ID,
	}, nil
}

func (s *Service) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		principal, err := s.authenticateRequest(r)
		if err != nil {
			writeAuthError(w, "invalid session")
			return
		}

		ctx := context.WithValue(r.Context(), principalContextKey, principal)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (s *Service) authenticateRequest(r *http.Request) (*Principal, error) {
	cookie, err := r.Cookie(SessionCookieName)
	if err != nil || cookie.Value == "" {
		return nil, errors.New("missing session")
	}

	tokenHash := hashToken(cookie.Value)
	sess, err := s.store.GetSessionByTokenHash(r.Context(), tokenHash)
	if err != nil || sess == nil {
		return nil, errors.New("invalid session")
	}

	user, err := s.store.GetUserByID(r.Context(), sess.UserID)
	if err != nil || user == nil || user.DisabledAt != nil {
		return nil, errors.New("invalid session")
	}

	if isUnsafeMethod(r.Method) {
		csrfCookie, err := r.Cookie(CSRFCookieName)
		if err != nil || csrfCookie.Value == "" {
			return nil, errors.New("missing csrf token")
		}
		csrfHeader := r.Header.Get(CSRFHeaderName)
		if csrfHeader == "" || subtle.ConstantTimeCompare([]byte(csrfCookie.Value), []byte(csrfHeader)) != 1 {
			return nil, errors.New("invalid csrf token")
		}
	}

	if time.Since(sess.LastSeenAt) >= sessionTouchMin {
		newExpiry := time.Now().Add(s.sessionTTL)
		if touchErr := s.store.TouchSession(r.Context(), sess.ID, newExpiry); touchErr != nil {
			s.logger.Warn("touch session failed",
				slog.String("session_id", sess.ID.String()),
				slog.Any("err", touchErr),
			)
		}
	}

	return &Principal{
		UserID:       user.ID,
		Email:        user.Email,
		Name:         user.Name,
		Role:         user.Role,
		AuthProvider: user.AuthProvider,
		PictureURL:   user.PictureURL,
		SessionID:    sess.ID,
		HasPassword:  user.PasswordHash != nil,
	}, nil
}

func RequireRole(roles ...store.Role) func(http.Handler) http.Handler {
	allowed := make(map[store.Role]struct{}, len(roles))
	for _, role := range roles {
		allowed[role] = struct{}{}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			principal, ok := PrincipalFromContext(r.Context())
			if !ok {
				writeAuthError(w, "unauthorized")
				return
			}

			if _, ok := allowed[principal.Role]; !ok {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				_, _ = w.Write([]byte(`{"error":"forbidden"}`))
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func (s *Service) Logout(ctx context.Context, sessionID uuid.UUID) error {
	return s.store.RevokeSession(ctx, sessionID)
}

func (s *Service) SetSessionCookies(w http.ResponseWriter, tokens *SessionTokens) {
	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookieName,
		Value:    tokens.SessionToken,
		Path:     "/",
		HttpOnly: true,
		Secure:   s.cookieSecure,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   int(s.sessionTTL.Seconds()),
	})
	http.SetCookie(w, &http.Cookie{
		Name:     CSRFCookieName,
		Value:    tokens.CSRFToken,
		Path:     "/",
		HttpOnly: false,
		Secure:   s.cookieSecure,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   int(s.sessionTTL.Seconds()),
	})
}

func (s *Service) ClearSessionCookies(w http.ResponseWriter) {
	clearCookie := func(name string, httpOnly bool) {
		http.SetCookie(w, &http.Cookie{
			Name:     name,
			Value:    "",
			Path:     "/",
			HttpOnly: httpOnly,
			Secure:   s.cookieSecure,
			SameSite: http.SameSiteStrictMode,
			MaxAge:   -1,
		})
	}
	clearCookie(SessionCookieName, true)
	clearCookie(CSRFCookieName, false)
}

func PrincipalFromContext(ctx context.Context) (*Principal, bool) {
	principal, ok := ctx.Value(principalContextKey).(*Principal)
	return principal, ok
}

func writeAuthError(w http.ResponseWriter, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	_, _ = w.Write([]byte(`{"error":"` + message + `"}`))
}

func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

func VerifyPassword(hash, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

func generateToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate token: %w", err)
	}
	return hex.EncodeToString(b), nil
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func isUnsafeMethod(method string) bool {
	switch strings.ToUpper(method) {
	case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
		return true
	default:
		return false
	}
}
