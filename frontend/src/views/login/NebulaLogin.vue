<script setup lang="ts">
// Tech style (aurora family): deep-space nebula with a twinkling star field, magenta/violet glow, glass card.
import { brand } from '@/theme'
import BrandLogo from './BrandLogo.vue'
import LoginForm from './LoginForm.vue'
import { techTheme } from './techTheme'

const t = techTheme('#e879f9')
const steps = ['合同', '排程', '标签', '扫码', '发货', '对账']
</script>

<template>
  <div class="page">
    <div class="stars" aria-hidden="true" />
    <div class="nebula" aria-hidden="true"><span class="n1" /><span class="n2" /><span class="n3" /><span class="n4" /></div>
    <div class="wrap">
      <section class="intro">
        <div class="brand"><BrandLogo bg="linear-gradient(135deg,#e879f9,#8b5cf6)" /> <span>{{ brand.name }}</span></div>
        <h2>把工厂的每一步<br /><em>连成一条光带</em></h2>
        <p>{{ brand.subtitle }}。从销售合同到发货对账，一个系统，一条数据链。</p>
        <ol class="chain">
          <li v-for="(s, i) in steps" :key="s" :style="{ '--i': i }">
            <i /><span>{{ s }}</span>
          </li>
        </ol>
      </section>
      <section class="card">
        <a-config-provider :theme="t">
          <div class="halo" aria-hidden="true" />
          <h1>登录</h1>
          <p class="sub">欢迎回到 {{ brand.name }}</p>
          <LoginForm dark accent="#e879f9" />
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
  color: #e9e5f5;
  background: radial-gradient(ellipse at 50% 120%, #1a0b2e 0%, #07050f 60%, #020105 100%);
}
.stars {
  position: absolute;
  inset: 0;
  background-image:
    radial-gradient(1px 1px at 12% 18%, #fff 50%, transparent 51%),
    radial-gradient(1px 1px at 28% 72%, #fff 50%, transparent 51%),
    radial-gradient(1.5px 1.5px at 41% 33%, #fff 50%, transparent 51%),
    radial-gradient(1px 1px at 57% 86%, #fff 50%, transparent 51%),
    radial-gradient(1px 1px at 66% 12%, #fff 50%, transparent 51%),
    radial-gradient(1.5px 1.5px at 79% 48%, #fff 50%, transparent 51%),
    radial-gradient(1px 1px at 88% 78%, #fff 50%, transparent 51%),
    radial-gradient(1px 1px at 93% 24%, #fff 50%, transparent 51%),
    radial-gradient(1px 1px at 6% 56%, #fff 50%, transparent 51%),
    radial-gradient(1px 1px at 35% 94%, #fff 50%, transparent 51%),
    radial-gradient(1px 1px at 50% 6%, #fff 50%, transparent 51%),
    radial-gradient(1px 1px at 73% 64%, #fff 50%, transparent 51%);
  background-size: 100% 100%;
  opacity: 0.7;
  animation: twinkle 5s ease-in-out infinite alternate;
}
@keyframes twinkle {
  to {
    opacity: 0.35;
  }
}
.nebula span {
  position: absolute;
  border-radius: 50%;
  filter: blur(80px);
  mix-blend-mode: screen;
  animation: breathe 16s ease-in-out infinite alternate;
}
.n1 {
  top: -20%;
  left: 5%;
  width: 50vw;
  height: 40vw;
  background: #7e22ce;
  opacity: 0.55;
}
.n2 {
  top: 10%;
  right: -10%;
  width: 45vw;
  height: 45vw;
  background: #db2777;
  opacity: 0.4;
  animation-delay: -5s;
}
.n3 {
  bottom: -30%;
  left: 25%;
  width: 55vw;
  height: 40vw;
  background: #4338ca;
  opacity: 0.5;
  animation-delay: -10s;
}
.n4 {
  top: 40%;
  left: 40%;
  width: 18vw;
  height: 18vw;
  background: #f0abfc;
  opacity: 0.25;
  filter: blur(60px);
  animation-delay: -3s;
}
@keyframes breathe {
  to {
    transform: translate(-6vw, 5vh) scale(1.2);
  }
}
.wrap {
  position: relative;
  display: flex;
  gap: 80px;
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
  margin-bottom: 40px;
  font-size: 18px;
  font-weight: 600;
}
h2 {
  margin: 0 0 18px;
  color: #fff;
  font-size: 52px;
  font-weight: 700;
  line-height: 1.15;
  letter-spacing: 1px;
}
h2 em {
  font-style: normal;
  background: linear-gradient(90deg, #f0abfc, #c4b5fd 50%, #fda4af);
  background-clip: text;
  -webkit-text-fill-color: transparent;
}
.intro > p {
  max-width: 480px;
  margin: 0 0 40px;
  color: rgb(233 229 245 / 60%);
  line-height: 1.8;
}
.chain {
  display: flex;
  gap: 0;
  margin: 0;
  padding: 0;
  list-style: none;
}
.chain li {
  position: relative;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 10px;
  width: 76px;
  font-size: 12px;
  color: rgb(233 229 245 / 65%);
}
.chain li::after {
  position: absolute;
  top: 5px;
  left: 50%;
  width: 76px;
  height: 1px;
  background: linear-gradient(90deg, rgb(232 121 249 / 60%), rgb(139 92 246 / 20%));
  content: '';
}
.chain li:last-child::after {
  display: none;
}
.chain i {
  position: relative;
  z-index: 1;
  width: 10px;
  height: 10px;
  border-radius: 50%;
  background: #e879f9;
  box-shadow: 0 0 0 4px rgb(232 121 249 / 20%), 0 0 16px #e879f9;
  animation: pulse 3s ease-in-out infinite;
  animation-delay: calc(var(--i) * 0.4s);
}
@keyframes pulse {
  0%,
  100% {
    transform: scale(0.8);
    opacity: 0.6;
  }
  50% {
    transform: scale(1.1);
    opacity: 1;
  }
}
.card {
  position: relative;
  width: 420px;
  padding: 40px 36px 32px;
  border-radius: 24px;
  background: rgb(20 12 40 / 55%);
  box-shadow:
    inset 0 1px 0 rgb(255 255 255 / 12%),
    inset 0 0 0 1px rgb(255 255 255 / 6%),
    0 40px 100px rgb(0 0 0 / 55%);
  backdrop-filter: blur(28px) saturate(140%);
}
.halo {
  position: absolute;
  top: -60px;
  left: 50%;
  width: 260px;
  height: 120px;
  border-radius: 50%;
  background: radial-gradient(closest-side, rgb(232 121 249 / 45%), transparent);
  transform: translateX(-50%);
  filter: blur(20px);
  pointer-events: none;
}
h1 {
  position: relative;
  margin: 0;
  color: #fff;
  font-size: 26px;
  font-weight: 600;
}
.sub {
  position: relative;
  margin: 6px 0 26px;
  color: rgb(233 229 245 / 50%);
}
.card :deep(.ant-btn-primary) {
  border: 0;
  background: linear-gradient(90deg, #e879f9, #8b5cf6);
  box-shadow: 0 10px 30px rgb(232 121 249 / 35%);
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
  .stars,
  .nebula span,
  .chain i {
    animation: none;
  }
}
</style>
