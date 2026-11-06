// 排产助手原型：批次 / 交期 / 紧急度取真实数据，材料、库存、预定、在途、结算、产能为前端模拟数据。
// 模拟值按批次 / 客户 / 材料编码做确定性生成，刷新页面结果不变。
import dayjs from 'dayjs'
import type { Schemas } from '@/api/client'

type Batch = Schemas['Batch']
export type PlanKind = Schemas['PlanKind']

export type FactorKey = 'due' | 'urgent' | 'material' | 'payment' | 'ready'
export const FACTORS: { key: FactorKey; label: string; max: number; color: string; desc: string }[] = [
  { key: 'due', label: '交期紧迫', max: 35, color: '#c2410c', desc: '交期余量 = 距交付天数 − 预计工期；余量越小分越高，已无余量得满分' },
  { key: 'urgent', label: '紧急标记', max: 15, color: '#d97706', desc: '批次被标记为加急' },
  { key: 'material', label: '材料齐套', max: 25, color: '#2563eb', desc: '各材料可用库存覆盖剩余需求的比例，取最差的一项' },
  { key: 'payment', label: '回款良好', max: 15, color: '#15803d', desc: '客户无逾期欠款得满分；逾期越多分越低，超过授信直接置灰' },
  { key: 'ready', label: '准备就绪', max: 10, color: '#64748b', desc: '生产中 10 分，已排程 7 分' },
]

export const CAPACITY_PER_DAY: Record<PlanKind, number> = { sheet_week: 120, sheet_daily: 120, spray_daily: 200 }
const CREDIT_LIMIT_OVERDUE = 150_000

function hash(s: string) {
  let h = 2166136261
  for (let i = 0; i < s.length; i++) h = Math.imul(h ^ s.charCodeAt(i), 16777619)
  return h >>> 0
}
/** 0..1 的确定性伪随机数。 */
function rnd(key: string) {
  return (hash(key) % 10000) / 10000
}
function between(key: string, lo: number, hi: number) {
  return lo + rnd(key) * (hi - lo)
}
const round1 = (n: number) => Math.round(n * 10) / 10

export interface Settlement {
  terms: string
  receivable: number
  overdue: number
  overdueDays: number
  blocked: boolean
}
const TERMS = ['月结 60 天', '月结 90 天', '发货后 30 天付 70%，质保金 5%', '预付 30%，提货前付清']
export function customerSettlement(customerId: number): Settlement {
  const k = `cust-${customerId}`
  const hasOverdue = rnd(`${k}-od`) < 0.35
  const overdue = hasOverdue ? Math.round(between(`${k}-ov`, 30_000, 260_000) / 1000) * 1000 : 0
  return {
    terms: TERMS[hash(k) % TERMS.length] as string,
    receivable: Math.round(between(`${k}-rc`, 80_000, 800_000) / 1000) * 1000 + overdue,
    overdue,
    overdueDays: hasOverdue ? Math.round(between(`${k}-dd`, 5, 60)) : 0,
    blocked: overdue >= CREDIT_LIMIT_OVERDUE,
  }
}

export interface Supplier {
  name: string
  leadDays: number
  terms: string
}
export interface Material {
  code: string
  name: string
  unit: string
  /** 每 ㎡ 产品消耗的材料量（含损耗）。 */
  perArea: number
  lossRate: number
  need: number
  planned: number
  remaining: number
  onHand: number
  reserved: number
  available: number
  inTransit: number
  transitEta: string | null
  supplier: Supplier
}

const SUPPLIERS: Record<string, Supplier[]> = {
  sheet: [
    { name: '南山铝业', leadDays: 7, terms: '预付 30%，到货月结 30 天' },
    { name: '明泰铝业', leadDays: 5, terms: '月结 45 天' },
  ],
  coat: [
    { name: '阿克苏诺贝尔', leadDays: 4, terms: '月结 60 天' },
    { name: '老虎粉末', leadDays: 3, terms: '款到发货' },
  ],
  film: [{ name: '永新包装', leadDays: 2, terms: '月结 30 天' }],
}

