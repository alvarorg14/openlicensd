import type {
  CreateLicenseInput,
  CreatePolicyInput,
  CreateProductInput,
  CreateUserInput,
  License,
  LicenseListQueryParams,
  LicenseMachine,
  LicenseStats,
  ListQueryParams,
  MachineListQueryParams,
  Paginated,
  Policy,
  PolicyListQueryParams,
  Product,
  UpdateLicenseInput,
  User
} from '~/types'

const buildQuery = (params?: Record<string, string | number | undefined | null>) => {
  const searchParams = new URLSearchParams()
  if (!params) {
    return ''
  }
  for (const [key, value] of Object.entries(params)) {
    if (value === undefined || value === null || value === '') {
      continue
    }
    searchParams.set(key, String(value))
  }
  const query = searchParams.toString()
  return query ? `?${query}` : ''
}

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

  const listLicenses = (params?: LicenseListQueryParams) =>
    authFetch<Paginated<License>>(`/api/v1/licenses${buildQuery(params as Record<string, string | number | undefined | null>)}`)

  const getLicenseStats = () => authFetch<LicenseStats>('/api/v1/licenses/stats')

  const createLicense = (input: CreateLicenseInput) =>
    authFetch<License>('/api/v1/licenses', {
      method: 'POST',
      body: input
    })

  const updateLicense = (id: string, input: UpdateLicenseInput) =>
    authFetch<License>(`/api/v1/licenses/${id}`, {
      method: 'PATCH',
      body: input
    })

  const listLicenseMachines = (licenseId: string, params?: MachineListQueryParams) =>
    authFetch<Paginated<LicenseMachine>>(`/api/v1/licenses/${licenseId}/machines${buildQuery(params as Record<string, string | number | undefined | null>)}`)

  const updateLicenseMachine = (licenseId: string, machineId: string, name: string | null) =>
    authFetch<LicenseMachine>(`/api/v1/licenses/${licenseId}/machines/${machineId}`, {
      method: 'PATCH',
      body: { name }
    })

  const releaseLicenseMachine = (licenseId: string, machineId: string) =>
    authFetch<LicenseMachine>(`/api/v1/licenses/${licenseId}/machines/${machineId}`, {
      method: 'DELETE'
    })

  const revokeLicense = (id: string) =>
    authFetch<License>(`/api/v1/licenses/${id}/revoke`, {
      method: 'PATCH'
    })

  const unrevokeLicense = (id: string) =>
    authFetch<License>(`/api/v1/licenses/${id}/unrevoke`, {
      method: 'PATCH'
    })

  const deleteLicense = (id: string) =>
    authFetch(`/api/v1/licenses/${id}`, {
      method: 'DELETE'
    })

  const listProducts = (params?: ListQueryParams) =>
    authFetch<Paginated<Product>>(`/api/v1/products${buildQuery(params as Record<string, string | number | undefined | null>)}`)

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

  const listPolicies = (params?: PolicyListQueryParams) =>
    authFetch<Paginated<Policy>>(`/api/v1/policies${buildQuery(params as Record<string, string | number | undefined | null>)}`)

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

  const changeOwnPassword = (currentPassword: string, password: string) =>
    authFetch('/api/v1/auth/password', {
      method: 'POST',
      body: { current_password: currentPassword, password }
    })

  return {
    authFetch,
    listLicenses,
    getLicenseStats,
    createLicense,
    updateLicense,
    revokeLicense,
    unrevokeLicense,
    deleteLicense,
    listLicenseMachines,
    updateLicenseMachine,
    releaseLicenseMachine,
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
    deleteUser,
    changeOwnPassword
  }
}
