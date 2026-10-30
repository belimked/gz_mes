<script setup lang="ts">
// Tech style: "digital twin" curtain-wall facade in 3D; panels light up as a scan line passes. HUD-framed form.
import { brand } from '@/theme'
import BrandLogo from './BrandLogo.vue'
import LoginForm from './LoginForm.vue'
import { techTheme } from './techTheme'

const t = techTheme('#22d3ee')
const COLS = 10
const ROWS = 7
const delay = (i: number) => `${Math.floor(i / COLS) * 0.5 + (i % COLS) * 0.04}s`
// Some panels are "pending" (not yet produced) to make the facade feel real.
const pending = new Set([7, 18, 19, 33, 46, 52, 61, 68])
const tags = [
  { top: '14%', left: '62%', text: 'BQ-20261030-00128 · 喷涂完成' },
  { top: '46%', left: '4%', text: 'PC-20261030-012 · 进度 78%' },
  { top: '78%', left: '55%', text: '外协A · 在线 · 312 件' },
]
</script>

<template>
  <div class="page">
    <div class="floor" aria-hidden="true" />
    <section class="stage" aria-hidden="true">
      <div class="facade">
        <span
          v-for="i in COLS * ROWS"
          :key="i"
          class="pn"
          :class="{ pending: pending.has(i - 1) }"
          :style="{ animationDelay: delay(i - 1) }"
        />
        <span class="scan" />
      </div>
      <span v-for="g in tags" :key="g.text" class="tag" :style="{ top: g.top, left: g.left }"><i />{{ g.text }}</span>
      <div class="caption">
        <b>数字孪生 · 幕墙板全生命周期</b>
        <span>{{ ROWS * COLS }} 块板 · {{ ROWS * COLS - pending.size }} 已完工 · 实时同步</span>
      </div>
    </section>

    <section class="panel">
      <span class="c tl" /><span class="c tr" /><span class="c bl" /><span class="c br" />
      <div class="brand"><BrandLogo bg="#22d3ee" fg="#04121a" :size="32" /> <span>{{ brand.name }}</span></div>
      <div class="mono">AUTH / 身份验证</div>
      <h1>登录控制台</h1>
      <a-config-provider :theme="t">
        <LoginForm dark accent="#22d3ee" />
      </a-config-provider>
    </section>
  </div>
</template>

<style scoped>
.page {
  position: relative;
  display: flex;
  gap: 48px;
  align-items: center;
  justify-content: center;
  min-height: 100vh;
  padding: 40px 48px 96px;
  overflow: hidden;
  color: #cffafe;
  background: radial-gradient(ellipse at 30% 40%, #0b2540 0%, #020617 60%);
}
.floor {
  position: absolute;
  right: -20%;
  bottom: -30%;
  left: -20%;
  height: 60%;
  background-image:
    linear-gradient(rgb(34 211 238 / 12%) 1px, transparent 1px),
    linear-gradient(90deg, rgb(34 211 238 / 12%) 1px, transparent 1px);
  background-size: 60px 60px;
  transform: perspective(600px) rotateX(65deg);
  mask-image: linear-gradient(to top, #000, transparent 80%);
}
.stage {
  position: relative;
  flex: 1;
  max-width: 640px;
  height: 520px;
}
.facade {
  position: absolute;
  inset: 40px 40px 80px;
  display: grid;
  grid-template-columns: repeat(10, 1fr);
  gap: 5px;
  overflow: hidden;
  padding: 10px;
  border: 1px solid rgb(34 211 238 / 25%);
  background: rgb(2 6 23 / 60%);
  transform: perspective(1100px) rotateY(24deg) rotateX(6deg);
  box-shadow: 0 0 60px rgb(34 211 238 / 12%);
}
.pn {
  border: 1px solid rgb(34 211 238 / 22%);
  background: rgb(34 211 238 / 4%);
  animation: lit 7s linear infinite;
}
.pn.pending {
  border-style: dashed;
  border-color: rgb(250 204 21 / 35%);
  background: transparent;
  animation: none;
}
@keyframes lit {
  0%,
  100% {
    background: rgb(34 211 238 / 4%);
  }
  6% {
    border-color: #67e8f9;
    background: rgb(34 211 238 / 55%);
    box-shadow: 0 0 14px rgb(34 211 238 / 70%);
  }
  30% {
    background: rgb(34 211 238 / 14%);
    box-shadow: none;
  }
}
.scan {
  position: absolute;
  right: 0;
  left: 0;
  height: 2px;
  background: linear-gradient(90deg, transparent, #67e8f9, transparent);
  box-shadow: 0 0 18px 4px rgb(103 232 249 / 55%);
  animation: sweep 3.5s linear infinite;
}
@keyframes sweep {
  from {
    top: 0;
  }
  to {
    top: 100%;
  }
}
.tag {
  position: absolute;
  display: inline-flex;
  align-items: center;
  gap: 8px;
  padding: 4px 10px;
  border: 1px solid rgb(34 211 238 / 35%);
  background: rgb(2 6 23 / 80%);
  color: #a5f3fc;
  font-family: ui-monospace, monospace;
  font-size: 12px;
  white-space: nowrap;
  backdrop-filter: blur(6px);
}
.tag i {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: #22d3ee;
  box-shadow: 0 0 8px #22d3ee;
}
.caption {
  position: absolute;
  bottom: 0;
  left: 40px;
  display: flex;
  flex-direction: column;
  gap: 4px;
}
.caption b {
  color: #fff;
  font-size: 22px;
  font-weight: 600;
}
.caption span {
  color: rgb(207 250 254 / 55%);
  font-family: ui-monospace, monospace;
  font-size: 12px;
}
.panel {
  position: relative;
  width: 400px;
  padding: 36px 32px 28px;
  border: 1px solid rgb(34 211 238 / 18%);
  background: rgb(4 14 30 / 82%);
  backdrop-filter: blur(16px);
}
.c {
  position: absolute;
  width: 16px;
  height: 16px;
  border-color: #22d3ee;
  border-style: solid;
}
.tl {
  top: -1px;
  left: -1px;
  border-width: 2px 0 0 2px;
}
.tr {
  top: -1px;
  right: -1px;
  border-width: 2px 2px 0 0;
}
.bl {
  bottom: -1px;
  left: -1px;
  border-width: 0 0 2px 2px;
}
.br {
  right: -1px;
  bottom: -1px;
  border-width: 0 2px 2px 0;
}
.brand {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 28px;
  color: #fff;
  font-weight: 600;
}
.mono {
  color: #22d3ee;
  font-family: ui-monospace, monospace;
  font-size: 12px;
  letter-spacing: 2px;
}
h1 {
  margin: 6px 0 24px;
  color: #fff;
  font-size: 26px;
  font-weight: 600;
}
.panel :deep(.ant-btn-primary) {
  color: #04121a;
  font-weight: 600;
  box-shadow: 0 0 24px rgb(34 211 238 / 40%);
}
@media (width <= 1000px) {
  .stage {
    display: none;
  }
  .panel {
    width: 100%;
    max-width: 400px;
  }
}
@media (prefers-reduced-motion: reduce) {
  .pn,
  .scan {
    animation: none;
  }
}
</style>
