import { beforeEach, describe, expect, it } from 'vitest'
import { call, makeClient } from '@/api/client'
import { demoFetch, resetDemo } from '@/api/demo'
import {
  jobs,
  seedBatchItems,
  seedBatches,
  seedContracts,
  seedCustomers,
  seedLabels,
  seedPlans,
  seedProjects,
  seedScans,
} from '@/api/demo/data'

// The demo API must behave like the Go backend; these tests mirror backend/internal/server/server_test.go.
const api = makeClient('http://demo.local/api', demoFetch)

async function login(username: string, password: string) {
  const r = await call(api.POST('/auth/login', { body: { username, password } }))
  return { headers: { Authorization: `Bearer ${r.token}` } }
}

beforeEach(() => {
  localStorage.clear()
  resetDemo()
})

describe('demo api', () => {
  it('serves scoped work-order pages and seven-day cockpit trends', async () => {
    const admin = await login('admin', 'admin123')
    const guest = await login('waixie', 'demo123')
    const board = await call(api.GET('/boards/work-orders', { ...admin, params: { query: { size: 2 } } }))
    expect(board.items).toHaveLength(Math.min(2, board.total))
    expect(board.jobs).toHaveLength(11)
    expect(board.items.every((r) => r.jobs.length === 11 && r.status !== 'done')).toBe(true)
    const filtered = await call(api.GET('/boards/work-orders', { ...admin, params: { query: { status: 'pending' } } }))
    expect(filtered.items.every((r) => r.status === 'pending')).toBe(true)
    expect((await api.GET('/boards/work-orders', { ...admin, params: { query: { size: 51 } } })).response.status).toBe(400)
    const cockpit = await call(api.GET('/boards/cockpit', admin))
    expect(cockpit.weeklySheet).toHaveLength(7)
    expect(cockpit.weeklySpray).toHaveLength(7)
    expect(cockpit.weeklyShipment).toHaveLength(7)
    expect(cockpit.jobOutput).toHaveLength(11)
    const scoped = await call(api.GET('/boards/work-orders', guest))
    const guestBatches = await call(api.GET('/batches', { ...guest }))
    expect(scoped.items.every((r) => guestBatches.items.some((b) => b.id === r.batchId))).toBe(true)
    const scopedCockpit = await call(api.GET('/boards/cockpit', guest))
    expect(scopedCockpit.kpi.openBatches).toBeLessThanOrEqual(cockpit.kpi.openBatches)
  })
  it('derives both inventories and project summary from active scans with scope and date filters', async () => {
    const admin = await login('admin', 'admin123')
    const guest = await login('waixie', 'demo123')
    const semi = await call(api.GET('/inventory', { ...admin, params: { query: { kind: 'semi' } } }))
    const finished = await call(api.GET('/inventory', { ...admin, params: { query: { kind: 'finished' } } }))
    expect(semi.totals.stockQty).toBeGreaterThan(0)
    expect(finished.totals.stockQty).toBeGreaterThan(0)
    const empty = await call(api.GET('/inventory', { ...admin, params: { query: { kind: 'semi', from: '2000-01-01', to: '2000-01-02' } } }))
    expect(empty.totals.inQty).toBe(0)
    expect(empty.totals.stockQty).toBe(semi.totals.stockQty)
    const all = await call(api.GET('/inventory', { ...admin, params: { query: { kind: 'semi', hasStock: false } } }))
    expect(all.total).toBeGreaterThanOrEqual(semi.total)
    const own = semi.items.find((r) => r.batchId === 9 || r.batchId === 10) ?? semi.items[0]!
    const hidden = await call(api.GET('/inventory', { ...guest, params: { query: { kind: 'semi', batchId: own.batchId } } }))
    if (own.batchId && !(await call(api.GET('/batches', { ...guest, params: { query: { keyword: own.lotNo } } }))).total)
      expect(hidden.total).toBe(0)
    const projects = await call(api.GET('/reports/project-summary', { ...admin }))
    expect(projects.total).toBeGreaterThan(0)
    expect(projects.summary.labelQty).toBeGreaterThan(0)
    expect(projects.items.some((r) => r.cells.some((c) => c.qty > 0))).toBe(true)
    const done = await call(api.GET('/reports/project-summary', { ...admin, params: { query: { status: 'done' } } }))
    expect(done.items.every((r) => r.status === 'done')).toBe(true)
  })
  it('records label events and audits mutations but not reads', async () => {
    const admin = await login('admin', 'admin123')
    const seeded = await call(api.GET('/label-logs', { ...admin }))
    expect(seeded.total).toBeGreaterThan(0)
    const label = seeded.items.find((x) => x.factoryId === 1)!
    const path = { barCode: label.barCode }
    expect((await api.POST('/labels/{barCode}/reprint', { ...admin, params: { path }, body: { reason: '' } })).response.status).toBe(400)
    await call(api.POST('/labels/{barCode}/reprint', { ...admin, params: { path }, body: { reason: '补印' } }))
    const detail = await call(api.GET('/labels/{barCode}', { ...admin, params: { path } }))
    expect(detail.logs.some((x) => x.opType === 'reprint' && x.reason === '补印')).toBe(true)
    const events = await call(api.GET('/label-logs', { ...admin, params: { query: { opType: 'reprint', keyword: label.barCode } } }))
    expect(events.total).toBe(1)
    expect(events.totals.reprintQty).toBe(1)
    const guest = await login('waixie', 'demo123')
    expect((await call(api.GET('/label-logs', { ...guest, params: { query: { keyword: label.barCode } } }))).total).toBe(0)
    expect((await api.GET('/audit-logs', guest)).response.status).toBe(403)
    const audit = await call(api.GET('/audit-logs', { ...admin, params: { query: { result: 'fail' } } }))
    expect(audit.items.some((x) => x.path.includes('/reprint') && x.status === 400)).toBe(true)
    expect(audit.items.every((x) => x.method !== 'GET')).toBe(true)
  })
  it('manages roles, user assignments and stopped menu grants', async () => {
    const admin = await login('admin', 'admin123')
    const guest = await login('waixie', 'demo123')
    const input = { code: 'auditor', name: '审核员', isStop: false, menuIds: [105], remark: '' }
    expect((await api.GET('/roles', guest)).response.status).toBe(403)
    const role = await call(api.POST('/roles', { ...admin, body: input }))
    expect((await api.POST('/roles', { ...admin, body: input })).response.status).toBe(409)
    expect((await api.POST('/roles', { ...admin, body: { ...input, code: 'user:bad' } })).response.status).toBe(400)
    const users = await call(api.GET('/users', { ...admin }))
    const person = users.items.find((u) => u.username === 'waixie')!
    const path = { id: person.id }
    const body = {
      displayName: person.displayName,
      staffNo: person.staffNo ?? '',
      enabled: true,
      isAdmin: false,
      factoryIds: person.factoryIds,
      menuIds: person.menuIds,
      roleIds: [role.id],
    }
    await call(api.PUT('/users/{id}', { ...admin, params: { path }, body }))
    expect((await api.DELETE('/roles/{id}', { ...admin, params: { path: { id: role.id } } })).response.status).toBe(409)
    const unchanged = await call(api.PUT('/users/{id}', { ...admin, params: { path }, body: { ...body, roleIds: undefined } }))
    expect(unchanged.roleIds).toEqual([role.id])
    await call(api.PUT('/roles/{id}', { ...admin, params: { path: { id: role.id } }, body: { ...input, isStop: true } }))
    expect((await call(api.GET('/auth/me', guest))).menus.flatMap((m) => m.children ?? []).some((m) => m.id === 105)).toBe(false)
    await call(api.PUT('/users/{id}', { ...admin, params: { path }, body: { ...body, roleIds: [] } }))
    await call(api.DELETE('/roles/{id}', { ...admin, params: { path: { id: role.id } } }))
    expect(await call(api.GET('/roles', admin))).toEqual([])
  })

  it('maintains dictionaries and cascades renamed values', async () => {
    const admin = await login('admin', 'admin123')
    const guest = await login('waixie', 'demo123')
    const seeded = await call(api.GET('/dicts', { ...admin, params: { query: { kind: 'shift' } } }))
    expect(seeded.map((d) => d.name)).toEqual(['早班', '中班', '晚班'])
    const input = { kind: 'workshop' as const, name: '钣金一车间', sort: 1, isStop: false }
    expect((await api.POST('/dicts', { ...admin, body: input })).response.status).toBe(409)
    expect((await api.POST('/dicts', { ...guest, body: { ...input, name: '其他' } })).response.status).toBe(403)
    const original = (await call(api.GET('/dicts', { ...admin, params: { query: { kind: 'workshop' } } })))[0]!
    await call(
      api.PUT('/dicts/{id}', { ...admin, params: { path: { id: original.id } }, body: { ...input, name: '改名车间', isStop: true } }),
    )
    const hidden = await call(api.GET('/dicts', { ...admin, params: { query: { kind: 'workshop' } } }))
    expect(hidden.some((d) => d.name === '改名车间')).toBe(false)
    const all = await call(api.GET('/dicts', { ...admin, params: { query: { kind: 'workshop', includeStopped: true } } }))
    expect(all.some((d) => d.name === '改名车间')).toBe(true)
    const batches = await call(api.GET('/batches', { ...admin, params: { query: { size: 200 } } }))
    expect(batches.items.some((b) => b.workshop === '改名车间')).toBe(true)
  })

  it('seeds no overlapping active plans for the same batch and target job', () => {
    const batches = seedBatches(seedContracts(seedProjects(), seedCustomers()))
    const scans = seedScans(batches, seedLabels(batches, seedBatchItems(batches)), jobs)
    const plans = seedPlans(batches, jobs, scans).filter((p) => p.stopState !== 'stopped')
    for (const [index, plan] of plans.entries()) {
      const overlaps = plans
        .slice(index + 1)
        .some(
          (other) =>
            other.kind === plan.kind &&
            other.batchId === plan.batchId &&
            (plan.kind !== 'sheet_daily' || other.jobId === plan.jobId) &&
            other.startDate <= plan.endDate &&
            other.endDate >= plan.startDate,
        )
      expect(overlaps, `${plan.kind} batch ${plan.batchId} job ${plan.jobId} ${plan.startDate}–${plan.endDate}`).toBe(false)
    }
  })
  it('spray board only includes audited plans for today and sums rows', async () => {
    const admin = await login('admin', 'admin123')
    const today = new Date().toISOString().slice(0, 10)
    const board = await call(api.GET('/boards/spray-daily', { ...admin, params: { query: { date: today } } }))
    expect(board.rows.length).toBeGreaterThan(0)
    expect(board.rows.every((r) => r.startDate <= today && r.endDate >= today)).toBe(true)
    expect(board.summary.count).toBe(board.rows.length)
    expect(board.rows.some((r) => r.dueToday)).toBe(true)
    expect(board.summary.daySprayedArea).toBeCloseTo(board.rows.reduce((s, r) => s + r.daySprayedArea, 0))
    const empty = await call(api.GET('/boards/spray-daily', { ...admin, params: { query: { date: '2020-01-01' } } }))
    expect(empty.rows).toEqual([])
    expect((await api.GET('/boards/spray-daily', { ...admin, params: { query: { date: 'bad' } } })).response.status).toBe(400)
  })
  it('sheet board sums daily areas and respects factory scope', async () => {
    const admin = await login('admin', 'admin123')
    const wx = await login('waixie', 'demo123')
    const today = new Date().toISOString().slice(0, 10)
    const board = await call(api.GET('/boards/sheet-daily', { ...admin, params: { query: { date: today } } }))
    expect(board.rows.length).toBeGreaterThan(0)
    expect(board.rows.every((r) => r.startDate <= today && r.endDate >= today)).toBe(true)
    expect(board.summary.count).toBe(board.rows.length)
    expect(board.summary.dayPlanArea).toBeCloseTo(board.rows.reduce((s, r) => s + r.dayPlanArea, 0))
    const scoped = await call(api.GET('/boards/sheet-daily', { ...wx, params: { query: { date: today } } }))
    expect(scoped.rows.every((r) => board.rows.some((x) => x.planId === r.planId))).toBe(true)
  })
  it('reports daily and monthly workshop output with zero periods and totals', async () => {
    const admin = await login('admin', 'admin123')
    for (const granularity of ['day', 'month'] as const) {
      const report = await call(
        api.GET('/reports/workshop-output', { ...admin, params: { query: { granularity, from: '2026-01-01', to: '2026-01-03' } } }),
      )
      expect(report.periods.length).toBe(granularity === 'day' ? 3 : 1)
      for (const field of ['batchCount', 'scheduledCount', 'sheetQty', 'sprayQty', 'deliveredQty', 'sheetArea'] as const)
        expect(report.total[field]).toBeCloseTo(report.periods.reduce((sum, row) => sum + row[field], 0))
      expect(report.periods.every((row) => row.workshopSheetArea.length === report.workshops.length)).toBe(true)
      expect(report.comparison === null).toBe(granularity === 'day')
    }
    const empty = await call(
      api.GET('/reports/workshop-output', { ...admin, params: { query: { granularity: 'day', from: '2020-01-01', to: '2020-01-02' } } }),
    )
    expect(empty.periods.map((r) => r.sheetArea)).toEqual([0, 0])
    expect(
      (
        await api.GET('/reports/workshop-output', {
          ...admin,
          params: { query: { granularity: 'day', from: '2020-01-01', to: '2022-01-01' } },
        })
      ).response.status,
    ).toBe(400)
  })
  it('manages machines and reports output with factory isolation', async () => {
    const admin = await login('admin', 'admin123')
    const wx = await login('waixie', 'demo123')
    const jobs = await call(api.GET('/jobs', admin))
    const input = { factoryId: 1, jobId: jobs[0]!.id, code: 'NEW-M', name: '新机台', capacityQty: 10, capacityArea: 5, isStop: false }
    const created = await call(api.POST('/machines', { ...admin, body: input }))
    expect(created.code).toBe('NEW-M')
    expect((await api.POST('/machines', { ...admin, body: input })).response.status).toBe(409)
    expect((await api.POST('/machines', { ...wx, body: input })).response.status).toBe(400)
    expect(await call(api.GET('/machines', wx))).toEqual([])
    const today = new Date().toISOString().slice(0, 10)
    const report = await call(api.GET('/reports/machine-output', { ...admin, params: { query: { from: today, to: today } } }))
    expect(report.items.some((r) => r.machineId === created.id && r.qty === 0)).toBe(true)
    const detail = await call(
      api.GET('/reports/machine-output', { ...admin, params: { query: { mode: 'detail', from: '2026-01-01', to: today } } }),
    )
    expect(detail.items.every((r) => r.batchId != null && r.capacityQty == null)).toBe(true)
    expect(
      (await api.GET('/reports/machine-output', { ...admin, params: { query: { from: today, to: '2020-01-01' } } })).response.status,
    ).toBe(400)
  })
  it('reports outsource orders with factory scope and scan progress', async () => {
    const admin = await login('admin', 'admin123')
    const wx = await login('waixie', 'demo123')
    const all = await call(api.GET('/reports/outsource-orders', admin))
    const scoped = await call(api.GET('/reports/outsource-orders', wx))
    expect(scoped.items.length).toBeGreaterThan(0)
    expect(scoped.items.every((r) => r.outsourceFactoryId === 2)).toBe(true)
    expect(all.total).toBeGreaterThanOrEqual(scoped.total)
    expect(scoped.items[0]!.cutting.unscannedQty).toBe(Math.max(0, scoped.items[0]!.itemQty - scoped.items[0]!.cutting.scannedQty))
  })

  it('creates, edits, confirms and protects delivery plans', async () => {
    const admin = await login('admin', 'admin123')
    const notes = await call(api.GET('/delivery-notes', { ...admin, params: { query: { factoryId: 1, unplanned: true } } }))
    expect(notes.items.length).toBeGreaterThan(0)
    const input = {
      factoryId: 1,
      estimatedDepartureDate: '2026-01-02',
      noteIds: [notes.items[0]!.id],
      logisticsCompany: '演示新物流',
      plateNo: '粤A12345',
    }
    const created = await call(api.POST('/delivery-plans', { ...admin, body: input }))
    expect(created.notes).toHaveLength(1)
    const linked = await call(api.GET('/delivery-notes/{id}', { ...admin, params: { path: { id: notes.items[0]!.id } } }))
    expect(linked.planNo).toBe(created.planNo)
    const dup = await api.POST('/delivery-plans', { ...admin, body: input })
    expect(dup.response.status).toBe(400)
    const confirmed = await call(
      api.POST('/delivery-plans/{id}/confirm', { ...admin, params: { path: { id: created.id } }, body: { departureDate: '2026-01-03' } }),
    )
    expect(confirmed.confirmedDepartureDate).toBe('2026-01-03')
    const blocked = await api.DELETE('/delivery-plans/{id}', { ...admin, params: { path: { id: created.id } } })
    expect(blocked.response.status).toBe(409)
  })

  it('rejects bad credentials and anonymous access', async () => {
    const bad = await api.POST('/auth/login', { body: { username: 'admin', password: 'x' } })
    expect(bad.response.status).toBe(401)
    const anon = await api.GET('/projects')
    expect(anon.response.status).toBe(401)
  })

  it('returns role-based menus', async () => {
    const admin = await call(api.GET('/auth/me', await login('admin', 'admin123')))
    expect(admin.menus.length).toBeGreaterThan(1)
    const wx = await call(api.GET('/auth/me', await login('waixie', 'demo123')))
    expect(wx.menus.map((m) => m.title)).toEqual(['看板管理', '销售管理', '排程管理'])
    expect(wx.menus[0]?.children?.map((c) => c.path)).toEqual([
      '/board/spray-daily',
      '/board/sheet-daily',
      '/board/work-orders',
      '/board/cockpit',
    ])
    expect(wx.menus[1]?.children?.map((c) => c.path)).toEqual(['/sale/projects', '/sale/contracts'])
    expect(wx.menus[2]?.children?.map((c) => c.path)).toEqual([
      '/scheduling/sheet-week',
      '/scheduling/sheet-daily',
      '/scheduling/spray-daily',
      '/scheduling/outsource-orders',
    ])
  })

  it('isolates data by factory', async () => {
    const wx = await login('waixie', 'demo123')
    const page = await call(api.GET('/projects', wx))
    expect(page.items.every((p) => p.factoryId === 2)).toBe(true)
    const forbidden = await api.POST('/projects', { ...wx, body: { factoryId: 1, code: 'X', fullName: 'X' } })
    expect(forbidden.response.status).toBe(403)
    const factories = await call(api.GET('/factories', wx))
    expect(factories.map((f) => f.id)).toEqual([2])
  })

  it('creates, rejects duplicates, and confirms projects', async () => {
    const admin = await login('admin', 'admin123')
    const created = await call(api.POST('/projects', { ...admin, body: { factoryId: 1, code: ' NEW-1 ', fullName: '新项目' } }))
    expect(created).toMatchObject({ code: 'NEW-1', confirmed: false, kind: 'offline', createdBy: '演示管理员' })

    const dup = await api.POST('/projects', { ...admin, body: { factoryId: 1, code: 'NEW-1', fullName: '重复' } })
    expect(dup.response.status).toBe(409)
    const blank = await api.POST('/projects', { ...admin, body: { factoryId: 1, code: '  ', fullName: 'x' } })
    expect(blank.response.status).toBe(400)

    const ok = await call(api.POST('/projects/{id}/confirm', { ...admin, params: { path: { id: created.id } } }))
    expect(ok.confirmed).toBe(true)
    const again = await api.POST('/projects/{id}/confirm', { ...admin, params: { path: { id: created.id } } })
    expect(again.response.status).toBe(409)
  })

  it('searches and pages', async () => {
    const admin = await login('admin', 'admin123')
    const all = await call(api.GET('/projects', { ...admin, params: { query: { size: 2 } } }))
    expect(all.items).toHaveLength(2)
    expect(all.total).toBe(5)
    const hit = await call(api.GET('/projects', { ...admin, params: { query: { keyword: '云谷' } } }))
    expect(hit.items.map((p) => p.code)).toEqual(['DEMO-002'])
    const tooBig = await api.GET('/projects', { ...admin, params: { query: { size: 1000 } } })
    expect(tooBig.response.status).toBe(400)
  })

  it('persists across reloads until reset', async () => {
    const admin = await login('admin', 'admin123')
    await call(api.POST('/projects', { ...admin, body: { factoryId: 1, code: 'KEEP', fullName: '保留' } }))
    expect((await call(api.GET('/projects', admin))).total).toBe(6)
    resetDemo()
    expect((await call(api.GET('/projects', admin))).total).toBe(5)
  })

  it('lists scan records newest first with keyword / staff filters', async () => {
    const admin = await login('admin', 'admin123')
    const page = await call(api.GET('/scans', { ...admin, params: { query: { size: 5 } } }))
    expect(page.total).toBeGreaterThan(5)
    expect(page.items).toHaveLength(5)
    for (let i = 1; i < page.items.length; i++) expect(page.items[i - 1]!.scannedAt >= page.items[i]!.scannedAt).toBe(true)
    const first = page.items[0]!
    expect(first.projectName).not.toBe('')
    expect(first.staffNo).not.toBe('')
    expect(first.pieceArea).toBeGreaterThan(0)

    const byCode = await call(api.GET('/scans', { ...admin, params: { query: { keyword: first.barCode } } }))
    expect(byCode.items.every((s) => s.barCode === first.barCode)).toBe(true)
    expect(byCode.total).toBeGreaterThan(0)

    const byStaff = await call(api.GET('/scans', { ...admin, params: { query: { staff: first.staffNo } } }))
    expect(byStaff.items.every((s) => s.staffNo === first.staffNo)).toBe(true)
    const none = await call(api.GET('/scans', { ...admin, params: { query: { staff: 'nobody-here' } } }))
    expect(none.total).toBe(0)

    const wx = await login('waixie', 'demo123')
    const scoped = await call(api.GET('/scans', { ...wx, params: { query: { size: 200 } } }))
    expect(scoped.total).toBeLessThan(page.total)
  })

  it('records QC verdicts on scan and feeds quality lists / reports', async () => {
    const admin = await login('admin', 'admin123')
    const jobs = await call(api.GET('/jobs', admin))
    expect(jobs.filter((j) => j.isQc).map((j) => j.code)).toEqual(['25', '45'])

    // Seeded data already contains inspections, including reinspected and pending fails.
    const seeded = await call(api.GET('/qc-scans', { ...admin, params: { query: { size: 5 } } }))
    expect(seeded.total).toBeGreaterThan(5)
    expect(seeded.summary.passed + seeded.summary.failed).toBe(seeded.summary.inspected)
    expect(seeded.summary.reinspected + seeded.summary.pending).toBe(seeded.summary.failed)
    expect(seeded.summary.pending).toBeGreaterThan(0)
    for (let i = 1; i < seeded.items.length; i++) expect(seeded.items[i - 1]!.inspectedAt >= seeded.items[i]!.inspectedAt).toBe(true)
    const pending = await call(api.GET('/qc-scans', { ...admin, params: { query: { reinspected: false, size: 200 } } }))
    expect(pending.total).toBe(seeded.summary.pending)
    expect(pending.items.every((r) => r.result === 'fail' && !r.reinspected)).toBe(true)

    // Pick a label that sits at 钣金下线 (20) and inspect it at 25.
    const labels = await call(api.GET('/labels', { ...admin, params: { query: { size: 200 } } }))
    const job20 = jobs.find((j) => j.code === '20')!
    const label = labels.items.find((l) => l.lastJobId === job20.id && !l.cancelled && !pending.items.some((p) => p.labelId === l.id))!
    expect(label).toBeDefined()
    await expect(call(api.POST('/scans', { ...admin, body: { barCode: label.barCode, jobCode: '25' } }))).rejects.toMatchObject({
      code: 'qc_result_required',
    })
    const failed = await call(
      api.POST('/scans', {
        ...admin,
        body: {
          barCode: label.barCode,
          jobCode: '25',
          result: 'fail',
          defectJobId: job20.id,
          defectCategory: '外观',
          defectReason: '测试划伤',
        },
      }),
    )
    expect(failed.scan).toBeNull()
    expect(failed.qc?.result).toBe('fail')
    expect(failed.qc?.defectJobName).toBe(job20.name)
    expect(failed.label.lastJobId).toBe(job20.id)
    await expect(call(api.POST('/scans', { ...admin, body: { barCode: label.barCode, jobCode: '30' } }))).rejects.toMatchObject({
      code: 'out_of_order',
    })
    const passed = await call(api.POST('/scans', { ...admin, body: { barCode: label.barCode, jobCode: '25', result: 'pass' } }))
    expect(passed.scan?.jobName).toBe('钣金质检')
    expect(passed.qc?.result).toBe('pass')

    const mine = await call(api.GET('/qc-scans', { ...admin, params: { query: { keyword: label.barCode } } }))
    expect(mine.total).toBe(2)
    expect(mine.items[0]!.result).toBe('pass')
    expect(mine.items[1]!.result).toBe('fail')
    expect(mine.items[1]!.reinspected).toBe(true)
    expect(mine.items[1]!.reinspectedAt).toBe(mine.items[0]!.inspectedAt)

    const report = await call(api.GET('/reports/quality', { ...admin, params: { query: { keyword: '测试划伤' } } }))
    expect(report.total).toBe(1)
    expect(report.items[0]).toMatchObject({ qty: 1, reinspectedQty: 1, defectCategory: '外观', lotNo: label.lotNo })
    expect(report.summary).toMatchObject({ inspected: 1, failed: 1, passRate: 0 })
    const overall = await call(api.GET('/reports/quality', { ...admin, params: { query: { from: '2000-01-01' } } }))
    expect(overall.summary.passRate).toBeGreaterThan(0)

    const daily = await call(api.GET('/reports/defect-daily', { ...admin, params: { query: { keyword: label.lotNo } } }))
    expect(daily.jobs.map((j) => j.code)).toEqual(['25', '45'])
    const row = daily.items.find((r) => r.batchId === label.batchId)!
    expect(row.cells).toHaveLength(2)
    expect(row.cells[0]!.failedQty).toBeGreaterThanOrEqual(1)
    expect(row.cells[0]!.reinspectedQty).toBeGreaterThanOrEqual(1)
    expect(daily.summary.failedQty).toBe(row.failedQty)

    const wx = await login('waixie', 'demo123')
    const scoped = await call(api.GET('/qc-scans', { ...wx, params: { query: { size: 200 } } }))
    expect(scoped.items.every((r) => r.barCode !== label.barCode)).toBe(true)
  })
})
