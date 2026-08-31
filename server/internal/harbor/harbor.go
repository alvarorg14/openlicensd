package harbor

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/alvarorg14/openlicensd/server/internal/logging"
)

type Client struct {
	baseURL    string
	registry   string
	username   string
	password   string
	debug      bool
	logger     *slog.Logger
	httpClient *http.Client
}

type Credentials struct {
	Name      string
	Secret    string
	ExpiresAt int64
}

type robotCreateRequest struct {
	Name        string            `json:"name"`
	Description string            `json:"description,omitempty"`
	Level       string            `json:"level"`
	Duration    int               `json:"duration"`
	Permissions []robotPermission `json:"permissions"`
}

type robotPermission struct {
	Kind      string        `json:"kind"`
	Namespace string        `json:"namespace"`
	Access    []robotAccess `json:"access"`
}

type robotAccess struct {
	Resource string `json:"resource"`
	Action   string `json:"action"`
}

type robotCreatedResponse struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	Secret    string `json:"secret"`
	ExpiresAt int64  `json:"expires_at"`
}

type robotListItem struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	ExpiresAt int64  `json:"expires_at"`
}

func New(baseURL, username, password string, insecureSkipVerify, debug bool, logger *slog.Logger) (*Client, error) {
	parsed, err := url.Parse(strings.TrimRight(baseURL, "/"))
	if err != nil {
		return nil, fmt.Errorf("parse harbor url: %w", err)
	}
	if parsed.Scheme == "" || parsed.Host == "" {
		return nil, fmt.Errorf("invalid harbor url: %s", baseURL)
	}

	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.TLSClientConfig = &tls.Config{
		MinVersion:         tls.VersionTLS12,
		InsecureSkipVerify: insecureSkipVerify, //nolint:gosec // configurable for self-signed Harbor instances
	}

	return &Client{
		baseURL:  parsed.String(),
		registry: parsed.Host,
		username: username,
		password: password,
		debug:    debug,
		logger:   logger,
		httpClient: &http.Client{
			Timeout:   30 * time.Second,
			Transport: transport,
		},
	}, nil
}

func (c *Client) RegistryHost() string {
	return c.registry
}

func (c *Client) CreateEphemeralRobot(ctx context.Context, projects []string, durationDays int, namePrefix, keyPrefix string) (*Credentials, error) {
	if len(projects) == 0 {
		return nil, fmt.Errorf("at least one harbor project is required")
	}

	name := buildRobotName(namePrefix, keyPrefix)
	level := "project"
	if len(projects) > 1 {
		level = "system"
	}

	permissions := make([]robotPermission, 0, len(projects))
	for _, project := range projects {
		permissions = append(permissions, robotPermission{
			Kind:      "project",
			Namespace: project,
			Access: []robotAccess{
				{Resource: "repository", Action: "pull"},
			},
		})
	}

	body := robotCreateRequest{
		Name:        name,
		Description: "ephemeral robot created by openlicensd",
		Level:       level,
		Duration:    durationDays,
		Permissions: permissions,
	}

	payload, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal robot request: %w", err)
	}

	endpoint := c.baseURL + "/api/v2.0/robots"
	c.logDebug(ctx, "harbor request", slog.String("method", http.MethodPost), slog.String("url", endpoint), slog.String("user", c.username), slog.String("body", string(payload)))

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("create robot request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Basic "+basicAuth(c.username, c.password))

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.logDebug(ctx, "harbor transport error", slog.String("method", http.MethodPost), slog.String("url", endpoint), slog.Any("err", err))
		return nil, fmt.Errorf("create robot: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read robot response: %w", err)
	}

	c.logResponse(ctx, http.MethodPost, endpoint, resp.StatusCode, respBody)

	if resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("create robot: status %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}

	var created robotCreatedResponse
	if err := json.Unmarshal(respBody, &created); err != nil {
		return nil, fmt.Errorf("decode robot response: %w", err)
	}
	if created.Name == "" || created.Secret == "" {
		return nil, fmt.Errorf("create robot: missing name or secret in response")
	}

	c.logDebug(ctx, "harbor robot created", slog.String("name", created.Name), slog.Int64("expires_at", created.ExpiresAt))

	return &Credentials{
		Name:      created.Name,
		Secret:    created.Secret,
		ExpiresAt: created.ExpiresAt,
	}, nil
}

