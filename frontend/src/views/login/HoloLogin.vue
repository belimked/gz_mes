<script setup lang="ts">
// Tech style: a slowly rotating wireframe aluminium panel (3D CSS) with holographic iridescence.
import { brand } from '@/theme'
import BrandLogo from './BrandLogo.vue'
import LoginForm from './LoginForm.vue'
import { techTheme } from './techTheme'

const t = techTheme('#ec4899')
const specs = [
  ['展开面积', '0.2793 m²'],
  ['计价面积', '0.5586 m²'],
  ['颜色', 'RAL 9006'],
  ['工序', '喷涂质检 ✓'],
]
</script>

<template>
  <div class="page">
    <div class="rings" aria-hidden="true"><span /><span /><span /></div>
    <section class="stage" aria-hidden="true">
      <div class="obj">
        <div class="face front"><i v-for="n in 24" :key="n" /></div>
        <div class="face back" />
        <div class="face side l" />
        <div class="face side r" />
      </div>
      <dl class="specs">
        <template v-for="[k, v] in specs" :key="k">
          <dt>{{ k }}</dt>
          <dd>{{ v }}</dd>
        </template>
      </dl>
    </section>
    <section class="panel">
      <div class="brand"><BrandLogo bg="linear-gradient(135deg,#ec4899,#8b5cf6,#06b6d4)" :size="32" /> <span>{{ brand.name }}</span></div>
      <h1>登录 <span>{{ brand.subtitle }}</span></h1>
      <a-config-provider :theme="t">
        <LoginForm dark accent="#ec4899" />
      </a-config-provider>
    </section>
  </div>
</template>

<style scoped>
.page {
  position: relative;
  display: flex;
  gap: 64px;
  align-items: center;
  justify-content: center;
  min-height: 100vh;
  padding: 40px 32px 96px;
  overflow: hidden;
  color: #f1f5f9;
  background: radial-gradient(circle at 30% 50%, #1e1036 0%, #07040f 60%);
}
.rings span {
  position: absolute;
  top: 50%;
  left: 30%;
  border: 1px solid rgb(236 72 153 / 15%);
  border-radius: 50%;
  transform: translate(-50%, -50%);
}
.rings span:nth-child(1) {
  width: 420px;
  height: 420px;
}
.rings span:nth-child(2) {
  width: 620px;
  height: 620px;
  border-color: rgb(139 92 246 / 12%);
}
.rings span:nth-child(3) {
  width: 840px;
  height: 840px;
  border-color: rgb(6 182 212 / 8%);
}
.stage {
  position: relative;
  display: flex;
  flex: 1;
  flex-direction: column;
  align-items: center;
  max-width: 560px;
  perspective: 1000px;
}
.obj {
  position: relative;
  width: 240px;
  height: 320px;
  transform-style: preserve-3d;
  animation: turn 14s linear infinite;
}
@keyframes turn {
  from {
    transform: rotateX(12deg) rotateY(0deg);
  }
  to {
    transform: rotateX(12deg) rotateY(360deg);
  }
}
.face {
  position: absolute;
  inset: 0;
  border: 1px solid rgb(255 255 255 / 45%);
}
.front {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 1px;
  background: linear-gradient(130deg, rgb(236 72 153 / 35%), rgb(139 92 246 / 25%) 40%, rgb(6 182 212 / 35%));
  transform: translateZ(14px);
  box-shadow: 0 0 60px rgb(139 92 246 / 35%);
}
.front i {
  border: 1px dashed rgb(255 255 255 / 12%);
}
.back {
  background: rgb(6 182 212 / 12%);
  transform: rotateY(180deg) translateZ(14px);
}
.side {
  width: 28px;
  background: rgb(255 255 255 / 18%);
}
.l {
  left: -14px;
  transform: rotateY(-90deg);
}
.r {
  left: auto;
  right: -14px;
  transform: rotateY(90deg);
}
.specs {
  display: grid;
  grid-template-columns: auto auto;
  gap: 6px 24px;
  margin: 56px 0 0;
  padding: 14px 20px;
  border: 1px solid rgb(255 255 255 / 10%);
  border-radius: 12px;
  background: rgb(255 255 255 / 4%);
  font-size: 13px;
  backdrop-filter: blur(8px);
}
dt {
  color: rgb(241 245 249 / 50%);
}
dd {
  margin: 0;
  font-family: ui-monospace, monospace;
}
.panel {
  position: relative;
  width: 400px;
  padding: 36px 32px 28px;
  border-radius: 20px;
  background: rgb(20 12 36 / 70%);
  box-shadow:
    inset 0 0 0 1px rgb(255 255 255 / 8%),
    0 30px 80px rgb(0 0 0 / 50%);
  backdrop-filter: blur(20px);
}
.brand {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 28px;
  font-weight: 600;
}
h1 {
  display: flex;
  flex-direction: column;
  gap: 4px;
  margin: 0 0 24px;
  color: #fff;
  font-size: 28px;
}
h1 span {
  color: rgb(241 245 249 / 50%);
  font-size: 14px;
  font-weight: 400;
}
.panel :deep(.ant-btn-primary) {
  border: 0;
  background: linear-gradient(90deg, #ec4899, #8b5cf6, #06b6d4);
  box-shadow: 0 10px 30px rgb(139 92 246 / 35%);
}
@media (width <= 960px) {
  .stage,
  .rings {
    display: none;
  }
  .panel {
    width: 100%;
  }
}
@media (prefers-reduced-motion: reduce) {
  .obj {
    animation: none;
  }
}
</style>
