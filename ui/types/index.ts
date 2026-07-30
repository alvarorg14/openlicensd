export type ExpirationBasis = 'on_creation' | 'on_first_validation'

export type UserRole = 'admin' | 'operator' | 'viewer'

export interface AuthUser {
  id: string
  email: string
  name: string
  role: UserRole
  auth_provider: string
}

export interface User {
  id: string
  email: string
  name: string
  role: UserRole
  auth_provider: string
  disabled_at?: string | null
  last_login_at?: string | null
  created_at: string
  updated_at: string
}

export interface Product {
  id: string
  name: string
  code: string
  description: string | null
  archived_at?: string | null
  created_at: string
  updated_at: string
}

export interface Policy {
  id: string
  product_id: string
  name: string
  description: string | null
  duration_days: number | null
  expiration_basis: ExpirationBasis
  grace_period_days: number
  archived_at?: string | null
  created_at: string
  updated_at: string
}

export interface License {
  id: string
  label: string
  key?: string
  key_prefix: string
  product_id: string
  product_code: string
  product_name: string
  policy_id: string
  policy_name: string
  expires_at: string | null
  activated_at: string | null
  revoked: boolean
  created_at: string
  last_validated_at: string | null
  validation_count: number
  created_by?: string | null
  created_by_name?: string | null
  created_by_email?: string | null
}

export interface LoginResponse {
  user: AuthUser
}

export interface ValidateResponse {
  valid: boolean
  expires_at?: string | null
  reason?: string
  product?: string
  policy?: string
  in_grace_period?: boolean
}

export type LicenseStatus = 'active' | 'expired' | 'revoked'

export interface DetailItem {
  label: string
  value: string
  mono?: boolean
  multiline?: boolean
}

export interface CreateLicenseInput {
  label: string
  product_id: string
  policy_id: string
  expires_at?: string | null
}

export interface CreateProductInput {
  name: string
  code: string
  description?: string | null
}

export interface CreatePolicyInput {
  product_id: string
  name: string
  description?: string | null
  duration_days?: number | null
  expiration_basis?: ExpirationBasis
  grace_period_days?: number
}

export interface CreateUserInput {
  email: string
  name: string
  password: string
  role: UserRole
}
