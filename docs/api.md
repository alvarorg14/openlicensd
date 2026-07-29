# API Reference

The OpenLicensd HTTP API is documented in [openapi.yaml](openapi.yaml) (OpenAPI 3.1). That file is the single source of truth for endpoints, request/response schemas, and status codes.

## Base URL

All API endpoints are served from the root of the server (default `http://localhost:8080`). Versioned routes live under `/api/v1`.

## Authentication

Admin endpoints require a JWT bearer token.

### 1. Login

```bash
curl -s -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"admin"}'
```

Response:

```json
{ "token": "eyJhbGciOiJIUzI1NiIs..." }
```

### 2. Use the token

Pass the token in the `Authorization` header for all admin requests:

```bash
TOKEN="eyJhbGciOiJIUzI1NiIs..."

curl -s http://localhost:8080/api/v1/licenses \
  -H "Authorization: Bearer $TOKEN"
```

Tokens are signed with HS256 using `OPENLICENSD_JWT_SECRET` and expire after 24 hours.

## Public endpoints

These endpoints do not require authentication:

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/api/v1/validate` | Validate a license key |
| `POST` | `/api/v1/registry-credentials` | Issue Harbor credentials (when enabled) |
| `GET` | `/healthz` | Liveness probe |
| `GET` | `/readyz` | Readiness probe (checks database) |

## Example: create a license

```bash
curl -s -X POST http://localhost:8080/api/v1/licenses \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"label":"Acme Corp","expires_at":"2027-01-01T00:00:00Z"}'
```

The response includes the raw `key` field **once**. Store it securely — it cannot be retrieved later.

## Example: validate a license

```bash
curl -s -X POST http://localhost:8080/api/v1/validate \
  -H "Content-Type: application/json" \
  -d '{"key":"X4F9K-7QP2M-3RH8N-BW6TG-YZ2CD"}'
```

Valid response:

```json
{ "valid": true, "expires_at": null }
```

Invalid response (always HTTP 200):

```json
{ "valid": false, "reason": "expired", "expires_at": "2026-01-01T00:00:00Z" }
```

Possible `reason` values: `not_found`, `expired`, `revoked`.

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
| `401` | Missing or invalid JWT |
| `403` | Invalid license (registry-credentials only) |
| `404` | License not found |
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
