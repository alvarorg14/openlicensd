import DefaultTheme from 'vitepress/theme'
import type { Theme } from 'vitepress'
import { theme, useOpenapi, useTheme } from 'vitepress-openapi/client'
import 'vitepress-openapi/dist/style.css'
import spec from '../../openapi.yaml'
import './custom.css'

export default {
  extends: DefaultTheme,
  async enhanceApp({ app }) {
    useOpenapi({ spec })
    useTheme({ headingLevels: { h1: 2, h2: 3 } })
    theme.enhanceApp({ app })
  },
} satisfies Theme
