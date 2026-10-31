<script setup lang="ts">
// Tech style: drifting aurora light, gradient headline, glass card with a rotating gradient border.
import { brand } from '@/theme'
import BrandLogo from './BrandLogo.vue'
import LoginForm from './LoginForm.vue'
import { techTheme } from './techTheme'

const t = techTheme('#7c6cff')
const metrics = [
  { value: '11', label: '工序在线' },
  { value: '100%', label: '一件一码' },
  { value: 'T+0', label: '发货即对账' },
]
</script>

<template>
  <div class="page">
    <div class="aurora" aria-hidden="true"><span class="a1" /><span class="a2" /><span class="a3" /></div>
    <div class="grain" aria-hidden="true" />
    <div class="wrap">
      <section class="intro">
        <div class="brand">
          <BrandLogo bg="linear-gradient(135deg,#7c6cff,#22d3ee)" /> <span>{{ brand.name }}</span>
        </div>
        <span class="pill"><i /> 系统运行中 · 全流程数字化</span>
        <h2>让每一块板<br /><em>在数据里流动</em></h2>
        <p>{{ brand.subtitle }}。合同、排程、扫码、发货、对账，一条数据链贯通。</p>
        <div class="metrics">
          <div v-for="m in metrics" :key="m.label">
            <b>{{ m.value }}</b
            ><span>{{ m.label }}</span>
          </div>
        </div>
      </section>
      <section class="card">
        <a-config-provider :theme="t">
          <h1>欢迎回来</h1>
          <p class="sub">登录以继续访问 {{ brand.name }}</p>
          <LoginForm dark accent="#7c6cff" />
        </a-config-provider>
      </section>
    </div>
  </div>
</template>

<style scoped>
@property --aurora-angle {
  syntax: '<angle>';
  initial-value: 0deg;
  inherits: false;
}
.page {
  position: relative;
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 100vh;
  padding: 40px 24px 96px;
  overflow: hidden;
  color: #e5e7eb;
  background: #030712;
}
.aurora span {
  position: absolute;
  border-radius: 50%;
  filter: blur(90px);
  opacity: 0.55;
  animation: drift 18s ease-in-out infinite alternate;
}
.a1 {
  top: -15%;
  left: -10%;
  width: 55vw;
  height: 55vw;
  background: #4f46e5;
}
.a2 {
  right: -15%;
  bottom: -25%;
  width: 50vw;
  height: 50vw;
  background: #0891b2;
  animation-delay: -6s !important;
}
.a3 {
  top: 30%;
  left: 35%;
  width: 30vw;
  height: 30vw;
  background: #a21caf;
  opacity: 0.3 !important;
  animation-delay: -12s !important;
}
@keyframes drift {
  to {
    transform: translate(8vw, 6vh) scale(1.15) rotate(25deg);
  }
}
.grain {
  position: absolute;
  inset: 0;
  background-image: radial-gradient(rgb(255 255 255 / 6%) 1px, transparent 1px);
  background-size: 3px 3px;
  mask-image: linear-gradient(to bottom, #000, transparent 85%);
}
.wrap {
  position: relative;
  display: flex;
  gap: 72px;
  align-items: center;
  width: 100%;
  max-width: 1120px;
}
.intro {
  flex: 1;
}
.brand {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 36px;
  font-size: 18px;
  font-weight: 600;
}
.pill {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  padding: 5px 12px;
  border: 1px solid rgb(255 255 255 / 12%);
  border-radius: 999px;
  background: rgb(255 255 255 / 5%);
  color: rgb(229 231 235 / 80%);
  font-size: 12px;
  backdrop-filter: blur(8px);
}
.pill i {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: #34d399;
  box-shadow: 0 0 10px #34d399;
}
h2 {
  margin: 22px 0 18px;
  color: #fff;
  font-size: 52px;
  font-weight: 700;
  line-height: 1.15;
  letter-spacing: 1px;
}
h2 em {
  font-style: normal;
  background: linear-gradient(90deg, #a5b4fc, #67e8f9 50%, #f0abfc);
  background-clip: text;
  -webkit-text-fill-color: transparent;
}
.intro > p {
  max-width: 480px;
  margin: 0 0 40px;
  color: rgb(229 231 235 / 60%);
  line-height: 1.8;
}
.metrics {
  display: flex;
  gap: 40px;
}
.metrics div {
  display: flex;
  flex-direction: column;
  gap: 4px;
}
.metrics b {
  color: #fff;
  font-size: 26px;
  font-weight: 600;
  font-variant-numeric: tabular-nums;
}
.metrics span {
  color: rgb(229 231 235 / 50%);
  font-size: 12px;
}
.card {
  position: relative;
  width: 420px;
  padding: 40px 36px 32px;
  border-radius: 20px;
  background: rgb(15 17 35 / 65%);
  box-shadow:
    inset 0 0 0 1px rgb(255 255 255 / 8%),
    0 30px 80px rgb(0 0 0 / 45%);
  backdrop-filter: blur(24px);
}
.card::before {
  position: absolute;
  inset: 0;
  padding: 1px;
  border-radius: inherit;
  background: conic-gradient(from var(--aurora-angle), transparent 0 55%, #7c6cff, #22d3ee, transparent 95%);
  content: '';
  pointer-events: none;
  mask:
    linear-gradient(#000 0 0) content-box,
    linear-gradient(#000 0 0);
  mask-composite: exclude;
  animation: spin 6s linear infinite;
}
@keyframes spin {
  to {
    --aurora-angle: 360deg;
  }
}
h1 {
  margin: 0;
  color: #fff;
  font-size: 26px;
  font-weight: 600;
}
.sub {
  margin: 6px 0 26px;
  color: rgb(229 231 235 / 50%);
}
.card :deep(.ant-btn-primary) {
  border: 0;
  background: linear-gradient(90deg, #7c6cff, #22d3ee);
  box-shadow: 0 10px 30px rgb(124 108 255 / 35%);
}
.card :deep(.ant-btn-primary:hover) {
  filter: brightness(1.1);
}
@media (width <= 900px) {
  .intro {
    display: none;
  }
  .card {
    width: 100%;
    max-width: 420px;
    margin: 0 auto;
  }
}
@media (prefers-reduced-motion: reduce) {
  .aurora span,
  .card::before {
    animation: none;
  }
}
</style>
