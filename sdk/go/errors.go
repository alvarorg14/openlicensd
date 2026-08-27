package openlicensd

import (
	"errors"
	"fmt"
	"net/http"
	"time"
)

// Sentinel errors for errors.Is checks.
var (
	ErrRateLimited  = errors.New("openlicensd: rate limited")
	ErrBadRequest   = errors.New("openlicensd: bad request")
	ErrUnavailable  = errors.New("openlicensd: service unavailable")
	ErrInvalidKey   = errors.New("openlicensd: invalid license key format")
	ErrInvalidURL   = errors.New("openlicensd: invalid base URL")
	ErrMissingEnv   = errors.New("openlicensd: missing required environment variable")
	ErrMissingProduct = errors.New("openlicensd: product is required")
)

// Reason describes why a license was rejected.
type Reason string

const (
	ReasonNotFound         Reason = "not_found"
	ReasonExpired          Reason = "expired"
	ReasonRevoked          Reason = "revoked"
	ReasonProductMismatch  Reason = "product_mismatch"
	ReasonActivationLimit  Reason = "activation_limit"
	ReasonFingerprintRequired Reason = "fingerprint_required"
	ReasonInvalid          Reason = "invalid"
)

// APIError is returned for HTTP-level failures (transport, 4xx other than
// license rejection, 5xx, rate limiting).
type APIError struct {
	StatusCode int
	Message    string
	RetryAfter time.Duration
}

func (e *APIError) Error() string {
	if e.RetryAfter > 0 {
		return fmt.Sprintf("openlicensd: %s (status %d, retry after %s)", e.Message, e.StatusCode, e.RetryAfter)
	}
	return fmt.Sprintf("openlicensd: %s (status %d)", e.Message, e.StatusCode)
}

func (e *APIError) Is(target error) bool {
	switch target {
	case ErrRateLimited:
		return e.StatusCode == http.StatusTooManyRequests
	case ErrBadRequest:
		return e.StatusCode == http.StatusBadRequest
	case ErrUnavailable:
		return e.StatusCode == http.StatusServiceUnavailable || e.StatusCode >= 500
	default:
		return false
	}
}

// LicenseError is returned by RegistryCredentials when the license is invalid.
type LicenseError struct {
	Reason Reason
}

func (e *LicenseError) Error() string {
	return fmt.Sprintf("openlicensd: license rejected: %s", e.Reason)
}

func parseReason(s string) Reason {
	switch Reason(s) {
	case ReasonNotFound, ReasonExpired, ReasonRevoked, ReasonProductMismatch, ReasonActivationLimit, ReasonFingerprintRequired, ReasonInvalid:
		return Reason(s)
	default:
		if s == "" {
			return ReasonInvalid
		}
		return Reason(s)
	}
}
