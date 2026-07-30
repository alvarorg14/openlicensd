import type {
  CreateLicenseInput,
  CreatePolicyInput,
  CreateProductInput,
  License,
  Policy,
  Product
} from '~/types'

export const useApi = () => {
  const { token } = useAuth()

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
    deletePolicy
  }
}
