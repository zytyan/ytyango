<template>
  <div class="app-shell">
    <header class="app-header">
      <div class="brand">
        <img alt="Telegram" src="https://telegram.org/img/t_logo.svg" />
        <div>
          <p class="eyebrow">ytyango</p>
          <h1>Telegram WebApp 控制台</h1>
          <p class="tagline">基于 http/backend 的 Vue + Vite SSG 前端</p>
        </div>
      </div>
    </header>

    <main class="app-main">
      <RouterView />
    </main>

    <footer class="app-footer">
      <p>已内置 <code>telegram-web-app.js</code> ，用于获取 WebApp initData 以访问后端。</p>
      <p class="muted">前端使用 Vite SSG 生成，可直接部署为静态资源。</p>
    </footer>
  </div>
</template>

<script setup lang="ts"></script>
