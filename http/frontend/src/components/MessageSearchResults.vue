<template>
  <section class="message-results">
    <header class="results-header">
      <div>
        <p class="eyebrow">搜索结果</p>
        <h3>找到 {{ result.estimatedTotalHits }} 条消息</h3>
        <p class="muted">耗时 {{ result.processingTimeMs }} ms</p>
      </div>
      <div class="result-summary" v-if="query">
        <span class="summary-label">关键词</span>
        <span class="summary-value">{{ query }}</span>
      </div>
    </header>

    <div v-if="result.hits.length === 0" class="empty-hint">没有匹配的消息。</div>
    <div v-else class="result-list">
      <article v-for="hit in result.hits" :key="hit.mongo_id" class="result-row">
        <div class="avatar" :style="{ backgroundColor: avatarColor(hit.from_id) }">
          <span>{{ avatarInitial(hit) }}</span>
        </div>
        <div class="result-content">
          <div class="result-top">
            <div class="user-meta">
              <span class="user-name">{{ displayName(hit) }}</span>
              <span class="user-id">ID: {{ hit.from_id }}</span>
            </div>
            <div class="message-meta">
              <span class="message-id">#{{ hit.msg_id }}</span>
              <span class="message-time">{{ formatDate(hit.date) }}</span>
            </div>
          </div>
          <p class="message-text">{{ messagePreview(hit) }}</p>
        </div>
      </article>
    </div>
  </section>
</template>

<script setup lang="ts">
import type { MeiliMsg, SearchResult } from '../services/api'

defineProps<{
  result: SearchResult
  query?: string
}>()

const colorPalette = [
  '#5E81F4',
  '#7C3AED',
  '#10B981',
  '#F59E0B',
  '#EF4444',
  '#14B8A6',
  '#F472B6',
  '#3B82F6',
  '#8B5CF6',
  '#6366F1',
]

function avatarColor(userId: number) {
  const idString = String(userId)
  let hash = 0
  for (let i = 0; i < idString.length; i += 1) {
    hash = (hash << 5) - hash + idString.charCodeAt(i)
    hash |= 0
  }
  const index = Math.abs(hash) % colorPalette.length
  return colorPalette[index]
}

function displayName(hit: MeiliMsg) {
  if ((hit as Record<string, unknown>).from_name) {
    return String((hit as Record<string, unknown>).from_name)
  }
  return `用户 ${hit.from_id}`
}

function avatarInitial(hit: MeiliMsg) {
  const name = displayName(hit).trim()
  return name.charAt(0).toUpperCase() || '#'
}

function messagePreview(hit: MeiliMsg) {
  return hit.message || hit.image_text || hit.qr_result || '无正文'
}

function formatDate(timestamp: number) {
  const date = new Date(timestamp * 1000)
  return date.toLocaleString()
}
</script>
