# openlicensd

Open source license server for creating and validating license keys

![Version: 0.0.0-dev](https://img.shields.io/badge/Version-0.0.0--dev-informational?style=flat-square) ![Type: application](https://img.shields.io/badge/Type-application-informational?style=flat-square) ![AppVersion: 0.0.0-dev](https://img.shields.io/badge/AppVersion-0.0.0--dev-informational?style=flat-square)

## Installing

Requires an external PostgreSQL database. The chart does not bundle a database.

```bash
helm install openlicensd oci://ghcr.io/alvarorg14/charts/openlicensd \
  --version X.Y.Z \
  --namespace openlicensd \
  --create-namespace \
  --set config.bootstrapAdmin.email=admin@example.com \
  --set secret.data.databaseUrl='postgres://user:pass@host:5432/openlicensd?sslmode=disable' \
  --set secret.data.bootstrapAdminPasswordHash='$2a$10$...'
```

Use an existing Secret:

```bash
helm install openlicensd oci://ghcr.io/alvarorg14/charts/openlicensd \
  --namespace openlicensd \
  --create-namespace \
  --set secret.mode=existing \
  --set secret.existingSecret=openlicensd-credentials
```

Use External Secrets Operator:

```bash
helm install openlicensd oci://ghcr.io/alvarorg14/charts/openlicensd \
  --namespace openlicensd \
  --create-namespace \
  --set secret.mode=externalSecrets \
  --set secret.externalSecrets.secretStoreRef.name=my-secret-store \
  --set-json 'secret.externalSecrets.remoteRefs=[{"secretKey":"OPENLICENSD_DATABASE_URL","remoteRef":{"key":"openlicensd/database","property":"url"}},{"secretKey":"OPENLICENSD_BOOTSTRAP_ADMIN_PASSWORD_HASH","remoteRef":{"key":"openlicensd/admin","property":"passwordHash"}}]'
```

For local development, install from the chart source:

```bash
helm install openlicensd ./charts/openlicensd \
  --namespace openlicensd \
  --create-namespace \
  -f charts/openlicensd/ci/test-values.yaml
```

## Health probes

The Deployment configures Kubernetes probes against the HTTP port:

| Probe | Path | Purpose |
|-------|------|---------|
| Liveness | `/healthz` | Process is alive (no dependency checks) |
| Readiness | `/readyz` | PostgreSQL is reachable |
| Startup | `/readyz` | Allows migrations to finish before liveness begins |

Liveness must not ping the database: a transient Postgres outage would restart pods instead of removing them from the load balancer. Readiness and startup use `/readyz`, which runs a lightweight `Store.Ping` with a 2-second timeout.

## Values
| Key | Type | Default | Description |
|-----|------|---------|-------------|
| affinity | object | `{}` | Affinity rules for pod assignment |
| autoscaling.annotations | object | `{}` | Annotations to add to the HorizontalPodAutoscaler |
| autoscaling.behavior | object | `{}` | HPA scaling behavior (scaleUp/scaleDown policies) |
| autoscaling.enabled | bool | `false` | Create a HorizontalPodAutoscaler for OpenLicensd pods |
| autoscaling.maxReplicas | int | `5` | Maximum replicas when autoscaling is enabled |
| autoscaling.minReplicas | int | `1` | Minimum replicas when autoscaling is enabled |
| autoscaling.targetCPUUtilizationPercentage | int | `80` | Target average CPU utilization percentage. Leave empty to omit the CPU metric. |
| autoscaling.targetMemoryUtilizationPercentage | string | `""` | Target average memory utilization percentage. Leave empty to omit the memory metric. |
| config.addr | string | `":8080"` | HTTP listen address (maps to OPENLICENSD_ADDR) |
| config.bootstrapAdmin.email | string | `""` | Email for the first admin user seeded on empty database (maps to OPENLICENSD_BOOTSTRAP_ADMIN_EMAIL) |
| config.bootstrapAdmin.name | string | `"Administrator"` | Display name for the bootstrap admin (maps to OPENLICENSD_BOOTSTRAP_ADMIN_NAME) |
| config.cookieSecure | bool | `true` | Set Secure flag on session cookies (maps to OPENLICENSD_COOKIE_SECURE) |
| config.database.maxConnIdleMinutes | int | `0` | Close idle connections after this many minutes; 0 uses pgx default (maps to OPENLICENSD_DATABASE_MAX_CONN_IDLE_MINUTES) |
| config.database.maxConns | int | `0` | Maximum pool connections; 0 uses pgx default (maps to OPENLICENSD_DATABASE_MAX_CONNS) |
| config.database.minConns | int | `0` | Minimum pool connections; 0 uses pgx default (maps to OPENLICENSD_DATABASE_MIN_CONNS) |
| config.database.statementTimeoutSeconds | int | `0` | PostgreSQL statement_timeout in seconds; 0 leaves server default (maps to OPENLICENSD_DATABASE_STATEMENT_TIMEOUT_SECONDS) |
| config.harbor.debug | bool | `false` | Log Harbor API requests/responses (maps to OPENLICENSD_HARBOR_DEBUG) |
| config.harbor.enabled | bool | `false` | Enable Harbor registry credentials endpoint |
| config.harbor.insecureSkipVerify | bool | `false` | Skip TLS verification for Harbor (maps to OPENLICENSD_HARBOR_INSECURE_SKIP_VERIFY) |
| config.harbor.projects | string | `""` | Comma-separated Harbor project namespaces (maps to OPENLICENSD_HARBOR_PROJECTS) |
| config.harbor.robotDurationDays | int | `1` | Robot account lifetime in days (maps to OPENLICENSD_HARBOR_ROBOT_DURATION_DAYS) |
| config.harbor.robotNamePrefix | string | `"openlicensd"` | Prefix for generated robot account names (maps to OPENLICENSD_HARBOR_ROBOT_NAME_PREFIX) |
| config.harbor.url | string | `""` | Harbor base URL (maps to OPENLICENSD_HARBOR_URL) |
| config.localLoginEnabled | bool | `true` |  |
| config.log.format | string | `"json"` | Log output format: json or text (maps to OPENLICENSD_LOG_FORMAT) |
| config.log.level | string | `"info"` | Log level: debug, info, warn, or error (maps to OPENLICENSD_LOG_LEVEL) |
| config.metrics.addr | string | `":9090"` | Metrics listen address (maps to OPENLICENSD_METRICS_ADDR) |
| config.metrics.enabled | bool | `true` | Enable Prometheus /metrics on a dedicated listener (maps to OPENLICENSD_METRICS_ENABLED) |
| config.oidc.adminEmails | string | `""` | Comma-separated admin emails on first SSO login (maps to OPENLICENSD_OIDC_ADMIN_EMAILS) |
| config.oidc.clientId | string | `""` | OAuth client ID (maps to OPENLICENSD_OIDC_CLIENT_ID) |
| config.oidc.defaultRole | string | `"viewer"` | Default role for new SSO users (maps to OPENLICENSD_OIDC_DEFAULT_ROLE) |
| config.oidc.enabled | bool | `false` | Enable OIDC SSO (maps to OPENLICENSD_OIDC_ENABLED) |
| config.oidc.issuerUrl | string | `""` | OIDC issuer URL (maps to OPENLICENSD_OIDC_ISSUER_URL) |
| config.oidc.providerName | string | `"SSO"` | SSO button label (maps to OPENLICENSD_OIDC_PROVIDER_NAME) |
| config.oidc.redirectUrl | string | `""` | Callback URL registered with the IdP (maps to OPENLICENSD_OIDC_REDIRECT_URL) |
| config.oidc.scopes | string | `"openid,profile,email"` | Comma-separated scopes (maps to OPENLICENSD_OIDC_SCOPES) |
| config.rateLimit.backend | string | `"memory"` | Rate limit backend: memory (per-replica) or postgres (shared across replicas) (maps to OPENLICENSD_RATE_LIMIT_BACKEND) |
| config.rateLimit.enabled | bool | `true` | Enable per-IP rate limiting on unauthenticated endpoints (maps to OPENLICENSD_RATE_LIMIT_ENABLED) |
| config.rateLimit.idleMinutes | int | `10` | Minutes before unused per-IP buckets are evicted (maps to OPENLICENSD_RATE_LIMIT_IDLE_MINUTES) |
| config.rateLimit.loginBurst | int | `10` | Burst capacity for login endpoints (maps to OPENLICENSD_RATE_LIMIT_LOGIN_BURST) |
| config.rateLimit.loginPerMinute | int | `30` | Sustained request rate for login and OIDC endpoints (maps to OPENLICENSD_RATE_LIMIT_LOGIN_PER_MINUTE) |
| config.rateLimit.publicBurst | int | `60` | Burst capacity for public endpoints (maps to OPENLICENSD_RATE_LIMIT_PUBLIC_BURST) |
| config.rateLimit.publicPerMinute | int | `600` | Sustained request rate for /validate and /registry-credentials (maps to OPENLICENSD_RATE_LIMIT_PUBLIC_PER_MINUTE) |
| config.requestTimeoutSeconds | int | `30` | Per-request context deadline in seconds; 0 disables (maps to OPENLICENSD_REQUEST_TIMEOUT_SECONDS) |
| config.sessionCleanupIntervalMinutes | int | `60` | Interval in minutes for deleting expired/revoked sessions; 0 disables (maps to OPENLICENSD_SESSION_CLEANUP_INTERVAL_MINUTES) |
| config.sessionTTLHours | int | `24` | Session TTL in hours (maps to OPENLICENSD_SESSION_TTL_HOURS) |
| config.trustedProxies | string | `""` | Comma-separated trusted proxy IPs or CIDRs (maps to OPENLICENSD_TRUSTED_PROXIES) |
| extraArgs | list | `[]` | Extra command-line arguments passed to OpenLicensd |
| extraEnv | list | `[]` | Extra environment variables for the OpenLicensd container |
| fullnameOverride | string | `""` | Override the full release name used for all resources |
| image.pullPolicy | string | `"IfNotPresent"` | Image pull policy |
| image.repository | string | `"ghcr.io/alvarorg14/openlicensd"` | Container image repository |
| image.tag | string | `""` | Image tag. Defaults to the chart appVersion when empty |
| imagePullSecrets | list | `[]` | Secrets for pulling images from private registries |
| ingress.annotations | object | `{}` | Ingress annotations |
| ingress.className | string | `""` | Ingress class name |
| ingress.enabled | bool | `false` | Create an Ingress resource |
| ingress.hosts | list | `[{"host":"openlicensd.local","paths":[{"path":"/","pathType":"Prefix"}]}]` | Ingress host rules |
| ingress.tls | list | `[]` | Ingress TLS configuration |
| nameOverride | string | `""` | Override the chart name used in labels and resource names |
| nodeSelector | object | `{}` | Node labels for pod assignment |
| networkPolicy.allowExternal | bool | `true` | Allow ingress from any source to the HTTP and metrics ports |
| networkPolicy.allowExternalEgress | bool | `true` | Allow egress to any destination (required for external PostgreSQL, OIDC, and Harbor) |
| networkPolicy.annotations | object | `{}` | Annotations to add to the NetworkPolicy |
| networkPolicy.enabled | bool | `false` | Create a NetworkPolicy for OpenLicensd pods |
| networkPolicy.extraEgress | list | `[]` | Additional egress rules appended to the policy |
| networkPolicy.extraIngress | list | `[]` | Additional ingress rules appended to the policy |
| pdb.annotations | object | `{}` | Annotations to add to the PodDisruptionBudget |
| pdb.enabled | bool | `false` | Create a PodDisruptionBudget for OpenLicensd pods |
| pdb.maxUnavailable | int | `1` | Maximum pods that may be unavailable during voluntary disruptions. Used when minAvailable is empty. |
| pdb.minAvailable | string | `""` | Minimum pods that must remain available during voluntary disruptions. When set (including 0), takes precedence over maxUnavailable. Leave empty to use maxUnavailable instead. |
| podAnnotations | object | `{}` | Annotations to add to OpenLicensd pods |
| podLabels | object | `{}` | Labels to add to OpenLicensd pods |
| podSecurityContext | object | `{"fsGroup":65532,"runAsGroup":65532,"runAsNonRoot":true,"runAsUser":65532}` | Pod-level security context |
| replicaCount | int | `1` | Number of OpenLicensd replicas |
| resources | object | `{"limits":{"memory":"256Mi"},"requests":{"cpu":"50m","memory":"256Mi"}}` | CPU and memory resource requests and limits for the OpenLicensd container |
| secret.data | object | `{"bootstrapAdminPasswordHash":"","databaseUrl":"","harborAdminPassword":"","harborAdminUsername":"","oidcClientSecret":""}` | Secret data when mode is `create` |
| secret.data.bootstrapAdminPasswordHash | string | `""` | Bcrypt hash for bootstrap admin password (maps to OPENLICENSD_BOOTSTRAP_ADMIN_PASSWORD_HASH) |
| secret.data.databaseUrl | string | `""` | PostgreSQL connection URL (maps to OPENLICENSD_DATABASE_URL) |
| secret.data.harborAdminPassword | string | `""` | Harbor admin password (maps to OPENLICENSD_HARBOR_ADMIN_PASSWORD) |
| secret.data.harborAdminUsername | string | `""` | Harbor admin username (maps to OPENLICENSD_HARBOR_ADMIN_USERNAME) |
| secret.data.oidcClientSecret | string | `""` | OIDC client secret (maps to OPENLICENSD_OIDC_CLIENT_SECRET) |
| secret.existingSecret | string | `""` | Name of an existing Secret when mode is `existing` |
| secret.externalSecrets.refreshInterval | string | `"1h"` | How often the ExternalSecret refreshes the target Secret |
| secret.externalSecrets.remoteRefs | list | `[]` | Remote secret references. Each entry maps a Kubernetes Secret key to a remote property. Example: remoteRefs:   - secretKey: OPENLICENSD_DATABASE_URL     remoteRef:       key: openlicensd/database       property: url |
| secret.externalSecrets.secretStoreRef | object | `{"kind":"SecretStore","name":""}` | Reference to a SecretStore or ClusterSecretStore |
| secret.mode | string | `"create"` | Secret provisioning mode: `create`, `existing`, or `externalSecrets` |
| secret.name | string | `""` | Secret name when mode is `create` or `externalSecrets`. Defaults to `<release>-openlicensd-secret` |
| securityContext | object | `{"allowPrivilegeEscalation":false,"capabilities":{"drop":["ALL"]},"readOnlyRootFilesystem":true}` | Container-level security context |
| service.annotations | object | `{}` | Annotations to add to the Service |
| service.port | int | `8080` | Service port exposed for the API and UI |
| service.type | string | `"ClusterIP"` | Kubernetes Service type |
| serviceAccount.annotations | object | `{}` | Annotations to add to the ServiceAccount |
| serviceAccount.create | bool | `true` | Create a dedicated ServiceAccount for OpenLicensd |
| serviceAccount.name | string | `""` | ServiceAccount name. Generated from the release fullname when empty and create is true |
| tolerations | list | `[]` | Tolerations for pod assignment |
| topologySpreadConstraints | list | `[]` | Topology spread constraints for pod assignment |

----------------------------------------------
Autogenerated from chart metadata using [helm-docs v1.14.2](https://github.com/norwoodj/helm-docs/releases/v1.14.2)
