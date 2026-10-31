import { describe, expect, it } from 'vitest'
import { exportFilename, fetchAllPages, toCsv } from '@/utils/export'

describe('export utils', () => {
  it('builds CSV with header, escaping and CRLF', () => {
    const csv = toCsv<{ a: string; n: number | null }>(
      [
        { title: '名称', value: (r) => r.a },
        { title: '数量', value: (r) => r.n },
      ],
      [
        { a: 'plain', n: 1 },
        { a: 'has,comma', n: null },
        { a: 'say "hi"\nline2', n: 0 },
      ],
    )
    expect(csv).toBe('名称,数量\r\nplain,1\r\n"has,comma",\r\n"say ""hi""\nline2",0')
  })

  it('fetches every page and stops at total / max', async () => {
    const calls: number[] = []
    const fetchPage = async (page: number, size: number) => {
      calls.push(page)
      const start = (page - 1) * size
      return { items: Array.from({ length: Math.max(0, Math.min(size, 450 - start)) }, (_, i) => start + i), total: 450 }
    }
    const all = await fetchAllPages(fetchPage, { size: 200 })
    expect(all).toHaveLength(450)
    expect(calls).toEqual([1, 2, 3])
    expect(await fetchAllPages(fetchPage, { size: 200, max: 250 })).toHaveLength(250)
  })

  it('names files with range and date stamp', () => {
    expect(exportFilename('扫描记录', ['2026-10-01', '2026-10-31'])).toMatch(/^扫描记录_2026-10-01_2026-10-31_\d{8}\.csv$/)
    expect(exportFilename('生产平衡表')).toMatch(/^生产平衡表_\d{8}\.csv$/)
  })
})
