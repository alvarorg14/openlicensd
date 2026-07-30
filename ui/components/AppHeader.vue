<template>
  <header class="lg:hidden sticky top-0 z-30 border-b border-slate-200/80 dark:border-slate-800/80 bg-white/80 dark:bg-slate-950/80 backdrop-blur-md">
    <UContainer class="flex h-14 items-center justify-between">
      <div class="flex items-center gap-3 min-w-0">
        <div class="flex h-8 w-8 items-center justify-center rounded-lg bg-indigo-600 shrink-0">
          <UIcon name="i-lucide-key-round" class="h-4 w-4 text-white" />
        </div>
        <h1 class="text-sm font-semibold text-slate-900 dark:text-white truncate">{{ APP_NAME }}</h1>
      </div>
      <UserMenu side="bottom" compact />
    </UContainer>
  </header>
</template>

<script setup lang="ts">
import { APP_NAME } from '~/constants/app'
</script>
