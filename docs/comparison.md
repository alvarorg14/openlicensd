# Comparison with Alternatives

This page helps you decide whether OpenLicensd fits your licensing needs compared to
[Keygen](https://keygen.sh/), [Cryptlex](https://cryptlex.com/), and
[LicenseSpring](https://licensespring.com/) — three widely used commercial licensing platforms.

Facts below were verified on **2026-09-02** from each vendor's public documentation and pricing
pages. Vendor plans and licenses change; if something is outdated, please open a pull request with
a link to the updated source.

## At a glance

| | [OpenLicensd](https://github.com/alvarorg14/openlicensd) | [Keygen](https://keygen.sh/) | [Cryptlex](https://cryptlex.com/) | [LicenseSpring](https://licensespring.com/) |
|---|---|---|---|---|
| **License** | [Apache 2.0](https://github.com/alvarorg14/openlicensd/blob/main/LICENSE) — fully open source | [Fair Core License](https://fcl.dev) — source-available; each version converts to Apache 2.0 after two years ([license](https://keygen.sh/license/)) | Proprietary | Proprietary |
| **Managed SaaS** | No | [Keygen Cloud](https://keygen.sh/) — metered by active licensed users | [SaaS tiers](https://cryptlex.com/pricing) from Starter upward | [SaaS tiers](https://licensespring.com/pricing) including a free tier |
| **Self-host option** | Yes (only deployment model) | Yes — [Community Edition](https://keygen.sh/docs/self-hosting/) (free) and [Enterprise Edition](https://keygen.sh/docs/self-hosting/) (paid license key) | Yes — [Enterprise plan](https://cryptlex.com/pricing) with commercial server license key | Yes — [Enterprise plan](https://licensespring.com/solutions/on-premises-licensing) |
| **Cost model** | Free (you pay for your own infrastructure) | CE free to self-host; EE flat-rate license; Cloud subscription by active licensed users | Subscription by tier; self-hosted requires Enterprise | Subscription by tier; self-hosted requires Enterprise |
| **Self-host runtime** | Single Go binary + PostgreSQL | Rails app — Ruby ≥ 3.1, PostgreSQL ≥ 13, Redis ≥ 6.2 ([docs](https://keygen.sh/docs/self-hosting/)) | Multi-service platform — Web API, portals, release server ([Helm chart](https://github.com/cryptlex/helm-charts)) | Vendor platform with on-premise floating license server and air-gapped workflows ([docs](https://licensespring.com/solutions/on-premises-licensing)) |

## About each alternative

### Keygen

Keygen is the closest peer in the self-hosted space. Its Community Edition is free to self-host for
personal and commercial use, and the codebase is source-available under the Fair Core License.
Keygen Cloud is a mature managed offering with a broad SDK ecosystem and a distribution API for
shipping software artifacts. Enterprise Edition adds audit logs, environments, and advanced
permissions behind a paid license key.

**Where Keygen wins today:** managed SaaS with no ops burden, distribution API, wider client SDK
coverage, and a longer track record in production.

### Cryptlex

Cryptlex is a proprietary platform available primarily as SaaS, with self-hosted deployment on the
Enterprise plan. It provides a full vendor stack — license management, customer portals, release
server, and client libraries (LexActivator, LexFloatClient). On-premise floating licenses via
LexFloatServer suit air-gapped customer networks.

**Where Cryptlex wins today:** hosted floating-license server (LexFloatServer), offline activation
workflows, release management, and a mature multi-language client SDK suite.

### LicenseSpring

LicenseSpring is a proprietary, API-first enterprise licensing platform. SaaS tiers cover most
use cases; the Enterprise plan adds self-hosted deployment, on-premise floating license servers,
air-gapped activation, and hardware-key binding for regulated industries.

**Where LicenseSpring wins today:** air-gapped and hardware-key licensing workflows, enterprise SLAs,
and a unified vendor platform for hybrid SaaS plus on-premise product portfolios.

## Choose OpenLicensd when

- You want a **fully open-source** license server under Apache 2.0 with no vendor lock-in.
- You prefer a **single binary** deployment (API + admin UI embedded) over a multi-service stack.
- You need **self-hosted** licensing with no per-seat or per-activation SaaS fees.
- Your stack is **Go-centric** or you are comfortable calling a simple REST validation API directly.
- You want **Harbor registry credential** issuance tied to license validation (uncommon among peers).
- You need an **append-only audit log** and **scoped API tokens** for admin automation.
- You operate **Kubernetes** and want a Helm chart with health probes, HPA, and NetworkPolicy out of the box.

## Consider an alternative when

- You want a **managed SaaS** offering — OpenLicensd is self-host only; you operate PostgreSQL and backups yourself.
- You need **offline or cryptographically signed licenses** that validate without contacting the server.
- You need **entitlements, feature flags, or usage-based metering** beyond validation counts.
- You need **webhooks** for license lifecycle events (creation, expiration, revocation).
- You need **floating/lease licensing with heartbeats** — OpenLicensd enforces max concurrent machine activations with admin release, which is not the same as a floating seat pool with automatic lease expiry.
- You need an **end-customer self-service portal** for license management.
- You need a **release or artifact distribution API** to ship software updates to licensed users.
- You need **built-in payment or billing integrations** (Stripe, Paddle, etc.).
- You need **client SDKs** beyond Go (Keygen, Cryptlex, and LicenseSpring cover many languages).
- You need a **stable v1 API** today — OpenLicensd is pre-1.0; see [ROADMAP.md](../ROADMAP.md) for the path to v1.0.0.

## What OpenLicensd does not do today

The following are on the [Post-1.0 backlog](../ROADMAP.md) or not planned — not gaps we are hiding:

| Capability | Status |
|---|---|
| Offline / signed licenses | Post-1.0 |
| Entitlements and metadata | Post-1.0 |
| Webhooks | Post-1.0 |
| Floating licenses with heartbeats | Post-1.0 |
| Client-side machine deactivation | Post-1.0 |
| Bulk import/export | Post-1.0 |
| End-customer portal | Not planned (admin UI only) |
| Release / artifact distribution | Not planned |
| Built-in payments | Not planned |
| Managed SaaS hosting | Not planned (self-host only) |

## See also

- [README.md](../README.md) — project overview and quick comparison table
- [ROADMAP.md](../ROADMAP.md) — release phases and Post-1.0 features
- [QUICKSTART.md](../QUICKSTART.md) — get running in minutes
