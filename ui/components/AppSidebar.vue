<template>
  <UDashboardSidebar
    collapsible
    resizable
    :toggle="false"
    :default-size="15"
    :min-size="14"
    :max-size="20"
    :collapsed-size="4"
    class="border-e-0 border-r border-white/10 bg-navy-900 dark:bg-navy-950"
    :ui="{
      header: 'border-b border-white/10 px-3',
      body: 'p-3',
      footer: 'flex-col items-stretch gap-1 border-t border-white/10 px-2 py-2'
    }"
  >
    <template #header="{ collapsed }">
      <div v-if="collapsed" class="group grid w-full place-items-center">
        <BrandMark class="col-start-1 row-start-1 h-8 w-auto text-white transition-opacity group-hover:opacity-0" />
        <UDashboardSidebarCollapse
          color="neutral"
          class="col-start-1 row-start-1 opacity-0 transition-opacity group-hover:opacity-100 focus-visible:opacity-100 text-navy-300 hover:bg-white/10 hover:text-white"
        />
      </div>
      <template v-else>
        <div class="flex flex-1 items-center gap-2 min-w-0 overflow-hidden">
          <BrandMark class="h-8 w-auto text-white shrink-0" />
          <BrandWordmark class="h-6 w-auto text-white shrink-0" />
        </div>
        <UDashboardSidebarCollapse
          color="neutral"
          class="text-navy-300 hover:text-white hover:bg-white/10"
        />
      </template>
    </template>

    <template #default="{ collapsed }">
      <UNavigationMenu
        :items="navItems"
        orientation="vertical"
        :collapsed="collapsed"
        tooltip
        :ui="navMenuUi(collapsed)"
      />
    </template>

    <template #footer="{ collapsed }">
      <a
        :href="GITHUB_URL"
        target="_blank"
        rel="noopener noreferrer"
        class="flex items-center gap-2 rounded-lg px-3 py-2 text-sm font-medium text-navy-300 hover:bg-white/10 hover:text-white transition-app"
        :class="collapsed ? 'justify-center' : ''"
        :title="collapsed ? 'Star us on GitHub' : undefined"
      >
        <UIcon name="i-lucide-github" class="h-4 w-4 shrink-0" />
        <span v-if="!collapsed">Star us on GitHub</span>
      </a>
      <UserMenu on-dark :compact="collapsed" />
      <p
        v-if="serverVersion && !collapsed"
        class="pt-1 pb-0.5 text-center text-[10px] leading-none text-navy-400 tabular-nums"
      >
        {{ serverVersion }}
      </p>
    </template>
  </UDashboardSidebar>

  <div class="lg:hidden fixed bottom-0 inset-x-0 z-40 border-t border-default bg-default/95 backdrop-blur-md">
    <nav class="flex justify-around p-2">
      <NuxtLink
        v-for="item in mobileNavItems"
        :key="item.to"
        :to="item.to"
        class="flex flex-col items-center gap-0.5 rounded-lg px-3 py-1.5 text-xs font-medium transition-app"
        :class="isActive(item.to)
          ? 'text-brand-600 dark:text-brand-400'
          : 'text-muted'"
      >
        <UIcon :name="item.icon" class="h-5 w-5" />
        {{ item.label }}
      </NuxtLink>
    </nav>
  </div>
</template>

<script setup lang="ts">
import type { NavigationMenuItem } from '@nuxt/ui'
import { GITHUB_URL } from '~/constants/app'

const route = useRoute()
const { isAdmin, serverVersion } = useAuth()

const baseNavItems: NavigationMenuItem[] = [
  { label: 'Licenses', to: '/licenses', icon: 'i-lucide-key' },
  { label: 'Products', to: '/products', icon: 'i-lucide-package' },
  { label: 'Policies', to: '/policies', icon: 'i-lucide-shield' }
]

const navItems = computed<NavigationMenuItem[]>(() => {
  if (isAdmin.value) {
    return [
      ...baseNavItems,
      { label: 'API Tokens', to: '/api-tokens', icon: 'i-lucide-key-round' },
      { label: 'Users', to: '/users', icon: 'i-lucide-users' }
    ]
  }
  return baseNavItems
})

const mobileNavItems = computed(() =>
  navItems.value
    .filter((item): item is NavigationMenuItem & { to: string } => typeof item.to === 'string')
)

const navMenuUi = (collapsed?: boolean) => ({
  list: 'flex flex-col gap-1.5',
  link: [
    collapsed ? 'justify-center px-2 py-2.5' : 'px-3 py-2.5',
    'text-navy-300 transition-app before:rounded-lg',
    'hover:text-white hover:before:bg-white/10',
    'data-[active]:text-white data-[active]:before:bg-brand-600'
  ].join(' '),
  linkLeadingIcon: 'text-current'
})

const isActive = (path: string) => route.path === path || route.path.startsWith(`${path}/`)
</script>
