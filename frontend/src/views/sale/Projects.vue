<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { message } from 'ant-design-vue'
import type { FormInstance, TablePaginationConfig } from 'ant-design-vue'
import dayjs from 'dayjs'
import { call, client, errorMessage, type Schemas } from '@/api/client'

type Project = Schemas['Project']

const factories = ref<Schemas['Factory'][]>([])
const factoryName = computed(() => new Map(factories.value.map((f) => [f.id, f.name])))

const keyword = ref('')
const loading = ref(false)
const rows = ref<Project[]>([])
const pager = reactive({ current: 1, pageSize: 20, total: 0 })

const columns = [
  { title: '项目编号', dataIndex: 'code', width: 140 },
  { title: '项目名称', dataIndex: 'fullName', ellipsis: true },
  { title: '简称', dataIndex: 'shortName', width: 140 },
  { title: '厂区', dataIndex: 'factoryId', width: 110 },
  { title: '类型', dataIndex: 'kind', width: 90 },
  { title: '确认状态', dataIndex: 'confirmed', width: 100 },
  { title: '创建人', dataIndex: 'createdBy', width: 120 },
  { title: '创建时间', dataIndex: 'createdAt', width: 170 },
  { title: '操作', key: 'action', width: 90, fixed: 'right' as const },
]

async function load() {
  loading.value = true
  try {
    const r = await call(
      client.GET('/projects', {
        params: { query: { page: pager.current, size: pager.pageSize, keyword: keyword.value.trim() || undefined } },
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
  pager.current = 1
  void load()
}

function onTableChange(p: TablePaginationConfig) {
  pager.current = p.current ?? 1
  pager.pageSize = p.pageSize ?? 20
  void load()
}

async function confirm(p: Project) {
  try {
    await call(client.POST('/projects/{id}/confirm', { params: { path: { id: p.id } } }))
    message.success('已确认')
    await load()
  } catch (e) {
    message.error(errorMessage(e))
  }
}

// ---- create dialog ----
const open = ref(false)
const saving = ref(false)
const formRef = ref<FormInstance>()
const emptyForm = (): Schemas['ProjectCreate'] => ({
  factoryId: factories.value[0]?.id ?? 0,
  code: '',
  fullName: '',
  shortName: '',
  address: '',
  kind: 'offline',
})
const form = ref<Schemas['ProjectCreate']>(emptyForm())
const required = (msg: string) => [{ required: true, whitespace: true, message: msg }]

function openCreate() {
  form.value = emptyForm()
  open.value = true
}

async function save() {
  try {
    await formRef.value?.validate()
  } catch {
    return
  }
  saving.value = true
  try {
    await call(client.POST('/projects', { body: form.value }))
    message.success('已创建')
    open.value = false
    search()
  } catch (e) {
    message.error(errorMessage(e))
  } finally {
    saving.value = false
  }
}

onMounted(async () => {
  try {
    factories.value = await call(client.GET('/factories'))
  } catch (e) {
    message.error(errorMessage(e))
  }
  await load()
})
</script>

<template>
  <div>
    <a-space class="toolbar" wrap>
      <a-input-search
        v-model:value="keyword"
        placeholder="项目编号 / 名称 / 简称"
        allow-clear
        style="width: 280px"
        data-testid="project-search"
        @search="search"
      />
      <a-button type="primary" data-testid="project-new" @click="openCreate">新建项目</a-button>
    </a-space>

    <a-table
      row-key="id"
      size="middle"
      :columns="columns"
      :data-source="rows"
      :loading="loading"
      :scroll="{ x: 1100 }"
      :pagination="{ ...pager, showSizeChanger: true, showTotal: (t: number) => `共 ${t} 条` }"
      data-testid="project-table"
      @change="onTableChange"
    >
      <template #bodyCell="{ column, record }">
        <template v-if="column.dataIndex === 'factoryId'">{{ factoryName.get(record.factoryId) ?? record.factoryId }}</template>
        <template v-else-if="column.dataIndex === 'kind'">{{ record.kind === 'online' ? '线上' : '线下' }}</template>
        <template v-else-if="column.dataIndex === 'confirmed'">
          <a-tag :color="record.confirmed ? 'green' : 'default'">{{ record.confirmed ? '已确认' : '未确认' }}</a-tag>
        </template>
        <template v-else-if="column.dataIndex === 'createdAt'">{{ dayjs(record.createdAt).format('YYYY-MM-DD HH:mm') }}</template>
        <template v-else-if="column.key === 'action'">
          <a-popconfirm v-if="!record.confirmed" title="确认后项目将锁定，不能再修改。确定？" @confirm="confirm(record as Project)">
            <a :data-testid="`confirm-${record.code}`">确认</a>
          </a-popconfirm>
        </template>
      </template>
    </a-table>

    <a-modal v-model:open="open" title="新建项目" :confirm-loading="saving" ok-text="保存" destroy-on-close @ok="save">
      <a-form ref="formRef" :model="form" :label-col="{ span: 5 }">
        <a-form-item label="厂区" name="factoryId" :rules="[{ required: true, message: '请选择厂区' }]">
          <a-select v-model:value="form.factoryId" :options="factories.map((f) => ({ value: f.id, label: f.name }))" />
        </a-form-item>
        <a-form-item label="项目编号" name="code" :rules="required('请输入项目编号')">
          <a-input v-model:value="form.code" :maxlength="64" data-testid="form-code" />
        </a-form-item>
        <a-form-item label="项目名称" name="fullName" :rules="required('请输入项目名称')">
          <a-input v-model:value="form.fullName" :maxlength="200" data-testid="form-fullName" />
        </a-form-item>
        <a-form-item label="简称" name="shortName">
          <a-input v-model:value="form.shortName" :maxlength="100" />
        </a-form-item>
        <a-form-item label="地址" name="address">
          <a-input v-model:value="form.address" :maxlength="300" />
        </a-form-item>
        <a-form-item label="类型" name="kind">
          <a-radio-group v-model:value="form.kind">
            <a-radio value="offline">线下</a-radio>
            <a-radio value="online">线上</a-radio>
          </a-radio-group>
        </a-form-item>
      </a-form>
    </a-modal>
  </div>
</template>

<style scoped>
.toolbar {
  margin-bottom: 16px;
}
</style>
