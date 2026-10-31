/** Column definition for CSV export. `value` turns a row into a cell string / number. */
export interface ExportColumn<T> {
  title: string
  value: (row: T) => string | number | null | undefined
}

function cell(v: string | number | null | undefined): string {
  if (v === null || v === undefined) return ''
  const s = String(v)
  return /[",\r\n]/.test(s) ? `"${s.replace(/"/g, '""')}"` : s
}

/** Builds CSV text (CRLF, with header). Exported for tests. */
export function toCsv<T>(columns: ExportColumn<T>[], rows: T[]): string {
  const lines = [columns.map((c) => cell(c.title)).join(',')]
  for (const r of rows) lines.push(columns.map((c) => cell(c.value(r))).join(','))
  return lines.join('\r\n')
}

/** Triggers a browser download of UTF-8 CSV (with BOM so Excel detects the encoding). */
export function downloadCsv<T>(filename: string, columns: ExportColumn<T>[], rows: T[]) {
  const blob = new Blob(['\ufeff', toCsv(columns, rows)], { type: 'text/csv;charset=utf-8' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = filename.endsWith('.csv') ? filename : `${filename}.csv`
  document.body.appendChild(a)
  a.click()
  a.remove()
  URL.revokeObjectURL(url)
}

/**
 * Fetches every page of a paged endpoint. `fetchPage(page, size)` must return items + total.
 * Stops at `max` rows to keep exports bounded.
 */
export async function fetchAllPages<T>(
  fetchPage: (page: number, size: number) => Promise<{ items: T[]; total: number }>,
  { size = 200, max = 20000 }: { size?: number; max?: number } = {},
): Promise<T[]> {
  const out: T[] = []
  for (let page = 1; ; page++) {
    const { items, total } = await fetchPage(page, size)
    out.push(...items)
    if (items.length < size || out.length >= total || out.length >= max) break
  }
  return out.slice(0, max)
}

export function exportFilename(prefix: string, range?: [string | undefined, string | undefined]) {
  const d = new Date()
  const stamp = `${d.getFullYear()}${String(d.getMonth() + 1).padStart(2, '0')}${String(d.getDate()).padStart(2, '0')}`
  const r = range?.[0] ? `_${range[0]}_${range[1] ?? range[0]}` : ''
  return `${prefix}${r}_${stamp}.csv`
}
