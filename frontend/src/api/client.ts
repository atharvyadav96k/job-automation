export const API_BASE = import.meta.env.VITE_API_BASE_URL ?? 'http://localhost:8080'
const CREDS_KEY = 'jobauto.basicAuth'

export function saveCredentials(user: string, pass: string) {
  localStorage.setItem(CREDS_KEY, btoa(`${user}:${pass}`))
}

export function clearCredentials() {
  localStorage.removeItem(CREDS_KEY)
}

export function hasCredentials(): boolean {
  return localStorage.getItem(CREDS_KEY) !== null
}

export function authHeader(): string | null {
  const encoded = localStorage.getItem(CREDS_KEY)
  return encoded ? `Basic ${encoded}` : null
}

export class ApiError extends Error {
  status: number
  constructor(status: number, message: string) {
    super(message)
    this.status = status
  }
}

export async function apiFetch<T>(path: string, init: RequestInit = {}): Promise<T> {
  const headers = new Headers(init.headers)
  const auth = authHeader()
  if (auth) headers.set('Authorization', auth)
  if (init.body && !(init.body instanceof FormData) && !headers.has('Content-Type')) {
    headers.set('Content-Type', 'application/json')
  }

  const res = await fetch(`${API_BASE}${path}`, { ...init, headers })
  if (res.status === 401) {
    clearCredentials()
    throw new ApiError(401, 'unauthorized')
  }
  if (!res.ok) {
    const body = await res.text()
    throw new ApiError(res.status, body || res.statusText)
  }
  if (res.status === 204) return undefined as T
  return (await res.json()) as T
}
