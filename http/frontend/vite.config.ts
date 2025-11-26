// vite.config.ts
import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

export default defineConfig({
    plugins: [vue()],
    build: {
        // 1️⃣ 顶层启用 SSR sourcemap
        sourcemap: true,

        // 2️⃣ 明确告知 SSR 构建也要 sourcemap
        rollupOptions: {
            output: {
                sourcemap: true,
            },
        },
    },

    // 3️⃣ SSG 的配置可以留着，但不是关键
    ssgOptions: {
        formatting: 'minify',
        script: 'async',
    },
})
