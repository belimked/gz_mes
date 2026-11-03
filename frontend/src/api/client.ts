import createClient, { type Middleware } from 'openapi-fetch'
import type { components, paths } from './schema'
import { demoFetch } from './demo'

export type Schemas = components['schemas']

export const isDemo = import.meta.env.VITE_DEMO === 'true'

const TOKEN_KEY = 'gz-mes-token'

export const tokenStore = {
  get: (): string | null => localStorage.getItem(TOKEN_KEY),
  set: (t: string) => localStorage.setItem(TOKEN_KEY, t),
  clear: () => localStorage.removeItem(TOKEN_KEY),
}

let onUnauthorized: () => void = () => {}

/** Called when an authenticated request returns 401 (expired or revoked token). */
export function setUnauthorizedHandler(fn: () => void) {
  onUnauthorized = fn
}

const authMiddleware: Middleware = {
  onRequest({ request }) {
    const t = tokenStore.get()
    if (t) request.headers.set('Authorization', `Bearer ${t}`)
    return request
  },
  onResponse({ request, response }) {
    if (response.status === 401 && !request.url.endsWith('/auth/login')) onUnauthorized()
    return response
  },
}

export function makeClient(baseUrl: string, fetchImpl?: (req: Request) => Promise<Response>) {
  return createClient<paths>({ baseUrl, ...(fetchImpl ? { fetch: fetchImpl } : {}) })
}

export const client = makeClient('/api', isDemo ? demoFetch : undefined)
client.use(authMiddleware)

/** Binary/multipart transport, sharing the same bearer token and demo adapter. */
export async function fileRequest(path: string, init: RequestInit = {}): Promise<Response> {
  const headers = new Headers(init.headers)
  const token = tokenStore.get()
  if (token) headers.set('Authorization', `Bearer ${token}`)
  const request = new Request(new URL(`/api${path}`, window.location.origin), { ...init, headers })
  const response = await (isDemo ? demoFetch(request) : fetch(request))
  if (response.status === 401) onUnauthorized()
  if (!response.ok) {
    const body = (await response.json().catch(() => ({}))) as Partial<Schemas['Error']>
    throw new ApiError(response.status, body.code ?? `http_${response.status}`, body.message ?? `请求失败（${response.status}）`)
  }
  return response
}

export class ApiError extends Error {
  constructor(
    public readonly status: number,
    public readonly code: string,
    message: string,
  ) {
    super(message)
  }
}

/** Unwraps an openapi-fetch result, throwing ApiError on non-2xx. */
export async function call<T>(p: Promise<{ data?: T; error?: unknown; response: Response }>): Promise<T> {
  const { data, error, response } = await p
  if (error !== undefined || !response.ok) {
    const e = (error ?? {}) as Partial<Schemas['Error']>
    throw new ApiError(response.status, e.code ?? `http_${response.status}`, e.message ?? `请求失败（${response.status}）`)
  }
  return data as T
}

export function errorMessage(e: unknown): string {
  if (e instanceof ApiError) return e.message
  if (e instanceof Error) return e.message
  return String(e)
}
