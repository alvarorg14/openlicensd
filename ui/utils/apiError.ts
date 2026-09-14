type ApiErrorBody = {
  error?: string
}

type FetchLikeError = {
  data?: ApiErrorBody
  statusMessage?: string
}

export const getApiErrorMessage = (err: unknown, fallback: string): string => {
  if (err && typeof err === 'object') {
    const fetchErr = err as FetchLikeError
    if (fetchErr.data?.error) {
      return fetchErr.data.error
    }
    if (fetchErr.statusMessage) {
      return fetchErr.statusMessage
    }
  }
  return fallback
}
