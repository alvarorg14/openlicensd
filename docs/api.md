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
    "auth_provider": "local",
    "has_password": true
  }
}
```

The response also sets `openlicensd_session` and `openlicensd_csrf` cookies.

### 2. Authenticated requests

```bash
CSRF=$(grep openlicensd_csrf cookies.txt | awk '{print $7}')

curl -s -b cookies.txt http://localhost:8080/api/v1/licenses

# Paginated list with search, filters, and sorting
curl -s -b cookies.txt "http://localhost:8080/api/v1/licenses?page=1&page_size=25&status=active&sort=created_at&order=desc"

# License status counts (unfiltered)
curl -s -b cookies.txt http://localhost:8080/api/v1/licenses/stats
```

List endpoints for licenses, products, and policies return a paginated envelope:

```json
{
  "items": [],
  "page": 1,
  "page_size": 25,
  "total": 0,
  "total_pages": 0
}
```

Common query parameters: `page` (default 1), `page_size` (default 25, max 100), `search`, `sort`, `order` (`asc` or `desc`). Licenses additionally support `status` (`active`, `expired`, `revoked`), `product_id`, and `policy_id`. Policies support `product_id`.

Allowed `sort` values vary by resource:

| Resource | Sort fields |
|----------|-------------|
| Licenses | `created_at`, `label`, `expires_at`, `product_name`, `policy_name`, `last_validated_at`, `validation_count`, `activation_count`, `max_activations` |
| Policies | `created_at`, `name`, `product_name`, `grace_period_days`, `max_activations` |
| Products | `created_at`, `updated_at`, `name`, `code` |

```bash
curl -s -b cookies.txt -X POST http://localhost:8080/api/v1/products \
  -H "Content-Type: application/json" \
  -H "X-CSRF-Token: $CSRF" \
  -d '{"name":"Acme Widget","code":"acme-widget"}'
```

Sessions expire after `OPENLICENSD_SESSION_TTL_HOURS` (default 24), with sliding renewal on activity.

### 3. Current user and logout

```bash
# Get the currently authenticated user (any role)
curl -s -b cookies.txt http://localhost:8080/api/v1/auth/me

# Log out (revokes session and clears cookies)
CSRF=$(grep openlicensd_csrf cookies.txt | awk '{print $7}')
curl -s -b cookies.txt -X POST http://localhost:8080/api/v1/auth/logout \
  -H "X-CSRF-Token: $CSRF"
```

`GET /api/v1/auth/me` returns the same `AuthUser` shape as the login response `user` object. `POST /api/v1/auth/logout` returns `204 No Content`.

### 4. Change your password

Available for accounts with a local password (`has_password: true`). OIDC-only accounts cannot use this endpoint.

```bash
CSRF=$(grep openlicensd_csrf cookies.txt | awk '{print $7}')

curl -s -b cookies.txt -X POST http://localhost:8080/api/v1/auth/password \
  -H "Content-Type: application/json" \
  -H "X-CSRF-Token: $CSRF" \
  -d '{"current_password":"admin","password":"new-secure-password"}'
```

Returns `204 No Content` on success. Other sessions for the same user are revoked; the current session remains valid.

### 5. OIDC SSO (optional)

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

## Roles and RBAC

| Role | Permissions |
|------|-------------|
| `admin` | Full access including user management |
| `operator` | Create, update, and delete licenses, products, and policies |
| `viewer` | Read-only access to licenses, products, and policies |

Insufficient role returns `403` with `{"error":"forbidden"}`.

### Endpoint access matrix

| Method | Path | Required role |
|--------|------|---------------|
| `GET` | `/api/v1/auth/me` | any authenticated user |
| `POST` | `/api/v1/auth/logout` | any authenticated user |
| `POST` | `/api/v1/auth/password` | any authenticated user |
| `GET` | `/api/v1/licenses/stats` | `viewer`, `operator`, or `admin` |
| `GET` | `/api/v1/licenses` | `viewer`, `operator`, or `admin` |
| `GET` | `/api/v1/products` | `viewer`, `operator`, or `admin` |
| `GET` | `/api/v1/policies` | `viewer`, `operator`, or `admin` |
| `POST` | `/api/v1/licenses` | `operator` or `admin` |
| `PATCH` | `/api/v1/licenses/{id}` | `operator` or `admin` |
| `DELETE` | `/api/v1/licenses/{id}` | `operator` or `admin` |
| `PATCH` | `/api/v1/licenses/{id}/revoke` | `operator` or `admin` |
| `PATCH` | `/api/v1/licenses/{id}/activate` | `operator` or `admin` |
| `GET` | `/api/v1/licenses/{id}/machines` | `viewer`, `operator`, or `admin` |
| `PATCH` | `/api/v1/licenses/{id}/machines/{machineId}` | `operator` or `admin` |
| `DELETE` | `/api/v1/licenses/{id}/machines/{machineId}` | `operator` or `admin` |
| `POST` | `/api/v1/products` | `operator` or `admin` |
| `PATCH` | `/api/v1/products/{id}` | `operator` or `admin` |
| `DELETE` | `/api/v1/products/{id}` | `operator` or `admin` |
| `POST` | `/api/v1/policies` | `operator` or `admin` |
| `PATCH` | `/api/v1/policies/{id}` | `operator` or `admin` |
| `DELETE` | `/api/v1/policies/{id}` | `operator` or `admin` |
| `GET` | `/api/v1/users` | `admin` |
| `POST` | `/api/v1/users` | `admin` |
| `PATCH` | `/api/v1/users/{id}` | `admin` |
| `PATCH` | `/api/v1/users/{id}/password` | `admin` |
| `PATCH` | `/api/v1/users/{id}/disable` | `admin` |
| `PATCH` | `/api/v1/users/{id}/enable` | `admin` |
| `DELETE` | `/api/v1/users/{id}` | `admin` |

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

## Authenticated endpoints (session required)

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/v1/auth/me` | Get current user profile |
| `POST` | `/api/v1/auth/logout` | Revoke session and clear cookies |
| `POST` | `/api/v1/auth/password` | Change own password (local accounts only) |

