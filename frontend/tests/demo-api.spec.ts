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
})
