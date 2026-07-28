package license

import (
	"testing"
	"time"
)

func TestGenerateKey(t *testing.T) {
	raw, hash, prefix, err := GenerateKey()
	if err != nil {
		t.Fatalf("GenerateKey: %v", err)
	}

	if raw == "" || hash == "" || prefix == "" {
		t.Fatal("expected non-empty key parts")
	}

	if HashKey(raw) != hash {
		t.Fatal("hash mismatch")
	}

	if len(prefix) != keyPrefixLen {
		t.Fatalf("expected prefix length %d, got %d", keyPrefixLen, len(prefix))
	}
}

func TestValidate(t *testing.T) {
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	future := now.Add(24 * time.Hour)
	past := now.Add(-24 * time.Hour)

	tests := []struct {
		name      string
		expiresAt *time.Time
		revoked   bool
		wantValid bool
		wantReason string
	}{
		{"valid forever", nil, false, true, ""},
		{"valid future", &future, false, true, ""},
		{"expired", &past, false, false, "expired"},
		{"revoked", nil, true, false, "revoked"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Validate(tt.expiresAt, tt.revoked, now)
			if result.Valid != tt.wantValid {
				t.Fatalf("valid=%v, want %v", result.Valid, tt.wantValid)
			}
			if result.Reason != tt.wantReason {
				t.Fatalf("reason=%q, want %q", result.Reason, tt.wantReason)
			}
		})
	}
}
