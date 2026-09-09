package api

import (
	"strings"
	"testing"
	"time"
)

func TestFormatRFC3339(t *testing.T) {
	t.Parallel()

	ts := time.Date(2026, 1, 15, 12, 30, 45, 123456789, time.UTC)
	formatted := formatRFC3339(ts)
	if strings.Contains(formatted, ".") {
		t.Fatalf("expected second precision without fractional seconds, got %q", formatted)
	}
	parsed, err := time.Parse(timeRFC3339, formatted)
	if err != nil {
		t.Fatalf("parse formatted timestamp: %v", err)
	}
	if !parsed.Equal(time.Date(2026, 1, 15, 12, 30, 45, 0, time.UTC)) {
		t.Fatalf("unexpected parsed time: %v", parsed)
	}
}

func TestFormatRFC3339Ptr(t *testing.T) {
	t.Parallel()

	if formatRFC3339Ptr(nil) != nil {
		t.Fatal("expected nil for nil input")
	}

	ts := time.Date(2026, 1, 15, 12, 30, 45, 0, time.UTC)
	formatted := formatRFC3339Ptr(&ts)
	if formatted == nil {
		t.Fatal("expected non-nil pointer")
	}
	if _, err := time.Parse(timeRFC3339, *formatted); err != nil {
		t.Fatalf("parse formatted timestamp: %v", err)
	}
}

func TestFormatUnixRFC3339(t *testing.T) {
	t.Parallel()

	formatted := formatUnixRFC3339(1725350400)
	if formatted != "2024-09-03T08:00:00Z" {
		t.Fatalf("unexpected formatted timestamp: %q", formatted)
	}
}
