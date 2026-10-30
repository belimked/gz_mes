<script setup lang="ts">
// Tech style: canvas particles stream along the process chain; nodes pulse as particles arrive.
import { onBeforeUnmount, onMounted, ref } from 'vue'
import { brand } from '@/theme'
import BrandLogo from './BrandLogo.vue'
import LoginForm from './LoginForm.vue'
import { techTheme } from './techTheme'

const t = techTheme('#3b82f6')
const steps = ['开料', '折弯', '钣金', '喷涂', '质检', '装架', '发货']
const canvas = ref<HTMLCanvasElement>()
let raf = 0

onMounted(() => {
  const c = canvas.value
  const ctx = c?.getContext('2d')
  if (!c || !ctx) return
  const reduce = window.matchMedia('(prefers-reduced-motion: reduce)').matches
  const dpr = window.devicePixelRatio || 1
  let w = 0
  let h = 0
  const resize = () => {
    w = c.clientWidth
    h = c.clientHeight
    c.width = w * dpr
    c.height = h * dpr
    ctx.setTransform(dpr, 0, 0, dpr, 0, 0)
  }
  resize()
  window.addEventListener('resize', resize)
  // Path: a gentle sine wave across the canvas; particles travel along it with jitter.
  const yAt = (x: number, lane: number) => h * 0.8 + Math.sin(x / w * Math.PI * 2 + lane) * h * 0.06 + lane * 18
  const ps = Array.from({ length: 160 }, () => ({ x: Math.random(), lane: (Math.random() - 0.5) * 4, v: 0.0008 + Math.random() * 0.0018, r: 0.6 + Math.random() * 1.8 }))
  const draw = () => {
    ctx.clearRect(0, 0, w, h)
    for (let lane = -2; lane <= 2; lane++) {
      ctx.beginPath()
      for (let x = 0; x <= w; x += 8) ctx.lineTo(x, yAt(x, lane * 0.5))
      ctx.strokeStyle = 'rgba(59,130,246,0.08)'
      ctx.stroke()
    }
    for (const p of ps) {
      if (!reduce) p.x = (p.x + p.v) % 1
      const x = p.x * w
      const y = yAt(x, p.lane * 0.5)
      const g = ctx.createRadialGradient(x, y, 0, x, y, p.r * 5)
      g.addColorStop(0, 'rgba(147,197,253,0.9)')
      g.addColorStop(1, 'rgba(59,130,246,0)')
      ctx.fillStyle = g
      ctx.beginPath()
      ctx.arc(x, y, p.r * 5, 0, Math.PI * 2)
      ctx.fill()
    }
    steps.forEach((_, i) => {
      const x = ((i + 0.5) / steps.length) * w
      const y = yAt(x, 0)
      ctx.beginPath()
      ctx.arc(x, y, 6, 0, Math.PI * 2)
      ctx.fillStyle = '#0b1220'
      ctx.fill()
      ctx.lineWidth = 2
      ctx.strokeStyle = '#60a5fa'
      ctx.stroke()
    })
    if (!reduce) raf = requestAnimationFrame(draw)
  }
  draw()
  onBeforeUnmount(() => window.removeEventListener('resize', resize))
})
onBeforeUnmount(() => cancelAnimationFrame(raf))
</script>

<template>
  <div class="page">
    <canvas ref="canvas" class="bg" aria-hidden="true" />
    <div class="labels" aria-hidden="true">
      <span v-for="(s, i) in steps" :key="s" :style="{ left: `${((i + 0.5) / steps.length) * 100}%` }">{{ s }}</span>
    </div>
    <header>
      <div class="brand"><BrandLogo bg="#3b82f6" :size="32" /> <span>{{ brand.name }}</span></div>
      <span class="live"><i /> 实时数据流</span>
    </header>
    <div class="center">
      <h2>数据驱动<span>每一道工序</span></h2>
      <section class="card">
        <a-config-provider :theme="t">
          <LoginForm dark accent="#3b82f6" />
        </a-config-provider>
      </section>
    </div>
  </div>
</template>

<style scoped>
.page {
  position: relative;
  min-height: 100vh;
  overflow: hidden;
  color: #e2e8f0;
  background: radial-gradient(ellipse at 50% 120%, #0f2a5c 0%, #050914 55%);
}
.bg {
  position: absolute;
  inset: 0;
  width: 100%;
  height: 100%;
}
.labels span {
  position: absolute;
  top: calc(80% + 22px);
  color: rgb(147 197 253 / 70%);
  font-size: 12px;
  transform: translateX(-50%);
}
header {
  position: relative;
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 24px 40px;
}
.brand {
  display: flex;
  align-items: center;
  gap: 10px;
  color: #fff;
  font-weight: 600;
}
.live {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  color: rgb(226 232 240 / 60%);
  font-size: 12px;
}
.live i {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: #34d399;
  box-shadow: 0 0 8px #34d399;
}
.center {
  position: relative;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 24px;
  padding: 2vh 16px 96px;
}
h2 {
  margin: 0;
  color: #fff;
  font-size: 36px;
  font-weight: 700;
  text-align: center;
}
h2 span {
  margin-left: 12px;
  background: linear-gradient(90deg, #60a5fa, #a78bfa);
  background-clip: text;
  -webkit-text-fill-color: transparent;
}
.card {
  width: 100%;
  max-width: 400px;
  padding: 32px;
  border: 1px solid rgb(96 165 250 / 20%);
  border-radius: 16px;
  background: rgb(8 15 32 / 72%);
  box-shadow: 0 30px 80px rgb(0 0 0 / 50%);
  backdrop-filter: blur(18px);
}
@media (width <= 700px) {
  .labels {
    display: none;
  }
  h2 {
    font-size: 26px;
  }
}
</style>
