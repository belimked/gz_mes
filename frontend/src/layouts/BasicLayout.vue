<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import {
  LogoutOutlined,
  MenuFoldOutlined,
  MenuUnfoldOutlined,
  BellOutlined,
  QuestionCircleOutlined,
  FullscreenOutlined,
  SearchOutlined,
} from '@ant-design/icons-vue'
import MenuIcon from '@/components/MenuIcon.vue'
import { isDemo } from '@/api/client'
import { factories } from '@/api/demo/data'
import { useAuthStore } from '@/stores/auth'
import { brand, layout } from '@/theme'

const auth = useAuthStore()
const route = useRoute()
const router = useRouter()
const collapsed = ref(false)
const mobile = ref(false)
const mobileOpen = ref(false)
const selectedKeys = ref<string[]>([])
const openKeys = ref<string[]>([])
const factory = ref<number>()
const availableFactories = computed(() =>
  auth.user?.isAdmin ? factories : factories.filter((f) => f.id === (auth.user?.username === 'waixie' ? 2 : 1)),
)
watch(
  availableFactories,
  (f) => {
    factory.value = f[0]?.id
  },
  { immediate: true },
)
const parent = computed(() => auth.menus.find((m) => m.children?.some((c) => c.path === route.path)))
const pageTitle = computed(() => String(route.meta.title ?? '首页'))
watch(
  () => route.path,
  (path) => {
    selectedKeys.value = [path]
    if (parent.value && !openKeys.value.includes(parent.value.path)) openKeys.value = [...openKeys.value, parent.value.path]
    mobileOpen.value = false
  },
  { immediate: true },
)
function resize() {
  mobile.value = window.innerWidth < 768
  collapsed.value = window.innerWidth < 1200
}
onMounted(() => {
  resize()
  window.addEventListener('resize', resize)
})
onUnmounted(() => window.removeEventListener('resize', resize))
function go({ key }: { key: string | number }) {
  void router.push(String(key))
}
function logout() {
  auth.logout()
  void router.replace({ name: 'login' })
}
function toggle() {
  if (mobile.value) mobileOpen.value = !mobileOpen.value
  else collapsed.value = !collapsed.value
}
function fullscreen() {
  if (document.fullscreenElement) void document.exitFullscreen()
  else void document.documentElement.requestFullscreen()
}
</script>

