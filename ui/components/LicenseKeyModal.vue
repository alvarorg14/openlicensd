<template>
  <UModal v-model:open="open" title="License created">
    <template #header>
      <div class="flex items-center gap-2">
        <div class="flex h-8 w-8 items-center justify-center rounded-lg bg-emerald-100 dark:bg-emerald-900/40">
          <UIcon name="i-lucide-check-circle" class="h-4 w-4 text-emerald-600 dark:text-emerald-400" />
        </div>
        <span class="font-semibold">License created</span>
      </div>
    </template>
    <template #body>
      <div class="space-y-4">
        <UAlert
          color="warning"
          variant="subtle"
          title="Copy this key now"
          description="The full license key is shown only once and cannot be retrieved later."
        />

        <div v-if="label" class="text-sm text-slate-500 dark:text-slate-400">
          Label: <span class="font-medium text-slate-900 dark:text-white">{{ label }}</span>
        </div>

        <div class="flex gap-2">
          <UInput :model-value="licenseKey" readonly class="flex-1 font-mono text-sm" />
          <UButton
            :icon="copied ? 'i-lucide-check' : 'i-lucide-copy'"
            :color="copied ? 'success' : 'primary'"
            class="transition-app shrink-0"
            @click="copyKey"
          >
            {{ copied ? 'Copied!' : 'Copy' }}
          </UButton>
        </div>

        <div class="flex justify-end pt-2">
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

const copied = ref(false)
let copyTimeout: ReturnType<typeof setTimeout> | null = null

watch(open, (value) => {
  if (!value) {
    copied.value = false
    if (copyTimeout) {
      clearTimeout(copyTimeout)
      copyTimeout = null
    }
  }
})

const copyKey = async () => {
  if (!props.licenseKey) return
  await navigator.clipboard.writeText(props.licenseKey)
  copied.value = true
  if (copyTimeout) clearTimeout(copyTimeout)
  copyTimeout = setTimeout(() => {
    copied.value = false
    copyTimeout = null
  }, 2000)
}
</script>
