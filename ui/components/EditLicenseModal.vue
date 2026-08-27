<template>
  <UModal v-model:open="open" title="Edit license">
    <template #header>
      <div class="flex items-center gap-2">
        <div class="flex h-8 w-8 items-center justify-center rounded-lg bg-brand-100 dark:bg-brand-900/40">
          <UIcon name="i-lucide-pencil" class="h-4 w-4 text-brand-600 dark:text-brand-400" />
        </div>
        <span class="font-semibold">Edit license</span>
      </div>
    </template>
    <template #body>
      <UForm :state="form" class="space-y-4" @submit="onSubmit">
        <UFormField label="Label" name="label" required>
          <UInput v-model="form.label" placeholder="e.g. Acme Corp production" />
        </UFormField>

        <div v-if="license" class="grid grid-cols-1 sm:grid-cols-2 gap-3">
          <UFormField label="Product">
            <UInput :model-value="license.product_name" disabled />
          </UFormField>
          <UFormField label="Policy">
            <UInput :model-value="license.policy_name" disabled />
          </UFormField>
        </div>

        <UFormField name="neverExpires">
          <UCheckbox v-model="form.neverExpires" label="Never expires" />
        </UFormField>

        <UFormField v-if="!form.neverExpires" label="Expiration date" name="expiresAt" required>
          <UInput v-model="form.expiresAt" type="datetime-local" />
        </UFormField>

        <UFormField name="unlimitedActivations">
          <UCheckbox v-model="form.unlimitedActivations" label="Unlimited machine activations" />
        </UFormField>

        <UFormField v-if="!form.unlimitedActivations" label="Max activations" name="maxActivations" required>
          <UInput v-model.number="form.maxActivations" type="number" min="1" />
        </UFormField>

        <UAlert v-if="error" color="error" variant="subtle" :title="error" class="animate-fade-in" />

        <div class="flex justify-end gap-2 pt-2">
          <UButton color="neutral" variant="outline" @click="close">
            Cancel
          </UButton>
          <UButton type="submit" :loading="loading">
            Save
          </UButton>
        </div>
      </UForm>
    </template>
  </UModal>
</template>

<script setup lang="ts">
import type { License } from '~/types'

const props = defineProps<{
  license: License | null
}>()

const open = defineModel<boolean>('open', { required: true })
const emit = defineEmits<{
  updated: [license: License]
}>()

const { updateLicense } = useApi()

const form = reactive({
  label: '',
  neverExpires: true,
  expiresAt: '',
  unlimitedActivations: true,
  maxActivations: 3
})

const loading = ref(false)
const error = ref('')

const close = () => {
  open.value = false
}

const toDatetimeLocal = (value: string) => {
  const date = new Date(value)
  const offset = date.getTimezoneOffset()
  const local = new Date(date.getTime() - offset * 60_000)
  return local.toISOString().slice(0, 16)
}

watch(open, (value) => {
  if (value && props.license) {
    form.label = props.license.label
    form.neverExpires = !props.license.expires_at
    form.expiresAt = props.license.expires_at ? toDatetimeLocal(props.license.expires_at) : ''
    form.unlimitedActivations = props.license.max_activations == null
    form.maxActivations = props.license.max_activations ?? 3
    error.value = ''
  }
})

const onSubmit = async () => {
  if (!props.license) {
    return
  }

  if (!form.label.trim()) {
    error.value = 'Label is required'
    return
  }

  if (!form.neverExpires && !form.expiresAt) {
    error.value = 'Expiration date is required'
    return
  }

  if (!form.unlimitedActivations && (!form.maxActivations || form.maxActivations < 1)) {
    error.value = 'Max activations must be at least 1'
    return
  }

  loading.value = true
  error.value = ''

  try {
    const expiresAt = form.neverExpires ? null : new Date(form.expiresAt).toISOString()
    const maxActivations = form.unlimitedActivations ? null : form.maxActivations
    const license = await updateLicense(props.license.id, {
      label: form.label.trim(),
      expires_at: expiresAt,
      max_activations: maxActivations
    })
    open.value = false
    emit('updated', license)
  } catch {
    error.value = 'Failed to update license'
  } finally {
    loading.value = false
  }
}
</script>
