<script setup lang="ts">
// Tech style (aurora family): warm aluminium-furnace glow — amber/rose light bleeding through a charcoal sky,
// with slow-moving light streaks and a glass card.
import { brand } from '@/theme'
import BrandLogo from './BrandLogo.vue'
import LoginForm from './LoginForm.vue'
import { techTheme } from './techTheme'

const t = techTheme('#fb923c')
const metrics = [
  { value: '11', label: '工序在线' },
  { value: '100%', label: '一件一码' },
  { value: 'T+0', label: '发货即对账' },
]
</script>

<template>
  <div class="page">
    <div class="glow" aria-hidden="true"><span class="g1" /><span class="g2" /><span class="g3" /></div>
    <div class="streaks" aria-hidden="true"><i /><i /><i /><i /><i /></div>
    <div class="wrap">
      <section class="intro">
        <div class="brand">
          <BrandLogo bg="linear-gradient(135deg,#fb923c,#f43f5e)" /> <span>{{ brand.name }}</span>
        </div>
        <span class="pill"><i /> 车间在线 · 工序实时回传</span>
        <h2>从一张合同<br /><em>到一车铝板</em></h2>
        <p>{{ brand.subtitle }}。排程、面积表、标签、扫码、发货、对账，全部在一个系统里完成。</p>
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
          <LoginForm dark accent="#fb923c" />
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
  color: #f5ece6;
  background: radial-gradient(ellipse at 20% 110%, #2a0f0a 0%, #120a0c 45%, #07070b 100%);
}
.glow span {
  position: absolute;
  border-radius: 50%;
  filter: blur(100px);
  mix-blend-mode: screen;
  animation: drift 20s ease-in-out infinite alternate;
}
.g1 {
  bottom: -30%;
  left: -15%;
  width: 60vw;
  height: 60vw;
  background: #ea580c;
  opacity: 0.55;
}
.g2 {
  top: -25%;
  right: -10%;
  width: 48vw;
  height: 48vw;
  background: #be123c;
  opacity: 0.4;
  animation-delay: -7s;
}
.g3 {
  top: 35%;
  left: 38%;
  width: 26vw;
  height: 26vw;
  background: #fbbf24;
  opacity: 0.22;
  animation-delay: -13s;
}
@keyframes drift {
  to {
    transform: translate(6vw, -6vh) scale(1.15) rotate(-20deg);
  }
}
.streaks {
  position: absolute;
  inset: 0;
  overflow: hidden;
  mask-image: linear-gradient(to right, transparent, #000 20%, #000 80%, transparent);
}
.streaks i {
  position: absolute;
  left: -40%;
  width: 180%;
  height: 1px;
  background: linear-gradient(90deg, transparent, rgb(251 146 60 / 55%) 40%, rgb(253 164 175 / 70%) 55%, transparent);
  transform: rotate(-14deg);
  animation: slide 9s linear infinite;
}
.streaks i:nth-child(1) {
  top: 22%;
  opacity: 0.6;
}
.streaks i:nth-child(2) {
  top: 38%;
  opacity: 0.35;
  animation-delay: -3s;
}
.streaks i:nth-child(3) {
  top: 52%;
  opacity: 0.5;
  animation-delay: -6s;
}
.streaks i:nth-child(4) {
  top: 68%;
  opacity: 0.3;
  animation-delay: -1.5s;
}
.streaks i:nth-child(5) {
  top: 84%;
  opacity: 0.45;
  animation-delay: -4.5s;
}
@keyframes slide {
  from {
    transform: translateX(-30%) rotate(-14deg);
  }
  to {
    transform: translateX(30%) rotate(-14deg);
  }
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
  border: 1px solid rgb(251 146 60 / 25%);
  border-radius: 999px;
  background: rgb(251 146 60 / 8%);
  color: rgb(245 236 230 / 85%);
  font-size: 12px;
  backdrop-filter: blur(8px);
}
.pill i {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: #fbbf24;
  box-shadow: 0 0 10px #fbbf24;
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
  background: linear-gradient(90deg, #fdba74, #fda4af 50%, #fde68a);
  background-clip: text;
  -webkit-text-fill-color: transparent;
}
.intro > p {
  max-width: 480px;
  margin: 0 0 40px;
  color: rgb(245 236 230 / 60%);
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
  color: rgb(245 236 230 / 50%);
  font-size: 12px;
}
.card {
  position: relative;
  width: 420px;
  padding: 40px 36px 32px;
  border-radius: 20px;
  background: rgb(28 18 20 / 62%);
  box-shadow:
    inset 0 1px 0 rgb(255 255 255 / 10%),
    inset 0 0 0 1px rgb(255 255 255 / 7%),
    0 30px 80px rgb(0 0 0 / 50%),
    0 0 80px rgb(251 146 60 / 10%);
  backdrop-filter: blur(24px) saturate(130%);
}
.card::before {
  position: absolute;
  inset: 0;
  border-radius: inherit;
  background: linear-gradient(135deg, rgb(251 146 60 / 35%), transparent 40%, transparent 60%, rgb(244 63 94 / 30%));
  content: '';
  pointer-events: none;
  mask:
    linear-gradient(#000 0 0) content-box,
    linear-gradient(#000 0 0);
  mask-composite: exclude;
  padding: 1px;
}
h1 {
  margin: 0;
  color: #fff;
  font-size: 26px;
  font-weight: 600;
}
.sub {
  margin: 6px 0 26px;
  color: rgb(245 236 230 / 50%);
}
.card :deep(.ant-btn-primary) {
  border: 0;
  background: linear-gradient(90deg, #fb923c, #f43f5e);
  box-shadow: 0 10px 30px rgb(251 146 60 / 35%);
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
  .glow span,
  .streaks i {
    animation: none;
  }
}
</style>
