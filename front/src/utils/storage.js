function getStorage() {
  try {
    return window.localStorage
  } catch {
    return null
  }
}

export function storageGet(key, fallback = null) {
  const storage = getStorage()
  if (!storage) return fallback
  try {
    const value = storage.getItem(key)
    return value == null ? fallback : value
  } catch {
    return fallback
  }
}

export function storageSet(key, value) {
  const storage = getStorage()
  if (!storage) return false
  try {
    storage.setItem(key, String(value))
    return true
  } catch {
    return false
  }
}

export function storageRemove(key) {
  const storage = getStorage()
  if (!storage) return false
  try {
    storage.removeItem(key)
    return true
  } catch {
    return false
  }
}

export function storageGetJSON(key, fallback = null) {
  const raw = storageGet(key, null)
  if (raw == null) return fallback
  try {
    return JSON.parse(raw)
  } catch {
    return fallback
  }
}

export function storageSetJSON(key, value) {
  const storage = getStorage()
  if (!storage) return false
  try {
    storage.setItem(key, JSON.stringify(value))
    return true
  } catch {
    return false
  }
}
