interface TelegramWebAppUser {
  id: number
  first_name?: string
  last_name?: string
  username?: string
  language_code?: string
  is_premium?: boolean
  allows_write_to_pm?: boolean
}

interface TelegramWebAppInitData {
  query_id?: string
  user?: TelegramWebAppUser
  auth_date?: number
  hash?: string
}

interface TelegramWebApp {
  initData: string
  initDataUnsafe: TelegramWebAppInitData
  isExpanded: boolean
  ready: () => void
  expand: () => void
}

interface TelegramNamespace {
  WebApp?: TelegramWebApp
}

declare global {
  interface Window {
    Telegram?: TelegramNamespace
  }
}

export {}
