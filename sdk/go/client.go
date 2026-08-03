package openlicensd

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	defaultTimeout    = 10 * time.Second
	defaultUserAgent  = "openlicensd-go-sdk"
	defaultRetryMax   = 2
	defaultRetryDelay = 200 * time.Millisecond

	envURL     = "OPENLICENSD_URL"
	envProduct = "OPENLICENSD_PRODUCT"
)

// Client is an OpenLicensd HTTP API client for license validation.
type Client struct {
	baseURL          *url.URL
	product          string
	allowAnyProduct  bool
	httpClient       *http.Client
	userAgent        string
	retryMaxAttempts int
	retryBaseDelay   time.Duration
	retryEnabled     bool
}

// Option configures a Client.
type Option func(*Client)

// WithHTTPClient sets a custom HTTP client.
func WithHTTPClient(c *http.Client) Option {
	return func(cl *Client) {
		cl.httpClient = c
	}
}

// WithTimeout sets the HTTP client timeout.
func WithTimeout(d time.Duration) Option {
	return func(cl *Client) {
		cl.httpClient.Timeout = d
	}
}

// WithUserAgent sets the User-Agent header.
func WithUserAgent(ua string) Option {
	return func(cl *Client) {
		cl.userAgent = ua
	}
}

// WithRetry configures retry behavior for Validate. maxAttempts includes the
// initial request (e.g. 2 means one retry). Set maxAttempts to 1 to disable.
func WithRetry(maxAttempts int, baseDelay time.Duration) Option {
	return func(cl *Client) {
		if maxAttempts < 1 {
			maxAttempts = 1
		}
		cl.retryMaxAttempts = maxAttempts
		cl.retryBaseDelay = baseDelay
		cl.retryEnabled = maxAttempts > 1
	}
}

// WithAnyProduct disables product scoping. Every validation request is sent
// without a product code, so the server accepts a valid key for any product.
// Use only when product scoping is intentionally not required.
func WithAnyProduct() Option {
	return func(cl *Client) {
		cl.allowAnyProduct = true
		cl.product = ""
	}
}

// New creates a client for the given OpenLicensd server base URL and product
// code. baseURL must be an absolute http or https URL. Trailing slashes are
// stripped. Product is always sent on validation requests unless
// WithAnyProduct is passed.
func New(baseURL, product string, opts ...Option) (*Client, error) {
	parsed, err := parseBaseURL(baseURL)
	if err != nil {
		return nil, err
	}

	cl := &Client{
		baseURL:          parsed,
		product:          strings.TrimSpace(product),
		httpClient:       &http.Client{Timeout: defaultTimeout},
		userAgent:        defaultUserAgent,
		retryMaxAttempts: defaultRetryMax,
		retryBaseDelay:   defaultRetryDelay,
		retryEnabled:     true,
	}

	for _, opt := range opts {
		opt(cl)
	}

	if cl.product == "" && !cl.allowAnyProduct {
		return nil, ErrMissingProduct
	}

	if cl.httpClient == nil {
		cl.httpClient = &http.Client{Timeout: defaultTimeout}
	}

	return cl, nil
}

// NewFromEnv creates a client using OPENLICENSD_URL and OPENLICENSD_PRODUCT
// from the environment.
//
// WARNING: Reading the server URL from the environment allows the operator to
// point validation at an arbitrary server. Use this only in self-hosted
// deployments where the operator legitimately controls the license server.
// Vendors embedding license enforcement should use New with a build-time URL.
func NewFromEnv(opts ...Option) (*Client, error) {
	baseURL := strings.TrimSpace(os.Getenv(envURL))
	if baseURL == "" {
		return nil, fmt.Errorf("%w: %s", ErrMissingEnv, envURL)
	}

	product := strings.TrimSpace(os.Getenv(envProduct))
	if product == "" {
		return nil, fmt.Errorf("%w: %s", ErrMissingEnv, envProduct)
	}

	return New(baseURL, product, opts...)
}

