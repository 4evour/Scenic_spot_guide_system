import { createRouter, createWebHashHistory, type RouteRecordRaw } from 'vue-router'
import { useAuthStore } from '../stores/auth'

const BasicLayout = () => import('../layout/BasicLayout.vue')
const LoginView = () => import('../views/LoginView.vue')

const routes: RouteRecordRaw[] = [
  {
    path: '/login',
    name: 'login',
    component: LoginView,
    meta: { requiresAuth: false, title: '登录' },
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
        meta: { title: '数据大屏', requiresAdmin: true },
      },
      // 景区管理
      {
        path: 'admin/spots',
        name: 'admin-spots',
        component: () => import('../views/AdminSpots.vue'),
        meta: { title: '景点管理', parentTitle: '景区管理', requiresAdmin: true },
      },
      {
        path: 'admin/routes',
        name: 'admin-routes',
        component: () => import('../views/AdminRoutes.vue'),
        meta: { title: '路线管理', parentTitle: '景区管理', requiresAdmin: true },
      },
      {
        path: 'admin/content',
        name: 'admin-content',
        component: () => import('../views/AdminContent.vue'),
        meta: { title: '讲解内容', parentTitle: '景区管理', requiresAdmin: true },
      },
      // 数字人中心
      {
        path: 'admin/avatar',
        name: 'admin-avatar',
        component: () => import('../views/AdminAvatar.vue'),
        meta: { title: '形象配置', parentTitle: '数字人中心', requiresAdmin: true },
      },
      {
        path: 'admin/reports',
        name: 'admin-reports',
        component: () => import('../views/AdminReports.vue'),
        meta: { title: '感受度报告', parentTitle: '数字人中心', requiresAdmin: true },
      },
      // 知识库
      {
        path: 'admin/knowledge',
        name: 'admin-knowledge',
        component: () => import('../views/AdminKnowledge.vue'),
        meta: { title: '知识库管理', requiresAdmin: true },
      },
      // 系统管理
      {
        path: 'admin/users',
        name: 'admin-users',
        component: () => import('../views/AdminUsers.vue'),
        meta: { title: '用户管理', parentTitle: '系统管理', requiresAdmin: true },
      },
      {
        path: 'admin/settings',
        name: 'admin-settings',
        component: () => import('../views/AdminSettings.vue'),
        meta: { title: '系统设置', parentTitle: '系统管理', requiresAdmin: true },
      },
    ],
  },
  // 游客端全屏路由（无 Layout）
  {
    path: '/map',
    name: 'map',
    component: () => import('../views/MapView.vue'),
    meta: { title: '地图导览', fullscreen: true },
  },
  {
    path: '/digital-human',
    name: 'digital-human',
    component: () => import('../views/DigitalHumanView.vue'),
    meta: { title: '数字人交互', fullscreen: true },
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

  const authenticated = await authStore.fetchUser()
  if (!authenticated) {
    return { name: 'login' }
  }

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

/** 上报页面访问（静默失败，不影响用户体验） */
export function trackPageVisit(path: string, title: string) {
  const authStore = useAuthStore()
  fetch('/api/v1/track', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
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
    headers: { 'Content-Type': 'application/json' },
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
