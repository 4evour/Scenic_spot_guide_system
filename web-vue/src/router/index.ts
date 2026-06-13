import { createRouter, createWebHashHistory, type RouteRecordRaw } from 'vue-router'
import { useAuthStore } from '../stores/auth'

const BasicLayout = () => import('../layout/BasicLayout.vue')
const LoginView = () => import('../views/LoginView.vue')

const routes: RouteRecordRaw[] = [
  {
    path: '/login',
    name: 'login',
    component: LoginView,
    meta: { requiresAuth: false, title: 'login.title' },
  },
  // 带侧边栏 Layout 的管理端路由
  {
    path: '/',
    component: BasicLayout,
    redirect: '/dashboard',
    children: [
      {
        path: 'dashboard',
        name: 'dashboard',
        component: () => import('../views/DashboardView.vue'),
        meta: { title: 'nav.dashboard', requiresAdmin: true },
      },
      // 景区管理
      {
        path: 'admin/spots',
        name: 'admin-spots',
        component: () => import('../views/AdminSpots.vue'),
        meta: { title: 'nav.spots', parentTitle: 'nav.scenicMgmt', requiresAdmin: true },
      },
      {
        path: 'admin/routes',
        name: 'admin-routes',
        component: () => import('../views/AdminRoutes.vue'),
        meta: { title: 'nav.routes', parentTitle: 'nav.scenicMgmt', requiresAdmin: true },
      },
      {
        path: 'admin/content',
        name: 'admin-content',
        component: () => import('../views/AdminContent.vue'),
        meta: { title: 'nav.content', parentTitle: 'nav.scenicMgmt', requiresAdmin: true },
      },
      // 数字人中心
      {
        path: 'admin/avatar',
        name: 'admin-avatar',
        component: () => import('../views/AdminAvatar.vue'),
        meta: { title: 'nav.avatar', parentTitle: 'nav.digitalCenter', requiresAdmin: true },
      },
      {
        path: 'admin/reports',
        name: 'admin-reports',
        component: () => import('../views/AdminReports.vue'),
        meta: { title: 'nav.reports', parentTitle: 'nav.digitalCenter', requiresAdmin: true },
      },
      // 知识库
      {
        path: 'admin/knowledge',
        name: 'admin-knowledge',
        component: () => import('../views/AdminKnowledge.vue'),
        meta: { title: 'nav.knowledge', requiresAdmin: true },
      },
      // 系统管理
      {
        path: 'admin/users',
        name: 'admin-users',
        component: () => import('../views/AdminUsers.vue'),
        meta: { title: 'nav.users', parentTitle: 'nav.systemMgmt', requiresAdmin: true },
      },
      {
        path: 'admin/settings',
        name: 'admin-settings',
        component: () => import('../views/AdminSettings.vue'),
        meta: { title: 'nav.settings', parentTitle: 'nav.systemMgmt', requiresAdmin: true },
      },
    ],
  },
  // 游客端全屏路由（无 Layout）
  {
    path: '/map',
    name: 'map',
    component: () => import('../views/MapView.vue'),
    meta: { title: 'map.title', fullscreen: true, requiresAuth: true },
  },
  {
    path: '/digital-human',
    name: 'digital-human',
    component: () => import('../views/DigitalHumanView.vue'),
    meta: { title: 'dh.title', fullscreen: true, requiresAuth: true },
  },
  {
    path: '/scan',
    name: 'qr-scan',
    component: () => import('../views/QRScanView.vue'),
    meta: { title: 'qr.scanning', fullscreen: true, requiresAuth: false },
  },
  {
    path: '/:pathMatch(.*)*',
    redirect: '/login',
  },
]

const router = createRouter({
  history: createWebHashHistory(),
  routes,
})

router.beforeEach(async (to) => {
  if (to.meta.requiresAuth === false) {
    return true
  }

  const authStore = useAuthStore()

  // 尝试获取用户信息（Cookie 有效时直接返回）
  const authenticated = await authStore.fetchUser()
  if (authenticated) {
    // 管理员页面检查
    if (to.meta.requiresAdmin && !authStore.isAdmin) {
      return { name: 'map' }
    }
    return true
  }

  // 未登录时自动创建游客账号（而非重定向到登录页）
  const guestOk = await authStore.ensureGuestSession()
  if (!guestOk) {
    // 游客登录失败，降级到登录页
    return { name: 'login' }
  }

  // 管理员页面不允许游客访问
  if (to.meta.requiresAdmin && !authStore.isAdmin) {
    return { name: 'map' }
  }

  return true
})

// 自动上报页面访问
router.afterEach((to) => {
  const title = (to.meta?.title as string) || to.name?.toString() || to.path
  trackPageVisit(to.fullPath, title)
})


/** 从 cookie 中读取 csrf_token */
function getCSRFToken(): string {
  const match = document.cookie.match(/(?:^|;\s*)csrf_token=([^;]*)/);
  return match ? decodeURIComponent(match[1]) : '';
}/** 上报页面访问（静默失败，不影响用户体验） */
export function trackPageVisit(path: string, title: string) {
  const authStore = useAuthStore()
  fetch('/api/v1/track', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json', 'X-CSRF-Token': getCSRFToken() },
    credentials: 'include',
    signal: AbortSignal.timeout(5000),
    body: JSON.stringify({
      page: path,
      action: 'visit',
      details: title,
      user_id: authStore.user?.userId ? Number(authStore.user.userId) : 0,
    }),
  }).catch(() => {})
}

/** 上报用户操作（静默失败） */
export function trackUserAction(action: string, details: string) {
  const authStore = useAuthStore()
  fetch('/api/v1/track', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json', 'X-CSRF-Token': getCSRFToken() },
    credentials: 'include',
    signal: AbortSignal.timeout(5000),
    body: JSON.stringify({
      page: window.location.hash || '/',
      action,
      details,
      user_id: authStore.user?.userId ? Number(authStore.user.userId) : 0,
    }),
  }).catch(() => {})
}

export default router
export { routes }
