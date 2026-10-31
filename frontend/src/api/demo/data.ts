// Fictitious demo data. Do NOT put real customer/project names here: the demo is public.
import type { components } from '../schema'

type S = components['schemas']

export interface DemoUser {
  id: number
  username: string
  password: string
  displayName: string
  isAdmin: boolean
  factoryIds: number[]
  menuIds: number[]
}

export const DEMO_ACCOUNTS = [
  { username: 'admin', password: 'admin123', note: '管理员，可看全部厂区' },
  { username: 'waixie', password: 'demo123', note: '外协厂账号，只能看外协厂A' },
]

export const users: DemoUser[] = [
  { id: 1, username: 'admin', password: 'admin123', displayName: '演示管理员', isAdmin: true, factoryIds: [], menuIds: [] },
  { id: 2, username: 'waixie', password: 'demo123', displayName: '外协A操作员', isAdmin: false, factoryIds: [2], menuIds: [201, 203] },
]

export const factories: S['Factory'][] = [
  { id: 1, code: 'F01', name: '本厂', kind: 'own' },
  { id: 2, code: 'W01', name: '外协厂A', kind: 'outsource' },
  { id: 3, code: 'W02', name: '外协厂B', kind: 'outsource' },
]

// Mirrors backend/db/migrations/00001_init.sql.
export const menus: { id: number; parentId: number | null; title: string; path: string; icon?: string }[] = [
  { id: 100, parentId: null, title: '系统管理', path: '/system', icon: 'setting' },
  { id: 101, parentId: 100, title: '厂区管理', path: '/system/factories' },
  { id: 102, parentId: 100, title: '用户管理', path: '/system/users' },
  { id: 200, parentId: null, title: '销售管理', path: '/sale', icon: 'shop' },
  { id: 201, parentId: 200, title: '项目管理', path: '/sale/projects' },
  { id: 202, parentId: 200, title: '客户管理', path: '/sale/customers' },
  { id: 203, parentId: 200, title: '合同查询', path: '/sale/contracts' },
  { id: 300, parentId: null, title: '排程管理', path: '/scheduling', icon: 'schedule' },
  { id: 301, parentId: 300, title: '订单批次排程', path: '/scheduling/batches' },
  { id: 400, parentId: null, title: '生产管理', path: '/production', icon: 'build' },
  { id: 401, parentId: 400, title: '面积表查询', path: '/production/area-tables' },
  { id: 402, parentId: 400, title: '交接单管理', path: '/production/handovers' },
  { id: 800, parentId: null, title: '发货管理', path: '/delivery', icon: 'car' },
  { id: 801, parentId: 800, title: '发货查询', path: '/delivery/notes' },
  { id: 900, parentId: null, title: '财务管理', path: '/finance', icon: 'account-book' },
  { id: 901, parentId: 900, title: '工程批次对账', path: '/finance/batch-valuation' },
  { id: 902, parentId: 900, title: '发货对账查询', path: '/finance/deliver-accounts' },
]

const day = (d: number) => new Date(Date.UTC(2026, 0, d, 2, 30)).toISOString()

export function seedCustomers(): S['Customer'][] {
  const names = [
    '演示·华南幕墙工程有限公司',
    '演示·滨湖建筑装饰有限公司',
    '演示·云谷建设集团',
    '演示·新城金属制品有限公司',
    '演示·远景工程有限公司',
    '演示·南湾建筑有限公司',
    '演示·江畔装饰工程有限公司',
    '演示·星河建设有限公司',
  ]
  return names.map((name, i) => ({
    id: i + 6,
    factoryId: i < 6 ? 1 : 2,
    name,
    contacts: `演示·联系人${i + 1}`,
    phone: `1380000000${i}`,
    address: `演示·示例市${i + 1}号`,
    bank: '演示·示例银行',
    account: `DEMO-${i + 1}`,
    easCode: `C${String(i + 1).padStart(4, '0')}`,
    easName: name,
    isInternal: i === 2 || i === 6,
    confirmed: i % 3 === 0,
    createdBy: '演示管理员',
    createdAt: new Date(Date.now() - (i + 1) * 5 * 86400000).toISOString(),
  }))
}

