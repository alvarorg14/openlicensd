import type {
  CreateLicenseInput,
  CreatePolicyInput,
  CreateProductInput,
  CreateUserInput,
  License,
  Policy,
  Product,
  User
} from '~/types'

export const useApi = () => {
  const { setUser } = useAuth()

  const authFetch = async <T>(url: string, options: Parameters<typeof $fetch<T>>[1] = {}) => {
    const csrf = getCsrfToken()
    const headers: Record<string, string> = {
      ...(options.headers as Record<string, string> | undefined)
    }
    if (csrf) {
      headers['X-CSRF-Token'] = csrf
    }

    try {
      return await $fetch<T>(url, {
        ...options,
        credentials: 'include',
        headers
      })
    } catch (err: unknown) {
      const status = (err as { statusCode?: number })?.statusCode
      if (status === 401) {
        setUser(null)
        await navigateTo('/login')
      }
      throw err
    }
  }

  const listLicenses = () => authFetch<License[]>('/api/v1/licenses')

  const createLicense = (input: CreateLicenseInput) =>
    authFetch<License>('/api/v1/licenses', {
      method: 'POST',
      body: input
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

  const listProducts = () => authFetch<Product[]>('/api/v1/products')

  const createProduct = (input: CreateProductInput) =>
    authFetch<Product>('/api/v1/products', {
      method: 'POST',
      body: input
    })

  const updateProduct = (id: string, input: CreateProductInput) =>
    authFetch<Product>(`/api/v1/products/${id}`, {
      method: 'PATCH',
      body: input
    })

  const deleteProduct = (id: string) =>
    authFetch(`/api/v1/products/${id}`, {
      method: 'DELETE'
    })

  const listPolicies = (productId?: string) => {
    const query = productId ? `?product_id=${productId}` : ''
    return authFetch<Policy[]>(`/api/v1/policies${query}`)
  }

  const createPolicy = (input: CreatePolicyInput) =>
    authFetch<Policy>('/api/v1/policies', {
      method: 'POST',
      body: input
    })

  const updatePolicy = (id: string, input: Omit<CreatePolicyInput, 'product_id'>) =>
    authFetch<Policy>(`/api/v1/policies/${id}`, {
      method: 'PATCH',
      body: input
    })

  const deletePolicy = (id: string) =>
    authFetch(`/api/v1/policies/${id}`, {
      method: 'DELETE'
    })

  const listUsers = () => authFetch<User[]>('/api/v1/users')

  const createUser = (input: CreateUserInput) =>
    authFetch<User>('/api/v1/users', {
      method: 'POST',
      body: input
    })

  const updateUser = (id: string, input: Omit<CreateUserInput, 'password'>) =>
    authFetch<User>(`/api/v1/users/${id}`, {
      method: 'PATCH',
      body: input
    })

  const setUserPassword = (id: string, password: string) =>
    authFetch(`/api/v1/users/${id}/password`, {
      method: 'PATCH',
      body: { password }
    })

  const disableUser = (id: string) =>
    authFetch<User>(`/api/v1/users/${id}/disable`, {
      method: 'PATCH'
    })

  const enableUser = (id: string) =>
    authFetch<User>(`/api/v1/users/${id}/enable`, {
      method: 'PATCH'
    })

  const deleteUser = (id: string) =>
    authFetch(`/api/v1/users/${id}`, {
      method: 'DELETE'
    })

  return {
    authFetch,
    listLicenses,
    createLicense,
    updateLicense,
    revokeLicense,
    activateLicense,
    deleteLicense,
    listProducts,
    createProduct,
    updateProduct,
    deleteProduct,
    listPolicies,
    createPolicy,
    updatePolicy,
    deletePolicy,
    listUsers,
    createUser,
    updateUser,
    setUserPassword,
    disableUser,
    enableUser,
    deleteUser
  }
}
