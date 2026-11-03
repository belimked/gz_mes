<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { message } from 'ant-design-vue'
import { call, client, errorMessage, type Schemas } from '@/api/client'
import PageCard from '@/components/PageCard.vue'
import TableToolbar from '@/components/TableToolbar.vue'
import CodeText from '@/components/CodeText.vue'

type Machine = Schemas['Machine']
const factories = ref<Schemas['Factory'][]>([])
const jobs = ref<Schemas['Job'][]>([])
const machines = ref<Machine[]>([])
const filters = reactive({ factoryId: undefined as number | undefined, jobId: undefined as number | undefined, includeStopped: false })
const factoryOptions = computed(() => factories.value.filter((f) => f.kind === 'own').map((f) => ({ value: f.id, label: f.name })))
const jobOptions = computed(() => jobs.value.map((j) => ({ value: j.id, label: `${j.code} ${j.name}` })))
const editing = ref<Machine>()
const visible = ref(false)
const saving = ref(false)
function blank(): Schemas['MachineInput'] {
  return {
    factoryId: factoryOptions.value[0]?.value ?? 0,
    jobId: jobs.value[0]?.id ?? 0,
    code: '',
    name: '',
    capacityQty: 0,
    capacityArea: 0,
    isStop: false,
    remark: '',
  }
}
const form = reactive<Schemas['MachineInput']>(blank())
const columns = [
  { title: '厂区', dataIndex: 'factoryName' },
  { title: '工序', dataIndex: 'jobName' },
  { title: '编码', dataIndex: 'code' },
  { title: '名称', dataIndex: 'name' },
  { title: '日产能件数', dataIndex: 'capacityQty' },
  { title: '日产能面积 m²', dataIndex: 'capacityArea' },
  { title: '状态', dataIndex: 'isStop' },
  { title: '备注', dataIndex: 'remark' },
  { title: '操作', dataIndex: 'action' },
]
async function load() {
  try {
    machines.value = await call(client.GET('/machines', { params: { query: filters } }))
  } catch (e) {
    message.error(errorMessage(e))
  }
}
function open(m?: Machine) {
  editing.value = m
  Object.assign(form, m ?? blank())
  visible.value = true
}
async function save() {
  saving.value = true
  try {
    if (editing.value) await call(client.PUT('/machines/{id}', { params: { path: { id: editing.value.id } }, body: { ...form } }))
    else await call(client.POST('/machines', { body: { ...form } }))
    visible.value = false
    message.success('保存成功')
    await load()
  } catch (e) {
    message.error(errorMessage(e))
  } finally {
    saving.value = false
  }
}
async function toggle(m: Machine, checked: boolean) {
  try {
    await call(
      client.PUT('/machines/{id}', {
        params: { path: { id: m.id } },
        body: {
          factoryId: m.factoryId,
          jobId: m.jobId,
          code: m.code,
          name: m.name,
          capacityQty: m.capacityQty,
          capacityArea: m.capacityArea,
          isStop: !checked,
          remark: m.remark,
        },
      }),
    )
    await load()
  } catch (e) {
    message.error(errorMessage(e))
  }
}
onMounted(async () => {
  try {
    ;[factories.value, jobs.value] = await Promise.all([call(client.GET('/factories')), call(client.GET('/jobs'))])
    await load()
  } catch (e) {
    message.error(errorMessage(e))
  }
})
</script>
<template>
  <div class="report">
    <div class="heading">
      <h1>机台管理</h1>
      <span>维护自有厂区的机台、所属工序及日产能；已有扫码的机台可停用但不能删除</span>
    </div>
    <PageCard
      ><div class="filters">
        <label
          >厂区<a-select v-model:value="filters.factoryId" allow-clear placeholder="全部" :options="factoryOptions" @change="load" /></label
        ><label>工序<a-select v-model:value="filters.jobId" allow-clear placeholder="全部" :options="jobOptions" @change="load" /></label
        ><label>显示停用<a-switch v-model:checked="filters.includeStopped" @change="load" /></label>
        <div class="actions"><a-button type="primary" @click="load">查询</a-button></div>
      </div></PageCard
    >
    <PageCard
      ><TableToolbar
        ><template #left>共 {{ machines.length }} 台机台</template
        ><template #right
          ><a-button type="primary" :disabled="!factoryOptions.length || !jobs.length" data-testid="machine-new" @click="open()"
            >新建机台</a-button
          ></template
        ></TableToolbar
      ><a-table
        row-key="id"
        bordered
        size="middle"
        :data-source="machines"
        :columns="columns"
        :pagination="{ showTotal: (n: number) => `共 ${n} 条` }"
        data-testid="machine-table"
        ><template #emptyText><a-empty description="暂无机台" /></template
        ><template #bodyCell="{ column, record }"
          ><template v-if="column.dataIndex === 'code'"><CodeText :value="record.code" /></template
          ><template v-else-if="column.dataIndex === 'isStop'"
            ><a-tag :color="record.isStop ? 'default' : 'green'">{{ record.isStop ? '停用' : '启用' }}</a-tag></template
          ><template v-else-if="column.dataIndex === 'action'"
            ><a-space
              ><a @click="open(record)">编辑</a
              ><a-switch
                :checked="!record.isStop"
                checked-children="启用"
                un-checked-children="停用"
                @change="(checked: boolean) => toggle(record, checked)" /></a-space></template></template></a-table
    ></PageCard>
    <a-modal v-model:open="visible" :title="editing ? '编辑机台' : '新建机台'" :confirm-loading="saving" @ok="save"
      ><a-form layout="vertical"
        ><a-form-item label="厂区" required
          ><a-select v-model:value="form.factoryId" :options="factoryOptions" data-testid="machine-factory" /></a-form-item
        ><a-form-item label="工序" required
          ><a-select v-model:value="form.jobId" :options="jobOptions" data-testid="machine-job" /></a-form-item
        ><a-form-item label="编码" required><a-input v-model:value="form.code" data-testid="machine-code" /></a-form-item
        ><a-form-item label="名称" required><a-input v-model:value="form.name" data-testid="machine-name" /></a-form-item
        ><a-form-item label="日产能件数"><a-input-number v-model:value="form.capacityQty" :min="0" style="width: 100%" /></a-form-item
        ><a-form-item label="日产能面积 m²"
          ><a-input-number v-model:value="form.capacityArea" :min="0" :precision="2" style="width: 100%" /></a-form-item
        ><a-form-item label="备注"><a-textarea v-model:value="form.remark" /></a-form-item></a-form
    ></a-modal>
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
.filters {
  display: flex;
  flex-wrap: wrap;
  align-items: end;
  gap: 12px;
}
.filters label {
  display: grid;
  gap: 6px;
  color: var(--gz-secondary);
  font-size: 12px;
  min-width: 150px;
}
.actions {
  display: flex;
  gap: 8px;
  margin-left: auto;
}
</style>
