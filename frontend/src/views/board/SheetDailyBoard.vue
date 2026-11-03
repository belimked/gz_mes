<script setup lang="ts">
import { computed, ref } from 'vue'
import { call, client, errorMessage, type Schemas } from '@/api/client'
import BoardShell from '@/components/BoardShell.vue'

type Board = Schemas['SheetDailyBoard']
const data = ref<Board>({ rows: [], summary: { count: 0, orderArea: 0, dayPlanArea: 0, dayDoneArea: 0, doneCount: 0, byJob: [] } })
const refreshed = ref('')
const error = ref('')
let requestId = 0
async function refresh(date: string, workshop: string) {
  const id = ++requestId
  try {
    const result = await call(client.GET('/boards/sheet-daily', { params: { query: { date, ...(workshop ? { workshop } : {}) } } }))
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
  { label: '订单面积', value: data.value.summary.orderArea, unit: '㎡' },
  { label: '当日计划', value: data.value.summary.dayPlanArea, unit: '㎡' },
  { label: '当日完成', value: data.value.summary.dayDoneArea, unit: '㎡' },
  { label: '完成数', value: data.value.summary.doneCount },
])
</script>
<template>
  <BoardShell
    title="钣金日排程看板"
    filter-label="车间筛选"
    :rows="data.rows"
    :kpis="kpis"
    :refreshed="refreshed"
    :chart-labels="data.summary.byJob.map((x) => x.jobName || '未分配')"
    :chart-planned="data.summary.byJob.map((x) => x.dayPlanArea)"
    :chart-done="data.summary.byJob.map((x) => x.dayDoneArea)"
    @refresh="refresh"
  >
    <template #head
      ><thead>
        <tr>
          <th>#</th>
          <th>项目名称</th>
          <th>批次</th>
          <th>车间 / 工序</th>
          <th>订单数量</th>
          <th>订单面积/㎡</th>
          <th>日面积/㎡</th>
          <th>当日完成</th>
          <th>完成件数</th>
          <th>完成面积/㎡</th>
          <th>进度%</th>
        </tr>
      </thead></template
    >
    <template #rows="{ rows }"
      ><tr v-for="r in rows" :key="r.planId" :class="{ urgent: r.urgency === 'urgent', due: r.dueToday }">
        <td>{{ r.seq }}</td>
        <td>{{ r.projectName }}</td>
        <td>{{ r.lotNo }}</td>
        <td>{{ r.workshop }} / {{ r.jobName }}</td>
        <td>{{ r.orderQty }}</td>
        <td>{{ r.orderArea.toFixed(2) }}</td>
        <td>{{ r.dayPlanArea.toFixed(2) }}</td>
        <td>{{ r.dayDoneQty }} 件 / {{ r.dayDoneArea.toFixed(2) }} ㎡</td>
        <td>{{ r.doneQty }}</td>
        <td>{{ r.doneArea.toFixed(2) }}</td>
        <td style="min-width: 125px">
          <a-progress :percent="r.progress" size="small" /><a-tag
            :color="r.state === 'done' ? 'green' : r.state === 'in_progress' ? 'blue' : 'orange'"
            >{{ r.state === 'done' ? '已完成' : r.state === 'in_progress' ? '进行中' : '未开始' }}</a-tag
          ><a-tag v-if="r.dueToday" color="red">今日到期</a-tag>
        </td>
      </tr></template
    >
  </BoardShell>
  <a-alert v-if="error" type="error" :message="error" show-icon />
</template>
