<template>
  <div class="min-h-screen flex items-center justify-center bg-gray-50 dark:bg-gray-900 p-4">
    <UCard class="w-full max-w-md">
      <template #header>
        <div class="text-center">
          <h1 class="text-2xl font-bold">openlicensd</h1>
          <p class="text-sm text-gray-500 mt-1">Sign in to manage licenses</p>
        </div>
      </template>

      <UForm :state="form" class="space-y-4" @submit="onSubmit">
        <UFormField label="Username" name="username" required>
          <UInput v-model="form.username" autocomplete="username" />
        </UFormField>

        <UFormField label="Password" name="password" required>
          <UInput v-model="form.password" type="password" autocomplete="current-password" />
        </UFormField>

        <UAlert v-if="error" color="error" variant="subtle" :title="error" />

        <UButton type="submit" block :loading="loading">
          Sign in
        </UButton>
      </UForm>
    </UCard>
  </div>
</template>

<script setup lang="ts">
definePageMeta({
  layout: false
})

const { login, isAuthenticated } = useAuth()
const router = useRouter()

const form = reactive({
  username: '',
  password: ''
})

const loading = ref(false)
const error = ref('')

onMounted(() => {
  if (isAuthenticated.value) {
    router.replace('/')
  }
})

const onSubmit = async () => {
  loading.value = true
  error.value = ''

  try {
    await login(form.username, form.password)
    await router.push('/')
  } catch {
    error.value = 'Invalid username or password'
  } finally {
    loading.value = false
  }
}
</script>
