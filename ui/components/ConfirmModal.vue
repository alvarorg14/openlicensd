<template>
  <UModal v-model:open="open" :title="title">
    <template #header>
      <div class="flex items-center gap-2">
        <div
          class="flex h-8 w-8 items-center justify-center rounded-lg"
          :class="confirmColor === 'error' ? 'bg-red-100 dark:bg-red-900/40' : 'bg-indigo-100 dark:bg-indigo-900/40'"
        >
          <UIcon
            :name="confirmColor === 'error' ? 'i-lucide-alert-triangle' : 'i-lucide-info'"
            class="h-4 w-4"
            :class="confirmColor === 'error' ? 'text-red-600 dark:text-red-400' : 'text-indigo-600 dark:text-indigo-400'"
          />
        </div>
        <span class="font-semibold">{{ title }}</span>
      </div>
    </template>
    <template #body>
      <div class="space-y-4">
        <p class="text-sm text-slate-600 dark:text-slate-300 leading-relaxed">
          {{ description }}
        </p>

        <div class="flex justify-end gap-2 pt-2">
          <UButton color="neutral" variant="outline" @click="close">
            Cancel
          </UButton>
          <UButton :color="confirmColor" :loading="loading" @click="onConfirm">
            {{ confirmLabel }}
          </UButton>
        </div>
      </div>
    </template>
  </UModal>
</template>

<script setup lang="ts">
defineProps<{
  title: string
  description: string
  confirmLabel: string
  confirmColor: 'error' | 'primary'
  loading?: boolean
}>()

const open = defineModel<boolean>('open', { required: true })
const emit = defineEmits<{
  confirm: []
}>()

const close = () => {
  open.value = false
}

const onConfirm = () => {
  emit('confirm')
}
</script>
