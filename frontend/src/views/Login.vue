<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { LockOutlined, LoginOutlined, UserOutlined } from '@ant-design/icons-vue'
import { errorMessage, isDemo } from '@/api/client'
import { DEMO_ACCOUNTS } from '@/api/demo/data'
import { useAuthStore } from '@/stores/auth'
import { brand } from '@/theme'

const LAST_USER_KEY = 'gz-mes-last-user'

const auth = useAuthStore()
const route = useRoute()
const router = useRouter()

const form = reactive({ username: '', password: '', remember: true })
const loading = ref(false)
const error = ref('')
const capsLock = ref(false)
const passwordRef = ref<HTMLInputElement>()

const flow = ['合同', '批次', '面积表', '标签', '工序扫描', '发货', '对账']
const features = [
  { title: '一件一码', desc: '每块板材独立标签，工序扫描全程可追溯' },
  { title: '厂区隔离', desc: '本厂与外协厂数据分开，按账号授权可见' },
  { title: '业财一体', desc: '面积自动核算，发货即对账' },
]

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
  <div class="page">
    <!-- Brand panel -->
    <section class="hero">
      <div class="hero-top">
        <span class="logo">M</span>
        <span class="hero-name">{{ brand.name }}</span>
      </div>

      <div class="hero-body">
        <h2>{{ brand.subtitle }}</h2>
        <p class="slogan">从合同到发货，每一块板都看得见</p>

        <div class="panels" aria-hidden="true">
          <span v-for="i in 12" :key="i" class="panel" :style="{ animationDelay: `${(i % 6) * 0.35}s` }" />
        </div>

        <div class="flow">
          <template v-for="(s, i) in flow" :key="s">
            <span class="step">{{ s }}</span>
            <span v-if="i < flow.length - 1" class="arrow">›</span>
          </template>
        </div>

        <ul class="features">
          <li v-for="f in features" :key="f.title">
            <b>{{ f.title }}</b>
            <span>{{ f.desc }}</span>
          </li>
        </ul>
      </div>

      <div class="hero-foot">© {{ new Date().getFullYear() }} {{ brand.name }}</div>
    </section>

    <!-- Form -->
    <section class="side">
      <div class="box">
        <div class="mobile-brand">
          <span class="logo">M</span>
          <span>{{ brand.name }}</span>
        </div>
        <h1>欢迎登录</h1>
        <p class="sub">{{ brand.subtitle }}</p>

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
    </section>
  </div>
</template>

<style scoped>
.page {
  display: flex;
  min-height: 100vh;
  background: #f5f7fa;
}

/* ---------- brand panel ---------- */
.hero {
  position: relative;
  display: flex;
  flex: 1.2;
  flex-direction: column;
  padding: 40px 56px;
  overflow: hidden;
  color: #fff;
  background:
    radial-gradient(circle at 85% 15%, rgb(64 150 255 / 35%), transparent 45%),
    radial-gradient(circle at 10% 90%, rgb(19 194 194 / 25%), transparent 40%),
    linear-gradient(150deg, #0b1f4b 0%, #0e3a8a 55%, #1677ff 100%);
}
.hero-top {
  display: flex;
  align-items: center;
  gap: 12px;
}
.hero-name {
  font-size: 20px;
  font-weight: 600;
  letter-spacing: 1px;
}
.logo {
  display: inline-flex;
  flex: none;
  align-items: center;
  justify-content: center;
  width: 36px;
  height: 36px;
  border-radius: 8px;
  background: #fff;
  color: #1677ff;
  font-size: 20px;
  font-weight: 800;
}
.hero-body {
  display: flex;
  flex: 1;
  flex-direction: column;
  justify-content: center;
  max-width: 560px;
}
.hero-body h2 {
  margin: 0;
  color: #fff;
  font-size: 36px;
  font-weight: 600;
  letter-spacing: 2px;
}
.slogan {
  margin: 12px 0 36px;
  color: rgb(255 255 255 / 75%);
  font-size: 16px;
}

/* abstract curtain-wall panel grid */
.panels {
  display: grid;
  grid-template-columns: repeat(6, 1fr);
  gap: 8px;
  width: 100%;
  max-width: 420px;
  margin-bottom: 32px;
  transform: perspective(600px) rotateX(18deg) rotateZ(-4deg);
}
.panel {
  aspect-ratio: 3 / 4;
  border: 1px solid rgb(255 255 255 / 35%);
  border-radius: 3px;
  background: linear-gradient(160deg, rgb(255 255 255 / 22%), rgb(255 255 255 / 4%));
  animation: shine 4.2s ease-in-out infinite;
}
@keyframes shine {
  0%,
  100% {
    background-color: transparent;
  }
  50% {
    background-color: rgb(255 255 255 / 18%);
  }
}

.flow {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 6px;
  margin-bottom: 32px;
}
.step {
  padding: 3px 10px;
  border: 1px solid rgb(255 255 255 / 30%);
  border-radius: 12px;
  background: rgb(255 255 255 / 8%);
  font-size: 13px;
}
.arrow {
  color: rgb(255 255 255 / 50%);
}
.features {
  display: grid;
  gap: 14px;
  margin: 0;
  padding: 0;
  list-style: none;
}
.features li {
  display: flex;
  gap: 12px;
  align-items: baseline;
}
.features b {
  flex: none;
  min-width: 64px;
  font-weight: 600;
}
.features span {
  color: rgb(255 255 255 / 70%);
}
.hero-foot {
  color: rgb(255 255 255 / 50%);
  font-size: 12px;
}

/* ---------- form side ---------- */
.side {
  display: flex;
  flex: 1;
  align-items: center;
  justify-content: center;
  padding: 40px 24px;
}
.box {
  width: 100%;
  max-width: 380px;
}
.mobile-brand {
  display: none;
  align-items: center;
  gap: 10px;
  margin-bottom: 24px;
  font-size: 18px;
  font-weight: 600;
}
.mobile-brand .logo {
  background: #1677ff;
  color: #fff;
}
h1 {
  margin: 0;
  font-size: 28px;
  font-weight: 600;
}
.sub {
  margin: 6px 0 28px;
  color: #8c8c8c;
}
.err {
  margin-bottom: 16px;
}
.ico {
  color: #bfbfbf;
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
  color: #8c8c8c;
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
  border: 1px solid #e5e8ef;
  border-radius: 8px;
  background: #fff;
  cursor: pointer;
  text-align: left;
  transition:
    border-color 0.2s,
    box-shadow 0.2s;
}
.account:hover:not(:disabled) {
  border-color: #1677ff;
  box-shadow: 0 2px 8px rgb(22 119 255 / 15%);
}
.account:disabled {
  cursor: not-allowed;
  opacity: 0.6;
}
.account b {
  color: #1f1f1f;
}
.account span {
  color: #8c8c8c;
  font-size: 12px;
}
.hint {
  margin: 12px 0 0;
  color: #bfbfbf;
  font-size: 12px;
  text-align: center;
}

@media (width <= 900px) {
  .hero {
    display: none;
  }
  .mobile-brand {
    display: flex;
  }
}

@media (prefers-reduced-motion: reduce) {
  .panel {
    animation: none;
  }
}
</style>
