package harbor_test

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/openlicensd/openlicensd/server/internal/harbor"
)

func TestCreateEphemeralRobot(t *testing.T) {
	t.Parallel()

	var gotAuth string
	var gotBody map[string]any

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/v2.0/robots" {
			http.NotFound(w, r)
			return
		}

		gotAuth = r.Header.Get("Authorization")
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read body: %v", err)
		}
		if err := json.Unmarshal(body, &gotBody); err != nil {
			t.Fatalf("decode body: %v", err)
		}

		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":         42,
			"name":       "robot$myproject+openlicensd-x4f9k-123",
			"secret":     "robot-secret",
			"expires_at": time.Now().Add(24 * time.Hour).Unix(),
		})
	}))
	t.Cleanup(server.Close)

	client, err := harbor.New(server.URL, "admin", "secret", false, false)
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	creds, err := client.CreateEphemeralRobot(context.Background(), []string{"myproject"}, 1, "openlicensd", "X4F9K")
	if err != nil {
		t.Fatalf("create robot: %v", err)
	}

	expectedAuth := "Basic " + base64.StdEncoding.EncodeToString([]byte("admin:secret"))
	if gotAuth != expectedAuth {
		t.Fatalf("auth=%q want %q", gotAuth, expectedAuth)
	}

	if gotBody["level"] != "project" {
		t.Fatalf("level=%v want project", gotBody["level"])
	}
	if gotBody["duration"] != float64(1) {
		t.Fatalf("duration=%v want 1", gotBody["duration"])
	}

	name, ok := gotBody["name"].(string)
	if !ok || !strings.HasPrefix(name, "openlicensd-x4f9k-") {
		t.Fatalf("unexpected robot name: %v", gotBody["name"])
	}
	for _, r := range name {
		if (r < 'a' || r > 'z') && (r < '0' || r > '9') && r != '-' {
			t.Fatalf("robot name must use only letters, digits, and hyphens, got %q", name)
		}
	}

	permissions, ok := gotBody["permissions"].([]any)
	if !ok || len(permissions) != 1 {
		t.Fatalf("unexpected permissions: %v", gotBody["permissions"])
	}

	if creds.Name != "robot$myproject+openlicensd-x4f9k-123" {
		t.Fatalf("name=%q", creds.Name)
	}
	if creds.Secret != "robot-secret" {
		t.Fatalf("secret=%q", creds.Secret)
	}
	if creds.ExpiresAt == 0 {
		t.Fatalf("expected expires_at to be set")
	}
	if client.RegistryHost() == "" {
		t.Fatalf("expected registry host")
	}
}

func TestCreateEphemeralRobotMultipleProjectsUsesSystemLevel(t *testing.T) {
	t.Parallel()

	var gotBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read body: %v", err)
		}
		if err := json.Unmarshal(body, &gotBody); err != nil {
			t.Fatalf("decode body: %v", err)
		}

		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"name":       "robot$openlicensd",
			"secret":     "robot-secret",
			"expires_at": time.Now().Add(24 * time.Hour).Unix(),
		})
	}))
	t.Cleanup(server.Close)

	client, err := harbor.New(server.URL, "admin", "secret", false, false)
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	_, err = client.CreateEphemeralRobot(context.Background(), []string{"project-a", "project-b"}, 1, "openlicensd", "abcd")
	if err != nil {
		t.Fatalf("create robot: %v", err)
	}

	if gotBody["level"] != "system" {
		t.Fatalf("level=%v want system", gotBody["level"])
	}

	permissions, ok := gotBody["permissions"].([]any)
	if !ok || len(permissions) != 2 {
		t.Fatalf("permissions=%v want 2 entries", gotBody["permissions"])
	}
}

func TestCleanupExpiredRobots(t *testing.T) {
	t.Parallel()

	deleted := make(map[int64]bool)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v2.0/robots":
			_ = json.NewEncoder(w).Encode([]map[string]any{
				{"id": 1, "name": "robot$openlicensd-old", "expires_at": time.Now().Add(-time.Hour).Unix()},
				{"id": 2, "name": "robot$openlicensd-active", "expires_at": time.Now().Add(time.Hour).Unix()},
				{"id": 3, "name": "robot$other", "expires_at": time.Now().Add(-time.Hour).Unix()},
			})
		case r.Method == http.MethodDelete && strings.HasPrefix(r.URL.Path, "/api/v2.0/robots/"):
			id := strings.TrimPrefix(r.URL.Path, "/api/v2.0/robots/")
			switch id {
			case "1":
				deleted[1] = true
				w.WriteHeader(http.StatusOK)
			default:
				http.NotFound(w, r)
			}
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(server.Close)

	client, err := harbor.New(server.URL, "admin", "secret", false, false)
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	if err := client.CleanupExpiredRobots(context.Background(), "openlicensd"); err != nil {
		t.Fatalf("cleanup: %v", err)
	}

	if !deleted[1] {
		t.Fatalf("expected expired openlicensd robot to be deleted")
	}
	if deleted[2] || deleted[3] {
		t.Fatalf("unexpected deletions: %+v", deleted)
	}
}
