<script setup lang="ts">
// Style 1: blue brand panel on the left, form on the right.
import { brand } from '@/theme'
import BrandLogo from './BrandLogo.vue'
import LoginForm from './LoginForm.vue'

const flow = ['合同', '批次', '面积表', '标签', '工序扫描', '发货', '对账']
const features = [
  { title: '一件一码', desc: '每块板材独立标签，工序扫描全程可追溯' },
  { title: '厂区隔离', desc: '本厂与外协厂数据分开，按账号授权可见' },
  { title: '业财一体', desc: '面积自动核算，发货即对账' },
]
</script>

<template>
  <div class="page">
    <section class="hero">
      <div class="hero-top">
        <BrandLogo bg="#fff" fg="#1677ff" />
        <span class="hero-name">{{ brand.name }}</span>
      </div>
      <div class="hero-body">
        <h2>{{ brand.subtitle }}</h2>
        <p class="slogan">从合同到发货，每一块板都看得见</p>
        <div class="panels" aria-hidden="true">
          <span v-for="i in 12" :key="i" class="panel" :style="{ animationDelay: `${(i % 6) * 0.35}s` }" />
        </div>
        <div class="flow">
          <template v-for="(s, i) in flow" :key="s">
            <span class="step">{{ s }}</span>
            <span v-if="i < flow.length - 1" class="arrow">›</span>
          </template>
        </div>
        <ul class="features">
          <li v-for="f in features" :key="f.title">
            <b>{{ f.title }}</b><span>{{ f.desc }}</span>
          </li>
        </ul>
      </div>
      <div class="hero-foot">© {{ new Date().getFullYear() }} {{ brand.name }}</div>
    </section>

    <section class="side">
      <div class="box">
        <div class="mobile-brand"><BrandLogo /> <span>{{ brand.name }}</span></div>
        <h1>欢迎登录</h1>
        <p class="sub">{{ brand.subtitle }}</p>
        <LoginForm />
      </div>
    </section>
  </div>
</template>

<style scoped>
.page {
  display: flex;
  min-height: 100vh;
  background: #f5f7fa;
}
.hero {
  display: flex;
  flex: 1.2;
  flex-direction: column;
  padding: 40px 56px;
  overflow: hidden;
  color: #fff;
  background:
    radial-gradient(circle at 85% 15%, rgb(64 150 255 / 35%), transparent 45%),
    radial-gradient(circle at 10% 90%, rgb(19 194 194 / 25%), transparent 40%),
    linear-gradient(150deg, #0b1f4b 0%, #0e3a8a 55%, #1677ff 100%);
}
.hero-top {
  display: flex;
  align-items: center;
  gap: 12px;
}
.hero-name {
  font-size: 20px;
  font-weight: 600;
  letter-spacing: 1px;
}
.hero-body {
  display: flex;
  flex: 1;
  flex-direction: column;
  justify-content: center;
  max-width: 560px;
}
.hero-body h2 {
  margin: 0;
  color: #fff;
  font-size: 36px;
  font-weight: 600;
  letter-spacing: 2px;
}
.slogan {
  margin: 12px 0 36px;
  color: rgb(255 255 255 / 75%);
  font-size: 16px;
}
.panels {
  display: grid;
  grid-template-columns: repeat(6, 1fr);
  gap: 8px;
  max-width: 420px;
  margin-bottom: 32px;
  transform: perspective(600px) rotateX(18deg) rotateZ(-4deg);
}
.panel {
  aspect-ratio: 3 / 4;
  border: 1px solid rgb(255 255 255 / 35%);
  border-radius: 3px;
  background: linear-gradient(160deg, rgb(255 255 255 / 22%), rgb(255 255 255 / 4%));
  animation: shine 4.2s ease-in-out infinite;
}
@keyframes shine {
  50% {
    background-color: rgb(255 255 255 / 18%);
  }
}
.flow {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 6px;
  margin-bottom: 32px;
}
.step {
  padding: 3px 10px;
  border: 1px solid rgb(255 255 255 / 30%);
  border-radius: 12px;
  background: rgb(255 255 255 / 8%);
  font-size: 13px;
}
.arrow {
  color: rgb(255 255 255 / 50%);
}
.features {
  display: grid;
  gap: 14px;
  margin: 0;
  padding: 0;
  list-style: none;
}
.features li {
  display: flex;
  gap: 12px;
}
.features b {
  min-width: 64px;
}
.features span {
  color: rgb(255 255 255 / 70%);
}
.hero-foot {
  color: rgb(255 255 255 / 50%);
  font-size: 12px;
}
.side {
  display: flex;
  flex: 1;
  align-items: center;
  justify-content: center;
  padding: 40px 24px 96px;
}
.box {
  width: 100%;
  max-width: 380px;
}
.mobile-brand {
  display: none;
  align-items: center;
  gap: 10px;
  margin-bottom: 24px;
  font-size: 18px;
  font-weight: 600;
}
h1 {
  margin: 0;
  font-size: 28px;
  font-weight: 600;
}
.sub {
  margin: 6px 0 28px;
  color: #8c8c8c;
}
@media (width <= 900px) {
  .hero {
    display: none;
  }
  .mobile-brand {
    display: flex;
  }
}
@media (prefers-reduced-motion: reduce) {
  .panel {
    animation: none;
  }
}
</style>
