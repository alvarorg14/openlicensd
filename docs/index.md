---
layout: home

hero:
  name: OpenLicensd
  text: Open source license server
  tagline: Self-hosted. Single binary. PostgreSQL-backed.
  image:
    src: /brand/logo-light.svg
    alt: OpenLicensd
  actions:
    - theme: brand
      text: Quick Start
      link: /quickstart
    - theme: alt
      text: API Reference
      link: /api-reference
    - theme: alt
      text: GitHub
      link: https://github.com/alvarorg14/openlicensd

features:
  - title: Self-hosted
    details: No third-party license service dependency. Deploy on your infrastructure with a single Go binary and PostgreSQL.
  - title: Simple validation API
    details: Public POST /validate endpoint for client applications, with optional product scoping and machine fingerprinting.
  - title: Admin UI
    details: Embedded Nuxt admin interface for licenses, products, policies, users, API tokens, and audit events.
  - title: Harbor integration
    details: Optional short-lived registry credentials for licensed container image distribution.
  - title: OIDC SSO
    details: Optional single sign-on for admin login via any standards-compliant identity provider.
  - title: Kubernetes-ready
    details: Helm chart with Ingress, health probes, HPA, PDB, NetworkPolicy, and ServiceMonitor support.
---

## Admin UI

![OpenLicensd admin UI licenses list](/screenshots/admin-ui-licenses.png)
