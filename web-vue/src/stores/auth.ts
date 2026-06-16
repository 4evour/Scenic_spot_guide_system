import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { generateDeviceFingerprint } from '../utils/fingerprint'
import { getCSRFToken } from '../utils/csrf'

const API_TIMEOUT_MS = Number(import.meta.env.VITE_API_TIMEOUT_MS) || 15000;

interface CachedAuth {
  valid: boolean
  role: string
  userId: string
  username: string
  displayName: string
  checkedAt: number
}

const AUTH_CACHE_TTL = 5 * 60 * 1000

export const useAuthStore = defineStore('auth', () => {
  const cache = ref<CachedAuth | null>(null)
  const isUpgrading = ref(false)

  const isAuthenticated = computed(() => cache.value?.valid ?? false)
  const isAdmin = computed(() => cache.value?.role === 'admin')
  const isGuest = computed(() => cache.value?.role === 'guest')
  const displayName = computed(() => cache.value?.displayName || cache.value?.username || '')
  const user = computed(() => cache.value ? {
    userId: cache.value.userId,
    username: cache.value.username,
    displayName: cache.value.displayName,
    role: cache.value.role,
  } : null)

  async function fetchUser(): Promise<boolean> {
    if (cache.value && Date.now() - cache.value.checkedAt < AUTH_CACHE_TTL) {
      return cache.value.valid
    }
    try {
      const res = await fetch('/api/v1/user/me', { credentials: 'include', signal: AbortSignal.timeout(API_TIMEOUT_MS) })
      if (!res.ok) {
        cache.value = { valid: false, role: '', userId: '', username: '', displayName: '', checkedAt: Date.now() }
        return false
      }
      const data = await res.json()
      const userData = data.data || data
      cache.value = {
        valid: true,
        role: userData.role || 'visitor',
        userId: userData.id || userData.ID || '',
        username: userData.username || '',
        displayName: userData.display_name || userData.displayName || userData.username || '',
        checkedAt: Date.now(),
      }
      return true
    } catch {
      cache.value = { valid: false, role: '', userId: '', username: '', displayName: '', checkedAt: Date.now() }
      return false
    }
  }

  /**
   * 确保游客会话 - 如果未登录则自动创建游客账号
   * 用于路由守卫中自动游客登录
   */
  async function ensureGuestSession(): Promise<boolean> {
    // 已登录直接返回
    if (isAuthenticated.value) return true

    try {
      const fingerprint = await generateDeviceFingerprint()
      const res = await fetch('/api/v1/auth/guest-login', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        credentials: 'include',
        signal: AbortSignal.timeout(API_TIMEOUT_MS),
        body: JSON.stringify({ device_fingerprint: fingerprint }),
      })
      if (!res.ok) return false

      const data = await res.json()
      const userData = data.data || data
      cache.value = {
        valid: true,
        role: userData.role || 'guest',
        userId: userData.id || '',
        username: userData.username || '',
        displayName: userData.display_name || userData.username || '',
        checkedAt: Date.now(),
      }
      return true
    } catch {
      return false
    }
  }

  /**
   * 游客升级为正式账号
   */
  async function upgradeAccount(username: string, password: string, email?: string): Promise<boolean> {
    isUpgrading.value = true
    try {
      const res = await fetch('/api/v1/auth/upgrade-guest', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'X-CSRF-Token': getCSRFToken(),
        },
        credentials: 'include',
        signal: AbortSignal.timeout(API_TIMEOUT_MS),
        body: JSON.stringify({ username, password, email: email || '' }),
      })
      if (!res.ok) return false

      const data = await res.json()
      const userData = data.data || data
      cache.value = {
        valid: true,
        role: userData.role || 'visitor',
        userId: userData.id || '',
        username: userData.username || '',
        displayName: userData.username || '',
        checkedAt: Date.now(),
      }
      return true
    } catch {
      return false
    } finally {
      isUpgrading.value = false
    }
  }

  function invalidateAuth() {
    cache.value = null
  }

  async function logout() {
    try {
      await fetch('/api/v1/logout', {
        method: 'POST',
        credentials: 'include',
        signal: AbortSignal.timeout(API_TIMEOUT_MS),
        headers: { 'X-CSRF-Token': getCSRFToken() },
      })
    } catch { /* ignore */ }
    invalidateAuth()
  }

  return {
    cache, isAuthenticated, isAdmin, isGuest, displayName, isUpgrading,
    user, fetchUser, ensureGuestSession, upgradeAccount, invalidateAuth, logout,
  }
})
