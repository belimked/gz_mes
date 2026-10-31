// Fictitious demo data. Do NOT put real customer/project names here: the demo is public.
import type { components } from '../schema'

type S = components['schemas']

export type DemoUser = S['UserDetail'] & { password: string }

export type DemoBatchValuation = Pick<
  S['BatchValuation'],
  'batchId' | 'basis' | 'basisArea' | 'unitPrice' | 'ingotPrice' | 'amount' | 'remark' | 'audited'
> & {
  auditedBy: string | null
  auditedAt: string | null
  updatedBy: string | null
  updatedAt: string | null
}

export const DEMO_ACCOUNTS = [
  { username: 'admin', password: 'admin123', note: '管理员，可看全部厂区' },
  { username: 'waixie', password: 'demo123', note: '外协厂账号，只能看外协厂A' },
]

export const factories: S['Factory'][] = [
  { id: 1, code: 'F01', name: '本厂', kind: 'own', isStop: false },
  { id: 2, code: 'W01', name: '外协厂A', kind: 'outsource', isStop: false },
  { id: 3, code: 'W02', name: '外协厂B', kind: 'outsource', isStop: false },
]

// Mirrors backend/db/migrations/00001_init.sql.
export const menus: { id: number; parentId: number | null; title: string; path: string; icon?: string }[] = [
  { id: 100, parentId: null, title: '系统管理', path: '/system', icon: 'setting' },
  { id: 101, parentId: 100, title: '厂区管理', path: '/system/factories', icon: 'home' },
  { id: 102, parentId: 100, title: '用户管理', path: '/system/users', icon: 'team' },
  { id: 200, parentId: null, title: '销售管理', path: '/sale', icon: 'shop' },
  { id: 201, parentId: 200, title: '项目管理', path: '/sale/projects', icon: 'project' },
  { id: 202, parentId: 200, title: '客户管理', path: '/sale/customers', icon: 'contacts' },
  { id: 203, parentId: 200, title: '合同查询', path: '/sale/contracts', icon: 'file-text' },
  { id: 300, parentId: null, title: '排程管理', path: '/scheduling', icon: 'schedule' },
  { id: 301, parentId: 300, title: '订单批次排程', path: '/scheduling/batches', icon: 'calendar' },
  { id: 400, parentId: null, title: '生产管理', path: '/production', icon: 'build' },
  { id: 401, parentId: 400, title: '标签管理', path: '/production/labels', icon: 'barcode' },
  { id: 402, parentId: 400, title: '交接单管理', path: '/production/handovers', icon: 'swap' },
  { id: 403, parentId: 400, title: '扫码报工', path: '/production/scan', icon: 'scan' },
  { id: 800, parentId: null, title: '发货管理', path: '/delivery', icon: 'car' },
  { id: 801, parentId: 800, title: '发货查询', path: '/delivery/notes', icon: 'truck' },
  { id: 900, parentId: null, title: '财务管理', path: '/finance', icon: 'account-book' },
  { id: 901, parentId: 900, title: '工程批次对账', path: '/finance/batch-valuation', icon: 'calculator' },
  { id: 902, parentId: 900, title: '发货对账查询', path: '/finance/deliver-accounts', icon: 'audit' },
]

