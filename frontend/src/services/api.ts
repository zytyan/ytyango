export interface PingResponse {
  message: string
}

export interface ChatStat {
  chat_id: number
  stat_date: number
  message_count: number
  photo_count: number
  video_count: number
  sticker_count: number
  forward_count: number
  mars_count: number
  max_mars_count: number
  racy_count: number
  adult_count: number
  download_video_count: number
  download_audio_count: number
  dio_add_user_count: number
  dio_ban_user_count: number
  user_msg_stat: Record<string, { MsgCount: number; MsgLen: number }>
  msg_count_by_time: number[]
  msg_id_at_time_start: number[]
}

export interface MeiliMsg {
  mongo_id: string
  peer_id: number
  from_id: number
  msg_id: number
  date: number
  message?: string
  image_text?: string
  qr_result?: string
}

export interface SearchResult {
  hits: MeiliMsg[]
  query: string
  processingTimeMs: number
  limit: number
  offset: number
  estimatedTotalHits: number
}

export interface ErrorResponse {
  status: string
  code: number
  error: string
}

export interface SearchParams {
  q: string
  ins_id: number
  page: number
  limit?: number
}

const API_BASE = import.meta.env.VITE_API_BASE_URL || 'http://127.0.0.1:4021'

async function requestJson<T>(path: string, init: RequestInit = {}) {
  const resp = await fetch(`${API_BASE}${path}`, {
    ...init,
    headers: {
      'Content-Type': 'application/json',
      ...(init.headers || {}),
    },
  })

  const data = await resp.json()
  if (!resp.ok) {
    const err = data as ErrorResponse
    throw new Error(err?.error || '请求失败')
  }
  return data as T
}

export async function ping(): Promise<PingResponse> {
  return requestJson('/ping')
}

export async function fetchGroupStat(groupWebId: number, initData?: string): Promise<ChatStat> {
  const headers = initData ? { Authorization: `Telegram ${initData}` } : undefined
  return requestJson(`/tg/group_stat?group_web_id=${groupWebId}`, { headers })
}

export async function searchMessages(params: SearchParams, initData?: string): Promise<SearchResult> {
  const headers = initData ? { Authorization: `Telegram ${initData}` } : undefined
  const qs = new URLSearchParams({
    q: params.q,
    ins_id: String(params.ins_id),
    page: String(params.page),
  })
  if (params.limit) {
    qs.set('limit', String(params.limit))
  }
  return requestJson(`/tg/search?${qs.toString()}`, { headers })
}
