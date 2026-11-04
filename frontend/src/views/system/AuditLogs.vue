<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { message } from 'ant-design-vue'
import dayjs from 'dayjs'
import { call, client, errorMessage, type Schemas } from '@/api/client'
import PageCard from '@/components/PageCard.vue'
import FilterCard from '@/components/FilterCard.vue'
import { auditAction } from '@/utils/auditActions'
const filters = reactive({
  dates: [] as string[],
  keyword: '',
  method: undefined as 'POST' | 'PUT' | 'PATCH' | 'DELETE' | undefined,
  result: undefined as 'ok' | 'fail' | undefined,
})
const applied = reactive({ ...filters })
const rows = ref<Schemas['AuditLog'][]>([])
const pager = reactive({ current: 1, pageSize: 20, total: 0 })
const loading = ref(false)
const columns = ['时间', '操作人', '动作', '方法', '路径', '状态', '耗时 ms', 'IP'].map((title, i) => ({
  title,
  dataIndex: ['createdAt', 'displayName', 'route', 'method', 'path', 'status', 'durationMs', 'ip'][i],
}))
async function load() {
  loading.value = true
  try {
    const r = await call(
      client.GET('/audit-logs', {
        params: {
          query: {
            page: pager.current,
            size: pager.pageSize,
            from: applied.dates[0],
            to: applied.dates[1],
            keyword: applied.keyword || undefined,
            method: applied.method,
            result: applied.result,
          },
        },
      }),
    )
    rows.value = r.items
    pager.total = r.total
  } catch (e) {
    message.error(errorMessage(e))
  } finally {
    loading.value = false
  }
}
function search() {
  Object.assign(applied, filters)
  pager.current = 1
  void load()
}
function reset() {
  Object.assign(filters, { dates: [], keyword: '', method: undefined, result: undefined })
  search()
}
onMounted(load)
</script>
<template>
  <h1>操作日志</h1>
  <FilterCard @search="search" @reset="reset"
    ><label
      >日期范围<a-range-picker
        :value="filters.dates.length ? [dayjs(filters.dates[0]), dayjs(filters.dates[1])] : undefined"
        @change="(_date: unknown, dates: [string, string]) => (filters.dates = dates || [])" /></label
    ><label>关键字<a-input v-model:value="filters.keyword" placeholder="用户 / 路径" @press-enter="search" /></label
    ><label
      >方法<a-select
        v-model:value="filters.method"
        allow-clear
        :options="['POST', 'PUT', 'PATCH', 'DELETE'].map((x) => ({ value: x, label: x }))" /></label
    ><label
      >结果<a-select
        v-model:value="filters.result"
        allow-clear
        :options="[
          { value: 'ok', label: '成功' },
          { value: 'fail', label: '失败' },
        ]" /></label></FilterCard
  ><PageCard
    ><a-table
      row-key="id"
      :columns="columns"
      :data-source="rows"
      :loading="loading"
      :pagination="{ ...pager, showSizeChanger: true }"
      @change="
        (p: { current?: number; pageSize?: number }) => {
          pager.current = p.current ?? 1
          pager.pageSize = p.pageSize ?? 20
          load()
        }
      "
      ><template #bodyCell="{ column, record }"
        ><template v-if="column.dataIndex === 'createdAt'">{{ dayjs(record.createdAt).format('YYYY-MM-DD HH:mm:ss') }}</template
        ><template v-else-if="column.dataIndex === 'displayName'"
          >{{ record.displayName || record.username }} ({{ record.username }})</template
        ><template v-else-if="column.dataIndex === 'route'">{{ auditAction(record.method, record.route) }}</template
        ><template v-else-if="column.dataIndex === 'status'"
          ><a-tag :color="record.status < 400 ? 'green' : 'red'">{{ record.status }}</a-tag></template
        ></template
      ></a-table
    ></PageCard
  >
</template>