export function seedContracts(projects: S['Project'][], customers: S['Customer'][]): S['Contract'][] {
  return Array.from({ length: 10 }, (_, i) => {
    const factoryId = i < 8 ? 1 : 2
    const customer = customers.filter((c) => c.factoryId === factoryId)[i % (factoryId === 1 ? 6 : 2)]!
    const project = projects.filter((p) => p.factoryId === factoryId)[i % 2]!
    return {
      id: i + 14,
      factoryId,
      contractNo: `HT-2026-${String(i + 1).padStart(4, '0')}`,
      customerContractNo: `KH-2026-${String(i + 1).padStart(4, '0')}`,
      innerContractNo: `NB-2026-${String(i + 1).padStart(4, '0')}`,
      customerId: customer.id,
      customerName: customer.name,
      projectId: project.id,
      projectName: project.fullName,
      salesman: ['演示·张华', '演示·李明', '演示·王芳'][i % 3]!,
      orderQty: 200 + i * 180,
      orderArea: Number((150.25 + i * 123.37).toFixed(2)),
      pricingManner: i % 2 ? '展开面积' : '计价面积',
      payment: '演示·按进度付款',
      paymentPoint: '演示·验收后',
      texture: '演示·铝板',
      paintType: i % 2 ? '氟碳' : '粉末',
      packing: '演示·纸箱',
      receiver: '演示·收货人',
      receiverPhone: '13800000000',
      receiverAddress: '演示·示例市收货地址',
      remark: '',
      customerValuationAudited: i % 3 === 0,
      outsourceValuationAudited: false,
      createdBy: '演示管理员',
      createdAt: new Date(Date.now() - (i + 1) * 4 * 86400000).toISOString(),
    }
  })
}

export const jobs: S['Job'][] = [
  ['10', '开料', 'sheet_metal'],
  ['15', '折弯', 'sheet_metal'],
  ['20', '钣金下线', 'sheet_metal'],
  ['25', '钣金质检', 'sheet_metal'],
  ['30', '喷涂接收', 'spray'],
  ['35', '上挂喷涂', 'spray'],
  ['40', '下挂冷却', 'spray'],
  ['45', '喷涂质检', 'spray'],
  ['50', '成品接收', 'finished'],
  ['55', '成品装架', 'finished'],
  ['60', '发货对账', 'finished'],
].map(([code, name, stage], i) => ({ id: i + 1, code: code!, name: name!, seq: i + 1, stage: stage as S['Job']['stage'] }))

const batchDate = (offset: number) => new Date(Date.UTC(2026, 4, 2 + offset)).toISOString().slice(0, 10)

