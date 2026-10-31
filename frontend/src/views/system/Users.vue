<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { message } from 'ant-design-vue'
import type { FormInstance, TablePaginationConfig } from 'ant-design-vue'
import dayjs from 'dayjs'
import { call, client, errorMessage, type Schemas } from '@/api/client'
import { useAuthStore } from '@/stores/auth'
import PageCard from '@/components/PageCard.vue'
import FilterCard from '@/components/FilterCard.vue'
import TableToolbar from '@/components/TableToolbar.vue'
import CodeText from '@/components/CodeText.vue'
import StatusTag from '@/components/StatusTag.vue'

type User = Schemas['UserDetail']
type Menu = Schemas['MenuItem'] & { parentId?: number | null }
const auth = useAuthStore()
const filters = reactive({ keyword: '', enabled: undefined as boolean | undefined })
const applied = reactive({ ...filters })
const rows = ref<User[]>([])
const factories = ref<Schemas['Factory'][]>([])
const factoryNames = computed(() => new Map(factories.value.map((f) => [f.id, f.name])))
const menus = ref<Menu[]>([])
const menuTree = computed(() => {
  type TreeNode = { key: number; title: string; children: TreeNode[] }
  const nodes = new Map<number, TreeNode>()
  const add = (item: Menu) => {
    if (nodes.has(item.id)) return
    nodes.set(item.id, { key: item.id, title: item.title, children: [] })
    item.children?.forEach((child) => add(child))
  }
  menus.value.forEach(add)
  const roots: TreeNode[] = []
  const attach = (item: Menu, parentId?: number | null) => {
    const node = nodes.get(item.id)!
    const parent = parentId == null ? undefined : nodes.get(parentId)
    if (parent) parent.children.push(node)
    else roots.push(node)
    item.children?.forEach((child) => attach(child, item.id))
  }
  menus.value.forEach((item) => attach(item, item.parentId))
  return roots
})
const pager = reactive({ current: 1, pageSize: 20, total: 0 })
const loading = ref(false)
const columns = [
  { title: '用户名', dataIndex: 'username', width: 130 },
  { title: '姓名', dataIndex: 'displayName', width: 120 },
  { title: '角色', dataIndex: 'isAdmin', width: 110 },
  { title: '厂区', dataIndex: 'factoryIds', width: 220 },
  { title: '菜单数', dataIndex: 'menuIds', width: 90 },
  { title: '状态', dataIndex: 'enabled', width: 100 },
  { title: '创建时间', dataIndex: 'createdAt', width: 170 },
  { title: '操作', key: 'action', width: 220, fixed: 'right' as const },
]
async function load() {
  if (!auth.user?.isAdmin) return
  loading.value = true
  try {
    const result = await call(
      client.GET('/users', {
        params: {
          query: { page: pager.current, size: pager.pageSize, keyword: applied.keyword.trim() || undefined, enabled: applied.enabled },
        },
      }),
    )
    rows.value = result.items
    pager.total = result.total
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
  Object.assign(filters, { keyword: '', enabled: undefined })
  search()
}
function onTableChange(p: TablePaginationConfig) {
  pager.current = p.current ?? 1
  pager.pageSize = p.pageSize ?? 20
  void load()
}
const open = ref(false)
const editing = ref<User>()
const saving = ref(false)
const formRef = ref<FormInstance>()
const form = ref({
  username: '',
  displayName: '',
  password: '',
  enabled: true,
  isAdmin: false,
  factoryIds: [] as number[],
  menuIds: [] as number[],
})
function openCreate() {
  editing.value = undefined
  form.value = { username: '', displayName: '', password: '', enabled: true, isAdmin: false, factoryIds: [], menuIds: [] }
  open.value = true
}
async function openEdit(row: User) {
  try {
    const detail = await call(client.GET('/users/{id}', { params: { path: { id: row.id } } }))
    editing.value = detail
    form.value = {
      username: detail.username,
      displayName: detail.displayName,
      password: '',
      enabled: detail.enabled,
      isAdmin: detail.isAdmin,
      factoryIds: [...detail.factoryIds],
      menuIds: [...detail.menuIds],
    }
    open.value = true
  } catch (e) {
    message.error(errorMessage(e))
  }
}
function onMenuCheck(keys: number[] | { checked: number[] }) {
  form.value.menuIds = Array.isArray(keys) ? keys : keys.checked
}
async function save() {
  try {
    await formRef.value?.validate()
  } catch {
    return
  }
  if (!form.value.isAdmin && !form.value.factoryIds.length) {
    message.warning('普通用户至少需要一个厂区')
    return
  }
  if (editing.value?.id === auth.user?.id && (!form.value.enabled || !form.value.isAdmin)) {
    message.warning('不能停用自己或取消自己的管理员权限')
    return
  }
  saving.value = true
  try {
    const body = {
      displayName: form.value.displayName.trim(),
      isAdmin: form.value.isAdmin,
      factoryIds: form.value.factoryIds,
      menuIds: form.value.menuIds,
    }
    if (editing.value) {
      await call(client.PUT('/users/{id}', { params: { path: { id: editing.value.id } }, body: { ...body, enabled: form.value.enabled } }))
    } else {
      await call(client.POST('/users', { body: { ...body, username: form.value.username.trim(), password: form.value.password } }))
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
async function toggle(row: User) {
  if (row.id === auth.user?.id && row.enabled) return
  try {
    const detail = await call(client.GET('/users/{id}', { params: { path: { id: row.id } } }))
    await call(
      client.PUT('/users/{id}', {
        params: { path: { id: row.id } },
        body: {
          displayName: detail.displayName,
          enabled: !detail.enabled,
          isAdmin: detail.isAdmin,
          factoryIds: detail.factoryIds,
          menuIds: detail.menuIds,
        },
      }),
    )
    message.success(row.enabled ? '已停用' : '已启用')
    await load()
  } catch (e) {
    message.error(errorMessage(e))
  }
}
const passwordUser = ref<User>()
const password = reactive({ value: '', confirm: '' })
const passwordRef = ref<FormInstance>()
const resetting = ref(false)
function validateConfirmation(_rule: unknown, value: string) {
  return value === password.value ? Promise.resolve() : Promise.reject(new Error('两次输入的密码不一致'))
}
function openPassword(row: User) {
  password.value = ''
  password.confirm = ''
  passwordUser.value = row
}
function closePassword() {
  passwordUser.value = undefined
}
async function resetPassword() {
  try {
    await passwordRef.value?.validate()
  } catch {
    return
  }
  if (!passwordUser.value) return
  resetting.value = true
  try {
    await call(client.PUT('/users/{id}/password', { params: { path: { id: passwordUser.value.id } }, body: { password: password.value } }))
    message.success('密码已重置')
    closePassword()
  } catch (e) {
    message.error(errorMessage(e))
  } finally {
    resetting.value = false
  }
}
onMounted(async () => {
  if (!auth.user?.isAdmin) return
  void load()
  try {
    const [fs, ms] = await Promise.all([call(client.GET('/factories')), call(client.GET('/menus'))])
    factories.value = fs
    menus.value = ms
  } catch (e) {
    message.error(errorMessage(e))
  }
})
</script>

<template>
  <a-result v-if="!auth.user?.isAdmin" status="403" title="403" sub-title="仅管理员可访问" />
  <div v-else class="system-page">
    <div class="heading">
      <div>
        <h1>用户管理</h1>
        <span>配置用户权限及厂区访问范围</span>
      </div>
      <a-tag>共 {{ pager.total }} 个用户</a-tag>
    </div>
    <FilterCard @search="search" @reset="reset">
      <label>关键字<a-input v-model:value="filters.keyword" allow-clear placeholder="用户名 / 姓名" @press-enter="search" /></label>
      <label
        >状态<a-select
          v-model:value="filters.enabled"
          allow-clear
          placeholder="全部状态"
          :options="[
            { value: true, label: '启用' },
            { value: false, label: '停用' },
          ]"
      /></label>
    </FilterCard>
    <PageCard>
      <TableToolbar
        ><template #left><a-button type="primary" data-testid="user-new" @click="openCreate">新建用户</a-button></template
        ><template #right><a-button @click="load">刷新</a-button></template></TableToolbar
      >
      <a-table
        row-key="id"
        size="middle"
        :columns="columns"
        :data-source="rows"
        :loading="loading"
        :scroll="{ x: 1200 }"
        :pagination="{ ...pager, showSizeChanger: true, showTotal: (n: number) => `共 ${n} 条` }"
        data-testid="user-table"
        @change="onTableChange"
      >
        <template #bodyCell="{ column, record }">
          <template v-if="column.dataIndex === 'username'"><CodeText :value="record.username" /></template>
          <template v-else-if="column.dataIndex === 'isAdmin'"
            ><StatusTag :tone="record.isAdmin ? 'processing' : 'neutral'" :label="record.isAdmin ? '管理员' : '普通'"
          /></template>
          <template v-else-if="column.dataIndex === 'factoryIds'"
            ><a-tag v-for="id in record.factoryIds" :key="id">{{ factoryNames.get(id) ?? id }}</a-tag
            ><span v-if="!record.factoryIds.length">—</span></template
          >
          <template v-else-if="column.dataIndex === 'menuIds'">{{ record.menuIds.length }}</template>
          <template v-else-if="column.dataIndex === 'enabled'"
            ><StatusTag :tone="record.enabled ? 'success' : 'neutral'" :label="record.enabled ? '启用' : '停用'"
          /></template>
          <template v-else-if="column.dataIndex === 'createdAt'">{{ dayjs(record.createdAt).format('YYYY-MM-DD HH:mm') }}</template>
          <template v-else-if="column.key === 'action'"
            ><a-space
              ><a @click="openEdit(record as User)">编辑</a><a @click="openPassword(record as User)">重置密码</a
              ><a-popconfirm v-if="record.enabled && record.id === auth.user?.id" title="不能停用自己" :disabled="true"
                ><a class="disabled">停用</a></a-popconfirm
              ><a-popconfirm v-else :title="`确定${record.enabled ? '停用' : '启用'}此用户？`" @confirm="toggle(record as User)"
                ><a>{{ record.enabled ? '停用' : '启用' }}</a></a-popconfirm
              ></a-space
            ></template
          >
        </template>
      </a-table>
    </PageCard>
    <a-modal
      v-model:open="open"
      :title="editing ? '编辑用户' : '新建用户'"
      ok-text="保存"
      :confirm-loading="saving"
      destroy-on-close
      @ok="save"
    >
      <a-form ref="formRef" :model="form" :label-col="{ span: 5 }">
        <a-form-item
          v-if="!editing"
          label="用户名"
          name="username"
          :rules="[{ required: true, pattern: /^[a-zA-Z0-9_.-]{2,32}$/, message: '请输入2-32位用户名（字母、数字、_.-）' }]"
          ><a-input v-model:value="form.username" :maxlength="32"
        /></a-form-item>
        <a-form-item label="姓名" name="displayName" :rules="[{ required: true, whitespace: true, message: '请输入姓名' }]"
          ><a-input v-model:value="form.displayName" :maxlength="50"
        /></a-form-item>
        <a-form-item
          v-if="!editing"
          label="初始密码"
          name="password"
          :rules="[{ required: true, min: 6, max: 72, message: '请输入6-72位密码' }]"
          ><a-input-password v-model:value="form.password"
        /></a-form-item>
        <a-form-item label="管理员"><a-switch v-model:checked="form.isAdmin" :disabled="editing?.id === auth.user?.id" /></a-form-item>
        <a-form-item v-if="editing" label="启用"
          ><a-switch v-model:checked="form.enabled" :disabled="editing.id === auth.user?.id"
        /></a-form-item>
        <a-form-item label="厂区"
          ><a-select
            v-model:value="form.factoryIds"
            mode="multiple"
            placeholder="选择厂区"
            :options="factories.map((f) => ({ value: f.id, label: f.name }))"
        /></a-form-item>
        <a-form-item label="菜单"><a-tree :tree-data="menuTree" checkable :checked-keys="form.menuIds" @check="onMenuCheck" /></a-form-item>
      </a-form>
    </a-modal>
    <a-modal
      :open="!!passwordUser"
      :title="`重置密码：${passwordUser?.username ?? ''}`"
      ok-text="确认"
      :confirm-loading="resetting"
      destroy-on-close
      @cancel="closePassword"
      @ok="resetPassword"
    >
      <a-form ref="passwordRef" :model="password" :label-col="{ span: 5 }">
        <a-form-item label="新密码" name="value" :rules="[{ required: true, min: 6, max: 72, message: '请输入6-72位密码' }]"
          ><a-input-password v-model:value="password.value"
        /></a-form-item>
        <a-form-item label="确认密码" name="confirm" :rules="[{ required: true, validator: validateConfirmation, trigger: 'change' }]"
          ><a-input-password v-model:value="password.confirm"
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
.disabled {
  color: var(--gz-muted);
  cursor: not-allowed;
  pointer-events: none;
}
</style>
