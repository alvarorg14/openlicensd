<template>
  <UModal v-model:open="open" title="Change password">
    <template #header>
      <div class="flex items-center gap-2">
        <div class="flex h-8 w-8 items-center justify-center rounded-lg bg-indigo-100 dark:bg-indigo-900/40">
          <UIcon name="i-lucide-key-round" class="h-4 w-4 text-indigo-600 dark:text-indigo-400" />
        </div>
        <span class="font-semibold">Change password</span>
      </div>
    </template>
    <template #body>
      <div v-if="success" class="space-y-4">
        <UAlert
          color="success"
          variant="subtle"
          title="Password updated"
          description="Other sessions have been signed out."
          class="animate-fade-in"
        />
        <div class="flex justify-end pt-2">
          <UButton @click="open = false">
            Close
          </UButton>
        </div>
      </div>
      <UForm v-else :state="form" class="space-y-4" @submit="onSubmit">
        <UFormField label="Current password" name="currentPassword" required>
          <UInput
            v-model="form.currentPassword"
            type="password"
            autocomplete="current-password"
          />
        </UFormField>

        <UFormField label="New password" name="password" required>
          <UInput
            v-model="form.password"
            type="password"
            autocomplete="new-password"
          />
        </UFormField>

        <UFormField label="Confirm new password" name="confirmPassword" required>
          <UInput
            v-model="form.confirmPassword"
            type="password"
            autocomplete="new-password"
          />
        </UFormField>

        <UAlert v-if="error" color="error" variant="subtle" :title="error" class="animate-fade-in" />

        <div class="flex justify-end gap-2 pt-2">
          <UButton color="neutral" variant="outline" @click="open = false">
            Cancel
          </UButton>
          <UButton type="submit" :loading="loading">
            Update password
          </UButton>
        </div>
      </UForm>
    </template>
  </UModal>
</template>

<script setup lang="ts">
const open = defineModel<boolean>('open', { required: true })

const { changeOwnPassword } = useApi()

const form = reactive({
  currentPassword: '',
  password: '',
  confirmPassword: ''
})

const loading = ref(false)
const error = ref('')
const success = ref(false)

watch(open, (value) => {
  if (!value) {
    return
  }
  form.currentPassword = ''
  form.password = ''
  form.confirmPassword = ''
  error.value = ''
  success.value = false
})

const onSubmit = async () => {
  if (!form.currentPassword) {
    error.value = 'Current password is required'
    return
  }
  if (!form.password) {
    error.value = 'New password is required'
    return
  }
  if (form.password.length < 8) {
    error.value = 'New password must be at least 8 characters'
    return
  }
  if (form.password !== form.confirmPassword) {
    error.value = 'Passwords do not match'
    return
  }

  loading.value = true
  error.value = ''

  try {
    await changeOwnPassword(form.currentPassword, form.password)
    success.value = true
  } catch (err: unknown) {
    const message = (err as { data?: { error?: string } })?.data?.error
    error.value = message || 'Failed to change password'
  } finally {
    loading.value = false
  }
}
</script>
