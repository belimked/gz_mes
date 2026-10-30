<script setup lang="ts">
import { reactive, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { message } from 'ant-design-vue'
import { errorMessage, isDemo } from '@/api/client'
import { DEMO_ACCOUNTS } from '@/api/demo/data'
import { useAuthStore } from '@/stores/auth'
import { brand } from '@/theme'

const auth = useAuthStore()
const route = useRoute()
const router = useRouter()

const form = reactive({ username: '', password: '' })
const loading = ref(false)

async function submit() {
  loading.value = true
  try {
    await auth.login(form.username.trim(), form.password)
    const r = route.query.redirect
    const redirect = typeof r === 'string' && r.startsWith('/') && !r.startsWith('//') ? r : '/'
    await router.replace(redirect)
  } catch (e) {
    message.error(errorMessage(e))
  } finally {
    loading.value = false
  }
}

function fill(username: string, password: string) {
  form.username = username
  form.password = password
}
</script>

<template>
  <div class="page">
    <a-card class="card" :bordered="false">
      <h1>{{ brand.name }}</h1>
      <p class="sub">{{ brand.subtitle }}</p>
      <a-form :model="form" layout="vertical" @finish="submit">
        <a-form-item label="账号" name="username" :rules="[{ required: true, message: '请输入账号' }]">
          <a-input v-model:value="form.username" autocomplete="username" data-testid="username" />
        </a-form-item>
        <a-form-item label="密码" name="password" :rules="[{ required: true, message: '请输入密码' }]">
          <a-input-password v-model:value="form.password" autocomplete="current-password" data-testid="password" />
        </a-form-item>
        <a-button type="primary" html-type="submit" block :loading="loading" data-testid="login-submit">登 录</a-button>
      </a-form>
      <a-alert v-if="isDemo" class="demo" type="info" show-icon>
        <template #message>演示环境：数据为虚构，只保存在当前浏览器</template>
        <template #description>
          <div v-for="a in DEMO_ACCOUNTS" :key="a.username">
            <a @click="fill(a.username, a.password)">{{ a.username }} / {{ a.password }}</a> — {{ a.note }}
          </div>
        </template>
      </a-alert>
    </a-card>
  </div>
</template>

<style scoped>
.page {
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 100vh;
  background: linear-gradient(135deg, #e6f0ff, #f5f7fa);
}
.card {
  width: 380px;
  box-shadow: 0 8px 24px rgb(0 0 0 / 8%);
}
h1 {
  margin: 0;
  font-size: 24px;
}
.sub {
  margin: 4px 0 24px;
  color: #8c8c8c;
}
.demo {
  margin-top: 16px;
}
</style>
