package openlicensd

import (
	"crypto/rand"
	"math/big"
	"strings"
	"time"
)

const (
	keyGroups   = 5
	keyGroupLen = 5
	keyLength   = keyGroups * keyGroupLen

	crockfordBase32 = "0123456789ABCDEFGHJKMNPQRSTVWXYZ"
)

var crockfordSet = func() map[byte]bool {
	set := make(map[byte]bool, len(crockfordBase32))
	for i := 0; i < len(crockfordBase32); i++ {
		set[crockfordBase32[i]] = true
	}
	return set
}()

// NormalizeKey uppercases, trims whitespace, maps ambiguous Crockford characters
// (I/L -> 1, O -> 0), strips dashes, and re-inserts them in 5x5 groups.
func NormalizeKey(raw string) string {
	raw = strings.TrimSpace(raw)
	raw = strings.ToUpper(raw)
	raw = strings.ReplaceAll(raw, "-", "")

	var b strings.Builder
	b.Grow(len(raw))
	for i := 0; i < len(raw); i++ {
		ch := raw[i]
		switch ch {
		case 'I', 'L':
			ch = '1'
		case 'O':
			ch = '0'
		}
		b.WriteByte(ch)
	}

	normalized := b.String()
	if len(normalized) != keyLength {
		return normalized
	}

	groups := make([]string, keyGroups)
	for i := 0; i < keyGroups; i++ {
		start := i * keyGroupLen
		groups[i] = normalized[start : start+keyGroupLen]
	}
	return strings.Join(groups, "-")
}

// ValidateKeyFormat reports whether key matches the Crockford Base32 5x5 format
// after normalization.
func ValidateKeyFormat(key string) bool {
	normalized := NormalizeKey(key)
	if len(normalized) != keyGroups*keyGroupLen+(keyGroups-1) {
		return false
	}

	parts := strings.Split(normalized, "-")
	if len(parts) != keyGroups {
		return false
	}

	for _, part := range parts {
		if len(part) != keyGroupLen {
			return false
		}
		for i := 0; i < len(part); i++ {
			if !crockfordSet[part[i]] {
				return false
			}
		}
	}

	return true
}

func jitter(d time.Duration) time.Duration {
	if d <= 0 {
		return 0
	}
	maxJitter := int64(d / 4)
	if maxJitter <= 0 {
		return d
	}
	n, err := rand.Int(rand.Reader, big.NewInt(maxJitter))
	if err != nil {
		return d
	}
	return d + time.Duration(n.Int64())
}
