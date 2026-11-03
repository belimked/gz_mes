// @vitest-environment node
// Node environment: under jsdom, Node 24's undici rejects jsdom's File in Request.formData().
import { beforeEach, describe, expect, it } from 'vitest'
import { call, makeClient } from '@/api/client'
import { demoFetch, resetDemo } from '@/api/demo'

const api = makeClient('http://demo.local/api', demoFetch)

async function login(username: string, password: string) {
  const r = await call(api.POST('/auth/login', { body: { username, password } }))
  return { headers: { Authorization: `Bearer ${r.token}` } }
}

beforeEach(() => {
  resetDemo()
})

describe('demo drawing api', () => {
  it('lists seeded drawings and supports upload/download/delete with scope', async () => {
    const admin = await login('admin', 'admin123')
    const page = await call(api.GET('/drawing-batches', { ...admin, params: { query: { uploadState: 'complete' } } }))
    expect(page.total).toBeGreaterThan(0)
    const id = page.items[0]!.id
    const seeded = await call(api.GET('/batches/{id}/drawings', { ...admin, params: { path: { id } } }))
    expect(seeded.length).toBeGreaterThan(0)
    const boundary = 'demo-drawing-boundary'
    const multipart = `--${boundary}\r\nContent-Disposition: form-data; name="kind"\r\n\r\nforming\r\n--${boundary}\r\nContent-Disposition: form-data; name="file"; filename="测试.pdf"\r\nContent-Type: application/pdf\r\n\r\n%PDF-1.4 demo\r\n--${boundary}--\r\n`
    const response = await demoFetch(
      new Request(`http://demo.local/api/batches/${id}/drawings`, {
        method: 'POST',
        headers: { ...admin.headers, 'Content-Type': `multipart/form-data; boundary=${boundary}` },
        body: multipart,
      }),
    )
    expect(response.status).toBe(201)
    const drawing = (await response.json()) as { id: number; sha256: string }
    expect(drawing.sha256).toHaveLength(64)
    const file = await demoFetch(new Request(`http://demo.local/api/drawings/${drawing.id}/file`, { headers: admin.headers }))
    expect(await file.text()).toBe('%PDF-1.4 demo')
    const outsider = await login('waixie', 'demo123')
    expect((await api.GET('/batches/{id}/drawings', { ...outsider, params: { path: { id } } })).response.status).toBe(404)
    expect((await api.DELETE('/drawings/{id}', { ...admin, params: { path: { id: drawing.id } } })).response.status).toBe(204)
    expect((await demoFetch(new Request(`http://demo.local/api/drawings/${drawing.id}/file`, { headers: admin.headers }))).status).toBe(404)
  })
})
