// In-browser implementation of the API for the public demo build.
// Behaviour mirrors the Go backend (validation, factory isolation, error codes).
import type { components } from '../schema'
import {
  factories as seedFactories,
  jobs,
  menus,
  operators,
  seedBatchItems,
  seedBatches,
  seedBatchValuations,
  seedContracts,
  seedCustomers,
  seedHandovers,
  seedDeliveryNotes,
  seedDeliverAccounts,
  seedLabels,
  seedPlans,
  seedProjects,
  seedScans,
  users as seedUsers,
  type DemoUser,
  type DemoBatchValuation,
  type DemoPlan,
} from './data'

type S = components['schemas']

interface DemoDB {
  version: 15
  factories: S['Factory'][]
  users: DemoUser[]
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
  batchValuations: DemoBatchValuation[]
  batchItems: S['BatchItem'][]
  labels: S['Label'][]
  scans: S['Scan'][]
  plans: DemoPlan[]
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
  const plans = seedPlans(batches, jobs, scans)
  for (const plan of plans) plan.id = nextId++
  return {
    version: 15,
    factories: seedFactories.map((f) => ({ ...f })),
    users: seedUsers.map((u) => ({ ...u, factoryIds: [...u.factoryIds], menuIds: [...u.menuIds] })),
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
    batchValuations: seedBatchValuations(batches),
    batchItems,
    labels,
    scans,
    plans,
  }
}

