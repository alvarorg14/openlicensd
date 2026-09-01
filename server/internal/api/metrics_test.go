package api_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/alvarorg14/openlicensd/server/internal/api"
	"github.com/alvarorg14/openlicensd/server/internal/config"
)

func TestMetricsHandlerEnabled(t *testing.T) {
	cfg := &config.Config{
		Metrics: config.MetricsConfig{
			Enabled: true,
			Addr:    ":9090",
		},
	}

	srv, err := api.New(context.Background(), cfg, nil, testLogger())
	if err != nil {
		t.Fatalf("api server: %v", err)
	}

	handler := srv.MetricsHandler()
	if handler == nil {
		t.Fatal("expected metrics handler")
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d want 200", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "openlicensd_build_info") {
		t.Fatalf("expected build_info in body: %s", rec.Body.String())
	}
}

func TestMetricsHandlerDisabled(t *testing.T) {
	cfg := &config.Config{
		Metrics: config.MetricsConfig{
			Enabled: false,
		},
	}

	srv, err := api.New(context.Background(), cfg, nil, testLogger())
	if err != nil {
		t.Fatalf("api server: %v", err)
	}

	if srv.MetricsHandler() != nil {
		t.Fatal("expected nil metrics handler when disabled")
	}
}
