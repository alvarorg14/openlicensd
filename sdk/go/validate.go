package openlicensd

import (
	"context"
	"net/http"
	"time"
)

// ValidationResult is the response from POST /api/v1/validate.
type ValidationResult struct {
	Valid           bool       `json:"valid"`
	ExpiresAt       *time.Time `json:"expires_at,omitempty"`
	Reason          Reason     `json:"reason,omitempty"`
	Product         string     `json:"product,omitempty"`
	Policy          string     `json:"policy,omitempty"`
	InGracePeriod   bool       `json:"in_grace_period,omitempty"`
	ActivationCount *int64     `json:"activation_count,omitempty"`
	MaxActivations  *int       `json:"max_activations,omitempty"`
}

// TimeUntilExpiry returns the duration until the license expires, or zero if
// there is no expiry or the license is already past expiry.
func (r ValidationResult) TimeUntilExpiry() time.Duration {
	if r.ExpiresAt == nil {
		return 0
	}
	remaining := time.Until(*r.ExpiresAt)
	if remaining < 0 {
		return 0
	}
	return remaining
}

// ExpiresWithin reports whether the license expires within d from now.
func (r ValidationResult) ExpiresWithin(d time.Duration) bool {
	if r.ExpiresAt == nil {
		return false
	}
	return time.Until(*r.ExpiresAt) <= d
}

type validateRequest struct {
	Key         string `json:"key"`
	Product     string `json:"product,omitempty"`
	Fingerprint string `json:"fingerprint,omitempty"`
	Hostname    string `json:"hostname,omitempty"`
}

// Validate checks a license key against the client's configured product.
// An invalid license returns a result with Valid=false and a nil error.
func (c *Client) Validate(ctx context.Context, key string) (ValidationResult, error) {
	return c.ValidateProduct(ctx, key, c.product)
}

// ValidateProduct checks a license key against the given product code.
// An invalid license returns a result with Valid=false and a nil error.
func (c *Client) ValidateProduct(ctx context.Context, key, product string) (ValidationResult, error) {
	req := c.validationRequest(key, product)

	var result ValidationResult
	err := c.doWithRetry(ctx, func() (int, error) {
		status, err := c.doJSON(ctx, http.MethodPost, "/api/v1/validate", req, &result)
		return status, err
	})
	if err != nil {
		return ValidationResult{}, err
	}

	return result, nil
}
