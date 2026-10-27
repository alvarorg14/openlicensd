import { defineComponent, h, onMounted, watch } from 'vue'
import { useData } from 'vitepress'
import { createMermaidRenderer } from 'vitepress-mermaid-renderer'
import DefaultTheme from 'vitepress/theme'
import type { Theme } from 'vitepress'
import { theme, useOpenapi, useTheme } from 'vitepress-openapi/client'
import 'vitepress-openapi/dist/style.css'
import spec from '../../openapi.yaml'
import './custom.css'

export default {
  extends: DefaultTheme,
  Layout: defineComponent({
    setup() {
      const { isDark } = useData()
      const renderDiagrams = () => createMermaidRenderer({
        static: true,
        startOnLoad: false,
        layout: 'dagre',
        look: 'classic',
        theme: isDark.value ? 'dark' : 'default',
      })
      onMounted(renderDiagrams)
      watch(isDark, renderDiagrams)
      return () => h(DefaultTheme.Layout)
    },
  }),
  async enhanceApp({ app }) {
    useOpenapi({ spec })
    useTheme({ headingLevels: { h1: 2, h2: 3 } })
    theme.enhanceApp({ app })
  },
} satisfies Theme
