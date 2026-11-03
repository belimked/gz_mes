<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { message } from 'ant-design-vue'
import { call, client, errorMessage, type Schemas } from '@/api/client'
import { invalidateDicts, loadDicts } from '@/composables/useDicts'
import PageCard from '@/components/PageCard.vue'
import TableToolbar from '@/components/TableToolbar.vue'

type Item = Schemas['DictItem']
type Kind = Item['kind']
const kind = ref<Kind>('workshop')
const items = ref<Item[]>([])
const editing = ref<Item>()
const visible = ref(false)
const saving = ref(false)
const form = reactive<Schemas['DictInput']>({
  kind: 'workshop',
  name: '',
  sort: 0,
  startTime: null,
  endTime: null,
  isStop: false,
  remark: '',
})
const columns = computed(() => [
  { title: '名称', dataIndex: 'name' },
  { title: '排序', dataIndex: 'sort' },
  ...(kind.value === 'shift' ? [{ title: '时间段', dataIndex: 'time' }] : []),
  { title: '状态', dataIndex: 'isStop' },
  { title: '备注', dataIndex: 'remark' },
  { title: '操作', dataIndex: 'action' },
])
async function load() {
  try {
    const current = kind.value
    const result = await loadDicts(current, true)
    if (current === kind.value) items.value = result
  } catch (e) {
    message.error(errorMessage(e))
  }
}
function open(item?: Item) {
  editing.value = item
  Object.assign(
    form,
    item ?? { kind: kind.value, name: '', sort: items.value.length + 1, startTime: null, endTime: null, isStop: false, remark: '' },
  )
  visible.value = true
}
async function save() {
  saving.value = true
  try {
    if (editing.value) await call(client.PUT('/dicts/{id}', { params: { path: { id: editing.value.id } }, body: { ...form } }))
    else await call(client.POST('/dicts', { body: { ...form } }))
    invalidateDicts(kind.value)
    visible.value = false
    message.success('保存成功')
    await load()
  } catch (e) {
    message.error(errorMessage(e))
  } finally {
    saving.value = false
  }
}
async function toggle(item: Item, checked: boolean) {
  try {
    await call(
      client.PUT('/dicts/{id}', {
        params: { path: { id: item.id } },
        body: {
          kind: item.kind,
          name: item.name,
          sort: item.sort,
          startTime: item.startTime,
          endTime: item.endTime,
          isStop: !checked,
          remark: item.remark,
        },
      }),
    )
    invalidateDicts(kind.value)
    await load()
  } catch (e) {
    message.error(errorMessage(e))
  }
}
onMounted(() => {
  void load()
})
</script>
<template>
  <div class="report">
    <div class="heading">
      <h1>基础字典</h1>
      <span>维护车间、喷涂线和班次；停用后已有数据仍可查看</span>
    </div>
    <PageCard>
      <a-tabs v-model:active-key="kind" @change="load">
        <a-tab-pane key="workshop" tab="车间" /><a-tab-pane key="paint_line" tab="喷涂线" /><a-tab-pane key="shift" tab="班次" />
      </a-tabs>
      <TableToolbar
        ><template #left>共 {{ items.length }} 条</template
        ><template #right><a-button type="primary" data-testid="dict-new" @click="open()">新增</a-button></template></TableToolbar
      >
      <a-table row-key="id" bordered size="middle" :data-source="items" :columns="columns" data-testid="dict-table">
        <template #bodyCell="{ column, record }">
          <template v-if="column.dataIndex === 'time'">{{
            record.startTime && record.endTime ? `${record.startTime}–${record.endTime}` : '—'
          }}</template>
          <template v-else-if="column.dataIndex === 'isStop'"
            ><a-tag :color="record.isStop ? 'default' : 'green'">{{ record.isStop ? '停用' : '启用' }}</a-tag></template
          >
          <template v-else-if="column.dataIndex === 'action'"
            ><a-space
              ><a @click="open(record)">编辑</a
              ><a-switch
                :checked="!record.isStop"
                checked-children="启用"
                un-checked-children="停用"
                @change="(checked: boolean) => toggle(record, checked)" /></a-space
          ></template>
        </template>
      </a-table>
    </PageCard>
    <a-modal v-model:open="visible" :title="editing ? '编辑字典项' : '新增字典项'" :confirm-loading="saving" @ok="save">
      <a-form layout="vertical">
        <a-form-item label="名称" required><a-input v-model:value="form.name" :maxlength="50" data-testid="dict-name" /></a-form-item>
        <a-form-item label="排序"><a-input-number v-model:value="form.sort" style="width: 100%" /></a-form-item>
        <template v-if="kind === 'shift'"
          ><a-form-item label="开始时间（HH:mm）"><a-input v-model:value="form.startTime" placeholder="08:00" /></a-form-item
          ><a-form-item label="结束时间（HH:mm）"><a-input v-model:value="form.endTime" placeholder="17:00" /></a-form-item
        ></template>
        <a-form-item label="备注"><a-textarea v-model:value="form.remark" /></a-form-item>
      </a-form>
    </a-modal>
  </div>
</template>
<style scoped>
.report {
  display: grid;
  gap: 16px;
}
.heading h1 {
  margin: 0 0 4px;
  font-size: 20px;
}
.heading span {
  color: var(--gz-muted);
  font-size: 12px;
}
</style>
