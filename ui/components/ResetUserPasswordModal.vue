<template>
  <UModal v-model:open="open" :title="modalTitle">
    <template #header>
      <div class="flex items-center gap-2">
        <div class="flex h-8 w-8 items-center justify-center rounded-lg bg-brand-100 dark:bg-brand-900/40">
          <UIcon name="i-lucide-key-round" class="h-4 w-4 text-brand-600 dark:text-brand-400" />
        </div>
        <span class="font-semibold">Reset password</span>
      </div>
    </template>
    <template #body>
      <div v-if="success" class="space-y-4">
        <UAlert
          color="success"
          variant="subtle"
          title="Password updated"
          class="animate-fade-in"
        />
        <div class="flex justify-end pt-2">
          <UButton @click="open = false">
            Close
          </UButton>
        </div>
      </div>
      <UForm v-else :state="form" class="space-y-4" @submit="onSubmit">
        <p v-if="user" class="text-sm text-muted">
          Set a new password for <span class="font-medium text-highlighted">{{ user.name }}</span>.
        </p>

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
            Reset password
          </UButton>
        </div>
      </UForm>
    </template>
  </UModal>
</template>

<script setup lang="ts">
import type { User } from '~/types'

const props = defineProps<{
  user: User | null
}>()

const open = defineModel<boolean>('open', { required: true })

const { setUserPassword } = useApi()

const form = reactive({
  password: '',
  confirmPassword: ''
})

const loading = ref(false)
const error = ref('')
const success = ref(false)

const modalTitle = computed(() => {
  if (!props.user) {
    return 'Reset password'
  }
  return `Reset password for ${props.user.name}`
})

watch(open, (value) => {
  if (!value) {
    return
  }
  form.password = ''
  form.confirmPassword = ''
  error.value = ''
  success.value = false
})

const onSubmit = async () => {
  if (!props.user) {
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
    await setUserPassword(props.user.id, form.password)
    success.value = true
  } catch (err: unknown) {
    const message = (err as { data?: { error?: string } })?.data?.error
    error.value = message || 'Failed to reset password'
  } finally {
    loading.value = false
  }
}
</script>
