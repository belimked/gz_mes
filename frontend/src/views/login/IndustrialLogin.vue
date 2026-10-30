<script setup lang="ts">
// Style 4: factory-floor feel — orange accent, hazard stripe, large photo-like hero block.
import { brand } from '@/theme'
import BrandLogo from './BrandLogo.vue'
import LoginForm from './LoginForm.vue'

const orangeTheme = { token: { colorPrimary: '#fa6400', borderRadius: 2 } }
const steps = ['开料', '折弯', '钣金', '喷涂', '质检', '装架', '发货']
</script>

<template>
  <div class="page">
    <section class="form-side">
      <div class="box">
        <div class="brand"><BrandLogo bg="#fa6400" /> <span>{{ brand.name }}</span></div>
        <h1>登录系统</h1>
        <p class="sub">{{ brand.subtitle }}</p>
        <a-config-provider :theme="orangeTheme">
          <LoginForm accent="#fa6400" />
        </a-config-provider>
      </div>
    </section>
    <section class="hero">
      <div class="stripe" aria-hidden="true" />
      <div class="hero-inner">
        <span class="tag">PRODUCTION LINE</span>
        <h2>每一道工序<br />都有记录</h2>
        <div class="line">
          <div v-for="(s, i) in steps" :key="s" class="node">
            <span class="num">{{ String(i + 1).padStart(2, '0') }}</span>
            <span class="name">{{ s }}</span>
          </div>
        </div>
        <p>扫码报工 · 交接留痕 · 不良可追</p>
      </div>
      <div class="stripe bottom" aria-hidden="true" />
    </section>
  </div>
</template>

<style scoped>
.page {
  display: flex;
  min-height: 100vh;
  background: #fff;
}
.form-side {
  display: flex;
  flex: 1;
  align-items: center;
  justify-content: center;
  padding: 40px 24px 96px;
  border-top: 6px solid #fa6400;
}
.box {
  width: 100%;
  max-width: 380px;
}
.brand {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 40px;
  font-size: 18px;
  font-weight: 700;
}
h1 {
  margin: 0;
  font-size: 30px;
  font-weight: 700;
}
.sub {
  margin: 6px 0 28px;
  color: #8c8c8c;
}
.hero {
  position: relative;
  display: flex;
  flex: 1.1;
  flex-direction: column;
  justify-content: center;
  overflow: hidden;
  color: #fff;
  background:
    linear-gradient(rgb(20 20 20 / 82%), rgb(20 20 20 / 92%)),
    repeating-linear-gradient(90deg, #3a3a3a 0 2px, #2a2a2a 2px 80px);
}
.stripe {
  position: absolute;
  top: 0;
  right: 0;
  left: 0;
  height: 14px;
  background: repeating-linear-gradient(-45deg, #fa6400 0 16px, #1a1a1a 16px 32px);
}
.stripe.bottom {
  top: auto;
  bottom: 0;
}
.hero-inner {
  padding: 0 64px;
}
.tag {
  color: #fa6400;
  font-size: 12px;
  font-weight: 700;
  letter-spacing: 4px;
}
h2 {
  margin: 12px 0 40px;
  color: #fff;
  font-size: 42px;
  font-weight: 800;
  line-height: 1.2;
}
.line {
  position: relative;
  display: flex;
  justify-content: space-between;
  max-width: 520px;
  margin-bottom: 32px;
}
.line::before {
  position: absolute;
  top: 17px;
  right: 18px;
  left: 18px;
  height: 2px;
  background: #fa6400;
  content: '';
}
.node {
  position: relative;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
}
.num {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 36px;
  height: 36px;
  border: 2px solid #fa6400;
  border-radius: 50%;
  background: #1a1a1a;
  color: #fa6400;
  font-size: 12px;
  font-weight: 700;
}
.name {
  font-size: 13px;
}
.hero-inner > p {
  color: rgb(255 255 255 / 60%);
  letter-spacing: 2px;
}
@media (width <= 900px) {
  .hero {
    display: none;
  }
}
</style>
