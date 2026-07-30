<template>
  <header class="sticky top-0 z-30 border-b border-slate-200/80 dark:border-slate-800/80 bg-white/80 dark:bg-slate-950/80 backdrop-blur-md">
    <UContainer class="flex h-14 items-center justify-between">
      <div v-if="user" class="flex items-center gap-2 min-w-0">
        <span class="text-sm font-medium text-slate-900 dark:text-white truncate">{{ user.name }}</span>
        <UBadge color="neutral" variant="subtle" size="sm" class="capitalize shrink-0">{{ user.role }}</UBadge>
      </div>
      <div class="flex items-center gap-2 ml-auto">
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
const { user, logout } = useAuth()
</script>
