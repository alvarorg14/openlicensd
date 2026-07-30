<template>
  <div class="min-h-screen flex">
    <!-- Brand panel (desktop) -->
    <div
      class="hidden lg:flex lg:w-1/2 xl:w-[45%] relative overflow-hidden bg-gradient-to-br from-indigo-600 via-indigo-700 to-indigo-900"
    >
      <div class="absolute inset-0 bg-[url('data:image/svg+xml,%3Csvg width=\'60\' height=\'60\' viewBox=\'0 0 60 60\' xmlns=\'http://www.w3.org/2000/svg\'%3E%3Cg fill=\'none\' fill-rule=\'evenodd\'%3E%3Cg fill=\'%23ffffff\' fill-opacity=\'0.05\'%3E%3Cpath d=\'M36 34v-4h-2v4h-4v2h4v4h2v-4h4v-2h-4zm0-30V0h-2v4h-4v2h4v4h2V6h4V4h-4zM6 34v-4H4v4H0v2h4v4h2v-4h4v-2H6zM6 4V0H4v4H0v2h4v4h2V6h4V4H6z\'/%3E%3C/g%3E%3C/g%3E%3C/svg%3E')] opacity-60" />
      <div class="relative z-10 flex flex-col justify-center px-12 xl:px-16 animate-fade-in-up">
        <div class="flex items-center gap-3 mb-8">
          <div class="flex h-12 w-12 items-center justify-center rounded-xl bg-white/20 backdrop-blur-sm">
            <UIcon name="i-lucide-key-round" class="h-6 w-6 text-white" />
          </div>
          <span class="text-2xl font-bold text-white tracking-tight">{{ APP_NAME }}</span>
        </div>
        <h2 class="text-3xl xl:text-4xl font-bold text-white leading-tight mb-4">
          License management,<br>simplified.
        </h2>
        <p class="text-indigo-100 text-lg max-w-md">
          Secure, self-hosted license server for your applications. Create, validate, and manage keys with ease.
        </p>
      </div>
    </div>

    <!-- Form panel -->
    <div class="flex-1 flex items-center justify-center p-6 sm:p-8 lg:p-12 bg-slate-50 dark:bg-slate-950">
      <div class="w-full max-w-md animate-fade-in-up stagger-1">
        <!-- Mobile logo -->
        <div class="lg:hidden flex items-center justify-center gap-2 mb-8">
          <div class="flex h-10 w-10 items-center justify-center rounded-lg bg-indigo-600">
            <UIcon name="i-lucide-key-round" class="h-5 w-5 text-white" />
          </div>
          <span class="text-xl font-bold text-slate-900 dark:text-white">{{ APP_NAME }}</span>
        </div>

        <UCard class="shadow-app-lg border-0 ring-1 ring-slate-200/80 dark:ring-slate-800/80">
          <template #header>
            <div>
              <h1 class="text-xl font-semibold text-slate-900 dark:text-white">Welcome back</h1>
              <p class="text-sm text-slate-500 dark:text-slate-400 mt-1">Sign in to manage your licenses</p>
            </div>
          </template>

          <UForm :state="form" class="space-y-5" @submit="onSubmit">
            <UFormField label="Email" name="email" required>
              <UInput
                v-model="form.email"
                type="email"
                autocomplete="email"
                placeholder="Enter your email"
                size="lg"
                :disabled="loading"
                class="transition-app"
              />
            </UFormField>

            <UFormField label="Password" name="password" required>
              <UInput
                v-model="form.password"
                :type="showPassword ? 'text' : 'password'"
                autocomplete="current-password"
                placeholder="Enter your password"
                size="lg"
                :disabled="loading"
                class="transition-app"
              >
                <template #trailing>
                  <UButton
                    type="button"
                    color="neutral"
                    variant="ghost"
                    :icon="showPassword ? 'i-lucide-eye-off' : 'i-lucide-eye'"
                    :padded="false"
                    size="xs"
                    @click="showPassword = !showPassword"
                  />
                </template>
              </UInput>
            </UFormField>

            <UAlert
              v-if="error"
              color="error"
              variant="subtle"
              :title="error"
              class="animate-fade-in"
            />

            <UButton
              type="submit"
              block
              size="lg"
              :loading="loading"
              :disabled="!form.email.trim() || !form.password"
              class="transition-app"
            >
              Sign in
            </UButton>
          </UForm>
        </UCard>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { APP_NAME } from '~/constants/app'

definePageMeta({
  layout: false
})

const { login, isAuthenticated } = useAuth()
const router = useRouter()

const form = reactive({
  email: '',
  password: ''
})

const loading = ref(false)
const error = ref('')
const showPassword = ref(false)

onMounted(() => {
  if (isAuthenticated.value) {
    router.replace('/licenses')
  }
})

const onSubmit = async () => {
  loading.value = true
  error.value = ''

  try {
    await login(form.email, form.password)
    await router.push('/licenses')
  } catch {
    error.value = 'Invalid email or password'
  } finally {
    loading.value = false
  }
}
</script>
