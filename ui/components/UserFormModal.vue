<template>
  <UModal v-model:open="open" :title="user ? 'Edit user' : 'Create user'">
    <template #header>
      <div class="flex items-center gap-2">
        <div class="flex h-8 w-8 items-center justify-center rounded-lg bg-indigo-100 dark:bg-indigo-900/40">
          <UIcon :name="user ? 'i-lucide-pencil' : 'i-lucide-plus'" class="h-4 w-4 text-indigo-600 dark:text-indigo-400" />
        </div>
        <span class="font-semibold">{{ user ? 'Edit user' : 'Create user' }}</span>
      </div>
    </template>
    <template #body>
      <UForm :state="form" class="space-y-4" @submit="onSubmit">
        <UFormField label="Name" name="name" required>
          <UInput v-model="form.name" placeholder="e.g. Jane Smith" />
        </UFormField>

        <UFormField label="Email" name="email" required>
          <UInput v-model="form.email" type="email" placeholder="e.g. jane@example.com" />
        </UFormField>

        <UFormField v-if="!user" label="Password" name="password" required>
          <UInput v-model="form.password" type="password" autocomplete="new-password" />
        </UFormField>

        <UFormField label="Role" name="role" required>
          <USelectMenu
            v-model="form.role"
            :items="roleOptions"
            value-key="value"
            label-key="label"
          />
        </UFormField>

        <UAlert v-if="error" color="error" variant="subtle" :title="error" class="animate-fade-in" />

        <div class="flex justify-end gap-2 pt-2">
          <UButton color="neutral" variant="outline" @click="open = false">
            Cancel
          </UButton>
          <UButton type="submit" :loading="loading">
            {{ user ? 'Save' : 'Create' }}
          </UButton>
        </div>
      </UForm>
    </template>
  </UModal>
</template>

<script setup lang="ts">
import type { User, UserRole } from '~/types'

const props = defineProps<{
  user: User | null
}>()

const open = defineModel<boolean>('open', { required: true })
const emit = defineEmits<{
  saved: []
}>()

const { createUser, updateUser } = useApi()

const roleOptions = [
  { label: 'Admin', value: 'admin' as UserRole },
  { label: 'Operator', value: 'operator' as UserRole },
  { label: 'Viewer', value: 'viewer' as UserRole }
]

const form = reactive({
  name: '',
  email: '',
  password: '',
  role: 'viewer' as UserRole
})

const loading = ref(false)
const error = ref('')

watch(open, (value) => {
  if (!value) {
    return
  }
  if (props.user) {
    form.name = props.user.name
    form.email = props.user.email
    form.password = ''
    form.role = props.user.role
  } else {
    form.name = ''
    form.email = ''
    form.password = ''
    form.role = 'viewer'
  }
  error.value = ''
})

const onSubmit = async () => {
  if (!form.name.trim()) {
    error.value = 'Name is required'
    return
  }
  if (!form.email.trim()) {
    error.value = 'Email is required'
    return
  }
  if (!props.user && !form.password) {
    error.value = 'Password is required'
    return
  }

  loading.value = true
  error.value = ''

  try {
    if (props.user) {
      await updateUser(props.user.id, {
        name: form.name.trim(),
        email: form.email.trim(),
        role: form.role
      })
    } else {
      await createUser({
        name: form.name.trim(),
        email: form.email.trim(),
        password: form.password,
        role: form.role
      })
    }
    open.value = false
    emit('saved')
  } catch {
    error.value = props.user ? 'Failed to update user' : 'Failed to create user'
  } finally {
    loading.value = false
  }
}
</script>
