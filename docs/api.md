# API Reference

The OpenLicensd HTTP API is documented in [openapi.yaml](openapi.yaml) (OpenAPI 3.1). That file is the single source of truth for endpoints, request/response schemas, and status codes.

## Base URL

All API endpoints are served from the root of the server (default `http://localhost:8080`). Versioned routes live under `/api/v1`.

## Authentication

Admin endpoints require a session cookie (`openlicensd_session`) or a scoped API token (`Authorization: Bearer <token>`). Session-based unsafe methods (POST, PATCH, DELETE) also require the `X-CSRF-Token` header matching the `openlicensd_csrf` cookie. Bearer requests do not use CSRF.

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

# Single license by ID
curl -s -b cookies.txt http://localhost:8080/api/v1/licenses/{id}
```

List endpoints for licenses, products, policies, and users return a paginated envelope:

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
| Users | `created_at`, `updated_at`, `name`, `email`, `role`, `last_login_at` |
| API tokens | `created_at`, `updated_at`, `name`, `role`, `last_used_at`, `expires_at` |
| Audit events | `occurred_at`, `action`, `resource_type`, `actor_name` |

```bash
curl -s -b cookies.txt -X POST http://localhost:8080/api/v1/products \
  -H "Content-Type: application/json" \
  -H "X-CSRF-Token: $CSRF" \
  -d '{"name":"Acme Widget","code":"acme-widget"}'
```

Sessions expire after `OPENLICENSD_SESSION_TTL_HOURS` (default 24), with sliding renewal on activity.

### 3. Current user and logout

```bash
# Get the currently authenticated user (session cookie or Bearer token)
curl -s -b cookies.txt http://localhost:8080/api/v1/auth/me
```

Session authentication returns the user profile:

```json
{
  "id": "…",
  "email": "admin@example.com",
  "name": "Administrator",
  "role": "admin",
  "auth_provider": "local",
  "auth_method": "session",
  "has_password": true,
  "picture_url": null,
  "server_version": "dev"
}
```

Bearer API token authentication returns the token identity:

```json
{
  "auth_method": "api_token",
  "name": "terraform",
  "role": "operator",
  "token_id": "…",
  "server_version": "dev"
}
```

The login response `user` object uses the same fields as the session `/auth/me` response except it omits `auth_method`, `picture_url`, and `server_version`.

```bash
# Log out (revokes session and clears cookies; session only)
CSRF=$(grep openlicensd_csrf cookies.txt | awk '{print $7}')
curl -s -b cookies.txt -X POST http://localhost:8080/api/v1/auth/logout \
  -H "X-CSRF-Token: $CSRF"
```

`POST /api/v1/auth/logout` returns `204 No Content`. Bearer API token authentication returns `403 Forbidden`.

### 4. Change your password

Available for accounts with a local password (`has_password: true`). OIDC-only accounts cannot use this endpoint.

```bash
CSRF=$(grep openlicensd_csrf cookies.txt | awk '{print $7}')

curl -s -b cookies.txt -X POST http://localhost:8080/api/v1/auth/password \
  -H "Content-Type: application/json" \
  -H "X-CSRF-Token: $CSRF" \
  -d '{"current_password":"admin","password":"new-secure-password"}'
```

Returns `204 No Content` on success. Other sessions for the same user are revoked; the current session remains valid. Bearer API token authentication returns `403 Forbidden`.

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

### 6. API tokens for automation

Admins can create scoped API tokens from the **API Tokens** page in the admin UI or via `POST /api/v1/api-tokens` (session cookie required). Tokens carry one of the existing roles (`admin`, `operator`, `viewer`) and authenticate machine-to-machine requests without CSRF.

```bash
# Create a token (admin session)
CSRF=$(grep openlicensd_csrf cookies.txt | awk '{print $7}')
curl -s -b cookies.txt -X POST http://localhost:8080/api/v1/api-tokens \
  -H "Content-Type: application/json" \
  -H "X-CSRF-Token: $CSRF" \
  -d '{"name":"terraform","role":"operator"}'
