package api

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/alvarorg14/openlicensd/server/internal/logging"
	appoidc "github.com/alvarorg14/openlicensd/server/internal/oidc"
	"github.com/alvarorg14/openlicensd/server/internal/store"
)

const (
	oidcStateCookie    = "openlicensd_oidc_state"
	oidcNonceCookie    = "openlicensd_oidc_nonce"
	oidcVerifierCookie = "openlicensd_oidc_verifier"
	oidcReturnToCookie = "openlicensd_oidc_return_to"
	oidcFlowMaxAge     = 10 * time.Minute
)

func (s *Server) handleOIDCLogin(w http.ResponseWriter, r *http.Request) {
	if s.oidc == nil {
		http.NotFound(w, r)
		return
	}

	state, err := randomToken()
	if err != nil {
		writeInternalError(w, r, err, "failed to start sso")
		return
	}
	nonce, err := randomToken()
	if err != nil {
		writeInternalError(w, r, err, "failed to start sso")
		return
	}
	verifier, err := randomToken()
	if err != nil {
		writeInternalError(w, r, err, "failed to start sso")
		return
	}

	returnTo := sanitizeReturnTo(r.URL.Query().Get("return_to"))
	setOIDCFlowCookie(w, oidcStateCookie, state, s.cfg.CookieSecure)
	setOIDCFlowCookie(w, oidcNonceCookie, nonce, s.cfg.CookieSecure)
	setOIDCFlowCookie(w, oidcVerifierCookie, verifier, s.cfg.CookieSecure)
	if returnTo != "" {
		setOIDCFlowCookie(w, oidcReturnToCookie, returnTo, s.cfg.CookieSecure)
	}

	authURL := s.oidc.AuthCodeURL(state, nonce, verifier)
	http.Redirect(w, r, authURL, http.StatusFound)
}

func (s *Server) handleOIDCCallback(w http.ResponseWriter, r *http.Request) {
	if s.oidc == nil {
		http.NotFound(w, r)
		return
	}

	defer clearOIDCFlowCookies(w, s.cfg.CookieSecure)

	if errMsg := r.URL.Query().Get("error"); errMsg != "" {
		redirectSSOError(w, r, "provider_error")
		return
	}

	stateCookie, err := r.Cookie(oidcStateCookie)
	if err != nil || stateCookie.Value == "" {
		redirectSSOError(w, r, "state_missing")
		return
	}
	queryState := r.URL.Query().Get("state")
	if queryState == "" || subtle.ConstantTimeCompare([]byte(stateCookie.Value), []byte(queryState)) != 1 {
		redirectSSOError(w, r, "state_mismatch")
		return
	}

	nonceCookie, err := r.Cookie(oidcNonceCookie)
	if err != nil || nonceCookie.Value == "" {
		redirectSSOError(w, r, "nonce_missing")
		return
	}
	verifierCookie, err := r.Cookie(oidcVerifierCookie)
	if err != nil || verifierCookie.Value == "" {
		redirectSSOError(w, r, "verifier_missing")
		return
	}

	code := r.URL.Query().Get("code")
	if code == "" {
		redirectSSOError(w, r, "code_missing")
		return
	}

	claims, err := s.oidc.Exchange(r.Context(), code, verifierCookie.Value, nonceCookie.Value)
	if err != nil {
		redirectSSOError(w, r, "exchange_failed")
		return
	}

	user, err := s.resolveOIDCUser(r, claims)
	if err != nil {
		redirectSSOError(w, r, "user_resolution_failed")
		return
	}

	userAgent := r.UserAgent()
	clientIP := s.clientIP.From(r)

	tokens, err := s.auth.CreateSessionForUser(r.Context(), user, store.AuthProviderOIDC, userAgent, clientIP)
	if err != nil {
		redirectSSOError(w, r, "session_creation_failed")
		return
	}

	if err := s.store.ClearFailedLogin(r.Context(), user.ID); err != nil {
		logging.FromContext(r.Context()).Warn("clear failed login failed",
			slog.String("user_id", user.ID.String()),
			slog.Any("err", err),
		)
	}

	s.auth.SetSessionCookies(w, tokens)

	returnTo := "/licenses"
	if cookie, err := r.Cookie(oidcReturnToCookie); err == nil && cookie.Value != "" {
		if sanitized := sanitizeReturnTo(cookie.Value); sanitized != "" {
			returnTo = sanitized
		}
	}

	http.Redirect(w, r, returnTo, http.StatusFound)
}

func (s *Server) resolveOIDCUser(r *http.Request, claims *appoidc.Claims) (*store.User, error) {
	ctx := r.Context()

	user, err := s.store.GetUserByExternalID(ctx, store.AuthProviderOIDC, claims.Subject)
	if err != nil {
		return nil, err
	}

	if user == nil {
		existing, err := s.store.GetUserByEmail(ctx, claims.Email)
		if err != nil {
			return nil, err
		}
		if existing != nil {
			user, err = s.store.LinkUserToProvider(ctx, existing.ID, store.AuthProviderOIDC, claims.Subject)
			if err != nil {
				return nil, err
			}
		}
	}

	if user == nil {
		role := store.Role(s.cfg.OIDC.DefaultRole)
		if s.cfg.OIDC.IsAdminEmail(claims.Email) {
			role = store.RoleAdmin
		}
		externalID := claims.Subject
		user, err = s.store.CreateUser(ctx, claims.Email, claims.Name, nil, role, store.AuthProviderOIDC, &externalID)
		if err != nil {
			return nil, err
		}
	}

	if user.DisabledAt != nil {
		return nil, fmt.Errorf("user is disabled")
	}

	user, err = s.store.SyncUserProfile(ctx, user.ID, claims.Email, claims.Name, pictureURLPtr(claims.Picture))
	if err != nil {
		return nil, err
	}

	return user, nil
}

func setOIDCFlowCookie(w http.ResponseWriter, name, value string, secure bool) {
	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Value:    value,
		Path:     "/",
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(oidcFlowMaxAge.Seconds()),
	})
}

func clearOIDCFlowCookies(w http.ResponseWriter, secure bool) {
	for _, name := range []string{oidcStateCookie, oidcNonceCookie, oidcVerifierCookie, oidcReturnToCookie} {
		http.SetCookie(w, &http.Cookie{
			Name:     name,
			Value:    "",
			Path:     "/",
			HttpOnly: true,
			Secure:   secure,
			SameSite: http.SameSiteLaxMode,
			MaxAge:   -1,
		})
	}
}

func sanitizeReturnTo(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if !strings.HasPrefix(value, "/") || strings.HasPrefix(value, "//") {
		return ""
	}
	if strings.Contains(value, "://") {
		return ""
	}
	parsed, err := url.Parse(value)
	if err != nil || parsed.Host != "" {
		return ""
	}
	return value
}

func redirectSSOError(w http.ResponseWriter, r *http.Request, reason string) {
	logging.FromContext(r.Context()).Warn("oidc callback failed", slog.String("reason", reason))
	http.Redirect(w, r, "/login?error=sso_failed", http.StatusFound)
}

func randomToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func pictureURLPtr(picture string) *string {
	if picture == "" {
		return nil
	}
	return &picture
}
