import { createRouter, createWebHistory, type RouteRecordRaw } from 'vue-router'
import { useAuthStore } from '@/stores/auth'

const placeholder = () => import('@/views/Placeholder.vue')

// Planned modules: menus exist so navigation and permissions can already be tested.
const planned: [string, string][] = [
  ['system/factories', '厂区管理'],
  ['system/users', '用户管理'],
  ['sale/customers', '客户管理'],
  ['sale/contracts', '合同查询'],
  ['production/area-tables', '面积表查询'],
  ['production/handovers', '交接单管理'],
  ['delivery/notes', '发货查询'],
  ['finance/batch-valuation', '工程批次对账'],
  ['finance/deliver-accounts', '发货对账查询'],
]

const children: RouteRecordRaw[] = [
  { path: '', name: 'home', component: () => import('@/views/Home.vue'), meta: { title: '首页', public: true } },
  { path: 'sale/projects', name: 'projects', component: () => import('@/views/sale/Projects.vue'), meta: { title: '项目管理', group: '销售管理' } },
  { path: 'scheduling/batches', name: 'batches', component: () => import('@/views/scheduling/Batches.vue'), meta: { title: '生产批次查询', group: '排程管理' } },
  ...planned.map(([path, title]): RouteRecordRaw => ({ path, component: placeholder, meta: { title } })),
]

export const routes: RouteRecordRaw[] = [
  { path: '/login', name: 'login', component: () => import('@/views/Login.vue'), meta: { title: '登录', anonymous: true } },
  { path: '/', component: () => import('@/layouts/BasicLayout.vue'), children },
  { path: '/:pathMatch(.*)*', name: 'not-found', component: () => import('@/views/NotFound.vue'), meta: { anonymous: true } },
]

export const router = createRouter({ history: createWebHistory(), routes })

router.beforeEach(async (to) => {
  const auth = useAuthStore()
  if (to.meta.anonymous) return true
  if (!auth.loggedIn) return { name: 'login', query: { redirect: to.fullPath } }
  if (!auth.user) {
    try {
      await auth.loadMe()
    } catch {
      auth.logout()
      return { name: 'login', query: { redirect: to.fullPath } }
    }
  }
  if (!to.meta.public && !auth.allowedPaths.has(to.path)) {
    return { name: 'not-found', params: { pathMatch: to.path.slice(1).split('/') } }
  }
  return true
})

router.afterEach((to) => {
  document.title = to.meta.title ? `${to.meta.title} - GZ MES` : 'GZ MES'
})