function load(): DemoDB {
  const ls = storage()
  if (ls) {
    try {
      const parsed = JSON.parse(ls.getItem(KEY) ?? 'null') as DemoDB | null
      if (parsed?.version === 15) return parsed
    } catch {
      // fall through to a fresh dataset
    }
  }
  if (memory?.version !== 15) memory = fresh()
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

function principal(req: Request, db: DemoDB): DemoUser | null {
  const m = /^Bearer demo-token-(\d+)$/.exec(req.headers.get('Authorization') ?? '')
  return m ? (db.users.find((u) => u.id === Number(m[1]) && u.enabled) ?? null) : null
}

function userDetail(u: DemoUser): S['UserDetail'] {
  return {
    id: u.id,
    username: u.username,
    displayName: u.displayName,
    staffNo: u.staffNo,
    isAdmin: u.isAdmin,
    enabled: u.enabled,
    factoryIds: [...u.factoryIds],
    menuIds: [...u.menuIds],
    createdAt: u.createdAt,
  }
}

const validIds = (value: unknown): value is number[] => Array.isArray(value) && value.every((id) => Number.isSafeInteger(id))
const validReferences = (db: DemoDB, factoryIds: number[], menuIds: number[]) =>
  factoryIds.every((id) => db.factories.some((f) => f.id === id)) && menuIds.every((id) => menus.some((m) => m.id === id))

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

function valuationRow(db: DemoDB, batch: S['Batch']): S['BatchValuation'] {
  const shippedIds = new Set(db.deliveryNoteLabels.map((link) => link.labelId))
  const delivered = db.labels.filter((label) => label.batchId === batch.id && shippedIds.has(label.id))
  const valuation = db.batchValuations.find((row) => row.batchId === batch.id)
  return {
    batchId: batch.id,
    lotNo: batch.lotNo,
    contractId: batch.contractId,
    contractNo: batch.contractNo,
    customerId: batch.customerId,
    customerName: batch.customerName,
    projectName: batch.projectName,
    thickness: batch.thickness,
    color: batch.color,
    orderQty: batch.orderQty,
    orderArea: batch.orderArea,
    itemArea: batch.itemArea,
    deliveredQty: delivered.length,
    deliveredArea: docArea(delivered),
    status: batch.status,
    basis: valuation?.basis ?? 'order_area',
    basisArea: valuation?.basisArea ?? batch.orderArea,
    unitPrice: valuation?.unitPrice ?? 0,
    ingotPrice: valuation?.ingotPrice ?? 0,
    amount: valuation?.amount ?? 0,
    remark: valuation?.remark ?? '',
    audited: valuation?.audited ?? false,
    auditedBy: valuation?.auditedBy ?? null,
    auditedAt: valuation?.auditedAt ?? null,
    updatedBy: valuation?.updatedBy ?? null,
    updatedAt: valuation?.updatedAt ?? null,
    createdAt: batch.createdAt,
  }
}

const dateStart = (value: string) => {
  if (!/^\d{4}-\d{2}-\d{2}$/.test(value)) return NaN
  const ms = Date.parse(`${value}T00:00:00Z`)
  return Number.isFinite(ms) && new Date(ms).toISOString().slice(0, 10) === value ? ms : NaN
}

const PLAN_KINDS: S['PlanKind'][] = ['sheet_week', 'sheet_daily', 'spray_daily']
const PLAN_TARGET_CODE: Record<Exclude<S['PlanKind'], 'sheet_daily'>, string> = { sheet_week: '20', spray_daily: '40' }

/** Builds Plan rows (batch join + live progress at targetJob) with per-request indexes. */
function planRows(db: DemoDB) {
  const today = new Date().toISOString().slice(0, 10)
  const batchById = new Map(db.batches.map((batch) => [batch.id, batch]))
  const activeByBatch = new Map<number, S['Label'][]>()
  for (const label of db.labels) {
    if (label.cancelled) continue
    const rows = activeByBatch.get(label.batchId) ?? []
    rows.push(label)
    activeByBatch.set(label.batchId, rows)
  }
  const scanAt = new Map<string, string>()
  for (const scan of db.scans) scanAt.set(`${scan.labelId}:${scan.jobId}`, scan.scannedAt)
  return (plan: DemoPlan): S['Plan'] => {
    const batch = batchById.get(plan.batchId)!
    const job = plan.kind === 'sheet_daily' ? jobs.find((j) => j.id === plan.jobId) : undefined
    const target = job ?? jobs.find((j) => j.code === PLAN_TARGET_CODE[plan.kind as 'sheet_week'])!
    const labels = activeByBatch.get(batch.id) ?? []
    let doneQty = 0
    let doneArea = 0
    let lastScanAt: string | null = null
    for (const label of labels) {
      const at = scanAt.get(`${label.id}:${target.id}`)
      if (at === undefined) continue
      doneQty++
      doneArea += label.pieceArea
      if (lastScanAt === null || at > lastScanAt) lastScanAt = at
    }
    const progress = labels.length ? Math.floor((doneQty * 100) / labels.length) : 0
    return {
      ...plan,
      lotNo: batch.lotNo,
      projectName: batch.projectName,
      customerName: batch.customerName,
      thickness: batch.thickness,
      color: batch.color,
      orderQty: batch.orderQty,
      orderArea: batch.orderArea,
      itemQty: batch.itemQty,
      itemArea: batch.itemArea,
      labelQty: labels.length,
      jobName: job?.name ?? null,
      targetJobId: target.id,
      targetJobName: target.name,
      doneQty,
      doneArea: Number(doneArea.toFixed(4)),
      progress,
      lastScanAt,
      overdue: today > plan.endDate && progress < 100 && plan.stopState !== 'stopped',
    }
  }
}

const optStr = (b: Record<string, unknown>, key: string, max: number) =>
  b[key] === undefined || (typeof b[key] === 'string' && b[key].trim().length <= max)

/** Validates PlanCreate / PlanUpdate field types; returns an error message or null. */
function planFieldError(b: Record<string, unknown>): string | null {
  if (typeof b.startDate !== 'string' || typeof b.endDate !== 'string') return '请填写开始和结束日期'
  if (Number.isNaN(dateStart(b.startDate)) || Number.isNaN(dateStart(b.endDate))) return '日期格式不合法'
  if (b.endDate < b.startDate) return '结束日期不能早于开始日期'
  if (
    !optStr(b, 'workshop', 50) ||
    !optStr(b, 'paintLine', 50) ||
    !optStr(b, 'shift', 20) ||
    !optStr(b, 'sprayType', 50) ||
    !optStr(b, 'remark', 500)
  )
    return '文本字段类型或长度不合法'
  if (b.urgency !== undefined && b.urgency !== 'normal' && b.urgency !== 'urgent') return '紧急程度不合法'
  if (b.jobId != null && !jobs.some((job) => job.id === b.jobId)) return '工序不存在'
  return null
}

const plansOverlap = (a: { startDate: string; endDate: string }, b: { startDate: string; endDate: string }) =>
  a.startDate <= b.endDate && b.startDate <= a.endDate

function duplicatePlan(db: DemoDB, candidate: DemoPlan) {
  return db.plans.some(
    (plan) =>
      plan.id !== candidate.id &&
      plan.kind === candidate.kind &&
      plan.batchId === candidate.batchId &&
      (candidate.kind !== 'sheet_daily' || plan.jobId === candidate.jobId) &&
      plan.stopState !== 'stopped' &&
      plansOverlap(plan, candidate),
  )
}

export async function demoFetch(req: Request): Promise<Response> {
  if (import.meta.env.MODE !== 'test') await new Promise((r) => setTimeout(r, 120))
  const url = new URL(req.url)
  const at = url.pathname.indexOf('/api/')
  const path = at >= 0 ? url.pathname.slice(at + 4) : url.pathname
  const route = `${req.method} ${path}`

  if (route === 'GET /health') return json(200, { status: 'ok', version: 'demo' })

  const db = load()
  const { factories, users } = db
  if (route === 'POST /auth/login') {
    const b = await body(req)
    if (!b || !str(b.username) || !str(b.password)) return fail(400, 'invalid_request', '请输入用户名和密码')
    const u = users.find((x) => x.username === b.username && x.password === b.password)
    if (!u?.enabled) return fail(401, 'invalid_credentials', '用户名或密码错误')
    return json(200, {
      token: `demo-token-${u.id}`,
      user: { id: u.id, username: u.username, displayName: u.displayName, staffNo: u.staffNo, isAdmin: u.isAdmin },
    })
  }

  const u = principal(req, db)
  if (!u) return fail(401, 'unauthorized', '请先登录')

  if (route === 'GET /auth/me') {
    return json(200, {
      user: { id: u.id, username: u.username, displayName: u.displayName, staffNo: u.staffNo, isAdmin: u.isAdmin },
      menus: menuTree(u),
    })
  }

  if (route === 'GET /factories')
    return json(
      200,
      factories.filter((f) => inScope(u, f.id)),
    )

  const factoryUpdate = /^PUT \/factories\/(\d+)$/.exec(route)
  const userRoute = /^(GET|PUT) \/users\/(\d+)(\/password)?$/.exec(route)
  if (
    route === 'POST /factories' ||
    factoryUpdate ||
    route === 'GET /menus' ||
    route === 'GET /users' ||
    route === 'POST /users' ||
    userRoute
  ) {
    if (!u.isAdmin) return fail(403, 'forbidden', '仅管理员可操作')
  }

  if (route === 'POST /factories') {
    const b = await body(req)
    const code = str(b?.code)
    const name = str(b?.name)
    if (!code || !name || (b?.kind !== 'own' && b?.kind !== 'outsource')) return fail(400, 'invalid_request', '厂区编码、名称或类型不合法')
    if (factories.some((f) => f.code === code)) return fail(409, 'duplicate_code', '厂区编码已存在')
    const factory: S['Factory'] = { id: Math.max(0, ...factories.map((f) => f.id)) + 1, code, name, kind: b.kind, isStop: false }
    factories.push(factory)
    save(db)
    return json(201, factory)
  }

  if (factoryUpdate) {
    const factory = factories.find((f) => f.id === Number(factoryUpdate[1]))
    if (!factory) return fail(404, 'not_found', '厂区不存在')
    const b = await body(req)
    if (!str(b?.name) || typeof b?.isStop !== 'boolean') return fail(400, 'invalid_request', '厂区名称或停用状态不合法')
    factory.name = str(b.name)
    factory.isStop = b.isStop
    save(db)
    return json(200, factory)
  }

  if (route === 'GET /menus') return json(200, menuTree(u))

  if (route === 'GET /users') {
    const page = Number(url.searchParams.get('page') ?? 1)
    const size = Number(url.searchParams.get('size') ?? 20)
    if (!validPage(page, size)) return fail(400, 'invalid_request', 'page/size 参数不合法')
    const enabled = url.searchParams.get('enabled')
    if (enabled !== null && enabled !== 'true' && enabled !== 'false') return fail(400, 'invalid_request', 'enabled 参数不合法')
    const kw = (url.searchParams.get('keyword') ?? '').trim().toLowerCase()
    const all = users
      .filter((x) => !kw || [x.username, x.displayName, x.staffNo].some((s) => s.toLowerCase().includes(kw)))
      .filter((x) => enabled === null || x.enabled === (enabled === 'true'))
      .sort((a, b) => b.createdAt.localeCompare(a.createdAt) || b.id - a.id)
    return json(200, { items: all.slice((page - 1) * size, page * size).map(userDetail), total: all.length })
  }

  if (route === 'POST /users') {
    const b = await body(req)
    const username = str(b?.username)
    const displayName = str(b?.displayName)
    if (
      !/^[a-zA-Z0-9_.-]{2,32}$/.test(username) ||
      !displayName ||
      typeof b?.password !== 'string' ||
      b.password.length < 6 ||
      (b.staffNo !== undefined && (typeof b.staffNo !== 'string' || b.staffNo.length > 20)) ||
      (b.isAdmin !== undefined && typeof b.isAdmin !== 'boolean') ||
      (b.factoryIds !== undefined && !validIds(b.factoryIds)) ||
      (b.menuIds !== undefined && !validIds(b.menuIds))
    )
      return fail(400, 'invalid_request', '用户参数不合法')
    const isAdmin = b.isAdmin === true
    const factoryIds = (b.factoryIds ?? []) as number[]
    const menuIds = (b.menuIds ?? []) as number[]
    if (!isAdmin && !factoryIds.length) return fail(400, 'invalid_request', '非管理员用户至少需要一个厂区')
    if (!validReferences(db, factoryIds, menuIds)) return fail(400, 'invalid_reference', '厂区或菜单不存在')
    if (users.some((x) => x.username === username)) return fail(409, 'duplicate_code', '用户名已存在')
    const created: DemoUser = {
      id: Math.max(0, ...users.map((x) => x.id)) + 1,
      username,
      displayName,
      staffNo: str(b.staffNo),
      password: b.password,
      isAdmin,
      enabled: true,
      factoryIds: [...new Set(factoryIds)],
      menuIds: [...new Set(menuIds)],
      createdAt: new Date().toISOString(),
    }
    users.push(created)
    save(db)
    return json(201, userDetail(created))
  }

  if (userRoute) {
    const target = users.find((x) => x.id === Number(userRoute[2]))
    if (!target) return fail(404, 'not_found', '用户不存在')
    if (userRoute[1] === 'GET' && !userRoute[3]) return json(200, userDetail(target))
    if (userRoute[1] === 'PUT' && userRoute[3] === '/password') {
      const b = await body(req)
      if (typeof b?.password !== 'string' || b.password.length < 6) return fail(400, 'invalid_request', '密码至少需要6位')
      target.password = b.password
      save(db)
      return new Response(null, { status: 204 })
    }
    if (userRoute[1] === 'PUT' && !userRoute[3]) {
      const b = await body(req)
      if (
        !b ||
        !str(b.displayName) ||
        (b.staffNo !== undefined && (typeof b.staffNo !== 'string' || b.staffNo.length > 20)) ||
        typeof b?.enabled !== 'boolean' ||
        typeof b.isAdmin !== 'boolean' ||
        !validIds(b.factoryIds) ||
        !validIds(b.menuIds)
      )
        return fail(400, 'invalid_request', '用户参数不合法')
      if (target.id === u.id && (!b.enabled || !b.isAdmin)) return fail(409, 'self_lock', '不能停用自己或取消自己的管理员权限')
      if (!b.isAdmin && !b.factoryIds.length) return fail(400, 'invalid_request', '非管理员用户至少需要一个厂区')
      if (!validReferences(db, b.factoryIds, b.menuIds)) return fail(400, 'invalid_reference', '厂区或菜单不存在')
      target.displayName = str(b.displayName)
      target.staffNo = str(b.staffNo)
      target.enabled = b.enabled
      target.isAdmin = b.isAdmin
      target.factoryIds = [...new Set(b.factoryIds)]
      target.menuIds = [...new Set(b.menuIds)]
      save(db)
      return json(200, userDetail(target))
    }
  }

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

  if (route === 'GET /dashboard') {
    const rawDays = url.searchParams.get('days')
    const days = rawDays === null ? 14 : Number(rawDays)
    if (rawDays !== null && (!/^\d+$/.test(rawDays) || !Number.isInteger(days) || days < 7 || days > 90))
      return fail(400, 'invalid_request', 'days 参数不合法')

    const today = new Date().toISOString().slice(0, 10)
    const startMs = Date.parse(`${today}T00:00:00Z`) - (days - 1) * 86400000
    const start = new Date(startMs).toISOString().slice(0, 10)
    const batches = db.batches.filter((batch) => inScope(u, batch.factoryId))
    const batchById = new Map(batches.map((batch) => [batch.id, batch]))
    const labels = db.labels.filter((label) => batchById.has(label.batchId))
    const labelById = new Map(labels.map((label) => [label.id, label]))
    const scans = db.scans.filter((scan) => batchById.has(scan.batchId))
    const notes = db.deliveryNotes.filter((note) => batchById.has(note.batchId))
    const area = (values: number[]) => Number(values.reduce((sum, value) => sum + value, 0).toFixed(4))
    const open = batches.filter((batch) => batch.status !== 'done')
    const producing = batches.filter((batch) => batch.status === 'in_production')
    const todayScans = scans.filter((scan) => scan.scannedAt.slice(0, 10) === today)
    const windowNotes = notes.filter((note) => note.deliveredAt >= start && note.deliveredAt <= today)
    const unaccounted = notes.filter((note) => !note.accounted)
    const batchStatus: S['StatusCount'][] = (['pending', 'scheduled', 'in_production', 'done'] as const).map((status) => ({
      status,
      count: batches.filter((batch) => batch.status === status).length,
    }))
    const producingIds = new Set(producing.map((batch) => batch.id))
    const now = Date.now()
    const hours = (durationMs: number) => durationMs / 3600000
    const roundHours = (value: number) => Number(value.toFixed(2))
    const jobBySeq = new Map(jobs.map((job) => [job.seq, job]))
    const cycleByJob = new Map<number, number[]>()
    const latestByLabelJob = new Map<number, Map<number, number>>()
    for (const scan of [...scans].sort((a, b) => a.scannedAt.localeCompare(b.scannedAt) || a.id - b.id)) {
      const scannedAt = Date.parse(scan.scannedAt)
      const job = jobs.find((row) => row.id === scan.jobId)
      if (!job) continue
      const previous =
        job.seq === 1
          ? Date.parse(labelById.get(scan.labelId)?.createdAt ?? '')
          : latestByLabelJob.get(scan.labelId)?.get(jobBySeq.get(job.seq - 1)?.id ?? -1)
      if (
        scan.scannedAt.slice(0, 10) >= start &&
        scan.scannedAt.slice(0, 10) <= today &&
        previous !== undefined &&
        Number.isFinite(previous) &&
        scannedAt >= previous
      ) {
        const samples = cycleByJob.get(job.id) ?? []
        samples.push(hours(scannedAt - previous))
        cycleByJob.set(job.id, samples)
      }
      const byJob = latestByLabelJob.get(scan.labelId) ?? new Map<number, number>()
      byJob.set(job.id, scannedAt)
      latestByLabelJob.set(scan.labelId, byJob)
    }
    const jobLoad: S['JobLoad'][] = [...jobs]
      .sort((a, b) => a.seq - b.seq)
      .map((job) => {
        const rows = labels.filter((label) => !label.cancelled && producingIds.has(label.batchId) && label.lastJobId === job.id)
        const cycles = (cycleByJob.get(job.id) ?? []).sort((a, b) => a - b)
        const waits = rows.map((label) => hours(now - Date.parse(label.lastScannedAt!)))
        return {
          jobId: job.id,
          jobCode: job.code,
          jobName: job.name,
          seq: job.seq,
          qty: rows.length,
          area: area(rows.map((l) => l.pieceArea)),
          cycleSamples: cycles.length,
          cycleHours: cycles.length ? roundHours(cycles.reduce((sum, value) => sum + value, 0) / cycles.length) : null,
          cycleP90Hours: cycles.length ? roundHours(cycles[Math.ceil(0.9 * cycles.length) - 1]!) : null,
          dwellHours: waits.length ? roundHours(waits.reduce((sum, value) => sum + value, 0) / waits.length) : null,
          dwellMaxHours: waits.length ? roundHours(Math.max(...waits)) : null,
        }
      })
    const scanTrend: S['DailyScan'][] = Array.from({ length: days }, (_, index) => {
      const date = new Date(startMs + index * 86400000).toISOString().slice(0, 10)
      const rows = scans.filter((scan) => scan.scannedAt.slice(0, 10) === date)
      const dailyNotes = notes.filter((note) => note.deliveredAt === date)
      return {
        date,
        qty: rows.length,
        area: area(rows.map((scan) => labelById.get(scan.labelId)?.pieceArea ?? 0)),
        deliveredQty: dailyNotes.reduce((sum, note) => sum + note.qty, 0),
        deliveredArea: area(dailyNotes.map((note) => note.area)),
        activeBatches: new Set(rows.map((scan) => scan.batchId)).size,
        doneBatches: batches.filter((batch) => batch.status === 'done' && batch.updatedAt.slice(0, 10) === date).length,
      }
    })
    const byCustomer = new Map<number, S['CustomerArea']>()
    for (const batch of open) {
      const row = byCustomer.get(batch.customerId) ?? {
        customerId: batch.customerId,
        customerName: db.customers.find((customer) => customer.id === batch.customerId)?.name ?? batch.customerName,
        area: 0,
        batches: 0,
      }
      row.area += batch.orderArea
      row.batches++
      byCustomer.set(batch.customerId, row)
    }
    const ranked = [...byCustomer.values()]
      .map((row) => ({ ...row, area: area([row.area]) }))
      .sort((a, b) => b.area - a.area || (a.customerId ?? 0) - (b.customerId ?? 0))
    const customerArea: S['CustomerArea'][] = ranked.slice(0, 6)
    if (ranked.length > 6) {
      const rest = ranked.slice(6)
      customerArea.push({
        customerId: null,
        customerName: '其他',
        area: area(rest.map((row) => row.area)),
        batches: rest.reduce((sum, row) => sum + row.batches, 0),
      })
    }
    const activeBatches: S['ActiveBatch'][] = producing
      .sort((a, b) => (a.requiredDate ?? '9999-12-31').localeCompare(b.requiredDate ?? '9999-12-31') || a.id - b.id)
      .slice(0, 8)
      .map((batch) => {
        const rows = labels.filter((label) => label.batchId === batch.id && !label.cancelled)
        const batchScans = scans.filter((scan) => scan.batchId === batch.id)
        const currentJob = jobs.filter((job) => batchScans.some((scan) => scan.jobId === job.id)).sort((a, b) => b.seq - a.seq)[0]
        return {
          id: batch.id,
          lotNo: batch.lotNo,
          projectName: batch.projectName,
          customerName: batch.customerName,
          thickness: batch.thickness,
          color: batch.color,
          itemQty: batch.itemQty,
          itemArea: batch.itemArea,
          labelQty: rows.length,
          currentSeq: currentJob?.seq ?? 0,
          currentJobName: currentJob?.name ?? '',
          progress: rows.length ? Math.floor((batchScans.length * 100) / (rows.length * jobs.length)) : 0,
          requiredDate: batch.requiredDate,
          urgency: batch.urgency,
        }
      })
    const activityGroups = new Map<string, S['ScanActivity']>()
    for (const scan of scans) {
      const bucket = Math.floor(Date.parse(scan.scannedAt) / 600000)
      const key = JSON.stringify([scan.batchId, scan.jobId, scan.scannedBy, bucket])
      const row = activityGroups.get(key)
      if (row) {
        row.qty++
        row.area += labelById.get(scan.labelId)?.pieceArea ?? 0
        if (scan.scannedAt > row.lastScannedAt) row.lastScannedAt = scan.scannedAt
      } else {
        activityGroups.set(key, {
          batchId: scan.batchId,
          lotNo: scan.lotNo,
          jobId: scan.jobId,
          jobName: scan.jobName,
          scannedBy: scan.scannedBy,
          staffNo:
            operators.find((operator) => operator.name === scan.scannedBy)?.staffNo ??
            users.find((user) => user.displayName === scan.scannedBy)?.staffNo ??
            '',
          qty: 1,
          area: labelById.get(scan.labelId)?.pieceArea ?? 0,
          lastScannedAt: scan.scannedAt,
        })
      }
    }
    const recentActivity = [...activityGroups.values()]
      .sort((a, b) => b.lastScannedAt.localeCompare(a.lastScannedAt))
      .slice(0, 8)
      .map((row) => ({ ...row, area: area([row.area]) }))
    const todayMs = Date.parse(`${today}T00:00:00Z`)
    const alerts: { value: S['DashboardAlert']; age: number }[] = []
    const addAlert = (value: S['DashboardAlert'], age = 0) => alerts.push({ value, age })
    const batchLink = (lotNo: string) => `/scheduling/batches?keyword=${encodeURIComponent(lotNo)}`
    for (const batch of open) {
      const lastScan = scans.filter((scan) => scan.batchId === batch.id).sort((a, b) => b.scannedAt.localeCompare(a.scannedAt))[0]
      const currentJob = lastScan?.jobName ?? '未开工'
      if (batch.requiredDate) {
        const remaining = Math.round((Date.parse(`${batch.requiredDate}T00:00:00Z`) - todayMs) / 86400000)
        if (remaining < 0) {
          const days = -remaining
          addAlert(
            {
              rule: 'overdue',
              level: 'error',
              title: `批次 ${batch.lotNo} 已逾期 ${days} 天`,
              detail: `${batch.projectName} · 需求交期 ${batch.requiredDate} · 当前工序 ${currentJob}`,
              link: batchLink(batch.lotNo),
              lotNo: batch.lotNo,
              days,
            },
            days,
          )
        } else if (remaining <= 3) {
          addAlert(
            {
              rule: 'due_soon',
              level: 'warning',
              title: `批次 ${batch.lotNo} ${remaining === 0 ? '今天到期' : `${remaining} 天后到期`}`,
              detail: `${batch.projectName} · 需求交期 ${batch.requiredDate} · 当前工序 ${currentJob}`,
              link: batchLink(batch.lotNo),
              lotNo: batch.lotNo,
              days: remaining,
            },
            remaining,
          )
        }
      }
      if (batch.status === 'scheduled') {
        const hasScan = scans.some((scan) => scan.batchId === batch.id)
        if (batch.planSheetStart && batch.planSheetStart < today && !hasScan)
          addAlert(
            {
              rule: 'idle',
              level: 'warning',
              title: `批次 ${batch.lotNo} 计划 ${batch.planSheetStart} 开工但未报工`,
              detail: `${batch.projectName} · 计划开工 ${batch.planSheetStart}`,
              link: batchLink(batch.lotNo),
              lotNo: batch.lotNo,
            },
            Math.round((todayMs - Date.parse(`${batch.planSheetStart}T00:00:00Z`)) / 86400000),
          )
        if (batch.itemCount > 0 && !labels.some((label) => label.batchId === batch.id) && batch.planCutDate && batch.planCutDate < today)
          addAlert(
            {
              rule: 'no_labels',
              level: 'info',
              title: `批次 ${batch.lotNo} 已过开料日期但未生成标签`,
              detail: `${batch.projectName} · 计划开料 ${batch.planCutDate}`,
              link: batchLink(batch.lotNo),
              lotNo: batch.lotNo,
            },
            Math.round((todayMs - Date.parse(`${batch.planCutDate}T00:00:00Z`)) / 86400000),
          )
      }
    }
    const stalledByJob = new Map<number, { rows: S['Label'][]; maxHours: number }>()
    for (const label of labels) {
      const lastJobId = label.lastJobId
      if (label.cancelled || !producingIds.has(label.batchId) || lastJobId == null || !label.lastScannedAt) continue
      const wait = hours(now - Date.parse(label.lastScannedAt))
      if (wait <= 24) continue
      const group = stalledByJob.get(lastJobId) ?? { rows: [], maxHours: 0 }
      group.rows.push(label)
      group.maxHours = Math.max(group.maxHours, wait)
      stalledByJob.set(lastJobId, group)
    }
    for (const [jobId, group] of stalledByJob) {
      const jobName = jobs.find((job) => job.id === jobId)?.name ?? group.rows[0]!.lastJobName ?? ''
      const fmt = group.maxHours < 48 ? `${Math.floor(group.maxHours)}h` : `${(group.maxHours / 24).toFixed(1)}d`
      const lotNos = [...new Set(group.rows.map((label) => label.lotNo))]
      addAlert(
        {
          rule: 'stalled',
          level: group.maxHours > 72 ? 'error' : 'warning',
          title: `${jobName} 有 ${group.rows.length} 件停留超 ${fmt}`,
          detail: `涉及批次 ${lotNos.slice(0, 3).join('、')}${lotNos.length > 3 ? ' 等' : ''}`,
          link: `/production/labels?jobId=${jobId}`,
          jobId,
          count: group.rows.length,
          hours: roundHours(group.maxHours),
        },
        group.maxHours,
      )
    }
    const oldNotes = unaccounted.filter((note) => Date.parse(`${note.deliveredAt}T00:00:00Z`) < now - 7 * 86400000)
    if (oldNotes.length) {
      const earliest = oldNotes.map((note) => note.deliveredAt).sort()[0]!
      const age = hours(now - Date.parse(`${earliest}T00:00:00Z`))
      addAlert(
        {
          rule: 'unaccounted',
          level: 'warning',
          title: `${oldNotes.length} 张发货单超 7 天未对账`,
          detail: `最早 ${earliest}，合计 ${area(oldNotes.map((note) => note.area))} m²`,
          link: '/finance/deliver-accounts',
          count: oldNotes.length,
          hours: roundHours(age),
        },
        age,
      )
    }
    const severity = { error: 3, warning: 2, info: 1 }
    const sortedAlerts = alerts
      .sort((a, b) => severity[b.value.level] - severity[a.value.level] || b.age - a.age || a.value.title.localeCompare(b.value.title))
      .slice(0, 20)
      .map((entry) => entry.value)
    return json(200, {
      generatedAt: new Date().toISOString(),
      kpi: {
        inProduction: producing.length,
        pendingSchedule: batchStatus[0]!.count,
        doneToday: batches.filter((batch) => batch.status === 'done' && batch.updatedAt.slice(0, 10) === today).length,
        todayScanQty: todayScans.length,
        todayScanArea: area(todayScans.map((scan) => labelById.get(scan.labelId)?.pieceArea ?? 0)),
        labelQty: labels.filter((label) => !label.cancelled && batchById.get(label.batchId)?.status !== 'done').length,
        deliveredQty: windowNotes.reduce((sum, note) => sum + note.qty, 0),
        deliveredArea: area(windowNotes.map((note) => note.area)),
        unaccountedNotes: unaccounted.length,
        unaccountedArea: area(unaccounted.map((note) => note.area)),
        urgentOpen: open.filter((batch) => batch.urgency === 'urgent').length,
        overdueOpen: open.filter((batch) => batch.requiredDate !== null && batch.requiredDate < today).length,
      },
      batchStatus,
      jobLoad,
      scanTrend,
      customerArea,
      activeBatches,
      recentScans: [...scans].sort((a, b) => b.scannedAt.localeCompare(a.scannedAt) || b.id - a.id).slice(0, 8),
      recentActivity,
      alerts: sortedAlerts,
    } satisfies S['Dashboard'])
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

  if (route === 'GET /batch-valuations') {
    const page = Number(url.searchParams.get('page') ?? 1)
    const size = Number(url.searchParams.get('size') ?? 20)
    if (!validPage(page, size)) return fail(400, 'invalid_request', 'page/size 参数不合法')
    const contractId = url.searchParams.get('contractId')
    const customerId = url.searchParams.get('customerId')
    if (!validId(contractId) || !validId(customerId)) return fail(400, 'invalid_request', '筛选参数不合法')
    const audited = url.searchParams.get('audited')
    if (audited !== null && audited !== 'true' && audited !== 'false') return fail(400, 'invalid_request', '审核参数不合法')
    const begin = url.searchParams.get('beginDate')
    const end = url.searchParams.get('endDate')
    if (
      (begin !== null && Number.isNaN(dateStart(begin))) ||
      (end !== null && Number.isNaN(dateStart(end))) ||
      (begin && end && begin > end)
    )
      return fail(400, 'invalid_request', '日期范围无效')
    const kw = (url.searchParams.get('keyword') ?? '').trim().toLowerCase()
    const all = db.batches
      .filter((batch) => inScope(u, batch.factoryId))
      .filter(
        (batch) => !kw || [batch.lotNo, batch.contractNo, batch.projectName, batch.customerName].some((s) => s.toLowerCase().includes(kw)),
      )
      .filter((batch) => contractId === null || batch.contractId === Number(contractId))
      .filter((batch) => customerId === null || batch.customerId === Number(customerId))
      .filter((batch) => (!begin || batch.createdAt.slice(0, 10) >= begin) && (!end || batch.createdAt.slice(0, 10) <= end))
      .filter(
        (batch) =>
          audited === null || (db.batchValuations.find((row) => row.batchId === batch.id)?.audited ?? false) === (audited === 'true'),
      )
      .sort((a, b) => b.createdAt.localeCompare(a.createdAt) || b.id - a.id)
    const rows = all.map((batch) => valuationRow(db, batch))
    return json(200, {
      items: rows.slice((page - 1) * size, page * size),
      total: rows.length,
      totalAmount: Math.round(rows.reduce((sum, row) => sum + row.amount, 0) * 100) / 100,
    } satisfies S['BatchValuationPage'])
  }

  const valuationRoute = /^(PUT|POST) \/batches\/(\d+)\/valuation(\/audit)?$/.exec(route)
  if (valuationRoute) {
    const batch = db.batches.find((x) => x.id === Number(valuationRoute[2]) && inScope(u, x.factoryId))
    if (!batch) return fail(404, 'not_found', '批次不存在')
    const existing = db.batchValuations.find((row) => row.batchId === batch.id)
    if (valuationRoute[1] === 'PUT' && !valuationRoute[3]) {
      if (existing?.audited) return fail(409, 'already_audited', '计价已审核')
      const b = await body(req)
      if (
        !b ||
        Array.isArray(b) ||
        !['order_area', 'item_area', 'delivered_area'].includes(b.basis as string) ||
        typeof b.unitPrice !== 'number' ||
        !Number.isFinite(b.unitPrice) ||
        b.unitPrice < 0 ||
        (b.ingotPrice !== undefined && (typeof b.ingotPrice !== 'number' || !Number.isFinite(b.ingotPrice) || b.ingotPrice < 0)) ||
        (b.remark !== undefined && (typeof b.remark !== 'string' || b.remark.length > 500))
      )
        return fail(400, 'invalid_request', '计价参数不合法')
      const basis = b.basis as S['ValuationBasis']
      const basisArea =
        basis === 'order_area' ? batch.orderArea : basis === 'item_area' ? batch.itemArea : valuationRow(db, batch).deliveredArea
      const now = new Date().toISOString()
      const value: DemoBatchValuation = {
        batchId: batch.id,
        basis,
        basisArea,
        unitPrice: b.unitPrice,
        ingotPrice: (b.ingotPrice ?? 0) as number,
        amount: Math.round(basisArea * b.unitPrice * 100) / 100,
        remark: b.remark === undefined ? '' : (b.remark as string),
        audited: false,
        auditedBy: null,
        auditedAt: null,
        updatedBy: u.displayName,
        updatedAt: now,
      }
      if (existing) Object.assign(existing, value)
      else db.batchValuations.push(value)
      save(db)
      return json(200, valuationRow(db, batch))
    }
    if (valuationRoute[1] === 'POST' && valuationRoute[3] === '/audit') {
      if (!existing) return fail(404, 'not_found', '该批次尚未计价')
      if (existing.audited) return fail(409, 'already_audited', '计价已审核')
      existing.audited = true
      existing.auditedBy = u.displayName
      existing.auditedAt = new Date().toISOString()
      save(db)
      return json(200, valuationRow(db, batch))
    }
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
    const kw = (url.searchParams.get('keyword') ?? '').trim().toLowerCase()
    const staff = (url.searchParams.get('staff') ?? '').trim().toLowerCase()
    const staffUsers = db.users.filter((x) => [x.displayName, x.staffNo, x.username].some((s) => s.toLowerCase().includes(staff)))
    const scopedBatches = new Set(db.batches.filter((batch) => inScope(u, batch.factoryId)).map((batch) => batch.id))
    const all = db.scans
      .filter((scan) => scopedBatches.has(scan.batchId))
      .filter((scan) => batchId === null || scan.batchId === Number(batchId))
      .filter((scan) => jobId === null || scan.jobId === Number(jobId))
      .filter((scan) => begin === null || Date.parse(scan.scannedAt) >= begin)
      .filter((scan) => end === null || Date.parse(scan.scannedAt) < end)
      .filter((scan) => !kw || [scan.barCode, scan.lotNo, scan.projectName, scan.pieceNo].some((s) => s.toLowerCase().includes(kw)))
      .filter(
        (scan) =>
          !staff ||
          scan.scannedBy.toLowerCase().includes(staff) ||
          scan.staffNo.toLowerCase().includes(staff) ||
          staffUsers.some((x) => x.displayName === scan.scannedBy),
      )
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
      projectName: label.projectName,
      pieceNo: label.pieceNo,
      pieceIndex: label.pieceIndex,
      pieceArea: label.pieceArea,
      jobId: job.id,
      jobName: job.name,
      scannedBy: u.displayName,
      staffNo: u.staffNo,
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
        projectName: label.projectName,
        pieceNo: label.pieceNo,
        pieceIndex: label.pieceIndex,
        pieceArea: label.pieceArea,
        jobId: job60.id,
        jobName: job60.name,
        scannedBy: u.displayName,
        staffNo: u.staffNo,
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

  if (route === 'GET /reports/batch-jobs') {
    const q = url.searchParams
    const page = Number(q.get('page') ?? 1)
    const size = Number(q.get('size') ?? 20)
    if (!validPage(page, size)) return fail(400, 'invalid_request', 'page/size 参数不合法')
    const mode = q.get('mode')
    if (mode !== 'daily' && mode !== 'progress') return fail(400, 'invalid_request', '报表类型不合法')
    const today = new Date().toISOString().slice(0, 10)
    const from = q.get('from') ?? (mode === 'daily' ? today : null)
    const to = q.get('to') ?? (mode === 'daily' ? today : null)
    if ((from !== null && Number.isNaN(dateStart(from))) || (to !== null && Number.isNaN(dateStart(to))) || (from && to && from > to))
      return fail(400, 'invalid_request', '日期范围无效')
    const status = q.get('status')
    if (status !== null && !['pending', 'scheduled', 'in_production', 'done'].includes(status))
      return fail(400, 'invalid_request', '状态参数不合法')
    const jobId = q.get('jobId')
    const outsourceFactoryId = q.get('outsourceFactoryId')
    if (!validId(jobId) || !validId(outsourceFactoryId)) return fail(400, 'invalid_request', '参数不合法')
    const scanned = q.get('scanned')
    if (scanned !== null && scanned !== 'true' && scanned !== 'false') return fail(400, 'invalid_request', '扫描参数不合法')
    const workshop = q.get('workshop')
    const kw = (q.get('keyword') ?? '').trim().toLowerCase()
    const daily = mode === 'daily'
    const inWindow = (at: string) => {
      if (!daily) return true
      const d = at.slice(0, 10)
      return (!from || d >= from) && (!to || d <= to)
    }
    const activeLabels = new Map(db.labels.filter((label) => !label.cancelled).map((label) => [label.id, label]))
    const counted = db.scans.filter((scan) => activeLabels.has(scan.labelId) && inWindow(scan.scannedAt))
    const byBatch = new Map<number, S['Scan'][]>()
    for (const scan of counted) byBatch.set(scan.batchId, [...(byBatch.get(scan.batchId) ?? []), scan])
    const jobSeq = new Map(jobs.map((job) => [job.id, job.seq]))
    const rowsAll = db.batches
      .filter((batch) => inScope(u, batch.factoryId))
      .filter((batch) => daily || ((!from || batch.createdAt.slice(0, 10) >= from) && (!to || batch.createdAt.slice(0, 10) <= to)))
      .filter((batch) => !daily || byBatch.has(batch.id))
      .filter((batch) => status === null || batch.status === status)
      .filter((batch) => !workshop || batch.workshop === workshop)
      .filter((batch) => outsourceFactoryId === null || batch.outsourceFactoryId === Number(outsourceFactoryId))
      .filter((batch) => jobId === null || (byBatch.get(batch.id) ?? []).some((scan) => scan.jobId === Number(jobId)))
      .filter((batch) => scanned === null || byBatch.has(batch.id) === (scanned === 'true'))
      .filter(
        (batch) =>
          !kw || [batch.lotNo, batch.projectName, batch.customerName, batch.clientOrderNo].some((s) => s.toLowerCase().includes(kw)),
      )
      .map((batch): S['BatchJobRow'] => {
        const labelQty = [...activeLabels.values()].filter((label) => label.batchId === batch.id).length
        const scans = byBatch.get(batch.id) ?? []
        const allScans = db.scans.filter((scan) => scan.batchId === batch.id && activeLabels.has(scan.labelId))
        const furthest = allScans.reduce<S['Scan'] | null>(
          (best, scan) => (!best || jobSeq.get(scan.jobId)! > jobSeq.get(best.jobId)! ? scan : best),
          null,
        )
        const cells = jobs.map((job): S['JobCell'] => {
          const hits = scans.filter((scan) => scan.jobId === job.id)
          const area = hits.reduce((sum, scan) => sum + activeLabels.get(scan.labelId)!.pieceArea, 0)
          return {
            jobId: job.id,
            qty: hits.length,
            area: Number(area.toFixed(4)),
            pct: labelQty ? Math.floor((hits.length * 100) / labelQty) : 0,
          }
        })
        return {
          batchId: batch.id,
          lotNo: batch.lotNo,
          projectName: batch.projectName,
          customerName: batch.customerName,
          clientOrderNo: batch.clientOrderNo,
          thickness: batch.thickness,
          color: batch.color,
          status: batch.status,
          urgency: batch.urgency,
          workshop: batch.workshop,
          outsourceFactoryName: batch.outsourceFactoryName ?? null,
          requiredDate: batch.requiredDate ?? null,
          itemQty: batch.itemQty,
          itemArea: batch.itemArea,
          labelQty,
          currentJobName: furthest?.jobName ?? '',
          lastScanAt: scans.reduce<string | null>((max, scan) => (max === null || scan.scannedAt > max ? scan.scannedAt : max), null),
          cells,
        }
      })
    rowsAll.sort((a, b) =>
      daily
        ? (b.lastScanAt ?? '').localeCompare(a.lastScanAt ?? '') || b.batchId - a.batchId
        : (a.requiredDate ?? '9999').localeCompare(b.requiredDate ?? '9999') || b.batchId - a.batchId,
    )
    const labelQty = rowsAll.reduce((sum, row) => sum + row.labelQty, 0)
    const summaryCells = jobs.map((job, k): S['JobCell'] => {
      const qty = rowsAll.reduce((sum, row) => sum + row.cells[k]!.qty, 0)
      const area = rowsAll.reduce((sum, row) => sum + row.cells[k]!.area, 0)
      return { jobId: job.id, qty, area: Number(area.toFixed(4)), pct: labelQty ? Math.floor((qty * 100) / labelQty) : 0 }
    })
    return json(200, {
      jobs,
      items: rowsAll.slice((page - 1) * size, page * size),
      total: rowsAll.length,
      page,
      size,
      summary: {
        batches: rowsAll.length,
        itemQty: rowsAll.reduce((sum, row) => sum + row.itemQty, 0),
        itemArea: Number(rowsAll.reduce((sum, row) => sum + row.itemArea, 0).toFixed(4)),
        labelQty,
        cells: summaryCells,
      },
    } satisfies S['BatchJobReport'])
  }

  if (route === 'GET /reports/staff-output') {
    const q = url.searchParams
    const page = Number(q.get('page') ?? 1)
    const size = Number(q.get('size') ?? 20)
    if (!validPage(page, size)) return fail(400, 'invalid_request', 'page/size 参数不合法')
    const today = new Date().toISOString().slice(0, 10)
    const from = q.get('from') ?? today
    const to = q.get('to') ?? today
    if (Number.isNaN(dateStart(from)) || Number.isNaN(dateStart(to)) || from > to) return fail(400, 'invalid_request', '日期范围无效')
    const jobId = q.get('jobId')
    if (!validId(jobId)) return fail(400, 'invalid_request', '工序参数不合法')
    const detail = q.get('detail')
    if (detail !== null && detail !== 'true' && detail !== 'false') return fail(400, 'invalid_request', 'detail 参数不合法')
    const staffKw = (q.get('staff') ?? '').trim().toLowerCase()
    const batchKw = (q.get('batchKeyword') ?? '').trim().toLowerCase()
    const isDetail = detail === 'true'
    const batchById = new Map(db.batches.map((batch) => [batch.id, batch]))
    const activeLabels = new Map(db.labels.filter((label) => !label.cancelled).map((label) => [label.id, label]))
    const itemById = new Map(db.batchItems.map((item) => [item.id, item]))
    const operatorByName = new Map(operators.map((op, index) => [op.name, { id: 1000 + index, ...op }]))
    const groups = new Map<string, S['StaffOutputRow']>()
    for (const scan of db.scans) {
      const batch = batchById.get(scan.batchId)
      const label = activeLabels.get(scan.labelId)
      if (!batch || !label || !inScope(u, batch.factoryId)) continue
      const day = scan.scannedAt.slice(0, 10)
      if (day < from || day > to) continue
      if (jobId !== null && scan.jobId !== Number(jobId)) continue
      const op = operatorByName.get(scan.scannedBy) ?? { id: 0, name: scan.scannedBy, staffNo: '' }
      if (staffKw && ![op.name, op.staffNo].some((s) => s.toLowerCase().includes(staffKw))) continue
      if (batchKw && ![batch.lotNo, batch.projectName].some((s) => s.toLowerCase().includes(batchKw))) continue
      const item = itemById.get(label.batchItemId)
      const key = `${op.id}:${scan.jobId}${isDetail ? `:${batch.id}` : ''}`
      const row = groups.get(key) ?? {
        userId: op.id,
        staffNo: op.staffNo,
        displayName: op.name,
        jobId: scan.jobId,
        jobName: scan.jobName,
        batchId: isDetail ? batch.id : null,
        lotNo: isDetail ? batch.lotNo : null,
        projectName: isDetail ? batch.projectName : null,
        qty: 0,
        area: 0,
        sprayArea: 0,
        weldLine: 0,
        firstScanAt: scan.scannedAt,
        lastScanAt: scan.scannedAt,
      }
      row.qty++
      row.area += label.pieceArea
      row.sprayArea += item?.sprayingArea ?? 0
      row.weldLine += item?.weldLine ?? 0
      if (scan.scannedAt < row.firstScanAt) row.firstScanAt = scan.scannedAt
      if (scan.scannedAt > row.lastScanAt) row.lastScanAt = scan.scannedAt
      groups.set(key, row)
    }
    const jobSeq = new Map(jobs.map((job) => [job.id, job.seq]))
    const all = [...groups.values()]
      .map((row) => ({
        ...row,
        area: Number(row.area.toFixed(4)),
        sprayArea: Number(row.sprayArea.toFixed(4)),
        weldLine: Number(row.weldLine.toFixed(2)),
      }))
      .sort(
        (a, b) =>
          b.qty - a.qty ||
          a.staffNo.localeCompare(b.staffNo) ||
          a.userId - b.userId ||
          jobSeq.get(a.jobId)! - jobSeq.get(b.jobId)! ||
          (a.lotNo ?? '').localeCompare(b.lotNo ?? ''),
      )
    const sum = (pick: (row: S['StaffOutputRow']) => number) => all.reduce((acc, row) => acc + pick(row), 0)
    return json(200, {
      items: all.slice((page - 1) * size, page * size),
      total: all.length,
      page,
      size,
      summary: {
        rows: all.length,
        staff: new Set(all.map((row) => row.userId)).size,
        qty: sum((row) => row.qty),
        area: Number(sum((row) => row.area).toFixed(4)),
        sprayArea: Number(sum((row) => row.sprayArea).toFixed(4)),
        weldLine: Number(sum((row) => row.weldLine).toFixed(2)),
      },
    } satisfies S['StaffOutputReport'])
  }

  if (route === 'GET /plans') {
    const q = url.searchParams
    const page = Number(q.get('page') ?? 1)
    const size = Number(q.get('size') ?? 20)
    if (!validPage(page, size)) return fail(400, 'invalid_request', 'page/size 参数不合法')
    const kind = q.get('kind') as S['PlanKind'] | null
    if (!kind || !PLAN_KINDS.includes(kind)) return fail(400, 'invalid_request', '计划类型不合法')
    const from = q.get('from')
    const to = q.get('to')
    if ((from !== null && Number.isNaN(dateStart(from))) || (to !== null && Number.isNaN(dateStart(to))) || (from && to && from > to))
      return fail(400, 'invalid_request', '日期范围无效')
    const auditState = q.get('auditState')
    const stopState = q.get('stopState')
    const completed = q.get('completed')
    const jobId = q.get('jobId')
    if (auditState !== null && auditState !== 'draft' && auditState !== 'audited') return fail(400, 'invalid_request', '审核状态参数不合法')
    if (stopState !== null && !['normal', 'paused', 'stopped'].includes(stopState))
      return fail(400, 'invalid_request', '暂停状态参数不合法')
    if (completed !== null && completed !== 'true' && completed !== 'false') return fail(400, 'invalid_request', '完成参数不合法')
    if (!validId(jobId)) return fail(400, 'invalid_request', '工序参数不合法')
    const workshop = q.get('workshop')
    const paintLine = q.get('paintLine')
    const kw = (q.get('keyword') ?? '').trim().toLowerCase()
    const toRow = planRows(db)
    const all = db.plans
      .filter((plan) => plan.kind === kind && inScope(u, plan.factoryId))
      .filter((plan) => (!from || plan.endDate >= from) && (!to || plan.startDate <= to))
      .filter((plan) => auditState === null || plan.auditState === auditState)
      .filter((plan) => stopState === null || plan.stopState === stopState)
      .filter((plan) => !workshop || plan.workshop === workshop)
      .filter((plan) => !paintLine || plan.paintLine === paintLine)
      .filter((plan) => jobId === null || plan.jobId === Number(jobId))
      .map(toRow)
      .filter((row) => completed === null || (row.progress === 100) === (completed === 'true'))
      .filter((row) => !kw || [row.lotNo, row.projectName, row.customerName].some((s) => s.toLowerCase().includes(kw)))
      .sort((a, b) => b.startDate.localeCompare(a.startDate) || b.id - a.id)
    const sum = (pick: (row: S['Plan']) => number) => all.reduce((acc, row) => acc + pick(row), 0)
    return json(200, {
      items: all.slice((page - 1) * size, page * size),
      total: all.length,
      page,
      size,
      summary: {
        plans: all.length,
        qty: sum((row) => row.itemQty),
        area: Number(sum((row) => row.itemArea).toFixed(4)),
        doneQty: sum((row) => row.doneQty),
        doneArea: Number(sum((row) => row.doneArea).toFixed(4)),
        overdue: all.filter((row) => row.overdue).length,
      },
    } satisfies S['PlanPage'])
  }

  if (route === 'POST /plans') {
    const b = await body(req)
    if (!b || Array.isArray(b) || !PLAN_KINDS.includes(b.kind as S['PlanKind']) || !Number.isSafeInteger(b.batchId))
      return fail(400, 'invalid_request', '计划类型或批次不合法')
    const kind = b.kind as S['PlanKind']
    const fieldError = planFieldError(b)
    if (fieldError) return fail(400, 'invalid_request', fieldError)
    if (kind === 'sheet_daily' && b.jobId == null) return fail(400, 'invalid_request', '钣金日计划必须指定工序')
    if (kind === 'spray_daily' && (!str(b.paintLine) || !str(b.shift))) return fail(400, 'invalid_request', '喷涂日计划必须指定产线和班次')
    const batch = db.batches.find((x) => x.id === b.batchId)
    if (!batch) return fail(404, 'not_found', '批次不存在')
    if (!inScope(u, batch.factoryId)) return fail(403, 'forbidden_factory', '无权在该厂区创建数据')
    const now = new Date().toISOString()
    const plan: DemoPlan = {
      id: 0,
      kind,
      factoryId: batch.factoryId,
      batchId: batch.id,
      startDate: b.startDate as string,
      endDate: b.endDate as string,
      workshop: str(b.workshop),
      jobId: kind === 'sheet_daily' ? (b.jobId as number) : null,
      paintLine: str(b.paintLine),
      shift: str(b.shift),
      sprayType: str(b.sprayType) || (kind === 'spray_daily' ? batch.sprayingType : ''),
      urgency: (b.urgency ?? batch.urgency) as S['Urgency'],
      auditState: 'draft',
      auditedBy: null,
      auditedAt: null,
      stopState: 'normal',
      stopReason: '',
      remark: str(b.remark),
      createdBy: u.displayName,
      createdAt: now,
      updatedAt: now,
    }
    if (duplicatePlan(db, plan)) return fail(409, 'duplicate_plan', '该批次在此日期范围内已有计划')
    plan.id = db.nextId++
    db.plans.push(plan)
    save(db)
    return json(201, planRows(db)(plan))
  }

  const planRoute = /^(GET|PUT|DELETE|POST) \/plans\/(\d+)(?:\/(audit|stop-state))?$/.exec(route)
  if (planRoute) {
    const [, method, rawId, action] = planRoute
    const plan = db.plans.find((x) => x.id === Number(rawId) && inScope(u, x.factoryId))
    if (!plan) return fail(404, 'not_found', '计划不存在')
    if (method === 'GET' && !action) return json(200, planRows(db)(plan))
    if (method === 'DELETE' && !action) {
      if (plan.auditState === 'audited') return fail(409, 'already_audited', '计划已审核，不能删除')
      db.plans = db.plans.filter((x) => x.id !== plan.id)
      save(db)
      return new Response(null, { status: 204 })
    }
    if (method === 'PUT' && !action) {
      if (plan.auditState === 'audited') return fail(409, 'already_audited', '计划已审核，不能修改')
      const b = await body(req)
      if (!b || Array.isArray(b)) return fail(400, 'invalid_request', '请求参数不合法')
      const fieldError = planFieldError(b)
      if (fieldError) return fail(400, 'invalid_request', fieldError)
      if (plan.kind === 'sheet_daily' && b.jobId === null) return fail(400, 'invalid_request', '钣金日计划必须指定工序')
      if (plan.kind === 'spray_daily' && ((b.paintLine !== undefined && !str(b.paintLine)) || (b.shift !== undefined && !str(b.shift))))
        return fail(400, 'invalid_request', '喷涂日计划必须指定产线和班次')
      const next: DemoPlan = { ...plan, startDate: b.startDate as string, endDate: b.endDate as string }
      if (b.workshop !== undefined) next.workshop = str(b.workshop)
      if (plan.kind === 'sheet_daily' && b.jobId != null) next.jobId = b.jobId as number
      if (b.paintLine !== undefined) next.paintLine = str(b.paintLine)
      if (b.shift !== undefined) next.shift = str(b.shift)
      if (b.sprayType !== undefined) next.sprayType = str(b.sprayType)
      if (b.urgency !== undefined) next.urgency = b.urgency as S['Urgency']
      if (b.remark !== undefined) next.remark = str(b.remark)
      if (next.stopState !== 'stopped' && duplicatePlan(db, next)) return fail(409, 'duplicate_plan', '该批次在此日期范围内已有计划')
      Object.assign(plan, next, { updatedAt: new Date().toISOString() })
      save(db)
      return json(200, planRows(db)(plan))
    }
    if (method === 'POST' && action === 'audit') {
      const b = await body(req)
      if (b?.action !== 'audit' && b?.action !== 'unaudit') return fail(400, 'invalid_request', '审核动作不合法')
      const now = new Date().toISOString()
      if (b.action === 'audit') {
        if (plan.auditState === 'audited') return fail(409, 'already_audited', '计划已审核')
        Object.assign(plan, { auditState: 'audited', auditedBy: u.displayName, auditedAt: now, updatedAt: now })
      } else {
        if (plan.auditState !== 'audited') return fail(409, 'not_audited', '计划未审核')
        Object.assign(plan, { auditState: 'draft', auditedBy: null, auditedAt: null, updatedAt: now })
      }
      save(db)
      return json(200, planRows(db)(plan))
    }
    if (method === 'POST' && action === 'stop-state') {
      const b = await body(req)
      if (
        !b ||
        !['pause', 'resume', 'stop'].includes(b.action as string) ||
        (b.reason !== undefined && (typeof b.reason !== 'string' || b.reason.trim().length > 200))
      )
        return fail(400, 'invalid_request', '状态动作不合法')
      const from = plan.stopState
      const to: S['StopState'] | null =
        b.action === 'pause'
          ? from === 'normal'
            ? 'paused'
            : null
          : b.action === 'resume'
            ? from === 'paused'
              ? 'normal'
              : null
            : from === 'stopped'
              ? null
              : 'stopped'
      if (!to) return fail(409, 'invalid_transition', '当前状态不允许该操作')
      plan.stopState = to
      plan.stopReason = to === 'normal' ? '' : str(b.reason)
      plan.updatedAt = new Date().toISOString()
      save(db)
      return json(200, planRows(db)(plan))
    }
  }

  return fail(404, 'not_found', `演示环境未实现：${route}`)
}
