const storageKey = 'kanpic-collaboration-client-id'

let cached = ''

export function collaborationClientId() {
  if (cached) return cached
  cached = sessionStorage.getItem(storageKey) ?? ''
  if (!cached) {
    cached = crypto.randomUUID?.() ?? `${Date.now()}-${Math.random().toString(36).slice(2)}`
    sessionStorage.setItem(storageKey, cached)
  }
  return cached
}
