<script setup lang="ts">
// Style 3: dark control-room look with a glowing grid and live-looking stats.
import { theme as antTheme } from 'ant-design-vue'
import { brand } from '@/theme'
import BrandLogo from './BrandLogo.vue'
import LoginForm from './LoginForm.vue'

const darkTheme = { algorithm: antTheme.darkAlgorithm, token: { colorPrimary: '#13c2c2', borderRadius: 6 } }
const stats = [
  { label: '工序节点', value: '11' },
  { label: '标签追溯', value: '一件一码' },
  { label: '厂区协同', value: '本厂 + 外协' },
]
</script>

<template>
  <div class="page">
    <div class="grid" aria-hidden="true" />
    <div class="glow" aria-hidden="true" />
    <div class="wrap">
      <section class="intro">
        <div class="brand"><BrandLogo bg="#13c2c2" fg="#001a1a" /> <span>{{ brand.name }}</span></div>
        <h2>生产数据<br /><em>实时在线</em></h2>
        <p>{{ brand.subtitle }} · 排程、扫描、发货、对账一屏掌握</p>
        <div class="stats">
          <div v-for="s in stats" :key="s.label" class="stat">
            <b>{{ s.value }}</b><span>{{ s.label }}</span>
          </div>
        </div>
        <div class="scan" aria-hidden="true">
          <span class="dot" /> 14:32:05 · 钣金下线 · 标签 BQ-20261030-00128 · 已扫描
        </div>
      </section>
      <section class="panel">
        <a-config-provider :theme="darkTheme">
          <h1>账号登录</h1>
          <p class="sub">请使用分配的工号登录</p>
          <LoginForm dark accent="#13c2c2" />
        </a-config-provider>
      </section>
    </div>
  </div>
</template>

<style scoped>
.page {
  position: relative;
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 100vh;
  padding: 40px 24px 96px;
  overflow: hidden;
  color: #e6f4ff;
  background: #050b18;
}
.grid {
  position: absolute;
  inset: 0;
  background-image:
    linear-gradient(rgb(19 194 194 / 8%) 1px, transparent 1px),
    linear-gradient(90deg, rgb(19 194 194 / 8%) 1px, transparent 1px);
  background-size: 40px 40px;
  mask-image: radial-gradient(ellipse at center, #000 30%, transparent 75%);
}
.glow {
  position: absolute;
  top: 10%;
  left: 20%;
  width: 600px;
  height: 600px;
  border-radius: 50%;
  background: radial-gradient(circle, rgb(19 194 194 / 22%), transparent 65%);
}
.wrap {
  position: relative;
  display: flex;
  gap: 64px;
  align-items: center;
  width: 100%;
  max-width: 1080px;
}
.intro {
  flex: 1;
}
.brand {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 40px;
  font-size: 18px;
  font-weight: 600;
}
h2 {
  margin: 0;
  color: #fff;
  font-size: 44px;
  line-height: 1.25;
}
h2 em {
  font-style: normal;
  background: linear-gradient(90deg, #36cfc9, #69b1ff);
  background-clip: text;
  -webkit-text-fill-color: transparent;
}
.intro > p {
  margin: 16px 0 36px;
  color: rgb(230 244 255 / 60%);
}
.stats {
  display: flex;
  gap: 16px;
  margin-bottom: 28px;
}
.stat {
  display: flex;
  flex-direction: column;
  gap: 4px;
  padding: 14px 18px;
  border: 1px solid rgb(19 194 194 / 25%);
  border-radius: 8px;
  background: rgb(19 194 194 / 6%);
}
.stat b {
  color: #5cdbd3;
  font-size: 20px;
}
.stat span {
  color: rgb(230 244 255 / 55%);
  font-size: 12px;
}
.scan {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  color: rgb(230 244 255 / 50%);
  font-family: ui-monospace, monospace;
  font-size: 12px;
}
.dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: #52c41a;
  box-shadow: 0 0 8px #52c41a;
  animation: blink 1.6s infinite;
}
@keyframes blink {
  50% {
    opacity: 0.3;
  }
}
.panel {
  width: 400px;
  padding: 36px 32px;
  border: 1px solid rgb(19 194 194 / 25%);
  border-radius: 12px;
  background: rgb(10 22 40 / 85%);
  box-shadow:
    0 0 0 1px rgb(255 255 255 / 3%),
    0 20px 60px rgb(0 0 0 / 50%);
}
h1 {
  margin: 0;
  color: #fff;
  font-size: 24px;
}
.sub {
  margin: 6px 0 24px;
  color: rgb(230 244 255 / 50%);
}
@media (width <= 900px) {
  .intro {
    display: none;
  }
  .panel {
    width: 100%;
    max-width: 400px;
    margin: 0 auto;
  }
}
@media (prefers-reduced-motion: reduce) {
  .dot {
    animation: none;
  }
}
</style>
