// In-browser implementation of the API for the public demo build.
// Behaviour mirrors the Go backend (validation, factory isolation, error codes).
import type { components } from '../schema'
import { factories, menus, seedContracts, seedCustomers, seedProjects, users, type DemoUser } from './data'

type S = components['schemas']

interface DemoDB {
  version: 2
  nextId: number
  projects: S['Project'][]
  customers: S['Customer'][]
  contracts: S['Contract'][]
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
  return {
    version: 2,
    nextId: Math.max(...[...projects, ...customers, ...contracts].map((item) => item.id)) + 1,
    projects,
    customers,
    contracts,
  }
}

function load(): DemoDB {
  const ls = storage()
  if (ls) {
    try {
      const parsed = JSON.parse(ls.getItem(KEY) ?? 'null') as DemoDB | null
      if (parsed?.version === 2) return parsed
    } catch {
      // fall through to a fresh dataset
    }
  }
  return (memory ??= fresh())
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

  return fail(404, 'not_found', `演示环境未实现：${route}`)
}
