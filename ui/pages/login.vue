<template>
  <div class="min-h-screen flex">
    <!-- Brand panel (desktop) -->
    <div
      class="hidden lg:flex lg:w-1/2 xl:w-[45%] relative overflow-hidden bg-gradient-to-br from-navy-900 via-navy-800 to-brand-900"
    >
      <div class="absolute inset-0 bg-[url('data:image/svg+xml,%3Csvg width=\'60\' height=\'60\' viewBox=\'0 0 60 60\' xmlns=\'http://www.w3.org/2000/svg\'%3E%3Cg fill=\'none\' fill-rule=\'evenodd\'%3E%3Cg fill=\'%23ffffff\' fill-opacity=\'0.05\'%3E%3Cpath d=\'M36 34v-4h-2v4h-4v2h4v4h2v-4h4v-2h-4zm0-30V0h-2v4h-4v2h4v4h2V6h4V4h-4zM6 34v-4H4v4H0v2h4v4h2v-4h4v-2H6zM6 4V0H4v4H0v2h4v4h2V6h4V4H6z\'/%3E%3C/g%3E%3C/g%3E%3C/svg%3E')] opacity-60" />
      <div class="relative z-10 flex flex-col justify-center px-12 xl:px-16 animate-fade-in-up">
        <div class="flex items-center gap-4 mb-10">
          <BrandMark class="h-20 w-auto text-white shrink-0" />
          <BrandWordmark class="h-14 w-auto text-white shrink-0 translate-x-1 translate-y-2.5" />
        </div>
        <h2 class="text-3xl xl:text-4xl font-medium tracking-brand text-white leading-tight mb-4">
          License management,<br>simplified.
        </h2>
        <p class="text-brand-200 text-lg max-w-md">
          Secure, self-hosted license server for your applications. Create, validate, and manage keys with ease.
        </p>
      </div>
    </div>

    <!-- Form panel -->
    <div class="flex-1 flex items-center justify-center p-6 sm:p-8 lg:p-12 bg-navy-50 dark:bg-navy-950">
      <div class="w-full max-w-md animate-fade-in-up stagger-1">
        <!-- Mobile logo -->
        <div class="lg:hidden flex items-center justify-center gap-3 mb-8">
          <BrandMark class="h-14 w-auto text-navy-900 dark:text-white shrink-0" />
          <BrandWordmark class="h-10 w-auto text-navy-900 dark:text-white shrink-0 translate-x-1 translate-y-2.5" />
        </div>

        <UCard class="shadow-app-lg border-0 ring-1 ring-default">
          <template #header>
            <div>
              <h1 class="text-xl font-medium tracking-brand text-highlighted">Welcome back</h1>
              <p class="text-sm text-muted mt-1">Sign in to manage your licenses</p>
            </div>
          </template>

          <div class="space-y-5">
            <UAlert
              v-if="error"
              color="error"
              variant="subtle"
              :title="error"
              class="animate-fade-in"
            />

            <UButton
              v-if="providers?.oidc && providers.oidc_login_url"
              block
              size="lg"
              color="neutral"
              variant="outline"
              icon="i-lucide-log-in"
              class="transition-app"
              @click="startSSO"
            >
              Sign in with {{ providers.oidc_name ?? 'SSO' }}
            </UButton>

            <div
              v-if="providers?.oidc && providers?.local"
              class="relative flex items-center py-1"
            >
              <div class="flex-grow border-t border-default" />
              <span class="mx-3 text-xs uppercase tracking-wide text-dimmed">or</span>
              <div class="flex-grow border-t border-default" />
            </div>

            <UForm
              v-if="providers?.local !== false"
              :state="form"
              class="space-y-5"
              @submit="onSubmit"
            >
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
          </div>
        </UCard>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
definePageMeta({
  layout: false
})

const { login, isAuthenticated, fetchProviders, providers } = useAuth()
const router = useRouter()
const route = useRoute()

const form = reactive({
  email: '',
  password: ''
})

const loading = ref(false)
const error = ref('')
const showPassword = ref(false)

onMounted(async () => {
  if (isAuthenticated.value) {
    router.replace('/licenses')
    return
  }

  await fetchProviders()

  if (route.query.error === 'sso_failed') {
    error.value = 'Single sign-on failed. Please try again or use your email and password.'
  }
})

const startSSO = () => {
  if (!providers.value?.oidc_login_url) {
    return
  }
  window.location.href = providers.value.oidc_login_url
}

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
