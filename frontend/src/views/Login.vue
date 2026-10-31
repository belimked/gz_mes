<script setup lang="ts">
import { computed, type Component } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { isDemo } from '@/api/client'
import { isLoginStyle, layout, loginStyles, type LoginStyle } from '@/theme'
import AuroraLogin from './login/AuroraLogin.vue'
import NebulaLogin from './login/NebulaLogin.vue'
import TideLogin from './login/TideLogin.vue'
import EmberLogin from './login/EmberLogin.vue'

const views: Record<LoginStyle, Component> = {
  aurora: AuroraLogin,
  nebula: NebulaLogin,
  tide: TideLogin,
  ember: EmberLogin,
}

const route = useRoute()
const router = useRouter()

const current = computed<LoginStyle>(() => (isLoginStyle(route.query.style) ? route.query.style : layout.loginStyle))
// The switcher is for previewing styles: always on the demo site, elsewhere only when ?style= is used.
const showSwitcher = computed(() => isDemo || route.query.style !== undefined)
const options = loginStyles.map((s) => ({ value: s.key, label: s.label }))

function pick(v: string | number) {
  if (isLoginStyle(v)) void router.replace({ query: { ...route.query, style: v } })
}
</script>

<template>
  <component :is="views[current]" />
  <div v-if="showSwitcher" class="switcher" data-testid="login-style-switcher">
    <span class="label">登录页风格</span>
    <a-segmented :value="current" :options="options" @change="pick" />
  </div>
</template>

<style scoped>
.switcher {
  position: fixed;
  right: 16px;
  bottom: 16px;
  z-index: 100;
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 6px 6px 6px 14px;
  border: 1px solid rgb(0 0 0 / 6%);
  border-radius: 10px;
  background: rgb(255 255 255 / 92%);
  box-shadow: 0 6px 24px rgb(0 0 0 / 15%);
  backdrop-filter: blur(8px);
}
.label {
  color: #595959;
  font-size: 12px;
  white-space: nowrap;
}
@media (width <= 700px) {
  .switcher {
    right: 8px;
    left: 8px;
    overflow-x: auto;
  }
  .label {
    display: none;
  }
}
</style>
