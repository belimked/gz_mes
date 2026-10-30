<script setup lang="ts">
// Shared login form used by every login page style. Styles only provide the surrounding layout.
import { onMounted, reactive, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { LockOutlined, LoginOutlined, UserOutlined } from '@ant-design/icons-vue'
import { errorMessage, isDemo } from '@/api/client'
import { DEMO_ACCOUNTS } from '@/api/demo/data'
import { useAuthStore } from '@/stores/auth'

const props = withDefaults(defineProps<{ dark?: boolean; accent?: string }>(), { dark: false, accent: '#1677ff' })

const LAST_USER_KEY = 'gz-mes-last-user'

const auth = useAuthStore()
const route = useRoute()
const router = useRouter()

const form = reactive({ username: '', password: '', remember: true })
const loading = ref(false)
const error = ref('')
const capsLock = ref(false)
const passwordRef = ref<HTMLInputElement>()

onMounted(() => {
  const last = localStorage.getItem(LAST_USER_KEY)
  if (last) {
    form.username = last
    passwordRef.value?.focus()
  }
})

function onKey(e: KeyboardEvent) {
  capsLock.value = e.getModifierState?.('CapsLock') ?? false
}

async function submit() {
  error.value = ''
  loading.value = true
  try {
    const username = form.username.trim()
    await auth.login(username, form.password)
    if (form.remember) localStorage.setItem(LAST_USER_KEY, username)
    else localStorage.removeItem(LAST_USER_KEY)
    const r = route.query.redirect
    const redirect = typeof r === 'string' && r.startsWith('/') && !r.startsWith('//') ? r : '/'
    await router.replace(redirect)
  } catch (e) {
    error.value = errorMessage(e)
  } finally {
    loading.value = false
  }
}

function quickLogin(username: string, password: string) {
  form.username = username
  form.password = password
  void submit()
}
</script>

<template>
  <div class="login-form" :class="{ dark: props.dark }" :style="{ '--accent': props.accent }">
    <a-alert v-if="error" class="err" type="error" :message="error" show-icon closable @close="error = ''" />

    <a-form :model="form" layout="vertical" size="large" :required-mark="false" @finish="submit">
      <a-form-item name="username" :rules="[{ required: true, whitespace: true, message: '请输入账号' }]">
        <a-input v-model:value="form.username" placeholder="账号" autocomplete="username" data-testid="username">
          <template #prefix><UserOutlined class="ico" /></template>
        </a-input>
      </a-form-item>
      <a-form-item name="password" :rules="[{ required: true, message: '请输入密码' }]">
        <a-input-password
          ref="passwordRef"
          v-model:value="form.password"
          placeholder="密码"
          autocomplete="current-password"
          data-testid="password"
          @keyup="onKey"
          @keydown="onKey"
        >
          <template #prefix><LockOutlined class="ico" /></template>
        </a-input-password>
        <div v-if="capsLock" class="caps">大写锁定已开启</div>
      </a-form-item>
      <div class="row">
        <a-checkbox v-model:checked="form.remember">记住账号</a-checkbox>
        <a-tooltip title="请联系系统管理员重置密码">
          <a class="muted">忘记密码？</a>
        </a-tooltip>
      </div>
      <a-button type="primary" html-type="submit" block :loading="loading" data-testid="login-submit">
        <template #icon><LoginOutlined /></template>
        登 录
      </a-button>
    </a-form>

    <div v-if="isDemo" class="demo">
      <a-divider plain><span class="muted">演示账号 · 点击直接登录</span></a-divider>
      <div class="accounts">
        <button
          v-for="a in DEMO_ACCOUNTS"
          :key="a.username"
          type="button"
          class="account"
          :disabled="loading"
          :data-testid="`demo-${a.username}`"
          @click="quickLogin(a.username, a.password)"
        >
          <b>{{ a.username }}</b>
          <span>{{ a.note }}</span>
        </button>
      </div>
      <p class="hint">演示环境数据为虚构，只保存在当前浏览器</p>
    </div>
  </div>
</template>

<style scoped>
.login-form {
  --muted: #8c8c8c;
  --hint: #bfbfbf;
  --title: #1f1f1f;
  --card-bg: #fff;
  --card-border: #e5e8ef;
}
.login-form.dark {
  --muted: rgb(255 255 255 / 55%);
  --hint: rgb(255 255 255 / 35%);
  --title: #fff;
  --card-bg: rgb(255 255 255 / 5%);
  --card-border: rgb(255 255 255 / 15%);
}
.err {
  margin-bottom: 16px;
}
.ico {
  color: var(--hint);
}
.caps {
  margin-top: 4px;
  color: #faad14;
  font-size: 12px;
}
.row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin: -8px 0 20px;
}
.muted {
  color: var(--muted);
  font-size: 13px;
}
.demo {
  margin-top: 8px;
}
.accounts {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 10px;
}
.account {
  display: flex;
  flex-direction: column;
  gap: 2px;
  padding: 10px 12px;
  border: 1px solid var(--card-border);
  border-radius: 8px;
  background: var(--card-bg);
  cursor: pointer;
  text-align: left;
  transition:
    border-color 0.2s,
    box-shadow 0.2s;
}
.account:hover:not(:disabled) {
  border-color: var(--accent);
  box-shadow: 0 2px 10px color-mix(in srgb, var(--accent) 25%, transparent);
}
.account:disabled {
  cursor: not-allowed;
  opacity: 0.6;
}
.account b {
  color: var(--title);
}
.account span {
  color: var(--muted);
  font-size: 12px;
}
.hint {
  margin: 12px 0 0;
  color: var(--hint);
  font-size: 12px;
  text-align: center;
}
</style>

<style scoped>
/* Dark themes: the inner <input> must not paint a second background inside the affix wrapper. */
.login-form.dark :deep(.ant-input-affix-wrapper .ant-input) {
  background: transparent;
}
</style>
