<template>
  <UModal v-model:open="open" title="Create API token">
    <template #header>
      <div class="flex items-center gap-2">
        <div class="flex h-8 w-8 items-center justify-center rounded-lg bg-brand-100 dark:bg-brand-900/40">
          <UIcon name="i-lucide-plus" class="h-4 w-4 text-brand-600 dark:text-brand-400" />
        </div>
        <span class="font-semibold">Create API token</span>
      </div>
    </template>
    <template #body>
      <UForm :state="form" class="space-y-4" @submit="onSubmit">
        <UFormField label="Name" name="name" required>
          <UInput v-model="form.name" placeholder="e.g. Terraform CI" />
        </UFormField>

        <UFormField label="Role" name="role" required>
          <USelectMenu
            v-model="form.role"
            :items="roleOptions"
            value-key="value"
            label-key="label"
          />
        </UFormField>

        <UFormField label="Expires at" name="expires_at" hint="Optional. Leave empty for no expiration.">
          <UInput v-model="form.expiresAt" type="datetime-local" />
        </UFormField>

        <UAlert v-if="error" color="error" variant="subtle" :title="error" class="animate-fade-in" />

        <div class="flex justify-end gap-2 pt-2">
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
import type { ApiToken, CreateApiTokenInput, UserRole } from '~/types'

const open = defineModel<boolean>('open', { required: true })
const emit = defineEmits<{
  created: [token: ApiToken]
}>()

const { createApiToken } = useApi()

const roleOptions = [
  { label: 'Admin', value: 'admin' as UserRole },
  { label: 'Operator', value: 'operator' as UserRole },
  { label: 'Viewer', value: 'viewer' as UserRole }
]

const form = reactive({
  name: '',
  role: 'operator' as UserRole,
  expiresAt: ''
})

const loading = ref(false)
const error = ref('')

watch(open, (value) => {
  if (!value) {
    return
  }
  form.name = ''
  form.role = 'operator'
  form.expiresAt = ''
  error.value = ''
})

const onSubmit = async () => {
  if (!form.name.trim()) {
    error.value = 'Name is required'
    return
  }

  loading.value = true
  error.value = ''

  const body: CreateApiTokenInput = {
    name: form.name.trim(),
    role: form.role
  }

  if (form.expiresAt) {
    const parsed = new Date(form.expiresAt)
    if (Number.isNaN(parsed.getTime())) {
      error.value = 'Expires at must be a valid date and time'
      loading.value = false
      return
    }
    if (parsed <= new Date()) {
      error.value = 'Expires at must be in the future'
      loading.value = false
      return
    }
    body.expires_at = parsed.toISOString()
  }

  try {
    const token = await createApiToken(body)
    open.value = false
    emit('created', token)
  } catch {
    error.value = 'Failed to create API token'
  } finally {
    loading.value = false
  }
}
</script>
