<template>
  <section class="panel">
    <h2>Telegram 登录状态</h2>
    <p class="muted">从 WebApp 注入的 <code>initData</code> 用作后端的 <code>Authorization: Telegram &lt;initData&gt;</code> 头。</p>
    <div v-if="telegram.available" class="alert info">
      <span>已检测到 Telegram WebApp。当前用户：{{ telegram.user?.value?.first_name || telegram.user?.value?.first_name || telegram.user?.value?.id }}</span>
    </div>
    <div v-else class="alert warn">
      未检测到 WebApp 环境，可能无法访问需要验证的接口。
    </div>
    <p class="code">{{ telegram.initData || 'initData 不可用' }}</p>
  </section>

  <section class="panel">
    <h2>后端状态</h2>
    <div v-if="pingMessage" class="alert info">后端响应：{{ pingMessage }}</div>
    <div v-else class="muted">正在测试 /ping ...</div>
  </section>

  <section class="panel">
    <h2>群统计 &amp; 消息搜索</h2>
    <div class="input-grid">
      <div>
        <label for="groupId">群组 Web ID (ins_id)</label>
        <input id="groupId" v-model.number="groupWebId" type="number" placeholder="例如：123456789" />
      </div>
      <div>
        <label for="query">搜索关键词</label>
        <input id="query" v-model.trim="query" type="text" placeholder="关键词，例如 hello" />
      </div>
      <div>
        <label for="limit">每页数量</label>
        <select id="limit" v-model.number="limit">
          <option :value="10">10</option>
          <option :value="20">20</option>
          <option :value="30">30</option>
          <option :value="50">50</option>
        </select>
      </div>
      <div>
        <label for="page">页码</label>
        <input id="page" v-model.number="page" type="number" min="1" />
      </div>
    </div>

    <div class="input-grid" style="margin-top: 12px">
      <button :disabled="!canFetchStat || loadingStat" @click="loadStats">
        {{ loadingStat ? '加载中...' : '拉取群统计' }}
      </button>
      <button :disabled="!canSearch || loadingSearch" @click="performSearch">
        {{ loadingSearch ? '搜索中...' : '搜索消息' }}
      </button>
    </div>

    <p v-if="error" class="alert error" style="margin-top: 12px">{{ error }}</p>

    <div v-if="stat" class="stats-grid" style="margin-top: 16px">
      <div class="stat-card">
        <p class="stat-label">消息总数</p>
        <p class="stat-value">{{ stat.message_count }}</p>
      </div>
      <div class="stat-card">
        <p class="stat-label">图片</p>
        <p class="stat-value">{{ stat.photo_count }}</p>
      </div>
      <div class="stat-card">
        <p class="stat-label">视频</p>
        <p class="stat-value">{{ stat.video_count }}</p>
      </div>
      <div class="stat-card">
        <p class="stat-label">贴纸</p>
        <p class="stat-value">{{ stat.sticker_count }}</p>
      </div>
      <div class="stat-card">
        <p class="stat-label">火星车</p>
        <p class="stat-value">{{ stat.mars_count }} / {{ stat.max_mars_count }}</p>
      </div>
      <div class="stat-card">
        <p class="stat-label">敏感内容 (R/A)</p>
        <p class="stat-value">{{ stat.racy_count }} / {{ stat.adult_count }}</p>
      </div>
      <div class="stat-card">
        <p class="stat-label">下载 (视频/音频)</p>
        <p class="stat-value">{{ stat.download_video_count }} / {{ stat.download_audio_count }}</p>
      </div>
      <div class="stat-card">
        <p class="stat-label">入群 / 封禁</p>
        <p class="stat-value">{{ stat.dio_add_user_count }} / {{ stat.dio_ban_user_count }}</p>
      </div>
    </div>

    <div v-if="searchResult" class="panel" style="margin-top: 16px; padding: 12px">
      <h3>搜索结果（{{ searchResult.estimatedTotalHits }} 条，耗时 {{ searchResult.processingTimeMs }} ms）</h3>
      <div v-if="searchResult.hits.length === 0" class="muted">没有匹配的消息。</div>
      <div v-else class="results">
        <article v-for="hit in searchResult.hits" :key="hit.mongo_id" class="result-card">
          <div class="result-meta">
            <span>用户: {{ hit.from_id }}</span>
            <span>消息ID: {{ hit.msg_id }}</span>
            <span>时间: {{ formatDate(hit.date) }}</span>
          </div>
          <p class="result-message">{{ hit.message || hit.image_text || hit.qr_result || '无正文' }}</p>
        </article>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { fetchGroupStat, ping, searchMessages, type ChatStat, type SearchResult } from '../services/api'
import { useTelegram } from '../services/telegram'
const telegram = useTelegram()
const groupWebId = ref<number | null>(null)
const query = ref('')
const limit = ref(20)
const page = ref(1)
const stat = ref<ChatStat | null>(null)
const searchResult = ref<SearchResult | null>(null)
const pingMessage = ref('')
const error = ref('')
const loadingStat = ref(false)
const loadingSearch = ref(false)

const canFetchStat = computed(() => !!groupWebId.value)
const canSearch = computed(() => !!groupWebId.value && page.value > 0 && query.value.trim().length > 0)

onMounted(async () => {
  try {
    const res = await ping()
    pingMessage.value = res.message
  } catch (err) {
    error.value = (err as Error).message
  }
})

function ensureInitData() {
  if (!telegram.available || !telegram.initData.value) {
    throw new Error('需要 Telegram WebApp 提供的 initData')
  }
  return telegram.initData.value
}

async function loadStats() {
  if (!groupWebId.value) return
  loadingStat.value = true
  error.value = ''
  try {
    const initData = ensureInitData()
    stat.value = await fetchGroupStat(groupWebId.value, initData)
  } catch (err) {
    error.value = (err as Error).message
  } finally {
    loadingStat.value = false
  }
}

async function performSearch() {
  if (!groupWebId.value || !query.value) return
  loadingSearch.value = true
  error.value = ''
  try {
    const initData = ensureInitData()
    searchResult.value = await searchMessages(
      {
        q: query.value,
        ins_id: groupWebId.value,
        page: page.value,
        limit: limit.value,
      },
      initData,
    )
  } catch (err) {
    error.value = (err as Error).message
  } finally {
    loadingSearch.value = false
  }
}

function formatDate(timestamp: number) {
  const date = new Date(timestamp * 1000)
  return date.toLocaleString()
}
</script>
