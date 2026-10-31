<script setup lang="ts">
// Tech style (aurora family): vertical aurora curtains sweeping across a teal night sky, with a horizon line.
import { brand } from '@/theme'
import BrandLogo from './BrandLogo.vue'
import LoginForm from './LoginForm.vue'
import { techTheme } from './techTheme'

const t = techTheme('#2dd4bf')
const metrics = [
  { value: '0', label: '纸质流转' },
  { value: '1', label: '条数据链' },
  { value: '24h', label: '实时看板' },
]
</script>

<template>
  <div class="page">
    <div class="curtain" aria-hidden="true"><span class="c1" /><span class="c2" /><span class="c3" /><span class="c4" /></div>
    <div class="horizon" aria-hidden="true" />
    <div class="wrap">
      <section class="intro">
        <div class="brand">
          <BrandLogo bg="linear-gradient(135deg,#2dd4bf,#4ade80)" /> <span>{{ brand.name }}</span>
        </div>
        <span class="pill"><i /> 在线 · 生产数据实时同步</span>
        <h2>看得见的生产<br /><em>算得清的发货</em></h2>
        <p>{{ brand.subtitle }}。每块板一件一码，每道工序扫码留痕，发货即对账。</p>
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
          <LoginForm dark accent="#2dd4bf" />
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
  color: #e2f5f1;
  background: linear-gradient(180deg, #020a12 0%, #041a22 55%, #06282c 100%);
}
.curtain span {
  position: absolute;
  top: -20%;
  height: 90%;
  border-radius: 50% 50% 40% 40% / 10% 10% 90% 90%;
  filter: blur(50px);
  mix-blend-mode: screen;
  transform-origin: top center;
  animation: sway 14s ease-in-out infinite alternate;
}
.c1 {
  left: 2%;
  width: 22vw;
  background: linear-gradient(180deg, transparent, #14b8a6 35%, #4ade80 60%, transparent 95%);
  opacity: 0.55;
}
.c2 {
  left: 24%;
  width: 16vw;
  background: linear-gradient(180deg, transparent, #22d3ee 30%, #2dd4bf 65%, transparent 95%);
  opacity: 0.45;
  animation-delay: -4s;
}
.c3 {
  left: 46%;
  width: 26vw;
  background: linear-gradient(180deg, transparent, #10b981 25%, #a7f3d0 55%, transparent 90%);
  opacity: 0.35;
  animation-delay: -8s;
}
.c4 {
  left: 74%;
  width: 20vw;
  background: linear-gradient(180deg, transparent, #0ea5e9 30%, #34d399 60%, transparent 95%);
  opacity: 0.4;
  animation-delay: -11s;
}
@keyframes sway {
  from {
    transform: skewX(-8deg) translateX(-3vw) scaleY(0.95);
  }
  to {
    transform: skewX(8deg) translateX(4vw) scaleY(1.08);
  }
}
.horizon {
  position: absolute;
  right: 0;
  bottom: 0;
  left: 0;
  height: 32%;
  background:
    linear-gradient(180deg, transparent, rgb(2 10 18 / 85%) 40%),
    repeating-linear-gradient(90deg, rgb(45 212 191 / 10%) 0 1px, transparent 1px 80px);
  mask-image: linear-gradient(to top, #000 30%, transparent);
}
.horizon::before {
  position: absolute;
  top: 0;
  right: 0;
  left: 0;
  height: 1px;
  background: linear-gradient(90deg, transparent, rgb(45 212 191 / 70%) 30%, rgb(74 222 128 / 70%) 70%, transparent);
  box-shadow: 0 0 24px rgb(45 212 191 / 50%);
  content: '';
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
  border: 1px solid rgb(45 212 191 / 25%);
  border-radius: 999px;
  background: rgb(45 212 191 / 8%);
  color: rgb(226 245 241 / 85%);
  font-size: 12px;
  backdrop-filter: blur(8px);
}
.pill i {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: #4ade80;
  box-shadow: 0 0 10px #4ade80;
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
  background: linear-gradient(90deg, #5eead4, #86efac 50%, #67e8f9);
  background-clip: text;
  -webkit-text-fill-color: transparent;
}
.intro > p {
  max-width: 480px;
  margin: 0 0 40px;
  color: rgb(226 245 241 / 60%);
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
  padding-left: 12px;
  border-left: 2px solid rgb(45 212 191 / 50%);
}
.metrics b {
  color: #fff;
  font-size: 26px;
  font-weight: 600;
  font-variant-numeric: tabular-nums;
}
.metrics span {
  color: rgb(226 245 241 / 50%);
  font-size: 12px;
}
.card {
  position: relative;
  width: 420px;
  padding: 40px 36px 32px;
  border: 1px solid rgb(45 212 191 / 18%);
  border-radius: 20px;
  background: linear-gradient(160deg, rgb(8 36 42 / 70%), rgb(4 18 26 / 70%));
  box-shadow:
    inset 0 1px 0 rgb(255 255 255 / 10%),
    0 0 0 1px rgb(0 0 0 / 30%),
    0 30px 80px rgb(0 0 0 / 50%),
    0 0 60px rgb(45 212 191 / 12%);
  backdrop-filter: blur(24px);
}
.card::after {
  position: absolute;
  top: 0;
  right: 15%;
  left: 15%;
  height: 1px;
  background: linear-gradient(90deg, transparent, #5eead4, transparent);
  content: '';
}
h1 {
  margin: 0;
  color: #fff;
  font-size: 26px;
  font-weight: 600;
}
.sub {
  margin: 6px 0 26px;
  color: rgb(226 245 241 / 50%);
}
.card :deep(.ant-btn-primary) {
  border: 0;
  background: linear-gradient(90deg, #14b8a6, #4ade80);
  box-shadow: 0 10px 30px rgb(45 212 191 / 30%);
  color: #04281f;
  font-weight: 600;
}
.card :deep(.ant-btn-primary:hover) {
  filter: brightness(1.08);
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
  .curtain span {
    animation: none;
  }
}
</style>
