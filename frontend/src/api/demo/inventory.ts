import type { components } from '../schema'
import { jobs } from './data'

type S = components['schemas']
type Data = { batches: S['Batch'][]; batchItems: S['BatchItem'][]; labels: S['Label'][]; scans: S['Scan'][]; contracts: S['Contract'][]; projects: S['Project'][]; customers: S['Customer'][] }
const reply = (status: number, body: unknown) => new Response(JSON.stringify(body), { status, headers: { 'Content-Type': 'application/json' } })
const bad = () => reply(400, { code: 'invalid_request', message: '参数或日期范围无效' })
export function inventoryReport(db: Data, q: URLSearchParams, scope: (id: number) => boolean, inventory: boolean): Response {
  const from = q.get('from'), to = q.get('to')
  const page = Number(q.get('page') ?? 1), size = Number(q.get('size') ?? 20)
  const kind = q.get('kind'), status = q.get('status') ?? 'all'
  if ((from && !/^\d{4}-\d\d-\d\d$/.test(from)) || (to && !/^\d{4}-\d\d-\d\d$/.test(to)) || (from && to && from > to) || !Number.isInteger(page) || page < 1 || !Number.isInteger(size) || size < 1 || size > 200 || (inventory ? !['semi', 'finished'].includes(kind ?? '') || (q.has('hasStock') && !['true', 'false'].includes(q.get('hasStock')!)) : !['all', 'done', 'not_done'].includes(status))) return bad()
  const kw = (q.get('keyword') ?? '').trim().toLowerCase()
  const batches = db.batches.filter((b) => scope(b.factoryId))
  const active = db.labels.filter((l) => !l.cancelled)
  const scans = new Map<number, Map<string, S['Scan']>>()
  for (const s of db.scans) {
    const code = jobs.find((j) => j.id === s.jobId)?.code
    if (!code) continue
    if (!scans.has(s.labelId)) scans.set(s.labelId, new Map())
    scans.get(s.labelId)!.set(code, s)
  }
  if (inventory) {
    const inCode = kind === 'semi' ? '20' : '40', outCode = kind === 'semi' ? '40' : '60'
    const within = (s?: S['Scan']) => s !== undefined && from !== null && to !== null && s.scannedAt.slice(0, 10) >= from && s.scannedAt.slice(0, 10) <= to
    const items: S['InventoryRow'][] = db.batchItems.flatMap((item) => {
      const b = batches.find((batch) => batch.id === item.batchId)
      if (!b || (q.has('batchId') && b.id !== Number(q.get('batchId')))) return []
      const labels = active.filter((l) => l.batchItemId === item.id)
      if (!labels.length || (kw && ![b.lotNo, b.projectName, b.clientOrderNo, item.pieceNo, item.drawingNo].some((s) => (s ?? '').toLowerCase().includes(kw)))) return []
      const inQty = labels.filter((l) => within(scans.get(l.id)?.get(inCode))).length
      const outQty = labels.filter((l) => within(scans.get(l.id)?.get(outCode))).length
      const stockQty = labels.filter((l) => scans.get(l.id)?.has(inCode) && !scans.get(l.id)?.has(outCode)).length
      if (q.get('hasStock') !== 'false' && !stockQty && !inQty && !outQty) return []
      return [{ batchItemId: item.id, batchId: b.id, lotNo: b.lotNo, projectName: b.projectName, clientOrderNo: b.clientOrderNo, orderDate: b.createdAt.slice(0, 10), thickness: item.thickness, color: item.color ?? '', pieceNo: item.pieceNo, seq: item.seq, drawingNo: item.drawingNo ?? '', planQty: item.qty, pieceArea: item.pieceArea, planArea: item.qty * item.pieceArea, inQty, outQty, stockQty, stockArea: stockQty * item.pieceArea }]
    }).sort((a, b) => a.lotNo.localeCompare(b.lotNo) || a.seq - b.seq)
    const totals = items.reduce((t, r) => ({ stockQty: t.stockQty + r.stockQty, stockArea: t.stockArea + r.stockArea, inQty: t.inQty + r.inQty, outQty: t.outQty + r.outQty }), { stockQty: 0, stockArea: 0, inQty: 0, outQty: 0 })
    return reply(200, { items: items.slice((page - 1) * size, page * size), total: items.length, totals } satisfies S['InventoryReport'])
  }
  const selected = batches.filter((b) => (!from || b.createdAt.slice(0, 10) >= from) && (!to || b.createdAt.slice(0, 10) <= to))
  const rows: S['ProjectSummaryRow'][] = db.projects.flatMap((p) => {
    if (kw && ![p.code, p.fullName].some((s) => s.toLowerCase().includes(kw))) return []
    const bs = selected.filter((b) => db.contracts.some((c) => c.id === b.contractId && c.projectId === p.id))
    if (!bs.length) return []
    const state: S['BatchStatus'] = bs.every((b) => b.status === 'done') ? 'done' : bs.some((b) => b.status === 'in_production') ? 'in_production' : 'pending'
    if (status === 'done' && state !== 'done' || status === 'not_done' && state === 'done') return []
    const ls = active.filter((l) => bs.some((b) => b.id === l.batchId))
    const cells = jobs.map((j) => { const hits = ls.flatMap((l) => { const scan = scans.get(l.id)?.get(j.code); return scan ? [{ scan, label: l }] : [] }); return { jobId: j.id, qty: hits.length, area: hits.reduce((n, h) => n + h.label.pieceArea, 0), pct: ls.length ? Math.floor(hits.length * 100 / ls.length) : 0 } })
    const names = [...new Set(bs.map((b) => b.customerName))].sort().join(', ')
    return [{ projectId: p.id, projectCode: p.code, projectName: p.fullName, customerName: names, batchCount: bs.length, planQty: bs.reduce((n, b) => n + b.itemQty, 0), planArea: bs.reduce((n, b) => n + b.itemArea, 0), labelQty: ls.length, status: state, cells }]
  }).sort((a, b) => a.projectCode.localeCompare(b.projectCode))
  const summary = { batchCount: rows.reduce((n, r) => n + r.batchCount, 0), planQty: rows.reduce((n, r) => n + r.planQty, 0), planArea: rows.reduce((n, r) => n + r.planArea, 0), labelQty: rows.reduce((n, r) => n + r.labelQty, 0), cells: jobs.map((j, i) => ({ jobId: j.id, qty: rows.reduce((n, r) => n + r.cells[i]!.qty, 0), area: rows.reduce((n, r) => n + r.cells[i]!.area, 0), pct: rows.reduce((n, r) => n + r.labelQty, 0) ? Math.floor(rows.reduce((n, r) => n + r.cells[i]!.qty, 0) * 100 / rows.reduce((n, r) => n + r.labelQty, 0)) : 0 })) }
  return reply(200, { jobs, items: rows.slice((page - 1) * size, page * size), total: rows.length, summary } satisfies S['ProjectSummaryReport'])
}
