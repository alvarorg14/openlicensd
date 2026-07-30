<template>
  <UModal v-model:open="open" :title="title">
    <template #header>
      <div class="flex items-center gap-2">
        <div
          class="flex h-8 w-8 items-center justify-center rounded-lg"
          :class="iconBgClass"
        >
          <UIcon :name="icon" class="h-4 w-4" :class="iconClass" />
        </div>
        <span class="font-semibold">{{ title }}</span>
      </div>
    </template>
    <template #body>
      <div class="space-y-4">
        <slot name="top" />

        <dl class="space-y-3">
          <template v-for="(item, index) in items" :key="index">
            <div
              v-if="item.multiline"
              class="space-y-1"
            >
              <dt class="text-xs font-medium uppercase tracking-wide text-slate-500 dark:text-slate-400">
                {{ item.label }}
              </dt>
              <dd class="text-sm text-slate-700 dark:text-slate-300 whitespace-pre-wrap break-words">
                {{ item.value }}
              </dd>
            </div>
            <div
              v-else
              class="grid grid-cols-[minmax(0,8rem)_1fr] gap-x-4 gap-y-1 items-baseline"
            >
              <dt class="text-xs font-medium uppercase tracking-wide text-slate-500 dark:text-slate-400">
                {{ item.label }}
              </dt>
              <dd
                class="text-sm text-slate-700 dark:text-slate-300"
                :class="item.mono ? 'font-mono text-xs' : ''"
              >
                {{ item.value }}
              </dd>
            </div>
          </template>
        </dl>

        <div class="flex justify-end pt-2">
          <UButton @click="open = false">
            Close
          </UButton>
        </div>
      </div>
    </template>
  </UModal>
</template>

<script setup lang="ts">
import type { DetailItem } from '~/types'

defineProps<{
  title: string
  icon: string
  iconBgClass: string
  iconClass: string
  items: DetailItem[]
}>()

const open = defineModel<boolean>('open', { required: true })
</script>
