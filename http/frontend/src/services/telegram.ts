import { onMounted, onUnmounted, ref } from 'vue'
import { WebApp, WebAppUser } from "telegram-web-app";

export function useTelegram() {
    const initData = ref('')
    const user = ref<WebAppUser | null>(null)
    const available = ref(false)
    let offViewportChanged: (() => void) | null = null

    onMounted(() => {
        const webapp: WebApp = window.Telegram?.WebApp
        if (!webapp || webapp.initData === '') return
        available.value = true
        initData.value = webapp.initData || ''
        user.value = webapp.initDataUnsafe.user ?? null

        const syncViewportVars = () => {
            const vh = webapp.viewportHeight || window.innerHeight
            const stableVh = webapp.viewportStableHeight || vh
            const rootStyle = document.documentElement.style
            rootStyle.setProperty('--tg-viewport-height', `${vh}px`)
            rootStyle.setProperty('--tg-viewport-stable-height', `${stableVh}px`)
        }

        // Telegram docs recommend calling ready() once the UI is loaded.
        webapp.ready()

        // Ask Telegram to expand to full height (mobile bottom sheet).
        if (!webapp.isExpanded) {
            webapp.expand()
        }

        // Sync CSS vars immediately and whenever Telegram reports viewport changes.
        syncViewportVars()
        webapp.onEvent('viewportChanged', syncViewportVars)
        offViewportChanged = () => webapp.offEvent('viewportChanged', syncViewportVars)
    })

    onUnmounted(() => {
        offViewportChanged?.()
    })

    return {
        initData,
        user,
        available,
    }
}
