package license

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"time"
)

const (
	keyPrefixLen = 8
	keyBytes     = 32
)

func GenerateKey() (raw string, hash string, prefix string, err error) {
	buf := make([]byte, keyBytes)
	if _, err = rand.Read(buf); err != nil {
		return "", "", "", fmt.Errorf("generate random bytes: %w", err)
	}

	raw = "ol_" + base64.RawURLEncoding.EncodeToString(buf)
	hash = HashKey(raw)

	if len(raw) < keyPrefixLen {
		prefix = raw
	} else {
		prefix = raw[:keyPrefixLen]
	}

	return raw, hash, prefix, nil
}

func HashKey(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

type ValidationResult struct {
	Valid     bool       `json:"valid"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
	Reason    string     `json:"reason,omitempty"`
}

func Validate(expiresAt *time.Time, revoked bool, now time.Time) ValidationResult {
	if revoked {
		return ValidationResult{Valid: false, Reason: "revoked"}
	}

	if expiresAt != nil && now.After(*expiresAt) {
		return ValidationResult{Valid: false, ExpiresAt: expiresAt, Reason: "expired"}
	}

	return ValidationResult{Valid: true, ExpiresAt: expiresAt}
}
