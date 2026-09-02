import path from 'node:path'
import type MarkdownIt from 'markdown-it'

const GITHUB_BLOB = 'https://github.com/alvarorg14/openlicensd/blob/main/'

/** Wrapper pages include root-level markdown; all other pages live under docs/. */
const ROOT_PAGES = new Set(['overview.md', 'quickstart.md', 'contributing.md'])

const SITE_PAGES: Record<string, string> = {
  'README.md': '/overview',
  'QUICKSTART.md': '/quickstart',
  'CONTRIBUTING.md': '/contributing',
  'docs/openapi.yaml': '/api-reference',
}

function getBaseDir(relativePath: string): string {
  if (ROOT_PAGES.has(relativePath)) {
    return ''
  }
  return 'docs'
}

function resolveRepoPath(href: string, relativePath: string): string {
  const hashIndex = href.indexOf('#')
  const queryIndex = href.indexOf('?')
  const endIndex = Math.min(
    hashIndex === -1 ? href.length : hashIndex,
    queryIndex === -1 ? href.length : queryIndex,
  )
  const filePart = href.slice(0, endIndex)

  if (!filePart || filePart.startsWith('/')) {
    return href
  }

  const baseDir = getBaseDir(relativePath)
  const resolved = path.posix.normalize(path.posix.join(baseDir || '.', filePart))
  const suffix = href.slice(endIndex)
  return resolved + suffix
}

function mapRepoPathToSite(repoPath: string): string {
  const hashIndex = repoPath.indexOf('#')
  const queryIndex = repoPath.indexOf('?')
  const endIndex = Math.min(
    hashIndex === -1 ? repoPath.length : hashIndex,
    queryIndex === -1 ? repoPath.length : queryIndex,
  )
  const filePart = repoPath.slice(0, endIndex)
  const suffix = repoPath.slice(endIndex)
  const normalized = filePart.replace(/^\.\//, '')

  if (SITE_PAGES[normalized]) {
    return SITE_PAGES[normalized]! + suffix
  }

  if (normalized.startsWith('docs/') && normalized.endsWith('.md')) {
    return `/${normalized.slice('docs/'.length, -'.md'.length)}${suffix}`
  }

  return `${GITHUB_BLOB}${normalized}${suffix}`
}

function shouldSkip(href: string): boolean {
  return (
    href.startsWith('http://')
    || href.startsWith('https://')
    || href.startsWith('mailto:')
    || href.startsWith('tel:')
    || href.startsWith('#')
  )
}

export function repoLinksPlugin(md: MarkdownIt): void {
  const defaultRender = md.renderer.rules.link_open
    ?? ((tokens, idx, options, _env, self) => self.renderToken(tokens, idx, options))

  md.renderer.rules.link_open = (tokens, idx, options, env, self) => {
    const token = tokens[idx]
    const hrefIndex = token.attrIndex('href')

    if (hrefIndex >= 0) {
      const href = token.attrs![hrefIndex][1]
      const relativePath = typeof env?.relativePath === 'string' ? env.relativePath : ''

      if (!shouldSkip(href) && relativePath) {
        const repoPath = resolveRepoPath(href, relativePath)
        const siteHref = mapRepoPathToSite(repoPath)
        token.attrs![hrefIndex][1] = siteHref
      }
    }

    return defaultRender(tokens, idx, options, env, self)
  }
}
