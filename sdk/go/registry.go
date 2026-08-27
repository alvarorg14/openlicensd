package openlicensd

import (
	"context"
	"errors"
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
	ExpiresAt int64  `json:"expires_at"`
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

	return &RegistryCredentials{
		Registry:  raw.Registry,
		Username:  raw.Username,
		Secret:    raw.Secret,
		ExpiresAt: time.Unix(raw.ExpiresAt, 0),
	}, nil
}
