import type { AuthUser, LoginResponse } from '~/types'

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

  const isAuthenticated = computed(() => !!user.value)
  const isAdmin = computed(() => user.value?.role === 'admin')
  const canWrite = computed(() => user.value?.role === 'admin' || user.value?.role === 'operator')

  const setUser = (value: AuthUser | null) => {
    user.value = value
  }

  const fetchMe = async () => {
    try {
      const me = await $fetch<AuthUser>('/api/v1/auth/me', {
        credentials: 'include'
      })
      setUser(me)
      return me
    } catch {
      setUser(null)
      return null
    }
  }

  const login = async (email: string, password: string) => {
    const res = await $fetch<LoginResponse>('/api/v1/auth/login', {
      method: 'POST',
      body: { email, password },
      credentials: 'include'
    })
    setUser(res.user)
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
    await navigateTo('/login')
  }

  return {
    user,
    authReady,
    isAuthenticated,
    isAdmin,
    canWrite,
    setUser,
    fetchMe,
    login,
    logout
  }
}
