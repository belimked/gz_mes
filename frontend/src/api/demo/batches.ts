import type { Tone } from '@/theme'
export interface Batch {
  id: string
  project: string
  section: string
  material: string
  thickness: string
  color: string
  quantity: number
  area: number
  priceArea: number
  status: string
  tone: Tone
  current: number
  progress: number
  team: string
  qc: string
  due: string
  start: string
  urgent: boolean
  factoryId: number
  uid: string
  components: { code: string; spec: string; quantity: number }[]
}
const projects = ['演示·滨湖科创中心', '演示·云谷产业园二期', '演示·江畔文化馆', '演示·星河学校体育馆', '演示·南湾商业综合体']
const stages = [1, 5, 3, 7, 9, 0, 10, 4, 2, 6, 8, 1, 5, 10, 0, 3, 7, 9]
export const batches: Batch[] = stages.map((current, i) => {
  const urgent = i === 1 || i === 11
  const done = current === 10
  const quantity = 40 + i * 8
  return {
    id: `PC-20241018-${String(i + 1).padStart(2, '0')}`,
    project: projects[i % 5]!,
    section: `${'ABCDE'[i % 5]}区幕墙 · ${i + 1}号板块`,
    material: i % 2 ? '铝合金 3003-H14' : '铝单板',
    thickness: `${(2.5 + (i % 4) * 0.5).toFixed(1)}mm`,
    color: i % 3 ? 'RAL 7016 亚光灰' : 'RAL 9003 象牙白',
    quantity,
    area: Number((quantity * 3.006).toFixed(2)),
    priceArea: Number((quantity * 3.226).toFixed(2)),
    status: urgent ? '加急逾期' : done ? '已完工' : current === 0 ? '待排产' : '生产中',
    tone: urgent ? 'error' : done ? 'success' : current === 0 ? 'neutral' : 'processing',
    current,
    progress: done ? 100 : current === 0 ? 0 : Math.min(95, current * 9 + 7),
    team: `${['钣金一班', '涂装二班', '成品班组'][i % 3]}`,
    qc: urgent ? '待复核' : done ? '合格' : '待检',
    due: `2026-05-${String(2 + i).padStart(2, '0')}`,
    start: `2026-04-${String(10 + i).padStart(2, '0')}`,
    urgent,
    factoryId: i % 5 === 2 ? 2 : 1,
    uid: `MES-AL-20241018-${String(9182 + i).padStart(6, '0')}-Q`,
    components: [1, 2, 3].map((n) => ({
      code: `YG-${String(i + 1).padStart(2, '0')}-${String(n).padStart(2, '0')}`,
      spec: `${1200 + n * 100} × ${800 + n * 50} × ${(2.5 + (i % 4) * 0.5).toFixed(1)} mm`,
      quantity: Math.floor(quantity / 3),
    })),
  }
})
