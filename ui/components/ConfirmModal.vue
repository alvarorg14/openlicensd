<template>
  <UModal v-model:open="open" :title="title">
    <template #body>
      <div class="space-y-4">
        <p class="text-sm text-gray-600 dark:text-gray-300">
          {{ description }}
        </p>

        <div class="flex justify-end gap-2">
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
