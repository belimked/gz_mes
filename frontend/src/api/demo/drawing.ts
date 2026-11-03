import type { components } from '../schema'
import type { DemoUser } from './data'

type S = components['schemas']
type DB = { drawings: S['Drawing'][]; batches: S['Batch'][]; contracts: S['Contract'][]; nextId: number }
const files = new Map<number, ArrayBuffer>()
const seedImage = 'iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVQIHWP4z8DwHwAFgAI/ScL/nwAAAABJRU5ErkJggg=='
export function seedDrawings(batches: S['Batch'][]): S['Drawing'][] {
  return batches
    .filter((b) => b.factoryId === 1)
    .slice(0, 5)
    .flatMap((b, i) =>
      (i < 2 ? ['unfold', 'forming', 'nesting'] : i < 4 ? ['unfold'] : []).map((kind, k) => ({
        id: 90000 + i * 10 + k,
        batchId: b.id,
        kind: kind as S['Drawing']['kind'],
        drawingNo: `DEMO-${i + 1}-${k + 1}`,
        remark: '演示图纸',
        originalName: `${b.lotNo}-${kind}.png`,
        contentType: 'image/png',
        sizeBytes: 70,
        sha256: '',
        storageKey: `demo/${i}/${k}`,
        uploadedBy: 1,
        uploaderName: '演示管理员',
        uploadedAt: b.createdAt,
      })),
    )
}
function error(status: number, message: string) {
  return new Response(JSON.stringify({ code: status === 404 ? 'not_found' : 'invalid_request', message }), {
    status,
    headers: { 'Content-Type': 'application/json' },
  })
}
function json(status: number, value: unknown) {
  return new Response(JSON.stringify(value), { status, headers: { 'Content-Type': 'application/json' } })
}
export async function drawingRoute(
  req: Request,
  path: string,
  params: URLSearchParams,
  db: DB,
  user: DemoUser,
  save: () => void,
): Promise<Response | null> {
  const inScope = (batchId: number) => db.batches.some((b) => b.id === batchId && (user.isAdmin || user.factoryIds.includes(b.factoryId)))
  if (req.method === 'GET' && path === '/drawing-batches') {
    const page = Number(params.get('page') ?? 1)
    const size = Number(params.get('size') ?? 20)
    if (!Number.isInteger(page) || page < 1 || !Number.isInteger(size) || size < 1 || size > 200) return error(400, '分页无效')
    const keyword = (params.get('keyword') ?? '').toLowerCase()
    const state = params.get('uploadState')
    const begin = params.get('beginDate')
    const end = params.get('endDate')
    if (begin && end && begin > end) return error(400, '日期无效')
    const items: S['DrawingBatch'][] = db.batches
      .filter((b) => inScope(b.id))
      .map((b) => {
        const drawings = db.drawings.filter((d) => d.batchId === b.id)
        return {
          id: b.id,
          lotNo: b.lotNo,
          projectName: b.projectName,
          customerName: b.customerName,
          createdAt: b.createdAt,
          unfoldCount: drawings.filter((d) => d.kind === 'unfold').length,
          formingCount: drawings.filter((d) => d.kind === 'forming').length,
          nestingCount: drawings.filter((d) => d.kind === 'nesting').length,
          lastUploadedAt:
            drawings
              .map((d) => d.uploadedAt)
              .sort()
              .at(-1) ?? null,
        }
      })
      .filter((b) => {
        const c = db.contracts.find((v) => v.id === db.batches.find((v) => v.id === b.id)?.contractId)
        const matched = `${b.lotNo} ${b.projectName} ${c?.contractNo ?? ''}`.toLowerCase().includes(keyword)
        const total = b.unfoldCount + b.formingCount + b.nestingCount
        return (
          matched &&
          (!begin || b.createdAt.slice(0, 10) >= begin) &&
          (!end || b.createdAt.slice(0, 10) <= end) &&
          (!state ||
            (state === 'none' && !total) ||
            (state === 'complete' && b.unfoldCount > 0 && b.formingCount > 0) ||
            (state === 'partial' && total > 0 && (!b.unfoldCount || !b.formingCount)))
        )
      })
      .sort((a, b) => b.createdAt.localeCompare(a.createdAt) || b.id - a.id)
    return json(200, { items: items.slice((page - 1) * size, page * size), total: items.length })
  }
  const batch = /^\/batches\/(\d+)\/drawings$/.exec(path)
  if (batch && (req.method === 'GET' || req.method === 'POST')) {
    const id = Number(batch[1])
    if (!inScope(id)) return error(404, '批次不存在')
    if (req.method === 'GET')
      return json(
        200,
        db.drawings.filter((d) => d.batchId === id).sort((a, b) => b.id - a.id),
      )
    const form = await req.formData()
    const file = form.get('file') as File | null
    const kind = form.get('kind')
    if (
      !file ||
      typeof file === 'string' ||
      typeof file.name !== 'string' ||
      !['unfold', 'forming', 'nesting'].includes(String(kind)) ||
      !/\.(pdf|dwg|dxf|png|jpe?g)$/i.test(file.name)
    )
      return error(400, '文件或类型无效')
    if (file.size > 50 * 1024 * 1024) return error(413, '文件超过上传限制')
    const ext = file.name.split('.').at(-1)!.toLowerCase()
    const contentType =
      ext === 'pdf'
        ? 'application/pdf'
        : ext === 'dwg'
          ? 'application/acad'
          : ext === 'dxf'
            ? 'application/dxf'
            : ext === 'png'
              ? 'image/png'
              : 'image/jpeg'
    const bytes = await file.arrayBuffer()
    const sha256 = Array.from(new Uint8Array(await crypto.subtle.digest('SHA-256', bytes)))
      .map((b) => b.toString(16).padStart(2, '0'))
      .join('')
    const drawing: S['Drawing'] = {
      id: db.nextId++,
      batchId: id,
      kind: kind as S['Drawing']['kind'],
      drawingNo: String(form.get('drawingNo') ?? ''),
      remark: String(form.get('remark') ?? ''),
      originalName: file.name,
      contentType,
      sizeBytes: file.size,
      sha256,
      storageKey: `demo/${Date.now()}/${file.name}`,
      uploadedBy: user.id,
      uploaderName: user.displayName,
      uploadedAt: new Date().toISOString(),
    }
    db.drawings.push(drawing)
    files.set(drawing.id, bytes)
    save()
    return json(201, drawing)
  }
  const fileRoute = /^\/drawings\/(\d+)\/file$/.exec(path)
  const deleteRoute = /^\/drawings\/(\d+)$/.exec(path)
  if ((fileRoute && req.method === 'GET') || (deleteRoute && req.method === 'DELETE')) {
    const id = Number((fileRoute ?? deleteRoute)![1])
    const d = db.drawings.find((v) => v.id === id)
    if (!d || !inScope(d.batchId)) return error(404, '图纸不存在')
    if (deleteRoute) {
      db.drawings.splice(db.drawings.indexOf(d), 1)
      files.delete(id)
      save()
      return new Response(null, { status: 204 })
    }
    const bytes = files.get(id) ?? Uint8Array.from(atob(seedImage), (c) => c.charCodeAt(0)).buffer
    return new Response(bytes, {
      headers: {
        'Content-Type': d.contentType,
        'Content-Disposition': `${params.get('download') === '1' ? 'attachment' : 'inline'}; filename*=UTF-8''${encodeURIComponent(d.originalName)}`,
      },
    })
  }
  return null
}
