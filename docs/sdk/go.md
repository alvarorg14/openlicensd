# Go SDK

The official Go client for integrating OpenLicensd license validation into CLIs, APIs, and services.

## Install

```bash
go get github.com/alvarorg14/openlicensd/sdk/go@v0.1.0
```

Package documentation: [pkg.go.dev/github.com/alvarorg14/openlicensd/sdk/go](https://pkg.go.dev/github.com/alvarorg14/openlicensd/sdk/go)

## Overview

The SDK covers the **public, unauthenticated** API:

- `POST /api/v1/validate` — license validation
- `POST /api/v1/registry-credentials` — Harbor credentials (when enabled on the server)
- `GET /healthz`, `GET /readyz` — health probes

Admin API endpoints (license/product/policy management) are not included. Those require browser session cookies and CSRF tokens.

## Basic usage

```go
import openlicensd "github.com/alvarorg14/openlicensd/sdk/go"

client, err := openlicensd.New("https://licenses.example.com", "acme-widget")
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

`/validate` always returns HTTP 200 for business outcomes. The SDK reflects this:

- `Validate` returns `(result, nil)` when the key is invalid — check `result.Valid` and `result.Reason`
- Rejection reasons: `not_found`, `expired`, `revoked`, `product_mismatch`

`/registry-credentials` returns HTTP 403 for invalid licenses. The SDK maps this to `*LicenseError`.

## Advanced patterns

### Cached validation

Reduce server round-trips with a TTL cache:

```go
validator := openlicensd.NewCachedValidator(client, 5*time.Minute)
```

### Background guard

For services that need continuous license enforcement with offline tolerance:

```go
guard, err := openlicensd.NewGuard(ctx, client, key,
    openlicensd.WithInterval(time.Hour),
    openlicensd.WithOfflineGrace(24*time.Hour),
)
defer guard.Stop()
```

### Key format validation

Validate key format locally before calling the server:

```go
if !openlicensd.ValidateKeyFormat(key) {
    return fmt.Errorf("invalid key format")
}
key = openlicensd.NormalizeKey(key)
```

## Retries

`Validate` retries on rate limiting (429), server errors (5xx), and network failures. Default: 2 attempts with exponential backoff and `Retry-After` support.

`RegistryCredentials` does **not** retry — the endpoint creates a Harbor robot account as a side effect.

## Compatibility

SDK releases are independent from server releases:

| SDK | Server |
|-----|--------|
| v0.1.x | >= 0.2.0 |

Tag format: `sdk/go/vX.Y.Z` (note the `v` prefix required by Go modules). Releases are published from Release Drafter drafts on GitHub (see [CONTRIBUTING.md](../../CONTRIBUTING.md)).

## See also

- [sdk/go/README.md](../../sdk/go/README.md) — package README with full API reference
- [api.md](../api.md) — HTTP API documentation
- [openapi.yaml](../openapi.yaml) — OpenAPI specification
