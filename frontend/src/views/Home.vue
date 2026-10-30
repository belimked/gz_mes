<script setup lang="ts">
import { message } from 'ant-design-vue'
import { isDemo } from '@/api/client'
import { resetDemo } from '@/api/demo'
import { useAuthStore } from '@/stores/auth'

const auth = useAuthStore()

function reset() {
  resetDemo()
  message.success('演示数据已重置')
}
</script>

<template>
  <a-result status="info" :title="`欢迎，${auth.user?.displayName ?? ''}`" sub-title="从左侧菜单进入各功能模块。">
    <template #extra>
      <router-link v-if="auth.allowedPaths.has('/sale/projects')" to="/sale/projects">
        <a-button type="primary">项目管理</a-button>
      </router-link>
      <a-button v-if="isDemo" @click="reset">重置演示数据</a-button>
    </template>
  </a-result>
</template>