export const users: DemoUser[] = [
  {
    id: 1,
    username: 'admin',
    password: 'admin123',
    displayName: '演示管理员',
    isAdmin: true,
    enabled: true,
    factoryIds: [],
    menuIds: menus.map((m) => m.id),
    createdAt: '2026-01-01T00:00:00.000Z',
  },
  {
    id: 2,
    username: 'waixie',
    password: 'demo123',
    displayName: '外协A操作员',
    isAdmin: false,
    enabled: true,
    factoryIds: [2],
    menuIds: [201, 203],
    createdAt: '2026-01-01T00:00:00.000Z',
  },
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
    const status = i < 3 ? 'pending' : i < 8 ? 'scheduled' : i < 15 ? 'in_production' : 'done'
    const dueInDays =
      status === 'done'
        ? -(1 + (i % 5))
        : status === 'in_production'
          ? i % 2 === 0
            ? 3 + ((i * 3) % 18)
            : -(1 + (i % 5))
          : 7 + ((i * 3) % 24)
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
      requiredDate: new Date(Date.now() + dueInDays * 86400000).toISOString().slice(0, 10),
      urgency: i === 1 || i === 11 ? 'urgent' : 'normal',
      status,
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

export function seedBatchValuations(batches: S['Batch'][]): DemoBatchValuation[] {
  let done = 0
  let producing = 0
  return batches.flatMap((batch) => {
    const audited = batch.status === 'done'
    if (!audited && (batch.status !== 'in_production' || producing++ % 2 !== 0)) return []
    const basis: S['ValuationBasis'] = audited ? 'order_area' : 'item_area'
    const basisArea = audited ? batch.orderArea : batch.itemArea
    const unitPrice = audited ? [180, 200, 220][done++ % 3]! : 190
    const now = '2026-10-28T08:00:00.000Z'
    return [
      {
        batchId: batch.id,
        basis,
        basisArea,
        unitPrice,
        ingotPrice: audited ? 19800 : 0,
        amount: Math.round(basisArea * unitPrice * 100) / 100,
        remark: '',
        audited,
        auditedBy: audited ? '演示管理员' : null,
        auditedAt: audited ? now : null,
        updatedBy: '演示管理员',
        updatedAt: now,
      },
    ]
  })
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

export function seedLabels(batches: S['Batch'][], batchItems: S['BatchItem'][]): S['Label'][] {
  let id = Math.max(0, ...batchItems.map((item) => item.id)) + 1
  return batches
    .filter((batch) => batch.status !== 'pending')
    .flatMap((batch) =>
      batchItems
        .filter((item) => item.batchId === batch.id)
        .flatMap((item) =>
          Array.from({ length: item.qty }, (_, i) => ({
            id: id++,
            barCode: `${String(batch.id % 10000).padStart(4, '0')}${String(item.seq).padStart(3, '0')}${String(i + 1).padStart(3, '0')}${String(Math.floor(batch.id / 10000) % 1000).padStart(3, '0')}`,
            factoryId: batch.factoryId,
            batchId: batch.id,
            lotNo: batch.lotNo,
            projectName: batch.projectName,
            batchItemId: item.id,
            pieceNo: item.pieceNo,
            pieceIndex: i + 1,
            qty: item.qty,
            drawingNo: item.drawingNo ?? '',
            pattern: item.pattern ?? '',
            thickness: item.thickness,
            color: item.color ?? '',
            pieceArea: item.pieceArea,
            cancelled: false,
            lastJobId: null,
            lastJobName: null,
            lastScannedAt: null,
            createdAt: batch.createdAt,
          })),
        ),
    )
}

export function seedScans(batches: S['Batch'][], labels: S['Label'][], jobList: S['Job'][]): S['Scan'][] {
  let id = Math.max(0, ...labels.map((label) => label.id)) + 1
  const today = Date.parse(new Date().toISOString().slice(0, 10) + 'T00:00:00Z')
  // Hours from the preceding step, with deterministic per-batch variation.
  const intervals = [
    [1, 3],
    [2, 5],
    [0.5, 1],
    [4, 10],
    [6, 12],
    [1, 2],
    [0.5, 0.5],
    [3, 8],
    [1, 3],
    [12, 36],
  ]
  return batches.flatMap((batch, index) => {
    const scannedBy = ['张师傅', '李工', '王师傅', '陈工', '演示管理员'][index % 5]!
    const start = today - (2 + (index % 10)) * 86400000 + 8 * 3600000 + (index % 5) * 17 * 60000
    const count = batch.status === 'done' ? jobList.length : batch.status === 'in_production' ? 2 + (index % 6) : 0
    let stepMs = start
    const stepTimes = jobList.slice(0, count).map((_, step) => {
      if (step) {
        const [min, max] = intervals[step - 1] as [number, number]
        stepMs += (min + (((index * 7 + step * 3) % 11) / 10) * (max - min)) * 3600000
      }
      return stepMs
    })
    return labels
      .filter((label) => label.batchId === batch.id)
      .flatMap((label, labelIndex) => {
        const labelOffset = labelIndex * (20 + (index % 21)) * 1000
        if (count) label.createdAt = new Date(start + labelOffset - 3600000).toISOString()
        return jobList.slice(0, count).map((job, step) => {
          const scannedAt = new Date(stepTimes[step]! + labelOffset).toISOString()
          label.lastJobId = job.id
          label.lastJobName = job.name
          label.lastScannedAt = scannedAt
          return {
            id: id++,
            labelId: label.id,
            barCode: label.barCode,
            batchId: batch.id,
            lotNo: label.lotNo,
            pieceNo: label.pieceNo,
            pieceIndex: label.pieceIndex,
            jobId: job.id,
            jobName: job.name,
            scannedBy,
            scannedAt,
            device: '',
          }
        })
      })
  })
}

export function seedHandovers(batches: S['Batch'][], labels: S['Label'][], scans: S['Scan'][], jobList: S['Job'][]) {
  const job = jobList.find((j) => j.code === '20')!
  const scanned = new Set(scans.filter((s) => s.jobId === job.id).map((s) => s.labelId))
  const handovers: S['Handover'][] = []
  const handoverLabels: { handoverId: number; labelId: number; jobId: number }[] = []
  let id = Math.max(0, ...scans.map((s) => s.id)) + 1
  for (const batch of batches.filter((b) => b.status === 'in_production' || b.status === 'done')) {
    const rows = labels.filter(
      (l) => l.batchId === batch.id && !l.cancelled && scanned.has(l.id) && jobList.some((j) => j.id === l.lastJobId && j.seq >= job.seq),
    )
    if (!rows.length) continue
    const received = batch.status === 'done'
    const handoverId = id++
    handovers.push({
      id: handoverId,
      handoverNo: `JJ20261020${String(handovers.length + 1).padStart(4, '0')}`,
      factoryId: batch.factoryId,
      batchId: batch.id,
      lotNo: batch.lotNo,
      projectName: batch.projectName,
      jobId: job.id,
      jobCode: job.code,
      jobName: job.name,
      qty: rows.length,
      area: Number(rows.reduce((sum, l) => sum + l.pieceArea, 0).toFixed(4)),
      remark: '',
      packedBy: '演示管理员',
      packedAt: '2026-10-20T08:00:00.000Z',
      received,
      receivedBy: received ? '演示管理员' : null,
      receivedAt: received ? '2026-10-21T08:00:00.000Z' : null,
    })
    handoverLabels.push(...rows.map((l) => ({ handoverId, labelId: l.id, jobId: job.id })))
  }
  return { handovers, handoverLabels }
}

export function seedDeliveryNotes(batches: S['Batch'][], labels: S['Label'][], contracts: S['Contract'][]) {
  const notes: S['DeliveryNote'][] = []
  const noteLabels: { noteId: number; labelId: number }[] = []
  let id = Math.max(0, ...labels.map((l) => l.id)) + 1
  for (const [index, batch] of batches.filter((b) => b.status === 'done').entries()) {
    const rows = labels.filter((l) => l.batchId === batch.id && !l.cancelled)
    const deliveredAt = new Date(Date.now() - index * 2 * 86400000).toISOString().slice(0, 10)
    const contract = contracts.find((c) => c.id === batch.contractId)!
    const noteId = id++
    notes.push({
      id: noteId,
      noteNo: `FH20261025${String(notes.length + 1).padStart(4, '0')}`,
      factoryId: batch.factoryId,
      batchId: batch.id,
      lotNo: batch.lotNo,
      projectName: batch.projectName,
      customerId: contract.customerId,
      customerName: contract.customerName,
      contractNo: contract.contractNo,
      deliveryBatch: '第1车',
      qty: rows.length,
      area: Number(rows.reduce((sum, l) => sum + l.pieceArea, 0).toFixed(4)),
      logisticsCompany: '演示物流',
      plateNo: '粤A·12345',
      driver: '张师傅',
      driverPhone: '',
      deliveredAt,
      remark: '',
      accounted: false,
      accountId: null,
      createdBy: '演示管理员',
      createdAt: '2026-10-25T08:00:00.000Z',
    })
    noteLabels.push(...rows.map((l) => ({ noteId, labelId: l.id })))
  }
  return { notes, noteLabels }
}

export function seedDeliverAccounts(notes: S['DeliveryNote'][]) {
  const accounts: S['DeliverAccount'][] = []
  const grouped = new Map<number, S['DeliveryNote'][]>()
  for (const note of notes) grouped.set(note.customerId, [...(grouped.get(note.customerId) ?? []), note])
  const id = Math.max(0, ...notes.map((n) => n.id)) + 1
  const rows = grouped.values().next().value
  if (rows) {
    const selected = rows.slice(0, 2)
    const first = selected[0]!
    const totalArea = Number(selected.reduce((sum, n) => sum + n.area, 0).toFixed(4))
    const accountId = id
    accounts.push({
      id: accountId,
      accountNo: `DZ20261028${String(accounts.length + 1).padStart(4, '0')}`,
      factoryId: first.factoryId,
      customerId: first.customerId,
      customerName: first.customerName,
      noteCount: selected.length,
      totalQty: selected.reduce((sum, n) => sum + n.qty, 0),
      totalArea,
      unitPrice: 180,
      extraAmount: 200,
      extraRemark: '运费',
      amount: Math.round((totalArea * 180 + 200) * 100) / 100,
      status: 'confirmed',
      remark: '',
      createdBy: '演示管理员',
      createdAt: '2026-10-28T08:00:00.000Z',
      confirmedBy: '演示管理员',
      confirmedAt: '2026-10-28T09:00:00.000Z',
    })
    for (const note of selected) {
      note.accounted = true
      note.accountId = accountId
    }
  }
  return accounts
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
