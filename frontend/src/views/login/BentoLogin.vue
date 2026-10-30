<script setup lang="ts">
// Modern style: Apple-like bento grid of live-looking tiles; the login form is one tile.
import { brand } from '@/theme'
import BrandLogo from './BrandLogo.vue'
import LoginForm from './LoginForm.vue'
import { techTheme } from './techTheme'

const t = techTheme('#a3e635')
const bars = [42, 58, 51, 66, 73, 61, 88]
const feed = [
  { time: '14:32', text: '钣金下线 · BQ-00128' },
  { time: '14:31', text: '喷涂接收 · BQ-00117' },
  { time: '14:29', text: '成品装架 · BQ-00093' },
]
</script>

<template>
  <div class="page">
    <div class="grid">
      <section class="tile login">
        <div class="brand"><BrandLogo bg="#a3e635" fg="#0a0a0a" :size="32" /> <span>{{ brand.name }}</span></div>
        <h1>登录</h1>
        <p class="sub">{{ brand.subtitle }}</p>
        <a-config-provider :theme="t">
          <LoginForm dark accent="#a3e635" />
        </a-config-provider>
      </section>

      <section class="tile hero" aria-hidden="true">
        <span class="k">今日完工面积</span>
        <b class="big">1,286.4<small>m²</small></b>
        <div class="bars">
          <span v-for="(b, i) in bars" :key="i" :style="{ height: `${b}%` }" :class="{ hot: i === bars.length - 1 }" />
        </div>
        <span class="k">近 7 日 · 较上周 <em>+12.6%</em></span>
      </section>

      <section class="tile ring" aria-hidden="true">
        <svg viewBox="0 0 120 120">
          <circle cx="60" cy="60" r="48" class="track" />
          <circle cx="60" cy="60" r="48" class="val" />
        </svg>
        <div class="ring-text"><b>96.8%</b><span>按期交付率</span></div>
      </section>

      <section class="tile feed" aria-hidden="true">
        <span class="k"><i class="dot" /> 实时扫描</span>
        <ul>
          <li v-for="f in feed" :key="f.time + f.text"><time>{{ f.time }}</time>{{ f.text }}</li>
        </ul>
      </section>

      <section class="tile quote" aria-hidden="true">
        <b>一件一码</b>
        <span>每块板从开料到发货全程可追溯</span>
      </section>
    </div>
  </div>
</template>

<style scoped>
.page {
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 100vh;
  padding: 32px 24px 96px;
  color: #e5e5e5;
  background: #0a0a0a;
}
.grid {
  display: grid;
  grid-template-areas:
    'login hero hero'
    'login ring feed'
    'login quote quote';
  grid-template-columns: 420px 1fr 1fr;
  gap: 14px;
  width: 100%;
  max-width: 1120px;
}
.tile {
  position: relative;
  padding: 24px;
  overflow: hidden;
  border: 1px solid rgb(255 255 255 / 7%);
  border-radius: 22px;
  background: #151515;
}
.login {
  grid-area: login;
  padding: 32px;
}
.hero {
  grid-area: hero;
  display: flex;
  flex-direction: column;
  gap: 10px;
  background: linear-gradient(140deg, #1c2a0a, #151515 60%);
}
.ring {
  grid-area: ring;
  display: flex;
  align-items: center;
  justify-content: center;
}
.feed {
  grid-area: feed;
}
.quote {
  grid-area: quote;
  display: flex;
  align-items: center;
  gap: 16px;
  background: #a3e635;
  color: #0a0a0a;
}
.brand {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 32px;
  color: #fff;
  font-weight: 600;
}
h1 {
  margin: 0;
  color: #fff;
  font-size: 30px;
  font-weight: 700;
}
.sub {
  margin: 4px 0 24px;
  color: rgb(229 229 229 / 50%);
}
.login :deep(.ant-btn-primary) {
  color: #0a0a0a;
  font-weight: 600;
}
.k {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  color: rgb(229 229 229 / 55%);
  font-size: 13px;
}
.k em {
  color: #a3e635;
  font-style: normal;
}
.big {
  color: #fff;
  font-size: 48px;
  font-weight: 700;
  font-variant-numeric: tabular-nums;
  line-height: 1;
}
.big small {
  margin-left: 6px;
  color: rgb(229 229 229 / 50%);
  font-size: 18px;
  font-weight: 400;
}
.bars {
  display: flex;
  flex: 1;
  gap: 10px;
  align-items: flex-end;
  min-height: 90px;
}
.bars span {
  flex: 1;
  border-radius: 6px;
  background: rgb(255 255 255 / 10%);
}
.bars span.hot {
  background: #a3e635;
  box-shadow: 0 0 20px rgb(163 230 53 / 40%);
}
.ring svg {
  width: 150px;
  height: 150px;
  transform: rotate(-90deg);
}
circle {
  fill: none;
  stroke-width: 10;
  stroke-linecap: round;
}
.track {
  stroke: rgb(255 255 255 / 8%);
}
.val {
  stroke: #a3e635;
  stroke-dasharray: 301.6;
  stroke-dashoffset: 9.65;
}
.ring-text {
  position: absolute;
  display: flex;
  flex-direction: column;
  align-items: center;
}
.ring-text b {
  color: #fff;
  font-size: 24px;
}
.ring-text span {
  color: rgb(229 229 229 / 55%);
  font-size: 12px;
}
.dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: #a3e635;
  box-shadow: 0 0 8px #a3e635;
}
.feed ul {
  display: grid;
  gap: 12px;
  margin: 16px 0 0;
  padding: 0;
  list-style: none;
  font-size: 13px;
}
.feed time {
  margin-right: 10px;
  color: rgb(229 229 229 / 40%);
  font-family: ui-monospace, monospace;
}
.quote b {
  font-size: 22px;
}
@media (width <= 960px) {
  .grid {
    grid-template-areas: 'login';
    grid-template-columns: 1fr;
    max-width: 420px;
  }
  .hero,
  .ring,
  .feed,
  .quote {
    display: none;
  }
}
</style>
