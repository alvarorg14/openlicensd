package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alvarorg14/openlicensd/server/internal/logging"
)

func TestWriteInternalError(t *testing.T) {
	var buf bytes.Buffer
	logger := logging.NewTestLogger(&buf, slog.LevelInfo)

	r := httptest.NewRequest(http.MethodGet, "/test", nil)
	ctx := logging.ContextWithLogger(r.Context(), logger)
	r = r.WithContext(ctx)

	rec := httptest.NewRecorder()
	writeInternalError(rec, r, errors.New("db timeout"), "failed to load license")

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status=%d", rec.Code)
	}
	var body map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal body: %v", err)
	}
	if body["error"] != "failed to load license" {
		t.Fatalf("body=%v", body)
	}
	if !bytes.Contains(buf.Bytes(), []byte("db timeout")) {
		t.Fatalf("log=%s", buf.String())
	}
}
