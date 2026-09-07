# OpenLicensd Go SDK

Official Go client for the OpenLicensd public validation API.

- **Module:** `github.com/alvarorg14/openlicensd/sdk/go`
- **Import:** `openlicensd "github.com/alvarorg14/openlicensd/sdk/go"`
- **Dependencies:** stdlib only
- **Go version:** 1.26+

## Install

```bash
go get github.com/alvarorg14/openlicensd/sdk/go@v0.1.0
```

## Quick start

```go
package main

import (
    "context"
    "fmt"
    "log"

    openlicensd "github.com/alvarorg14/openlicensd/sdk/go"
)

func main() {
    client, err := openlicensd.New("https://licenses.example.com", "acme-widget")
    if err != nil {
        log.Fatal(err)
    }

    result, err := client.Validate(context.Background(), "X4F9K-7QP2M-3RH8N-BW6TG-YZ2CD")
    if err != nil {
        log.Fatal(err) // transport or server error
    }
    if !result.Valid {
        log.Fatalf("license rejected: %s", result.Reason)
    }

    fmt.Println("licensed until", result.ExpiresAt)
}
```

## Configuration

### Constructor

`New(baseURL, product string, opts ...Option)` requires both arguments:

- **baseURL** — absolute `http` or `https` URL of your OpenLicensd server. Trailing slashes are stripped. Path prefixes are supported (`https://example.com/licenses`).
- **product** — product code sent on every validation request. Required to prevent accidentally accepting a valid key for any product on the server.

Pass `WithAnyProduct()` only when you intentionally want unscoped validation.

### Environment variables (opt-in)

`NewFromEnv()` reads `OPENLICENSD_URL` and `OPENLICENSD_PRODUCT`. Use this for self-hosted deployments where the operator controls the server. **Do not use in vendor-shipped binaries** — an operator could point validation at a server they control.

### Options

| Option | Description |
|--------|-------------|
| `WithTimeout(d)` | HTTP client timeout (default 10s) |
| `WithUserAgent(ua)` | Custom User-Agent header |
| `WithHTTPClient(c)` | Custom `*http.Client` |
| `WithRetry(n, delay)` | Retry `Validate` on 429/5xx/network errors (default 2 attempts) |
| `WithAnyProduct()` | Disable product scoping |
| `WithFingerprint(fp)` | Machine fingerprint sent on validation requests (required when the server enforces `max_activations`) |
| `WithHostname(hostname)` | Send this hostname instead of `os.Hostname()` |
| `WithoutHostname()` | Do not send a hostname on validation requests |

### Machine fingerprint

When the server enforces `max_activations`, configure the client with a stable machine fingerprint. Without one, validation returns `fingerprint_required`; new fingerprints beyond the limit return `activation_limit`.

```go
fp, err := openlicensd.Fingerprint("my-cli")
if err != nil {
    log.Fatal(err)
}

client, err := openlicensd.New(
    "https://licenses.example.com",
    "acme-widget",
    openlicensd.WithFingerprint(fp),
)
```

`Fingerprint(appName)` persists a UUID under the OS user config directory at `<UserConfigDir>/<appName>/machine-id`. Use `FingerprintAt(path)` when you need a custom location (for example a mounted volume in CI). The SDK sends `os.Hostname()` by default; pass `WithHostname(name)` to override it or `WithoutHostname()` to omit it.

## API methods

| Method | Endpoint | Notes |
|--------|----------|-------|
| `Validate(ctx, key)` | `POST /api/v1/validate` | Invalid license returns `Valid: false`, not an error |
| `ValidateProduct(ctx, key, product)` | `POST /api/v1/validate` | Per-call product override |
| `RegistryCredentials(ctx, key)` | `POST /api/v1/registry-credentials` | Invalid license returns `*LicenseError` |
| `Health(ctx)` | `GET /healthz` | Liveness probe |
| `Ready(ctx)` | `GET /readyz` | Readiness probe |

## Error handling

```go
result, err := client.Validate(ctx, key)
if err != nil {
    var apiErr *openlicensd.APIError
    if errors.As(err, &apiErr) {
        if errors.Is(err, openlicensd.ErrRateLimited) {
            time.Sleep(apiErr.RetryAfter)
        }
    }
    return err
}
if !result.Valid {
    switch result.Reason {
    case openlicensd.ReasonExpired:
        // ...
    }
}
```

`RegistryCredentials` returns `*LicenseError` with a typed `Reason` on 403.

## Key helpers

```go
key := openlicensd.NormalizeKey("x4f9k7qp2m3rh8nbw6tgyz2cd")
if !openlicensd.ValidateKeyFormat(key) {
    return openlicensd.ErrInvalidKey
}
```

## Cached validation

Reduce server round-trips with a TTL cache over `Validate`:

```go
validator := openlicensd.NewCachedValidator(client, 5*time.Minute)
result, err := validator.Validate(ctx, key)
```

| API | Description |
|-----|-------------|
| `NewCachedValidator(client, ttl)` | Wrap a client with a TTL cache (defaults to 5m when `ttl <= 0`) |
| `Validate(ctx, key)` | Return a cached result when available and not expired |
| `Invalidate(key)` | Remove one key from the cache |
| `Clear()` | Remove all cached entries |

## Background guard

For long-running services, `Guard` revalidates on an interval and tolerates transient outages:

```go
guard, err := openlicensd.NewGuard(ctx, client, key,
    openlicensd.WithInterval(time.Hour),
    openlicensd.WithOfflineGrace(24*time.Hour),
)
if err != nil {
    log.Fatal(err)
}
defer guard.Stop()

if !guard.Valid() {
    log.Fatal("license invalid")
}
```

| Option / method | Description |
|-----------------|-------------|
| `WithInterval(d)` | How often to revalidate (default 1h) |
| `WithOfflineGrace(d)` | How long the guard stays valid after the last successful validation when the server is unreachable (default 24h) |
| `Valid()` | Whether the license is currently considered valid |
| `Last()` | Most recent validation result |
| `LastError()` | Most recent validation error, if any |
| `Stop()` | End background revalidation and wait for the goroutine to exit |

## Compatibility

| SDK version | Server version |
|-------------|----------------|
| v0.1.x | >= 0.2.0 |

The SDK targets the public API contract (`/validate`, `/registry-credentials`, health probes). Server releases and SDK releases are versioned independently.

## Development

```bash
make test-sdk
make lint-sdk
```

## License

Apache License 2.0 — see [LICENSE](../../LICENSE).
