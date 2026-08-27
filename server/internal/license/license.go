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

func ComputeExpiry(durationDays *int, from time.Time) *time.Time {
	if durationDays == nil {
		return nil
	}
	expires := from.Add(time.Duration(*durationDays) * 24 * time.Hour)
	return &expires
}

type ValidationResult struct {
	Valid            bool       `json:"valid"`
	ExpiresAt        *time.Time `json:"expires_at,omitempty"`
	Reason           string     `json:"reason,omitempty"`
	Product          string     `json:"product,omitempty"`
	Policy           string     `json:"policy,omitempty"`
	InGracePeriod    bool       `json:"in_grace_period,omitempty"`
	ActivationCount  *int64     `json:"activation_count,omitempty"`
	MaxActivations   *int       `json:"max_activations,omitempty"`
}

func Validate(expiresAt *time.Time, gracePeriodDays int, revoked bool, requestedProduct, licenseProduct string, now time.Time) ValidationResult {
	result := ValidationResult{
		ExpiresAt: expiresAt,
		Product:   licenseProduct,
	}

	if requestedProduct != "" && requestedProduct != licenseProduct {
		result.Valid = false
		result.Reason = "product_mismatch"
		return result
	}

	if revoked {
		result.Valid = false
		result.Reason = "revoked"
		return result
	}

	if expiresAt != nil {
		graceEnd := expiresAt.Add(time.Duration(gracePeriodDays) * 24 * time.Hour)
		if now.After(graceEnd) {
			result.Valid = false
			result.Reason = "expired"
			return result
		}
		if now.After(*expiresAt) {
			result.Valid = true
			result.InGracePeriod = true
			return result
		}
	}

	result.Valid = true
	return result
}
