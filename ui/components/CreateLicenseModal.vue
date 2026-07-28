<template>
  <UModal v-model:open="open" title="Create license">
    <template #body>
      <UForm :state="form" class="space-y-4" @submit="onSubmit">
        <UFormField label="Label" name="label" required>
          <UInput v-model="form.label" placeholder="e.g. Acme Corp production" />
        </UFormField>

        <UFormField name="neverExpires">
          <UCheckbox v-model="form.neverExpires" label="Never expires" />
        </UFormField>

        <UFormField v-if="!form.neverExpires" label="Expiration date" name="expiresAt" required>
          <UInput v-model="form.expiresAt" type="datetime-local" />
        </UFormField>

        <UAlert v-if="error" color="error" variant="subtle" :title="error" />

        <div class="flex justify-end gap-2">
          <UButton color="neutral" variant="outline" @click="open = false">
            Cancel
          </UButton>
          <UButton type="submit" :loading="loading">
            Create
          </UButton>
        </div>
      </UForm>
    </template>
  </UModal>
</template>

<script setup lang="ts">
import type { License } from '~/types'

const open = defineModel<boolean>('open', { required: true })
const emit = defineEmits<{
  created: [license: License]
}>()

const { createLicense } = useAuth()

const form = reactive({
  label: '',
  neverExpires: true,
  expiresAt: ''
})

const loading = ref(false)
const error = ref('')

watch(open, (value) => {
  if (value) {
    form.label = ''
    form.neverExpires = true
    form.expiresAt = ''
    error.value = ''
  }
})

const onSubmit = async () => {
  if (!form.label.trim()) {
    error.value = 'Label is required'
    return
  }

  loading.value = true
  error.value = ''

  try {
    const expiresAt = form.neverExpires ? null : new Date(form.expiresAt).toISOString()
    const license = await createLicense(form.label.trim(), expiresAt)
    open.value = false
    emit('created', license)
  } catch {
    error.value = 'Failed to create license'
  } finally {
    loading.value = false
  }
}
</script>
