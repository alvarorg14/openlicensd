package auth_test

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/alvarorg14/openlicensd/server/internal/auth"
	"github.com/alvarorg14/openlicensd/server/internal/logging"
	"github.com/alvarorg14/openlicensd/server/internal/store"
)

func setupAuthTest(t *testing.T) (*auth.Service, *store.Store, *bytes.Buffer) {
	t.Helper()

	databaseURL := os.Getenv("OPENLICENSD_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("OPENLICENSD_DATABASE_URL not set")
	}

	ctx := context.Background()
	st, err := store.New(ctx, databaseURL)
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	t.Cleanup(st.Close)

	var buf bytes.Buffer
	logger := logging.NewTestLogger(&buf, slog.LevelInfo)
	svc := auth.NewService(st, time.Hour, false, logger)
	return svc, st, &buf
}

func lastLogReason(t *testing.T, buf *bytes.Buffer) string {
	t.Helper()
	lines := bytes.Split(bytes.TrimSpace(buf.Bytes()), []byte("\n"))
	if len(lines) == 0 {
		t.Fatalf("no log output")
	}
	var record map[string]any
	if err := json.Unmarshal(lines[len(lines)-1], &record); err != nil {
		t.Fatalf("unmarshal log: %v", err)
	}
	reason, _ := record["reason"].(string)
	return reason
}

func TestLoginUnknownUser(t *testing.T) {
	svc, _, buf := setupAuthTest(t)
	ctx := context.Background()

	_, _, err := svc.Login(ctx, "missing@example.com", "password", "ua", "127.0.0.1")
	if err == nil {
		t.Fatalf("expected login error")
	}
	if got := lastLogReason(t, buf); got != "bad_password" {
		t.Fatalf("reason=%q", got)
	}
}

func TestLoginBadPassword(t *testing.T) {
	svc, st, buf := setupAuthTest(t)
	ctx := context.Background()

	hash, err := auth.HashPassword("correct-password")
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	email := "bad-password-" + time.Now().Format("150405.000000") + "@example.com"
	if _, err := st.CreateUser(ctx, email, "User", &hash, store.RoleViewer, store.AuthProviderLocal, nil); err != nil {
		t.Fatalf("create user: %v", err)
	}

	_, _, err = svc.Login(ctx, email, "wrong-password", "ua", "127.0.0.1")
	if err == nil {
		t.Fatalf("expected login error")
	}
	if got := lastLogReason(t, buf); got != "bad_password" {
		t.Fatalf("reason=%q", got)
	}
}

func TestLoginUserDisabled(t *testing.T) {
	svc, st, buf := setupAuthTest(t)
	ctx := context.Background()

	hash, err := auth.HashPassword("password")
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	email := "disabled-" + time.Now().Format("150405.000000") + "@example.com"
	user, err := st.CreateUser(ctx, email, "User", &hash, store.RoleViewer, store.AuthProviderLocal, nil)
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	if _, err := st.SetUserDisabled(ctx, user.ID, true); err != nil {
		t.Fatalf("disable user: %v", err)
	}

	_, _, err = svc.Login(ctx, email, "password", "ua", "127.0.0.1")
	if err == nil {
		t.Fatalf("expected login error")
	}
	if got := lastLogReason(t, buf); got != "user_disabled" {
		t.Fatalf("reason=%q", got)
	}
}

func TestLoginAccountLocked(t *testing.T) {
	svc, st, buf := setupAuthTest(t)
	ctx := context.Background()

	hash, err := auth.HashPassword("password")
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	email := "locked-" + time.Now().Format("150405.000000") + "@example.com"
	if _, err := st.CreateUser(ctx, email, "User", &hash, store.RoleViewer, store.AuthProviderLocal, nil); err != nil {
		t.Fatalf("create user: %v", err)
	}

	for i := 0; i < 5; i++ {
		_, _, _ = svc.Login(ctx, email, "wrong-password", "ua", "127.0.0.1")
		buf.Reset()
	}

	_, _, err = svc.Login(ctx, email, "password", "ua", "127.0.0.1")
	if err == nil {
		t.Fatalf("expected login error")
	}
	if got := lastLogReason(t, buf); got != "account_locked" {
		t.Fatalf("reason=%q", got)
	}
}

func TestLoginNoPasswordSet(t *testing.T) {
	svc, st, buf := setupAuthTest(t)
	ctx := context.Background()

	email := "nopass-" + time.Now().Format("150405.000000") + "@example.com"
	if _, err := st.CreateUser(ctx, email, "User", nil, store.RoleViewer, store.AuthProviderOIDC, nil); err != nil {
		t.Fatalf("create user: %v", err)
	}

	_, _, err := svc.Login(ctx, email, "password", "ua", "127.0.0.1")
	if err == nil {
		t.Fatalf("expected login error")
	}
	if got := lastLogReason(t, buf); got != "no_password_set" {
		t.Fatalf("reason=%q", got)
	}
}
