package api

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/alvarorg14/openlicensd/server/internal/license"
	"github.com/google/uuid"
)

func assertRFC3339StringField(t *testing.T, field string, value any) {
	t.Helper()

	formatted, ok := value.(string)
	if !ok {
		t.Fatalf("%s: expected string, got %T (%v)", field, value, value)
	}
	if _, err := time.Parse(timeRFC3339, formatted); err != nil {
		t.Fatalf("%s: invalid RFC3339 timestamp %q: %v", field, formatted, err)
	}
}

func TestValidationToResponse(t *testing.T) {
	t.Parallel()

	expires := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	result := license.ValidationResult{
		Valid:     true,
		ExpiresAt: &expires,
		Product:   "acme-widget",
	}

	raw, err := json.Marshal(validationToResponse(result))
	if err != nil {
		t.Fatalf("marshal validation response: %v", err)
	}

	var decoded map[string]any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("unmarshal validation response: %v", err)
	}
	assertRFC3339StringField(t, "expires_at", decoded["expires_at"])
}

func TestLicenseResponseTimestamps(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 3, 15, 10, 0, 0, 0, time.UTC)
	expires := now.Add(24 * time.Hour)
	lastValidated := now.Add(time.Hour)

	raw, err := json.Marshal(licenseResponse{
		ID:              mustParseUUID("00000000-0000-0000-0000-000000000001"),
		Label:           "test",
		KeyPrefix:       "abcd",
		ProductID:       mustParseUUID("00000000-0000-0000-0000-000000000002"),
		ProductCode:     "acme",
		ProductName:     "Acme",
		PolicyID:        mustParseUUID("00000000-0000-0000-0000-000000000003"),
		PolicyName:      "Default",
		ExpiresAt:       formatRFC3339Ptr(&expires),
		ActivatedAt:     formatRFC3339Ptr(&now),
		CreatedAt:       formatRFC3339(now),
		LastValidatedAt: formatRFC3339Ptr(&lastValidated),
	})
	if err != nil {
		t.Fatalf("marshal license response: %v", err)
	}

	var decoded map[string]any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("unmarshal license response: %v", err)
	}

	for _, field := range []string{"expires_at", "activated_at", "created_at", "last_validated_at"} {
		assertRFC3339StringField(t, field, decoded[field])
	}
}

func TestRegistryCredentialsResponseTimestamp(t *testing.T) {
	t.Parallel()

	raw, err := json.Marshal(registryCredentialsResponse{
		Registry:  "harbor.example.com",
		Username:  "robot$user",
		Secret:    "secret",
		ExpiresAt: formatUnixRFC3339(1725350400),
	})
	if err != nil {
		t.Fatalf("marshal registry credentials response: %v", err)
	}

	var decoded map[string]any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("unmarshal registry credentials response: %v", err)
	}
	assertRFC3339StringField(t, "expires_at", decoded["expires_at"])
}

func mustParseUUID(value string) uuid.UUID {
	id, err := uuid.Parse(value)
	if err != nil {
		panic(err)
	}
	return id
}
