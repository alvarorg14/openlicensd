<template>
  <aside class="hidden lg:flex w-60 shrink-0 flex-col border-r border-slate-200/80 dark:border-slate-800/80 bg-white/80 dark:bg-slate-950/80 backdrop-blur-md">
    <div class="flex h-14 items-center gap-3 px-4 border-b border-slate-200/80 dark:border-slate-800/80">
      <div class="flex h-8 w-8 items-center justify-center rounded-lg bg-indigo-600">
        <UIcon name="i-lucide-key-round" class="h-4 w-4 text-white" />
      </div>
      <div>
        <h1 class="text-sm font-semibold text-slate-900 dark:text-white leading-tight">{{ APP_NAME }}</h1>
        <p class="text-xs text-slate-500 dark:text-slate-400">License management</p>
      </div>
    </div>

    <nav class="flex-1 p-3 space-y-1">
      <NuxtLink
        v-for="item in navItems"
        :key="item.to"
        :to="item.to"
        class="flex items-center gap-2 rounded-lg px-3 py-2 text-sm font-medium transition-app"
        :class="isActive(item.to)
          ? 'bg-indigo-50 text-indigo-700 dark:bg-indigo-900/30 dark:text-indigo-300'
          : 'text-slate-600 hover:bg-slate-100 dark:text-slate-400 dark:hover:bg-slate-800/50'"
      >
        <UIcon :name="item.icon" class="h-4 w-4 shrink-0" />
        {{ item.label }}
      </NuxtLink>
    </nav>

    <div class="border-t border-slate-200/80 dark:border-slate-800/80 p-2">
      <UserMenu />
    </div>
  </aside>

  <div class="lg:hidden fixed bottom-0 inset-x-0 z-40 border-t border-slate-200/80 dark:border-slate-800/80 bg-white/95 dark:bg-slate-950/95 backdrop-blur-md">
    <nav class="flex justify-around p-2">
      <NuxtLink
        v-for="item in navItems"
        :key="item.to"
        :to="item.to"
        class="flex flex-col items-center gap-0.5 rounded-lg px-3 py-1.5 text-xs font-medium transition-app"
        :class="isActive(item.to)
          ? 'text-indigo-600 dark:text-indigo-400'
          : 'text-slate-500 dark:text-slate-400'"
      >
        <UIcon :name="item.icon" class="h-5 w-5" />
        {{ item.label }}
      </NuxtLink>
    </nav>
  </div>
</template>

<script setup lang="ts">
import { APP_NAME } from '~/constants/app'

const route = useRoute()
const { isAdmin } = useAuth()

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
