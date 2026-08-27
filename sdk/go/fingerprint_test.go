package openlicensd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFingerprintAtCreatesAndReuses(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "machine-id")

	first, err := FingerprintAt(path)
	if err != nil {
		t.Fatalf("FingerprintAt first: %v", err)
	}
	if first == "" {
		t.Fatal("expected non-empty fingerprint")
	}

	second, err := FingerprintAt(path)
	if err != nil {
		t.Fatalf("FingerprintAt second: %v", err)
	}
	if second != first {
		t.Fatalf("expected stable fingerprint, got %q then %q", first, second)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat fingerprint file: %v", err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("expected mode 0600, got %o", info.Mode().Perm())
	}
}

func TestFingerprintRequiresAppName(t *testing.T) {
	if _, err := Fingerprint(""); err == nil {
		t.Fatal("expected error for empty app name")
	}
}

func TestFingerprintAtRequiresPath(t *testing.T) {
	if _, err := FingerprintAt("  "); err == nil {
		t.Fatal("expected error for empty path")
	}
}

func TestClientValidationRequestIncludesFingerprintAndHostname(t *testing.T) {
	client, err := New("https://example.com", "acme", WithFingerprint("fp-123"), WithHostname("host-a"))
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	req := client.validationRequest("KEY", "acme")
	if req.Fingerprint != "fp-123" {
		t.Fatalf("fingerprint=%q", req.Fingerprint)
	}
	if req.Hostname != "host-a" {
		t.Fatalf("hostname=%q", req.Hostname)
	}
}

func TestClientValidationRequestWithoutHostname(t *testing.T) {
	client, err := New("https://example.com", "acme", WithoutHostname())
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	req := client.validationRequest("KEY", "acme")
	if req.Hostname != "" {
		t.Fatalf("expected empty hostname, got %q", req.Hostname)
	}
}

func TestClientValidationRequestDefaultHostname(t *testing.T) {
	client, err := New("https://example.com", "acme")
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	req := client.validationRequest("KEY", "acme")
	if req.Hostname == "" {
		t.Fatal("expected default hostname")
	}
	if strings.TrimSpace(req.Hostname) == "" {
		t.Fatal("expected non-blank default hostname")
	}
}
