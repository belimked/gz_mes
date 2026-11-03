import { beforeEach, describe, expect, it } from 'vitest'
import { call, makeClient } from '@/api/client'
import { demoFetch, resetDemo } from '@/api/demo'

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
    expect(wx.menus.map((m) => m.title)).toEqual(['销售管理', '排程管理'])
    expect(wx.menus[0]?.children?.map((c) => c.path)).toEqual(['/sale/projects', '/sale/contracts'])
    expect(wx.menus[1]?.children?.map((c) => c.path)).toEqual([
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
