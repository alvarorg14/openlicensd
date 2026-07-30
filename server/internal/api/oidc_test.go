package api_test

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/go-jose/go-jose/v4"
	josejwt "github.com/go-jose/go-jose/v4/jwt"
	"github.com/google/uuid"
	"github.com/openlicensd/openlicensd/server/internal/api"
	"github.com/openlicensd/openlicensd/server/internal/auth"
	"github.com/openlicensd/openlicensd/server/internal/config"
	appoidc "github.com/openlicensd/openlicensd/server/internal/oidc"
	"github.com/openlicensd/openlicensd/server/internal/store"
)

type mockOIDCProvider struct {
	server   *httptest.Server
	issuer   string
	key      *rsa.PrivateKey
	kid      string
	clientID string
	pending  map[string]mockOIDCPending
}

type mockOIDCPending struct {
	nonce string
	sub   string
	email string
	name  string
}

func newMockOIDCProvider(t *testing.T, clientID string) *mockOIDCProvider {
	t.Helper()

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	m := &mockOIDCProvider{
		key:      key,
		kid:      "test-key",
		clientID: clientID,
		pending:  make(map[string]mockOIDCPending),
	}

	mux := http.NewServeMux()
	m.server = httptest.NewServer(mux)
	m.issuer = m.server.URL

	mux.HandleFunc("/.well-known/openid-configuration", func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"issuer":                                m.issuer,
			"authorization_endpoint":                m.issuer + "/authorize",
			"token_endpoint":                        m.issuer + "/token",
			"jwks_uri":                              m.issuer + "/jwks",
			"response_types_supported":              []string{"code"},
			"subject_types_supported":               []string{"public"},
			"id_token_signing_alg_values_supported": []string{"RS256"},
		})
	})

	mux.HandleFunc("/jwks", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		jwk := jose.JSONWebKey{Key: &key.PublicKey, KeyID: m.kid, Use: "sig", Algorithm: string(jose.RS256)}
		_ = json.NewEncoder(w).Encode(map[string]any{"keys": []jose.JSONWebKey{jwk}})
	})

	mux.HandleFunc("/authorize", func(w http.ResponseWriter, r *http.Request) {
		redirectURI := r.URL.Query().Get("redirect_uri")
		state := r.URL.Query().Get("state")
		nonce := r.URL.Query().Get("nonce")
		code := "test-code"
		m.pending[code] = mockOIDCPending{nonce: nonce}
		http.Redirect(w, r, redirectURI+"?code="+code+"&state="+url.QueryEscape(state), http.StatusFound)
	})

	mux.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		code := r.Form.Get("code")
		pending, ok := m.pending[code]
		if !ok {
			http.Error(w, "unknown code", http.StatusBadRequest)
			return
		}

		sub := pending.sub
		email := pending.email
		name := pending.name
		if sub == "" {
			sub = "oidc-subject"
		}
		if email == "" {
			email = "oidc-user@example.com"
		}
		if name == "" {
			name = "OIDC User"
		}

		idToken, err := m.signIDToken(sub, email, name, pending.nonce)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"access_token": "access-token",
			"token_type":   "Bearer",
			"expires_in":   3600,
			"id_token":     idToken,
		})
	})

	t.Cleanup(m.server.Close)
	return m
}

func (m *mockOIDCProvider) setPending(code, nonce, sub, email, name string) {
	m.pending[code] = mockOIDCPending{
		nonce: nonce,
		sub:   sub,
		email: email,
		name:  name,
	}
}

func (m *mockOIDCProvider) signIDToken(sub, email, name, nonce string) (string, error) {
	signer, err := jose.NewSigner(
		jose.SigningKey{Algorithm: jose.RS256, Key: m.key},
		(&jose.SignerOptions{}).WithType("JWT").WithHeader("kid", m.kid),
	)
	if err != nil {
		return "", err
	}

	now := time.Now()
	claims := map[string]any{
		"iss":   m.issuer,
		"aud":   m.clientID,
		"sub":   sub,
		"email": email,
		"name":  name,
		"nonce": nonce,
		"iat":   now.Unix(),
		"exp":   now.Add(time.Hour).Unix(),
	}

	return josejwt.Signed(signer).Claims(claims).Serialize()
}

func setupOIDCTestEnv(t *testing.T, idp *mockOIDCProvider, redirectURL string) (http.Handler, *store.Store) {
	t.Helper()

	databaseURL := os.Getenv("OPENLICENSD_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("OPENLICENSD_DATABASE_URL not set")
	}

	cfg := &config.Config{
		Addr:              ":8080",
		DatabaseURL:       databaseURL,
		SessionTTLHours:   24,
		CookieSecure:      false,
		LocalLoginEnabled: true,
		OIDC: config.OIDCConfig{
			Enabled:      true,
			IssuerURL:    idp.issuer,
			ClientID:     idp.clientID,
			ClientSecret: "client-secret",
			RedirectURL:  redirectURL,
			Scopes:       []string{"openid", "profile", "email"},
			DefaultRole:  "viewer",
			ProviderName: "Test SSO",
		},
	}

	ctx := context.Background()
	st, err := store.New(ctx, cfg.DatabaseURL)
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	t.Cleanup(st.Close)

	srv := api.New(ctx, cfg, st)
	return srv.Router(nil), st
}

func TestOIDCRoutesDisabledWhenOIDCOff(t *testing.T) {
	env := setupTestEnv(t)
	resp := doJSON(t, env.Handler, http.MethodGet, "/api/v1/auth/oidc/login", nil, nil)
	if resp.Code != http.StatusNotFound {
		t.Fatalf("status=%d want 404", resp.Code)
	}
}

