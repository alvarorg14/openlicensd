<template>
  <header class="sticky top-0 z-30 border-b border-slate-200/80 dark:border-slate-800/80 bg-white/80 dark:bg-slate-950/80 backdrop-blur-md">
    <UContainer class="flex h-14 items-center justify-end">
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
const { logout } = useAuth()
</script>