<template>
  <a-layout class="layout">
    <a-layout-sider
      v-if="!mobile"
      v-model:collapsed="collapsed"
      :width="layout.siderWidth"
      :collapsed-width="48"
      :theme="layout.siderTheme"
      :trigger="null"
      class="sider"
    >
      <router-link to="/" class="brand">
        <span class="logo">M</span>
        <span v-if="!collapsed" class="name">{{ brand.name }} <small>MES</small> </span>
      </router-link>
      <a-menu
        v-model:selected-keys="selectedKeys"
        v-model:open-keys="openKeys"
        mode="inline"
        :theme="layout.siderTheme"
        data-testid="side-menu"
        @click="go"
      >
        <a-menu-item key="/">
          <template #icon>
            <MenuIcon name="dashboard" /> </template
          >首页
        </a-menu-item>
        <template v-for="m in auth.menus" :key="m.path">
          <a-sub-menu v-if="m.children?.length" :key="m.path">
            <template #icon>
              <MenuIcon :name="m.icon" />
            </template>
            <template #title>{{ m.title }}</template>
            <a-menu-item v-for="c in m.children" :key="c.path">{{ c.title }}</a-menu-item>
          </a-sub-menu>
          <a-menu-item v-else :key="m.path">
            <template #icon>
              <MenuIcon :name="m.icon" /> </template
            >{{ m.title }}
          </a-menu-item>
        </template>
      </a-menu>
    </a-layout-sider>
    <a-drawer v-model:open="mobileOpen" placement="left" :width="208" :closable="false" class="mobile-nav">
      <div class="brand">
        <span class="logo">M</span>
        <span class="name">{{ brand.name }}</span>
      </div>
      <a-menu v-model:selected-keys="selectedKeys" mode="inline" :theme="layout.siderTheme" data-testid="side-menu" @click="go">
        <a-menu-item key="/">首页</a-menu-item>
        <a-sub-menu v-for="m in auth.menus.filter((x) => x.children?.length)" :key="m.path" :title="m.title">
          <a-menu-item v-for="c in m.children" :key="c.path">{{ c.title }}</a-menu-item>
        </a-sub-menu>
      </a-menu>
    </a-drawer>
    <a-layout class="main">
      <a-layout-header class="header">
        <button class="icon-button" aria-label="切换菜单" @click="toggle">
          <MenuUnfoldOutlined v-if="collapsed || mobile" />
          <MenuFoldOutlined v-else />
        </button>
        <a-breadcrumb class="crumb">
          <a-breadcrumb-item v-if="parent">{{ parent.title }}</a-breadcrumb-item>
          <a-breadcrumb-item>{{ pageTitle }}</a-breadcrumb-item>
        </a-breadcrumb>
        <a-select
          v-model:value="factory"
          class="factory desktop"
          :options="availableFactories.map((f) => ({ value: f.id, label: f.name }))"
          :disabled="availableFactories.length < 2"
        />
        <a-input class="search desktop" placeholder="搜索项目 / 批次 / QR…">
          <template #prefix>
            <SearchOutlined />
          </template>
        </a-input>
        <div class="spacer" />
        <div class="shift desktop"><span class="live-dot" />早班 08:00–17:00 <b>节拍: 正常</b></div>
        <a-tag v-if="isDemo" class="desktop" color="orange">演示环境 · 数据为虚构</a-tag>
        <a-badge dot class="desktop">
          <button class="icon-button" aria-label="通知">
            <BellOutlined />
          </button>
        </a-badge>
        <button class="icon-button desktop" aria-label="帮助">
          <QuestionCircleOutlined />
        </button>
        <button class="icon-button desktop" aria-label="全屏" @click="fullscreen">
          <FullscreenOutlined />
        </button>
        <a-dropdown>
          <span class="user" data-testid="user-menu">
            <a-avatar class="avatar">{{ auth.user?.displayName?.slice(0, 1) }}</a-avatar>
            <span class="user-name">{{ auth.user?.displayName }}{{ auth.user?.isAdmin ? ' (管理员)' : ' (操作员)' }}</span>
          </span>
          <template #overlay>
            <a-menu>
              <a-menu-item key="logout" data-testid="logout" @click="logout"> <LogoutOutlined /> 退出登录 </a-menu-item>
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
.sider {
  min-height: 100vh;
  background: var(--gz-card);
  border-right: 1px solid var(--gz-border);
}
.brand {
  display: flex;
  align-items: center;
  gap: 10px;
  height: 56px;
  padding: 0 12px;
  color: var(--gz-text);
  white-space: nowrap;
  overflow: hidden;
}
.logo {
  display: grid;
  place-items: center;
  flex: none;
  width: 28px;
  height: 28px;
  border-radius: 6px;
  background: var(--gz-primary);
  font-weight: 700;
}
.name {
  font-size: 16px;
  font-weight: 700;
}
.name small {
  font-size: 10px;
  color: var(--gz-primary);
}
.main {
  min-width: 0;
}
.header {
  height: 56px;
  line-height: normal;
  display: flex;
  align-items: center;
  gap: 16px;
  padding: 0 16px;
  background: var(--gz-card);
  border-bottom: 1px solid var(--gz-border);
}
.icon-button {
  border: 0;
  background: none;
  cursor: pointer;
  font-size: 18px;
  color: var(--gz-secondary);
  padding: 4px;
}
.crumb {
  min-width: 130px;
}
.factory {
  width: 145px;
}
.search {
  max-width: 230px;
}
.spacer {
  flex: 1;
}
.shift {
  display: flex;
  align-items: center;
  gap: 8px;
  white-space: nowrap;
  font-size: 12px;
  color: var(--gz-secondary);
  background: var(--gz-bg);
  border-radius: 9999px;
  padding: 8px 12px;
}
.shift b {
  color: var(--gz-text);
  border-left: 1px solid var(--gz-border);
  padding-left: 8px;
}
.live-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: var(--gz-success);
}
.user {
  display: flex;
  align-items: center;
  gap: 8px;
  cursor: pointer;
  white-space: nowrap;
}
.avatar {
  background: var(--gz-primary);
}
.content {
  padding: 16px;
  background: var(--gz-bg);
  min-height: calc(100vh - 56px);
}
:deep(.ant-menu-light) {
  background: var(--gz-card);
  border-inline-end: 0 !important;
}
:deep(.ant-menu-item-selected) {
  background: #e6f4ff;
  font-weight: 500;
}
:global(.mobile-nav .ant-drawer-body) {
  background: var(--gz-card);
  padding: 0;
}
@media (max-width: 1100px) {
  .shift {
    display: none;
  }
}
@media (max-width: 768px) {
  .desktop {
    display: none;
  }
  .header {
    gap: 8px;
  }
  .crumb {
    min-width: 0;
  }
  .user-name {
    display: none;
  }
  .content {
    padding: 12px;
  }
}
</style>
