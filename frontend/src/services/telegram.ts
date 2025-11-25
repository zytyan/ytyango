import { onMounted, ref } from 'vue'

export function useTelegram() {
  const initData = ref('')
  const user = ref<TelegramWebAppUser | null>(null)
  const available = ref(false)

  onMounted(() => {
    const webapp = window.Telegram?.WebApp
    if (!webapp) return

    available.value = true
    initData.value = webapp.initData || ''
    user.value = webapp.initDataUnsafe?.user || null

    if (!webapp.isExpanded) {
      webapp.expand()
    }
    webapp.ready()
  })

  return {
    initData,
    user,
    available,
  }
}