export function seedBatches(contracts: S['Contract'][]): S['Batch'][] {
  const batches: S['Batch'][] = Array.from({ length: 18 }, (_, i) => {
    const contract = contracts[i % contracts.length]!
    const scheduled = i >= 3
    const orderQty = 40 + i * 8
    const createdAt = new Date(Date.now() - (i + 1) * 86400000).toISOString()
    return {
      id: i + 24,
      factoryId: contract.factoryId,
      contractId: contract.id,
      contractNo: contract.contractNo,
      customerId: contract.customerId,
      customerName: contract.customerName,
      projectId: contract.projectId,
      projectName: contract.projectName,
      lotNo: `PC-20241018-${String(i + 1).padStart(2, '0')}`,
      clientOrderNo: `KH-PC-${String(i + 1).padStart(3, '0')}`,
      thickness: [2.5, 3.0, 3.5, 4.0][i % 4]!,
      color: ['RAL 7016 亚光灰', 'RAL 9003 象牙白', 'RAL 5015 天蓝'][i % 3]!,
      colorNo: ['RAL 7016', 'RAL 9003', 'RAL 5015'][i % 3]!,
      texture: i % 2 ? '铝合金 3003-H14' : '铝单板',
      sprayingType: i % 2 ? '粉末' : '氟碳',
      orderQty,
      orderArea: Number((orderQty * 3.006).toFixed(2)),
      deliveryAddress: '演示·示例市收货地址',
      specialRequest: '',
      requiredDate: batchDate(i),
      urgency: i === 1 || i === 11 ? 'urgent' : 'normal',
      status: i < 3 ? 'pending' : i < 8 ? 'scheduled' : i < 15 ? 'in_production' : 'done',
      itemCount: 0,
      itemQty: 0,
      itemArea: 0,
      ...(scheduled
        ? {
            planTechDate: batchDate(i - 12),
            planDrawingDate: batchDate(i - 11),
            planCutDate: batchDate(i - 10),
            planSheetStart: batchDate(i - 9),
            planSheetDelivery: batchDate(i - 6),
            planSprayStart: batchDate(i - 5),
            planSprayDelivery: batchDate(i - 2),
          }
        : {}),
      ...(contract.factoryId === 2 ? { outsourceFactoryId: 2, outsourceFactoryName: '外协厂A' } : {}),
      workshop: scheduled ? '钣金一车间' : '',
      remark: '',
      createdBy: '演示管理员',
      createdAt,
      updatedAt: createdAt,
    }
  })
  const items = seedBatchItems(batches)
  for (const batch of batches) {
    const rows = items.filter((item) => item.batchId === batch.id)
    batch.itemCount = rows.length
    batch.itemQty = rows.reduce((sum, item) => sum + item.qty, 0)
    batch.itemArea = Number(rows.reduce((sum, item) => sum + item.pieceArea * item.qty, 0).toFixed(4))
  }
  return batches
}

export function seedBatchItems(batches: S['Batch'][]): S['BatchItem'][] {
  return batches.flatMap((batch, i) =>
    Array.from({ length: 3 }, (_, n) => {
      const length = 1200 + n * 100
      const width = 800 + n * 50
      const pieceArea = Number(((length * width) / 1e6).toFixed(4))
      return {
        id: 42 + i * 3 + n,
        batchId: batch.id,
        seq: n + 1,
        pieceNo: `YG-${String(i + 1).padStart(2, '0')}-${String(n + 1).padStart(2, '0')}`,
        drawingNo: '',
        pattern: '',
        thickness: batch.thickness,
        color: batch.color,
        length,
        width,
        pieceArea,
        sprayingArea: Number((pieceArea * 1.08).toFixed(4)),
        qty: Math.floor(batch.orderQty / 3),
        packageQty: 0,
        weldLine: 0,
        bendingTimes: 0,
        remark: '',
      }
    }),
  )
}

export function seedProjects(): S['Project'][] {
  return [
    {
      id: 1,
      factoryId: 1,
      code: 'DEMO-001',
      fullName: '演示·滨湖科创中心幕墙工程',
      shortName: '滨湖科创',
      address: '示例市滨湖区1号',
      kind: 'online',
      confirmed: true,
      createdBy: '演示管理员',
      createdAt: day(5),
    },
    {
      id: 2,
      factoryId: 1,
      code: 'DEMO-002',
      fullName: '演示·云谷产业园二期',
      shortName: '云谷二期',
      address: '示例市高新区2号',
      kind: 'offline',
      confirmed: false,
      createdBy: '演示管理员',
      createdAt: day(8),
    },
    {
      id: 3,
      factoryId: 2,
      code: 'DEMO-003',
      fullName: '演示·江畔文化馆外立面',
      shortName: '江畔文化馆',
      address: '示例市江畔路3号',
      kind: 'offline',
      confirmed: false,
      createdBy: '外协A操作员',
      createdAt: day(10),
    },
    {
      id: 4,
      factoryId: 2,
      code: 'DEMO-004',
      fullName: '演示·星河学校体育馆',
      shortName: '星河体育馆',
      address: '示例市星河路4号',
      kind: 'online',
      confirmed: true,
      createdBy: '演示管理员',
      createdAt: day(12),
    },
    {
      id: 5,
      factoryId: 3,
      code: 'DEMO-005',
      fullName: '演示·南湾商业综合体',
      shortName: '南湾商业',
      address: '示例市南湾5号',
      kind: 'offline',
      confirmed: false,
      createdBy: '演示管理员',
      createdAt: day(15),
    },
  ]
}
