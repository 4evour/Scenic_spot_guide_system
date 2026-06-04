import { defineStore } from 'pinia'
import { ref, computed } from 'vue'

interface CachedAuth {
  valid: boolean
  role: string
  userId: string
  username: string
  checkedAt: number
}

const AUTH_CACHE_TTL = 5 * 60 * 1000

export const useAuthStore = defineStore('auth', () => {
  const cache = ref<CachedAuth | null>(null)

  const isAuthenticated = computed(() => cache.value?.valid ?? false)
  const isAdmin = computed(() => cache.value?.role === 'admin')
  const user = computed(() => cache.value ? {
    userId: cache.value.userId,
    username: cache.value.username,
    role: cache.value.role,
  } : null)

  async function fetchUser(): Promise<boolean> {
    if (cache.value && Date.now() - cache.value.checkedAt < AUTH_CACHE_TTL) {
      return cache.value.valid
    }
    try {
      const res = await fetch('/api/v1/user/me', { credentials: 'include' })
      if (!res.ok) {
        cache.value = { valid: false, role: '', userId: '', username: '', checkedAt: Date.now() }
        return false
      }
      const data = await res.json()
      const userData = data.data || data
      cache.value = {
        valid: true,
        role: userData.role || 'visitor',
        userId: userData.id || userData.ID || '',
        username: userData.username || '',
        checkedAt: Date.now(),
      }
      return true
    } catch {
      cache.value = { valid: false, role: '', userId: '', username: '', checkedAt: Date.now() }
      return false
    }
  }

  function invalidateAuth() {
    cache.value = null
  }

  async function logout() {
    try {
      await fetch('/api/v1/logout', { method: 'POST', credentials: 'include' })
    } catch { /* ignore */ }
    invalidateAuth()
  }

  return { cache, isAuthenticated, isAdmin, user, fetchUser, invalidateAuth, logout }
})
