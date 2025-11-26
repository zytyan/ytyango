import {onMounted, ref} from 'vue'
import {WebApp} from "telegram-web-app";

export function useTelegram() {
    const initData = ref('')
    const user = ref<TelegramWebAppUser | null>(null)
    const available = ref(false)

    onMounted(() => {
        const webapp: WebApp = window.Telegram?.WebApp
        if (!webapp) return {available: false, initData: '', user: null}
        if (webapp.initData === '') {
            return {available: false, initData: '', user: null}
        }
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
