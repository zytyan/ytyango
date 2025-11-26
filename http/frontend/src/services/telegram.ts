import { onMounted, ref } from 'vue'
import { WebApp, WebAppUser } from "telegram-web-app";

export function useTelegram() {
    const initData = ref('')
    const user = ref<WebAppUser | null>(null)
    const available = ref(false)

    onMounted(() => {
        const webapp: WebApp = window.Telegram?.WebApp
        if (!webapp || webapp.initData === '') return
        available.value = true
        initData.value = webapp.initData || ''
        user.value = webapp.initDataUnsafe.user ?? null
        console.log(available, initData, user)
        
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