```

The response includes a `token` field shown exactly once. Store it securely.

```bash
# Use the token (no CSRF header required)
curl -s http://localhost:8080/api/v1/licenses \
  -H "Authorization: Bearer olsd_..."

# Verify token identity
curl -s http://localhost:8080/api/v1/auth/me \
  -H "Authorization: Bearer olsd_..."
```

Token management endpoints (`/api/v1/api-tokens`) require an admin session — a Bearer token cannot create or revoke other tokens. Revoke with `PATCH /api/v1/api-tokens/{id}/revoke` or delete with `DELETE /api/v1/api-tokens/{id}`.

### 7. Audit log

Every successful admin mutation is recorded in an append-only `audit_events` table (actor, action, resource, IP, user agent, request ID). Admins can browse the log in the **Audit Log** UI page or export it via the API. Retention and pruning are configured with environment variables (`OPENLICENSD_AUDIT_RETENTION_DAYS`, `OPENLICENSD_AUDIT_CLEANUP_INTERVAL_MINUTES`); there is no admin API to delete audit rows.

```bash
# List recent audit events (admin session or Bearer token)
curl -s -b cookies.txt "http://localhost:8080/api/v1/audit-events?page_size=10&sort=occurred_at&order=desc"

# Filter by action and resource type
curl -s -b cookies.txt "http://localhost:8080/api/v1/audit-events?action=product.create&resource_type=product"
```

Audit events are never updated or deleted through the API. Only successful mutations (HTTP 2xx) are recorded.

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
| `GET` | `/api/v1/licenses/{id}` | `viewer`, `operator`, or `admin` |
| `GET` | `/api/v1/products` | `viewer`, `operator`, or `admin` |
| `GET` | `/api/v1/products/{id}` | `viewer`, `operator`, or `admin` |
| `GET` | `/api/v1/policies` | `viewer`, `operator`, or `admin` |
| `GET` | `/api/v1/policies/{id}` | `viewer`, `operator`, or `admin` |
| `POST` | `/api/v1/licenses` | `operator` or `admin` |
| `PATCH` | `/api/v1/licenses/{id}` | `operator` or `admin` |
| `DELETE` | `/api/v1/licenses/{id}` | `operator` or `admin` |
| `PATCH` | `/api/v1/licenses/{id}/revoke` | `operator` or `admin` |
| `PATCH` | `/api/v1/licenses/{id}/unrevoke` | `operator` or `admin` |
| `GET` | `/api/v1/licenses/{id}/machines` | `viewer`, `operator`, or `admin` |
| `PATCH` | `/api/v1/licenses/{id}/machines/{machine_id}` | `operator` or `admin` |
| `DELETE` | `/api/v1/licenses/{id}/machines/{machine_id}` | `operator` or `admin` |
| `POST` | `/api/v1/products` | `operator` or `admin` |
| `PATCH` | `/api/v1/products/{id}` | `operator` or `admin` |
| `DELETE` | `/api/v1/products/{id}` | `operator` or `admin` |
| `POST` | `/api/v1/policies` | `operator` or `admin` |
| `PATCH` | `/api/v1/policies/{id}` | `operator` or `admin` |
| `DELETE` | `/api/v1/policies/{id}` | `operator` or `admin` |
| `GET` | `/api/v1/users` | `admin` |
| `GET` | `/api/v1/users/{id}` | `admin` |
| `POST` | `/api/v1/users` | `admin` |
| `PATCH` | `/api/v1/users/{id}` | `admin` |
| `PATCH` | `/api/v1/users/{id}/password` | `admin` |
| `PATCH` | `/api/v1/users/{id}/disable` | `admin` |
| `PATCH` | `/api/v1/users/{id}/enable` | `admin` |
| `DELETE` | `/api/v1/users/{id}` | `admin` |
| `GET` | `/api/v1/api-tokens` | `admin` (session only) |
| `POST` | `/api/v1/api-tokens` | `admin` (session only) |
| `PATCH` | `/api/v1/api-tokens/{id}/revoke` | `admin` (session only) |
| `DELETE` | `/api/v1/api-tokens/{id}` | `admin` (session only) |
| `GET` | `/api/v1/audit-events` | `admin` |

## Public endpoints

These endpoints do not require authentication:

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/api/v1/validate` | Validate a license key |
| `POST` | `/api/v1/registry-credentials` | Issue Harbor credentials (when enabled) |
| `GET` | `/api/v1/auth/providers` | List enabled login methods |
| `GET` | `/api/v1/auth/oidc/login` | Start OIDC login (when enabled) |
| `GET` | `/api/v1/auth/oidc/callback` | OIDC callback (when enabled) |
| `GET` | `/healthz` | Liveness probe (no dependency checks) |
| `GET` | `/readyz` | Readiness probe (PostgreSQL ping) |