function material(
  b: Batch,
  area: number,
  code: string,
  name: string,
  unit: string,
  perArea: number,
  lossRate: number,
  group: string,
  stockRange: [number, number],
): Material {
  const need = round1(area * perArea * (1 + lossRate))
  const planned = b.status === 'in_production' ? round1(need * between(`${b.id}-${code}-pl`, 0.1, 0.5)) : 0
  const onHand = round1(between(`${code}-oh`, stockRange[0], stockRange[1]))
  const reserved = round1(onHand * between(`${code}-rs`, 0.1, 0.45))
  const hasTransit = rnd(`${code}-tr`) < 0.5
  const suppliers = SUPPLIERS[group] ?? []
  return {
    code,
    name,
    unit,
    perArea: perArea * (1 + lossRate),
    lossRate,
    need,
    planned,
    remaining: round1(need - planned),
    onHand,
    reserved,
    available: round1(onHand - reserved),
    inTransit: hasTransit ? round1(between(`${code}-tq`, stockRange[0] * 0.5, stockRange[1] * 0.5)) : 0,
    transitEta: hasTransit
      ? dayjs()
          .add(Math.round(between(`${code}-te`, 1, 8)), 'day')
          .format('YYYY-MM-DD')
      : null,
    supplier: suppliers[hash(code) % suppliers.length] as Supplier,
  }
}

export function batchArea(b: Batch) {
  return Number(b.itemArea) || Number(b.orderArea) || 100
}

export function batchMaterials(b: Batch): Material[] {
  const area = batchArea(b)
  const texture = b.texture || '3003'
  const isFluoro = (b.sprayingType || '').includes('氟碳')
  const colour = b.colorNo || b.color || '标准色'
  const thick = Number(b.thickness) || 2
  return [
    material(b, area, `AL-${thick}-${texture}`, `铝板 ${thick}mm ${texture}`, '㎡', 1, 0.08, 'sheet', [150, 1400]),
    material(b, area * 0.12, `AL-${thick + 1}-${texture}-RIB`, `加强筋铝板 ${thick + 1}mm ${texture}`, '㎡', 1, 0.1, 'sheet', [20, 180]),
    isFluoro
      ? material(b, area, `FC-${colour}`, `氟碳漆 ${colour}`, 'L', 0.3, 0.05, 'coat', [20, 260])
      : material(b, area, `PW-${colour}`, `粉末 ${colour}`, 'kg', 0.15, 0.05, 'coat', [15, 220]),
    material(b, area, 'FILM-PE60', '保护膜 PE 60μ', '㎡', 1, 0.02, 'film', [300, 2400]),
  ]
}

export type Tone = 'good' | 'warn' | 'bad' | 'info'
export interface Reason {
  tone: Tone
  text: string
}
export interface PlanRange {
  start: string
  end: string
}
export interface Candidate {
  batch: Batch
  area: number
  /** 同类型、未停止、未结束的已有计划。 */
  existing: PlanRange[]
  scores: Record<FactorKey, number>
  total: number
  blocked: boolean
  reasons: Reason[]
  materials: Material[]
  coverage: number
  settlement: Settlement
  estDays: number
  daysLeft: number | null
  latestStart: string | null
  slack: number | null
}

export function estimateDays(area: number, kind: PlanKind) {
  return Math.max(1, Math.ceil(area / CAPACITY_PER_DAY[kind])) + (kind === 'spray_daily' ? 1 : 2)
}

