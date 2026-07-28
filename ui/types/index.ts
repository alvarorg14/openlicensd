export interface License {
  id: string
  label: string
  key?: string
  key_prefix: string
  expires_at: string | null
  revoked: boolean
  created_at: string
}

export interface LoginResponse {
  token: string
}

export interface ValidateResponse {
  valid: boolean
  expires_at?: string | null
  reason?: string
}
