import { onMounted, onUnmounted, ref } from 'vue'
import { WebApp, WebAppUser } from 'telegram-web-app'

export function useTelegram() {
  const initData = ref('')
  const user = ref<WebAppUser | null>(null)
  const available = ref(false)
  let removeResizeFallback: (() => void) | null = null

  onMounted(() => {
    // In browsers (non-Telegram) CSS vars are absent; provide gentle fallbacks without overriding Telegram-provided vars.
    const applyFallbackVars = () => {
      const root = document.documentElement
      const getVar = (name: string) => getComputedStyle(root).getPropertyValue(name).trim()
      const setIfMissing = (name: string, value: string) => {
        if (!getVar(name)) root.style.setProperty(name, value)
      }
      const vh = `${window.innerHeight}px`
      setIfMissing('--tg-viewport-height', vh)
      setIfMissing('--tg-viewport-stable-height', vh)
      const zero = '0px'
      ;['safe-area-inset', 'content-safe-area-inset'].forEach((prefix) => {
        ;['top', 'bottom', 'left', 'right'].forEach((side) => setIfMissing(`--tg-${prefix}-${side}`, zero))
      })
    }
    applyFallbackVars()
    window.addEventListener('resize', applyFallbackVars)
    removeResizeFallback = () => window.removeEventListener('resize', applyFallbackVars)

    const webapp: WebApp = window.Telegram?.WebApp
    if (!webapp) return
    // initData may be empty in debug mode, but viewport and safe area data are still useful.
    if (webapp.initData !== '') {
      available.value = true
      initData.value = webapp.initData || ''
      user.value = webapp.initDataUnsafe.user ?? null
    }

    // Telegram docs recommend calling ready() once the UI is loaded.
    webapp.ready()

    // Ask Telegram to expand to full height (mobile bottom sheet).
    if (!webapp.isExpanded) {
      webapp.expand()
    }
  })

  onUnmounted(() => {
    removeResizeFallback?.()
  })

  return {
    initData,
    user,
    available,
  }
}
