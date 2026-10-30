import 'vue-router'

declare module 'vue-router' {
  interface RouteMeta {
    title?: string
    /** Reachable without login. */
    anonymous?: boolean
    /** Reachable by any logged-in user regardless of menu grants. */
    public?: boolean
  }
}
