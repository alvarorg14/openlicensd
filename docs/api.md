# API Reference

The OpenLicensd HTTP API is documented in [openapi.yaml](openapi.yaml) (OpenAPI 3.1). That file is the single source of truth for endpoints, request/response schemas, and status codes.

## Base URL

All API endpoints are served from the root of the server (default `http://localhost:8080`). Versioned routes live under `/api/v1`.

## Authentication

Admin endpoints require a session cookie (`openlicensd_session`). Unsafe methods (POST, PATCH, DELETE) also require the `X-CSRF-Token` header matching the `openlicensd_csrf` cookie.

### 1. Login

```bash
curl -s -c cookies.txt -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"admin@example.com","password":"admin"}'
```

Response:

```json
{
  "user": {
    "id": "...",
    "email": "admin@example.com",
    "name": "Administrator",
    "role": "admin",
    "auth_provider": "local"
  }
}
```

The response also sets `openlicensd_session` and `openlicensd_csrf` cookies.

### 2. Authenticated requests

```bash
CSRF=$(grep openlicensd_csrf cookies.txt | awk '{print $7}')

curl -s -b cookies.txt http://localhost:8080/api/v1/licenses

curl -s -b cookies.txt -X POST http://localhost:8080/api/v1/products \
  -H "Content-Type: application/json" \
  -H "X-CSRF-Token: $CSRF" \
  -d '{"name":"Acme Widget","code":"acme-widget"}'
```

Sessions expire after `OPENLICENSD_SESSION_TTL_HOURS` (default 24), with sliding renewal on activity.

### 3. OIDC SSO (optional)

When `OPENLICENSD_OIDC_ENABLED=true`, users can sign in via your identity provider:

```bash
# Discover enabled login methods
curl -s http://localhost:8080/api/v1/auth/providers
```

Response when OIDC is enabled:

```json
{
  "local": true,
  "oidc": true,
  "oidc_name": "SSO",
  "oidc_login_url": "/api/v1/auth/oidc/login"
}
```

Redirect the browser to `oidc_login_url` (or open `/api/v1/auth/oidc/login` directly). After successful authentication, the callback sets the same session cookies as local login.

See [oidc-sso.md](oidc-sso.md) for full setup instructions.

## Roles

| Role | Permissions |
|------|-------------|
| `admin` | Full access including user management |
| `operator` | Create/update/delete licenses, products, policies |
| `viewer` | Read-only access |

## Public endpoints

These endpoints do not require authentication:

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/api/v1/validate` | Validate a license key |
| `POST` | `/api/v1/registry-credentials` | Issue Harbor credentials (when enabled) |
| `GET` | `/api/v1/auth/providers` | List enabled login methods |
| `GET` | `/api/v1/auth/oidc/login` | Start OIDC login (when enabled) |
| `GET` | `/api/v1/auth/oidc/callback` | OIDC callback (when enabled) |
| `GET` | `/healthz` | Liveness probe |
| `GET` | `/readyz` | Readiness probe (checks database) |

## Example: create a product and policy

```bash
CSRF=$(grep openlicensd_csrf cookies.txt | awk '{print $7}')

PRODUCT=$(curl -s -b cookies.txt -X POST http://localhost:8080/api/v1/products \
  -H "Content-Type: application/json" \
  -H "X-CSRF-Token: $CSRF" \
  -d '{"name":"Acme Widget","code":"acme-widget"}')

PRODUCT_ID=$(echo "$PRODUCT" | jq -r .id)

POLICY=$(curl -s -b cookies.txt -X POST http://localhost:8080/api/v1/policies \
  -H "Content-Type: application/json" \
  -H "X-CSRF-Token: $CSRF" \
  -d "{\"product_id\":\"$PRODUCT_ID\",\"name\":\"30-day trial\",\"duration_days\":30,\"expiration_basis\":\"on_first_validation\"}")

POLICY_ID=$(echo "$POLICY" | jq -r .id)
```

## Example: create a license

```bash
curl -s -b cookies.txt -X POST http://localhost:8080/api/v1/licenses \
  -H "Content-Type: application/json" \
  -H "X-CSRF-Token: $CSRF" \
  -d "{\"label\":\"Acme Corp\",\"product_id\":\"$PRODUCT_ID\",\"policy_id\":\"$POLICY_ID\"}" | jq
```

The response includes the raw `key` field **once**. Store it securely — it cannot be retrieved later.

You can optionally override the policy-derived expiration with `expires_at`.

## Example: validate a license

```bash
curl -s -X POST http://localhost:8080/api/v1/validate \
  -H "Content-Type: application/json" \
  -d '{"key":"X4F9K-7QP2M-3RH8N-BW6TG-YZ2CD","product":"acme-widget"}' | jq
```

Valid response:

```json
{ "valid": true, "product": "acme-widget", "policy": "30-day trial" }
```

Invalid response (always HTTP 200):

```json
{ "valid": false, "reason": "expired", "expires_at": "2026-01-01T00:00:00Z" }
```

Possible `reason` values: `not_found`, `expired`, `revoked`, `product_mismatch`.

When a license is within the policy grace period after expiry:

```json
{ "valid": true, "in_grace_period": true, "expires_at": "2026-01-01T00:00:00Z" }
```

> **Note:** `/validate` always returns HTTP 200 with a `valid` boolean. The `/registry-credentials` endpoint returns HTTP 403 with an `error` field for invalid licenses instead.

## Error responses

All errors use the same shape:

```json
{ "error": "human-readable message" }
```

Common status codes:

| Code | Meaning |
|------|---------|
| `400` | Invalid request body or parameters |
| `401` | Missing or invalid session |
| `403` | Forbidden (insufficient role or invalid license for registry-credentials) |
| `404` | Resource not found |
| `409` | Resource is referenced by other records |
| `502` | Harbor API failure (registry-credentials only) |
| `503` | Database unavailable (readyz only) |

## Viewing the OpenAPI spec

### Redocly CLI

```bash
npx @redocly/cli lint docs/openapi.yaml --config docs/redocly.yaml
npx @redocly/cli preview-docs docs/openapi.yaml
```

### Swagger UI (Docker)

```bash
docker run -p 8081:8080 \
  -e SWAGGER_JSON=/spec/openapi.yaml \
  -v "$(pwd)/docs/openapi.yaml:/spec/openapi.yaml" \
  swaggerapi/swagger-ui
```

Open http://localhost:8081.

## Related

- [architecture.md](architecture.md) — how requests flow through the server
- [harbor-registry-credentials.md](harbor-registry-credentials.md) — registry credentials endpoint
- [configuration.md](configuration.md) — environment variables
