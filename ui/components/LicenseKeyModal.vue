<template>
  <UModal v-model:open="open" title="License created">
    <template #body>
      <div class="space-y-4">
        <UAlert
          color="warning"
          variant="subtle"
          title="Copy this key now"
          description="The full license key is shown only once and cannot be retrieved later."
        />

        <div v-if="label" class="text-sm text-gray-500">
          Label: <span class="font-medium text-gray-900 dark:text-white">{{ label }}</span>
        </div>

        <div class="flex gap-2">
          <UInput :model-value="licenseKey" readonly class="flex-1 font-mono text-sm" />
          <UButton icon="i-lucide-copy" @click="copyKey">
            Copy
          </UButton>
        </div>

        <div class="flex justify-end">
          <UButton @click="open = false">
            Done
          </UButton>
        </div>
      </div>
    </template>
  </UModal>
</template>

<script setup lang="ts">
const open = defineModel<boolean>('open', { required: true })

const props = defineProps<{
  licenseKey: string
  label: string
}>()

const copyKey = async () => {
  if (!props.licenseKey) return
  await navigator.clipboard.writeText(props.licenseKey)
}
</script>
