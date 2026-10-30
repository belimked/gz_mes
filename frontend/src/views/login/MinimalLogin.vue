<script setup lang="ts">
// Style 5: plain white, typography only. Fast to load, good for shop-floor terminals.
import { brand } from '@/theme'
import LoginForm from './LoginForm.vue'
</script>

<template>
  <div class="page">
    <header>
      <span class="word">{{ brand.name }}</span>
    </header>
    <main>
      <div class="box">
        <h1>登录</h1>
        <p class="sub">{{ brand.subtitle }}</p>
        <LoginForm />
      </div>
    </main>
    <footer>© {{ new Date().getFullYear() }} {{ brand.name }}</footer>
  </div>
</template>

<style scoped>
.page {
  display: flex;
  flex-direction: column;
  min-height: 100vh;
  background: #fff;
}
header {
  padding: 24px 32px;
}
.word {
  font-size: 16px;
  font-weight: 700;
  letter-spacing: 3px;
}
main {
  display: flex;
  flex: 1;
  align-items: center;
  justify-content: center;
  padding: 24px 24px 72px;
}
.box {
  width: 100%;
  max-width: 340px;
}
h1 {
  margin: 0;
  font-size: 32px;
  font-weight: 300;
  letter-spacing: 4px;
}
.sub {
  margin: 8px 0 32px;
  padding-bottom: 20px;
  border-bottom: 1px solid #f0f0f0;
  color: #8c8c8c;
}
footer {
  padding: 16px 32px 80px;
  color: #bfbfbf;
  font-size: 12px;
}
</style>
