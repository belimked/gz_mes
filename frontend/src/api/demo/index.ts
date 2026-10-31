// In-browser implementation of the API for the public demo build.
// Behaviour mirrors the Go backend (validation, factory isolation, error codes).
import type { components } from '../schema'
import {
  factories,
  jobs,
  menus,
  seedBatchItems,
  seedBatches,
  seedContracts,
  seedCustomers,
  seedHandovers,
  seedDeliveryNotes,
  seedDeliverAccounts,
  seedLabels,
  seedProjects,
  seedScans,
  users,
  type DemoUser,
} from './data'

type S = components['schemas']

interface DemoDB {
  version: 5
  nextId: number
  docSeq: Record<string, number>
  handovers: S['Handover'][]
  handoverLabels: { handoverId: number; labelId: number; jobId: number }[]
  deliveryNotes: S['DeliveryNote'][]
  deliveryNoteLabels: { noteId: number; labelId: number }[]
  deliverAccounts: S['DeliverAccount'][]
  projects: S['Project'][]
  customers: S['Customer'][]
  contracts: S['Contract'][]
  batches: S['Batch'][]
  batchItems: S['BatchItem'][]
  labels: S['Label'][]
  scans: S['Scan'][]
}

const KEY = 'gz-mes-demo-v1'
let memory: DemoDB | null = null

function storage(): Storage | null {
  try {
    return typeof localStorage === 'undefined' ? null : localStorage
  } catch {
    return null
  }
}

function fresh(): DemoDB {
  const projects = seedProjects()
  const customers = seedCustomers()
  const contracts = seedContracts(projects, customers)
  const batches = seedBatches(contracts)
  const batchItems = seedBatchItems(batches)
  const labels = seedLabels(batches, batchItems)
  const scans = seedScans(batches, labels, jobs)
  const { handovers, handoverLabels } = seedHandovers(batches, labels, scans, jobs)
  const { notes: deliveryNotes, noteLabels: deliveryNoteLabels } = seedDeliveryNotes(batches, labels, contracts)
  let nextId =
    Math.max(
      ...[...projects, ...customers, ...contracts, ...batches, ...batchItems, ...labels, ...scans, ...handovers].map((item) => item.id),
    ) + 1
  for (const note of deliveryNotes) {
    const oldId = note.id
    note.id = nextId++
    for (const link of deliveryNoteLabels) if (link.noteId === oldId) link.noteId = note.id
  }
  const deliverAccounts = seedDeliverAccounts(deliveryNotes)
  for (const account of deliverAccounts) {
    const oldId = account.id
    account.id = nextId++
    for (const note of deliveryNotes) if (note.accountId === oldId) note.accountId = account.id
  }
  return {
    version: 5,
    nextId,
    docSeq: {},
    handovers,
    handoverLabels,
    deliveryNotes,
    deliveryNoteLabels,
    deliverAccounts,
    projects,
    customers,
    contracts,
    batches,
    batchItems,
    labels,
    scans,
  }
}

function load(): DemoDB {
  const ls = storage()
  if (ls) {
    try {
      const parsed = JSON.parse(ls.getItem(KEY) ?? 'null') as DemoDB | null
      if (parsed?.version === 5) return parsed
    } catch {
      // fall through to a fresh dataset
    }
  }
  if (memory?.version !== 5) memory = fresh()
  return memory
}

function save(db: DemoDB) {
  memory = db
  storage()?.setItem(KEY, JSON.stringify(db))
}

/** Restores the original demo dataset. */
export function resetDemo() {
  memory = null
  storage()?.removeItem(KEY)
}

const json = (status: number, body: unknown) =>
  new Response(JSON.stringify(body), { status, headers: { 'Content-Type': 'application/json' } })
const fail = (status: number, code: string, message: string) => json(status, { code, message })

function principal(req: Request): DemoUser | null {
  const m = /^Bearer demo-token-(\d+)$/.exec(req.headers.get('Authorization') ?? '')
  return m ? (users.find((u) => u.id === Number(m[1])) ?? null) : null
}

const inScope = (u: DemoUser, factoryId: number) => u.isAdmin || u.factoryIds.includes(factoryId)

function menuTree(u: DemoUser): S['MenuItem'][] {
  const visible = new Set<number>()
  for (const m of menus) {
    if (u.isAdmin || u.menuIds.includes(m.id)) {
      visible.add(m.id)
      if (m.parentId !== null) visible.add(m.parentId)
    }
  }
  const build = (parentId: number | null): S['MenuItem'][] =>
    menus
      .filter((m) => m.parentId === parentId && visible.has(m.id))
      .map((m) => {
        const children = build(m.id)
        return {
          id: m.id,
          title: m.title,
          path: m.path,
          ...(m.icon ? { icon: m.icon } : {}),
          ...(children.length ? { children } : {}),
        }
      })
  return build(null)
}

async function body(req: Request): Promise<Record<string, unknown> | null> {
  try {
    const v: unknown = await req.json()
    return v && typeof v === 'object' ? (v as Record<string, unknown>) : null
  } catch {
    return null
  }
}

const str = (v: unknown) => (typeof v === 'string' ? v.trim() : '')

function progress(db: DemoDB, batch: S['Batch']): S['BatchProgress'] {
  const labels = db.labels.filter((label) => label.batchId === batch.id)
  const byId = new Map(labels.map((label) => [label.id, label]))
  const scans = db.scans.filter((scan) => scan.batchId === batch.id)
  return {
    batchId: batch.id,
    status: batch.status,
    totalQty: batch.itemQty,
    totalArea: batch.itemArea,
    labelQty: labels.filter((label) => !label.cancelled).length,
    jobs: jobs.map((job) => {
      const rows = scans.filter((scan) => scan.jobId === job.id)
      return {
        jobId: job.id,
        jobCode: job.code,
        jobName: job.name,
        seq: job.seq,
        scannedQty: rows.length,
        scannedArea: Number(rows.reduce((sum, scan) => sum + (byId.get(scan.labelId)?.pieceArea ?? 0), 0).toFixed(4)),
      }
    }),
  }
}

