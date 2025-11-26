import { onMounted, onUnmounted, ref } from 'vue'
import { WebApp, WebAppUser } from 'telegram-web-app'

export function useTelegram() {
  const initData = ref('')
  const user = ref<WebAppUser | null>(null)
  const available = ref(false)
  let offViewportChanged: (() => void) | null = null
  let offSafeAreaChanged: (() => void) | null = null
  let offContentSafeAreaChanged: (() => void) | null = null

  onMounted(() => {
    const webapp: WebApp = window.Telegram?.WebApp
    if (!webapp) return
    // initData may be empty in debug mode, but viewport and safe area data are still useful.
    if (webapp.initData !== '') {
      available.value = true
      initData.value = webapp.initData || ''
      user.value = webapp.initDataUnsafe.user ?? null
    }

    const syncCssVars = () => {
      const vh = webapp.viewportHeight || window.innerHeight
      const stableVh = webapp.viewportStableHeight || vh
      const rootStyle = document.documentElement.style
      rootStyle.setProperty('--tg-viewport-height', `${vh}px`)
      rootStyle.setProperty('--tg-viewport-stable-height', `${stableVh}px`)

      const applyInset = (prefix: string, inset?: { top?: number; bottom?: number; left?: number; right?: number }) => {
        rootStyle.setProperty(`--tg-${prefix}-top`, `${inset?.top ?? 0}px`)
        rootStyle.setProperty(`--tg-${prefix}-bottom`, `${inset?.bottom ?? 0}px`)
        rootStyle.setProperty(`--tg-${prefix}-left`, `${inset?.left ?? 0}px`)
        rootStyle.setProperty(`--tg-${prefix}-right`, `${inset?.right ?? 0}px`)
      }

      applyInset('safe-area-inset', webapp.safeAreaInset)
      applyInset('content-safe-area-inset', webapp.contentSafeAreaInset)
    }

    // Telegram docs recommend calling ready() once the UI is loaded.
    webapp.ready()

    // Ask Telegram to expand to full height (mobile bottom sheet).
    if (!webapp.isExpanded) {
      webapp.expand()
    }

    // Sync CSS vars immediately and whenever Telegram reports changes.
    syncCssVars()
    webapp.onEvent('viewportChanged', syncCssVars)
    webapp.onEvent('safeAreaChanged', syncCssVars)
    webapp.onEvent('contentSafeAreaChanged', syncCssVars)

    offViewportChanged = () => webapp.offEvent('viewportChanged', syncCssVars)
    offSafeAreaChanged = () => webapp.offEvent('safeAreaChanged', syncCssVars)
    offContentSafeAreaChanged = () => webapp.offEvent('contentSafeAreaChanged', syncCssVars)
  })

  onUnmounted(() => {
    offViewportChanged?.()
    offSafeAreaChanged?.()
    offContentSafeAreaChanged?.()
  })

  return {
    initData,
    user,
    available,
  }
}
