package logging_test

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alvarorg14/openlicensd/server/internal/logging"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func TestRequestLoggerSetsRequestIDAndStatus(t *testing.T) {
	var buf bytes.Buffer
	base := logging.NewTestLogger(&buf, slog.LevelInfo)

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(logging.RequestLogger(base))
	r.Get("/test", func(w http.ResponseWriter, r *http.Request) {
		requestID := middleware.GetReqID(r.Context())
		logger := logging.FromContext(r.Context())
		logger.Info("inside handler")
		w.Header().Set("X-Test-Request-Id", requestID)
		w.WriteHeader(http.StatusTeapot)
		_, _ = w.Write([]byte("hi"))
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusTeapot {
		t.Fatalf("status=%d", rec.Code)
	}
	requestID := rec.Header().Get("X-Test-Request-Id")
	if requestID == "" {
		t.Fatalf("expected request id in handler context")
	}

	lines := bytes.Split(bytes.TrimSpace(buf.Bytes()), []byte("\n"))
	if len(lines) < 2 {
		t.Fatalf("expected at least 2 log lines, got %d: %s", len(lines), buf.String())
	}

	var handlerRecord map[string]any
	if err := json.Unmarshal(lines[0], &handlerRecord); err != nil {
		t.Fatalf("unmarshal handler log: %v", err)
	}
	if handlerRecord["request_id"] != requestID {
		t.Fatalf("handler request_id=%v header=%s", handlerRecord["request_id"], requestID)
	}

	var requestRecord map[string]any
	if err := json.Unmarshal(lines[len(lines)-1], &requestRecord); err != nil {
		t.Fatalf("unmarshal request log: %v", err)
	}
	if requestRecord["status"] != float64(http.StatusTeapot) {
		t.Fatalf("status=%v", requestRecord["status"])
	}
	if requestRecord["bytes"] != float64(2) {
		t.Fatalf("bytes=%v", requestRecord["bytes"])
	}
}

func TestRequestLoggerProbePaths(t *testing.T) {
	var buf bytes.Buffer
	base := logging.NewTestLogger(&buf, slog.LevelDebug)

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(logging.RequestLogger(base))
	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	r.Get("/readyz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	})

	healthReq := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	healthRec := httptest.NewRecorder()
	r.ServeHTTP(healthRec, healthReq)
	if healthRec.Code != http.StatusOK {
		t.Fatalf("healthz status=%d", healthRec.Code)
	}

	readyReq := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	readyRec := httptest.NewRecorder()
	r.ServeHTTP(readyRec, readyReq)
	if readyRec.Code != http.StatusServiceUnavailable {
		t.Fatalf("readyz status=%d", readyRec.Code)
	}

	lines := bytes.Split(bytes.TrimSpace(buf.Bytes()), []byte("\n"))
	if len(lines) != 2 {
		t.Fatalf("expected 2 log lines, got %d: %s", len(lines), buf.String())
	}

	var healthRecord map[string]any
	if err := json.Unmarshal(lines[0], &healthRecord); err != nil {
		t.Fatalf("unmarshal health log: %v", err)
	}
	if healthRecord["level"] != "DEBUG" {
		t.Fatalf("healthz log level=%v, want DEBUG", healthRecord["level"])
	}
	if healthRecord["path"] != "/healthz" {
		t.Fatalf("healthz path=%v", healthRecord["path"])
	}

	var failRecord map[string]any
	if err := json.Unmarshal(lines[1], &failRecord); err != nil {
		t.Fatalf("unmarshal fail log: %v", err)
	}
	if failRecord["level"] != "WARN" {
		t.Fatalf("readyz fail log level=%v, want WARN", failRecord["level"])
	}
	if failRecord["status"] != float64(http.StatusServiceUnavailable) {
		t.Fatalf("readyz fail status=%v", failRecord["status"])
	}
}
