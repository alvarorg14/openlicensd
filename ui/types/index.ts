export type ExpirationBasis = 'on_creation' | 'on_first_validation'

export type UserRole = 'admin' | 'operator' | 'viewer'

export interface AuthUser {
  id: string
  email: string
  name: string
  role: UserRole
  auth_provider: string
  has_password: boolean
  picture_url?: string | null
}

export interface MeResponse extends AuthUser {
  server_version?: string
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
  created_at: string
  updated_at: string
}

export interface Policy {
  id: string
  product_id: string
  product_name: string
  name: string
  description: string | null
  duration_days: number | null
  expiration_basis: ExpirationBasis
  grace_period_days: number
  max_activations: number | null
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
  max_activations: number | null
  activation_count: number
  created_by?: string | null
  created_by_name?: string | null
  created_by_email?: string | null
}

export interface LoginResponse {
  user: AuthUser
}

export interface AuthProviders {
  local: boolean
  oidc: boolean
  oidc_name?: string
  oidc_login_url?: string
}

export interface ValidateResponse {
  valid: boolean
  expires_at?: string | null
  reason?: string
  product?: string
  policy?: string
  in_grace_period?: boolean
  activation_count?: number
  max_activations?: number | null
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
  max_activations?: number | null
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
  max_activations?: number | null
}

export interface UpdateLicenseInput {
  label: string
  expires_at: string | null
  max_activations?: number | null
}

export interface CreateUserInput {
  email: string
  name: string
  password: string
  role: UserRole
}

export interface ApiToken {
  id: string
  name: string
  token?: string
  token_prefix: string
  role: UserRole
  created_by?: string | null
  last_used_at?: string | null
  expires_at?: string | null
  revoked_at?: string | null
  created_at: string
  updated_at: string
}

export interface CreateApiTokenInput {
  name: string
  role: UserRole
  expires_at?: string
}

export interface AuditEvent {
  id: string
  occurred_at: string
  action: string
  resource_type: string
  resource_id?: string | null
  resource_label?: string | null
  actor_user_id?: string | null
  actor_token_id?: string | null
  actor_name: string
  actor_email?: string | null
  actor_role: string
  auth_method: string
  client_ip?: string | null
  user_agent?: string | null
  request_id?: string | null
  request_method: string
  request_path: string
  response_status: number
  metadata?: Record<string, unknown> | null
}

export interface AuditEventListQueryParams extends ListQueryParams {
  action?: string
  resource_type?: string
  actor_user_id?: string
  from?: string
  to?: string
}

export interface Paginated<T> {
  items: T[]
  page: number
  page_size: number
  total: number
  total_pages: number
}

export interface LicenseStats {
  total: number
  active: number
  expired: number
  revoked: number
}

export interface ListQueryParams {
  page?: number
  page_size?: number
  search?: string
  sort?: string
  order?: 'asc' | 'desc'
}

export interface LicenseListQueryParams extends ListQueryParams {
  status?: LicenseStatus
  product_id?: string
  policy_id?: string
}

export interface PolicyListQueryParams extends ListQueryParams {
  product_id?: string
}

export interface LicenseMachine {
  id: string
  license_id: string
  fingerprint: string
  name: string | null
  hostname: string | null
  display_name: string
  first_seen_at: string
  last_seen_at: string
  last_seen_ip: string | null
  validation_count: number
  deactivated_at: string | null
  deactivated_by: string | null
}

export interface MachineListQueryParams extends ListQueryParams {
  status?: 'active' | 'released'
}
