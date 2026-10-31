<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { message } from 'ant-design-vue'
import type { FormInstance } from 'ant-design-vue'
import { call, client, errorMessage, type Schemas } from '@/api/client'
import { useAuthStore } from '@/stores/auth'
import PageCard from '@/components/PageCard.vue'
import FilterCard from '@/components/FilterCard.vue'
import TableToolbar from '@/components/TableToolbar.vue'
import CodeText from '@/components/CodeText.vue'
import StatusTag from '@/components/StatusTag.vue'

type Factory = Schemas['Factory']
const auth = useAuthStore()
const rows = ref<Factory[]>([])
const loading = ref(false)
const filters = ref({ keyword: '', kind: undefined as Factory['kind'] | undefined, isStop: undefined as boolean | undefined })
const applied = ref({ ...filters.value })
const visible = computed(() =>
  rows.value.filter(
    (f) =>
      (!applied.value.keyword || `${f.code} ${f.name}`.toLowerCase().includes(applied.value.keyword.trim().toLowerCase())) &&
      (applied.value.kind === undefined || f.kind === applied.value.kind) &&
      (applied.value.isStop === undefined || f.isStop === applied.value.isStop),
  ),
)
const columns = computed(() => [
  { title: '编码', dataIndex: 'code', width: 160 },
  { title: '名称', dataIndex: 'name' },
  { title: '类型', dataIndex: 'kind', width: 130 },
  { title: '状态', dataIndex: 'isStop', width: 120 },
  ...(auth.user?.isAdmin ? [{ title: '操作', key: 'action', width: 160 }] : []),
])
async function load() {
  loading.value = true
  try {
    rows.value = await call(client.GET('/factories'))
  } catch (e) {
    message.error(errorMessage(e))
  } finally {
    loading.value = false
  }
}
function search() {
  applied.value = { ...filters.value }
}
function reset() {
  filters.value = { keyword: '', kind: undefined, isStop: undefined }
  search()
}
const open = ref(false)
const editing = ref<Factory>()
const saving = ref(false)
const formRef = ref<FormInstance>()
const form = ref({ code: '', name: '', kind: 'own' as Factory['kind'], isStop: false })
function openCreate() {
  editing.value = undefined
  form.value = { code: '', name: '', kind: 'own', isStop: false }
  open.value = true
}
function openEdit(row: Factory) {
  editing.value = row
  form.value = { code: row.code, name: row.name, kind: row.kind, isStop: row.isStop }
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
    if (editing.value) {
      await call(
        client.PUT('/factories/{id}', {
          params: { path: { id: editing.value.id } },
          body: { name: form.value.name.trim(), isStop: form.value.isStop },
        }),
      )
    } else {
      await call(client.POST('/factories', { body: { code: form.value.code.trim(), name: form.value.name.trim(), kind: form.value.kind } }))
    }
    message.success('保存成功')
    open.value = false
    await load()
  } catch (e) {
    message.error(errorMessage(e))
  } finally {
    saving.value = false
  }
}
async function toggle(row: Factory) {
  try {
    await call(client.PUT('/factories/{id}', { params: { path: { id: row.id } }, body: { name: row.name, isStop: !row.isStop } }))
    message.success(row.isStop ? '已启用' : '已停用')
    await load()
  } catch (e) {
    message.error(errorMessage(e))
  }
}
onMounted(load)
</script>

<template>
  <div class="system-page">
    <div class="heading">
      <div>
        <h1>厂区管理</h1>
        <span>维护本厂与外协厂区档案</span>
      </div>
      <a-tag>共 {{ rows.length }} 个厂区</a-tag>
    </div>
    <a-alert v-if="!auth.user?.isAdmin" type="info" show-icon message="仅管理员可维护厂区" />
    <FilterCard @search="search" @reset="reset">
      <label>关键字<a-input v-model:value="filters.keyword" allow-clear placeholder="编码 / 名称" @press-enter="search" /></label>
      <label
        >类型<a-select
          v-model:value="filters.kind"
          allow-clear
          placeholder="全部类型"
          :options="[
            { value: 'own', label: '本厂' },
            { value: 'outsource', label: '外协' },
          ]"
      /></label>
      <label
        >状态<a-select
          v-model:value="filters.isStop"
          allow-clear
          placeholder="全部状态"
          :options="[
            { value: false, label: '启用' },
            { value: true, label: '停用' },
          ]"
      /></label>
    </FilterCard>
    <PageCard>
      <TableToolbar
        ><template #left
          ><a-button v-if="auth.user?.isAdmin" type="primary" data-testid="factory-new" @click="openCreate">新建厂区</a-button></template
        ><template #right><a-button @click="load">刷新</a-button></template></TableToolbar
      >
      <a-table
        row-key="id"
        size="middle"
        :columns="columns"
        :data-source="visible"
        :loading="loading"
        :pagination="false"
        data-testid="factory-table"
      >
        <template #bodyCell="{ column, record }">
          <template v-if="column.dataIndex === 'code'"><CodeText :value="record.code" /></template>
          <template v-else-if="column.dataIndex === 'kind'"
            ><StatusTag :tone="record.kind === 'own' ? 'processing' : 'warning'" :label="record.kind === 'own' ? '本厂' : '外协'"
          /></template>
          <template v-else-if="column.dataIndex === 'isStop'"
            ><StatusTag :tone="record.isStop ? 'neutral' : 'success'" :label="record.isStop ? '停用' : '启用'"
          /></template>
          <template v-else-if="column.key === 'action'"
            ><a-space
              ><a @click="openEdit(record as Factory)">编辑</a
              ><a-popconfirm :title="`确定${record.isStop ? '启用' : '停用'}此厂区？`" @confirm="toggle(record as Factory)"
                ><a>{{ record.isStop ? '启用' : '停用' }}</a></a-popconfirm
              ></a-space
            ></template
          >
        </template>
      </a-table>
    </PageCard>
    <a-modal
      v-model:open="open"
      :title="editing ? '编辑厂区' : '新建厂区'"
      ok-text="保存"
      :confirm-loading="saving"
      destroy-on-close
      @ok="save"
    >
      <a-form ref="formRef" :model="form" :label-col="{ span: 5 }">
        <a-form-item v-if="!editing" label="编码" name="code" :rules="[{ required: true, whitespace: true, message: '请输入编码' }]"
          ><a-input v-model:value="form.code" :maxlength="32"
        /></a-form-item>
        <a-form-item label="名称" name="name" :rules="[{ required: true, whitespace: true, message: '请输入名称' }]"
          ><a-input v-model:value="form.name" :maxlength="100"
        /></a-form-item>
        <a-form-item v-if="!editing" label="类型"
          ><a-select
            v-model:value="form.kind"
            :options="[
              { value: 'own', label: '本厂' },
              { value: 'outsource', label: '外协' },
            ]"
        /></a-form-item>
        <a-form-item v-if="editing" label="启用"
          ><a-switch :checked="!form.isStop" @change="(checked: boolean) => (form.isStop = !checked)"
        /></a-form-item>
      </a-form>
    </a-modal>
  </div>
</template>

<style scoped>
.system-page {
  display: grid;
  gap: 16px;
}
.heading {
  display: flex;
  justify-content: space-between;
  align-items: center;
}
.heading h1 {
  margin: 0 0 4px;
  font-size: 20px;
}
.heading span {
  color: var(--gz-muted);
  font-size: 12px;
}
label {
  display: grid;
  gap: 6px;
  min-width: 180px;
  color: var(--gz-secondary);
  font-size: 12px;
}
</style>