const validPage = (page: number, size: number) => Number.isInteger(page) && page >= 1 && Number.isInteger(size) && size >= 1 && size <= 200
const validId = (value: string | null) => value === null || (/^-?\d+$/.test(value) && Number.isSafeInteger(Number(value)))
function nextDocNo(db: DemoDB, prefix: string) {
  const day = new Date().toISOString().slice(0, 10).replaceAll('-', '')
  const key = prefix + day
  db.docSeq[key] = (db.docSeq[key] ?? 0) + 1
  return `${key}${String(db.docSeq[key]).padStart(4, '0')}`
}

const docCodes = (value: unknown): string[] | null =>
  Array.isArray(value) && value.length > 0 && value.every((code) => typeof code === 'string' && str(code))
    ? [...new Set(value.map((code: string) => code.trim()))]
    : null
const docArea = (labels: S['Label'][]) => Number(labels.reduce((sum, label) => sum + label.pieceArea, 0).toFixed(4))

const dateStart = (value: string) => {
  if (!/^\d{4}-\d{2}-\d{2}$/.test(value)) return NaN
  const ms = Date.parse(`${value}T00:00:00Z`)
  return Number.isFinite(ms) && new Date(ms).toISOString().slice(0, 10) === value ? ms : NaN
}

export async function demoFetch(req: Request): Promise<Response> {
  if (import.meta.env.MODE !== 'test') await new Promise((r) => setTimeout(r, 120))
  const url = new URL(req.url)
  const at = url.pathname.indexOf('/api/')
  const path = at >= 0 ? url.pathname.slice(at + 4) : url.pathname
  const route = `${req.method} ${path}`

  if (route === 'GET /health') return json(200, { status: 'ok', version: 'demo' })

  if (route === 'POST /auth/login') {
    const b = await body(req)
    if (!b || !str(b.username) || !str(b.password)) return fail(400, 'invalid_request', '请输入用户名和密码')
    const u = users.find((x) => x.username === b.username && x.password === b.password)
    if (!u) return fail(401, 'invalid_credentials', '用户名或密码错误')
    return json(200, {
      token: `demo-token-${u.id}`,
      user: { id: u.id, username: u.username, displayName: u.displayName, isAdmin: u.isAdmin },
    })
  }

  const u = principal(req)
  if (!u) return fail(401, 'unauthorized', '请先登录')

  if (route === 'GET /auth/me') {
    return json(200, {
      user: { id: u.id, username: u.username, displayName: u.displayName, isAdmin: u.isAdmin },
      menus: menuTree(u),
    })
  }

  if (route === 'GET /factories')
    return json(
      200,
      factories.filter((f) => inScope(u, f.id)),
    )

  const db = load()

  if (route === 'GET /projects') {
    const page = Number(url.searchParams.get('page') ?? 1)
    const size = Number(url.searchParams.get('size') ?? 20)
    if (!Number.isInteger(page) || page < 1 || !Number.isInteger(size) || size < 1 || size > 200) {
      return fail(400, 'invalid_request', 'page/size 参数不合法')
    }
    const kw = (url.searchParams.get('keyword') ?? '').trim().toLowerCase()
    const all = db.projects
      .filter((p) => inScope(u, p.factoryId))
      .filter((p) => !kw || [p.code, p.fullName, p.shortName].some((s) => s.toLowerCase().includes(kw)))
      .sort((a, b) => b.createdAt.localeCompare(a.createdAt) || b.id - a.id)
    return json(200, { items: all.slice((page - 1) * size, page * size), total: all.length })
  }

  if (route === 'POST /projects') {
    const b = await body(req)
    const factoryId = Number(b?.factoryId)
    if (!b || !Number.isInteger(factoryId) || typeof b.code !== 'string' || typeof b.fullName !== 'string') {
      return fail(400, 'invalid_request', '缺少必填字段')
    }
    if (!inScope(u, factoryId)) return fail(403, 'forbidden_factory', '无权在该厂区创建数据')
    const code = str(b.code)
    const fullName = str(b.fullName)
    if (!code || !fullName) return fail(400, 'invalid_request', '项目编号和名称不能为空')
    if (db.projects.some((p) => p.factoryId === factoryId && p.code === code)) {
      return fail(409, 'duplicate_code', '项目编号已存在')
    }
    const kind = b.kind === 'online' ? 'online' : 'offline'
    const p: S['Project'] = {
      id: db.nextId++,
      factoryId,
      code,
      fullName,
      shortName: str(b.shortName),
      address: str(b.address),
      kind,
      confirmed: false,
      createdBy: u.displayName,
      createdAt: new Date().toISOString(),
    }
    db.projects.push(p)
    save(db)
    return json(201, p)
  }

  const confirm = /^POST \/projects\/(\d+)\/confirm$/.exec(route)
  if (confirm) {
    const p = db.projects.find((x) => x.id === Number(confirm[1]) && inScope(u, x.factoryId))
    if (!p) return fail(404, 'not_found', '项目不存在')
    if (p.confirmed) return fail(409, 'already_confirmed', '项目已确认')
    p.confirmed = true
    save(db)
    return json(200, p)
  }

  if (route === 'GET /customers' || route === 'GET /contracts') {
    const page = Number(url.searchParams.get('page') ?? 1)
    const size = Number(url.searchParams.get('size') ?? 20)
    if (!Number.isInteger(page) || page < 1 || !Number.isInteger(size) || size < 1 || size > 200) {
      return fail(400, 'invalid_request', 'page/size 参数不合法')
    }
    const kw = (url.searchParams.get('keyword') ?? '').trim().toLowerCase()
    if (route === 'GET /customers') {
      const all = db.customers
        .filter((c) => inScope(u, c.factoryId))
        .filter((c) => !kw || [c.name, c.easCode, c.easName].some((s) => s.toLowerCase().includes(kw)))
        .sort((a, b) => b.createdAt.localeCompare(a.createdAt) || b.id - a.id)
      return json(200, { items: all.slice((page - 1) * size, page * size), total: all.length })
    }
    const customerId = url.searchParams.get('customerId')
    const projectId = url.searchParams.get('projectId')
    if ([customerId, projectId].some((v) => v !== null && (!/^\d+$/.test(v) || !Number.isSafeInteger(Number(v))))) {
      return fail(400, 'invalid_request', '客户/项目参数不合法')
    }
    const beginDate = url.searchParams.get('beginDate')
    const endDate = url.searchParams.get('endDate')
    const dateStart = (v: string) => {
      if (!/^\d{4}-\d{2}-\d{2}$/.test(v)) return NaN
      const ms = Date.parse(`${v}T00:00:00Z`)
      return Number.isFinite(ms) && new Date(ms).toISOString().slice(0, 10) === v ? ms : NaN
    }
    const begin = beginDate ? dateStart(beginDate) : null
    const end = endDate ? dateStart(endDate) + 86400000 : null
    if (Number.isNaN(begin) || Number.isNaN(end)) return fail(400, 'invalid_request', '日期参数不合法')
    const all = db.contracts
      .filter((c) => inScope(u, c.factoryId))
      .filter(
        (c) =>
          !kw ||
          [c.contractNo, c.customerContractNo, c.innerContractNo, c.customerName, c.projectName, c.salesman].some((s) =>
            s.toLowerCase().includes(kw),
          ),
      )
      .filter((c) => customerId === null || c.customerId === Number(customerId))
      .filter((c) => projectId === null || c.projectId === Number(projectId))
      .filter((c) => begin === null || Date.parse(c.createdAt) >= begin)
      .filter((c) => end === null || Date.parse(c.createdAt) < end)
      .sort((a, b) => b.createdAt.localeCompare(a.createdAt) || b.id - a.id)
    return json(200, { items: all.slice((page - 1) * size, page * size), total: all.length })
  }

  if (route === 'POST /customers') {
    const b = await body(req)
    const factoryId = Number(b?.factoryId)
    if (!b || b.factoryId == null || !Number.isInteger(factoryId) || typeof b.name !== 'string') {
      return fail(400, 'invalid_request', '缺少必填字段')
    }
    if (!inScope(u, factoryId)) return fail(403, 'forbidden_factory', '无权在该厂区创建数据')
    const name = str(b.name)
    if (!name) return fail(400, 'invalid_request', '客户名称不能为空')
    if (db.customers.some((c) => c.factoryId === factoryId && c.name === name)) {
      return fail(409, 'duplicate_code', '客户名称已存在')
    }
    const c: S['Customer'] = {
      id: db.nextId++,
      factoryId,
      name,
      contacts: str(b.contacts),
      phone: str(b.phone),
      address: str(b.address),
      bank: str(b.bank),
      account: str(b.account),
      easCode: str(b.easCode),
      easName: str(b.easName),
      isInternal: b.isInternal === true,
      confirmed: false,
      createdBy: u.displayName,
      createdAt: new Date().toISOString(),
    }
    db.customers.push(c)
    save(db)
    return json(201, c)
  }

  const confirmCustomer = /^POST \/customers\/(\d+)\/confirm$/.exec(route)
  if (confirmCustomer) {
    const c = db.customers.find((x) => x.id === Number(confirmCustomer[1]) && inScope(u, x.factoryId))
    if (!c) return fail(404, 'not_found', '客户不存在')
    if (c.confirmed) return fail(409, 'already_confirmed', '客户已确认')
    c.confirmed = true
    save(db)
    return json(200, c)
  }

  if (route === 'POST /contracts') {
    const b = await body(req)
    const factoryId = Number(b?.factoryId)
    const customerId = Number(b?.customerId)
    const projectId = Number(b?.projectId)
    if (
      !b ||
      b.factoryId == null ||
      b.customerId == null ||
      b.projectId == null ||
      !Number.isInteger(factoryId) ||
      typeof b.contractNo !== 'string' ||
      !Number.isInteger(customerId) ||
      !Number.isInteger(projectId)
    ) {
      return fail(400, 'invalid_request', '缺少必填字段')
    }
    if (!inScope(u, factoryId)) return fail(403, 'forbidden_factory', '无权在该厂区创建数据')
    const contractNo = str(b.contractNo)
    if (!contractNo) return fail(400, 'invalid_request', '合同单号不能为空')
    const customer = db.customers.find((c) => c.id === customerId && c.factoryId === factoryId)
    const project = db.projects.find((p) => p.id === projectId && p.factoryId === factoryId)
    if (!customer || !project) return fail(400, 'invalid_reference', '厂区/客户/项目不存在或不属于该厂区')
    if (db.contracts.some((c) => c.factoryId === factoryId && c.contractNo === contractNo)) {
      return fail(409, 'duplicate_code', '合同单号已存在')
    }
    const orderQty = b.orderQty === undefined ? 0 : b.orderQty
    const orderArea = b.orderArea === undefined ? 0 : b.orderArea
    if (
      typeof orderQty !== 'number' ||
      !Number.isInteger(orderQty) ||
      orderQty < 0 ||
      typeof orderArea !== 'number' ||
      !Number.isFinite(orderArea) ||
      orderArea < 0
    ) {
      return fail(400, 'invalid_request', '订单数量或面积不合法')
    }
    const c: S['Contract'] = {
      id: db.nextId++,
      factoryId,
      contractNo,
      customerContractNo: str(b.customerContractNo),
      innerContractNo: str(b.innerContractNo),
      customerId,
      customerName: customer.name,
      projectId,
      projectName: project.fullName,
      salesman: str(b.salesman),
      orderQty,
      orderArea,
      pricingManner: str(b.pricingManner),
      payment: str(b.payment),
      paymentPoint: str(b.paymentPoint),
      texture: str(b.texture),
      paintType: str(b.paintType),
      packing: str(b.packing),
      receiver: str(b.receiver),
      receiverPhone: str(b.receiverPhone),
      receiverAddress: str(b.receiverAddress),
      remark: str(b.remark),
      customerValuationAudited: false,
      outsourceValuationAudited: false,
      createdBy: u.displayName,
      createdAt: new Date().toISOString(),
    }
    db.contracts.push(c)
    save(db)
    return json(201, c)
  }

  const getContract = /^GET \/contracts\/(\d+)$/.exec(route)
  if (getContract) {
    const c = db.contracts.find((x) => x.id === Number(getContract[1]) && inScope(u, x.factoryId))
    return c ? json(200, c) : fail(404, 'not_found', '合同不存在')
  }

  const audit = /^POST \/contracts\/(\d+)\/audit-valuation$/.exec(route)
  if (audit) {
    const b = await body(req)
    if (b?.kind !== 'customer' && b?.kind !== 'outsource') return fail(400, 'invalid_request', '审核类型不合法')
    const c = db.contracts.find((x) => x.id === Number(audit[1]) && inScope(u, x.factoryId))
    if (!c) return fail(404, 'not_found', '合同不存在')
    const key = b.kind === 'customer' ? 'customerValuationAudited' : 'outsourceValuationAudited'
    if (c[key]) return fail(409, 'already_audited', '已审核')
    c[key] = true
    save(db)
    return json(200, c)
  }

  if (route === 'GET /jobs')
    return json(
      200,
      [...jobs].sort((a, b) => a.seq - b.seq),
    )

  if (route === 'GET /batches') {
    const page = Number(url.searchParams.get('page') ?? 1)
    const size = Number(url.searchParams.get('size') ?? 20)
    if (!Number.isInteger(page) || page < 1 || !Number.isInteger(size) || size < 1 || size > 200) {
      return fail(400, 'invalid_request', 'page/size 参数不合法')
    }
    const ids = ['contractId', 'projectId', 'customerId'] as const
    const filters = ids.map((key) => url.searchParams.get(key))
    if (filters.some((value) => value !== null && (!/^-?\d+$/.test(value) || !Number.isSafeInteger(Number(value))))) {
      return fail(400, 'invalid_request', '合同/项目/客户参数不合法')
    }
    const status = url.searchParams.get('status')
    const urgency = url.searchParams.get('urgency')
    if (status !== null && !['pending', 'scheduled', 'in_production', 'done'].includes(status))
      return fail(400, 'invalid_request', '状态参数不合法')
    if (urgency !== null && !['normal', 'urgent'].includes(urgency)) return fail(400, 'invalid_request', '紧急程度参数不合法')
    const beginDate = url.searchParams.get('beginDate')
    const endDate = url.searchParams.get('endDate')
    const validDate = (value: string) =>
      /^\d{4}-\d{2}-\d{2}$/.test(value) &&
      !Number.isNaN(Date.parse(value)) &&
      new Date(Date.parse(value)).toISOString().slice(0, 10) === value
    if ((beginDate && !validDate(beginDate)) || (endDate && !validDate(endDate))) return fail(400, 'invalid_request', '日期参数不合法')
    const kw = (url.searchParams.get('keyword') ?? '').trim().toLowerCase()
    const all = db.batches
      .filter((batch) => inScope(u, batch.factoryId))
      .filter(
        (batch) =>
          !kw ||
          [batch.lotNo, batch.clientOrderNo, batch.contractNo, batch.customerName, batch.projectName].some((s) =>
            s.toLowerCase().includes(kw),
          ),
      )
      .filter((batch) => ids.every((key, i) => filters[i] === null || batch[key] === Number(filters[i])))
      .filter((batch) => status === null || batch.status === status)
      .filter((batch) => urgency === null || batch.urgency === urgency)
      .filter(
        (batch) =>
          (!beginDate && !endDate) ||
          (batch.requiredDate !== null && (!beginDate || batch.requiredDate >= beginDate) && (!endDate || batch.requiredDate <= endDate)),
      )
      .sort((a, b) => b.createdAt.localeCompare(a.createdAt) || b.id - a.id)
    return json(200, { items: all.slice((page - 1) * size, page * size), total: all.length })
  }

  if (route === 'POST /batches') {
    const b = await body(req)
    if (!b || b.contractId == null || !Number.isSafeInteger(b.contractId) || typeof b.lotNo !== 'string')
      return fail(400, 'invalid_request', '缺少必填字段')
    const contract = db.contracts.find((c) => c.id === b.contractId && inScope(u, c.factoryId))
    if (!contract) return fail(404, 'not_found', '合同不存在')
    const lotNo = str(b.lotNo)
    if (!lotNo) return fail(400, 'invalid_request', '批次号不能为空')
    if (db.batches.some((batch) => batch.factoryId === contract.factoryId && batch.lotNo === lotNo))
      return fail(409, 'duplicate_code', '批次号已存在')
    const thickness = b.thickness === undefined ? 0 : b.thickness
    const orderQty = b.orderQty === undefined ? 0 : b.orderQty
    const orderArea = b.orderArea === undefined ? 0 : b.orderArea
    if (
      [thickness, orderArea].some((v) => typeof v !== 'number' || !Number.isFinite(v) || v < 0) ||
      typeof orderQty !== 'number' ||
      !Number.isInteger(orderQty) ||
      orderQty < 0
    )
      return fail(400, 'invalid_request', '订单数量、面积或板厚不合法')
    if (b.urgency != null && b.urgency !== 'normal' && b.urgency !== 'urgent') return fail(400, 'invalid_request', '紧急程度不合法')
    const now = new Date().toISOString()
    const batch: S['Batch'] = {
      id: db.nextId++,
      factoryId: contract.factoryId,
      contractId: contract.id,
      contractNo: contract.contractNo,
      customerId: contract.customerId,
      customerName: contract.customerName,
      projectId: contract.projectId,
      projectName: contract.projectName,
      lotNo,
      clientOrderNo: str(b.clientOrderNo),
      thickness: thickness as number,
      color: str(b.color),
      colorNo: str(b.colorNo),
      texture: str(b.texture),
      sprayingType: str(b.sprayingType),
      orderQty,
      orderArea: orderArea as number,
      deliveryAddress: str(b.deliveryAddress),
      specialRequest: str(b.specialRequest),
      requiredDate: str(b.requiredDate) || null,
      urgency: (b.urgency ?? 'normal') as S['Urgency'],
      status: 'pending',
      itemCount: 0,
      itemQty: 0,
      itemArea: 0,
      workshop: '',
      remark: str(b.remark),
      createdBy: u.displayName,
      createdAt: now,
      updatedAt: now,
    }
    db.batches.push(batch)
    save(db)
    return json(201, batch)
  }

  const batchLabels = /^POST \/batches\/(\d+)\/labels\/generate$/.exec(route)
  if (batchLabels) {
    const batch = db.batches.find((x) => x.id === Number(batchLabels[1]) && inScope(u, x.factoryId))
    if (!batch) return fail(404, 'not_found', '批次不存在')
    if (batch.itemCount === 0) return fail(400, 'no_items', '批次尚无构件清单')
    let created = 0
    const existing = new Set(
      db.labels.filter((label) => label.batchId === batch.id).map((label) => `${label.batchItemId}:${label.pieceIndex}`),
    )
    for (const item of db.batchItems.filter((x) => x.batchId === batch.id)) {
      for (let pieceIndex = 1; pieceIndex <= item.qty; pieceIndex++) {
        if (existing.has(`${item.id}:${pieceIndex}`)) continue
        db.labels.push({
          id: db.nextId++,
          barCode: `${String(batch.id % 10000).padStart(4, '0')}${String(item.seq).padStart(3, '0')}${String(pieceIndex).padStart(3, '0')}${String(Math.floor(batch.id / 10000) % 1000).padStart(3, '0')}`,
          factoryId: batch.factoryId,
          batchId: batch.id,
          lotNo: batch.lotNo,
          projectName: batch.projectName,
          batchItemId: item.id,
          pieceNo: item.pieceNo,
          pieceIndex,
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
          createdAt: new Date().toISOString(),
        })
        created++
      }
    }
    if (created) save(db)
    return json(200, { created, total: db.labels.filter((label) => label.batchId === batch.id && !label.cancelled).length })
  }

  const batchProgress = /^GET \/batches\/(\d+)\/progress$/.exec(route)
  if (batchProgress) {
    const batch = db.batches.find((x) => x.id === Number(batchProgress[1]) && inScope(u, x.factoryId))
    return batch ? json(200, progress(db, batch)) : fail(404, 'not_found', '批次不存在')
  }

  if (route === 'GET /labels') {
    const page = Number(url.searchParams.get('page') ?? 1)
    const size = Number(url.searchParams.get('size') ?? 20)
    if (!validPage(page, size)) return fail(400, 'invalid_request', 'page/size 参数不合法')
    const batchId = url.searchParams.get('batchId')
    if (!validId(batchId)) return fail(400, 'invalid_request', '批次参数不合法')
    const cancelled = url.searchParams.get('cancelled')
    if (cancelled !== null && cancelled !== 'true' && cancelled !== 'false') return fail(400, 'invalid_request', '作废参数不合法')
    const kw = (url.searchParams.get('keyword') ?? '').trim().toLowerCase()
    const all = db.labels
      .filter((label) => inScope(u, label.factoryId))
      .filter((label) => batchId === null || label.batchId === Number(batchId))
      .filter((label) => cancelled === null || label.cancelled === (cancelled === 'true'))
      .filter((label) => !kw || [label.barCode, label.pieceNo, label.lotNo, label.projectName].some((s) => s.toLowerCase().includes(kw)))
      .sort((a, b) => b.createdAt.localeCompare(a.createdAt) || b.id - a.id)
    return json(200, { items: all.slice((page - 1) * size, page * size), total: all.length })
  }

  const labelRoute = /^(GET|POST) \/labels\/([^/]+?)(\/cancel)?$/.exec(route)
  if (labelRoute) {
    const label = db.labels.find((x) => x.barCode === decodeURIComponent(labelRoute[2]!) && inScope(u, x.factoryId))
    if (!label) return fail(404, 'not_found', '标签不存在')
    if (labelRoute[1] === 'GET' && !labelRoute[3]) {
      const scans = db.scans
        .filter((scan) => scan.labelId === label.id)
        .sort((a, b) => a.scannedAt.localeCompare(b.scannedAt) || a.id - b.id)
      return json(200, { ...label, scans })
    }
    if (labelRoute[1] === 'POST' && labelRoute[3]) {
      if (label.cancelled) return fail(409, 'already_cancelled', '标签已作废')
      if (db.scans.some((scan) => scan.labelId === label.id)) return fail(409, 'has_scans', '标签已有扫描记录，不能作废')
      label.cancelled = true
      save(db)
      return json(200, label)
    }
  }

  if (route === 'GET /scans') {
    const page = Number(url.searchParams.get('page') ?? 1)
    const size = Number(url.searchParams.get('size') ?? 20)
    if (!validPage(page, size)) return fail(400, 'invalid_request', 'page/size 参数不合法')
    const batchId = url.searchParams.get('batchId')
    const jobId = url.searchParams.get('jobId')
    if (!validId(batchId) || !validId(jobId)) return fail(400, 'invalid_request', '批次/工序参数不合法')
    const beginDate = url.searchParams.get('beginDate')
    const endDate = url.searchParams.get('endDate')
    const begin = beginDate !== null ? dateStart(beginDate) : null
    const end = endDate !== null ? dateStart(endDate) + 86400000 : null
    if (Number.isNaN(begin) || Number.isNaN(end)) return fail(400, 'invalid_request', '日期参数不合法')
    const scopedBatches = new Set(db.batches.filter((batch) => inScope(u, batch.factoryId)).map((batch) => batch.id))
    const all = db.scans
      .filter((scan) => scopedBatches.has(scan.batchId))
      .filter((scan) => batchId === null || scan.batchId === Number(batchId))
      .filter((scan) => jobId === null || scan.jobId === Number(jobId))
      .filter((scan) => begin === null || Date.parse(scan.scannedAt) >= begin)
      .filter((scan) => end === null || Date.parse(scan.scannedAt) < end)
      .sort((a, b) => b.scannedAt.localeCompare(a.scannedAt) || b.id - a.id)
    return json(200, { items: all.slice((page - 1) * size, page * size), total: all.length })
  }

  if (route === 'POST /scans') {
    const b = await body(req)
    if (!b || !str(b.barCode) || !str(b.jobCode)) return fail(400, 'invalid_request', '条码和工序代码不能为空')
    const label = db.labels.find((x) => x.barCode === str(b.barCode) && inScope(u, x.factoryId))
    if (!label) return fail(404, 'not_found', '标签不存在')
    const job = jobs.find((x) => x.code === str(b.jobCode))
    if (!job) return fail(400, 'invalid_job', '工序代码不存在')
    if (label.cancelled) return fail(409, 'label_cancelled', '标签已作废')
    if (db.scans.some((scan) => scan.labelId === label.id && scan.jobId === job.id))
      return fail(409, 'already_scanned', '该标签在此工序已扫描')
    if (job.seq > 1) {
      const previous = jobs.find((x) => x.seq === job.seq - 1)!
      if (!db.scans.some((scan) => scan.labelId === label.id && scan.jobId === previous.id))
        return fail(409, 'out_of_order', `请先完成上一道工序: ${previous.name}`)
    }
    const scannedAt = new Date().toISOString()
    const scan: S['Scan'] = {
      id: db.nextId++,
      labelId: label.id,
      barCode: label.barCode,
      batchId: label.batchId,
      lotNo: label.lotNo,
      pieceNo: label.pieceNo,
      pieceIndex: label.pieceIndex,
      jobId: job.id,
      jobName: job.name,
      scannedBy: u.displayName,
      scannedAt,
      device: str(b.device),
    }
    db.scans.push(scan)
    label.lastJobId = job.id
    label.lastJobName = job.name
    label.lastScannedAt = scannedAt
    const batch = db.batches.find((x) => x.id === label.batchId)!
    if (batch.status === 'pending' || batch.status === 'scheduled') batch.status = 'in_production'
    if (job.seq === jobs.length) {
      const completed = new Set(db.scans.filter((row) => row.batchId === batch.id && row.jobId === job.id).map((row) => row.labelId))
      if (db.labels.filter((x) => x.batchId === batch.id && !x.cancelled).every((x) => completed.has(x.id))) batch.status = 'done'
    }
    batch.updatedAt = scannedAt
    save(db)
    return json(201, { scan, label, progress: progress(db, batch) })
  }

  if (route === 'GET /handovers' || route === 'GET /delivery-notes' || route === 'GET /deliver-accounts') {
    const page = Number(url.searchParams.get('page') ?? 1)
    const size = Number(url.searchParams.get('size') ?? 20)
    if (!validPage(page, size)) return fail(400, 'invalid_request', 'page/size 参数不合法')
    const ids =
      route === 'GET /deliver-accounts' ? ['customerId'] : route === 'GET /handovers' ? ['batchId', 'jobId'] : ['batchId', 'customerId']
    if (ids.some((key) => !validId(url.searchParams.get(key)))) return fail(400, 'invalid_request', '筛选参数不合法')
    const kw = (url.searchParams.get('keyword') ?? '').trim().toLowerCase()
    const matches = (values: string[]) => !kw || values.some((value) => value.toLowerCase().includes(kw))
    const filterId = (key: string, value: number) => url.searchParams.get(key) === null || value === Number(url.searchParams.get(key))
    if (route === 'GET /handovers') {
      const received = url.searchParams.get('received')
      if (received !== null && !['true', 'false'].includes(received)) return fail(400, 'invalid_request', '接收参数不合法')
      const all = db.handovers
        .filter((h) => inScope(u, h.factoryId) && filterId('batchId', h.batchId) && filterId('jobId', h.jobId))
        .filter((h) => received === null || h.received === (received === 'true'))
        .filter((h) => matches([h.handoverNo, h.lotNo, h.projectName]))
        .sort((a, b) => b.packedAt.localeCompare(a.packedAt) || b.id - a.id)
      return json(200, { items: all.slice((page - 1) * size, page * size), total: all.length })
    }
    if (route === 'GET /delivery-notes') {
      const accounted = url.searchParams.get('accounted')
      if (accounted !== null && !['true', 'false'].includes(accounted)) return fail(400, 'invalid_request', '对账参数不合法')
      const begin = url.searchParams.get('beginDate')
      const end = url.searchParams.get('endDate')
      if (
        (begin !== null && Number.isNaN(dateStart(begin))) ||
        (end !== null && Number.isNaN(dateStart(end))) ||
        (begin && end && begin > end)
      )
        return fail(400, 'invalid_request', '日期范围无效')
      const all = db.deliveryNotes
        .filter((n) => inScope(u, n.factoryId) && filterId('batchId', n.batchId) && filterId('customerId', n.customerId))
        .filter((n) => accounted === null || n.accounted === (accounted === 'true'))
        .filter((n) => (!begin || n.deliveredAt >= begin) && (!end || n.deliveredAt <= end))
        .filter((n) => matches([n.noteNo, n.lotNo, n.projectName, n.plateNo]))
        .sort((a, b) => b.deliveredAt.localeCompare(a.deliveredAt) || b.id - a.id)
      return json(200, { items: all.slice((page - 1) * size, page * size), total: all.length })
    }
    const status = url.searchParams.get('status')
    if (status !== null && status !== 'draft' && status !== 'confirmed') return fail(400, 'invalid_request', '状态参数不合法')
    const all = db.deliverAccounts
      .filter((a) => inScope(u, a.factoryId) && filterId('customerId', a.customerId))
      .filter((a) => status === null || a.status === status)
      .filter((a) => matches([a.accountNo, a.customerName]))
      .sort((a, b) => b.createdAt.localeCompare(a.createdAt) || b.id - a.id)
    return json(200, { items: all.slice((page - 1) * size, page * size), total: all.length })
  }

  if (route === 'POST /handovers' || route === 'POST /delivery-notes') {
    const b = await body(req)
    const codes = docCodes(b?.barCodes)
    const isHandover = route === 'POST /handovers'
    if (
      !b ||
      !Number.isSafeInteger(b.batchId) ||
      !codes ||
      (isHandover ? !str(b.jobCode) : typeof b.deliveredAt !== 'string' || Number.isNaN(dateStart(b.deliveredAt)))
    )
      return fail(400, 'invalid_request', isHandover ? '条码和工序不能为空' : '条码和发货日期不能为空')
    const batch = db.batches.find((x) => x.id === b.batchId && inScope(u, x.factoryId))
    if (!batch) return fail(404, 'not_found', '批次不存在')
    const job = isHandover ? jobs.find((x) => x.code === str(b.jobCode)) : jobs.find((x) => x.code === '55')
    if (!job) return fail(400, 'invalid_job', '工序代码不存在')
    const labels: S['Label'][] = []
    for (const code of codes) {
      const label = db.labels.find((l) => l.barCode === code)
      if (!label || label.batchId !== batch.id || label.cancelled)
        return fail(400, 'invalid_reference', `条码不存在/不属于该批次/已作废: ${code}`)
      labels.push(label)
    }
    labels.sort((a, b) => a.id - b.id)
    for (const label of labels) {
      if (!db.scans.some((s) => s.labelId === label.id && s.jobId === job.id))
        return fail(409, 'not_scanned', `${isHandover ? '有标签尚未在该工序扫描' : '有标签尚未完成成品装架'}: ${label.barCode}`)
    }
    for (const label of labels) {
      if (isHandover && db.handoverLabels.some((link) => link.labelId === label.id && link.jobId === job.id))
        return fail(409, 'label_in_handover', `标签已在其他交接单: ${label.barCode}`)
      if (!isHandover && db.deliveryNoteLabels.some((link) => link.labelId === label.id))
        return fail(409, 'label_delivered', `标签已发货: ${label.barCode}`)
    }
    const now = new Date().toISOString()
    if (isHandover) {
      const h: S['Handover'] = {
        id: db.nextId++,
        handoverNo: nextDocNo(db, 'JJ'),
        factoryId: batch.factoryId,
        batchId: batch.id,
        lotNo: batch.lotNo,
        projectName: batch.projectName,
        jobId: job.id,
        jobCode: job.code,
        jobName: job.name,
        qty: labels.length,
        area: docArea(labels),
        remark: str(b.remark),
        packedBy: u.displayName,
        packedAt: now,
        received: false,
        receivedBy: null,
        receivedAt: null,
      }
      db.handovers.push(h)
      db.handoverLabels.push(...labels.map((l) => ({ handoverId: h.id, labelId: l.id, jobId: job.id })))
      save(db)
      return json(201, { ...h, labels })
    }
    const contract = db.contracts.find((c) => c.id === batch.contractId)!
    const n: S['DeliveryNote'] = {
      id: db.nextId++,
      noteNo: nextDocNo(db, 'FH'),
      factoryId: batch.factoryId,
      batchId: batch.id,
      lotNo: batch.lotNo,
      projectName: batch.projectName,
      customerId: contract.customerId,
      customerName: contract.customerName,
      contractNo: contract.contractNo,
      deliveryBatch: str(b.deliveryBatch),
      qty: labels.length,
      area: docArea(labels),
      logisticsCompany: str(b.logisticsCompany),
      plateNo: str(b.plateNo),
      driver: str(b.driver),
      driverPhone: str(b.driverPhone),
      deliveredAt: b.deliveredAt as string,
      remark: str(b.remark),
      accounted: false,
      accountId: null,
      createdBy: u.displayName,
      createdAt: now,
    }
    db.deliveryNotes.push(n)
    db.deliveryNoteLabels.push(...labels.map((l) => ({ noteId: n.id, labelId: l.id })))
    const job60 = jobs.find((j) => j.code === '60')!
    for (const label of labels) {
      if (db.scans.some((s) => s.labelId === label.id && s.jobId === job60.id)) continue
      db.scans.push({
        id: db.nextId++,
        labelId: label.id,
        barCode: label.barCode,
        batchId: batch.id,
        lotNo: batch.lotNo,
        pieceNo: label.pieceNo,
        pieceIndex: label.pieceIndex,
        jobId: job60.id,
        jobName: job60.name,
        scannedBy: u.displayName,
        scannedAt: now,
        device: 'delivery',
      })
      label.lastJobId = job60.id
      label.lastJobName = job60.name
      label.lastScannedAt = now
    }
    if (batch.status === 'pending' || batch.status === 'scheduled') batch.status = 'in_production'
    const shipped = new Set(db.scans.filter((s) => s.batchId === batch.id && s.jobId === job60.id).map((s) => s.labelId))
    if (db.labels.filter((l) => l.batchId === batch.id && !l.cancelled).every((l) => shipped.has(l.id))) batch.status = 'done'
    batch.updatedAt = now
    save(db)
    return json(201, { ...n, labels })
  }

  if (route === 'POST /deliver-accounts') {
    const b = await body(req)
    if (
      !b ||
      !Number.isSafeInteger(b.customerId) ||
      !Array.isArray(b.deliveryNoteIds) ||
      !b.deliveryNoteIds.length ||
      !b.deliveryNoteIds.every((id: unknown) => Number.isSafeInteger(id)) ||
      typeof b.unitPrice !== 'number' ||
      !Number.isFinite(b.unitPrice) ||
      b.unitPrice < 0 ||
      (b.extraAmount !== undefined && (typeof b.extraAmount !== 'number' || !Number.isFinite(b.extraAmount)))
    )
      return fail(400, 'invalid_request', '发货单不能为空且单价不能为负')
    const customer = db.customers.find((c) => c.id === b.customerId && inScope(u, c.factoryId))
    if (!customer) return fail(400, 'invalid_reference', '客户不存在或无权访问')
    const ids = [...new Set(b.deliveryNoteIds as number[])]
    const notes: S['DeliveryNote'][] = []
    for (const id of ids.sort((a, b) => a - b)) {
      const note = db.deliveryNotes.find((n) => n.id === id && inScope(u, n.factoryId))
      if (!note) return fail(400, 'invalid_reference', '发货单不存在')
      notes.push(note)
    }
    for (const note of notes) {
      if (note.customerId !== customer.id) return fail(400, 'invalid_reference', '发货单不属于该客户')
      if (note.accounted) return fail(409, 'note_accounted', `发货单已对账: ${note.noteNo}`)
    }
    const totalArea = Number(notes.reduce((sum, n) => sum + n.area, 0).toFixed(4))
    const extraAmount = (b.extraAmount ?? 0) as number
    const a: S['DeliverAccount'] = {
      id: db.nextId++,
      accountNo: nextDocNo(db, 'DZ'),
      factoryId: customer.factoryId,
      customerId: customer.id,
      customerName: customer.name,
      noteCount: notes.length,
      totalQty: notes.reduce((sum, n) => sum + n.qty, 0),
      totalArea,
      unitPrice: b.unitPrice,
      extraAmount,
      extraRemark: str(b.extraRemark),
      amount: Math.round((totalArea * b.unitPrice + extraAmount) * 100) / 100,
      status: 'draft',
      remark: str(b.remark),
      createdBy: u.displayName,
      createdAt: new Date().toISOString(),
      confirmedBy: null,
      confirmedAt: null,
    }
    db.deliverAccounts.push(a)
    for (const note of notes) {
      note.accounted = true
      note.accountId = a.id
    }
    save(db)
    return json(201, { ...a, notes })
  }

  const docRoute = /^(GET|POST) \/(handovers|delivery-notes|deliver-accounts)\/(\d+)(?:\/(receive|confirm))?$/.exec(route)
  if (docRoute) {
    const [, method, kind, rawId, action] = docRoute
    const id = Number(rawId)
    if (kind === 'handovers') {
      const h = db.handovers.find((x) => x.id === id && inScope(u, x.factoryId))
      if (!h) return fail(404, 'not_found', '交接单不存在')
      if (method === 'GET' && !action)
        return json(200, {
          ...h,
          labels: db.handoverLabels.filter((x) => x.handoverId === id).map((x) => db.labels.find((l) => l.id === x.labelId)!),
        })
      if (method === 'POST' && action === 'receive') {
        if (h.received) return fail(409, 'already_received', '交接单已接收')
        h.received = true
        h.receivedBy = u.displayName
        h.receivedAt = new Date().toISOString()
        save(db)
        return json(200, h)
      }
    } else if (kind === 'delivery-notes') {
      const n = db.deliveryNotes.find((x) => x.id === id && inScope(u, x.factoryId))
      if (!n) return fail(404, 'not_found', '发货单不存在')
      if (method === 'GET' && !action)
        return json(200, {
          ...n,
          labels: db.deliveryNoteLabels.filter((x) => x.noteId === id).map((x) => db.labels.find((l) => l.id === x.labelId)!),
        })
    } else {
      const a = db.deliverAccounts.find((x) => x.id === id && inScope(u, x.factoryId))
      if (!a) return fail(404, 'not_found', '对账单不存在')
      if (method === 'GET' && !action)
        return json(200, { ...a, notes: db.deliveryNotes.filter((n) => n.accountId === id).sort((x, y) => x.id - y.id) })
      if (method === 'POST' && action === 'confirm') {
        if (a.status === 'confirmed') return fail(409, 'already_confirmed', '对账单已确认')
        a.status = 'confirmed'
        a.confirmedBy = u.displayName
        a.confirmedAt = new Date().toISOString()
        save(db)
        return json(200, a)
      }
    }
  }

  const batchRoute = /^(GET|PUT) \/batches\/(\d+)(?:\/(schedule|items))?$/.exec(route)
  if (batchRoute) {
    const batch = db.batches.find((x) => x.id === Number(batchRoute[2]) && inScope(u, x.factoryId))
    if (!batch) return fail(404, 'not_found', '批次不存在')
    const action = batchRoute[3]
    if (batchRoute[1] === 'GET' && !action) return json(200, batch)
    if (batchRoute[1] === 'GET' && action === 'items')
      return json(
        200,
        db.batchItems.filter((item) => item.batchId === batch.id).sort((a, b) => a.seq - b.seq),
      )
    if (batchRoute[1] === 'PUT' && (action === 'schedule' || action === 'items')) {
      if (batch.status === 'done') return fail(409, 'batch_done', '批次已完工，不能修改')
      if (action === 'schedule') {
        const b = await body(req)
        if (!b || Array.isArray(b)) return fail(400, 'invalid_request', '请求参数不合法')
        const outsourceFactory =
          b.outsourceFactoryId == null ? null : factories.find((f) => f.id === b.outsourceFactoryId && f.kind === 'outsource')
        if (b.outsourceFactoryId != null && !outsourceFactory) return fail(400, 'invalid_reference', '外协厂不存在或不是外协类型')
        if (b.urgency != null && b.urgency !== 'normal' && b.urgency !== 'urgent') return fail(400, 'invalid_request', '紧急程度不合法')
        const dateFields = [
          'planTechDate',
          'planDrawingDate',
          'planCutDate',
          'planSheetStart',
          'planSheetDelivery',
          'planSprayStart',
          'planSprayDelivery',
        ] as const
        for (const key of dateFields) if (b[key] != null) batch[key] = b[key] as string
        if (outsourceFactory) {
          batch.outsourceFactoryId = outsourceFactory.id
          batch.outsourceFactoryName = outsourceFactory.name
        }
        if (b.workshop != null) batch.workshop = str(b.workshop)
        if (b.remark != null) batch.remark = str(b.remark)
        if (b.urgency != null) batch.urgency = b.urgency as S['Urgency']
        if (batch.planSheetStart && batch.status === 'pending') batch.status = 'scheduled'
        batch.updatedAt = new Date().toISOString()
        save(db)
        return json(200, batch)
      }
      if (db.labels.some((label) => label.batchId === batch.id && !label.cancelled))
        return fail(409, 'has_labels', '已生成标签，不能再替换构件清单')
      let input: unknown
      try {
        input = await req.json()
      } catch {
        /* invalid request below */
      }
      if (
        !Array.isArray(input) ||
        input.some(
          (item) =>
            !item ||
            typeof item !== 'object' ||
            !str(item.pieceNo) ||
            typeof item.qty !== 'number' ||
            !Number.isInteger(item.qty) ||
            item.qty < 1,
        )
      )
        return fail(400, 'invalid_request', '构件编号或数量不合法')
      const seen = new Set<string>()
      for (const item of input) {
        const pieceNo = str(item.pieceNo)
        if (seen.has(pieceNo)) return fail(400, 'duplicate_piece_no', `构件编号重复: ${pieceNo}`)
        seen.add(pieceNo)
      }
      const items: S['BatchItem'][] = input.map((item, i) => ({
        id: db.nextId++,
        batchId: batch.id,
        seq: i + 1,
        pieceNo: str(item.pieceNo),
        drawingNo: str(item.drawingNo),
        pattern: str(item.pattern),
        thickness: item.thickness ?? 0,
        color: str(item.color),
        length: item.length ?? 0,
        width: item.width ?? 0,
        pieceArea: item.pieceArea ?? 0,
        sprayingArea: item.sprayingArea ?? 0,
        qty: item.qty,
        packageQty: item.packageQty ?? 0,
        weldLine: item.weldLine ?? 0,
        bendingTimes: item.bendingTimes ?? 0,
        remark: str(item.remark),
      }))
      db.batchItems = db.batchItems.filter((item) => item.batchId !== batch.id).concat(items)
      batch.itemCount = items.length
      batch.itemQty = items.reduce((sum, item) => sum + item.qty, 0)
      batch.itemArea = Number(items.reduce((sum, item) => sum + item.pieceArea * item.qty, 0).toFixed(4))
      batch.updatedAt = new Date().toISOString()
      save(db)
      return json(200, items)
    }
  }

  return fail(404, 'not_found', `演示环境未实现：${route}`)
}
