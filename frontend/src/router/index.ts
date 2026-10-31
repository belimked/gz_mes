import { createRouter, createWebHistory, type RouteRecordRaw } from 'vue-router'
import { useAuthStore } from '@/stores/auth'

const children: RouteRecordRaw[] = [
  { path: '', name: 'home', component: () => import('@/views/Home.vue'), meta: { title: '首页', public: true } },
  {
    path: 'sale/projects',
    name: 'projects',
    component: () => import('@/views/sale/Projects.vue'),
    meta: { title: '项目管理', group: '销售管理' },
  },
  {
    path: 'sale/customers',
    name: 'customers',
    component: () => import('@/views/sale/Customers.vue'),
    meta: { title: '客户管理', group: '销售管理' },
  },
  {
    path: 'sale/contracts',
    name: 'contracts',
    component: () => import('@/views/sale/Contracts.vue'),
    meta: { title: '合同查询', group: '销售管理' },
  },
  {
    path: 'scheduling/batches',
    name: 'batches',
    component: () => import('@/views/scheduling/Batches.vue'),
    meta: { title: '生产批次查询', group: '排程管理' },
  },
  {
    path: 'scheduling/sheet-week',
    name: 'sheet-week',
    component: () => import('@/views/scheduling/SheetWeek.vue'),
    meta: { title: '钣金周计划', group: '排程管理' },
  },
  {
    path: 'scheduling/sheet-daily',
    name: 'sheet-daily',
    component: () => import('@/views/scheduling/SheetDaily.vue'),
    meta: { title: '钣金日计划', group: '排程管理' },
  },
  {
    path: 'scheduling/spray-daily',
    name: 'spray-daily',
    component: () => import('@/views/scheduling/SprayDaily.vue'),
    meta: { title: '喷涂日计划', group: '排程管理' },
  },
  {
    path: 'production/labels',
    name: 'labels',
    component: () => import('@/views/production/Labels.vue'),
    meta: { title: '标签管理', group: '生产管理' },
  },
  {
    path: 'production/scan',
    name: 'scan',
    component: () => import('@/views/production/Scan.vue'),
    meta: { title: '扫码报工', group: '生产管理' },
  },
  {
    path: 'production/handovers',
    name: 'handovers',
    component: () => import('@/views/production/Handovers.vue'),
    meta: { title: '交接单管理', group: '生产管理' },
  },
  {
    path: 'delivery/notes',
    name: 'delivery-notes',
    component: () => import('@/views/delivery/Notes.vue'),
    meta: { title: '发货查询', group: '发货管理' },
  },
  {
    path: 'finance/batch-valuation',
    name: 'batch-valuation',
    component: () => import('@/views/finance/BatchValuation.vue'),
    meta: { title: '工程批次对账', group: '财务管理' },
  },
  {
    path: 'finance/deliver-accounts',
    name: 'deliver-accounts',
    component: () => import('@/views/finance/DeliverAccounts.vue'),
    meta: { title: '发货对账查询', group: '财务管理' },
  },
  {
    path: 'system/factories',
    name: 'factories',
    component: () => import('@/views/system/Factories.vue'),
    meta: { title: '厂区管理', group: '系统管理' },
  },
  {
    path: 'system/users',
    name: 'users',
    component: () => import('@/views/system/Users.vue'),
    meta: { title: '用户管理', group: '系统管理' },
  },
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
