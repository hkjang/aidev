const jsonHeaders = { 'Content-Type': 'application/json' }

export class ApiError extends Error {
  constructor(public status: number, message: string) { super(message) }
}

export async function api<T>(path: string, init?: RequestInit): Promise<T> {
  const defaultHeaders = init?.body instanceof FormData ? {} : jsonHeaders
  const response = await fetch(path, { credentials: 'same-origin', ...init, headers: { ...defaultHeaders, ...(init?.headers ?? {}) } })
  if (!response.ok) {
    const payload = await response.json().catch(() => null) as {error?:{message?:string}} | null
    throw new ApiError(response.status, payload?.error?.message ?? `요청 실패 (${response.status})`)
  }
  if (response.status === 204) return undefined as T
  return response.json() as Promise<T>
}

export function newIdempotencyKey() {
  return crypto.randomUUID?.() ?? `${Date.now()}-${Math.random().toString(36).slice(2)}`
}

export function address(row: number, column: number) {
  let n = column
  let letters = ''
  while (n > 0) { n -= 1; letters = String.fromCharCode(65 + (n % 26)) + letters; n = Math.floor(n / 26) }
  return `${letters}${row}`
}
