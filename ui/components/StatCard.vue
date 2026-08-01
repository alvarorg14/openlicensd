<template>
  <div
    class="rounded-xl border border-default bg-default p-4 shadow-app transition-app hover:shadow-app-lg hover:border-brand-200/50 dark:hover:border-brand-800/30 animate-fade-in-up"
    :class="staggerClass"
  >
    <div class="flex items-start justify-between">
      <div>
        <p class="text-xs font-medium uppercase tracking-wider text-muted dark:text-dimmed">
          {{ label }}
        </p>
        <p class="mt-1 text-2xl font-medium tracking-brand text-highlighted tabular-nums">
          {{ count }}
        </p>
      </div>
      <div
        class="flex h-9 w-9 items-center justify-center rounded-lg"
        :class="iconBgClass"
      >
        <UIcon :name="icon" class="h-4 w-4" :class="iconClass" />
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
defineProps<{
  label: string
  count: number
  icon: string
  iconBgClass?: string
  iconClass?: string
  staggerClass?: string
}>()
</script>
