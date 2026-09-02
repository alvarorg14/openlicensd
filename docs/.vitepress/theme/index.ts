import DefaultTheme from 'vitepress/theme'
import type { Theme } from 'vitepress'
import { theme, useOpenapi } from 'vitepress-openapi/client'
import 'vitepress-openapi/dist/style.css'
import spec from '../../openapi.yaml'
import './custom.css'

export default {
  extends: DefaultTheme,
  async enhanceApp({ app }) {
    useOpenapi({ spec })
    theme.enhanceApp({ app })
  },
} satisfies Theme