func (c *Client) CleanupExpiredRobots(ctx context.Context, namePrefix string) error {
	endpoint := c.baseURL + "/api/v2.0/robots"
	c.logDebug(ctx, "harbor request", slog.String("method", http.MethodGet), slog.String("url", endpoint), slog.String("user", c.username))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return fmt.Errorf("list robots request: %w", err)
	}
	req.Header.Set("Authorization", "Basic "+basicAuth(c.username, c.password))

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.logDebug(ctx, "harbor transport error", slog.String("method", http.MethodGet), slog.String("url", endpoint), slog.Any("err", err))
		return fmt.Errorf("list robots: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read robots response: %w", err)
	}

	c.logResponse(ctx, http.MethodGet, endpoint, resp.StatusCode, respBody)

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("list robots: status %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}

	var robots []robotListItem
	if err := json.Unmarshal(respBody, &robots); err != nil {
		return fmt.Errorf("decode robots response: %w", err)
	}

	now := time.Now().Unix()
	for _, robot := range robots {
		if !strings.Contains(strings.ToLower(robot.Name), strings.ToLower(namePrefix)) {
			continue
		}
		if robot.ExpiresAt == -1 || robot.ExpiresAt > now {
			continue
		}

		deleteEndpoint := fmt.Sprintf("%s/api/v2.0/robots/%d", c.baseURL, robot.ID)
		c.logDebug(ctx, "harbor request", slog.String("method", http.MethodDelete), slog.String("url", deleteEndpoint), slog.String("user", c.username), slog.String("robot", robot.Name))

		deleteReq, err := http.NewRequestWithContext(ctx, http.MethodDelete, deleteEndpoint, nil)
		if err != nil {
			continue
		}
		deleteReq.Header.Set("Authorization", "Basic "+basicAuth(c.username, c.password))

		deleteResp, err := c.httpClient.Do(deleteReq)
		if err != nil {
			c.logDebug(ctx, "harbor transport error", slog.String("method", http.MethodDelete), slog.String("url", deleteEndpoint), slog.Any("err", err))
			continue
		}

		deleteBody, _ := io.ReadAll(deleteResp.Body)
		_ = deleteResp.Body.Close()
		c.logResponse(ctx, http.MethodDelete, deleteEndpoint, deleteResp.StatusCode, deleteBody)
	}

	return nil
}

func (c *Client) logDebug(ctx context.Context, msg string, attrs ...any) {
	if !c.debug {
		return
	}
	c.loggerFor(ctx).Debug(msg, attrs...)
}

func (c *Client) logResponse(ctx context.Context, method, endpoint string, status int, body []byte) {
	if !c.debug && status >= 200 && status < 300 {
		return
	}

	response := redactSecrets(strings.TrimSpace(string(body)))
	logger := c.loggerFor(ctx)
	if c.debug {
		logger.Debug("harbor response",
			slog.String("method", method),
			slog.String("url", endpoint),
			slog.Int("status", status),
			slog.String("body", response),
		)
		return
	}

	logger.Warn("harbor response",
		slog.String("method", method),
		slog.String("url", endpoint),
		slog.Int("status", status),
		slog.String("body", response),
	)
}

func (c *Client) loggerFor(ctx context.Context) *slog.Logger {
	logger := logging.FromContext(ctx)
	if c.logger != nil && logger == slog.Default() {
		return c.logger
	}
	return logger
}

func redactSecrets(body string) string {
	if body == "" {
		return body
	}

	var parsed map[string]any
	if err := json.Unmarshal([]byte(body), &parsed); err != nil {
		return body
	}

	if _, ok := parsed["secret"]; ok {
		parsed["secret"] = "[REDACTED]"
	}

	redacted, err := json.Marshal(parsed)
	if err != nil {
		return body
	}
	return string(redacted)
}

func buildRobotName(namePrefix, keyPrefix string) string {
	sanitizedPrefix := sanitizeRobotNamePart(namePrefix)
	sanitizedKeyPrefix := sanitizeRobotNamePart(keyPrefix)
	return strings.ToLower(fmt.Sprintf("%s-%s-%d", sanitizedPrefix, sanitizedKeyPrefix, time.Now().UnixNano()))
}

func sanitizeRobotNamePart(value string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(value) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			b.WriteRune(r)
		}
	}

	out := strings.Trim(b.String(), "-")
	if out == "" {
		return "robot"
	}
	return out
}

func basicAuth(username, password string) string {
	return base64.StdEncoding.EncodeToString([]byte(username + ":" + password))
}
