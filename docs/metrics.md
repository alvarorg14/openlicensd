# Prometheus metrics

OpenLicensd exposes a Prometheus scrape endpoint on a **dedicated HTTP listener** separate from the API/UI server. This keeps `/metrics` off the Ingress and away from public traffic.

## Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `OPENLICENSD_METRICS_ENABLED` | `true` | Enable the metrics listener |
| `OPENLICENSD_METRICS_ADDR` | `:9090` | Listen address for `/metrics` (must differ from `OPENLICENSD_ADDR`) |

When enabled, the process listens on `OPENLICENSD_METRICS_ADDR` and serves `GET /metrics` in Prometheus text format.

## Scrape configuration

Example Prometheus scrape config (adjust host/port for your deployment):

```yaml
scrape_configs:
  - job_name: openlicensd
    static_configs:
      - targets: ["openlicensd:9090"]
    metrics_path: /metrics
```

In Kubernetes, scrape the `metrics` container port (see the Helm chart). A `ServiceMonitor` template is tracked in [issue #96](https://github.com/alvarorg14/openlicensd/issues/96).

## Exported metrics

All application metrics use the `openlicensd_` prefix.

### HTTP

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `openlicensd_http_requests_total` | Counter | `method`, `route`, `status` | API requests by chi route pattern (not raw URL path) |
| `openlicensd_http_request_duration_seconds` | Histogram | `method`, `route` | Request latency; `route="/api/v1/validate"` covers validation latency |

Unmatched routes (static SPA assets, 404s) are labeled `route="other"`.

### License validation

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `openlicensd_license_validations_total` | Counter | `result`, `reason` | Validation outcomes from `/validate` and `/registry-credentials` |

`result` is `valid` or `invalid`. `reason` is one of: `ok`, `not_found`, `expired`, `revoked`, `product_mismatch`, `fingerprint_required`, `activation_limit`, or `unknown`.

### Database pool

| Metric | Type | Description |
|--------|------|-------------|
| `openlicensd_db_pool_acquired_connections` | Gauge | Currently acquired connections |
| `openlicensd_db_pool_idle_connections` | Gauge | Idle connections |
| `openlicensd_db_pool_total_connections` | Gauge | Total connections in the pool |
| `openlicensd_db_pool_max_connections` | Gauge | Configured pool maximum |
| `openlicensd_db_pool_acquire_count_total` | Counter | Successful acquires |
| `openlicensd_db_pool_empty_acquire_count_total` | Counter | Acquires that waited on an empty pool |
| `openlicensd_db_pool_canceled_acquire_count_total` | Counter | Acquires canceled by context |
| `openlicensd_db_pool_new_connections_count_total` | Counter | New connections opened |

Pool tuning is separate from these gauges — see [issue #87](https://github.com/alvarorg14/openlicensd/issues/87).

### Build and runtime

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `openlicensd_build_info` | Gauge | `version` | Always `1`; version label carries the server build |

Standard Go runtime and process metrics from `client_golang` collectors are also included (`go_*`, `process_*`).

## Disabling metrics

Set `OPENLICENSD_METRICS_ENABLED=false` to disable the listener and all application metric registration.
