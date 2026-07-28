package license

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"
)

const (
	keyGroups    = 5
	keyGroupLen  = 5
	keyPrefixLen = keyGroupLen
)

const crockfordBase32 = "0123456789ABCDEFGHJKMNPQRSTVWXYZ"

func GenerateKey() (raw string, hash string, prefix string, err error) {
	groups := make([]string, keyGroups)
	for i := 0; i < keyGroups; i++ {
		group := make([]byte, keyGroupLen)
		for j := 0; j < keyGroupLen; j++ {
			b := make([]byte, 1)
			if _, err = rand.Read(b); err != nil {
				return "", "", "", fmt.Errorf("generate random bytes: %w", err)
			}
			group[j] = crockfordBase32[b[0]>>3]
		}
		groups[i] = string(group)
	}

	raw = strings.Join(groups, "-")
	hash = HashKey(raw)
	prefix = groups[0]

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