func parseBaseURL(raw string) (*url.URL, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, fmt.Errorf("%w: empty URL", ErrInvalidURL)
	}

	parsed, err := url.Parse(raw)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrInvalidURL, err)
	}

	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return nil, fmt.Errorf("%w: scheme must be http or https", ErrInvalidURL)
	}

	if parsed.Host == "" {
		return nil, fmt.Errorf("%w: missing host", ErrInvalidURL)
	}

	parsed.Path = strings.TrimRight(parsed.Path, "/")
	parsed.RawQuery = ""
	parsed.Fragment = ""

	return parsed, nil
}

func (c *Client) endpoint(path string) (string, error) {
	return url.JoinPath(c.baseURL.String(), path)
}

type apiErrorResponse struct {
	Error string `json:"error"`
}

func (c *Client) doJSON(ctx context.Context, method, path string, reqBody any, respBody any) (int, error) {
	endpoint, err := c.endpoint(path)
	if err != nil {
		return 0, err
	}

	var body io.Reader
	if reqBody != nil {
		encoded, err := json.Marshal(reqBody)
		if err != nil {
			return 0, fmt.Errorf("openlicensd: encode request: %w", err)
		}
		body = bytes.NewReader(encoded)
	}

	httpReq, err := http.NewRequestWithContext(ctx, method, endpoint, body)
	if err != nil {
		return 0, fmt.Errorf("openlicensd: create request: %w", err)
	}

	if reqBody != nil {
		httpReq.Header.Set("Content-Type", "application/json")
	}
	httpReq.Header.Set("Accept", "application/json")
	httpReq.Header.Set("User-Agent", c.userAgent)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return 0, err
	}
	defer func() { _ = resp.Body.Close() }()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return resp.StatusCode, fmt.Errorf("openlicensd: read response: %w", err)
	}

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		if respBody != nil && len(respBytes) > 0 {
			if err := json.Unmarshal(respBytes, respBody); err != nil {
				return resp.StatusCode, fmt.Errorf("openlicensd: decode response: %w", err)
			}
		}
		return resp.StatusCode, nil
	}

	return resp.StatusCode, c.decodeError(resp, respBytes)
}

func (c *Client) decodeError(resp *http.Response, body []byte) error {
	var apiErr apiErrorResponse
	_ = json.Unmarshal(body, &apiErr)

	message := apiErr.Error
	if message == "" {
		message = http.StatusText(resp.StatusCode)
	}

	err := &APIError{
		StatusCode: resp.StatusCode,
		Message:    message,
	}

	if resp.StatusCode == http.StatusTooManyRequests {
		if retryAfter := resp.Header.Get("Retry-After"); retryAfter != "" {
			if seconds, parseErr := strconv.Atoi(retryAfter); parseErr == nil && seconds > 0 {
				err.RetryAfter = time.Duration(seconds) * time.Second
			}
		}
	}

	return err
}

func (c *Client) doWithRetry(ctx context.Context, fn func() (int, error)) error {
	attempts := c.retryMaxAttempts
	if !c.retryEnabled || attempts < 1 {
		attempts = 1
	}

	var lastErr error
	for attempt := 0; attempt < attempts; attempt++ {
		if attempt > 0 {
			delay := c.retryDelay(attempt, lastErr)
			if err := sleepContext(ctx, delay); err != nil {
				return err
			}
		}

		status, err := fn()
		if err == nil {
			return nil
		}

		lastErr = err
		if !isRetryable(err, status) || attempt == attempts-1 {
			return err
		}
	}

	return lastErr
}

func (c *Client) retryDelay(attempt int, err error) time.Duration {
	var apiErr *APIError
	if errors.As(err, &apiErr) && apiErr.RetryAfter > 0 {
		return apiErr.RetryAfter
	}

	delay := c.retryBaseDelay
	for i := 1; i < attempt; i++ {
		delay *= 2
	}
	return jitter(delay)
}

func isRetryable(err error, status int) bool {
	if err == nil {
		return false
	}

	var apiErr *APIError
	if errors.As(err, &apiErr) {
		if apiErr.StatusCode == http.StatusTooManyRequests {
			return true
		}
		if apiErr.StatusCode >= 500 {
			return true
		}
		return false
	}

	// Network errors are retryable.
	return status == 0
}

func sleepContext(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		return nil
	}
	timer := time.NewTimer(d)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
