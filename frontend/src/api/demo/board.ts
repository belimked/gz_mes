import type { components } from '../schema'
export { workOrderBoard, cockpitBoard } from './board_more'
import { jobs, type DemoPlan } from './data'

type S = components['schemas']
type BoardDB = {
  plans: DemoPlan[]
  batches: S['Batch'][]
  labels: S['Label'][]
  scans: S['Scan'][]
}
const round = (v: number, digits: number) => Number(v.toFixed(digits))
export function dailyBoard(db: BoardDB, params: URLSearchParams, kind: 'spray_daily' | 'sheet_daily', scope: (id: number) => boolean) {
  const date = params.get('date') ?? new Date().toISOString().slice(0, 10)
  if (!/^\d{4}-\d{2}-\d{2}$/.test(date) || Number.isNaN(Date.parse(date)) || new Date(date).toISOString().slice(0, 10) !== date)
    return { status: 400, body: { code: 'invalid_request', message: '日期格式不合法' } }
  const jobId = params.get('jobId')
  if (jobId !== null && (!/^\d+$/.test(jobId) || Number(jobId) < 1))
    return { status: 400, body: { code: 'invalid_request', message: '工序编号不合法' } }
  const batches = new Map(db.batches.map((b) => [b.id, b]))
  const plans = db.plans
    .filter(
      (p) =>
        p.kind === kind &&
        p.auditState === 'audited' &&
        p.stopState !== 'stopped' &&
        p.startDate <= date &&
        p.endDate >= date &&
        scope(p.factoryId),
    )
    .filter((p) =>
      kind === 'spray_daily'
        ? !params.get('paintLine') || p.paintLine === params.get('paintLine')
        : (!params.get('workshop') || p.workshop === params.get('workshop')) && (!jobId || p.jobId === Number(jobId)),
    )
    .sort(
      (a, b) =>
        Number(b.urgency === 'urgent') - Number(a.urgency === 'urgent') ||
        (kind === 'spray_daily'
          ? a.paintLine.localeCompare(b.paintLine) || a.shift.localeCompare(b.shift)
          : a.workshop.localeCompare(b.workshop) ||
            (jobs.find((j) => j.id === a.jobId)?.seq ?? 0) - (jobs.find((j) => j.id === b.jobId)?.seq ?? 0)) ||
        (batches.get(a.batchId)?.lotNo ?? '').localeCompare(batches.get(b.batchId)?.lotNo ?? ''),
    )
  const sprayRows: S['SprayBoardRow'][] = []
  const sheetRows: S['SheetBoardRow'][] = []
  for (const p of plans) {
    const b = batches.get(p.batchId)!
    const labels = db.labels.filter((l) => l.batchId === b.id && !l.cancelled)
    const active = new Set(labels.map((l) => l.id))
    const scanned = (code: string) =>
      db.scans.filter(
        (s) =>
          s.batchId === b.id &&
          active.has(s.labelId) &&
          (kind === 'sheet_daily' && code === 'target' ? s.jobId === p.jobId : s.jobId === (jobs.find((j) => j.code === code)?.id ?? -1)),
      )
    const target =
      kind === 'sheet_daily' ? db.scans.filter((s) => s.batchId === b.id && active.has(s.labelId) && s.jobId === p.jobId) : scanned('40')
    const day = target.filter((s) => s.scannedAt.slice(0, 10) === date)
    const area = (scans: S['Scan'][]) =>
      round(
        scans.reduce((sum, s) => sum + s.pieceArea, 0),
        4,
      )
    const common = {
      planId: p.id,
      seq: sprayRows.length + sheetRows.length + 1,
      projectName: b.projectName,
      lotNo: b.lotNo,
      urgency: p.urgency,
      startDate: p.startDate,
      endDate: p.endDate,
    }
    if (kind === 'spray_daily') {
      const sprayedArea = area(target)
      const progress = b.itemArea ? round((sprayedArea / b.itemArea) * 100, 1) : 0
      sprayRows.push({
        ...common,
        color: b.color,
        colorNo: b.colorNo ?? '',
        sprayingType: p.sprayType ?? '',
        paintLine: p.paintLine,
        shift: p.shift,
        plannedArea: b.itemArea,
        plannedQty: labels.length,
        sheetArea: area(scanned('20')),
        sprayedArea,
        daySprayedArea: area(day),
        unsprayedArea: Math.max(0, round(b.itemArea - sprayedArea, 4)),
        progress,
        state: progress >= 100 ? 'done' : sprayedArea > 0 ? 'in_progress' : scanned('20').length ? 'ready' : 'waiting_sheet',
        dueToday: p.endDate === date && progress < 100,
      })
    } else {
      const doneQty = target.length
      const progress = labels.length ? round((doneQty / labels.length) * 100, 1) : 0
      sheetRows.push({
        ...common,
        workshop: p.workshop,
        jobId: p.jobId!,
        jobName: jobs.find((j) => j.id === p.jobId)?.name ?? '',
        orderQty: b.orderQty,
        orderArea: b.orderArea || b.itemArea,
        dayPlanArea: round(b.itemArea / (1 + (Date.parse(p.endDate) - Date.parse(p.startDate)) / 86400000), 2),
        doneQty,
        doneArea: area(target),
        dayDoneQty: day.length,
        dayDoneArea: area(day),
        remainingQty: Math.max(0, labels.length - doneQty),
        progress,
        state: progress >= 100 ? 'done' : doneQty ? 'in_progress' : 'not_started',
        dueToday: p.endDate === date && progress < 100,
      })
    }
  }
  if (kind === 'spray_daily') {
    const byLine: S['SprayLineSummary'][] = []
    for (const r of sprayRows) {
      let line = byLine.find((l) => l.paintLine === r.paintLine)
      if (!line) {
        line = { paintLine: r.paintLine, count: 0, plannedArea: 0, daySprayedArea: 0 }
        byLine.push(line)
      }
      line.count++
      line.plannedArea += r.plannedArea
      line.daySprayedArea += r.daySprayedArea
    }
    return {
      status: 200,
      body: {
        rows: sprayRows,
        summary: {
          count: sprayRows.length,
          plannedArea: sprayRows.reduce((s, r) => s + r.plannedArea, 0),
          sprayedArea: sprayRows.reduce((s, r) => s + r.sprayedArea, 0),
          daySprayedArea: sprayRows.reduce((s, r) => s + r.daySprayedArea, 0),
          doneCount: sprayRows.filter((r) => r.state === 'done').length,
          byLine,
        },
      },
    }
  }
  const byJob: S['SheetJobSummary'][] = []
  for (const r of sheetRows) {
    let job = byJob.find((j) => j.jobName === r.jobName)
    if (!job) {
      job = { jobName: r.jobName, count: 0, dayPlanArea: 0, dayDoneArea: 0 }
      byJob.push(job)
    }
    job.count++
    job.dayPlanArea += r.dayPlanArea
    job.dayDoneArea += r.dayDoneArea
  }
  return {
    status: 200,
    body: {
      rows: sheetRows,
      summary: {
        count: sheetRows.length,
        orderArea: sheetRows.reduce((s, r) => s + r.orderArea, 0),
        dayPlanArea: sheetRows.reduce((s, r) => s + r.dayPlanArea, 0),
        dayDoneArea: sheetRows.reduce((s, r) => s + r.dayDoneArea, 0),
        doneCount: sheetRows.filter((r) => r.state === 'done').length,
        byJob,
      },
    },
  }
}
