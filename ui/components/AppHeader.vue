<template>
  <header class="sticky top-0 z-40 border-b border-slate-200/80 dark:border-slate-800/80 bg-white/80 dark:bg-slate-950/80 backdrop-blur-md">
    <UContainer class="flex h-14 items-center justify-between">
      <div class="flex items-center gap-3">
        <div class="flex h-8 w-8 items-center justify-center rounded-lg bg-indigo-600">
          <UIcon name="i-lucide-key-round" class="h-4 w-4 text-white" />
        </div>
        <div>
          <h1 class="text-sm font-semibold text-slate-900 dark:text-white leading-tight">{{ APP_NAME }}</h1>
          <p class="text-xs text-slate-500 dark:text-slate-400">License management</p>
        </div>
      </div>

      <div class="flex items-center gap-2">
        <UColorModeButton />
        <UButton
          color="neutral"
          variant="ghost"
          icon="i-lucide-log-out"
          size="sm"
          @click="logout"
        >
          Logout
        </UButton>
      </div>
    </UContainer>
  </header>
</template>

<script setup lang="ts">
import { APP_NAME } from '~/constants/app'

const { logout } = useAuth()
</script>
