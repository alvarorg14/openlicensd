# Go SDK

The official Go client for integrating OpenLicensd license validation into CLIs, APIs, and services. The SDK runs on any GOOS/GOARCH supported by Go 1.27+; the **server** publishes Linux amd64/arm64 binaries and images only (see [deployment.md](../deployment.md#platforms)).

## Install

```bash
go get github.com/alvarorg14/openlicensd/sdk/go@v1.0.0
```

Package documentation: [pkg.go.dev/github.com/alvarorg14/openlicensd/sdk/go](https://pkg.go.dev/github.com/alvarorg14/openlicensd/sdk/go)

## Overview

The SDK covers the **public, unauthenticated** API:

- `POST /api/v1/validate` — license validation
- `POST /api/v1/registry-credentials` — Harbor credentials (when enabled on the server)
- `GET /healthz`, `GET /readyz` — health probes

Admin API endpoints (license, product, policy, user, and audit management) are **not included in the SDK**. Automate admin operations with the HTTP API using scoped Bearer API tokens (`Authorization: Bearer <token>`) — no CSRF header required. Since v0.8.0, most admin routes accept Bearer tokens with role-based access (`admin`, `operator`, or `viewer`).

Session cookies and CSRF remain required for browser-based admin UI flows and for session-only routes: API token management (`/api/v1/api-tokens`), logout, and password change.

```bash
curl -s https://licenses.example.com/api/v1/licenses \
  -H "Authorization: Bearer olsd_..."
```

See [api.md](../api.md#6-api-tokens-for-automation) for creating tokens, role requirements, and the full endpoint access matrix.

## Basic usage

```go
import openlicensd "github.com/alvarorg14/openlicensd/sdk/go"

fp, err := openlicensd.Fingerprint("my-cli")
if err != nil {
    log.Fatal(err)
}

client, err := openlicensd.New(
    "https://licenses.example.com",
    "acme-widget",
    openlicensd.WithFingerprint(fp),
)
if err != nil {
    log.Fatal(err)
}

result, err := client.Validate(ctx, licenseKey)
if err != nil {
    log.Fatal(err)
}
if !result.Valid {
    log.Fatalf("rejected: %s", result.Reason)
}
```

## Configuration

### Required arguments

| Argument | Description |
|----------|-------------|
| `baseURL` | OpenLicensd server URL (`https://licenses.example.com`) |
| `product` | Product code to scope validation (`acme-widget`) |

Both are required in `New()`. An empty product disables the server's product-mismatch check, so the SDK enforces it at construction time.

### Vendor vs self-hosted

**Vendor binaries** (software you ship to customers) should hard-code the server URL at build time:

```bash
go build -ldflags "-X main.licenseURL=https://licenses.example.com" ./cmd/myapp
```

**Self-hosted deployments** where the operator owns the license server may use:

```go
client, err := openlicensd.NewFromEnv()
```

This reads `OPENLICENSD_URL` and `OPENLICENSD_PRODUCT` from the environment. The function documents the trust implications — operators can redirect validation to an arbitrary server.

## Validation semantics

The dual validation envelope is frozen before v1.0. `/validate` always returns HTTP 200 for business outcomes; `/registry-credentials` returns HTTP 403 with the same reason codes in the `error` field. The SDK reflects this:

- `Validate` returns `(result, nil)` when the key is invalid — check `result.Valid` and `result.Reason`
- Rejection reasons: `not_found`, `expired`, `revoked`, `product_mismatch`, `fingerprint_required`, `activation_limit`

When the server enforces `max_activations`, configure the client with a stable machine fingerprint:

```go
fp, err := openlicensd.Fingerprint("my-cli")
client, err := openlicensd.New(baseURL, product, openlicensd.WithFingerprint(fp))
```

`Fingerprint` persists a UUID under the OS user config directory (`<UserConfigDir>/my-cli/machine-id`). Use `FingerprintAt(path)` when you need a custom location (for example a mounted volume in CI). The SDK sends `os.Hostname()` by default; pass `openlicensd.WithoutHostname()` to omit it.

`/registry-credentials` returns HTTP 403 for invalid licenses (same reason codes, plus `invalid` as a fallback). The SDK maps this to `*LicenseError`.

`ErrInvalidKey` is a sentinel for client-side `ValidateKeyFormat` checks. `Validate` and `ValidateProduct` do not return it; malformed keys are rejected by the server as `Valid=false` (typically `ReasonInvalid`).

## Advanced patterns

### Cached validation

Reduce server round-trips with a TTL cache:

```go
validator := openlicensd.NewCachedValidator(client, 5*time.Minute)
result, err := validator.Validate(ctx, key)
```

Any `ValidationResult` returned without error is cached, including `Valid: false`. Transport errors are not cached. The cache map has no size limit; expired entries are skipped on read but not pruned automatically. Call `Invalidate`, `InvalidateProduct`, or `Clear` when validating many distinct keys.

Use `ValidateProduct(ctx, key, product)` and `InvalidateProduct(key, product)` when validating against a product other than the client's configured product.

### Background guard

For services that need continuous license enforcement with offline tolerance:

```go
guard, err := openlicensd.NewGuard(ctx, client, key,
    openlicensd.WithInterval(time.Hour),
    openlicensd.WithOfflineGrace(24*time.Hour),
)
defer guard.Stop()
```

`NewGuard` runs the first `ValidateProduct` synchronously. If it returns a non-nil error (for example when the server is unreachable), construction fails and the background loop never starts. Offline grace applies only to later transport failures after a successful start. Pass `WithProduct(product)` when revalidating against a product other than the client's configured product. `Stop()` is safe to call more than once.

### Key format validation

Validate key format locally before calling the server:

```go
if !openlicensd.ValidateKeyFormat(key) {
    return fmt.Errorf("invalid key format")
}
key = openlicensd.NormalizeKey(key)
```

The server also normalizes keys before hash lookup on `/validate` and `/registry-credentials`, so client-side normalization is optional for matching but still recommended for local format checks and UX.

You may return `openlicensd.ErrInvalidKey` instead of a custom error when using `errors.Is`.

## Retries

`Validate` retries on rate limiting (429), server errors (5xx), and network failures. Default: 2 attempts with exponential backoff and `Retry-After` support.

`RegistryCredentials` does **not** retry — the endpoint creates a Harbor robot account as a side effect.

`expires_at` on both `ValidationResult` and `RegistryCredentials` is an RFC3339 timestamp string on the wire. The SDK unmarshals it into `time.Time` (or `*time.Time` for validation).

## Compatibility

SDK releases are independent from server releases:

| SDK | Server |
|-----|--------|
| v1.0.x | >= 1.0.0 |

Tag format: `sdk/go/vX.Y.Z` (note the `v` prefix required by Go modules). Releases are published from Release Drafter drafts on GitHub (see [CONTRIBUTING.md](../../CONTRIBUTING.md)).

HTTP `/api/v1` stability rules (independent of SDK versioning) are defined in [COMPATIBILITY.md](../../COMPATIBILITY.md).

## See also

- [sdk/go/README.md](../../sdk/go/README.md) — package README with full API reference
- [api.md](../api.md) — HTTP API documentation
- [openapi.yaml](../openapi.yaml) — OpenAPI specification
