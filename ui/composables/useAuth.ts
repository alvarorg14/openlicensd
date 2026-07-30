import type { LoginResponse } from '~/types'

const TOKEN_KEY = 'openlicensd_token'

export const useAuth = () => {
  const token = useState<string | null>('auth_token', () => {
    if (import.meta.client) {
      return localStorage.getItem(TOKEN_KEY)
    }
    return null
  })

  const isAuthenticated = computed(() => !!token.value)

  const setToken = (value: string) => {
    token.value = value
    if (import.meta.client) {
      localStorage.setItem(TOKEN_KEY, value)
    }
  }

  const clearToken = () => {
    token.value = null
    if (import.meta.client) {
      localStorage.removeItem(TOKEN_KEY)
    }
  }

  const login = async (username: string, password: string) => {
    const res = await $fetch<LoginResponse>('/api/v1/auth/login', {
      method: 'POST',
      body: { username, password }
    })
    setToken(res.token)
  }

  const logout = () => {
    clearToken()
    navigateTo('/login')
  }

  return {
    token,
    isAuthenticated,
    login,
    logout
  }
}
