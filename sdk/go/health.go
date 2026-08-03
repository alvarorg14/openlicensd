package openlicensd

import (
	"context"
	"net/http"
)

// Health checks whether the server process is alive (GET /healthz).
func (c *Client) Health(ctx context.Context) error {
	_, err := c.doJSON(ctx, http.MethodGet, "/healthz", nil, nil)
	return err
}

// Ready checks whether the server is ready to serve traffic (GET /readyz).
// Returns an error when the database is unavailable (503).
func (c *Client) Ready(ctx context.Context) error {
	_, err := c.doJSON(ctx, http.MethodGet, "/readyz", nil, nil)
	return err
}
