package metrics_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/alvarorg14/openlicensd/server/internal/metrics"
	"github.com/go-chi/chi/v5"
)

func scrapeMetrics(m *metrics.Metrics) string {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	m.Handler().ServeHTTP(rec, req)
	return rec.Body.String()
}

func TestRecordValidation(t *testing.T) {
	m := metrics.New("test", nil)

	m.RecordValidation(true, "")
	m.RecordValidation(false, "activation_limit")

	body := scrapeMetrics(m)
	if !strings.Contains(body, `openlicensd_license_validations_total{reason="ok",result="valid"} 1`) {
		t.Fatalf("missing valid validation metric:\n%s", body)
	}
	if !strings.Contains(body, `openlicensd_license_validations_total{reason="activation_limit",result="invalid"} 1`) {
		t.Fatalf("missing activation_limit metric:\n%s", body)
	}
}

func TestBuildInfo(t *testing.T) {
	m := metrics.New("v0.7.0-test", nil)

	body := scrapeMetrics(m)
	if !strings.Contains(body, `openlicensd_build_info{version="v0.7.0-test"} 1`) {
		t.Fatalf("build_info missing from body:\n%s", body)
	}
}

func TestPoolCollectorRegistered(t *testing.T) {
	m := metrics.New("test", func() *metrics.PoolStat { return nil })

	body := scrapeMetrics(m)
	if !strings.Contains(body, "openlicensd_db_pool_acquired_connections") {
		t.Fatalf("expected db pool metric names in body:\n%s", body)
	}
}

func TestMiddlewareRoutePattern(t *testing.T) {
	m := metrics.New("test", nil)

	r := chi.NewRouter()
	r.Use(metrics.Middleware(m))
	r.Get("/licenses/{id}", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/licenses/abc-123", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	body := scrapeMetrics(m)
	if !strings.Contains(body, `openlicensd_http_requests_total{method="GET",route="/licenses/{id}",status="200"} 1`) {
		t.Fatalf("missing route pattern metric:\n%s", body)
	}
}

func TestMiddlewareUnmatchedRoute(t *testing.T) {
	m := metrics.New("test", nil)

	r := chi.NewRouter()
	r.Use(metrics.Middleware(m))
	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/missing", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	body := scrapeMetrics(m)
	if !strings.Contains(body, `openlicensd_http_requests_total{method="GET",route="other",status="404"} 1`) {
		t.Fatalf("missing other route metric:\n%s", body)
	}
}
