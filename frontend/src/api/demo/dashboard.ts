import type { Tone } from '@/theme'
export const kpis = [
  { title: '今日计划投产', value: '1,280', unit: '件 / 3,456.8 m²', sub: '较昨日 +8.4%', note: '环比上周 98.0%', icon: '▣' },
  { title: '钣金加工进度', value: '892', unit: '件已下线 (69.7%)', sub: '开料 / 折弯 / 冲角成型', note: '工段稼动 92%', icon: '⚒' },
  { title: '喷涂完工', value: '645', unit: '件 / 1,820.4 m²', sub: '质检合格率 99.2%', note: '膜厚均值 42μm', icon: '▤' },
  { title: '成品装架与待发', value: '420', unit: '件 / 18 铁架', sub: '待发 ¥186,400', note: '3 车排队', icon: '▥' },
]
export const pipeline: { count: string; label: string; tone: Tone }[] = [
  ['160 件', '连续数控切', 'neutral'],
  ['185 件', '4 台折弯机', 'processing'],
  ['98 件', '已码盘缓冲', 'neutral'],
  ['12 件待检', '角位复核中', 'warning'],
  ['140 件', '前处理除油', 'neutral'],
  ['220 件', '立式氟碳线', 'processing'],
  ['85 件', '固化炉出件', 'neutral'],
  ['99.4%', '测厚色差合格', 'success'],
  ['110 件', '贴保护膜', 'neutral'],
  ['18 架 / 340 件', '铁架打包', 'processing'],
  ['已发 2 车', '签收联生成', 'success'],
].map(([count, label, tone]) => ({ count: count!, label: label!, tone: tone as Tone }))
// Dashboard-only presentation fixtures; scheduling queries use the batches API.
export const activeBatches = [1, 5, 3, 7, 9, 0].map((current, i) => ({
  id: `PC-20241018-${String(i + 1).padStart(2, '0')}`,
  project: ['演示·滨湖科创中心', '演示·云谷产业园二期', '演示·江畔文化馆'][i % 3]!,
  section: `${'ABCDEF'[i]}区幕墙 · ${i + 1}号板块`,
  material: i % 2 ? '铝合金 3003-H14' : '铝单板',
  thickness: `${(2.5 + (i % 4) * 0.5).toFixed(1)}mm`,
  quantity: 40 + i * 8,
  area: Number(((40 + i * 8) * 3.006).toFixed(2)),
  current,
  progress: current * 9 + 7,
  team: ['钣金一班', '涂装二班', '成品班组'][i % 3]!,
  qc: i === 1 ? '待复核' : '待检',
  urgent: i === 1,
}))
export const alerts: { title: string; detail: string; tone: Tone }[] = [
  { title: '现场吊装加急催单', detail: '演示·云谷产业园二期 · 今晚前完成喷涂下挂入库', tone: 'error' },
  { title: '质检偏差提示', detail: 'PC-20241018-04 · 喷涂膜厚待复核', tone: 'warning' },
  { title: '交接缺件核查', detail: '钣金转喷涂环节缺少 2 件扫码固定板', tone: 'neutral' },
]