## Authenticated endpoints (session required)

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/api/v1/auth/logout` | Revoke session and clear cookies |
| `POST` | `/api/v1/auth/password` | Change own password (local accounts only) |

## Authenticated endpoints (session or Bearer token)

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/v1/auth/me` | Get current user profile (session) or token identity (Bearer) |

See the [endpoint access matrix](#endpoint-access-matrix) for role requirements on all other authenticated routes.

### PATCH semantics (licenses, products, policies)

`PATCH` endpoints accept partial JSON bodies. Only keys present in the request are updated; omitted keys keep their current database values. Set a nullable field to JSON `null` to clear it (for example, `expires_at: null` on a license). An empty object (`{}`) returns `400` with `no fields to update`. Non-nullable fields such as `label`, `name`, and `code` cannot be empty or `null` when included.

#### Policy edits and existing licenses

Not all policy fields behave the same when you `PATCH /api/v1/policies/{id}` after licenses have been issued:

- **Live on validation** — `grace_period_days` is read from the current policy on every validation. Changing it retroactively affects whether existing licenses are `expired` or `in_grace_period`.
- **Live while pending first validation** — `duration_days` and `expiration_basis` apply to licenses whose `expires_at` is still unset (`on_first_validation` not yet activated).
- **Not backfilled** — `max_activations` is copied to each license at create time. Policy edits do not change activation limits on existing licenses; use `PATCH /api/v1/licenses/{id}` instead.
- **Not recalculated** — `expires_at` on a license stays as written once set (at create, first validation, or license override).

See [architecture — Expiry semantics](architecture.md#expiry-semantics) for the full field-by-field table.

## User management (admin only)

```bash
CSRF=$(grep openlicensd_csrf cookies.txt | awk '{print $7}')

# List users (paginated)
curl -s -b cookies.txt "http://localhost:8080/api/v1/users?page=1&page_size=25&sort=created_at&order=desc" | jq

# Create a user (password must be at least 8 characters)
curl -s -b cookies.txt -X POST http://localhost:8080/api/v1/users \
  -H "Content-Type: application/json" \
  -H "X-CSRF-Token: $CSRF" \
  -d '{"email":"operator@example.com","name":"Operator","password":"secure-pass","role":"operator"}'

# Update a user (cannot demote the last enabled admin)
USER_ID="..."
curl -s -b cookies.txt -X PATCH "http://localhost:8080/api/v1/users/$USER_ID" \
  -H "Content-Type: application/json" \
  -H "X-CSRF-Token: $CSRF" \
  -d '{"email":"operator@example.com","name":"Updated Name","role":"viewer"}'

# Set or reset a user's password (admin action; minimum 8 characters; revokes all of their sessions)
curl -s -b cookies.txt -X PATCH "http://localhost:8080/api/v1/users/$USER_ID/password" \
  -H "Content-Type: application/json" \
  -H "X-CSRF-Token: $CSRF" \
  -d '{"password":"new-password"}'

# Disable a user (revokes all their sessions; cannot disable yourself or the last enabled admin)
curl -s -b cookies.txt -X PATCH "http://localhost:8080/api/v1/users/$USER_ID/disable" \
  -H "X-CSRF-Token: $CSRF"

# Re-enable a user
curl -s -b cookies.txt -X PATCH "http://localhost:8080/api/v1/users/$USER_ID/enable" \
  -H "X-CSRF-Token: $CSRF"

# Delete a user (cannot delete yourself or the last enabled admin)
curl -s -b cookies.txt -X DELETE "http://localhost:8080/api/v1/users/$USER_ID" \
  -H "X-CSRF-Token: $CSRF"
```

User objects include `id`, `email`, `name`, `role`, `auth_provider`, `created_at`, and `updated_at`. `disabled_at` and `last_login_at` are present only when applicable.

The last enabled admin cannot be demoted, disabled, or deleted. Those mutations return `400` so the server cannot be left with zero admins. The same guard applies to admin API tokens (`Authorization: Bearer`), which do not have a user identity and therefore bypass the self-disable/self-delete checks.

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

You can optionally override the policy-derived expiration with `expires_at` and the activation limit with `max_activations` (null = unlimited). Both values are **snapshotted** onto the license at create and are not updated when the policy changes later.

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

When a license is within the policy grace period after expiry:

```json
{ "valid": true, "in_grace_period": true, "expires_at": "2026-01-01T00:00:00Z" }
```

### Frozen validation error contract

Before v1.0, the dual envelope for license validation is frozen:

| Endpoint | Business outcome | HTTP status | Response shape |
|----------|------------------|-------------|----------------|
| `POST /api/v1/validate` | Valid or invalid license | **200** | `{ "valid": true\|false, "reason"?: "<code>", ... }` |
| `POST /api/v1/registry-credentials` | Invalid license | **403** | `{ "error": "<code>" }` |

Both endpoints share the same validation logic. `/validate` is a soft check clients poll via `valid` and `reason`. `/registry-credentials` treats an invalid license as a hard HTTP failure because issuing Harbor credentials is a privileged side effect.

The `reason` / `error` codes are pinned in OpenAPI as `ValidationReason`. See `docs/openapi.yaml` for per-reason examples on both endpoints.

| Code | Meaning |
|------|---------|
| `not_found` | License key hash not found |
| `expired` | License is past expiry and the policy grace period |
| `revoked` | License has been revoked |
| `product_mismatch` | Optional `product` code does not match the license |
| `fingerprint_required` | License has `max_activations` but the request omitted `fingerprint` |
| `activation_limit` | All activation seats are in use by other machine fingerprints |

On `/registry-credentials` only, the server may return `{ "error": "invalid" }` when no specific reason is available. This code is not used on `/validate`.

Transport and protocol errors (`400`, `429`, `500`, `504`, and `502` on registry-credentials) use the standard `{ "error": "..." }` envelope described below.

## Error responses

All errors use the same shape:

```json
{ "error": "human-readable message" }
```

Common status codes:

| Code | Meaning |
|------|---------|
| `400` | Invalid request body or parameters |
| `413` | Request body exceeds configured size limit (`OPENLICENSD_REQUEST_BODY_MAX_BYTES`) |
| `401` | Missing or invalid session |
| `403` | Forbidden (insufficient role, session-only endpoint called with a Bearer API token, or invalid license for registry-credentials) |
| `404` | Resource not found |
| `409` | Resource conflict (unique constraint or referential integrity violation) |
| `429` | Rate limit exceeded (public, login, and authenticated endpoints; includes `Retry-After` header) |
| `502` | Harbor API failure (registry-credentials only) |
| `503` | Database unavailable (readyz only) |
| `504` | Request deadline exceeded (`OPENLICENSD_REQUEST_TIMEOUT_SECONDS`) |

## Viewing the OpenAPI spec

### Redocly CLI

```bash
npx @redocly/cli lint docs/openapi.yaml --config docs/redocly.yaml
npx @redocly/cli preview-docs docs/openapi.yaml
```

### Contract tests

CI runs `TestOpenAPIContract` in `server/internal/api/openapi_contract_test.go` as part of the Server job (`make test`). The test drives each documented `operationId` through the live HTTP handlers (with PostgreSQL) and validates response status and JSON bodies against `docs/openapi.yaml` using `pb33f/libopenapi-validator`. A completeness check fails if the spec gains a new operation without a matching scenario.

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
