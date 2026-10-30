import { defineStore } from 'pinia'
import { computed, ref } from 'vue'
import { call, client, tokenStore, type Schemas } from '@/api/client'

export const useAuthStore = defineStore('auth', () => {
  const token = ref<string | null>(tokenStore.get())
  const user = ref<Schemas['User'] | null>(null)
  const menus = ref<Schemas['MenuItem'][]>([])

  const loggedIn = computed(() => token.value !== null)

  /** Leaf menu paths the user may open. */
  const allowedPaths = computed(() => {
    const out = new Set<string>()
    const walk = (items: Schemas['MenuItem'][]) => {
      for (const m of items) {
        if (m.children?.length) walk(m.children)
        else out.add(m.path)
      }
    }
    walk(menus.value)
    return out
  })

  async function login(username: string, password: string) {
    const r = await call(client.POST('/auth/login', { body: { username, password } }))
    token.value = r.token
    tokenStore.set(r.token)
    user.value = r.user
    await loadMe()
  }

  async function loadMe() {
    const r = await call(client.GET('/auth/me'))
    user.value = r.user
    menus.value = r.menus
  }

  function logout() {
    token.value = null
    user.value = null
    menus.value = []
    tokenStore.clear()
  }

  return { token, user, menus, loggedIn, allowedPaths, login, loadMe, logout }
})