export function buildCandidate(b: Batch, kind: PlanKind, plans: PlanRange[] = []): Candidate {
  const area = batchArea(b)
  const materials = batchMaterials(b)
  const settlement = customerSettlement(b.customerId)
  const estDays = estimateDays(area, kind)
  const today = dayjs().startOf('day')
  const daysLeft = b.requiredDate ? dayjs(b.requiredDate).diff(today, 'day') : null
  const slack = daysLeft == null ? null : daysLeft - estDays
  const latestStart = b.requiredDate ? dayjs(b.requiredDate).subtract(estDays, 'day').format('YYYY-MM-DD') : null
  const coverage = Math.min(...materials.map((m) => (m.remaining <= 0 ? 1 : Math.min(1, Math.max(0, m.available) / m.remaining))))
  const reasons: Reason[] = []

  let due = 5
  if (daysLeft != null && slack != null) {
    due = slack <= 0 ? 35 : Math.round(35 * Math.max(0, 1 - slack / 20))
    if (daysLeft < 0) reasons.push({ tone: 'bad', text: `已逾期 ${-daysLeft} 天` })
    else if (slack <= 0) reasons.push({ tone: 'bad', text: `交期剩 ${daysLeft} 天，预计需 ${estDays} 天，可能赶不上` })
    else if (slack <= 5) reasons.push({ tone: 'warn', text: `交期剩 ${daysLeft} 天（预计需 ${estDays} 天）` })
    else reasons.push({ tone: 'info', text: `交期剩 ${daysLeft} 天，余量充足` })
  } else reasons.push({ tone: 'info', text: '未设交期' })

  const urgent = b.urgency === 'urgent' ? 15 : 0
  if (urgent) reasons.push({ tone: 'warn', text: '加急单' })

  const material = Math.round(25 * coverage)
  const short = materials.filter((m) => m.available < m.remaining)
  if (!short.length) reasons.push({ tone: 'good', text: '材料齐套' })
  else
    for (const m of short) reasons.push({ tone: 'warn', text: `缺 ${m.name} ${round1(m.remaining - Math.max(0, m.available))}${m.unit}` })

  const payment = settlement.overdue === 0 ? 15 : Math.round(15 * Math.max(0, 1 - settlement.overdue / CREDIT_LIMIT_OVERDUE))
  if (settlement.blocked)
    reasons.push({
      tone: 'bad',
      text: `客户逾期欠款 ${(settlement.overdue / 10000).toFixed(1)} 万（超 ${settlement.overdueDays} 天），超出授信`,
    })
  else if (settlement.overdue) reasons.push({ tone: 'warn', text: `客户逾期 ${(settlement.overdue / 10000).toFixed(1)} 万` })
  else reasons.push({ tone: 'good', text: '无逾期欠款' })

  const existing = plans.filter((p) => !dayjs(p.end).isBefore(today)).sort((a, b) => a.start.localeCompare(b.start))
  const ready = existing.length ? 0 : b.status === 'in_production' ? 10 : 7
  if (existing.length)
    reasons.unshift({ tone: 'info', text: `已有计划 ${existing.map((p) => `${p.start.slice(5)}~${p.end.slice(5)}`).join('、')}` })

  const scores = { due, urgent, material, payment, ready }
  const total = Object.values(scores).reduce((a, c) => a + c, 0)
  return {
    batch: b,
    area,
    existing,
    scores,
    total,
    blocked: settlement.blocked,
    reasons,
    materials,
    coverage,
    settlement,
    estDays,
    daysLeft,
    latestStart,
    slack,
  }
}

/** 排序：可排 → 已有计划 → 不建议排产，组内按总分。 */
export function rankCandidates(batches: Batch[], kind: PlanKind, plansByBatch: Map<number, PlanRange[]> = new Map()) {
  const group = (c: Candidate) => (c.blocked ? 2 : c.existing.length ? 1 : 0)
  return batches
    .filter((b) => b.status === 'scheduled' || b.status === 'in_production')
    .map((b) => buildCandidate(b, kind, plansByBatch.get(b.id)))
    .sort((a, b) => group(a) - group(b) || b.total - a.total)
}

export function overlaps(c: Candidate, dates: [string, string]) {
  return c.existing.find((p) => p.start <= dates[1] && p.end >= dates[0])
}

/** 建议计划日期：余量充足时从最晚开工日前 2 天开始，否则今天开始。 */
export function suggestDates(c: Candidate): [string, string] {
  const today = dayjs().startOf('day')
  let start = today
  const lastEnd = c.existing.at(-1)?.end
  if (lastEnd) {
    // 已有计划时接在最后一个计划之后，避开重复计划校验。
    const s = dayjs(lastEnd).add(1, 'day')
    return [s.format('YYYY-MM-DD'), s.add(c.estDays - 1, 'day').format('YYYY-MM-DD')]
  }
  if (c.latestStart && c.slack != null && c.slack > 2) {
    const s = dayjs(c.latestStart).subtract(2, 'day')
    if (s.isAfter(today)) start = s
  }
  return [start.format('YYYY-MM-DD'), start.add(c.estDays - 1, 'day').format('YYYY-MM-DD')]
}

/** 所选日期内产线剩余产能（㎡）；已占用比例为模拟值。 */
export function freeCapacity(kind: PlanKind, dates: [string, string]) {
  const days = dayjs(dates[1]).diff(dayjs(dates[0]), 'day') + 1
  const used = between(`cap-${kind}-${dates[0]}`, 0.3, 0.7)
  return { days, total: days * CAPACITY_PER_DAY[kind], free: Math.round(days * CAPACITY_PER_DAY[kind] * (1 - used)) }
}

export type Limit = 'need' | 'stock' | 'capacity'
export const LIMIT_TEXT: Record<Limit, string> = { need: '按剩余需求', stock: '受可用库存限制', capacity: '受产能限制' }

export function suggestQty(m: Material, freeArea: number): { qty: number; limit: Limit } {
  const byCap = round1(freeArea * m.perArea)
  const opts: [number, Limit][] = [
    [m.remaining, 'need'],
    [Math.max(0, m.available), 'stock'],
    [byCap, 'capacity'],
  ]
  opts.sort((a, b) => a[0] - b[0])
  const [qty, limit] = opts[0] as [number, Limit]
  return { qty: round1(qty), limit }
}
