package license

import (
	"strings"
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

	groups := strings.Split(raw, "-")
	if len(groups) != keyGroups {
		t.Fatalf("expected %d groups, got %d", keyGroups, len(groups))
	}

	for i, group := range groups {
		if len(group) != keyGroupLen {
			t.Fatalf("group %d: expected length %d, got %d", i, keyGroupLen, len(group))
		}
		for _, ch := range group {
			if !strings.ContainsRune(crockfordBase32, ch) {
				t.Fatalf("group %d: invalid character %q", i, ch)
			}
		}
	}

	if prefix != groups[0] {
		t.Fatalf("prefix %q does not match first group %q", prefix, groups[0])
	}
}

func TestComputeExpiry(t *testing.T) {
	from := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)

	if ComputeExpiry(nil, from) != nil {
		t.Fatal("expected nil expiry for perpetual policy")
	}

	days := 30
	expires := ComputeExpiry(&days, from)
	if expires == nil {
		t.Fatal("expected non-nil expiry")
	}
	want := from.Add(30 * 24 * time.Hour)
	if !expires.Equal(want) {
		t.Fatalf("expected %v, got %v", want, expires)
	}
}

func TestValidate(t *testing.T) {
	now := time.Date(2026, 1, 15, 12, 0, 0, 0, time.UTC)
	future := now.Add(24 * time.Hour)
	past := now.Add(-24 * time.Hour)
	graceExpired := now.Add(-72 * time.Hour)

	tests := []struct {
		name              string
		expiresAt         *time.Time
		gracePeriodDays   int
		revoked           bool
		requestedProduct  string
		licenseProduct    string
		wantValid         bool
		wantReason        string
		wantInGracePeriod bool
	}{
		{"valid forever", nil, 0, false, "", "acme", true, "", false},
		{"valid future", &future, 0, false, "", "acme", true, "", false},
		{"expired", &past, 0, false, "", "acme", false, "expired", false},
		{"revoked", nil, 0, true, "", "acme", false, "revoked", false},
		{"product mismatch", nil, 0, false, "other", "acme", false, "product_mismatch", false},
		{"matching product", nil, 0, false, "acme", "acme", true, "", false},
		{"in grace period", &past, 2, false, "", "acme", true, "", true},
		{"past grace period", &graceExpired, 1, false, "", "acme", false, "expired", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Validate(tt.expiresAt, tt.gracePeriodDays, tt.revoked, tt.requestedProduct, tt.licenseProduct, now)
			if result.Valid != tt.wantValid {
				t.Fatalf("valid=%v, want %v", result.Valid, tt.wantValid)
			}
			if result.Reason != tt.wantReason {
				t.Fatalf("reason=%q, want %q", result.Reason, tt.wantReason)
			}
			if result.InGracePeriod != tt.wantInGracePeriod {
				t.Fatalf("in_grace_period=%v, want %v", result.InGracePeriod, tt.wantInGracePeriod)
			}
		})
	}
}
