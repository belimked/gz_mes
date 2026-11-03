import { createRouter, createWebHistory, type RouteRecordRaw } from 'vue-router'
import { useAuthStore } from '@/stores/auth'

const children: RouteRecordRaw[] = [
  {
    path: 'drawing/list',
    name: 'drawings',
    component: () => import('@/views/drawing/Drawings.vue'),
    meta: { title: '图纸查询', group: '图纸管理' },
  },
  {
    path: 'board/spray-daily',
    name: 'board-spray-daily',
    component: () => import('@/views/board/SprayDailyBoard.vue'),
    meta: { title: '喷涂日排产看板', group: '看板管理' },
  },
  {
    path: 'board/sheet-daily',
    name: 'board-sheet-daily',
    component: () => import('@/views/board/SheetDailyBoard.vue'),
    meta: { title: '钣金日排程看板', group: '看板管理' },
  },
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
    path: 'scheduling/outsource-orders',
    name: 'outsource-orders',
    component: () => import('@/views/scheduling/OutsourceOrders.vue'),
    meta: { title: '外协单查看', group: '排程管理' },
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
    path: 'delivery/plans',
    name: 'delivery-plans',
    component: () => import('@/views/delivery/Plans.vue'),
    meta: { title: '发货计划', group: '发货管理' },
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
    path: 'report/batch-daily',
    name: 'report-batch-daily',
    component: () => import('@/views/report/BatchDaily.vue'),
    meta: { title: '工程日报', group: '生产报表' },
  },
  {
    path: 'report/batch-progress',
    name: 'report-batch-progress',
    component: () => import('@/views/report/BatchProgress.vue'),
    meta: { title: '工程进度', group: '生产报表' },
  },
  {
    path: 'report/staff-output',
    name: 'report-staff-output',
    component: () => import('@/views/report/StaffOutput.vue'),
    meta: { title: '员工产量', group: '生产报表' },
  },
  {
    path: 'report/scans',
    name: 'report-scans',
    component: () => import('@/views/report/ScanRecords.vue'),
    meta: { title: '扫描记录', group: '生产报表' },
  },
  {
    path: 'report/workshop-daily',
    name: 'report-workshop-daily',
    component: () => import('@/views/report/WorkshopOutput.vue'),
    props: { mode: 'day' },
    meta: { title: '车间产量日报', group: '生产报表' },
  },
  {
    path: 'report/workshop-monthly',
    name: 'report-workshop-monthly',
    component: () => import('@/views/report/WorkshopOutput.vue'),
    props: { mode: 'month' },
    meta: { title: '车间产量月报', group: '生产报表' },
  },
  {
    path: 'report/machine-output',
    name: 'report-machine-output',
    component: () => import('@/views/report/MachineOutput.vue'),
    meta: { title: '机台产量', group: '生产报表' },
  },
  {
    path: 'report/defect-daily',
    name: 'defect-daily',
    component: () => import('@/views/report/DefectDaily.vue'),
    meta: { title: '不良品日报', group: '生产报表' },
  },
  {
    path: 'quality/qc-scans',
    name: 'qc-scans',
    component: () => import('@/views/quality/QcScans.vue'),
    meta: { title: '质检扫描记录', group: '质量管理' },
  },
  {
    path: 'quality/report',
    name: 'quality-report',
    component: () => import('@/views/quality/QualityReport.vue'),
    meta: { title: '质量报表', group: '质量管理' },
  },
  {
    path: 'report/production-balance',
    name: 'report-production-balance',
    component: () => import('@/views/report/ProductionBalance.vue'),
    meta: { title: '生产平衡表', group: '生产报表' },
  },
  {
    path: 'system/factories',
    name: 'factories',
    component: () => import('@/views/system/Factories.vue'),
    meta: { title: '厂区管理', group: '系统管理' },
  },
  {
    path: 'system/machines',
    name: 'machines',
    component: () => import('@/views/system/Machines.vue'),
    meta: { title: '机台管理', group: '系统管理' },
  },
  {
    path: 'system/dicts',
    name: 'dicts',
    component: () => import('@/views/system/Dicts.vue'),
    meta: { title: '基础字典' },
  },
  {
    path: 'system/roles',
    name: 'roles',
    component: () => import('@/views/system/Roles.vue'),
    meta: { title: '角色管理', group: '系统管理' },
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
