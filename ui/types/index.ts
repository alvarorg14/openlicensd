export interface License {
  id: string
  label: string
  key?: string
  key_prefix: string
  expires_at: string | null
  revoked: boolean
  created_at: string
  last_validated_at: string | null
  validation_count: number
}

export interface LoginResponse {
  token: string
}

export interface ValidateResponse {
  valid: boolean
  expires_at?: string | null
  reason?: string
}

export type LicenseStatus = 'active' | 'expired' | 'revoked'
