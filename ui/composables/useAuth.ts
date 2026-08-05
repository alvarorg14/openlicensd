import type { AuthProviders, AuthUser, LoginResponse, MeResponse } from '~/types'

const CSRF_COOKIE = 'openlicensd_csrf'

export const getCsrfToken = (): string | null => {
  if (!import.meta.client) {
    return null
  }
  const match = document.cookie.match(new RegExp(`(?:^|; )${CSRF_COOKIE}=([^;]*)`))
  return match ? decodeURIComponent(match[1]) : null
}

export const useAuth = () => {
  const user = useState<AuthUser | null>('auth_user', () => null)
  const authReady = useState('auth_ready', () => false)
  const providers = useState<AuthProviders | null>('auth_providers', () => null)
  const serverVersion = useState<string | null>('server_version', () => null)

  const isAuthenticated = computed(() => !!user.value)
  const isAdmin = computed(() => user.value?.role === 'admin')
  const canWrite = computed(() => user.value?.role === 'admin' || user.value?.role === 'operator')

  const setUser = (value: AuthUser | null) => {
    user.value = value
  }

  const fetchMe = async () => {
    try {
      const me = await $fetch<MeResponse>('/api/v1/auth/me', {
        credentials: 'include'
      })
      setUser(me)
      serverVersion.value = me.server_version ?? null
      return me
    } catch {
      setUser(null)
      serverVersion.value = null
      return null
    }
  }

  const fetchProviders = async () => {
    try {
      const res = await $fetch<AuthProviders>('/api/v1/auth/providers')
      providers.value = res
      return res
    } catch {
      providers.value = { local: true, oidc: false }
      return providers.value
    }
  }

  const login = async (email: string, password: string) => {
    await $fetch<LoginResponse>('/api/v1/auth/login', {
      method: 'POST',
      body: { email, password },
      credentials: 'include'
    })
    await fetchMe()
  }

  const logout = async () => {
    const csrf = getCsrfToken()
    try {
      await $fetch('/api/v1/auth/logout', {
        method: 'POST',
        credentials: 'include',
        headers: csrf ? { 'X-CSRF-Token': csrf } : {}
      })
    } catch {
      // ignore — clear local state regardless
    }
    setUser(null)
    serverVersion.value = null
    await navigateTo('/login')
  }

  return {
    user,
    authReady,
    providers,
    serverVersion,
    isAuthenticated,
    isAdmin,
    canWrite,
    setUser,
    fetchMe,
    fetchProviders,
    login,
    logout
  }
}
