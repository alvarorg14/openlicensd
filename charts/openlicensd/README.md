# openlicensd

Open source license server for creating and validating license keys

![Version: 0.1.0](https://img.shields.io/badge/Version-0.1.0-informational?style=flat-square) ![Type: application](https://img.shields.io/badge/Type-application-informational?style=flat-square) ![AppVersion: 0.1.0](https://img.shields.io/badge/AppVersion-0.1.0-informational?style=flat-square)

## Installing

Requires an external PostgreSQL database. The chart does not bundle a database.

```bash
helm install openlicensd oci://ghcr.io/alvarorg14/charts/openlicensd \
  --version X.Y.Z \
  --namespace openlicensd \
  --create-namespace \
  --set secret.data.databaseUrl='postgres://user:pass@host:5432/openlicensd?sslmode=disable' \
  --set secret.data.adminPasswordHash='$2a$10$...' \
  --set secret.data.jwtSecret='change-me'
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
  --set-json 'secret.externalSecrets.remoteRefs=[{"secretKey":"OPENLICENSD_DATABASE_URL","remoteRef":{"key":"openlicensd/database","property":"url"}},{"secretKey":"OPENLICENSD_ADMIN_PASSWORD_HASH","remoteRef":{"key":"openlicensd/admin","property":"passwordHash"}},{"secretKey":"OPENLICENSD_JWT_SECRET","remoteRef":{"key":"openlicensd/jwt","property":"secret"}}]'
```

For local development, install from the chart source:

```bash
helm install openlicensd ./charts/openlicensd \
  --namespace openlicensd \
  --create-namespace \
  -f charts/openlicensd/ci/test-values.yaml
```

## Values

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| affinity | object | `{}` | Affinity rules for pod assignment |
| config.addr | string | `":8080"` | HTTP listen address (maps to OPENLICENSD_ADDR) |
| config.adminUser | string | `"admin"` | Admin username (maps to OPENLICENSD_ADMIN_USER) |
| config.harbor.debug | bool | `false` | Log Harbor API requests/responses (maps to OPENLICENSD_HARBOR_DEBUG) |
| config.harbor.enabled | bool | `false` | Enable Harbor registry credentials endpoint |
| config.harbor.insecureSkipVerify | bool | `false` | Skip TLS verification for Harbor (maps to OPENLICENSD_HARBOR_INSECURE_SKIP_VERIFY) |
| config.harbor.projects | string | `""` | Comma-separated Harbor project namespaces (maps to OPENLICENSD_HARBOR_PROJECTS) |
| config.harbor.robotDurationDays | int | `1` | Robot account lifetime in days (maps to OPENLICENSD_HARBOR_ROBOT_DURATION_DAYS) |
| config.harbor.robotNamePrefix | string | `"openlicensd"` | Prefix for generated robot account names (maps to OPENLICENSD_HARBOR_ROBOT_NAME_PREFIX) |
| config.harbor.url | string | `""` | Harbor base URL (maps to OPENLICENSD_HARBOR_URL) |
| extraArgs | list | `[]` | Extra command-line arguments passed to openlicensd |
| extraEnv | list | `[]` | Extra environment variables for the openlicensd container |
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
| podAnnotations | object | `{}` | Annotations to add to openlicensd pods |
| podLabels | object | `{}` | Labels to add to openlicensd pods |
| podSecurityContext | object | `{"fsGroup":65532,"runAsGroup":65532,"runAsNonRoot":true,"runAsUser":65532}` | Pod-level security context |
| replicaCount | int | `1` | Number of openlicensd replicas |
| resources | object | `{"limits":{"memory":"256Mi"},"requests":{"cpu":"50m","memory":"128Mi"}}` | CPU and memory resource requests and limits for the openlicensd container |
| secret.data.adminPasswordHash | string | `""` | Bcrypt hash of the admin password (maps to OPENLICENSD_ADMIN_PASSWORD_HASH) |
| secret.data.databaseUrl | string | `""` | PostgreSQL connection URL (maps to OPENLICENSD_DATABASE_URL) |
| secret.data.harborAdminPassword | string | `""` | Harbor admin password (maps to OPENLICENSD_HARBOR_ADMIN_PASSWORD) |
| secret.data.harborAdminUsername | string | `""` | Harbor admin username (maps to OPENLICENSD_HARBOR_ADMIN_USERNAME) |
| secret.data.jwtSecret | string | `""` | JWT signing secret (maps to OPENLICENSD_JWT_SECRET) |
| secret.existingSecret | string | `""` | Name of an existing Secret when mode is `existing` |
| secret.externalSecrets.refreshInterval | string | `"1h"` | How often the ExternalSecret refreshes the target Secret |
| secret.externalSecrets.remoteRefs | list | `[]` | Remote secret references |
| secret.externalSecrets.secretStoreRef.kind | string | `"SecretStore"` | Reference kind |
| secret.externalSecrets.secretStoreRef.name | string | `""` | Reference to a SecretStore or ClusterSecretStore |
| secret.mode | string | `"create"` | Secret provisioning mode: `create`, `existing`, or `externalSecrets` |
| secret.name | string | `""` | Secret name when mode is `create` or `externalSecrets` |
| securityContext | object | `{"allowPrivilegeEscalation":false,"capabilities":{"drop":["ALL"]},"readOnlyRootFilesystem":true}` | Container-level security context |
| service.annotations | object | `{}` | Annotations to add to the Service |
| service.port | int | `8080` | Service port exposed for the API and UI |
| service.type | string | `"ClusterIP"` | Kubernetes Service type |
| serviceAccount.annotations | object | `{}` | Annotations to add to the ServiceAccount |
| serviceAccount.create | bool | `true` | Create a dedicated ServiceAccount for openlicensd |
| serviceAccount.name | string | `""` | ServiceAccount name |
| tolerations | list | `[]` | Tolerations for pod assignment |

----------------------------------------------
Autogenerated from chart metadata using [helm-docs v1.14.2](https://github.com/norwoodj/helm-docs/releases/v1.14.2)
