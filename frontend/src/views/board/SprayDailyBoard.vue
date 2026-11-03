<script setup lang="ts">
import { computed, ref } from 'vue'
import { call, client, errorMessage, type Schemas } from '@/api/client'
import BoardShell from '@/components/BoardShell.vue'

type Board = Schemas['SprayDailyBoard']
const data = ref<Board>({ rows: [], summary: { count: 0, plannedArea: 0, daySprayedArea: 0, sprayedArea: 0, doneCount: 0, byLine: [] } })
const refreshed = ref('')
const error = ref('')
let requestId = 0
async function refresh(date: string, paintLine: string) {
  const id = ++requestId
  try {
    const result = await call(client.GET('/boards/spray-daily', { params: { query: { date, ...(paintLine ? { paintLine } : {}) } } }))
    if (id === requestId) {
      data.value = result
      refreshed.value = new Date().toLocaleTimeString('zh-CN')
      error.value = ''
    }
  } catch (e) {
    if (id === requestId) error.value = errorMessage(e)
  }
}
const kpis = computed(() => [
  { label: '计划数', value: data.value.summary.count },
  { label: '计划面积', value: data.value.summary.plannedArea, unit: '㎡' },
  { label: '当日已喷', value: data.value.summary.daySprayedArea, unit: '㎡' },
  { label: '累计已喷', value: data.value.summary.sprayedArea, unit: '㎡' },
  { label: '完成数', value: data.value.summary.doneCount },
])
const stateName: Record<string, string> = { done: '已完成', in_progress: '喷涂中', ready: '待喷涂', waiting_sheet: '待钣金' }
const stateColor: Record<string, string> = { done: 'green', in_progress: 'blue', ready: 'cyan', waiting_sheet: 'orange' }
</script>
<template>
  <BoardShell
    title="喷涂日排产看板"
    filter-label="喷涂线筛选"
    :rows="data.rows"
    :kpis="kpis"
    :refreshed="refreshed"
    :chart-labels="data.summary.byLine.map((x) => x.paintLine || '未分配')"
    :chart-planned="data.summary.byLine.map((x) => x.plannedArea)"
    :chart-done="data.summary.byLine.map((x) => x.daySprayedArea)"
    @refresh="refresh"
  >
    <template #head
      ><thead>
        <tr>
          <th>#</th>
          <th>项目名称</th>
          <th>批次</th>
          <th>颜色</th>
          <th>喷涂线</th>
          <th>计划/㎡</th>
          <th>钣金/㎡</th>
          <th>已喷/㎡</th>
          <th>日喷/㎡</th>
          <th>未喷/㎡</th>
          <th>进度%</th>
          <th>状态</th>
        </tr>
      </thead></template
    >
    <template #rows="{ rows }"
      ><tr v-for="r in rows" :key="r.planId" :class="{ urgent: r.urgency === 'urgent', due: r.dueToday }">
        <td>{{ r.seq }}</td>
        <td>{{ r.projectName }}</td>
        <td>{{ r.lotNo }}</td>
        <td>{{ r.color }} {{ r.colorNo }}</td>
        <td>{{ r.paintLine }} {{ r.shift }}</td>
        <td>{{ r.plannedArea.toFixed(2) }}</td>
        <td>{{ r.sheetArea.toFixed(2) }}</td>
        <td>{{ r.sprayedArea.toFixed(2) }}</td>
        <td>{{ r.daySprayedArea.toFixed(2) }}</td>
        <td>{{ r.unsprayedArea.toFixed(2) }}</td>
        <td style="min-width: 125px"><a-progress :percent="r.progress" :show-info="true" size="small" /></td>
        <td>
          <a-tag :color="stateColor[r.state]">{{ stateName[r.state] }}</a-tag
          ><a-tag v-if="r.dueToday" color="red">今日到期</a-tag>
        </td>
      </tr></template
    >
  </BoardShell>
  <a-alert v-if="error" type="error" :message="error" show-icon />
</template>