func TestLocalLoginDisabledReturnsForbidden(t *testing.T) {
	databaseURL := os.Getenv("OPENLICENSD_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("OPENLICENSD_DATABASE_URL not set")
	}

	idp := newMockOIDCProvider(t, "test-client")
	redirectURL := "http://example.com/api/v1/auth/oidc/callback"

	cfg := &config.Config{
		Addr:              ":8080",
		DatabaseURL:       databaseURL,
		SessionTTLHours:   24,
		CookieSecure:      false,
		LocalLoginEnabled: false,
		OIDC: config.OIDCConfig{
			Enabled:      true,
			IssuerURL:    idp.issuer,
			ClientID:     idp.clientID,
			ClientSecret: "client-secret",
			RedirectURL:  redirectURL,
			Scopes:       []string{"openid", "profile", "email"},
			DefaultRole:  "viewer",
			ProviderName: "Test SSO",
		},
	}

	ctx := context.Background()
	st, err := store.New(ctx, cfg.DatabaseURL)
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	t.Cleanup(st.Close)

	handler := api.New(ctx, cfg, st).Router(nil)
	resp := doJSON(t, handler, http.MethodPost, "/api/v1/auth/login", map[string]string{
		"email":    "admin@example.com",
		"password": "password",
	}, nil)
	if resp.Code != http.StatusForbidden {
		t.Fatalf("status=%d want 403 body=%s", resp.Code, resp.Body.String())
	}
}

func TestAuthProvidersReflectsOIDCConfig(t *testing.T) {
	databaseURL := os.Getenv("OPENLICENSD_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("OPENLICENSD_DATABASE_URL not set")
	}

	idp := newMockOIDCProvider(t, "providers-client")
	handler, _ := setupOIDCTestEnv(t, idp, "http://example.com/api/v1/auth/oidc/callback")

	resp := doJSON(t, handler, http.MethodGet, "/api/v1/auth/providers", nil, nil)
	if resp.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", resp.Code, resp.Body.String())
	}

	var body map[string]any
	if err := json.Unmarshal(resp.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body["oidc"] != true {
		t.Fatalf("oidc=%v want true", body["oidc"])
	}
	if body["oidc_name"] != "Test SSO" {
		t.Fatalf("oidc_name=%v", body["oidc_name"])
	}
}

func TestOIDCExchangeDirect(t *testing.T) {
	idp := newMockOIDCProvider(t, "exchange-client")
	ctx := context.Background()

	client, err := appoidc.New(ctx, appoidc.Config{
		IssuerURL:    idp.issuer,
		ClientID:     idp.clientID,
		ClientSecret: "client-secret",
		RedirectURL:  "http://example.com/api/v1/auth/oidc/callback",
		Scopes:       []string{"openid", "profile", "email"},
	})
	if err != nil {
		t.Fatalf("oidc client: %v", err)
	}

	idp.setPending("test-code", "test-nonce", "sub-1", "exchange@example.com", "Exchange User")
	_, err = client.Exchange(ctx, "test-code", "test-verifier", "test-nonce")
	if err != nil {
		t.Fatalf("exchange: %v", err)
	}
}

func TestOIDCCallbackCreatesUser(t *testing.T) {
	idp := newMockOIDCProvider(t, "callback-client")
	redirectURL := "http://example.com/api/v1/auth/oidc/callback"
	handler, st := setupOIDCTestEnv(t, idp, redirectURL)

	email := fmt.Sprintf("oidc-new-%s@example.com", uuid.NewString())
	sub := "sub-" + uuid.NewString()

	loginReq := httptest.NewRequest(http.MethodGet, "/api/v1/auth/oidc/login", nil)
	loginRec := httptest.NewRecorder()
	handler.ServeHTTP(loginRec, loginReq)
	if loginRec.Code != http.StatusFound {
		t.Fatalf("login status=%d", loginRec.Code)
	}

	state := cookieValue(loginRec, "openlicensd_oidc_state")
	nonce := cookieValue(loginRec, "openlicensd_oidc_nonce")
	verifier := cookieValue(loginRec, "openlicensd_oidc_verifier")
	if state == "" || nonce == "" || verifier == "" {
		t.Fatalf("missing oidc flow cookies")
	}

	idp.setPending("test-code", nonce, sub, email, "OIDC New User")

	callbackURL := "/api/v1/auth/oidc/callback?code=test-code&state=" + url.QueryEscape(state)
	callbackReq := httptest.NewRequest(http.MethodGet, callbackURL, nil)
	for _, c := range loginRec.Result().Cookies() {
		callbackReq.AddCookie(c)
	}

	callbackRec := httptest.NewRecorder()
	handler.ServeHTTP(callbackRec, callbackReq)
	if callbackRec.Code != http.StatusFound {
		t.Fatalf("callback status=%d body=%s", callbackRec.Code, callbackRec.Body.String())
	}
	if loc := callbackRec.Header().Get("Location"); loc != "/licenses" {
		t.Fatalf("unexpected redirect %q", loc)
	}

	user, err := st.GetUserByEmail(context.Background(), email)
	if err != nil {
		t.Fatalf("get user: %v", err)
	}
	if user == nil {
		t.Fatalf("expected user created")
	}
	if user.AuthProvider != store.AuthProviderOIDC {
		t.Fatalf("auth_provider=%q", user.AuthProvider)
	}
	if user.Role != store.RoleViewer {
		t.Fatalf("role=%q want viewer", user.Role)
	}

	sessionCookie := findCookie(callbackRec.Result().Cookies(), auth.SessionCookieName)
	if sessionCookie == nil {
		t.Fatalf("expected session cookie")
	}
}

func cookieValue(rec *httptest.ResponseRecorder, name string) string {
	for _, c := range rec.Result().Cookies() {
		if c.Name == name {
			return c.Value
		}
	}
	return ""
}
