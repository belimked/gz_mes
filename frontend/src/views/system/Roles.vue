<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { message } from 'ant-design-vue'
import { call, client, errorMessage, type Schemas } from '@/api/client'
import { useAuthStore } from '@/stores/auth'
import PageCard from '@/components/PageCard.vue'
import TableToolbar from '@/components/TableToolbar.vue'

type Role = Schemas['Role']
const auth = useAuthStore()
const rows = ref<Role[]>([])
const menus = ref<Schemas['MenuItem'][]>([])
const tree = computed(() => menus.value.map(function node(m): { key: number; title: string; children?: ReturnType<typeof node>[] } {
  return { key: m.id, title: m.title, children: m.children?.map(node) }
}))
const columns = [
  { title: '编码', dataIndex: 'code' }, { title: '名称', dataIndex: 'name' },
  { title: '菜单数', dataIndex: 'menuIds' }, { title: '用户数', dataIndex: 'userCount' },
  { title: '状态', dataIndex: 'isStop' }, { title: '备注', dataIndex: 'remark' }, { title: '操作', key: 'action' },
]
const editing = ref<Role>()
const visible = ref(false)
const saving = ref(false)
const form = reactive<Schemas['RoleInput']>({ code: '', name: '', remark: '', isStop: false, menuIds: [] })
async function load() {
  try { rows.value = await call(client.GET('/roles')) } catch (e) { message.error(errorMessage(e)) }
}
function open(role?: Role) {
  editing.value = role
  Object.assign(form, { code: role?.code ?? '', name: role?.name ?? '', remark: role?.remark ?? '', isStop: role?.isStop ?? false, menuIds: [...(role?.menuIds ?? [])] })
  visible.value = true
}
function check(keys: number[] | { checked: number[] }) { form.menuIds = Array.isArray(keys) ? keys : keys.checked }
async function save() {
  if (!form.code.trim() || !form.name.trim()) { message.warning('角色编码或名称无效'); return }
  saving.value = true
  try {
    const body = { ...form, code: form.code.trim(), name: form.name.trim() }
    if (editing.value) await call(client.PUT('/roles/{id}', { params: { path: { id: editing.value.id } }, body }))
    else await call(client.POST('/roles', { body }))
    visible.value = false
    message.success('保存成功')
    await load()
  } catch (e) { message.error(errorMessage(e)) } finally { saving.value = false }
}
async function toggle(role: Role) {
  try {
    await call(client.PUT('/roles/{id}', { params: { path: { id: role.id } }, body: { code: role.code, name: role.name, remark: role.remark, isStop: !role.isStop, menuIds: role.menuIds } }))
    await load()
  } catch (e) { message.error(errorMessage(e)) }
}
async function remove(role: Role) {
  try { await call(client.DELETE('/roles/{id}', { params: { path: { id: role.id } } })); message.success('已删除'); await load() }
  catch (e) { message.error(errorMessage(e)) }
}
onMounted(() => { if (auth.user?.isAdmin) { void load(); void call(client.GET('/menus')).then((m) => { menus.value = m }).catch((e: unknown) => message.error(errorMessage(e))) } })
</script>
<template>
  <a-result v-if="!auth.user?.isAdmin" status="403" title="403" sub-title="仅管理员可访问" />
  <div v-else class="roles-page">
    <div class="heading"><h1>角色管理</h1><span>分配角色菜单；停用角色后其菜单权限不再生效</span></div>
    <PageCard>
      <TableToolbar><template #left>共 {{ rows.length }} 条</template><template #right><a-button type="primary" data-testid="role-new" @click="open()">新增角色</a-button></template></TableToolbar>
      <a-table row-key="id" :data-source="rows" :columns="columns" data-testid="role-table">
        <template #bodyCell="{ column, record }">
          <template v-if="column.dataIndex === 'menuIds'">{{ record.menuIds.length }}</template>
          <template v-else-if="column.dataIndex === 'isStop'"><a-tag :color="record.isStop ? 'default' : 'green'">{{ record.isStop ? '停用' : '启用' }}</a-tag></template>
          <template v-else-if="column.key === 'action'"><a-space><a @click="open(record as Role)">编辑</a><a-popconfirm :title="`确定${record.isStop ? '启用' : '停用'}此角色？`" @confirm="toggle(record as Role)"><a>{{ record.isStop ? '启用' : '停用' }}</a></a-popconfirm><a-popconfirm title="确定删除此角色？" @confirm="remove(record as Role)"><a>删除</a></a-popconfirm></a-space></template>
        </template>
      </a-table>
    </PageCard>
    <a-modal v-model:open="visible" :title="editing ? '编辑角色' : '新增角色'" :confirm-loading="saving" @ok="save">
      <a-form layout="vertical">
        <a-form-item label="编码" required><a-input v-model:value="form.code" :maxlength="50" data-testid="role-code" /></a-form-item>
        <a-form-item label="名称" required><a-input v-model:value="form.name" :maxlength="50" data-testid="role-name" /></a-form-item>
        <a-form-item label="备注"><a-textarea v-model:value="form.remark" /></a-form-item>
        <a-form-item label="停用"><a-switch v-model:checked="form.isStop" /></a-form-item>
        <a-form-item label="菜单"><a-tree :tree-data="tree" checkable :checked-keys="form.menuIds" @check="check" /></a-form-item>
      </a-form>
    </a-modal>
  </div>
</template>
<style scoped>
.roles-page { display: grid; gap: 16px; }
.heading h1 { margin: 0 0 4px; font-size: 20px; }
.heading span { color: var(--gz-muted); font-size: 12px; }
</style>
