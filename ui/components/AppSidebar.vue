<template>
  <aside class="hidden lg:flex w-60 shrink-0 flex-col min-h-screen border-r border-white/10 bg-navy-900 dark:bg-navy-950">
    <div class="shrink-0 flex h-16 items-center gap-3 px-4 border-b border-white/10">
      <BrandMark class="h-9 w-auto text-white shrink-0" />
      <BrandWordmark class="h-7 w-auto text-white shrink-0" />
    </div>

    <nav class="flex-1 min-h-0 overflow-y-auto p-3 space-y-1">
      <NuxtLink
        v-for="item in navItems"
        :key="item.to"
        :to="item.to"
        class="flex items-center gap-2 rounded-lg px-3 py-2 text-sm font-medium transition-app"
        :class="isActive(item.to)
          ? 'bg-brand-600 text-white'
          : 'text-navy-300 hover:bg-white/10 hover:text-white'"
      >
        <UIcon :name="item.icon" class="h-4 w-4 shrink-0" />
        {{ item.label }}
      </NuxtLink>
    </nav>

    <div class="mt-auto shrink-0 border-t border-white/10 p-2 space-y-1">
      <a
        :href="GITHUB_URL"
        target="_blank"
        rel="noopener noreferrer"
        class="flex items-center gap-2 rounded-lg px-3 py-2 text-sm font-medium text-navy-300 hover:bg-white/10 hover:text-white transition-app"
      >
        <UIcon name="i-lucide-github" class="h-4 w-4 shrink-0" />
        Star us on GitHub
      </a>
      <UserMenu on-dark />
      <p v-if="serverVersion" class="pt-1 pb-0.5 text-center text-[10px] leading-none text-navy-400 tabular-nums">
        {{ serverVersion }}
      </p>
    </div>
  </aside>

  <div class="lg:hidden fixed bottom-0 inset-x-0 z-40 border-t border-default bg-default/95 backdrop-blur-md">
    <nav class="flex justify-around p-2">
      <NuxtLink
        v-for="item in navItems"
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
import { GITHUB_URL } from '~/constants/app'

const route = useRoute()
const { isAdmin, serverVersion } = useAuth()

const baseNavItems = [
  { label: 'Licenses', to: '/licenses', icon: 'i-lucide-key' },
  { label: 'Products', to: '/products', icon: 'i-lucide-package' },
  { label: 'Policies', to: '/policies', icon: 'i-lucide-shield' }
]

const navItems = computed(() => {
  if (isAdmin.value) {
    return [...baseNavItems, { label: 'Users', to: '/users', icon: 'i-lucide-users' }]
  }
  return baseNavItems
})

const isActive = (path: string) => route.path === path || route.path.startsWith(`${path}/`)
</script>
