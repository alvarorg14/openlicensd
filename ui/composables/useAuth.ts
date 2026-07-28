import type { License, LoginResponse } from '~/types'

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

  const authFetch = async <T>(url: string, options: Parameters<typeof $fetch<T>>[1] = {}) => {
    if (!token.value) {
      throw new Error('Not authenticated')
    }

    return $fetch<T>(url, {
      ...options,
      headers: {
        ...(options.headers || {}),
        Authorization: `Bearer ${token.value}`
      }
    })
  }

  const listLicenses = () => authFetch<License[]>('/api/v1/licenses')

  const createLicense = (label: string, expiresAt: string | null) =>
    authFetch<License>('/api/v1/licenses', {
      method: 'POST',
      body: {
        label,
        expires_at: expiresAt
      }
    })

  const updateLicense = (id: string, label: string, expiresAt: string | null) =>
    authFetch<License>(`/api/v1/licenses/${id}`, {
      method: 'PATCH',
      body: {
        label,
        expires_at: expiresAt
      }
    })

  const revokeLicense = (id: string) =>
    authFetch<License>(`/api/v1/licenses/${id}/revoke`, {
      method: 'PATCH'
    })

  const activateLicense = (id: string) =>
    authFetch<License>(`/api/v1/licenses/${id}/activate`, {
      method: 'PATCH'
    })

  const deleteLicense = (id: string) =>
    authFetch(`/api/v1/licenses/${id}`, {
      method: 'DELETE'
    })

  return {
    token,
    isAuthenticated,
    login,
    logout,
    listLicenses,
    createLicense,
    updateLicense,
    revokeLicense,
    activateLicense,
    deleteLicense
  }
}
