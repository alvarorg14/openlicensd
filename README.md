# openlicensd

Open source license server for creating and validating license keys.

## Features

- Admin UI to create, list, and revoke license keys
- Optional expiration dates (or never expires)
- Public validation endpoint for license key checks
- Single binary distribution with embedded UI
- PostgreSQL storage

## Quick start

### Prerequisites

- Go 1.23+
- Node.js 24+
- Docker (for local PostgreSQL)

### Development

1. Start PostgreSQL:

```bash
make dev-db
```

2. Copy environment variables:

```bash
cp .env.example .env
```

3. In one terminal, start the API server:

```bash
make dev-server
```

4. In another terminal, start the UI dev server:

```bash
make dev-ui
```

Open http://localhost:3000 and sign in with:

- Username: `admin`
- Password: `admin`

### Production build

Build the UI and compile the server into a single binary:

```bash
make build
```

The binary is written to `bin/openlicensd`.

Run it with the required environment variables:

```bash
export OPENLICENSD_DATABASE_URL=postgres://user:pass@host:5432/openlicensd?sslmode=disable
export OPENLICENSD_ADMIN_USER=admin
export OPENLICENSD_ADMIN_PASSWORD_HASH=$(make hash-password PASSWORD=your-secure-password)
export OPENLICENSD_JWT_SECRET=your-jwt-secret

./bin/openlicensd
```

The server serves the API and embedded UI on `OPENLICENSD_ADDR` (default `:8080`).

## API

All endpoints are under `/api/v1`.

### `POST /auth/login`

```json
{ "username": "admin", "password": "admin" }
```

Returns `{ "token": "..." }`.

### `POST /licenses` (admin)

```json
{ "label": "Acme Corp", "expires_at": "2027-01-01T00:00:00Z" }
```

Omit `expires_at` or set it to `null` for a key that never expires. Returns the raw key once.

### `GET /licenses` (admin)

Lists licenses (without raw keys).

### `PATCH /licenses/{id}/revoke` (admin)

Revokes a license.

### `POST /validate` (public)

```json
{ "key": "ol_..." }
```

Returns:

```json
{ "valid": true, "expires_at": null }
```

Or when invalid:

```json
{ "valid": false, "reason": "expired" }
```

Possible reasons: `not_found`, `expired`, `revoked`.

## Configuration

| Variable | Description |
|---|---|
| `OPENLICENSD_ADDR` | HTTP listen address (default `:8080`) |
| `OPENLICENSD_DATABASE_URL` | PostgreSQL connection URL |
| `OPENLICENSD_ADMIN_USER` | Admin username |
| `OPENLICENSD_ADMIN_PASSWORD_HASH` | Bcrypt hash of admin password |
| `OPENLICENSD_JWT_SECRET` | Secret for signing JWT tokens |

Generate a password hash:

```bash
make hash-password PASSWORD=yourpassword
```

## Project layout

```
server/   Go API and embedded static handler
ui/       Nuxt 3 (Vue) admin UI
```

## Releases

Tag a version to trigger a GitHub release with cross-platform binaries:

```bash
git tag v0.1.0
git push origin v0.1.0
```

## License

Apache License 2.0. See [LICENSE](LICENSE).
