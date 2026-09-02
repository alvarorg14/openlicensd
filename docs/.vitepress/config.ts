import yaml from '@rollup/plugin-yaml'
import { defineConfig } from 'vitepress'
import { repoLinksPlugin } from './plugins/repoLinks'

export default defineConfig({
  base: '/openlicensd/',
  title: 'OpenLicensd',
  description: 'Open source license server for creating and validating license keys',
  srcDir: '.',
  srcExclude: ['README.md', 'brand/README.md', 'node_modules/**'],

  ignoreDeadLinks: [
    /^https?:\/\/localhost(?::\d+)?/,
  ],

  head: [
    ['link', { rel: 'icon', href: '/openlicensd/brand/mark-light.svg', type: 'image/svg+xml' }],
  ],

  vite: {
    plugins: [
      yaml(),
    ],
  },

  markdown: {
    config(md) {
      md.use(repoLinksPlugin)
    },
  },

  themeConfig: {
    logo: '/brand/mark-light.svg',
    siteTitle: 'OpenLicensd',

    nav: [
      { text: 'Overview', link: '/overview' },
      { text: 'Quick Start', link: '/quickstart' },
      { text: 'API Reference', link: '/api-reference' },
      {
        text: 'GitHub',
        link: 'https://github.com/alvarorg14/openlicensd',
      },
    ],

    sidebar: [
      {
        text: 'Introduction',
        items: [
          { text: 'Home', link: '/' },
          { text: 'Overview', link: '/overview' },
          { text: 'Quick Start', link: '/quickstart' },
          { text: 'Comparison', link: '/comparison' },
          { text: 'Architecture', link: '/architecture' },
        ],
      },
      {
        text: 'Reference',
        items: [
          { text: 'API Guide', link: '/api' },
          { text: 'OpenAPI Spec', link: '/api-reference' },
          { text: 'Configuration', link: '/configuration' },
          { text: 'Metrics', link: '/metrics' },
          { text: 'Go SDK', link: '/sdk/go' },
        ],
      },
      {
        text: 'Integrations',
        items: [
          { text: 'OIDC SSO', link: '/oidc-sso' },
          { text: 'Harbor Registry', link: '/harbor-registry-credentials' },
        ],
      },
      {
        text: 'Operations',
        items: [
          { text: 'Deployment', link: '/deployment' },
          { text: 'Upgrade', link: '/upgrade' },
          { text: 'Backup & Restore', link: '/backup-restore' },
          { text: 'Scaling', link: '/scaling' },
          { text: 'Troubleshooting', link: '/troubleshooting' },
        ],
      },
      {
        text: 'Project',
        items: [
          { text: 'Contributing', link: '/contributing' },
        ],
      },
    ],

    socialLinks: [
      { icon: 'github', link: 'https://github.com/alvarorg14/openlicensd' },
    ],

    search: {
      provider: 'local',
    },

    editLink: {
      pattern: 'https://github.com/alvarorg14/openlicensd/edit/main/:path',
      text: 'Edit this page on GitHub',
    },

    footer: {
      message: 'Apache License 2.0',
      copyright: 'OpenLicensd contributors',
    },
  },

  sitemap: {
    hostname: 'https://alvarorg14.github.io',
  },
})
