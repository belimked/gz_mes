<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { LogoutOutlined, MenuFoldOutlined, MenuUnfoldOutlined, UserOutlined } from '@ant-design/icons-vue'
import MenuIcon from '@/components/MenuIcon.vue'
import { isDemo } from '@/api/client'
import { useAuthStore } from '@/stores/auth'
import { brand, layout } from '@/theme'

const auth = useAuthStore()
const route = useRoute()
const router = useRouter()

const collapsed = ref(false)
const selectedKeys = ref<string[]>([])
const openKeys = ref<string[]>([])

watch(
  () => route.path,
  (path) => {
    selectedKeys.value = [path]
    const parent = auth.menus.find((m) => m.children?.some((c) => c.path === path))
    if (parent && !openKeys.value.includes(parent.path)) openKeys.value = [...openKeys.value, parent.path]
  },
  { immediate: true },
)

const pageTitle = computed(() => route.meta.title ?? '')

function go({ key }: { key: string | number }) {
  void router.push(String(key))
}

function logout() {
  auth.logout()
  void router.replace({ name: 'login' })
}
</script>

<template>
  <a-layout class="layout">
    <a-layout-sider v-model:collapsed="collapsed" :width="layout.siderWidth" :theme="layout.siderTheme" collapsible :trigger="null">
      <router-link to="/" class="brand" :class="layout.siderTheme">
        <span class="logo">M</span>
        <span v-if="!collapsed" class="name">{{ brand.name }}</span>
      </router-link>
      <a-menu
        v-model:selected-keys="selectedKeys"
        v-model:open-keys="openKeys"
        mode="inline"
        :theme="layout.siderTheme"
        data-testid="side-menu"
        @click="go"
      >
        <template v-for="m in auth.menus" :key="m.path">
          <a-sub-menu v-if="m.children?.length" :key="m.path">
            <template #icon><MenuIcon :name="m.icon" /></template>
            <template #title>{{ m.title }}</template>
            <a-menu-item v-for="c in m.children" :key="c.path">{{ c.title }}</a-menu-item>
          </a-sub-menu>
          <a-menu-item v-else :key="m.path">
            <template #icon><MenuIcon :name="m.icon" /></template>
            {{ m.title }}
          </a-menu-item>
        </template>
      </a-menu>
    </a-layout-sider>
    <a-layout>
      <a-layout-header class="header">
        <span class="trigger" @click="collapsed = !collapsed">
          <MenuUnfoldOutlined v-if="collapsed" />
          <MenuFoldOutlined v-else />
        </span>
        <span class="title">{{ pageTitle }}</span>
        <a-tag v-if="isDemo" color="orange">演示环境 · 数据为虚构</a-tag>
        <a-dropdown>
          <span class="user" data-testid="user-menu"><UserOutlined /> {{ auth.user?.displayName }}</span>
          <template #overlay>
            <a-menu>
              <a-menu-item key="logout" data-testid="logout" @click="logout"><LogoutOutlined /> 退出登录</a-menu-item>
            </a-menu>
          </template>
        </a-dropdown>
      </a-layout-header>
      <a-layout-content class="content">
        <router-view />
      </a-layout-content>
    </a-layout>
  </a-layout>
</template>

<style scoped>
.layout {
  min-height: 100vh;
}
.brand {
  display: flex;
  align-items: center;
  gap: 10px;
  height: 56px;
  padding: 0 20px;
  overflow: hidden;
  white-space: nowrap;
}
.brand.dark .name {
  color: #fff;
}
.logo {
  display: inline-flex;
  flex: none;
  align-items: center;
  justify-content: center;
  width: 28px;
  height: 28px;
  border-radius: 6px;
  background: #1677ff;
  color: #fff;
  font-weight: 700;
}
.name {
  font-size: 16px;
  font-weight: 600;
}
.header {
  display: flex;
  align-items: center;
  gap: 16px;
  padding: 0 16px;
  background: #fff;
  border-bottom: 1px solid #f0f0f0;
}
.trigger {
  cursor: pointer;
  font-size: 18px;
}
.title {
  flex: 1;
  font-size: 16px;
  font-weight: 500;
}
.user {
  cursor: pointer;
}
.content {
  margin: 16px;
  padding: 16px;
  background: #fff;
  border-radius: 4px;
}
</style>
