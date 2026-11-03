import type { components } from '../schema'
import { jobs } from './data'
type S = components['schemas']
export interface WorkshopData {
  batches: S['Batch'][]
  plans: { batchId: number; auditState: string; stopState: string; auditedAt?: string | null }[]
  scans: S['Scan'][]
  labels: S['Label'][]
  deliveryNotes: S['DeliveryNote'][]
}
export function workshopReport(db: WorkshopData, q: URLSearchParams, inScope: (id: number) => boolean): Response {
  const mode = q.get('granularity')
  const month = mode === 'month'
  const today = new Date().toISOString().slice(0, 10)
  const rawFrom = q.get('from') ?? (month ? `${today.slice(0, 4)}-01-01` : `${today.slice(0, 7)}-01`)
  const rawTo = q.get('to') ?? today
  const from = rawFrom.slice(0, month ? 7 : 10)
  const to = rawTo.slice(0, month ? 7 : 10)
  const start = Date.parse(`${from}${month ? '-01' : ''}T00:00:00Z`)
  const end = Date.parse(`${to}${month ? '-01' : ''}T00:00:00Z`)
  const count = month
    ? (Number(to.slice(0, 4)) - Number(from.slice(0, 4))) * 12 + Number(to.slice(5)) - Number(from.slice(5)) + 1
    : Math.round((end - start) / 86400000) + 1
  const factoryId = q.get('factoryId')
  const json = (status: number, data: unknown) =>
    new Response(JSON.stringify(data), { status, headers: { 'Content-Type': 'application/json' } })
  if (
    !['day', 'month'].includes(mode ?? '') ||
    (factoryId !== null && (!/^\d+$/.test(factoryId) || Number(factoryId) <= 0)) ||
    !/^\d{4}-\d{2}-\d{2}$/.test(rawFrom) ||
    !/^\d{4}-\d{2}-\d{2}$/.test(rawTo) ||
    !Number.isFinite(start) ||
    !Number.isFinite(end) ||
    count < 1 ||
    count > (month ? 36 : 366)
  )
    return json(400, { code: 'invalid_request', message: '日期范围无效' })
  const batches = db.batches.filter(
    (b) =>
      inScope(b.factoryId) &&
      (!factoryId || b.factoryId === Number(factoryId)) &&
      (!q.get('workshop') || (b.workshop || '未分配') === q.get('workshop')),
  )
  const byBatch = new Map(batches.map((b) => [b.id, b]))
  type Row = S['WorkshopOutputRow']
  const fields = [
    'batchCount',
    'batchArea',
    'scheduledCount',
    'scheduledArea',
    'sheetQty',
    'sheetArea',
    'sheetOwnArea',
    'sheetOutsourceArea',
    'sprayQty',
    'sprayArea',
    'deliveredQty',
    'deliveredArea',
  ] as const
  const empty = (period: string): Row => ({
    period,
    batchCount: 0,
    batchArea: 0,
    scheduledCount: 0,
    scheduledArea: 0,
    sheetQty: 0,
    sheetArea: 0,
    sheetOwnArea: 0,
    sheetOutsourceArea: 0,
    sprayQty: 0,
    sprayArea: 0,
    deliveredQty: 0,
    deliveredArea: 0,
    workshopSheetArea: [],
  })
  const periods: Row[] = Array.from({ length: count }, (_, i) =>
    empty(
      month
        ? new Date(Date.UTC(Number(from.slice(0, 4)), Number(from.slice(5)) - 1 + i, 1)).toISOString().slice(0, 7)
        : new Date(start + i * 86400000).toISOString().slice(0, 10),
    ),
  )
  const byPeriod = new Map(periods.map((r) => [r.period, r]))
  const key = (date: string) => date.slice(0, month ? 7 : 10)
  const sheet = new Map<string, number>()
  const addSheet = (date: string, name: string, area: number) =>
    sheet.set(`${date.slice(0, 7)}:${name}`, (sheet.get(`${date.slice(0, 7)}:${name}`) ?? 0) + area)
  const distribution = new Map<string, number>()
  for (const b of batches) {
    const row = byPeriod.get(key(b.createdAt))
    if (row) {
      row.batchCount++
      row.batchArea += b.orderArea || b.itemArea
    }
    const first = db.plans
      .filter((p) => p.batchId === b.id && p.auditState === 'audited' && p.stopState !== 'stopped' && p.auditedAt)
      .map((p) => p.auditedAt!)
      .sort()[0]
    const scheduled = first && byPeriod.get(key(first))
    if (scheduled) {
      scheduled.scheduledCount++
      scheduled.scheduledArea += b.itemArea
    }
  }
  for (const scan of db.scans) {
    const b = byBatch.get(scan.batchId)
    const label = db.labels.find((l) => l.id === scan.labelId && !l.cancelled)
    if (!b || !label) continue
    const code = jobs.find((j) => j.id === scan.jobId)?.code
    if (code !== '20' && code !== '40') continue
    const area = label.pieceArea
    const name = b.workshop || '未分配'
    if (code === '20') {
      addSheet(scan.scannedAt, name, area)
      const k = `${key(scan.scannedAt)}:${name}`
      distribution.set(k, (distribution.get(k) ?? 0) + area)
    }
    const row = byPeriod.get(key(scan.scannedAt))
    if (!row) continue
    if (code === '20') {
      row.sheetQty++
      row.sheetArea += area
      if (b.outsourceFactoryId == null) row.sheetOwnArea += area
      else row.sheetOutsourceArea += area
    } else {
      row.sprayQty++
      row.sprayArea += area
    }
  }
  for (const dn of db.deliveryNotes) {
    if (!inScope(dn.factoryId) || (factoryId && dn.factoryId !== Number(factoryId)) || !byBatch.has(dn.batchId)) continue
    const row = byPeriod.get(key(dn.deliveredAt))
    if (row) {
      row.deliveredQty += dn.qty
      row.deliveredArea += dn.area
    }
  }
  const sortNames = (a: string, b: string) => (a === '未分配' ? 1 : b === '未分配' ? -1 : a.localeCompare(b, 'zh'))
  const workshops = [...new Set([...distribution.keys()].map((k) => k.slice((month ? 7 : 10) + 1)))].sort(sortNames)
  for (const row of periods) row.workshopSheetArea = workshops.map((w) => distribution.get(`${row.period}:${w}`) ?? 0)
  const total = empty('合计')
  for (const row of periods) for (const field of fields) total[field] += row[field]
  total.workshopSheetArea = workshops.map((_, i) => periods.reduce((sum, row) => sum + row.workshopSheetArea[i]!, 0))
  let comparison: S['WorkshopComparison'][] | null = null
  if (month) {
    const current = new Date(`${to}-01T00:00:00Z`)
    const keys = [-2, -1, 0].map((n) =>
      new Date(Date.UTC(current.getUTCFullYear(), current.getUTCMonth() + n, 1)).toISOString().slice(0, 7),
    )
    const names = [
      ...new Set([...workshops, ...[...sheet.keys()].filter((k) => keys.includes(k.slice(0, 7))).map((k) => k.slice(8))]),
    ].sort(sortNames)
    comparison = names.map((name) => {
      const [prevPrevArea, prevArea, currentArea] = keys.map((k) => sheet.get(`${k}:${name}`) ?? 0) as [number, number, number]
      return {
        workshop: name,
        prevPrevArea,
        currentArea,
        prevArea,
        chainRate: prevArea ? ((currentArea - prevArea) / prevArea) * 100 : null,
      }
    })
    const sums = keys.map((k) => names.reduce((n, name) => n + (sheet.get(`${k}:${name}`) ?? 0), 0)) as [number, number, number]
    comparison.push({
      workshop: '合计',
      prevPrevArea: sums[0],
      prevArea: sums[1],
      currentArea: sums[2],
      chainRate: sums[1] ? ((sums[2] - sums[1]) / sums[1]) * 100 : null,
    })
  }
  return json(200, { periods, workshops, total, comparison } satisfies S['WorkshopOutputReport'])
}
