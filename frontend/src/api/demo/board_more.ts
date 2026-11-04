import type { components } from '../schema'
import { jobs, type DemoPlan } from './data'
type S = components['schemas']
type DB = {
 batches: S['Batch'][]; labels: S['Label'][]; scans: S['Scan'][]; plans: DemoPlan[]; machines: S['Machine'][]
 deliveryNotes: S['DeliveryNote'][]; deliveryNoteLabels: { noteId: number; labelId: number }[]
}
const today = () => new Date().toLocaleDateString('en-CA')
const area = (rows: { pieceArea: number }[]) => Number(rows.reduce((n, r) => n + r.pieceArea, 0).toFixed(4))
export function workOrderBoard(db: DB, params: URLSearchParams, scope: (id: number) => boolean) {
 const rawPage = params.get('page') ?? '1', rawSize = params.get('size') ?? '10', status = params.get('status') ?? ''
 if (!/^\d+$/.test(rawPage) || !/^\d+$/.test(rawSize) || +rawPage < 1 || +rawSize < 1 || +rawSize > 50 || (status && !['pending', 'scheduled', 'in_production'].includes(status)))
  return { status: 400, body: { code: 'invalid_request', message: '分页或状态无效' } }
 const keyword = (params.get('keyword') ?? '').toLowerCase()
 const all = db.batches.filter((b) => scope(b.factoryId) && b.status !== 'done' && (!status || b.status === status) &&
  (!keyword || [b.lotNo, b.projectName, b.customerName].some((v) => v.toLowerCase().includes(keyword))))
 .sort((a,b) => Number(!!b.requiredDate && b.requiredDate < today()) - Number(!!a.requiredDate && a.requiredDate < today()) ||
  Number(b.urgency === 'urgent') - Number(a.urgency === 'urgent') || (a.requiredDate ?? '9999').localeCompare(b.requiredDate ?? '9999') || a.id - b.id)
 const items: S['WorkOrderRow'][] = all.slice((+rawPage-1)*+rawSize, +rawPage*+rawSize).map((b) => {
  const active = db.labels.filter((l) => l.batchId === b.id && !l.cancelled)
  const activeIds = new Set(active.map((l) => l.id))
  return { batchId:b.id, lotNo:b.lotNo, projectName:b.projectName, customerName:b.customerName, workshop:b.workshop,
   urgency:b.urgency, status:b.status, orderDate:b.createdAt.slice(0,10), requiredDate:b.requiredDate, planCutDate:b.planCutDate ?? null,
   itemQty:b.itemQty, itemArea:b.itemArea, labelQty:active.length, overdue:!!b.requiredDate && b.requiredDate < today(),
   printState:active.length === 0 ? 'none' : active.length >= b.itemQty ? 'all' : 'partial',
   jobs:jobs.map((j) => { const qty = db.scans.filter((s) => s.jobId === j.id && activeIds.has(s.labelId)).length
    return {jobId:j.id,code:j.code,name:j.name,qty,pct:active.length ? Math.floor(qty*100/active.length) : 0} }) }
 })
 return {status:200,body:{items,total:all.length,jobs}}
}
export function cockpitBoard(db: DB, scope: (id: number) => boolean): S['CockpitBoard'] {
 const day = today(), dates = Array.from({length:7},(_,i) => {const d=new Date(`${day}T12:00:00`);d.setDate(d.getDate()-6+i);return d.toLocaleDateString('en-CA')})
 const batches=db.batches.filter((b) => scope(b.factoryId)), ids=new Set(batches.map((b) => b.id))
 const labels=db.labels.filter((l) => ids.has(l.batchId) && !l.cancelled), labelIds=new Set(labels.map((l) => l.id))
 const scans=db.scans.filter((s) => labelIds.has(s.labelId)), scanned=(id:number,code:string) => scans.some((s) => s.labelId===id && jobs.find((j) => j.id===s.jobId)?.code===code)
 const semi=labels.filter((l) => scanned(l.id,'20') && !scanned(l.id,'40')), finished=labels.filter((l) => scanned(l.id,'40') && !scanned(l.id,'60'))
 const todayScans=scans.filter((s) => s.scannedAt.slice(0,10)===day)
 const noteIds=new Set(db.deliveryNotes.filter((n) => scope(n.factoryId) && n.createdAt.slice(0,10)===day).map((n) => n.id))
 const shipped=db.deliveryNoteLabels.filter((n) => noteIds.has(n.noteId) && labelIds.has(n.labelId)).map((n) => labels.find((l) => l.id===n.labelId)!)
 const weekly=(code:string):S['BoardDailyMetric'][] => dates.map((date) => {const rows=scans.filter((s) => s.scannedAt.slice(0,10)===date && jobs.find((j) => j.id===s.jobId)?.code===code);return {date,qty:rows.length,area:area(rows)}})
 const weeklyShipment:S['BoardDailyMetric'][]=dates.map((date) => {const noteIds=new Set(db.deliveryNotes.filter((n) => scope(n.factoryId) && n.createdAt.slice(0,10)===date).map((n) => n.id));const rows=db.deliveryNoteLabels.filter((n) => noteIds.has(n.noteId) && labelIds.has(n.labelId)).map((n) => labels.find((l) => l.id===n.labelId)!);return {date,qty:rows.length,area:area(rows)}})
 const plans=db.plans.filter((p) => p.kind==='sheet_daily' && scope(p.factoryId) && p.auditState==='audited' && p.stopState!=='stopped' && p.startDate<=day && p.endDate>=day)
 const allQty=plans.reduce((v,p) => v+labels.filter((l) => l.batchId===p.batchId).length,0),allArea=plans.reduce((v,p) => v+(batches.find((b) => b.id===p.batchId)?.itemArea ?? 0),0)
 const done=plans.flatMap((p) => scans.filter((s) => s.batchId===p.batchId && s.jobId===p.jobId))
 const backlog=new Map<string,S['CockpitBoard']['workshopBacklog'][number]>()
 for(const l of labels){const b=batches.find((b) => b.id===l.batchId)!;if(b.status==='done'||scanned(l.id,'20'))continue;const workshop=b.workshop || '未分配';const r=backlog.get(workshop) ?? {workshop,noScanQty:0,noScanArea:0};r.noScanQty++;r.noScanArea+=l.pieceArea;backlog.set(workshop,r)}
 const projectIds=[...new Set(batches.filter((b) => b.status!=='done').map((b) => b.projectId))].sort((a,b) => {
  const due=(id:number) => batches.filter((v) => v.projectId===id && v.status!=='done').map((v) => v.requiredDate ?? '9999').sort()[0] ?? '9999';return due(a).localeCompare(due(b)) || a-b }).slice(0,10)
 return { kpi:{inProductionBatches:batches.filter((b) => b.status==='in_production').length,openBatches:batches.filter((b) => b.status!=='done').length,
 overdueBatches:batches.filter((b) => b.status!=='done' && !!b.requiredDate && b.requiredDate<day).length,todayScanQty:todayScans.length,todayScanArea:area(todayScans),todayShipQty:shipped.length,todayShipArea:area(shipped),semiStockQty:semi.length,semiStockArea:area(semi),finishedStockQty:finished.length,finishedStockArea:area(finished)},
 weeklySheet:weekly('20'),weeklySpray:weekly('40'),weeklyShipment,
 jobOutput:jobs.map((j) => {const rows=todayScans.filter((s) => s.jobId===j.id);return {jobId:j.id,code:j.code,name:j.name,qty:rows.length,area:area(rows)}}),
 machineCapacity:db.machines.filter((m) => scope(m.factoryId) && !m.isStop).map((m) => {const rows=todayScans.filter((s) => s.machineId===m.id);return {machineId:m.id,name:m.name,qty:rows.length,area:area(rows)}}),
 sheetDaily:{allQty,allArea,doneQty:done.length,doneArea:area(done),qtyRate:allQty ? done.length*100/allQty:0,areaRate:allArea ? area(done)*100/allArea:0},
 workshopBacklog:[...backlog.values()].sort((a,b) => a.workshop.localeCompare(b.workshop)),
 projects:projectIds.map((id) => {const bs=batches.filter((b) => b.projectId===id && b.status!=='done'),set=new Set(bs.map((b) => b.id));const active=labels.filter((l) => set.has(l.batchId)),f=active.filter((l) => scanned(l.id,'55'));const planQty=bs.reduce((n,b) => n+b.itemQty,0);return {projectName:bs[0]!.projectName,lotNos:bs.map((b) => b.lotNo).sort().join(', '),planQty,planArea:bs.reduce((n,b) => n+b.itemArea,0),finishQty:f.length,finishArea:area(f),pct:planQty ? Math.floor(f.length*100/planQty):0}}) }
}
