package logging_test

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"

	"github.com/alvarorg14/openlicensd/server/internal/config"
	"github.com/alvarorg14/openlicensd/server/internal/logging"
)

func TestNewJSONLogger(t *testing.T) {
	var buf bytes.Buffer
	logger := logging.NewTestLogger(&buf, slog.LevelInfo)
	logger.Info("hello", slog.String("key", "value"))

	var record map[string]any
	if err := json.Unmarshal(buf.Bytes(), &record); err != nil {
		t.Fatalf("unmarshal log: %v", err)
	}
	if record["msg"] != "hello" {
		t.Fatalf("msg=%v", record["msg"])
	}
	if record["key"] != "value" {
		t.Fatalf("key=%v", record["key"])
	}
}

func TestNewFromConfigTextFormat(t *testing.T) {
	logger, err := logging.New(config.LogConfig{Level: "info", Format: "text"})
	if err != nil {
		t.Fatalf("new logger: %v", err)
	}
	if logger == nil {
		t.Fatalf("expected logger")
	}
}

func TestNewInvalidLevel(t *testing.T) {
	_, err := logging.New(config.LogConfig{Level: "trace", Format: "json"})
	if err == nil {
		t.Fatalf("expected error for invalid level")
	}
}

func TestFromContextFallback(t *testing.T) {
	logger := logging.FromContext(t.Context())
	if logger == nil {
		t.Fatalf("expected default logger")
	}
}

func TestContextWithLogger(t *testing.T) {
	var buf bytes.Buffer
	base := logging.NewTestLogger(&buf, slog.LevelInfo)
	ctx := logging.ContextWithLogger(t.Context(), base.With(slog.String("request_id", "req-123")))
	logging.FromContext(ctx).Info("test")

	if !strings.Contains(buf.String(), "req-123") {
		t.Fatalf("log=%s", buf.String())
	}
}
