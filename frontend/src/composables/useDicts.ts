import { computed, ref } from 'vue'
import { call, client, type Schemas } from '@/api/client'

type Kind = Schemas['DictItem']['kind']
const cache = ref<Partial<Record<Kind, Schemas['DictItem'][]>>>({})

export async function loadDicts(kind: Kind, force = false) {
  if (!cache.value[kind] || force) {
    const items = await call(client.GET('/dicts', { params: { query: { kind, includeStopped: true } } }))
    cache.value = { ...cache.value, [kind]: items }
  }
  return cache.value[kind] ?? []
}
export function invalidateDicts(kind?: Kind) {
  if (kind) {
    const next = { ...cache.value }
    delete next[kind]
    cache.value = next
  } else cache.value = {}
}
export function useDicts(kind: Kind) {
  const items = computed(() => cache.value[kind] ?? [])
  const options = (current?: () => string | undefined) =>
    computed(() =>
      items.value
        .filter((d) => !d.isStop || d.name === current?.())
        .map((d) => ({ value: d.name, label: d.name + (d.isStop ? '（已停用）' : '') })),
    )
  return { items, options, load: () => loadDicts(kind) }
}
