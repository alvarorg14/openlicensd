package openlicensd

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"
)

// RegistryCredentials holds ephemeral Harbor registry credentials.
type RegistryCredentials struct {
	Registry  string    `json:"registry"`
	Username  string    `json:"username"`
	Secret    string    `json:"secret"`
	ExpiresAt time.Time `json:"-"`
}

type registryCredentialsResponse struct {
	Registry  string `json:"registry"`
	Username  string `json:"username"`
	Secret    string `json:"secret"`
	ExpiresAt string `json:"expires_at"`
}

// RegistryCredentials issues short-lived Harbor registry credentials for a
// valid license key. Unlike Validate, an invalid license returns a LicenseError.
// This method does not retry because the endpoint creates a Harbor robot account
// as a side effect.
func (c *Client) RegistryCredentials(ctx context.Context, key string) (*RegistryCredentials, error) {
	req := c.validationRequest(key, c.product)

	var raw registryCredentialsResponse
	status, err := c.doJSON(ctx, http.MethodPost, "/api/v1/registry-credentials", req, &raw)
	if err != nil {
		if status == http.StatusForbidden {
			var apiErr *APIError
			if errors.As(err, &apiErr) {
				return nil, &LicenseError{Reason: parseReason(apiErr.Message)}
			}
		}
		return nil, err
	}

	expiresAt, err := time.Parse(time.RFC3339, raw.ExpiresAt)
	if err != nil {
		return nil, fmt.Errorf("parse expires_at: %w", err)
	}

	return &RegistryCredentials{
		Registry:  raw.Registry,
		Username:  raw.Username,
		Secret:    raw.Secret,
		ExpiresAt: expiresAt,
	}, nil
}