See the [endpoint access matrix](#endpoint-access-matrix) for role requirements on all other authenticated routes.

## User management (admin only)

`GET /api/v1/users` returns an **unpaginated array** of users (not the paginated envelope used by licenses, products, and policies).

```bash
CSRF=$(grep openlicensd_csrf cookies.txt | awk '{print $7}')

# List all users
curl -s -b cookies.txt http://localhost:8080/api/v1/users | jq

# Create a user (password must be at least 8 characters)
curl -s -b cookies.txt -X POST http://localhost:8080/api/v1/users \
  -H "Content-Type: application/json" \
  -H "X-CSRF-Token: $CSRF" \
  -d '{"email":"operator@example.com","name":"Operator","password":"secure-pass","role":"operator"}'

# Update a user
USER_ID="..."
curl -s -b cookies.txt -X PATCH "http://localhost:8080/api/v1/users/$USER_ID" \
  -H "Content-Type: application/json" \
  -H "X-CSRF-Token: $CSRF" \
  -d '{"email":"operator@example.com","name":"Updated Name","role":"viewer"}'

# Set or reset a user's password (admin action; minimum 8 characters)
curl -s -b cookies.txt -X PATCH "http://localhost:8080/api/v1/users/$USER_ID/password" \
  -H "Content-Type: application/json" \
  -H "X-CSRF-Token: $CSRF" \
  -d '{"password":"new-password"}'

# Disable a user (revokes all their sessions; cannot disable yourself)
curl -s -b cookies.txt -X PATCH "http://localhost:8080/api/v1/users/$USER_ID/disable" \
  -H "X-CSRF-Token: $CSRF"

# Re-enable a user
curl -s -b cookies.txt -X PATCH "http://localhost:8080/api/v1/users/$USER_ID/enable" \
  -H "X-CSRF-Token: $CSRF"

# Delete a user (cannot delete yourself)
curl -s -b cookies.txt -X DELETE "http://localhost:8080/api/v1/users/$USER_ID" \
  -H "X-CSRF-Token: $CSRF"
```

User objects include `id`, `email`, `name`, `role`, `auth_provider`, `created_at`, and `updated_at`. `disabled_at` and `last_login_at` are present only when applicable.

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

You can optionally override the policy-derived expiration with `expires_at` and the activation limit with `max_activations` (null = unlimited).

## Example: validate a license

```bash
curl -s -X POST http://localhost:8080/api/v1/validate \
  -H "Content-Type: application/json" \
  -d '{"key":"X4F9K-7QP2M-3RH8N-BW6TG-YZ2CD","product":"acme-widget","fingerprint":"550e8400-e29b-41d4-a716-446655440000","hostname":"dev-macbook.local"}' | jq
```

When the license has `max_activations`, `fingerprint` is required. Known fingerprints reuse a seat; new fingerprints beyond the limit return:

```json
{ "valid": false, "reason": "activation_limit", "activation_count": 2, "max_activations": 2 }
```

Valid response:

```json
{ "valid": true, "product": "acme-widget", "policy": "30-day trial", "activation_count": 1, "max_activations": 2 }
```

Invalid response (HTTP 200 when the request is allowed):

```json
{ "valid": false, "reason": "expired", "expires_at": "2026-01-01T00:00:00Z" }
```

Possible `reason` values: `not_found`, `expired`, `revoked`, `product_mismatch`.

When a license is within the policy grace period after expiry:

```json
{ "valid": true, "in_grace_period": true, "expires_at": "2026-01-01T00:00:00Z" }
```

> **Note:** `/validate` returns HTTP 200 with a `valid` boolean when the request is allowed. Rate-limited requests return HTTP 429. The `/registry-credentials` endpoint returns HTTP 403 with an `error` field for invalid licenses instead.

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
| `409` | Resource conflict (unique constraint or referential integrity violation) |
| `429` | Rate limit exceeded (unauthenticated endpoints; includes `Retry-After` header) |
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
