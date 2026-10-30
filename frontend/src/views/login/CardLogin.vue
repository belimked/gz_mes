<script setup lang="ts">
// Style 2: soft gradient background with a centered glass card.
import { brand } from '@/theme'
import BrandLogo from './BrandLogo.vue'
import LoginForm from './LoginForm.vue'
</script>

<template>
  <div class="page">
    <span class="blob b1" /><span class="blob b2" /><span class="blob b3" />
    <div class="card">
      <div class="head">
        <BrandLogo :size="48" />
        <h1>{{ brand.name }}</h1>
        <p>{{ brand.subtitle }}</p>
      </div>
      <LoginForm />
    </div>
    <footer>© {{ new Date().getFullYear() }} {{ brand.name }} · 建议使用 Chrome / Edge 浏览器</footer>
  </div>
</template>

<style scoped>
.page {
  position: relative;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  min-height: 100vh;
  padding: 40px 16px 96px;
  overflow: hidden;
  background: linear-gradient(135deg, #eef4ff 0%, #f7f9fc 50%, #eafaf7 100%);
}
.blob {
  position: absolute;
  border-radius: 50%;
  filter: blur(60px);
  opacity: 0.55;
}
.b1 {
  top: -120px;
  left: -80px;
  width: 420px;
  height: 420px;
  background: #91caff;
}
.b2 {
  right: -100px;
  bottom: -140px;
  width: 480px;
  height: 480px;
  background: #87e8de;
}
.b3 {
  top: 40%;
  left: 60%;
  width: 260px;
  height: 260px;
  background: #d3adf7;
  opacity: 0.35;
}
.card {
  position: relative;
  width: 100%;
  max-width: 420px;
  padding: 40px 36px 32px;
  border: 1px solid rgb(255 255 255 / 70%);
  border-radius: 16px;
  background: rgb(255 255 255 / 78%);
  box-shadow: 0 20px 60px rgb(22 119 255 / 12%);
  backdrop-filter: blur(16px);
}
.head {
  margin-bottom: 28px;
  text-align: center;
}
h1 {
  margin: 14px 0 4px;
  font-size: 24px;
  font-weight: 600;
}
.head p {
  margin: 0;
  color: #8c8c8c;
}
footer {
  position: relative;
  margin-top: 24px;
  color: #a0a7b4;
  font-size: 12px;
  text-align: center;
}
</style>
